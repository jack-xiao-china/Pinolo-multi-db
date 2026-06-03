package stage2

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qaqcatz/impomysql/connector"
	"github.com/qaqcatz/impomysql/mutation/oracle"
	"github.com/qaqcatz/impomysql/mutation/stage1"
)

// loadTestEnv: load environment variables from test_db.env file
// This keeps DB credentials out of command-line history
func loadTestEnv() {
	// Find project root (where test_db.env lives)
	envPaths := []string{
		"test_db.env",
		filepath.Join("..", "..", "test_db.env"),
	}
	for _, p := range envPaths {
		if _, err := os.Stat(p); err == nil {
			loadEnvFile(p)
			return
		}
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Only set if not already defined in environment
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func init() {
	loadTestEnv()
}

// =====================================================================
// GaussDB-M integration test
// =====================================================================

func getGaussDBMConnector(t *testing.T) *connector.OpenGaussConnector {
	t.Helper()
	host := os.Getenv("GAUSSDBM_HOST")
	portStr := os.Getenv("GAUSSDBM_PORT")
	username := os.Getenv("GAUSSDBM_USERNAME")
	password := os.Getenv("GAUSSDBM_PASSWORD")
	dbname := os.Getenv("GAUSSDBM_DBNAME")

	if host == "" {
		host = "121.37.186.131"
	}
	if portStr == "" {
		portStr = "19995"
	}
	if username == "" {
		username = "sqlbuilder1"
	}
	if password == "" {
		t.Skip("GAUSSDBM_PASSWORD not set, skipping GaussDB-M integration test")
	}
	if dbname == "" {
		dbname = "testm"
	}

	port := 19995
	if p, err := parsePort(portStr); err == nil {
		port = p
	}

	conn, err := connector.NewOpenGaussConnector(host, port, username, password, dbname)
	if err != nil {
		t.Fatalf("NewOpenGaussConnector error: %v", err)
	}
	return conn
}

func setupGaussDBMTable(t *testing.T, conn *connector.OpenGaussConnector) {
	t.Helper()
	// Create test table with IF-compatible data for M-mode testing
	result := conn.ExecSQL("DROP TABLE IF EXISTS test_m_company")
	if result.Err != nil {
		t.Logf("DROP error (may be ok): %v", result.Err)
	}
	result = conn.ExecSQL("CREATE TABLE test_m_company (id INT PRIMARY KEY, name VARCHAR(100), age INT, city VARCHAR(50))")
	if result.Err != nil {
		t.Fatalf("CREATE TABLE error: %v", result.Err)
	}
	result = conn.ExecSQL("INSERT INTO test_m_company VALUES (1, 'Alice', 25, 'Beijing'), (2, 'Bob', 30, 'Shanghai'), (3, 'Charlie', 22, 'Guangzhou'), (4, 'David', 35, 'Beijing'), (5, 'Eve', 28, 'Shanghai')")
	if result.Err != nil {
		t.Fatalf("INSERT error: %v", result.Err)
	}
}

// TestGaussDBMIntegration: end-to-end test on GaussDB-M
// Tests M-mode mutations (FixMIfToCase, FixMConcatToPipe) with oracle check
func TestGaussDBMIntegration(t *testing.T) {
	conn := getGaussDBMConnector(t)
	defer conn.Close()
	setupGaussDBMTable(t, conn)

	testSQLs := []struct {
		name string
		sql  string
		seed int64
	}{
		// Basic WHERE clause (inherited MySQL mutations + EET)
		{"BasicWhere", "SELECT * FROM test_m_company WHERE age > 25", 10001},
		// IF function in WHERE (M-mode specific mutation)
		{"IfToCase", "SELECT * FROM test_m_company WHERE IF(age > 25, 1, 0) = 1", 10002},
		// CONCAT function in WHERE (M-mode specific mutation)
		{"ConcatToPipe", "SELECT * FROM test_m_company WHERE CONCAT(name, 'x') = 'Alicex'", 10003},
		// AND in WHERE (DeMorgan EET mutation)
		{"DeMorganAnd", "SELECT * FROM test_m_company WHERE age > 20 AND city = 'Beijing'", 10004},
		// OR in WHERE (DeMorgan EET mutation)
		{"DeMorganOr", "SELECT * FROM test_m_company WHERE age > 25 OR city = 'Shanghai'", 10005},
		// BETWEEN (BetweenToCmp EET mutation)
		{"Between", "SELECT * FROM test_m_company WHERE age BETWEEN 20 AND 30", 10006},
	}

	for _, tc := range testSQLs {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Println("==================================================")
			t.Logf("Testing SQL: %s", tc.sql)

			// 1. Stage1 preprocessing
			stage1Result := stage1.InitAndExecForMMode(tc.sql, conn)
			if stage1Result.Skipped {
				t.Logf("[Skipped] reason: %s", stage1Result.SkipReason)
				return
			}
			if stage1Result.Err != nil {
				t.Fatalf("Stage1 error: %v", stage1Result.Err)
			}
			if stage1Result.ExecResult.Err != nil {
				t.Fatalf("Stage1 exec error: %v", stage1Result.ExecResult.Err)
			}

			originalSql := stage1Result.InitSql
			originalResult := stage1Result.ExecResult

			// 2. Stage2 mutation
			stage2Result := MutateAllAndExecForMMode(originalSql, tc.seed, conn)
			if stage2Result.Err != nil {
				t.Fatalf("Stage2 error: %v", stage2Result.Err)
			}

			t.Logf("Total mutations: %d", len(stage2Result.MutateUnits))

			// 3. Oracle check for each mutation
			bugCount := 0
			equivalenceCount := 0
			for _, mu := range stage2Result.MutateUnits {
				if mu.Err != nil {
					t.Logf("Mutation %s error: %v", mu.Name, mu.Err)
					continue
				}
				if mu.ExecResult == nil || mu.ExecResult.Err != nil {
					t.Logf("Mutation %s exec error: %v", mu.Name, mu.ExecResult.Err)
					continue
				}

				// Oracle check based on IsEquivalence
				var check bool
				var oracleErr error
				if mu.IsEquivalence {
					equivalenceCount++
					check, oracleErr = oracle.CheckEquivalence(originalResult, mu.ExecResult)
				} else {
					check, oracleErr = oracle.Check(originalResult, mu.ExecResult, mu.IsUpper)
				}
				if oracleErr != nil {
					t.Logf("Oracle error for %s: %v", mu.Name, oracleErr)
					continue
				}
				if !check {
					bugCount++
					t.Logf("BUG! mutation=%s isUpper=%v isEquivalence=%v sql=%s", mu.Name, mu.IsUpper, mu.IsEquivalence, mu.Sql)
				}
			}

			t.Logf("Result: %d mutations, %d equivalence mutations, %d potential bugs", len(stage2Result.MutateUnits), equivalenceCount, bugCount)
			// Note: bugs are expected - they may be real logical bugs in GaussDB-M
			// We just verify the pipeline works end-to-end without crashes
		})
	}
}

