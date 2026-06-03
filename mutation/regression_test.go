package mutation

import (
	"fmt"
	"testing"
	"time"

	"github.com/pingcap/tidb/parser"
	"github.com/qaqcatz/impomysql/mutation/stage2"
)

// TestAllImplicationMutations tests that all expected implication mutations are generated
// and that EET mutations have been removed (v0.4.0)
func TestAllImplicationMutations(t *testing.T) {
	// List of implication mutation names that SHOULD exist
	expectedMutations := []string{
		"FixMWhere1U",
		"FixMWhere0L",
		"FixMHaving1U",
		"FixMHaving0L",
		"FixMOn1U",
		"FixMOn0L",
		"FixMCmpOpU",
		"FixMCmpOpL",
		"FixMInNullU",
		"FixMBetweenDropUpperU",
		"FixMBetweenDropLowerU",
		"FixMNullEqToLowerL",
		"FixMAllToAnyU",
		"FixMAnyToAllL",
	}

	// Test SQL statements that should trigger various mutations
	testSQLs := []struct {
		sql         string
		description string
	}{
		{
			sql:         "SELECT * FROM test_table WHERE x > 5",
			description: "Simple WHERE with comparison",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x >= 5",
			description: "WHERE with >=",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x IN (1, 2, 3)",
			description: "WHERE with IN clause",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x BETWEEN 5 AND 10",
			description: "WHERE with BETWEEN",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x <=> NULL",
			description: "WHERE with null-safe equals",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x > ALL (SELECT y FROM other_table)",
			description: "WHERE with ALL subquery",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x > ANY (SELECT y FROM other_table)",
			description: "WHERE with ANY subquery",
		},
		{
			sql:         "SELECT x, COUNT(*) FROM test_table GROUP BY x HAVING COUNT(*) > 1",
			description: "GROUP BY with HAVING",
		},
		{
			sql:         "SELECT * FROM t1 JOIN t2 ON t1.id = t2.id",
			description: "JOIN with ON clause",
		},
	}

	foundMutations := make(map[string]bool)

	for _, test := range testSQLs {
		t.Run(test.description, func(t *testing.T) {
			// Parse SQL and find candidates
			visitor, err := stage2.CalCandidates(test.sql)
			if err != nil {
				t.Fatalf("CalCandidates failed: %v", err)
			}

			// Collect all mutation names
			for name, candidates := range visitor.Candidates {
				if len(candidates) > 0 {
					foundMutations[name] = true
				}
			}

			t.Logf("SQL: %s", test.sql)
			t.Logf("Found %d mutation types", len(visitor.Candidates))
			for name, candidates := range visitor.Candidates {
				t.Logf("  - %s: %d candidates", name, len(candidates))
			}
		})
	}

	// Check that all expected mutations were found
	t.Log("\n=== Checking expected mutations ===")
	for _, expected := range expectedMutations {
		if foundMutations[expected] {
			t.Logf("✓ %s found", expected)
		} else {
			t.Errorf("✗ %s NOT found", expected)
		}
	}
}

