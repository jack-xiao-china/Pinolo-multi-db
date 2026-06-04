package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"reflect"
)

// FixMCmpOpULE: a = b -> a <= b (upper mutation)
//
// Containment: {x | x = 5} ⊂ {x | x <= 5} → original ⊂ mutated → upper
// This complements FixMCmpOpU's = → >= with the other direction.

// addFixMCmpOpULE: FixMCmpOpULE, *ast.BinaryOperationExpr / *ast.CompareSubqueryExpr: a = b -> a <= b
func (v *MutateVisitor) addFixMCmpOpULE(in ast.Node, flag int) {
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
	if *myOp == opcode.EQ {
		v.addCandidate(FixMCmpOpULE, 1, in, flag)
	}
}

// doFixMCmpOpULE: FixMCmpOpULE, a = b -> a <= b
func doFixMCmpOpULE(rootNode ast.Node, in ast.Node) ([]byte, error) {
	var myOp *opcode.Op = nil
	switch in.(type) {
	case *ast.BinaryOperationExpr:
		bin := in.(*ast.BinaryOperationExpr)
		myOp = &bin.Op
	case *ast.CompareSubqueryExpr:
		cmp := in.(*ast.CompareSubqueryExpr)
		myOp = &cmp.Op
	case nil:
		return nil, errors.New("[doFixMCmpOpULE]type nil")
	default:
		return nil, errors.New("[doFixMCmpOpULE]type default " + reflect.TypeOf(in).String())
	}

	if *myOp != opcode.EQ {
		return nil, errors.New("[doFixMCmpOpULE]expected EQ operator, got " + myOp.String())
	}

	oldOp := *myOp
	*myOp = opcode.LE
	sql, err := restore(rootNode)
	if err != nil {
		return nil, errors.Wrap(err, "[doFixMCmpOpULE]restore error")
	}
	*myOp = oldOp
	return sql, nil
}