// =====================================================================
// MySQL integration test (EET mutations)
// =====================================================================

func getMySQLConnector(t *testing.T) *connector.Connector {
	t.Helper()
	host := os.Getenv("TEST_DB_HOST")
	portStr := os.Getenv("TEST_DB_PORT_MYSQL")
	username := os.Getenv("TEST_DB_USERNAME")
	password := os.Getenv("TEST_DB_PASSWORD")
	dbname := os.Getenv("TEST_DB_NAME")

	if host == "" {
		host = "127.0.0.1"
	}
	if portStr == "" {
		portStr = "3306"
	}
	if username == "" {
		username = "tpcc"
	}
	if password == "" {
		t.Skip("TEST_DB_PASSWORD not set, skipping MySQL integration test")
	}
	if dbname == "" {
		dbname = "TEST"
	}

	port := 3306
	if p, err := parsePort(portStr); err == nil {
		port = p
	}

	conn, err := connector.NewConnector(host, port, username, password, dbname)
	if err != nil {
		t.Fatalf("NewConnector error: %v", err)
	}
	return conn
}

func setupMySQLTable(t *testing.T, conn *connector.Connector) {
	t.Helper()
	result := conn.ExecSQL("DROP TABLE IF EXISTS test_eet_company")
	if result.Err != nil {
		t.Logf("DROP error (may be ok): %v", result.Err)
	}
	result = conn.ExecSQL("CREATE TABLE test_eet_company (id INT PRIMARY KEY, name TEXT, age INT, city TEXT, KEY(id), KEY(age))")
	if result.Err != nil {
		t.Fatalf("CREATE TABLE error: %v", result.Err)
	}
	result = conn.ExecSQL("INSERT INTO test_eet_company VALUES (1, 'A', 18, 'a'), (2, 'B', 19, 'b'), (3, 'C', 20, 'c'), (4, 'A', 19, 'c'), (5, 'A', 19, 'c'), (6, 'B', 18, 'b')")
	if result.Err != nil {
		t.Fatalf("INSERT error: %v", result.Err)
	}
}

