package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/qaqcatz/impomysql/connector"
)

// CalCandidates: visit the sub-AST defined in resources/impo.yy and get the candidate set of mutation points. see MutateVisitor.
//
// Each mutation has its own name, see:
//
//   FixMDistinctU
//	 FixMDistinctL
//	 FixMCmpOpU
//	 FixMCmpOpL
//	 FixMUnionAllU
//	 FixMUnionAllL
//   FixMInNullU
//	 FixMWhere1U
//	 FixMWhere0L
//	 FixMHaving1U
//	 FixMHaving0L
//	 FixMOn1U
//	 FixMOn0L
//	 FixMRmUnionAllL
//	 RdMLikeU
//	 RdMLikeL
//	 RdMRegExpU
//	 RdMRegExpL
//	 FixMAndTrueU     (EET tautology wrapping)
//	 FixMOrFalseL     (EET contradiction wrapping)
//	 FixMCaseTrueU    (EET CASE WHEN TRUE)
//	 FixMCaseFalseL   (EET CASE WHEN FALSE)
//	 FixMCaseRandEq   (EET CASE WHEN rand)
//
// about the prefix {FixM|RdM}(currently not working):
//   FixM means fixed mutation;
//   RdM means random mutation;
// about the suffix {U|L}:
//   U means upper mutation,
//   L means lower mutation.
func CalCandidates(sql string) (*MutateVisitor, error) {
	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "[CalCandidates]parse error")
	}
	if stmtNodes == nil || len(stmtNodes) == 0 {
		return nil, errors.New("[CalCandidates]stmtNodes == nil || len(stmtNodes) == 0")
	}
	rootNode := &stmtNodes[0]
	v := &MutateVisitor{
		Root: *rootNode,
		Candidates: make(map[string][]*Candidate)}
	v.visit(*rootNode, 1)
	return v, nil
}

// ImpoMutate: you can choose any candidate calculated by CalCandidates() to mutate, each mutation has no side effects.
func ImpoMutate(rootNode ast.Node, candidate *Candidate, seed int64) (string, error) {
	var sql []byte = nil
	var err error = nil
	switch candidate.MutationName {
	case FixMDistinctU:
		sql, err = doFixMDistinctU(rootNode, candidate.Node)
	case FixMDistinctL:
		sql, err = doFixMDistinctL(rootNode, candidate.Node)
	case FixMCmpOpU:
		sql, err = doFixMCmpOpU(rootNode, candidate.Node)
	case FixMCmpOpULE:
		sql, err = doFixMCmpOpULE(rootNode, candidate.Node)
	case FixMCmpOpL:
		sql, err = doFixMCmpOpL(rootNode, candidate.Node)
	case FixMUnionAllU:
		sql, err = doFixMUnionAllU(rootNode, candidate.Node)
	case FixMUnionAllL:
		sql, err = doFixMUnionAllL(rootNode, candidate.Node)
	case FixMInNullU:
		sql, err = doFixMInNullU(rootNode, candidate.Node)
	case FixMWhere1U:
		sql, err = doFixMWhere1U(rootNode, candidate.Node)
	case FixMWhere0L:
		sql, err = doFixMWhere0L(rootNode, candidate.Node)
	case FixMHaving1U:
		sql, err = doFixMHaving1U(rootNode, candidate.Node)
	case FixMHaving0L:
		sql, err = doFixMHaving0L(rootNode, candidate.Node)
	case FixMOn1U:
		sql, err = doFixMOn1U(rootNode, candidate.Node)
	case FixMOn0L:
		sql, err = doFixMOn0L(rootNode, candidate.Node)
	case FixMRmUnionAllL:
		sql, err = doFixMRmUnionAllL(rootNode, candidate.Node)
	case RdMLikeU:
		sql, err = doRdMLikeU(rootNode, candidate.Node, seed)
	case RdMLikeL:
		sql, err = doRdMLikeL(rootNode, candidate.Node, seed)
	case RdMRegExpU:
		sql, err = doRdMRegExpU(rootNode, candidate.Node, seed)
	case RdMRegExpL:
		sql, err = doRdMRegExpL(rootNode, candidate.Node, seed)
	// Implication mutations for BETWEEN
	case FixMBetweenDropUpperU:
		sql, err = doFixMBetweenDropUpperU(rootNode, candidate.Node, seed)
	case FixMBetweenDropLowerU:
		sql, err = doFixMBetweenDropLowerU(rootNode, candidate.Node, seed)
	// Implication mutations for comparison/subquery
	case FixMNullEqToLowerL:
		sql, err = doFixMNullEqToLowerL(rootNode, candidate.Node)
	case FixMAllToAnyU:
		sql, err = doFixMAllToAnyU(rootNode, candidate.Node, seed)
	case FixMAnyToAllL:
		sql, err = doFixMAnyToAllL(rootNode, candidate.Node, seed)
	// IS NULL / IS NOT NULL implication mutations
	case FixMIsNullToFalseL:
		sql, err = doFixMIsNullToFalseL(rootNode, candidate.Node)
	case FixMIsNotNullToTrueU:
		sql, err = doFixMIsNotNullToTrueU(rootNode, candidate.Node)
	}
	if err != nil {
		return "", err
	}
	return string(sql), nil
}

