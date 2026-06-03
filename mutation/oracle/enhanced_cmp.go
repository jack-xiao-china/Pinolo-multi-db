// Package oracle: Enhanced comparison logic with type awareness
package oracle

import (
	"math"
	"strconv"
	"strings"

	"github.com/qaqcatz/impomysql/connector"
)

// EnhancedCMP provides type-aware comparison with improved handling of:
// - NULL values (represented as "\x00")
// - Floating point precision (with configurable epsilon)
// - Type coercion (numeric vs string)
type EnhancedCMP struct {
	Epsilon float64 // Tolerance for floating point comparison (default: 1e-9)
	NullStr string  // String representation of NULL (default: "\x00")
}

// NewEnhancedCMP creates a new EnhancedCMP instance with default settings
func NewEnhancedCMP() *EnhancedCMP {
	return &EnhancedCMP{
		Epsilon: 1e-9,
		NullStr: "\x00",
	}
}

// NewEnhancedCMPWithConfig creates a new EnhancedCMP instance with custom settings
func NewEnhancedCMPWithConfig(epsilon float64, nullStr string) *EnhancedCMP {
	return &EnhancedCMP{
		Epsilon: epsilon,
		NullStr: nullStr,
	}
}

// CMPResult represents the comparison result
type CMPResult int

const (
	CMPError       CMPResult = -2 // Error occurred
	CMPSubset      CMPResult = -1 // this ⊆ another (this is subset of another)
	CMPEqual       CMPResult = 0  // this == another
	CMPSuperset    CMPResult = 1  // this ⊇ another (this is superset of another)
	CMPIncomparable CMPResult = 2 // Incomparable (neither subset nor superset)
)

// Compare performs type-aware comparison between two results
// Returns:
//
//	-1: another contains this (this ⊆ another)
//	 0: equal
//	 1: this contains another (this ⊇ another)
//	 2: incomparable
//	-2: error
func (ecmp *EnhancedCMP) Compare(this *connector.Result, another *connector.Result) (CMPResult, error) {
	if this.Err != nil {
		return CMPError, this.Err
	}
	if another.Err != nil {
		return CMPError, another.Err
	}

	empty1 := this.IsEmpty()
	empty2 := another.IsEmpty()
	if empty1 || empty2 {
		if empty1 && empty2 {
			return CMPEqual, nil
		}
		if empty1 {
			return CMPSubset, nil // empty ⊆ non-empty
		}
		return CMPSuperset, nil // non-empty ⊇ empty
	}

	// Check column count
	if len(this.ColumnNames) != len(another.ColumnNames) {
		return CMPIncomparable, nil
	}

	// Normalize and flatten rows with type awareness
	res1 := ecmp.flatRowsWithType(this)
	res2 := ecmp.flatRowsWithType(another)

	// Build frequency map for res2
	mp := make(map[string]int)
	for _, row := range res2 {
		mp[row]++
	}

	// Check if all rows in res1 are in res2
	allInAnother := true
	for _, row := range res1 {
		if count, ok := mp[row]; ok {
			if count <= 1 {
				delete(mp, row)
			} else {
				mp[row] = count - 1
			}
		} else {
			allInAnother = false
		}
	}

	if allInAnother {
		if len(mp) == 0 {
			return CMPEqual, nil
		}
		return CMPSubset, nil // this ⊆ another
	}

	if len(mp) == 0 {
		return CMPSuperset, nil // this ⊇ another
	}
	return CMPIncomparable, nil
}

// flatRowsWithType normalizes rows with type-aware conversion
// Each row is converted to a canonical string representation
func (ecmp *EnhancedCMP) flatRowsWithType(result *connector.Result) []string {
	flt := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		normalized := make([]string, len(row))
		for i, val := range row {
			normalized[i] = ecmp.normalizeValue(val, result.ColumnTypes[i])
		}
		flt = append(flt, strings.Join(normalized, ","))
	}
	return flt
}

