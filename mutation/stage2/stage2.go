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
	// EET transformation mutations
	case FixMAndTrueU:
		sql, err = doFixMAndTrueU(rootNode, candidate.Node, seed)
	case FixMOrFalseL:
		sql, err = doFixMOrFalseL(rootNode, candidate.Node, seed)
	case FixMCaseTrueU:
		sql, err = doFixMCaseTrueU(rootNode, candidate.Node, seed)
	case FixMCaseFalseL:
		sql, err = doFixMCaseFalseL(rootNode, candidate.Node, seed)
	case FixMCaseRandEq:
		sql, err = doFixMCaseRandEq(rootNode, candidate.Node, seed)
		// EET semantic rewrite mutations
		case FixMDeMorganAnd:
			sql, err = doFixMDeMorganAnd(rootNode, candidate.Node, seed)
		case FixMDeMorganOr:
			sql, err = doFixMDeMorganOr(rootNode, candidate.Node, seed)
		case FixMBetweenToCmp:
			sql, err = doFixMBetweenToCmp(rootNode, candidate.Node, seed)
		case FixMCoalesceToCase:
			sql, err = doFixMCoalesceToCase(rootNode, candidate.Node, seed)
		case FixMNullifToCase:
			sql, err = doFixMNullifToCase(rootNode, candidate.Node, seed)
		case FixMExistsToIn:
			sql, err = doFixMExistsToIn(rootNode, candidate.Node, seed)
		case FixMInToExists:
			sql, err = doFixMInToExists(rootNode, candidate.Node, seed)
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
	IsEquivalence bool // true for EET semantic rewrite mutations (use CheckEquivalence oracle)
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
	for mutationName, candidateList := range v.Candidates {
		for _, candidate := range candidateList {
			newSql, err := ImpoMutate(root, candidate, seed)
			mutateResult.MutateUnits = append(mutateResult.MutateUnits, &MutateUnit{
				Name: mutationName,
				Sql: newSql,
				IsUpper: ((candidate.U^candidate.Flag)^1) == 1,
				IsEquivalence: isEquivalenceMutation(mutationName),
				Err: err,

				ExecResult: nil,
			})
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

// isEquivalenceMutation: returns true if the mutation name is an EET semantic rewrite mutation
// that should use the equivalence oracle (CheckEquivalence) instead of implication oracle (Check).
// Equivalence mutations produce SQL that should have identical result sets to the original.
// NOTE: Tautology/contradiction/CASE wrapping are ALSO equivalence transformations:
//   FixMAndTrueU:  (tautology) AND E ≡ TRUE AND E ≡ E  (equivalent)
//   FixMOrFalseL:  (contradiction) OR E ≡ FALSE OR E ≡ E (equivalent)
//   FixMCaseTrueU: CASE WHEN TRUE THEN E ELSE rand ≡ E   (equivalent, TRUE always fires THEN)
//   FixMCaseFalseL: CASE WHEN FALSE THEN rand ELSE E ≡ E  (equivalent, FALSE always fires ELSE)
func isEquivalenceMutation(name string) bool {
	equivalenceMutations := []string{
		// Wrapping mutations (equivalent despite U/L naming)
		FixMAndTrueU, FixMOrFalseL, FixMCaseTrueU, FixMCaseFalseL,
		// Semantic rewrite mutations (equivalent)
		FixMCaseRandEq,
		FixMDeMorganAnd, FixMDeMorganOr,
		FixMBetweenToCmp,
		FixMCoalesceToCase, FixMNullifToCase,
		FixMExistsToIn, FixMInToExists,
		FixMIfToCase, FixMConcatToPipe,
	}
	for _, m := range equivalenceMutations {
		if name == m {
			return true
		}
	}
	return false
}

// isEquivalenceMutationPg: returns true for PG equivalence mutations.
// NOTE: Tautology/contradiction/CASE wrapping are ALSO equivalence transformations.
func isEquivalenceMutationPg(name string) bool {
	equivalenceMutationsPg := []string{
		// Wrapping mutations (equivalent despite U/L naming)
		FixMAndTrueU_Pg, FixMOrFalseL_Pg, FixMCaseTrueU_Pg, FixMCaseFalseL_Pg,
		// Semantic rewrite mutations (equivalent)
		FixMCaseRandEq_Pg,
		FixMDeMorganAnd_Pg, FixMDeMorganOr_Pg,
		FixMBetweenToCmp_Pg,
		FixMCoalesceToCase_Pg, FixMNullifToCase_Pg,
		FixMExistsToIn_Pg, FixMInToExists_Pg,
	}
	for _, m := range equivalenceMutationsPg {
		if name == m {
			return true
		}
	}
	return false
}