// ImpoMutateAndExec: ImpoMutate + exec.
func ImpoMutateAndExec(rootNode ast.Node, candidate *Candidate, seed int64,
	conn connector.SQLExecutor) (string, *connector.Result, error) {
	sql, err := ImpoMutate(rootNode, candidate, seed)
	if err != nil {
		return "", nil, err
	}
	result := conn.ExecSQL(sql)
	return sql, result, nil
}

// MutateUnit (mutation name, mutated sql, isUpper, error, execute result).
//
// IsUppers: true: the theoretical execution result of the current mutated statement will increase.
// It is actually ((Candidate.U ^ Candidate.Flag)^1) == 1
type MutateUnit struct {
	Name string
	Sql string
	IsUpper bool
	Err error

	ExecResult *connector.Result
}

// MutateResult: []*MutateUnit + error
type MutateResult struct {
	MutateUnits []*MutateUnit
	Err      error
}

// MutateAll: For the input sql, try all of its mutation points.
// We will save the mutated sqls into *MutateResult.
func MutateAll(sql string, seed int64) *MutateResult {
	mutateResult := &MutateResult {
		MutateUnits: make([]*MutateUnit, 0),
		Err:      nil,
	}

	v, err := CalCandidates(sql)
	if err != nil {
		mutateResult.Err = err
		return mutateResult
	}

	root := v.Root

	// Phase 1: Single-point mutations (k=1)
	singleUnits := make([]*MutateUnit, 0)
	for mutationName, candidateList := range v.Candidates {
		for _, candidate := range candidateList {
			newSql, err := ImpoMutate(root, candidate, seed)
			unit := &MutateUnit{
				Name: mutationName,
				Sql: newSql,
				IsUpper: ((candidate.U^candidate.Flag)^1) == 1,
				Err: err,
				ExecResult: nil,
			}
			singleUnits = append(singleUnits, unit)
			mutateResult.MutateUnits = append(mutateResult.MutateUnits, unit)
		}
	}

	// Phase 2: k=2 combinatorial mutations
	// Combine pairs of mutations with the same direction (both upper or both lower)
	// that target different AST nodes. Only combine if both succeed.
	// Soundness: if mutated1 ⊇ original AND mutated2 ⊇ original,
	// then applying both still gives mutated12 ⊇ original (upper).
	maxK2Pairs := 50 // limit to avoid explosion
	k2Count := 0
	for i := 0; i < len(singleUnits) && k2Count < maxK2Pairs; i++ {
		if singleUnits[i].Err != nil {
			continue
		}
		for j := i + 1; j < len(singleUnits) && k2Count < maxK2Pairs; j++ {
			if singleUnits[j].Err != nil {
				continue
			}
			// Only combine same-direction mutations
			if singleUnits[i].IsUpper != singleUnits[j].IsUpper {
				continue
			}
			// Skip if same mutation name (likely same node type, high conflict risk)
			if singleUnits[i].Name == singleUnits[j].Name {
				continue
			}
			// Try to re-mutate the first mutation's result with the second mutation
			k2Result := MutateAll(singleUnits[i].Sql, seed+int64(j))
			if k2Result.Err != nil {
				continue
			}
			// Find a matching mutation in the re-parsed result
			for _, k2Unit := range k2Result.MutateUnits {
				if k2Unit.Err != nil {
					continue
				}
				if k2Unit.Name == singleUnits[j].Name {
					mutateResult.MutateUnits = append(mutateResult.MutateUnits, &MutateUnit{
						Name: singleUnits[i].Name + "+" + singleUnits[j].Name,
						Sql: k2Unit.Sql,
						IsUpper: singleUnits[i].IsUpper,
						Err: nil,
						ExecResult: nil,
					})
					k2Count++
					break
				}
			}
		}
	}

	return mutateResult
}

// MutateAllAndExec: MutateAll and exec.
func MutateAllAndExec(sql string, seed int64, conn connector.SQLExecutor) *MutateResult {
	mutateResult := MutateAll(sql, seed)
	if mutateResult.Err != nil {
		return mutateResult
	}
	for _, mutateUnit := range mutateResult.MutateUnits {
		if mutateUnit.Err != nil {
			continue
		}
		mutateUnit.ExecResult = conn.ExecSQL(mutateUnit.Sql)
	}
	return mutateResult
}
