package stage1

import (
	"regexp"
	"strings"
)

// RmConnectBy: detect and skip CONNECT BY hierarchical queries
// Oracle CONNECT BY syntax is not supported in PostgreSQL/GaussDB A mode preprocessing
// Returns (processedSQL, shouldSkip)
func RmConnectBy(sql string) (string, bool) {
	// Pattern to detect CONNECT BY clause
	// CONNECT BY can appear in various forms:
	// - CONNECT BY PRIOR col = col
	// - CONNECT BY col = PRIOR col
	// - CONNECT BY NOCYCLE ...
	connectByPattern := regexp.MustCompile(`\bCONNECT\s+BY\b`)

	if connectByPattern.MatchString(strings.ToUpper(sql)) {
		// Skip this SQL as it contains CONNECT BY hierarchical query
		return "", true
	}

	return sql, false
}

// HasConnectBy: check if SQL contains CONNECT BY clause
func HasConnectBy(sql string) bool {
	connectByPattern := regexp.MustCompile(`\bCONNECT\s+BY\b`)
	return connectByPattern.MatchString(strings.ToUpper(sql))
}

// HasStartWith: check if SQL contains START WITH clause (often paired with CONNECT BY)
func HasStartWith(sql string) bool {
	startWithPattern := regexp.MustCompile(`\bSTART\s+WITH\b`)
	return startWithPattern.MatchString(strings.ToUpper(sql))
}