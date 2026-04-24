package parser

// ASTNode: unified AST node abstraction
// Supports both TiDB parser and pg_query parser
type ASTNode interface {
	// GetOriginal returns the original node for type-specific operations
	GetOriginal() interface{}
}

// MutationCandidate: mutation candidate point
type MutationCandidate struct {
	Name    string  // mutation name, e.g., FixMWhere1U, FixMCmpOpU
	Node    ASTNode // candidate node
	IsUpper bool    // true: upper mutation (result expands), false: lower mutation (result shrinks)
	Flag    int     // 1: positive, 0: negative (for semantic analysis)
}

// Parser: SQL parser interface
type Parser interface {
	// Parse parses SQL string into AST
	Parse(sql string) (ASTNode, error)

	// Restore converts AST back to SQL string
	Restore(node ASTNode) (string, error)

	// GetParserType returns parser type name
	GetParserType() string
}

// CandidateFinder: finds mutation candidates from AST
type CandidateFinder interface {
	// FindCandidates returns all mutation candidates for the given AST
	FindCandidates(node ASTNode) []MutationCandidate
}

// Mutator: applies mutation to AST
type Mutator interface {
	// Mutate applies mutation and returns mutated SQL
	Mutate(node ASTNode, candidate MutationCandidate) (string, error)
}