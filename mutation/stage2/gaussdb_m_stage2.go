package stage2

import (
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/qaqcatz/impomysql/connector"
)

// ImpoMutateForMMode: mutate a SQL statement for GaussDB-M using the candidate mutation point.
//
// For standard MySQL mutations, delegates to ImpoMutate.
// For M-mode specific mutations (FixMIfToCase, FixMConcatToPipe), handles directly.
func ImpoMutateForMMode(rootNode ast.Node, candidate *Candidate, seed int64) (string, error) {
	// Check if this is a M-mode specific mutation
	switch candidate.MutationName {
	case FixMIfToCase:
		sql, err := doFixMIfToCase(rootNode, candidate.Node, seed)
		if err != nil {
			return "", err
		}
		return string(sql), nil
	case FixMConcatToPipe:
		sql, err := doFixMConcatToPipe(rootNode, candidate.Node, seed)
		if err != nil {
			return "", err
		}
		return string(sql), nil
	default:
		// Delegate to standard MySQL ImpoMutate for all other mutations
		return ImpoMutate(rootNode, candidate, seed)
	}
}

// ImpoMutateAndExecForMMode: ImpoMutateForMMode + exec
func ImpoMutateAndExecForMMode(rootNode ast.Node, candidate *Candidate, seed int64,
	conn connector.SQLExecutor) (string, *connector.Result, error) {
	sql, err := ImpoMutateForMMode(rootNode, candidate, seed)
	if err != nil {
		return "", nil, err
	}
	result := conn.ExecSQL(sql)
	return sql, result, nil
}

// MutateAllForMMode: For the input SQL, try all mutation points for GaussDB-M.
func MutateAllForMMode(sql string, seed int64) *MutateResult {
	mutateResult := &MutateResult{
		MutateUnits: make([]*MutateUnit, 0),
		Err:         nil,
	}

	v, err := CalCandidatesForMMode(sql)
	if err != nil {
		mutateResult.Err = err
		return mutateResult
	}

	root := v.Root
	for mutationName, candidateList := range v.Candidates {
		for _, candidate := range candidateList {
			newSql, err := ImpoMutateForMMode(root, candidate, seed)
			mutateResult.MutateUnits = append(mutateResult.MutateUnits, &MutateUnit{
				Name:      mutationName,
				Sql:       newSql,
				IsUpper:   ((candidate.U ^ candidate.Flag) ^ 1) == 1,
				IsEquivalence: isEquivalenceMutation(mutationName),
				Err:       err,
				ExecResult: nil,
			})
		}
	}

	return mutateResult
}

// MutateAllAndExecForMMode: MutateAllForMMode and exec
func MutateAllAndExecForMMode(sql string, seed int64, conn connector.SQLExecutor) *MutateResult {
	mutateResult := MutateAllForMMode(sql, seed)
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