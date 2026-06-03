package stage1

import (
	"github.com/pingcap/tidb/parser/ast"
)

// rmLimit: remove LIMIT clause completely (conservative approach)
//
// Conservative strategy:
// - Remove LIMIT clause entirely instead of replacing with large value
// - This allows the database to return all rows naturally
// - Preserves query semantics better than arbitrary large limits
//
// For example:
//
// SELECT * FROM T LIMIT 10 OFFSET 5 -> SELECT * FROM T
func rmLimit(in ast.Node) bool {
	if selectStmt, ok := in.(*ast.SelectStmt); ok {
		if selectStmt.Limit != nil {
			selectStmt.Limit = nil
			return true
		}
	}
	return false
}