// TestMySQLEETIntegration: end-to-end EET mutation test on MySQL
func TestMySQLEETIntegration(t *testing.T) {
	conn := getMySQLConnector(t)
	defer conn.Close()
	setupMySQLTable(t, conn)

	testSQLs := []struct {
		name string
		sql  string
		seed int64
	}{
		// EET tautology/contradiction wrapping
		{"BasicWhere", "SELECT * FROM test_eet_company WHERE id > 0", 10001},
		// DeMorgan EET
		{"DeMorganAnd", "SELECT * FROM test_eet_company WHERE id > 0 AND age > 18", 10002},
		{"DeMorganOr", "SELECT * FROM test_eet_company WHERE id > 0 OR age > 18", 10003},
		// BETWEEN EET
		{"Between", "SELECT * FROM test_eet_company WHERE age BETWEEN 18 AND 20", 10004},
		// COALESCE EET
		{"Coalesce", "SELECT * FROM test_eet_company WHERE COALESCE(age, 0) > 18", 10005},
		// NULLIF EET
		{"Nullif", "SELECT * FROM test_eet_company WHERE NULLIF(age, 18) IS NOT NULL", 10006},
	}

	for _, tc := range testSQLs {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Println("==================================================")
			t.Logf("Testing SQL: %s", tc.sql)

			// 1. Stage1 preprocessing
			stage1Result := stage1.InitAndExec(tc.sql, conn)
			if stage1Result.Err != nil {
				t.Fatalf("Stage1 error: %v", stage1Result.Err)
			}
			if stage1Result.ExecResult.Err != nil {
				t.Fatalf("Stage1 exec error: %v", stage1Result.ExecResult.Err)
			}

			originalSql := stage1Result.InitSql
			originalResult := stage1Result.ExecResult

			// 2. Stage2 mutation
			stage2Result := MutateAllAndExec(originalSql, tc.seed, conn)
			if stage2Result.Err != nil {
				t.Fatalf("Stage2 error: %v", stage2Result.Err)
			}

			t.Logf("Total mutations: %d", len(stage2Result.MutateUnits))

			// 3. Oracle check
			equivalenceCount := 0
			for _, mu := range stage2Result.MutateUnits {
				if mu.Err != nil {
					continue
				}
				if mu.ExecResult == nil || mu.ExecResult.Err != nil {
					continue
				}

				var check bool
				var oracleErr error
				if mu.IsEquivalence {
					equivalenceCount++
					check, oracleErr = oracle.CheckEquivalence(originalResult, mu.ExecResult)
				} else {
					check, oracleErr = oracle.Check(originalResult, mu.ExecResult, mu.IsUpper)
				}
				if oracleErr != nil {
					continue
				}
				if !check {
					t.Logf("BUG! mutation=%s isUpper=%v isEquivalence=%v sql=%s", mu.Name, mu.IsUpper, mu.IsEquivalence, mu.Sql)
				}
			}

			t.Logf("Result: %d mutations, %d equivalence mutations", len(stage2Result.MutateUnits), equivalenceCount)
		})
	}
}

// =====================================================================
// PostgreSQL integration test (PG EET mutations)
// =====================================================================

func getPostgreSQLConnector(t *testing.T) *connector.PostgreSQLConnector {
	t.Helper()
	host := os.Getenv("PG_HOST")
	portStr := os.Getenv("PG_PORT")
	username := os.Getenv("PG_USERNAME")
	password := os.Getenv("PG_PASSWORD")
	dbname := os.Getenv("PG_DBNAME")

	if host == "" {
		host = "localhost"
	}
	if portStr == "" {
		portStr = "5432"
	}
	if username == "" {
		username = "tpcc"
	}
	if password == "" {
		t.Skip("PG_PASSWORD not set, skipping PostgreSQL integration test")
	}
	if dbname == "" {
		dbname = "postgres"
	}

	port := 5432
	if p, err := parsePort(portStr); err == nil {
		port = p
	}

	conn, err := connector.NewPostgreSQLConnector(host, port, username, password, dbname)
	if err != nil {
		t.Fatalf("NewPostgreSQLConnector error: %v", err)
	}
	return conn
}

