package stage1

import (
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/test_driver"
)

// rmAgg: remove aggregate functions and GROUP BY in a conservative way.
//
// Conservative strategy:
// - Remove GROUP BY clause
// - Replace aggregate functions with their first argument (instead of constant 1)
//
// For example:
//
// SELECT C1, SUM(C2) FROM T GROUP BY C1 -> SELECT C1, C2 FROM T
// SELECT C1, COUNT(*) FROM T GROUP BY C1 -> SELECT C1, 1 FROM T (COUNT(*) has no meaningful argument)
func rmAgg(in ast.Node) bool {
	change := false
	if selectStmt, ok := in.(*ast.SelectStmt); ok {
		if selectStmt.GroupBy != nil {
			change = true
			selectStmt.GroupBy = nil
		}
	}
	if aggFunExpr, ok := in.(*ast.AggregateFuncExpr); ok {
		change = true

		// Conservative replacement: use the first argument instead of constant 1
		// For COUNT(*), we still use constant 1 since it has no meaningful argument
		if len(aggFunExpr.Args) > 0 {
			// Check if it's COUNT(*)
			if aggFunExpr.F == "count" && len(aggFunExpr.Args) == 1 {
				if _, isValueExpr := aggFunExpr.Args[0].(*test_driver.ValueExpr); isValueExpr {
					// COUNT(*) - replace with constant 1
					aggFunExpr.F = ""
					aggFunExpr.Distinct = false
					aggFunExpr.Order = nil
					aggFunExpr.Args = make([]ast.ExprNode, 0)
					aggFunExpr.Args = append(aggFunExpr.Args, &test_driver.ValueExpr{
						Datum: test_driver.NewDatum(1),
					})
				} else {
					// COUNT(column) - replace with the column itself
					firstArg := aggFunExpr.Args[0]
					aggFunExpr.F = ""
					aggFunExpr.Distinct = false
					aggFunExpr.Order = nil
					aggFunExpr.Args = make([]ast.ExprNode, 0)
					aggFunExpr.Args = append(aggFunExpr.Args, firstArg)
				}
			} else {
				// Other aggregate functions (SUM, AVG, MIN, MAX, etc.) - replace with first argument
				firstArg := aggFunExpr.Args[0]
				aggFunExpr.F = ""
				aggFunExpr.Distinct = false
				aggFunExpr.Order = nil
				aggFunExpr.Args = make([]ast.ExprNode, 0)
				aggFunExpr.Args = append(aggFunExpr.Args, firstArg)
			}
		} else {
			// Fallback: no arguments, use constant 1
			aggFunExpr.F = ""
			aggFunExpr.Distinct = false
			aggFunExpr.Order = nil
			aggFunExpr.Args = make([]ast.ExprNode, 0)
			aggFunExpr.Args = append(aggFunExpr.Args, &test_driver.ValueExpr{
				Datum: test_driver.NewDatum(1),
			})
		}
	}
	return change
}
