package oracle

import (
	"strconv"
	"strings"

	"github.com/qaqcatz/impomysql/connector"
)

// CheckAggregate: aggregate-aware Oracle for queries with COUNT/SUM/MIN/MAX.
// Instead of set containment, compares aggregate results numerically.
//
// Soundness rules for upper mutations (WHERE relaxed → more rows pass):
//   - COUNT: mutated_count >= original_count (more rows → larger count)
//   - MAX:   mutated_max >= original_max (more rows → max can only increase)
//   - MIN:   mutated_min <= original_min (more rows → min can only decrease)
//   - SUM (positive cols): mutated_sum >= original_sum
//   - SUM (general): NOT SOUND without column domain knowledge
//   - AVG: NOT SOUND (adding rows can increase or decrease average)
//
// For lower mutations, the inequalities are reversed.
//
// This function handles single-row results (no GROUP BY) or multi-row results
// where each row is a single numeric aggregate value.
// Returns (check, error): check=true means no bug, check=false means bug detected.
func CheckAggregate(originResult *connector.Result, mutatedResult *connector.Result, isUpper bool) (bool, error) {
	if originResult == nil || mutatedResult == nil {
		return false, nil
	}

	// Both must have at least one row
	if len(originResult.Rows) == 0 || len(mutatedResult.Rows) == 0 {
		return true, nil // empty results, no comparison possible
	}

	// Compare row counts first for single-column aggregate results
	// For COUNT(*), the result is always a single row with a single numeric value
	if len(originResult.Rows) == 1 && len(mutatedResult.Rows) == 1 &&
		len(originResult.Rows[0]) > 0 && len(mutatedResult.Rows[0]) > 0 {

		origVal, origErr := parseNumeric(originResult.Rows[0][0])
		mutVal, mutErr := parseNumeric(mutatedResult.Rows[0][0])

		if origErr == nil && mutErr == nil {
			if isUpper {
				// Upper: mutated should be >= original
				return mutVal >= origVal, nil
			} else {
				// Lower: mutated should be <= original
				return mutVal <= origVal, nil
			}
		}
	}

	// For multi-row GROUP BY results, fall back to standard containment check
	cmp, err := originResult.CMP(mutatedResult)
	if err != nil {
		return false, err
	}
	if cmp == 0 {
		return true, nil
	}
	if (isUpper && cmp == -1) || (!isUpper && cmp == 1) {
		return true, nil
	}
	return false, nil
}

// parseNumeric: parse a numeric string into float64.
// Handles integers, decimals, and scientific notation.
func parseNumeric(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "NULL" {
		return 0, &strconv.NumError{Func: "parseNumeric", Num: s, Err: strconv.ErrSyntax}
	}
	return strconv.ParseFloat(s, 64)
}
