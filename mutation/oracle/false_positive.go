package oracle

import (
	"time"

	"github.com/qaqcatz/impomysql/connector"
)

// FalsePositiveDetector: Detects potential false positives by re-executing queries
// and checking for consistency. A false positive occurs when:
// 1. The bug is not consistently reproducible (non-deterministic behavior)
// 2. The result difference is very small (possibly due to floating point precision)
// 3. The mutation involves known edge cases (e.g., empty subqueries with ALL/ANY)
type FalsePositiveDetector struct {
	conn            *connector.Connector
	reExecutions    int     // Number of times to re-execute for verification
	consistencyThreshold float64 // Threshold for considering a bug consistent (0.0-1.0)
	smallDiffThreshold int     // Threshold for "small" result difference
}

// NewFalsePositiveDetector: Create a new false positive detector
// reExecutions: number of times to re-execute queries (default: 3)
// consistencyThreshold: minimum ratio of consistent reproductions (default: 0.67, i.e., 2/3)
// smallDiffThreshold: row count difference threshold for "small" difference (default: 5)
func NewFalsePositiveDetector(conn *connector.Connector, reExecutions int, consistencyThreshold float64, smallDiffThreshold int) *FalsePositiveDetector {
	if reExecutions <= 0 {
		reExecutions = 3
	}
	if consistencyThreshold <= 0 || consistencyThreshold > 1.0 {
		consistencyThreshold = 0.67
	}
	if smallDiffThreshold <= 0 {
		smallDiffThreshold = 5
	}
	return &FalsePositiveDetector{
		conn:                 conn,
		reExecutions:         reExecutions,
		consistencyThreshold: consistencyThreshold,
		smallDiffThreshold:   smallDiffThreshold,
	}
}

// FalsePositiveAnalysis: Result of false positive analysis
type FalsePositiveAnalysis struct {
	IsPotentialFalsePositive bool   // Whether this is suspected as a false positive
	SuspicionReason          string // Reason for suspicion
	ConsistentCount          int    // Number of consistent reproductions
	TotalExecutions          int    // Total number of executions (including original)
	OriginalRows             int    // Row count from original execution
	MutatedRows              int    // Row count from mutated execution
}

// AnalyzePotentialFalsePositive: Analyze whether a detected bug might be a false positive
// This is called after a bug is detected to verify if it's real or a false positive
func (fpd *FalsePositiveDetector) AnalyzePotentialFalsePositive(
	originalSql string,
	mutatedSql string,
	isUpper bool,
	mutationName string,
) (*FalsePositiveAnalysis, error) {
	analysis := &FalsePositiveAnalysis{
		IsPotentialFalsePositive: false,
		ConsistentCount:          1, // Original detection counts as 1
		TotalExecutions:          1 + fpd.reExecutions,
	}

	// Execute original query to get baseline
	originalResult := fpd.conn.ExecSQL(originalSql)
	if originalResult.Err != nil {
		return nil, originalResult.Err
	}
	analysis.OriginalRows = len(originalResult.Rows)

	// Execute mutated query to get baseline
	mutatedResult := fpd.conn.ExecSQL(mutatedSql)
	if mutatedResult.Err != nil {
		return nil, mutatedResult.Err
	}
	analysis.MutatedRows = len(mutatedResult.Rows)

	// Check 1: Very small result difference
	rowDiff := abs(analysis.OriginalRows - analysis.MutatedRows)
	if rowDiff <= fpd.smallDiffThreshold && rowDiff > 0 {
		analysis.IsPotentialFalsePositive = true
		analysis.SuspicionReason = "Very small result difference (<= " + string(rune(fpd.smallDiffThreshold)) + " rows)"
	}

	// Check 2: Known edge cases for specific mutations
	if isKnownEdgeCase(mutationName, analysis.OriginalRows, analysis.MutatedRows) {
		analysis.IsPotentialFalsePositive = true
		if analysis.SuspicionReason != "" {
			analysis.SuspicionReason += "; "
		}
		analysis.SuspicionReason += "Known edge case for mutation " + mutationName
	}

	// Check 3: Consistency verification through re-execution
	consistentBugCount := 0
	for i := 0; i < fpd.reExecutions; i++ {
		// Re-execute both queries
		origReExec := fpd.conn.ExecSQL(originalSql)
		if origReExec.Err != nil {
			continue
		}

		mutReExec := fpd.conn.ExecSQL(mutatedSql)
		if mutReExec.Err != nil {
			continue
		}

		// Check if the bug is still present
		check, err := Check(origReExec, mutReExec, isUpper)
		if err != nil {
			continue
		}

		// If check returns false, the bug is still present (consistent)
		if !check {
			consistentBugCount++
		}
	}

	analysis.ConsistentCount += consistentBugCount
	consistencyRatio := float64(analysis.ConsistentCount) / float64(analysis.TotalExecutions)

	// If bug is not consistently reproduced, it's likely a false positive
	if consistencyRatio < fpd.consistencyThreshold {
		analysis.IsPotentialFalsePositive = true
		if analysis.SuspicionReason != "" {
			analysis.SuspicionReason += "; "
		}
		analysis.SuspicionReason += "Bug not consistently reproduced (" +
			string(rune(analysis.ConsistentCount)) + "/" + string(rune(analysis.TotalExecutions)) + " times)"
	}

	return analysis, nil
}

// isKnownEdgeCase: Check if the mutation involves known edge cases that might cause false positives
func isKnownEdgeCase(mutationName string, originalRows int, mutatedRows int) bool {
	// Empty subquery edge cases for ALL/ANY mutations
	if mutationName == "FixMAllToAnyU" || mutationName == "FixMAnyToAllL" {
		// If original query returns all rows and mutated returns none (or vice versa),
		// this might be due to empty subquery edge case
		if (originalRows > 0 && mutatedRows == 0) || (originalRows == 0 && mutatedRows > 0) {
			return true
		}
	}

	// NULL handling edge cases
	if mutationName == "FixMNullEqToLowerL" {
		// If the difference is very small, might be due to NULL handling
		if abs(originalRows-mutatedRows) <= 2 {
			return true
		}
	}

	return false
}

// abs: Absolute value helper
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GenerateFalsePositiveReport: Generate a detailed report for a potential false positive
func GenerateFalsePositiveReport(
	bugId int,
	sqlId int,
	mutationName string,
	originalSql string,
	mutatedSql string,
	analysis *FalsePositiveAnalysis,
) map[string]interface{} {
	return map[string]interface{}{
		"bugId":                    bugId,
		"sqlId":                    sqlId,
		"mutationName":             mutationName,
		"originalSql":              originalSql,
		"mutatedSql":               mutatedSql,
		"isPotentialFalsePositive": analysis.IsPotentialFalsePositive,
		"suspicionReason":          analysis.SuspicionReason,
		"consistentCount":          analysis.ConsistentCount,
		"totalExecutions":          analysis.TotalExecutions,
		"originalRows":             analysis.OriginalRows,
		"mutatedRows":              analysis.MutatedRows,
		"timestamp":                time.Now().Format(time.RFC3339),
	}
}
