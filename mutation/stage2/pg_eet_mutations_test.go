package stage2

import (
	 pgquery "github.com/pganalyze/pg_query_go/v6"
	"testing"
)

// TestPgEETMutations: test PostgreSQL EET transformation mutations
func TestPgEETMutations(t *testing.T) {
	// Test SQL with WHERE clause
	sql := "SELECT * FROM t0 WHERE id > 0"

	// Parse and check
	result, err := pgquery.Parse(sql)
	if err != nil {
		t.Fatalf("pgquery.Parse error: %v", err)
	}

	// Debug: check if WHERE clause exists
	for _, stmt := range result.Stmts {
		if stmt.Stmt != nil {
			sel := stmt.Stmt.GetSelectStmt()
			if sel != nil {
				t.Logf("SelectStmt found, WhereClause != nil: %v", sel.WhereClause != nil)
			}
		}
	}

	v, err := CalCandidatesForPostgreSQL(sql)
	if err != nil {
		t.Fatalf("CalCandidatesForPostgreSQL error: %v", err)
	}

	// Debug: print all candidates
	t.Logf("Total candidates found: %d", len(v.Candidates))
	for name, cands := range v.Candidates {
		t.Logf("  %s: %d candidates", name, len(cands))
	}

	mutateResult := MutateAllForPostgreSQL(sql, 12345)
	if mutateResult.Err != nil {
		t.Fatalf("MutateAllForPostgreSQL error: %v", mutateResult.Err)
	}

	// Check EET mutations are present
	eetMutations := []string{
		FixMAndTrueU_Pg,
		FixMOrFalseL_Pg,
		FixMCaseTrueU_Pg,
		FixMCaseFalseL_Pg,
		FixMCaseRandEq_Pg,
	}

	foundEET := make(map[string]bool)
	for _, mu := range mutateResult.MutateUnits {
		for _, eetName := range eetMutations {
			if mu.Name == eetName {
				foundEET[eetName] = true
				t.Logf("EET mutation %s: %s", eetName, mu.Sql)
				if mu.Err != nil {
					t.Errorf("EET mutation %s error: %v", eetName, mu.Err)
				}
			}
		}
	}

	for _, eetName := range eetMutations {
		if !foundEET[eetName] {
			t.Errorf("EET mutation %s not found", eetName)
		}
	}

	// Verify mutation count
	t.Logf("Total mutations: %d", len(mutateResult.MutateUnits))
}

// TestPgEETMutationTautology: verify tautology wrapping produces valid PostgreSQL SQL
func TestPgEETMutationTautology(t *testing.T) {
	sql := "SELECT * FROM t0 WHERE id > 0"

	v, err := CalCandidatesForPostgreSQL(sql)
	if err != nil {
		t.Fatalf("CalCandidatesForPostgreSQL error: %v", err)
	}

	candidates := v.Candidates[FixMAndTrueU_Pg]
	if len(candidates) == 0 {
		t.Fatal("No FixMAndTrueU_Pg candidates")
	}

	mutatedSql, err := PgImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("PgImpoMutate error: %v", err)
	}

	t.Logf("Original: %s", sql)
	t.Logf("Mutated:  %s", mutatedSql)

	// Verify the mutated SQL contains AND
	if !containsSubstringPg(mutatedSql, "AND") {
		t.Errorf("Tautology mutation should contain AND")
	}

	// Verify the mutated SQL contains OR (for tautology)
	if !containsSubstringPg(mutatedSql, "OR") {
		t.Errorf("Tautology mutation should contain OR")
	}

	// Verify the mutated SQL contains IS NULL
	if !containsSubstringPg(mutatedSql, "IS NULL") {
		t.Errorf("Tautology mutation should contain IS NULL")
	}
}

// TestPgEETMutationContradiction: verify contradiction wrapping produces valid PostgreSQL SQL
func TestPgEETMutationContradiction(t *testing.T) {
	sql := "SELECT * FROM t0 WHERE id > 0"

	v, err := CalCandidatesForPostgreSQL(sql)
	if err != nil {
		t.Fatalf("CalCandidatesForPostgreSQL error: %v", err)
	}

	candidates := v.Candidates[FixMOrFalseL_Pg]
	if len(candidates) == 0 {
		t.Fatal("No FixMOrFalseL_Pg candidates")
	}

	mutatedSql, err := PgImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("PgImpoMutate error: %v", err)
	}

	t.Logf("Original: %s", sql)
	t.Logf("Mutated:  %s", mutatedSql)

	// Verify the mutated SQL contains OR
	if !containsSubstringPg(mutatedSql, "OR") {
		t.Errorf("Contradiction mutation should contain OR")
	}

	// Verify the mutated SQL contains IS NOT NULL
	if !containsSubstringPg(mutatedSql, "IS NOT NULL") {
		t.Errorf("Contradiction mutation should contain IS NOT NULL")
	}
}

// TestPgEETMutationCase: verify CASE WHEN mutations produce valid PostgreSQL SQL
func TestPgEETMutationCase(t *testing.T) {
	sql := "SELECT * FROM t0 WHERE id > 0"

	v, err := CalCandidatesForPostgreSQL(sql)
	if err != nil {
		t.Fatalf("CalCandidatesForPostgreSQL error: %v", err)
	}

	// Test FixMCaseTrueU_Pg
	candidates := v.Candidates[FixMCaseTrueU_Pg]
	if len(candidates) == 0 {
		t.Fatal("No FixMCaseTrueU_Pg candidates")
	}

	mutatedSql, err := PgImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("PgImpoMutate FixMCaseTrueU_Pg error: %v", err)
	}

	t.Logf("FixMCaseTrueU_Pg: %s", mutatedSql)

	// Verify the mutated SQL contains CASE
	if !containsSubstringPg(mutatedSql, "CASE") {
		t.Errorf("CASE mutation should contain CASE")
	}

	// Verify the mutated SQL contains WHEN
	if !containsSubstringPg(mutatedSql, "WHEN") {
		t.Errorf("CASE mutation should contain WHEN")
	}

	// Test FixMCaseFalseL_Pg
	candidates = v.Candidates[FixMCaseFalseL_Pg]
	if len(candidates) == 0 {
		t.Fatal("No FixMCaseFalseL_Pg candidates")
	}

	mutatedSql, err = PgImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("PgImpoMutate FixMCaseFalseL_Pg error: %v", err)
	}

	t.Logf("FixMCaseFalseL_Pg: %s", mutatedSql)
}

// TestPgEETMutationRandEq: verify CASE WHEN rand THEN E ELSE E END
func TestPgEETMutationRandEq(t *testing.T) {
	sql := "SELECT * FROM t0 WHERE id > 0"

	v, err := CalCandidatesForPostgreSQL(sql)
	if err != nil {
		t.Fatalf("CalCandidatesForPostgreSQL error: %v", err)
	}

	candidates := v.Candidates[FixMCaseRandEq_Pg]
	if len(candidates) == 0 {
		t.Fatal("No FixMCaseRandEq_Pg candidates")
	}

	mutatedSql, err := PgImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("PgImpoMutate FixMCaseRandEq_Pg error: %v", err)
	}

	t.Logf("FixMCaseRandEq_Pg: %s", mutatedSql)

	// Verify the mutated SQL contains CASE
	if !containsSubstringPg(mutatedSql, "CASE") {
		t.Errorf("CASE mutation should contain CASE")
	}

	// Verify the mutated SQL contains ELSE (both branches are E)
	if !containsSubstringPg(mutatedSql, "ELSE") {
		t.Errorf("CASE mutation should contain ELSE")
	}
}

func containsSubstringPg(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}