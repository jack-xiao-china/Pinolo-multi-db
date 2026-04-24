package stage1

import (
	"regexp"
	"strings"
)

// RmPLSQL: detect and skip PL/SQL blocks and stored procedures
// PL/SQL blocks are not SELECT statements and should be filtered
// Returns (processedSQL, shouldSkip)
func RmPLSQL(sql string) (string, bool) {
	upperSQL := strings.ToUpper(sql)

	// Pattern to detect PL/SQL blocks
	// - DECLARE ... BEGIN ... END
	// - BEGIN ... END (anonymous block)
	// - CREATE [OR REPLACE] PROCEDURE/FUNCTION
	plsqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\bDECLARE\b`),
		regexp.MustCompile(`\bBEGIN\s*;?\s*[^;]*\bEND\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*PROCEDURE\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*FUNCTION\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*PACKAGE\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*TRIGGER\b`),
	}

	for _, pattern := range plsqlPatterns {
		if pattern.MatchString(upperSQL) {
			return "", true
		}
	}

	return sql, false
}

// HasPLSQL: check if SQL contains PL/SQL block
func HasPLSQL(sql string) bool {
	upperSQL := strings.ToUpper(sql)

	plsqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\bDECLARE\b`),
		regexp.MustCompile(`\bBEGIN\s*[^;]*\bEND\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*PROCEDURE\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*FUNCTION\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*PACKAGE\b`),
		regexp.MustCompile(`\bCREATE\s+(OR\s+REPLACE\s+)?\s*TRIGGER\b`),
	}

	for _, pattern := range plsqlPatterns {
		if pattern.MatchString(upperSQL) {
			return true
		}
	}

	return false
}

// HasDBMSPackage: check if SQL references Oracle DBMS_* packages
func HasDBMSPackage(sql string) bool {
	dbmsPattern := regexp.MustCompile(`\bDBMS_[A-Z_]+\b`)
	return dbmsPattern.MatchString(strings.ToUpper(sql))
}

// HasUTLPackage: check if SQL references Oracle UTL_* packages
func HasUTLPackage(sql string) bool {
	utlPattern := regexp.MustCompile(`\bUTL_[A-Z_]+\b`)
	return utlPattern.MatchString(strings.ToUpper(sql))
}