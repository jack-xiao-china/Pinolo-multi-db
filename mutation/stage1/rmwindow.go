package stage1

import (
	"github.com/pingcap/tidb/parser/ast"
)

// rmWindow: remove window functions in a conservative way.
//
// Conservative strategy:
// - Keep window function structure
// - Simplify window definition to empty OVER() clause
// - Remove named window specifications from SELECT statement
//
// For example:
//
// SELECT SUM(C1) OVER w as sum_c1 FROM T WINDOW w AS (PARTITION BY C2 ORDER BY C3)
// -> SELECT SUM(C1) OVER () as sum_c1 FROM T
func rmWindow(in ast.Node) bool {
	change := false
	if selectStmt, ok := in.(*ast.SelectStmt); ok {
		if selectStmt.WindowSpecs != nil {
			change = true
			selectStmt.WindowSpecs = nil
		}
	}
	if fieldList, ok := in.(*ast.FieldList); ok {
		for _, field := range fieldList.Fields {
			if field.Expr == nil {
				continue
			}
			if windowFunc, ok := field.Expr.(*ast.WindowFuncExpr); ok {
				change = true
				// Simplify window definition: remove partition and order by
				// Spec is a value type, so we can directly modify its pointer fields
				windowFunc.Spec.PartitionBy = nil
				windowFunc.Spec.OrderBy = nil
				windowFunc.Spec.Frame = nil
			}
		}
	}
	return change
}
