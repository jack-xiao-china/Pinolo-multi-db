package stage2

import (
	"testing"
)

// TestEETMutations: test EET transformation mutations
func TestEETMutations(t *testing.T) {
	// Test SQL with WHERE clause
	sql := "SELECT * FROM COMPANY WHERE ID > 0"

	mutateResult := MutateAll(sql, 12345)
	if mutateResult.Err != nil {
		t.Fatalf("MutateAll error: %v", mutateResult.Err)
	}

	// Check EET mutations are present
	eetMutations := []string{
		FixMAndTrueU,
		FixMOrFalseL,
		FixMCaseTrueU,
		FixMCaseFalseL,
		FixMCaseRandEq,
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

// TestEETMutationTautology: verify tautology wrapping produces valid SQL
func TestEETMutationTautology(t *testing.T) {
	sql := "SELECT * FROM COMPANY WHERE ID > 0"

	v, err := CalCandidates(sql)
	if err != nil {
		t.Fatalf("CalCandidates error: %v", err)
	}

	candidates := v.Candidates[FixMAndTrueU]
	if len(candidates) == 0 {
		t.Fatal("No FixMAndTrueU candidates")
	}

	mutatedSql, err := ImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("ImpoMutate error: %v", err)
	}

	t.Logf("Original: %s", sql)
	t.Logf("Mutated:  %s", mutatedSql)

	// Verify the mutated SQL contains AND
	if !containsSubstring(mutatedSql, "AND") {
		t.Errorf("Tautology mutation should contain AND")
	}

	// Verify the mutated SQL contains OR (for tautology)
	if !containsSubstring(mutatedSql, "OR") {
		t.Errorf("Tautology mutation should contain OR")
	}
}

// TestEETMutationContradiction: verify contradiction wrapping produces valid SQL
func TestEETMutationContradiction(t *testing.T) {
	sql := "SELECT * FROM COMPANY WHERE ID > 0"

	v, err := CalCandidates(sql)
	if err != nil {
		t.Fatalf("CalCandidates error: %v", err)
	}

	candidates := v.Candidates[FixMOrFalseL]
	if len(candidates) == 0 {
		t.Fatal("No FixMOrFalseL candidates")
	}

	mutatedSql, err := ImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("ImpoMutate error: %v", err)
	}

	t.Logf("Original: %s", sql)
	t.Logf("Mutated:  %s", mutatedSql)

	// Verify the mutated SQL contains OR
	if !containsSubstring(mutatedSql, "OR") {
		t.Errorf("Contradiction mutation should contain OR")
	}
}

// TestEETMutationCase: verify CASE WHEN mutations produce valid SQL
func TestEETMutationCase(t *testing.T) {
	sql := "SELECT * FROM COMPANY WHERE ID > 0"

	v, err := CalCandidates(sql)
	if err != nil {
		t.Fatalf("CalCandidates error: %v", err)
	}

	// Test FixMCaseTrueU
	candidates := v.Candidates[FixMCaseTrueU]
	if len(candidates) == 0 {
		t.Fatal("No FixMCaseTrueU candidates")
	}

	mutatedSql, err := ImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("ImpoMutate FixMCaseTrueU error: %v", err)
	}

	t.Logf("FixMCaseTrueU: %s", mutatedSql)

	// Verify the mutated SQL contains CASE
	if !containsSubstring(mutatedSql, "CASE") {
		t.Errorf("CASE mutation should contain CASE")
	}

	// Test FixMCaseFalseL
	candidates = v.Candidates[FixMCaseFalseL]
	if len(candidates) == 0 {
		t.Fatal("No FixMCaseFalseL candidates")
	}

	mutatedSql, err = ImpoMutate(v.Root, candidates[0], 12345)
	if err != nil {
		t.Fatalf("ImpoMutate FixMCaseFalseL error: %v", err)
	}

	t.Logf("FixMCaseFalseL: %s", mutatedSql)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}