// normalizeValue normalizes a single value based on its type
func (ecmp *EnhancedCMP) normalizeValue(val string, colType string) string {
	// Handle NULL values
	if val == ecmp.NullStr || val == "NULL" || val == "" {
		return "NULL"
	}

	// Type-aware normalization
	colTypeLower := strings.ToLower(colType)

	// Integer types
	if isIntegerType(colTypeLower) {
		return ecmp.normalizeInteger(val)
	}

	// Floating point types
	if isFloatType(colTypeLower) {
		return ecmp.normalizeFloat(val)
	}

	// Decimal/Numeric types (preserve precision)
	if isDecimalType(colTypeLower) {
		return ecmp.normalizeDecimal(val)
	}

	// String types (no normalization)
	return val
}

// normalizeInteger normalizes integer values
func (ecmp *EnhancedCMP) normalizeInteger(val string) string {
	// Try to parse as integer
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return strconv.FormatInt(i, 10)
	}
	// Try to parse as float and convert to int
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return strconv.FormatInt(int64(f), 10)
	}
	return val
}

// normalizeFloat normalizes floating point values with epsilon comparison
func (ecmp *EnhancedCMP) normalizeFloat(val string) string {
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return val
	}

	// Round to avoid precision issues
	// Use epsilon-based rounding: round to nearest multiple of epsilon
	rounded := math.Round(f/ecmp.Epsilon) * ecmp.Epsilon

	// Format with enough precision
	return strconv.FormatFloat(rounded, 'f', 10, 64)
}

// normalizeDecimal normalizes decimal values (preserve precision)
func (ecmp *EnhancedCMP) normalizeDecimal(val string) string {
	// For decimals, we preserve the exact representation
	// but remove trailing zeros after decimal point
	if !strings.Contains(val, ".") {
		return val
	}

	// Remove trailing zeros
	val = strings.TrimRight(val, "0")
	val = strings.TrimRight(val, ".")

	return val
}

// isIntegerType checks if the column type is an integer type
func isIntegerType(colType string) bool {
	colTypeLower := strings.ToLower(colType)
	intTypes := []string{"int", "integer", "bigint", "smallint", "tinyint", "mediumint"}
	for _, t := range intTypes {
		if strings.Contains(colTypeLower, t) {
			return true
		}
	}
	return false
}

// isFloatType checks if the column type is a floating point type
func isFloatType(colType string) bool {
	colTypeLower := strings.ToLower(colType)
	floatTypes := []string{"float", "double", "real"}
	for _, t := range floatTypes {
		if strings.Contains(colTypeLower, t) {
			return true
		}
	}
	return false
}

// isDecimalType checks if the column type is a decimal/numeric type
func isDecimalType(colType string) bool {
	colTypeLower := strings.ToLower(colType)
	decimalTypes := []string{"decimal", "numeric"}
	for _, t := range decimalTypes {
		if strings.Contains(colTypeLower, t) {
			return true
		}
	}
	return false
}

// EnhancedCheck provides an enhanced Check function with type awareness
// This is a drop-in replacement for the original Check function
func EnhancedCheck(originResult *connector.Result, mutatedResult *connector.Result, isUpper bool) (bool, error) {
	ecmp := NewEnhancedCMP()
	cmp, err := ecmp.Compare(originResult, mutatedResult)
	if err != nil {
		return false, err
	}

	if cmp == CMPEqual {
		return true, nil
	}
	if (isUpper && cmp == CMPSubset) || (!isUpper && cmp == CMPSuperset) {
		return true, nil
	}
	return false, nil
}

// EnhancedCheckWithConfig provides an enhanced Check function with custom configuration
func EnhancedCheckWithConfig(originResult *connector.Result, mutatedResult *connector.Result, isUpper bool, epsilon float64) (bool, error) {
	ecmp := NewEnhancedCMPWithConfig(epsilon, "\x00")
	cmp, err := ecmp.Compare(originResult, mutatedResult)
	if err != nil {
		return false, err
	}

	if cmp == CMPEqual {
		return true, nil
	}
	if (isUpper && cmp == CMPSubset) || (!isUpper && cmp == CMPSuperset) {
		return true, nil
	}
	return false, nil
}
