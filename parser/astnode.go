package parser

// TiDBASTNode: AST node wrapper for TiDB parser
type TiDBASTNode struct {
	original interface{}
}

func NewTiDBASTNode(node interface{}) *TiDBASTNode {
	return &TiDBASTNode{original: node}
}

func (n *TiDBASTNode) GetOriginal() interface{} {
	return n.original
}

// PgQueryASTNode: AST node wrapper for pg_query parser
type PgQueryASTNode struct {
	original interface{}
	// Additional metadata for Oracle-specific handling
	oracleNodes map[string]interface{} // stores Oracle-specific nodes (ROWNUM, etc.)
}

func NewPgQueryASTNode(node interface{}) *PgQueryASTNode {
	return &PgQueryASTNode{
		original:    node,
		oracleNodes: make(map[string]interface{}),
	}
}

func (n *PgQueryASTNode) GetOriginal() interface{} {
	return n.original
}

func (n *PgQueryASTNode) SetOracleNode(key string, node interface{}) {
	n.oracleNodes[key] = node
}

func (n *PgQueryASTNode) GetOracleNode(key string) interface{} {
	return n.oracleNodes[key]
}

func (n *PgQueryASTNode) HasOracleNodes() bool {
	return len(n.oracleNodes) > 0
}