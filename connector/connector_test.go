package connector

import (
	"os"
	"strconv"
	"testing"
)

// Connection config from environment variables (with defaults for local dev)
// Env vars: TEST_DB_HOST, TEST_DB_PORT, TEST_DB_USERNAME, TEST_DB_PASSWORD, TEST_DB_NAME
// Example: docker run -itd --name test -p 13306:3306 -e MYSQL_ROOT_PASSWORD=your_password mysql
var (
	testHost     = getEnvOrDefault("TEST_DB_HOST", "127.0.0.1")
	testPort     = getEnvIntOrDefault("TEST_DB_PORT", 13306)
	testUsername = getEnvOrDefault("TEST_DB_USERNAME", "root")
	testPassword = getEnvOrDefault("TEST_DB_PASSWORD", "your_password")
	testDBname   = getEnvOrDefault("TEST_DB_NAME", "TEST")
)

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvIntOrDefault(key, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func TestConnector_ExecSQL(t *testing.T) {
	conn, err := NewConnector(testHost, testPort, testUsername, testPassword, testDBname)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// create table
	result := conn.ExecSQL("DROP TABLE IF EXISTS T")
	if result.Err != nil {
		t.Fatalf("%+v", result.Err)
	} else {
		t.Log(result.ToString())
	}
	result = conn.ExecSQL("CREATE TABLE T(ID INT, NAME TEXT, X DOUBLE)")
	if result.Err != nil {
		t.Fatalf("%+v", result.Err)
	} else {
		t.Log(result.ToString())
	}

	for i := 0; i < 3; i++ {
		result := conn.ExecSQL("INSERT INTO T VALUES ("+strconv.Itoa(i)+", '"+string(rune(i+'A'))+"', -" + strconv.Itoa(i) + ")")
		if result.Err != nil {
			t.Fatalf("%+v", result.Err)
		} else {
			t.Log(result.ToString())
		}
	}

	// normal
	result = conn.ExecSQL("SELECT 1+2, ID, NAME, X FROM T;")
	if result.Err != nil {
		t.Fatalf("%+v", result.Err)
	} else {
		t.Log(result.ToString())
	}

	// error
	result = conn.ExecSQL("select 9223372036854775807 + 1")
	if result.Err != nil {
		t.Logf("%+v", result.Err)
	} else {
		t.Fatal("must error!")
	}

	testSql := "SELECT (~DEGREES(0.9219647951826007)|FORMAT_BYTES(`f1`)), (~1^`f1`) FROM (SELECT (X^_UTF8MB4'do'-X) AS `f1` FROM (SELECT X FROM T) AS `t1`) AS `t2`;"

	result = conn.ExecSQL(testSql)
	if result.Err != nil {
		t.Logf("%+v", result.Err)
	} else {
		t.Fatal("must error!")
	}

	errCode, err := result.GetErrorCode()
	if err == nil {
		t.Log("error code = " ,errCode)
	} else {
		t.Fatalf("%+v", err)
	}

	// result cmp
	result1 := conn.ExecSQL("SELECT ID, NAME, X FROM T;")
	if result1.Err != nil {
		t.Fatalf("%+v", result1.Err)
	}
	result2 := conn.ExecSQL("SELECT ID, NAME, X FROM T WHERE ID != 1;")
	if result2.Err != nil {
		t.Fatalf("%+v", result2.Err)
	}
	result3 := conn.ExecSQL("SELECT ID, NAME, X FROM T WHERE ID != 2;")
	if result3.Err != nil {
		t.Fatalf("%+v", result3.Err)
	}

	cmp, err := result1.CMP(result1)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if cmp != 0 {
		t.Fatal("must 0")
	}

	cmp, err = result1.CMP(result2)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if cmp != 1 {
		t.Fatal("must 1")
	}

	cmp, err = result2.CMP(result1)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if cmp != -1 {
		t.Fatal("must -1")
	}

	cmp, err = result3.CMP(result2)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if cmp != 2 {
		t.Fatal("must 2")
	}
}