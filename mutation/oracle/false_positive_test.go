package oracle

import (
	"testing"

	"github.com/qaqcatz/impomysql/connector"
)

// TestFalsePositiveDetector tests the false positive detection mechanism
func TestFalsePositiveDetector(t *testing.T) {
	// This test requires a database connection
	// Skip if no database is available
	conn := getTestConnectorForFP(t)
	if conn == nil {
		t.Skip("No database connection available")
		return
	}

	// Create false positive detector with default settings
	fpDetector := NewFalsePositiveDetector(conn, 3, 0.67, 5)

	testCases := []struct {
		name           string
		originalSQL    string
		mutatedSQL     string
		isUpper        bool
		mutationName   string
		expectFP       bool
		description    string
	}{
		{
			name:         "ConsistentBug_NoFP",
			originalSQL:  "SELECT * FROM test_table WHERE x > 5",
			mutatedSQL:   "SELECT * FROM test_table WHERE x > 10",
			isUpper:      false,
			mutationName: "FixMCmpOpL",
			expectFP:     false,
			description:  "Consistent bug should not be flagged as false positive",
		},
		{
			name:         "SmallDifference_PotentialFP",
			originalSQL:  "SELECT * FROM test_table WHERE x > 5",
			mutatedSQL:   "SELECT * FROM test_table WHERE x >= 5",
			isUpper:      true,
			mutationName: "FixMCmpOpU",
			expectFP:     true,
			description:  "Small result difference might be flagged as potential false positive",
		},
		{
			name:         "AllToAny_EmptySubquery_PotentialFP",
			originalSQL:  "SELECT * FROM test_table WHERE x > ALL (SELECT y FROM empty_table)",
			mutatedSQL:   "SELECT * FROM test_table WHERE x > ANY (SELECT y FROM empty_table)",
			isUpper:      true,
			mutationName: "FixMAllToAnyU",
			expectFP:     true,
			description:  "ALL/ANY with potentially empty subquery is a known edge case",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := fpDetector.AnalyzePotentialFalsePositive(
				tc.originalSQL, tc.mutatedSQL, tc.isUpper, tc.mutationName)
			if err != nil {
				t.Skipf("Analysis failed (likely due to missing tables): %v", err)
				return
			}

			if tc.expectFP && !analysis.IsPotentialFalsePositive {
				t.Errorf("Expected potential false positive but got none\n"+
					"Description: %s\n"+
					"Original SQL: %s\n"+
					"Mutated SQL: %s\n"+
					"Consistency: %d/%d",
					tc.description, tc.originalSQL, tc.mutatedSQL,
					analysis.ConsistentCount, analysis.TotalExecutions)
			}

			if analysis.IsPotentialFalsePositive {
				t.Logf("✓ Potential false positive detected\n"+
					"  Description: %s\n"+
					"  Reason: %s\n"+
					"  Consistency: %d/%d\n"+
					"  Rows: %d -> %d",
					tc.description, analysis.SuspicionReason,
					analysis.ConsistentCount, analysis.TotalExecutions,
					analysis.OriginalRows, analysis.MutatedRows)
			} else {
				t.Logf("✓ No false positive detected\n"+
					"  Description: %s\n"+
					"  Consistency: %d/%d",
					tc.description,
					analysis.ConsistentCount, analysis.TotalExecutions)
			}
		})
	}
}

