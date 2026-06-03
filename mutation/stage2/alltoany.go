package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"reflect"
)

// EET ALL <-> ANY quantifier transformation mutations (MySQL)

// addFixMAllToAnyU: FixMAllToAnyU, *ast.CompareSubqueryExpr: ALL -> ANY/SOME
// Implication (upper mutation): x > ALL(subq) ⊆ x > ANY(subq)
// Satisfying ALL values implies satisfying SOME value.
// Warning: NULL boundary may break containment (empty subquery: ALL→TRUE, ANY→FALSE).
func (v *MutateVisitor) addFixMAllToAnyU(in *ast.CompareSubqueryExpr, flag int) {
	if in != nil && in.All && in.L != nil && in.R != nil {
		v.addCandidate(FixMAllToAnyU, 1, in, flag)
	}
}

// addFixMAnyToAllL: FixMAnyToAllL, *ast.CompareSubqueryExpr: ANY/SOME -> ALL
// Implication (lower mutation): x > ANY(subq) ⊇ x > ALL(subq)
// Satisfying SOME value is broader than satisfying ALL values.
// Warning: NULL boundary may break containment.
func (v *MutateVisitor) addFixMAnyToAllL(in *ast.CompareSubqueryExpr, flag int) {
	if in != nil && !in.All && in.L != nil && in.R != nil {
		v.addCandidate(FixMAnyToAllL, 0, in, flag)
	}
}

// doFixMAllToAnyU: FixMAllToAnyU, x > ALL(subq) -> x > ANY(subq)
// Changes the quantifier from ALL to ANY/SOME, keeping the same comparison operator.
func doFixMAllToAnyU(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.CompareSubqueryExpr:
		expr := in.(*ast.CompareSubqueryExpr)
		if !expr.All {
			return nil, errors.New("[FixMAllToAnyU]expected ALL quantifier")
		}

		oldAll := expr.All
		// Mutate: ALL -> ANY (false)
		expr.All = false

		sql, err := restore(rootNode)
		if err != nil {
			expr.All = oldAll
			return nil, errors.Wrap(err, "[FixMAllToAnyU]restore error")
		}

		// Recover
		expr.All = oldAll
		return sql, nil
	case nil:
		return nil, errors.New("[FixMAllToAnyU]type nil")
	default:
		return nil, errors.New("[FixMAllToAnyU]type default " + reflect.TypeOf(in).String())
	}
}

// doFixMAnyToAllL: FixMAnyToAllL, x > ANY(subq) -> x > ALL(subq)
// Changes the quantifier from ANY/SOME to ALL, keeping the same comparison operator.
func doFixMAnyToAllL(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.CompareSubqueryExpr:
		expr := in.(*ast.CompareSubqueryExpr)
		if expr.All {
			return nil, errors.New("[FixMAnyToAllL]expected ANY/SOME quantifier")
		}

		oldAll := expr.All
		// Mutate: ANY -> ALL (true)
		expr.All = true

		sql, err := restore(rootNode)
		if err != nil {
			expr.All = oldAll
			return nil, errors.Wrap(err, "[FixMAnyToAllL]restore error")
		}

		// Recover
		expr.All = oldAll
		return sql, nil
	case nil:
		return nil, errors.New("[FixMAnyToAllL]type nil")
	default:
		return nil, errors.New("[FixMAnyToAllL]type default " + reflect.TypeOf(in).String())
	}
}