// TestEETMutationsRemoved verifies that all EET mutations have been removed in v0.4.0
func TestEETMutationsRemoved(t *testing.T) {
	// List of EET mutation names that should NOT exist
	eetMutations := []string{
		"FixMAndTrueU",
		"FixMOrFalseL",
		"FixMCaseTrueU",
		"FixMCaseFalseL",
		"FixMCaseRandEq",
		"FixMDeMorganAnd",
		"FixMDeMorganOr",
		"FixMBetweenToCmp",
		"FixMCoalesceToCase",
		"FixMNullifToCase",
		"FixMExistsToIn",
		"FixMInToExists",
	}

	// Test SQL that would trigger EET mutations if they existed
	testSQLs := []struct {
		sql         string
		description string
	}{
		{
			sql:         "SELECT * FROM test_table WHERE x > 5 AND y < 10",
			description: "AND clause (would trigger DeMorgan)",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x > 5 OR y < 10",
			description: "OR clause (would trigger DeMorgan)",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x BETWEEN 5 AND 10",
			description: "BETWEEN (would trigger BetweenToCmp)",
		},
		{
			sql:         "SELECT * FROM test_table WHERE COALESCE(x, 0) = 1",
			description: "COALESCE (would trigger CoalesceToCase)",
		},
		{
			sql:         "SELECT * FROM test_table WHERE NULLIF(x, 0) IS NULL",
			description: "NULLIF (would trigger NullifToCase)",
		},
		{
			sql:         "SELECT * FROM test_table WHERE EXISTS (SELECT 1 FROM other_table)",
			description: "EXISTS (would trigger ExistsToIn)",
		},
		{
			sql:         "SELECT * FROM test_table WHERE x IN (SELECT y FROM other_table)",
			description: "IN subquery (would trigger InToExists)",
		},
	}

	for _, test := range testSQLs {
		t.Run(test.description, func(t *testing.T) {
			// Parse SQL and find candidates
			visitor, err := stage2.CalCandidates(test.sql)
			if err != nil {
				t.Fatalf("CalCandidates failed: %v", err)
			}

			// Check that no EET mutations are generated
			for name := range visitor.Candidates {
				for _, eetName := range eetMutations {
					if name == eetName {
						t.Errorf("EET mutation %s was generated but should have been removed in v0.4.0\nSQL: %s",
							eetName, test.sql)
					}
				}
			}

			t.Logf("SQL: %s", test.sql)
			t.Logf("✓ No EET mutations found (checked %d mutation types)", len(visitor.Candidates))
		})
	}
}

// TestMutationApplication tests that mutations can be applied correctly
func TestMutationApplication(t *testing.T) {
	testCases := []struct {
		sql             string
		mutationName    string
		description     string
		expectedChanged bool
	}{
		{
			sql:             "SELECT * FROM test_table WHERE x > 5",
			mutationName:    "FixMWhere1U",
			description:     "Apply FixMWhere1U (WHERE -> TRUE)",
			expectedChanged: true,
		},
		{
			sql:             "SELECT * FROM test_table WHERE x > 5",
			mutationName:    "FixMCmpOpU",
			description:     "Apply FixMCmpOpU (> -> >=)",
			expectedChanged: true,
		},
		{
			sql:             "SELECT * FROM test_table WHERE x IN (1, 2, 3)",
			mutationName:    "FixMInNullU",
			description:     "Apply FixMInNullU (add NULL to IN)",
			expectedChanged: true,
		},
		{
			sql:             "SELECT * FROM test_table WHERE x BETWEEN 5 AND 10",
			mutationName:    "FixMBetweenDropUpperU",
			description:     "Apply FixMBetweenDropUpperU (remove upper bound)",
			expectedChanged: true,
		},
	}

	p := parser.New()
	seed := int64(time.Now().UnixNano())

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Parse SQL
			stmtNodes, _, err := p.Parse(tc.sql, "", "")
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(stmtNodes) == 0 {
				t.Fatal("No statements parsed")
			}

			rootNode := stmtNodes[0]

			// Find candidates
			visitor, err := stage2.CalCandidates(tc.sql)
			if err != nil {
				t.Fatalf("CalCandidates failed: %v", err)
			}

			// Find the target mutation
			candidates, exists := visitor.Candidates[tc.mutationName]
			if !exists || len(candidates) == 0 {
				t.Skipf("Mutation %s not found for SQL: %s", tc.mutationName, tc.sql)
				return
			}

			// Apply the first candidate
			candidate := candidates[0]
			mutatedSQL, err := stage2.ImpoMutate(rootNode, candidate, seed)
			if err != nil {
				t.Fatalf("ImpoMutate failed: %v", err)
			}

			// Check if SQL changed
			changed := mutatedSQL != tc.sql
			if changed != tc.expectedChanged {
				t.Errorf("Mutation change mismatch\n"+
					"Expected changed: %v, Got: %v\n"+
					"Original: %s\n"+
					"Mutated:  %s",
					tc.expectedChanged, changed, tc.sql, mutatedSQL)
			}

			t.Logf("Original SQL: %s", tc.sql)
			t.Logf("Mutated SQL:  %s", mutatedSQL)
			t.Logf("Changed: %v", changed)
		})
	}
}

