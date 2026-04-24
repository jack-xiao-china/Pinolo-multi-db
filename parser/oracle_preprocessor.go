package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// OraclePreprocessor: preprocesses Oracle-specific SQL syntax
// Converts Oracle syntax to PostgreSQL-compatible syntax
type OraclePreprocessor struct {
	// KeepRownum: if true, ROWNUM syntax is preserved (not converted to LIMIT)
	KeepRownum bool
}

func NewOraclePreprocessor() *OraclePreprocessor {
	return &OraclePreprocessor{
		KeepRownum: true, // Default: preserve ROWNUM syntax per user requirement
	}
}

// Normalize: converts Oracle-specific syntax to PostgreSQL-compatible syntax
func (o *OraclePreprocessor) Normalize(sql string) string {
	result := sql

	// 1. Convert (+) outer join syntax to LEFT JOIN
	result = o.convertOuterJoin(result)

	// 2. Convert NVL to COALESCE
	result = o.convertNVL(result)

	// 3. Convert DECODE to CASE WHEN
	result = o.convertDECODE(result)

	// 4. Convert SYSDATE to NOW()
	result = o.convertSysdate(result)

	// 5. Remove DUAL table reference (if FROM DUAL)
	result = o.removeDual(result)

	// 6. Convert TO_CHAR/TO_DATE (simplified)
	result = o.convertToDateFunctions(result)

	// Note: ROWNUM is preserved (not converted to LIMIT) per user requirement

	return result
}

