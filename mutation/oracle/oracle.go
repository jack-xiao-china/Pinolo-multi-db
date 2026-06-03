// Package oracle: check results to see if there is a logical bug according to implication oracle
package oracle

import (
	"github.com/qaqcatz/impomysql/connector"
)

// Check: check results to see if there is a logical bug according to implication oracle.
// return false if there is a logical bug, otherwise return true.
//
// Note that implication oracle cannot support error oracle.
// You cannot have any errors in your results, otherwise we will return an error
func Check(originResult *connector.Result, mutatedResult *connector.Result, isUpper bool) (bool, error) {
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

// CheckEquivalence: check if two result sets are semantically equivalent.
// Used for EET semantic rewrite mutations (De Morgan, BETWEEN→Cmp, COALESCE→CASE, NULLIF→CASE).
// These transformations should produce identical result sets.
// If the result sets differ → logical bug detected.
// Returns true if equivalent (no bug), false if not equivalent (bug).
func CheckEquivalence(originResult *connector.Result, mutatedResult *connector.Result) (bool, error) {
	cmp, err := originResult.CMP(mutatedResult)
	if err != nil {
		return false, err
	}
	// cmp == 0 means the result sets are identical → no bug
	return cmp == 0, nil
}