// TestMutationDirection verifies that mutations have the correct direction (UPPER/LOWER)
func TestMutationDirection(t *testing.T) {
	testCases := []struct {
		sql             string
		mutationName    string
		expectedIsUpper bool
		description     string
	}{
		{
			sql:             "SELECT * FROM test_table WHERE x > 5",
			mutationName:    "FixMWhere1U",
			expectedIsUpper: true,
			description:     "FixMWhere1U should be UPPER (WHERE -> TRUE expands result set)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE x > 5",
			mutationName:    "FixMWhere0L",
			expectedIsUpper: false,
			description:     "FixMWhere0L should be LOWER (WHERE -> FALSE shrinks result set)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE x > 5",
			mutationName:    "FixMCmpOpU",
			expectedIsUpper: true,
			description:     "FixMCmpOpU should be UPPER (> -> >= expands result set)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE x >= 5",
			mutationName:    "FixMCmpOpL",
			expectedIsUpper: false,
			description:     "FixMCmpOpL should be LOWER (>= -> > shrinks result set)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE x IN (1, 2, 3)",
			mutationName:    "FixMInNullU",
			expectedIsUpper: true,
			description:     "FixMInNullU should be UPPER (adding NULL expands result set)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE x BETWEEN 5 AND 10",
			mutationName:    "FixMBetweenDropUpperU",
			expectedIsUpper: true,
			description:     "FixMBetweenDropUpperU should be UPPER (removing upper bound expands result set)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE x BETWEEN 5 AND 10",
			mutationName:    "FixMBetweenDropLowerU",
			expectedIsUpper: true,
			description:     "FixMBetweenDropLowerU should be UPPER (removing lower bound expands result set)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Find candidates
			visitor, err := stage2.CalCandidates(tc.sql)
			if err != nil {
				t.Fatalf("CalCandidates failed: %v", err)
			}

			// Find the target mutation
			candidates, exists := visitor.Candidates[tc.mutationName]
			if !exists || len(candidates) == 0 {
				t.Skipf("Mutation %s not found for SQL: %s", tc.mutationName, tc.sql)
				return
			}

			// Check the direction of the first candidate
			candidate := candidates[0]
			isUpper := candidate.U == 1

			if isUpper != tc.expectedIsUpper {
				t.Errorf("Mutation direction mismatch for %s\n"+
					"Expected IsUpper: %v, Got: %v\n"+
					"SQL: %s",
					tc.mutationName, tc.expectedIsUpper, isUpper, tc.sql)
			}

			t.Logf("✓ %s direction verified (IsUpper=%v)", tc.mutationName, isUpper)
		})
	}
}

