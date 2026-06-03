package stage2

import (
	"github.com/pingcap/tidb/parser/ast"
)

// AST replacement utilities
// These are general-purpose helpers for replacing expression nodes in the AST.

// replaceExprInRoot: replace an expression node in the AST root.
// Walks the AST and replaces the target node with the replacement node.
// Uses modify AST, restore, then undo pattern.
func replaceExprInRoot(rootNode ast.Node, target ast.ExprNode, replacement ast.ExprNode) {
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