// convertOuterJoin: converts Oracle (+) outer join to LEFT JOIN
// Example: WHERE t1.id = t2.id(+) -> LEFT JOIN t2 ON t1.id = t2.id
func (o *OraclePreprocessor) convertOuterJoin(sql string) string {
	// This is a simplified implementation
	// Full implementation would require AST parsing

	// Pattern: column = table.column(+)
	// This indicates table is the outer joined table
	pattern := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*)\(\+\)`)

	matches := pattern.FindAllString(sql, -1)
	if len(matches) == 0 {
		return sql
	}

	// For each match, identify the table with (+)
	// This is a placeholder - full implementation needs AST analysis
	for _, match := range matches {
		// Remove (+) marker
		colWithPlus := match
		colWithoutPlus := strings.TrimSuffix(colWithPlus, "(+)")
		sql = strings.Replace(sql, colWithPlus, colWithoutPlus, 1)
	}

	// Note: Full LEFT JOIN conversion requires complex SQL restructuring
	// This simplified version just removes (+) markers
	// The actual LEFT JOIN syntax should be handled by proper SQL parser

	return sql
}

// convertNVL: converts NVL(x, y) to COALESCE(x, y)
func (o *OraclePreprocessor) convertNVL(sql string) string {
	// Pattern: NVL(expr1, expr2)
	pattern := regexp.MustCompile(`\bNVL\s*\(`)
	return pattern.ReplaceAllStringFunc(strings.ToUpper(sql), func(match string) string {
		// Find the actual NVL in original SQL and replace
		upperSQL := strings.ToUpper(sql)
		idx := strings.Index(upperSQL, "NVL(")
		if idx == -1 {
			idx = strings.Index(upperSQL, "NVL (")
		}
		if idx != -1 {
			// Replace NVL with COALESCE in original SQL
			sql = sql[:idx] + "COALESCE" + sql[idx+3:]
		}
		return "COALESCE("
	})
}

// convertDECODE: converts DECODE(x, v1, r1, v2, r2, default) to CASE WHEN
// Simplified implementation - full implementation needs expression parsing
func (o *OraclePreprocessor) convertDECODE(sql string) string {
	upperSQL := strings.ToUpper(sql)

	// Pattern: DECODE(expr, search1, result1, ..., default)
	if !strings.Contains(upperSQL, "DECODE") {
		return sql
	}

	// Find DECODE position
	decodePattern := regexp.MustCompile(`\bDECODE\s*\(`)
	if !decodePattern.MatchString(upperSQL) {
		return sql
	}

	// Simplified: Just mark that DECODE needs conversion
	// Full implementation requires parsing the DECODE arguments
	// and generating CASE WHEN x = v1 THEN r1 ... ELSE default END

	// Placeholder: For now, keep DECODE as-is for A mode execution
	// (GaussDB A mode supports DECODE directly)

	return sql
}

// convertSysdate: converts SYSDATE to NOW()
func (o *OraclePreprocessor) convertSysdate(sql string) string {
	upperSQL := strings.ToUpper(sql)

	// SYSDATE -> NOW()
	if strings.Contains(upperSQL, "SYSDATE") {
		// Find position in original SQL
		idx := strings.Index(upperSQL, "SYSDATE")
		if idx != -1 {
			// Replace SYSDATE with NOW()
			// Keep original case for rest of SQL
			originalIdx := idx
			sql = sql[:originalIdx] + "NOW()" + sql[originalIdx+7:]
		}
	}

	return sql
}

// removeDual: removes FROM DUAL clause
// Oracle: SELECT x FROM DUAL -> PostgreSQL: SELECT x
func (o *OraclePreprocessor) removeDual(sql string) string {
	upperSQL := strings.ToUpper(sql)

	// Pattern: FROM DUAL
	dualPattern := regexp.MustCompile(`\bFROM\s+DUAL\b`)
	if dualPattern.MatchString(upperSQL) {
		// Remove FROM DUAL
		match := dualPattern.FindString(upperSQL)
		// Find the same pattern in original SQL
		idx := strings.Index(upperSQL, match)
		if idx != -1 {
			// Calculate the length of FROM DUAL in original SQL
			// (accounting for possible case differences)
			fromDualLen := len(match)
			sql = sql[:idx] + sql[idx+fromDualLen:]
		}
	}

	return strings.TrimSpace(sql)
}

// convertToDateFunctions: converts TO_CHAR/TO_DATE (simplified)
// Note: GaussDB A mode supports TO_CHAR/TO_DATE directly
func (o *OraclePreprocessor) convertToDateFunctions(sql string) string {
	// GaussDB A mode supports Oracle TO_CHAR/TO_DATE functions
	// No conversion needed for A mode execution
	return sql
}

// ParseRownumCondition: parses ROWNUM conditions for mutation
// Returns (operator, value) where operator is <=, <, =, >, >= and value is the number
func ParseRownumCondition(sql string) (operator string, value int64, found bool) {
	upperSQL := strings.ToUpper(sql)

	// Pattern: ROWNUM <= N, ROWNUM < N, ROWNUM = N, etc.
	patterns := map[string]*regexp.Regexp{
		"<=": regexp.MustCompile(`\bROWNUM\s*<=\s*(\d+)`),
		"<":  regexp.MustCompile(`\bROWNUM\s*<\s*(\d+)`),
		"=":  regexp.MustCompile(`\bROWNUM\s*=\s*(\d+)`),
		">":  regexp.MustCompile(`\bROWNUM\s*>\s*(\d+)`),
		">=": regexp.MustCompile(`\bROWNUM\s*>=\s*(\d+)`),
	}

	for op, pattern := range patterns {
		matches := pattern.FindStringSubmatch(upperSQL)
		if len(matches) >= 2 {
			// Parse the number
			var num int64
			_, err := fmt.Sscanf(matches[1], "%d", &num)
			if err == nil {
				return op, num, true
			}
		}
	}

	return "", 0, false
}

// MutateRownum: modifies ROWNUM condition for mutation testing
// isUpper=true: ROWNUM <= N -> ROWNUM <= N+1 (expands result)
// isUpper=false: ROWNUM <= N -> ROWNUM <= N-1 (shrinks result)
func MutateRownum(sql string, operator string, value int64, isUpper bool) string {
	newValue := value
	if isUpper {
		newValue = value + 1
	} else {
		if value > 1 {
			newValue = value - 1
		} else {
			// Cannot reduce below 1, return original
			return sql
		}
	}

	upperSQL := strings.ToUpper(sql)
	// Find and replace ROWNUM condition
	rownumPattern := regexp.MustCompile(fmt.Sprintf(`\bROWNUM\s*%s\s*%d`, operator, value))

	// Find the actual position in original SQL
	match := rownumPattern.FindString(upperSQL)
	if match != "" {
		idx := strings.Index(upperSQL, match)
		if idx != -1 {
			// Build new condition
			newCondition := fmt.Sprintf("ROWNUM %s %d", operator, newValue)
			// Replace in original SQL (preserve case where possible)
			matchLen := len(match)
			sql = sql[:idx] + newCondition + sql[idx+matchLen:]
		}
	}

	return sql
}