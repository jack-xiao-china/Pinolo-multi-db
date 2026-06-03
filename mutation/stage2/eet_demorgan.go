package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	"reflect"
)

// EET De Morgan's Law transformation mutations (MySQL)

// addFixMDeMorganAnd: FixMDeMorganAnd, *ast.BinaryOperationExpr: (A AND B) → NOT(NOT(A) OR NOT(B))
// Semantically equivalent. If result sets differ → bug detected.
func (v *MutateVisitor) addFixMDeMorganAnd(in *ast.BinaryOperationExpr, flag int) {
	if in != nil && in.Op == opcode.LogicAnd && in.L != nil && in.R != nil {
		v.addCandidate(FixMDeMorganAnd, 1, in, flag)
	}
}

// addFixMDeMorganOr: FixMDeMorganOr, *ast.BinaryOperationExpr: (A OR B) → NOT(NOT(A) AND NOT(B))
// Semantically equivalent. If result sets differ → bug detected.
func (v *MutateVisitor) addFixMDeMorganOr(in *ast.BinaryOperationExpr, flag int) {
	if in != nil && in.Op == opcode.LogicOr && in.L != nil && in.R != nil {
		v.addCandidate(FixMDeMorganOr, 1, in, flag)
	}
}

// doFixMDeMorganAnd: FixMDeMorganAnd, (A AND B) → NOT(NOT(A) OR NOT(B))
func doFixMDeMorganAnd(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.BinaryOperationExpr:
		expr := in.(*ast.BinaryOperationExpr)
		if expr.Op != opcode.LogicAnd {
			return nil, errors.New("[FixMDeMorganAnd]expected LogicAnd operator")
		}

		oldL := expr.L
		oldR := expr.R

		// NOT(A)
		notA := &ast.UnaryOperationExpr{
			Op: opcode.Not,
			V:  oldL,
		}
		// NOT(B)
		notB := &ast.UnaryOperationExpr{
			Op: opcode.Not,
			V:  oldR,
		}
		// NOT(A) OR NOT(B)
		orExpr := &ast.BinaryOperationExpr{
			Op: opcode.LogicOr,
			L:  notA,
			R:  notB,
		}
		// NOT(NOT(A) OR NOT(B))
		result := &ast.UnaryOperationExpr{
			Op: opcode.Not,
			V:  &ast.ParenthesesExpr{Expr: orExpr},
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: result,
		}

		// Replace in parent context (do NOT modify expr before walking, it causes nil pointer crash)
		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMDeMorganAnd]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)

		return sql, nil
	case nil:
		return nil, errors.New("[FixMDeMorganAnd]type nil")
	default:
		return nil, errors.New("[FixMDeMorganAnd]type default " + reflect.TypeOf(in).String())
	}
}

// doFixMDeMorganOr: FixMDeMorganOr, (A OR B) → NOT(NOT(A) AND NOT(B))
func doFixMDeMorganOr(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.BinaryOperationExpr:
		expr := in.(*ast.BinaryOperationExpr)
		if expr.Op != opcode.LogicOr {
			return nil, errors.New("[FixMDeMorganOr]expected LogicOr operator")
		}

		oldL := expr.L
		oldR := expr.R

		// NOT(A)
		notA := &ast.UnaryOperationExpr{
			Op: opcode.Not,
			V:  oldL,
		}
		// NOT(B)
		notB := &ast.UnaryOperationExpr{
			Op: opcode.Not,
			V:  oldR,
		}
		// NOT(A) AND NOT(B)
		andExpr := &ast.BinaryOperationExpr{
			Op: opcode.LogicAnd,
			L:  notA,
			R:  notB,
		}
		// NOT(NOT(A) AND NOT(B))
		result := &ast.UnaryOperationExpr{
			Op: opcode.Not,
			V:  &ast.ParenthesesExpr{Expr: andExpr},
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: result,
		}

		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMDeMorganOr]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)

		return sql, nil
	case nil:
		return nil, errors.New("[FixMDeMorganOr]type nil")
	default:
		return nil, errors.New("[FixMDeMorganOr]type default " + reflect.TypeOf(in).String())
	}
}