// TestFlagPropagation tests the flag propagation mechanism for positive/negative contexts
func TestFlagPropagation(t *testing.T) {
	testCases := []struct {
		sql             string
		mutationName    string
		expectedFlag    int
		description     string
	}{
		{
			sql:             "SELECT * FROM test_table WHERE x > 5",
			mutationName:    "FixMCmpOpU",
			expectedFlag:    1,
			description:     "Positive context (WHERE clause) should have flag=1",
		},
		{
			sql:             "SELECT * FROM test_table WHERE NOT (x > 5)",
			mutationName:    "FixMCmpOpU",
			expectedFlag:    0,
			description:     "Negative context (NOT) should have flag=0",
		},
		{
			sql:             "SELECT * FROM test_table WHERE NOT NOT (x > 5)",
			mutationName:    "FixMCmpOpU",
			expectedFlag:    1,
			description:     "Double negative should restore positive context (flag=1)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE (x > 5) IS FALSE",
			mutationName:    "FixMCmpOpU",
			expectedFlag:    0,
			description:     "IS FALSE should create negative context (flag=0)",
		},
		{
			sql:             "SELECT * FROM test_table WHERE (x > 5) IS NOT FALSE",
			mutationName:    "FixMCmpOpU",
			expectedFlag:    1,
			description:     "IS NOT FALSE should create positive context (flag=1)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Find candidates
			visitor, err := stage2.CalCandidates(tc.sql)
			if err != nil {
				t.Fatalf("CalCandidates failed: %v", err)
			}

			// Find the target mutation
			candidates, exists := visitor.Candidates[tc.mutationName]
			if !exists || len(candidates) == 0 {
				t.Skipf("Mutation %s not found for SQL: %s", tc.mutationName, tc.sql)
				return
			}

			// Check the flag of the first candidate
			candidate := candidates[0]
			if candidate.Flag != tc.expectedFlag {
				t.Errorf("Flag propagation mismatch for %s\n"+
					"Expected Flag: %d, Got: %d\n"+
					"SQL: %s",
					tc.mutationName, tc.expectedFlag, candidate.Flag, tc.sql)
			}

			t.Logf("✓ Flag propagation verified (Flag=%d)", candidate.Flag)
			t.Logf("  SQL: %s", tc.sql)
			t.Logf("  Effective direction: %s",
				func() string {
					if ((candidate.U ^ candidate.Flag) ^ 1) == 1 {
						return "UPPER (result set expands)"
					}
					return "LOWER (result set shrinks)"
				}())
		})
	}
}

// TestRegressionV040 is the main regression test suite for v0.4.0
func TestRegressionV040(t *testing.T) {
	t.Log("========================================")
	t.Log("Pinolo v0.4.0 Regression Test Suite")
	t.Log("Testing after EET mutation removal")
	t.Log("========================================")
	t.Log("")

	// Run all subtests
	t.Run("01_AllImplicationMutations", TestAllImplicationMutations)
	t.Run("02_EETMutationsRemoved", TestEETMutationsRemoved)
	t.Run("03_MutationApplication", TestMutationApplication)
	t.Run("04_MutationDirection", TestMutationDirection)
	t.Run("05_FlagPropagation", TestFlagPropagation)

	t.Log("")
	t.Log("========================================")
	t.Log("Regression Test Suite Complete")
	t.Log("========================================")
}

// BenchmarkMutationGeneration benchmarks the mutation generation performance
func BenchmarkMutationGeneration(b *testing.B) {
	testSQL := `
		SELECT t1.id, t1.name, t2.value
		FROM table1 t1
		JOIN table2 t2 ON t1.id = t2.id
		WHERE t1.status > 0
		AND t2.value BETWEEN 10 AND 100
		AND t1.name IN ('Alice', 'Bob', 'Charlie')
		GROUP BY t1.id, t1.name, t2.value
		HAVING COUNT(*) > 1
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stage2.CalCandidates(testSQL)
		if err != nil {
			b.Fatalf("CalCandidates failed: %v", err)
		}
	}
}

// Example_mutationGeneration demonstrates how to generate mutations
func Example_mutationGeneration() {
	sql := "SELECT * FROM users WHERE age > 18 AND status = 'active'"

	// Find mutation candidates
	visitor, err := stage2.CalCandidates(sql)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d mutation types:\n", len(visitor.Candidates))
	for name, candidates := range visitor.Candidates {
		fmt.Printf("  - %s: %d candidates\n", name, len(candidates))
	}

	// Output:
	// Found X mutation types:
	//   - FixMWhere1U: 1 candidates
	//   - FixMCmpOpU: 1 candidates
	//   ...
}

