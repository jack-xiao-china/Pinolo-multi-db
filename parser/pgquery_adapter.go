package parser

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

const ParserTypePgQuery = "pgquery"

// PgQueryParser: pg_query parser adapter for PostgreSQL/A mode
type PgQueryParser struct {
	preprocessor *OraclePreprocessor
}

func NewPgQueryParser() *PgQueryParser {
	return &PgQueryParser{
		preprocessor: NewOraclePreprocessor(),
	}
}

// Parse: parse SQL string into PostgreSQL AST
func (pgp *PgQueryParser) Parse(sql string) (ASTNode, error) {
	// First, preprocess Oracle-specific syntax
	normalizedSQL := pgp.preprocessor.Normalize(sql)

	// Parse using pg_query
	result, err := pgquery.Parse(normalizedSQL)
	if err != nil {
		return nil, errors.Wrap(err, "[PgQueryParser.Parse]parse error")
	}

	// Wrap in PgQueryASTNode
	node := NewPgQueryASTNode(result)

	// Check for ROWNUM in original SQL and store it
	op, value, found := ParseRownumCondition(sql)
	if found {
		node.SetOracleNode("rownum", map[string]interface{}{
			"operator": op,
			"value":    value,
		})
	}

	return node, nil
}

// Restore: convert PostgreSQL AST back to SQL string
func (pgp *PgQueryParser) Restore(node ASTNode) (string, error) {
	pgNode, ok := node.(*PgQueryASTNode)
	if !ok {
		return "", errors.New("[PgQueryParser.Restore]node is not PgQueryASTNode")
	}

	original := pgNode.GetOriginal()

	// Handle pg_query ParseResult
	parseResult, ok := original.(*pgquery.ParseResult)
	if !ok {
		return "", errors.New("[PgQueryParser.Restore]original is not ParseResult")
	}

	// Use pg_query's deparse function
	sql, err := pgquery.Deparse(parseResult)
	if err != nil {
		return "", errors.Wrap(err, "[PgQueryParser.Restore]deparse error")
	}

	return sql, nil
}

func (pgp *PgQueryParser) GetParserType() string {
	return ParserTypePgQuery
}

// NormalizeSQL: normalize SQL for consistent representation
func NormalizeSQL(sql string) (string, error) {
	result, err := pgquery.Normalize(sql)
	if err != nil {
		return "", errors.Wrap(err, "[NormalizeSQL]normalize error")
	}
	return result, nil
}

// Fingerprints: generate SQL fingerprint for comparison
func Fingerprints(sql string) (string, error) {
	result, err := pgquery.Fingerprint(sql)
	if err != nil {
		return "", errors.Wrap(err, "[Fingerprints]fingerprint error")
	}
	return result, nil
}

// ParseToJSON: parse SQL and return JSON representation
func ParseToJSON(sql string) (string, error) {
	result, err := pgquery.ParseToJSON(sql)
	if err != nil {
		return "", errors.Wrap(err, "[ParseToJSON]parse error")
	}
	return result, nil
}