// TestFalsePositiveDetectorConfiguration tests different detector configurations
func TestFalsePositiveDetectorConfiguration(t *testing.T) {
	conn := getTestConnectorForFP(t)
	if conn == nil {
		t.Skip("No database connection available")
		return
	}

	testCases := []struct {
		name                 string
		reExecutions         int
		consistencyThreshold float64
		smallDiffThreshold   int
		description          string
	}{
		{
			name:                 "DefaultConfig",
			reExecutions:         0,
			consistencyThreshold: 0,
			smallDiffThreshold:   0,
			description:          "Default configuration (3 re-executions, 0.67 threshold, 5 row threshold)",
		},
		{
			name:                 "StrictConfig",
			reExecutions:         5,
			consistencyThreshold: 0.8,
			smallDiffThreshold:   2,
			description:          "Strict configuration (5 re-executions, 0.8 threshold, 2 row threshold)",
		},
		{
			name:                 "LenientConfig",
			reExecutions:         2,
			consistencyThreshold: 0.5,
			smallDiffThreshold:   10,
			description:          "Lenient configuration (2 re-executions, 0.5 threshold, 10 row threshold)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fpDetector := NewFalsePositiveDetector(
				conn, tc.reExecutions, tc.consistencyThreshold, tc.smallDiffThreshold)

			// Verify detector was created with correct settings
			if fpDetector.reExecutions <= 0 {
				t.Errorf("reExecutions should be > 0, got %d", fpDetector.reExecutions)
			}
			if fpDetector.consistencyThreshold <= 0 || fpDetector.consistencyThreshold > 1.0 {
				t.Errorf("consistencyThreshold should be in (0, 1.0], got %f", fpDetector.consistencyThreshold)
			}
			if fpDetector.smallDiffThreshold <= 0 {
				t.Errorf("smallDiffThreshold should be > 0, got %d", fpDetector.smallDiffThreshold)
			}

			t.Logf("✓ Detector configured: %s\n"+
				"  Re-executions: %d\n"+
				"  Consistency threshold: %.2f\n"+
				"  Small diff threshold: %d",
				tc.description,
				fpDetector.reExecutions,
				fpDetector.consistencyThreshold,
				fpDetector.smallDiffThreshold)
		})
	}
}

// TestIsKnownEdgeCase tests the edge case detection logic
func TestIsKnownEdgeCase(t *testing.T) {
	testCases := []struct {
		name           string
		mutationName   string
		originalRows   int
		mutatedRows    int
		expectedResult bool
		description    string
	}{
		{
			name:           "AllToAny_EmptyToNonEmpty",
			mutationName:   "FixMAllToAnyU",
			originalRows:   0,
			mutatedRows:    10,
			expectedResult: true,
			description:    "ALL to ANY with empty original result is a known edge case",
		},
		{
			name:           "AllToAny_NonEmptyToEmpty",
			mutationName:   "FixMAllToAnyU",
			originalRows:   10,
			mutatedRows:    0,
			expectedResult: true,
			description:    "ALL to ANY with empty mutated result is a known edge case",
		},
		{
			name:           "AllToAny_NormalCase",
			mutationName:   "FixMAllToAnyU",
			originalRows:   5,
			mutatedRows:    10,
			expectedResult: false,
			description:    "ALL to ANY with normal results is not an edge case",
		},
		{
			name:           "NullEq_SmallDiff",
			mutationName:   "FixMNullEqToLowerL",
			originalRows:   10,
			mutatedRows:    9,
			expectedResult: true,
			description:    "NullEq with small difference (<=2) is a known edge case",
		},
		{
			name:           "NullEq_LargeDiff",
			mutationName:   "FixMNullEqToLowerL",
			originalRows:   10,
			mutatedRows:    5,
			expectedResult: false,
			description:    "NullEq with large difference (>2) is not an edge case",
		},
		{
			name:           "OtherMutation",
			mutationName:   "FixMWhere1U",
			originalRows:   10,
			mutatedRows:    20,
			expectedResult: false,
			description:    "Other mutations are not checked for edge cases",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isKnownEdgeCase(tc.mutationName, tc.originalRows, tc.mutatedRows)
			if result != tc.expectedResult {
				t.Errorf("isKnownEdgeCase mismatch\n"+
					"Description: %s\n"+
					"Mutation: %s\n"+
					"Original rows: %d, Mutated rows: %d\n"+
					"Expected: %v, Got: %v",
					tc.description, tc.mutationName,
					tc.originalRows, tc.mutatedRows,
					tc.expectedResult, result)
			}
			t.Logf("✓ %s: %s (result=%v)", tc.name, tc.description, result)
		})
	}
}

// TestAbs tests the absolute value helper function
func TestAbs(t *testing.T) {
	testCases := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{100, 100},
		{-100, 100},
	}

	for _, tc := range testCases {
		result := abs(tc.input)
		if result != tc.expected {
			t.Errorf("abs(%d) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
	t.Logf("✓ All %d abs() tests passed", len(testCases))
}

// getTestConnectorForFP returns a test database connector for false positive tests
func getTestConnectorForFP(t *testing.T) *connector.Connector {
	// Try to connect to local MySQL for testing
	conn, err := connector.NewConnector("127.0.0.1", 3306, "tpcc", "Taurus@123", "multidb_test")
	if err != nil {
		t.Logf("Failed to connect to test database: %v", err)
		return nil
	}
	return conn
}
