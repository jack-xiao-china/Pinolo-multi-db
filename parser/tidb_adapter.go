package parser

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

const ParserTypeTiDB = "tidb"

// TiDBParser: TiDB parser adapter for MySQL/M mode
type TiDBParser struct {
	p *parser.Parser
}

func NewTiDBParser() *TiDBParser {
	return &TiDBParser{
		p: parser.New(),
	}
}

func (tp *TiDBParser) Parse(sql string) (ASTNode, error) {
	stmtNodes, _, err := tp.p.Parse(sql, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "[TiDBParser.Parse]parse error")
	}
	if stmtNodes == nil || len(stmtNodes) == 0 {
		return nil, errors.New("[TiDBParser.Parse]stmtNodes is empty")
	}
	return NewTiDBASTNode(&stmtNodes[0]), nil
}

func (tp *TiDBParser) Restore(node ASTNode) (string, error) {
	tidbNode, ok := node.(*TiDBASTNode)
	if !ok {
		return "", errors.New("[TiDBParser.Restore]node is not TiDBASTNode")
	}

	rootNode, ok := tidbNode.GetOriginal().(*ast.StmtNode)
	if !ok {
		// Try direct ast.Node
		astNode, ok := tidbNode.GetOriginal().(ast.Node)
		if !ok {
			return "", errors.New("[TiDBParser.Restore]original is not ast.Node")
		}

		buf := new(bytes.Buffer)
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags|format.RestoreStringWithoutCharset, buf)
		err := astNode.Restore(ctx)
		if err != nil {
			return "", errors.Wrap(err, "[TiDBParser.Restore]restore error")
		}
		return buf.String(), nil
	}

	buf := new(bytes.Buffer)
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags|format.RestoreStringWithoutCharset, buf)
	err := (*rootNode).Restore(ctx)
	if err != nil {
		return "", errors.Wrap(err, "[TiDBParser.Restore]restore error")
	}
	return buf.String(), nil
}

func (tp *TiDBParser) GetParserType() string {
	return ParserTypeTiDB
}

// GetTiDBRootNode extracts the root ast.Node from TiDBASTNode
func GetTiDBRootNode(node ASTNode) (ast.Node, error) {
	tidbNode, ok := node.(*TiDBASTNode)
	if !ok {
		return nil, errors.New("[GetTiDBRootNode]node is not TiDBASTNode")
	}

	original := tidbNode.GetOriginal()

	// Handle *ast.StmtNode
	stmtNode, ok := original.(*ast.StmtNode)
	if ok {
		return *stmtNode, nil
	}

	// Handle direct ast.Node
	astNode, ok := original.(ast.Node)
	if ok {
		return astNode, nil
	}

	return nil, errors.New("[GetTiDBRootNode]cannot extract ast.Node")
}