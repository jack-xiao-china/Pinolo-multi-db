package connector

import (
	"github.com/pkg/errors"
	"github.com/go-sql-driver/mysql"
	"reflect"
	"strconv"
	"time"
)

// Result:
//
// query result, for example:
//   +-----+------+------+
//   | 1+2 | ID   | NAME | -> ColumnNames: 1+2,    ID,  NAME
//   +-----+------+------+ -> ColumnTypes: BIGINT, INT, TEXT
//   |   3 |    1 | H    | -> Rows[0]:     3,      1,   H
//   |   3 |    2 | Z    | -> Rows[1]:     3,      2,   Z
//   |   3 |    3 | Y    | -> Rows[2]:     3,      3,   Y
//   +-----+------+------+
// or error, for example:
//  Err: ERROR 1054 (42S22): Unknown column 'T' in 'field list'
//
// note that:
//
// len(ColumnNames) = len(ColumnTypes) = len(Rows[i]);
//
// if the statement is not SELECT, then the ColumnNames, ColumnTypes and Rows are empty
type Result struct {
	ColumnNames []string
	ColumnTypes []string
	Rows [][]string
	Err error
	Time time.Duration // total time
}

func (result *Result) ToString() string {
	str := ""
	str += "ColumnName(ColumnType)s: "
	for i, columnName := range result.ColumnNames {
		str += " " + columnName + "(" + result.ColumnTypes[i] + ")"
	}
	str += "\n"
	for i, row := range result.Rows {
		str += "row " + strconv.Itoa(i) + ":"
		for _, data := range row {
			str += " " + data
		}
		str += "\n"
	}
	if result.Err != nil {
		str += "Error: " + result.Err.Error() + "\n"
	}

	str += result.Time.String()
	return str
}

// Result.FlatRows: [["1","2"],["3","4"]] -> ["1,2", "3,4"]
// Numeric values are normalized to prevent "0" vs "0.0000" false mismatches
func (result *Result) FlatRows() []string {
	flt := make([]string, 0)
	for _, r := range result.Rows {
		t := ""
		for i, e := range r {
			if i != 0 {
				t += ","
			}
			t += normalizeNumeric(e)
		}
		flt = append(flt, t)
	}
	return flt
}

// normalizeNumeric: normalize a numeric string to a canonical form
// "0.0000" → "0", "1.5000" → "1.5", "4294967295.0000" → "4294967295"
// Non-numeric strings are returned unchanged
func normalizeNumeric(s string) string {
	if len(s) == 0 {
		return s
	}

	// Quick check: must start with digit, '-', '+' or '.' to be numeric
	ch := s[0]
	if ch != '-' && ch != '+' && ch != '.' && (ch < '0' || ch > '9') {
		return s
	}

	// Check if the string is a valid number (integer or decimal)
	hasDot := false
	allDigits := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if hasDot {
				return s // two dots, not a number
			}
			hasDot = true
		} else if c == '-' || c == '+' {
			if i != 0 {
				return s // sign not at start
			}
		} else if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}
	if !allDigits {
		return s
	}

	// If no decimal point, return as-is (integer)
	if !hasDot {
		return s
	}

	// Strip trailing zeros after decimal point
	// "1.5000" → "1.5", "0.0000" → "0"
	dotIdx := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			dotIdx = i
			break
		}
	}

	end := len(s)
	for end > dotIdx+1 && s[end-1] == '0' {
		end--
	}
	// If we stripped all decimal digits, also remove the dot
	if end == dotIdx+1 {
		end = dotIdx
	}

	result := s[:end]
	// Handle "-0" → "0"
	if result == "-0" || result == "+0" {
		return "0"
	}
	return result
}

// Result.IsEmpty: if the result is empty
func (result *Result) IsEmpty() bool {
	return len(result.ColumnNames) == 0
}

func (result *Result) GetErrorCode() (int, error) {
	if result.Err == nil {
		return -1, errors.New("[Result.GetErrorCode]result.Err == nil")
	}
	rootCause := errors.Cause(result.Err)
	if driverErr, ok := rootCause.(*mysql.MySQLError); ok { // Now the error number is accessible directly
		return int(driverErr.Number), nil
	} else {
		return -1, errors.New("[Result.GetErrorCode]not *mysql.MySQLError " + reflect.TypeOf(rootCause).String())
	}
}

// Result.CMP:
//   -1: another contains this
//   0: eq
//   1: this contains another
//   2: others
//   error: this.Err or another.Err
//   do not consider the column name
func (this *Result) CMP(another *Result) (int, error) {
	if this.Err != nil {
		return -2, errors.New("[Result.CMP]this error")
	}
	if another.Err != nil {
		return -2, errors.New("[Result.CMP]another error")
	}

	empty1 := this.IsEmpty()
	empty2 := another.IsEmpty()
	if empty1 || empty2 {
		// empty1&&!empty2, !empty1&&empty2, empty1&&empty2
		if (empty1 && empty2) {
			return 0, nil
		}
		if empty1 {
			// empty1&&!empty2
			return -1, nil;
		} else {
			// !empty1&&empty2
			return 1, nil;
		}
	}

	if len(this.ColumnNames) != len(another.ColumnNames) {
		return 2, nil
	}

	res1 := this.FlatRows()
	res2 := another.FlatRows()

	mp := make(map[string]int)
	for i := 0; i < len(res2); i++ {
		if num, ok := mp[res2[i]]; ok {
			mp[res2[i]] = num + 1
		} else {
			mp[res2[i]] = 1
		}
	}
	allInAnother := true
	for i := 0; i < len(res1); i++ {
		if num, ok := mp[res1[i]]; ok {
			if num <= 1 {
				delete(mp, res1[i])
			} else {
				mp[res1[i]] = num - 1
			}
		} else {
			allInAnother = false
		}
	}

	if allInAnother {
		if len(mp) == 0 {
			return 0, nil
		} else {
			return -1, nil
		}
	} else {
		if len(mp) == 0 {
			return 1, nil
		} else {
			return 2, nil
		}
	}
}