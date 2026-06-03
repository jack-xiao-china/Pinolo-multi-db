package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/opcode"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/pingcap/tidb/parser/ast"
	"reflect"
)

// addFixMCmpOpL: FixMCmpOpL, *ast.BinaryOperationExpr, *ast.CompareSubqueryExpr: a {>=|<=} b -> a {>|<} b
// NOTE: NE(!=) is NOT included because != -> < has no valid containment relationship.
// a!=b being TRUE does not imply a<b being TRUE (e.g., a=5, b=3: 5!=3 is TRUE but 5<3 is FALSE).
func (v *MutateVisitor) addFixMCmpOpL(in ast.Node, flag int) {
	var myOp *opcode.Op = nil
	switch in.(type) {
	case *ast.BinaryOperationExpr:
		bin := in.(*ast.BinaryOperationExpr)
		myOp = &bin.Op
	case *ast.CompareSubqueryExpr:
		cmp := in.(*ast.CompareSubqueryExpr)
		myOp = &cmp.Op
	default:
		return
	}
	switch *myOp {
	case opcode.LE:
	case opcode.GE:
	default:
		return
	}
	v.addCandidate(FixMCmpOpL, 0, in, flag)
}

// doFixMCmpOpL: FixMCmpOpL, *ast.BinaryOperationExpr, *ast.CompareSubqueryExpr: a {>=|<=} b -> a {>|<} b
func doFixMCmpOpL(rootNode ast.Node, in ast.Node) ([]byte, error) {
	// check
	var myOp *opcode.Op = nil
	switch in.(type) {
	case *ast.BinaryOperationExpr:
		bin := in.(*ast.BinaryOperationExpr)
		myOp = &bin.Op
	case *ast.CompareSubqueryExpr:
		cmp := in.(*ast.CompareSubqueryExpr)
		myOp = &cmp.Op
	case nil:
		return nil, errors.New("[doFixMCmpOpL]type nil")
	default:
		return nil, errors.New("[doFixMCmpOpL]type default " + reflect.TypeOf(in).String())
	}

	oldOp := *myOp
	var newOp opcode.Op
	switch oldOp {
	case opcode.LE:
		newOp = opcode.LT
	case opcode.GE:
		newOp = opcode.GT
	default:
		return nil, errors.New("[doFixMCmpOpL]unsupported Op " + oldOp.String())
	}
	// mutate
	*myOp = newOp
	sql, err := restore(rootNode)
	if err != nil {
		return nil, errors.Wrap(err, "[doFixMCmpOpL]restore error")
	}
	// recover
	*myOp = oldOp
	return sql, nil
}
// addFixMNullEqToLowerL: FixMNullEqToLowerL, *ast.BinaryOperationExpr: a <=> b -> a = b
// Implication (lower mutation): a=b TRUE ⊆ a<=>b TRUE (normal equality is a subset of NULL-safe equality)
// <=> matches NULL=NULL as TRUE, while = returns NULL. So <=> gives more TRUE rows.
func (v *MutateVisitor) addFixMNullEqToLowerL(in *ast.BinaryOperationExpr, flag int) {
	if in != nil && in.Op == opcode.NullEQ && in.L != nil && in.R != nil {
		v.addCandidate(FixMNullEqToLowerL, 0, in, flag)
	}
}

// doFixMNullEqToLowerL: FixMNullEqToLowerL, a <=> b -> a = b
func doFixMNullEqToLowerL(rootNode ast.Node, in ast.Node) ([]byte, error) {
	switch in.(type) {
	case *ast.BinaryOperationExpr:
		expr := in.(*ast.BinaryOperationExpr)
		if expr.Op != opcode.NullEQ {
			return nil, errors.New("[FixMNullEqToLowerL]expected NullEQ operator")
		}

		oldOp := expr.Op
		// Mutate: <=> -> =
		expr.Op = opcode.EQ

		sql, err := restore(rootNode)
		if err != nil {
			expr.Op = oldOp
			return nil, errors.Wrap(err, "[FixMNullEqToLowerL]restore error")
		}

		// Recover
		expr.Op = oldOp
		return sql, nil
	case nil:
		return nil, errors.New("[FixMNullEqToLowerL]type nil")
	default:
		return nil, errors.New("[FixMNullEqToLowerL]type default " + reflect.TypeOf(in).String())
	}
}