// replaceExprInRoot: replace an expression node in the AST root.
// This is a general-purpose helper that walks the AST and replaces
// the target node with the replacement node.
// It uses the existing pattern from eet_mutations.go (modify AST, restore, then undo).
func replaceExprInRoot(rootNode ast.Node, target ast.ExprNode, replacement ast.ExprNode) {
	// Walk the AST and replace the target expression with replacement
	// This works by modifying the parent node's expression field
	walker := &exprReplacer{
		target:      target,
		replacement: replacement,
		replaced:    false,
	}
	rootNode.Accept(walker)
}

// exprReplacer: AST visitor that replaces a target expression with a replacement
type exprReplacer struct {
	target      ast.ExprNode
	replacement ast.ExprNode
	replaced    bool
}

func (r *exprReplacer) Enter(in ast.Node) (ast.Node, bool) {
	return in, false // continue visiting
}

func (r *exprReplacer) Leave(in ast.Node) (ast.Node, bool) {
	// Check if this node is a parent that holds our target as a child expression
	switch in.(type) {
	case *ast.SelectStmt:
		sel := in.(*ast.SelectStmt)
		if sel.Where == r.target {
			sel.Where = r.replacement
			r.replaced = true
		}
		if sel.Having != nil && sel.Having.Expr == r.target {
			sel.Having.Expr = r.replacement
			r.replaced = true
		}
	case *ast.OnCondition:
		on := in.(*ast.OnCondition)
		if on.Expr == r.target {
			on.Expr = r.replacement
			r.replaced = true
		}
	case *ast.SelectField:
		field := in.(*ast.SelectField)
		if field.Expr == r.target {
			field.Expr = r.replacement
			r.replaced = true
		}
	case *ast.BinaryOperationExpr:
		bin := in.(*ast.BinaryOperationExpr)
		if bin.L == r.target {
			bin.L = r.replacement
			r.replaced = true
		}
		if bin.R == r.target {
			bin.R = r.replacement
			r.replaced = true
		}
	case *ast.UnaryOperationExpr:
		un := in.(*ast.UnaryOperationExpr)
		if un.V == r.target {
			un.V = r.replacement
			r.replaced = true
		}
	case *ast.ParenthesesExpr:
		paren := in.(*ast.ParenthesesExpr)
		if paren.Expr == r.target {
			paren.Expr = r.replacement
			r.replaced = true
		}
	case *ast.IsNullExpr:
		isNull := in.(*ast.IsNullExpr)
		if isNull.Expr == r.target {
			isNull.Expr = r.replacement
			r.replaced = true
		}
	case *ast.IsTruthExpr:
		isTruth := in.(*ast.IsTruthExpr)
		if isTruth.Expr == r.target {
			isTruth.Expr = r.replacement
			r.replaced = true
		}
	case *ast.PatternInExpr:
		patternIn := in.(*ast.PatternInExpr)
		if patternIn.Expr == r.target {
			patternIn.Expr = r.replacement
			r.replaced = true
		}
	case *ast.PatternLikeExpr:
		like := in.(*ast.PatternLikeExpr)
		if like.Expr == r.target {
			like.Expr = r.replacement
			r.replaced = true
		}
	case *ast.PatternRegexpExpr:
		regexp := in.(*ast.PatternRegexpExpr)
		if regexp.Expr == r.target {
			regexp.Expr = r.replacement
			r.replaced = true
		}
	case *ast.CompareSubqueryExpr:
		cmpSub := in.(*ast.CompareSubqueryExpr)
		if cmpSub.L == r.target {
			cmpSub.L = r.replacement
			r.replaced = true
		}
	case *ast.FuncCallExpr:
		fn := in.(*ast.FuncCallExpr)
		for i, arg := range fn.Args {
			if arg == r.target {
				fn.Args[i] = r.replacement
				r.replaced = true
			}
		}
	case *ast.HavingClause:
		having := in.(*ast.HavingClause)
		if having.Expr == r.target {
			having.Expr = r.replacement
			r.replaced = true
		}
	}
	return in, true
}