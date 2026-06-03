package oracle

import (
	"testing"

	"github.com/qaqcatz/impomysql/connector"
)

// TestEnhancedCMP_Basic tests basic comparison functionality
func TestEnhancedCMP_Basic(t *testing.T) {
	ecmp := NewEnhancedCMP()

	testCases := []struct {
		name     string
		result1  *connector.Result
		result2  *connector.Result
		expected CMPResult
	}{
		{
			name: "Equal results",
			result1: &connector.Result{
				ColumnNames: []string{"id", "name"},
				ColumnTypes: []string{"int", "varchar"},
				Rows: [][]string{
					{"1", "Alice"},
					{"2", "Bob"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id", "name"},
				ColumnTypes: []string{"int", "varchar"},
				Rows: [][]string{
					{"1", "Alice"},
					{"2", "Bob"},
				},
			},
			expected: CMPEqual,
		},
		{
			name: "Subset result",
			result1: &connector.Result{
				ColumnNames: []string{"id", "name"},
				ColumnTypes: []string{"int", "varchar"},
				Rows: [][]string{
					{"1", "Alice"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id", "name"},
				ColumnTypes: []string{"int", "varchar"},
				Rows: [][]string{
					{"1", "Alice"},
					{"2", "Bob"},
				},
			},
			expected: CMPSubset,
		},
		{
			name: "Superset result",
			result1: &connector.Result{
				ColumnNames: []string{"id", "name"},
				ColumnTypes: []string{"int", "varchar"},
				Rows: [][]string{
					{"1", "Alice"},
					{"2", "Bob"},
					{"3", "Charlie"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id", "name"},
				ColumnTypes: []string{"int", "varchar"},
				Rows: [][]string{
					{"1", "Alice"},
					{"2", "Bob"},
				},
			},
			expected: CMPSuperset,
		},
		{
			name: "Empty results",
			result1: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows:        [][]string{},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows:        [][]string{},
			},
			expected: CMPEqual,
		},
		{
			name: "Empty vs non-empty",
			result1: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows:        [][]string{},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
				},
			},
			expected: CMPSubset,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ecmp.Compare(tc.result1, tc.result2)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestEnhancedCMP_IntegerNormalization tests integer type normalization
func TestEnhancedCMP_IntegerNormalization(t *testing.T) {
	ecmp := NewEnhancedCMP()

	testCases := []struct {
		name     string
		result1  *connector.Result
		result2  *connector.Result
		expected CMPResult
	}{
		{
			name: "Integer normalization: 1 vs 1.0",
			result1: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1.0"},
				},
			},
			expected: CMPEqual,
		},
		{
			name: "Integer normalization: 42 vs 42.0000",
			result1: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"bigint"},
				Rows: [][]string{
					{"42"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"bigint"},
				Rows: [][]string{
					{"42.0000"},
				},
			},
			expected: CMPEqual,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ecmp.Compare(tc.result1, tc.result2)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestEnhancedCMP_FloatNormalization tests floating point normalization
func TestEnhancedCMP_FloatNormalization(t *testing.T) {
	ecmp := NewEnhancedCMP()

	testCases := []struct {
		name     string
		result1  *connector.Result
		result2  *connector.Result
		expected CMPResult
	}{
		{
			name: "Float normalization: 1.0 vs 1.0000000001",
			result1: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"float"},
				Rows: [][]string{
					{"1.0"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"float"},
				Rows: [][]string{
					{"1.0000000001"},
				},
			},
			expected: CMPEqual, // Should be equal due to epsilon
		},
		{
			name: "Float normalization: 0.0 vs 0.0000",
			result1: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"double"},
				Rows: [][]string{
					{"0.0"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"double"},
				Rows: [][]string{
					{"0.0000"},
				},
			},
			expected: CMPEqual,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ecmp.Compare(tc.result1, tc.result2)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestEnhancedCMP_DecimalNormalization tests decimal type normalization
func TestEnhancedCMP_DecimalNormalization(t *testing.T) {
	ecmp := NewEnhancedCMP()

	testCases := []struct {
		name     string
		result1  *connector.Result
		result2  *connector.Result
		expected CMPResult
	}{
		{
			name: "Decimal normalization: 1.50 vs 1.5",
			result1: &connector.Result{
				ColumnNames: []string{"price"},
				ColumnTypes: []string{"decimal(10,2)"},
				Rows: [][]string{
					{"1.50"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"price"},
				ColumnTypes: []string{"decimal(10,2)"},
				Rows: [][]string{
					{"1.5"},
				},
			},
			expected: CMPEqual,
		},
		{
			name: "Decimal normalization: 100.00 vs 100",
			result1: &connector.Result{
				ColumnNames: []string{"amount"},
				ColumnTypes: []string{"numeric(15,2)"},
				Rows: [][]string{
					{"100.00"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"amount"},
				ColumnTypes: []string{"numeric(15,2)"},
				Rows: [][]string{
					{"100"},
				},
			},
			expected: CMPEqual,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ecmp.Compare(tc.result1, tc.result2)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestEnhancedCMP_NULLHandling tests NULL value handling
func TestEnhancedCMP_NULLHandling(t *testing.T) {
	ecmp := NewEnhancedCMP()

	testCases := []struct {
		name     string
		result1  *connector.Result
		result2  *connector.Result
		expected CMPResult
	}{
		{
			name: "NULL handling: \\x00 vs NULL",
			result1: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"varchar"},
				Rows: [][]string{
					{"\x00"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"varchar"},
				Rows: [][]string{
					{"NULL"},
				},
			},
			expected: CMPEqual,
		},
		{
			name: "NULL handling: empty string vs NULL",
			result1: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"varchar"},
				Rows: [][]string{
					{""},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"value"},
				ColumnTypes: []string{"varchar"},
				Rows: [][]string{
					{"NULL"},
				},
			},
			expected: CMPEqual,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ecmp.Compare(tc.result1, tc.result2)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestEnhancedCMP_DuplicateRows tests handling of duplicate rows
func TestEnhancedCMP_DuplicateRows(t *testing.T) {
	ecmp := NewEnhancedCMP()

	testCases := []struct {
		name     string
		result1  *connector.Result
		result2  *connector.Result
		expected CMPResult
	}{
		{
			name: "Duplicate rows: equal",
			result1: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"1"},
					{"2"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"1"},
					{"2"},
				},
			},
			expected: CMPEqual,
		},
		{
			name: "Duplicate rows: subset",
			result1: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"1"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"1"},
					{"2"},
				},
			},
			expected: CMPSubset,
		},
		{
			name: "Duplicate rows: not subset (count mismatch)",
			result1: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"1"},
					{"1"},
				},
			},
			result2: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"1"},
					{"2"},
				},
			},
			expected: CMPIncomparable, // 3x "1" vs 2x "1" + 1x "2"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ecmp.Compare(tc.result1, tc.result2)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestEnhancedCheck tests the EnhancedCheck function
func TestEnhancedCheck(t *testing.T) {
	testCases := []struct {
		name           string
		originResult   *connector.Result
		mutatedResult  *connector.Result
		isUpper        bool
		expectedResult bool
	}{
		{
			name: "UPPER mutation: subset (should pass)",
			originResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
				},
			},
			mutatedResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			isUpper:        true,
			expectedResult: true,
		},
		{
			name: "UPPER mutation: equal (should pass)",
			originResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			mutatedResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			isUpper:        true,
			expectedResult: true,
		},
		{
			name: "UPPER mutation: superset (should fail)",
			originResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
					{"3"},
				},
			},
			mutatedResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			isUpper:        true,
			expectedResult: false,
		},
		{
			name: "LOWER mutation: superset (should pass)",
			originResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
					{"3"},
				},
			},
			mutatedResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			isUpper:        false,
			expectedResult: true,
		},
		{
			name: "LOWER mutation: equal (should pass)",
			originResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			mutatedResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			isUpper:        false,
			expectedResult: true,
		},
		{
			name: "LOWER mutation: subset (should fail)",
			originResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
				},
			},
			mutatedResult: &connector.Result{
				ColumnNames: []string{"id"},
				ColumnTypes: []string{"int"},
				Rows: [][]string{
					{"1"},
					{"2"},
				},
			},
			isUpper:        false,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EnhancedCheck(tc.originResult, tc.mutatedResult, tc.isUpper)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tc.expectedResult {
				t.Errorf("Expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

// TestEnhancedCheckWithConfig tests the EnhancedCheckWithConfig function
func TestEnhancedCheckWithConfig(t *testing.T) {
	originResult := &connector.Result{
		ColumnNames: []string{"value"},
		ColumnTypes: []string{"float"},
		Rows: [][]string{
			{"1.0"},
		},
	}
	mutatedResult := &connector.Result{
		ColumnNames: []string{"value"},
		ColumnTypes: []string{"float"},
		Rows: [][]string{
			{"1.0000000001"},
		},
	}

	// With default epsilon (1e-9), should be equal (difference is 1e-10 < 1e-9)
	result1, err := EnhancedCheckWithConfig(originResult, mutatedResult, true, 1e-9)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if !result1 {
		t.Errorf("Expected true with default epsilon (1e-9), got false. Difference is 1e-10 which is < 1e-9")
	}

	// With very small epsilon (1e-12), should not be equal (difference is 1e-10 > 1e-12)
	result2, err := EnhancedCheckWithConfig(originResult, mutatedResult, true, 1e-12)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if result2 {
		t.Errorf("Expected false with small epsilon (1e-12), got true. Difference is 1e-10 which is > 1e-12")
	}
}

// TestNormalizeValue tests the normalizeValue function
func TestNormalizeValue(t *testing.T) {
	ecmp := NewEnhancedCMP()

	testCases := []struct {
		name     string
		val      string
		colType  string
		expected string
	}{
		{"Integer: 1", "1", "int", "1"},
		{"Integer: 1.0", "1.0", "int", "1"},
		{"Integer: 42.0000", "42.0000", "bigint", "42"},
		{"Float: 1.5", "1.5", "float", "1.5000000000"},
		{"Decimal: 1.50", "1.50", "decimal(10,2)", "1.5"},
		{"String: hello", "hello", "varchar", "hello"},
		{"NULL: \\x00", "\x00", "int", "NULL"},
		{"NULL: NULL", "NULL", "varchar", "NULL"},
		{"NULL: empty", "", "int", "NULL"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ecmp.normalizeValue(tc.val, tc.colType)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestTypeDetection tests the type detection functions
func TestTypeDetection(t *testing.T) {
	testCases := []struct {
		name     string
		colType  string
		isInt    bool
		isFloat  bool
		isDec    bool
	}{
		{"int", "int", true, false, false},
		{"INT", "INT", true, false, false},
		{"bigint", "bigint", true, false, false},
		{"float", "float", false, true, false},
		{"double", "double", false, true, false},
		{"decimal(10,2)", "decimal(10,2)", false, false, true},
		{"numeric(15,4)", "numeric(15,4)", false, false, true},
		{"varchar(255)", "varchar(255)", false, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			colTypeLower := tc.colType
			if isInt := isIntegerType(colTypeLower); isInt != tc.isInt {
				t.Errorf("isIntegerType(%q) = %v, expected %v", tc.colType, isInt, tc.isInt)
			}
			if isFloat := isFloatType(colTypeLower); isFloat != tc.isFloat {
				t.Errorf("isFloatType(%q) = %v, expected %v", tc.colType, isFloat, tc.isFloat)
			}
			if isDec := isDecimalType(colTypeLower); isDec != tc.isDec {
				t.Errorf("isDecimalType(%q) = %v, expected %v", tc.colType, isDec, tc.isDec)
			}
		})
	}
}
