package stage1

import (
	"testing"
)

// TestInitWithoutDB tests Stage1 transformations without database connection
func TestInitWithoutDB(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove LIMIT",
			input:    "SELECT * FROM t LIMIT 10",
			expected: "SELECT * FROM `t`",
		},
		{
			name:     "Remove LIMIT with OFFSET",
			input:    "SELECT * FROM t LIMIT 10 OFFSET 5",
			expected: "SELECT * FROM `t`",
		},
		{
			name:     "Remove GROUP BY and aggregate",
			input:    "SELECT c1, SUM(c2) FROM t GROUP BY c1",
			expected: "SELECT `c1`,(`c2`) FROM `t`",
		},
		{
			name:     "Remove COUNT(*)",
			input:    "SELECT COUNT(*) FROM t",
			expected: "SELECT (1) FROM `t`",
		},
		{
			name:     "Remove COUNT(column)",
			input:    "SELECT COUNT(c1) FROM t",
			expected: "SELECT (`c1`) FROM `t`",
		},
		{
			name:     "Remove LEFT JOIN to INNER JOIN",
			input:    "SELECT * FROM t1 LEFT JOIN t2 ON t1.id = t2.id",
			expected: "SELECT * FROM `t1` JOIN `t2` ON `t1`.`id`=`t2`.`id`",
		},
		{
			name:     "Remove RIGHT JOIN to INNER JOIN",
			input:    "SELECT * FROM t1 RIGHT JOIN t2 ON t1.id = t2.id",
			expected: "SELECT * FROM `t1` JOIN `t2` ON `t1`.`id`=`t2`.`id`",
		},
		{
			name:     "Replace RAND()",
			input:    "SELECT RAND() FROM t",
			expected: "SELECT (5e-01) FROM `t`",
		},
		{
			name:     "Replace NOW()",
			input:    "SELECT NOW() FROM t",
			expected: "SELECT ('2020-01-01 00:00:00') FROM `t`",
		},
		{
			name:     "Replace UUID()",
			input:    "SELECT UUID() FROM t",
			expected: "SELECT ('00000000-0000-0000-0000-000000000000') FROM `t`",
		},
		{
			name:     "Replace CURDATE()",
			input:    "SELECT CURDATE() FROM t",
			expected: "SELECT ('2020-01-01') FROM `t`",
		},
		{
			name:     "Replace DATABASE()",
			input:    "SELECT DATABASE() FROM t",
			expected: "SELECT ('test') FROM `t`",
		},
		{
			name:     "Replace CONNECTION_ID()",
			input:    "SELECT CONNECTION_ID() FROM t",
			expected: "SELECT (0) FROM `t`",
		},
		{
			name:     "Replace SLEEP()",
			input:    "SELECT SLEEP(1) FROM t",
			expected: "SELECT (0) FROM `t`",
		},
		{
			name:     "Complex query with multiple transformations",
			input:    "SELECT c1, SUM(c2), RAND() FROM t LEFT JOIN t2 ON t.id = t2.id GROUP BY c1 LIMIT 10",
			expected: "SELECT `c1`,(`c2`),(5e-01) FROM `t` JOIN `t2` ON `t`.`id`=`t2`.`id`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Init(tc.input)
			if result.Err != nil {
				t.Fatalf("Init failed: %v", result.Err)
			}
			if result.InitSql != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result.InitSql)
			}
		})
	}
}