func setupPostgreSQLTable(t *testing.T, conn *connector.PostgreSQLConnector) {
	t.Helper()
	result := conn.ExecSQL("DROP TABLE IF EXISTS test_pg_company")
	if result.Err != nil {
		t.Logf("DROP error (may be ok): %v", result.Err)
	}
	result = conn.ExecSQL("CREATE TABLE test_pg_company (id INT PRIMARY KEY, name VARCHAR(100), age INT, city VARCHAR(50))")
	if result.Err != nil {
		t.Fatalf("CREATE TABLE error: %v", result.Err)
	}
	result = conn.ExecSQL("INSERT INTO test_pg_company VALUES (1, 'Alice', 18, 'a'), (2, 'Bob', 19, 'b'), (3, 'Charlie', 20, 'c'), (4, 'David', 19, 'c'), (5, 'Eve', 19, 'c'), (6, 'Frank', 18, 'b')")
	if result.Err != nil {
		t.Fatalf("INSERT error: %v", result.Err)
	}
}

// TestPostgreSQLEETIntegration: end-to-end EET mutation test on PostgreSQL
func TestPostgreSQLEETIntegration(t *testing.T) {
	conn := getPostgreSQLConnector(t)
	defer conn.Close()
	setupPostgreSQLTable(t, conn)

	testSQLs := []struct {
		name string
		sql  string
		seed int64
	}{
		// Basic WHERE clause (PG EET mutations)
		{"BasicWhere", "SELECT * FROM test_pg_company WHERE id > 0", 10001},
		// DeMorgan EET
		{"DeMorganAnd", "SELECT * FROM test_pg_company WHERE id > 0 AND age > 18", 10002},
		{"DeMorganOr", "SELECT * FROM test_pg_company WHERE id > 0 OR age > 18", 10003},
		// BETWEEN EET
		{"Between", "SELECT * FROM test_pg_company WHERE age BETWEEN 18 AND 20", 10004},
		// COALESCE EET
		{"Coalesce", "SELECT * FROM test_pg_company WHERE COALESCE(age, 0) > 18", 10005},
		// NULLIF EET
		{"Nullif", "SELECT * FROM test_pg_company WHERE NULLIF(age, 18) IS NOT NULL", 10006},
	}

	for _, tc := range testSQLs {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Println("==================================================")
			t.Logf("Testing SQL: %s", tc.sql)

			// 1. Stage1 preprocessing (PG)
			stage1Result := stage1.InitForPostgreSQLAndExec(tc.sql, conn)
			if stage1Result.Err != nil {
				t.Fatalf("Stage1 error: %v", stage1Result.Err)
			}
			if stage1Result.ExecResult.Err != nil {
				t.Fatalf("Stage1 exec error: %v", stage1Result.ExecResult.Err)
			}

			originalSql := stage1Result.InitSql
			originalResult := stage1Result.ExecResult

			// 2. Stage2 mutation (PG)
			stage2Result := MutateAllAndExecForPostgreSQL(originalSql, tc.seed, conn)
			if stage2Result.Err != nil {
				t.Fatalf("Stage2 error: %v", stage2Result.Err)
			}

			t.Logf("Total mutations: %d", len(stage2Result.MutateUnits))

			// 3. Oracle check
			equivalenceCount := 0
			for _, mu := range stage2Result.MutateUnits {
				if mu.Err != nil {
					continue
				}
				if mu.ExecResult == nil || mu.ExecResult.Err != nil {
					continue
				}

				var check bool
				var oracleErr error
				if mu.IsEquivalence {
					equivalenceCount++
					check, oracleErr = oracle.CheckEquivalence(originalResult, mu.ExecResult)
				} else {
					check, oracleErr = oracle.Check(originalResult, mu.ExecResult, mu.IsUpper)
				}
				if oracleErr != nil {
					continue
				}
				if !check {
					t.Logf("BUG! mutation=%s isUpper=%v isEquivalence=%v sql=%s", mu.Name, mu.IsUpper, mu.IsEquivalence, mu.Sql)
				}
			}

			t.Logf("Result: %d mutations, %d equivalence mutations", len(stage2Result.MutateUnits), equivalenceCount)
		})
	}
}

// Helper: parse port string to int
func parsePort(s string) (int, error) {
	var port int
	_, err := fmt.Sscanf(s, "%d", &port)
	return port, err
}