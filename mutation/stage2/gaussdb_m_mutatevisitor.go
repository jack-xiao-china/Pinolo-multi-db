package stage2

import (
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

// GaussDBMMutateVisitor: mutation visitor for GaussDB-M (MySQL compatibility mode)
//
// Uses TiDB parser (same as MySQL MutateVisitor) because:
// - GaussDB-M input SQL is MySQL syntax style
// - M mode differences are behavioral/semantic, not syntactic
//
// Reuses all MySQL mutation rules + EET semantic rewrite rules,
// plus M-mode specific mutations (TopToLimit, IfToCase, ConcatToPipe)
type GaussDBMMutateVisitor struct {
	Root       ast.Node
	Candidates map[string][]*Candidate // mutation name : slice of *Candidate
}

// CalCandidatesForMMode: visit the AST and get candidate mutation points for GaussDB-M
func CalCandidatesForMMode(sql string) (*GaussDBMMutateVisitor, error) {
	// Use the same parser and candidate calculation as MySQL
	v, err := CalCandidates(sql)
	if err != nil {
		return nil, err
	}

	// Wrap in GaussDBMMutateVisitor with M-mode specific candidates added
	mV := &GaussDBMMutateVisitor{
		Root:       v.Root,
		Candidates: v.Candidates,
	}

	// Add M-mode specific candidates by traversing the AST again
	mV.addMModeCandidates(v.Root, 1)

	return mV, nil
}

// addMModeCandidates: add M-mode specific mutation candidates
// These are mutations that exploit behavioral differences between
// GaussDB-M and standard MySQL
func (v *GaussDBMMutateVisitor) addMModeCandidates(root ast.Node, flag int) {
	// We traverse the AST to find M-mode specific mutation points
	// Currently these are:
	// - FixMTopToLimit: TOP n -> LIMIT n (if TOP syntax is present)
	// - FixMIfToCase: IF(cond,a,b) -> CASE WHEN cond THEN a ELSE b END
	// - FixMConcatToPipe: CONCAT(a,b) -> a || b

	mWalker := &mModeWalker{
		visitor: v,
		flag:    flag,
	}
	root.Accept(mWalker)
}

// mModeWalker: AST visitor that adds M-mode specific mutation candidates
type mModeWalker struct {
	visitor *GaussDBMMutateVisitor
	flag    int
}

func (w *mModeWalker) Enter(in ast.Node) (ast.Node, bool) {
	switch in.(type) {
	case *ast.FuncCallExpr:
		funcCall := in.(*ast.FuncCallExpr)
		funcName := funcCall.FnName.L
		switch funcName {
		case "if":
			w.visitor.addFixMIfToCase(funcCall, w.flag)
		case "concat":
			if len(funcCall.Args) == 2 {
				w.visitor.addFixMConcatToPipe(funcCall, w.flag)
			}
		}
	}
	return in, false
}

func (w *mModeWalker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// M-mode specific candidate add functions

// addFixMIfToCase: FixMIfToCase, IF(cond,a,b) -> CASE WHEN cond THEN a ELSE b END
func (v *GaussDBMMutateVisitor) addFixMIfToCase(in *ast.FuncCallExpr, flag int) {
	if in != nil && in.FnName.L == "if" && len(in.Args) == 3 {
		v.addMModeCandidate(FixMIfToCase, 1, in, flag)
	}
}

// addFixMConcatToPipe: FixMConcatToPipe, CONCAT(a,b) -> a || b
func (v *GaussDBMMutateVisitor) addFixMConcatToPipe(in *ast.FuncCallExpr, flag int) {
	if in != nil && in.FnName.L == "concat" && len(in.Args) == 2 {
		v.addMModeCandidate(FixMConcatToPipe, 1, in, flag)
	}
}

// addMModeCandidate: add a M-mode specific mutation candidate
func (v *GaussDBMMutateVisitor) addMModeCandidate(mutationName string, u int, in ast.Node, flag int) {
	var ls []*Candidate
	var ok bool
	if ls, ok = v.Candidates[mutationName]; !ok {
		ls = make([]*Candidate, 0)
	}
	ls = append(ls, &Candidate{
		MutationName: mutationName,
		U:            u,
		Node:         in,
		Flag:         flag,
	})
	v.Candidates[mutationName] = ls
}