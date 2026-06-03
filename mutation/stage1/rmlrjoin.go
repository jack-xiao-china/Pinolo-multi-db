package stage1

import (
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

// rmLRJoin: change LEFT/RIGHT [OUTER] JOIN to INNER JOIN (conservative approach)
//
// Conservative strategy:
// - Convert LEFT JOIN to INNER JOIN (represented as CrossJoin with ON condition)
// - Convert RIGHT JOIN to INNER JOIN (represented as CrossJoin with ON condition)
// - Keep ON conditions to preserve join semantics
// - Preserve NaturalJoin and StraightJoin flags when appropriate
//
// For example:
//
// SELECT * FROM T1 LEFT JOIN T2 ON T1.id = T2.id
// -> SELECT * FROM T1 JOIN T2 ON T1.id = T2.id
//
// Note: In TiDB parser, CrossJoin with ON condition is equivalent to INNER JOIN.
// This is more conservative than removing ON conditions, as it preserves
// the join semantics and maintains a closer relationship to the original query.
func rmLRJoin(in ast.Node) bool {
	if join, ok := in.(*ast.Join); ok {
		if join.Tp == ast.LeftJoin || join.Tp == ast.RightJoin {
			// Convert to CrossJoin (INNER JOIN when ON condition exists)
			join.Tp = ast.CrossJoin
			// Keep ON conditions (stored in join.On) to preserve join semantics
			// Keep NaturalJoin and StraightJoin flags as they may affect query execution
			return true
		}
	}
	return false
}
