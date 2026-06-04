package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"github.com/qaqcatz/impomysql/connector"
)

// CalCandidatesForPostgreSQL: visit the PostgreSQL AST and get the candidate set of mutation points.
// Similar to CalCandidates but uses pg_query_go parser instead of TiDB parser.
//
// Each mutation has its own name, see pg_mutate_functions.go for PostgreSQL-specific mutations:
//
//   FixMWhere1U_Pg, FixMWhere0L_Pg
//   FixMHaving1U_Pg, FixMHaving0L_Pg
//   FixMOn1U_Pg, FixMOn0L_Pg
//   FixMDistinctU_Pg, FixMDistinctL_Pg
//   FixMUnionAllU_Pg, FixMUnionAllL_Pg
//   FixMCmpOpU_Pg, FixMCmpOpL_Pg
//   FixMInNullU_Pg
//   RdMLikePgU, RdMLikePgL
//   RdMRegExpPgU, RdMRegExpPgL
func CalCandidatesForPostgreSQL(sql string) (*PgMutateVisitor, error) {
	result, err := pgquery.Parse(sql)
	if err != nil {
		return nil, errors.Wrap(err, "[CalCandidatesForPostgreSQL]parse error")
	}
	if result == nil || len(result.Stmts) == 0 {
		return nil, errors.New("[CalCandidatesForPostgreSQL]result == nil || len(result.Stmts) == 0")
	}

	v := &PgMutateVisitor{
		Root:      result,
		Candidates: make(map[string][]*PgCandidate),
	}
	v.FindCandidates(result, 1)
	return v, nil
}

// PgImpoMutate: mutate a PostgreSQL SQL statement using the candidate mutation point.
// Each mutation has no side effects on the original AST (the AST is restored after mutation).
func PgImpoMutate(rootNode *pgquery.ParseResult, candidate *PgCandidate, seed int64) (string, error) {
	var sql string
	var err error
	switch candidate.MutationName {
	case FixMWhere1U_Pg:
		sql, err = doFixMWhere1U_Pg(rootNode, candidate.Node)
	case FixMWhere0L_Pg:
		sql, err = doFixMWhere0L_Pg(rootNode, candidate.Node)
	case FixMHaving1U_Pg:
		sql, err = doFixMHaving1U_Pg(rootNode, candidate.Node)
	case FixMHaving0L_Pg:
		sql, err = doFixMHaving0L_Pg(rootNode, candidate.Node)
	case FixMOn1U_Pg:
		sql, err = doFixMOn1U_Pg(rootNode, candidate.Node)
	case FixMOn0L_Pg:
		sql, err = doFixMOn0L_Pg(rootNode, candidate.Node)
	case FixMDistinctU_Pg:
		sql, err = doFixMDistinctU_Pg(rootNode, candidate.Node)
	case FixMDistinctL_Pg:
		sql, err = doFixMDistinctL_Pg(rootNode, candidate.Node)
	case FixMUnionAllU_Pg:
		sql, err = doFixMUnionAllU_Pg(rootNode, candidate.Node)
	case FixMUnionAllL_Pg:
		sql, err = doFixMUnionAllL_Pg(rootNode, candidate.Node)
	case FixMCmpOpU_Pg:
		sql, err = doFixMCmpOpU_Pg(rootNode, candidate.Node)
	case FixMCmpOpULE_Pg:
		sql, err = doFixMCmpOpULE_Pg(rootNode, candidate.Node)
	case FixMCmpOpL_Pg:
		sql, err = doFixMCmpOpL_Pg(rootNode, candidate.Node)
	case FixMInNullU_Pg:
		sql, err = doFixMInNullU_Pg(rootNode, candidate.Node)
	case RdMLikePgU:
		sql, err = doRdMLikePgU(rootNode, candidate.Node, seed)
	case RdMLikePgL:
		sql, err = doRdMLikePgL(rootNode, candidate.Node, seed)
	case RdMRegExpPgU:
		sql, err = doRdMRegExpPgU(rootNode, candidate.Node, seed)
	case RdMRegExpPgL:
		sql, err = doRdMRegExpPgL(rootNode, candidate.Node, seed)
		case FixMBetweenDropUpperU_Pg:
			sql, err = doFixMBetweenDropUpperU_Pg(rootNode, candidate.Node, seed)
		case FixMBetweenDropLowerU_Pg:
			sql, err = doFixMBetweenDropLowerU_Pg(rootNode, candidate.Node, seed)
		case FixMAllToAnyU_Pg:
			sql, err = doFixMAllToAnyU_Pg(rootNode, candidate.Node, seed)
		case FixMAnyToAllL_Pg:
			sql, err = doFixMAnyToAllL_Pg(rootNode, candidate.Node, seed)
		case FixMIsNotDistinctFromToLowerL_Pg:
			sql, err = doFixMIsNotDistinctFromToLowerL_Pg(rootNode, candidate.Node, seed)
	default:
		return "", errors.New("[PgImpoMutate]unknown mutation name: " + candidate.MutationName)
	}
	if err != nil {
		return "", err
	}
	return sql, nil
}

// PgImpoMutateAndExec: PgImpoMutate + exec.
func PgImpoMutateAndExec(rootNode *pgquery.ParseResult, candidate *PgCandidate, seed int64,
	conn connector.SQLExecutor) (string, *connector.Result, error) {
	sql, err := PgImpoMutate(rootNode, candidate, seed)
	if err != nil {
		return "", nil, err
	}
	result := conn.ExecSQL(sql)
	return sql, result, nil
}

// PgMutateUnit: mutation result unit for PostgreSQL
type PgMutateUnit struct {
	Name       string
	Sql        string
	IsUpper    bool
	Err        error
	ExecResult *connector.Result
}

// PgMutateResult: mutation result for PostgreSQL
type PgMutateResult struct {
	MutateUnits []*PgMutateUnit
	Err         error
}

// MutateAllForPostgreSQL: For the input PostgreSQL SQL, try all of its mutation points.
func MutateAllForPostgreSQL(sql string, seed int64) *PgMutateResult {
	mutateResult := &PgMutateResult{
		MutateUnits: make([]*PgMutateUnit, 0),
		Err:         nil,
	}

	v, err := CalCandidatesForPostgreSQL(sql)
	if err != nil {
		mutateResult.Err = err
		return mutateResult
	}

	root := v.Root
	for mutationName, candidateList := range v.Candidates {
		for _, candidate := range candidateList {
			newSql, err := PgImpoMutate(root, candidate, seed)
			mutateResult.MutateUnits = append(mutateResult.MutateUnits, &PgMutateUnit{
				Name:           mutationName,
				Sql:            newSql,
				IsUpper:        ((candidate.U ^ candidate.Flag) ^ 1) == 1,
				Err:            err,
				ExecResult:     nil,
			})
		}
	}

	return mutateResult
}

// MutateAllAndExecForPostgreSQL: MutateAllForPostgreSQL and exec.
func MutateAllAndExecForPostgreSQL(sql string, seed int64, conn connector.SQLExecutor) *PgMutateResult {
	mutateResult := MutateAllForPostgreSQL(sql, seed)
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