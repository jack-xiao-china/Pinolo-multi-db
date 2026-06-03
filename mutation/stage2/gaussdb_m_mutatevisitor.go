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
// Reuses all MySQL Implication mutation rules.
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

	// Wrap in GaussDBMMutateVisitor
	mV := &GaussDBMMutateVisitor{
		Root:       v.Root,
		Candidates: v.Candidates,
	}

	return mV, nil
}
