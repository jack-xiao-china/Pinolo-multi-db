package stage2

// all mutations
const (
	// *ast.SelectStmt: Distinct true -> false
	FixMDistinctU = "FixMDistinctU"
	// *ast.SelectStmt: Distinct false -> true
	FixMDistinctL = "FixMDistinctL"

	// *ast.SelectStmt: AfterSetOperator UNION -> UNION ALL
	FixMUnionAllU = "FixMUnionAllU"
	// *ast.SelectStmt: AfterSetOperator UNION ALL -> UNION
	FixMUnionAllL = "FixMUnionAllL"

	// *ast.BinaryOperationExpr, *ast.CompareSubqueryExpr: a {>|<|=} b -> a {>=|<=|>=} b
	FixMCmpOpU = "FixMCmpOpU"
	// *ast.BinaryOperationExpr, *ast.CompareSubqueryExpr: a {>=|<=} b -> a {>|<} b
	FixMCmpOpL = "FixMCmpOpL"

	// *ast.PatternInExpr: in(x,x,x) -> in(x,x,x,null)
	FixMInNullU = "FixMInNullU"

	// *ast.SelectStmt: WHERE xxx -> WHERE 1
	FixMWhere1U = "FixMWhere1U"
	// *ast.SelectStmt: WHERE xxx -> WHERE 0
	FixMWhere0L = "FixMWhere0L"

	// *ast.SelectStmt: HAVING xxx -> HAVING 1
	FixMHaving1U = "FixMHaving1U"
	// *ast.SelectStmt: HAVING xxx -> HAVING 0
	FixMHaving0L = "FixMHaving0L"

	// *ast.Join: ON xxx -> ON 1
	FixMOn1U = "FixMOn1U"
	// *ast.Join: ON xxx -> ON 0
	FixMOn0L = "FixMOn0L"

	// *ast.SetOprSelectList: remove Selects[1:] for UNION ALL
	FixMRmUnionAllL = "FixMRmUnionAllL"

	// *ast.PatternLikeExpr: normal char -> '_'|'%',  '_' -> '%'
	RdMLikeU = "RdMLikeU"
	// *ast.PatternLikeExpr: '%' -> '_'
	RdMLikeL = "RdMLikeL"

	// *ast.PatternRegexpExpr: '^'|'$' -> '', normal char -> '.', '+'|'?' -> '*'
	RdMRegExpU = "RdMRegExpU"
	// *ast.PatternRegexpExpr: '*' -> '+'|'?'
	RdMRegExpL = "RdMRegExpL"

	// EET (Equivalent Expression Testing) transformation mutations
	// Inspired by SQLancer's EET Oracle transformation rules

	// Rule 1: E → (p OR NOT p OR p IS NULL) AND E (tautology wrapping)
	// The left side is always TRUE (three-valued logic), so E's result set is contained in mutated result set.
	FixMAndTrueU = "FixMAndTrueU"

	// Rule 2: E → (p AND NOT p AND p IS NOT NULL) OR E (contradiction wrapping)
	// The left side is always FALSE, so mutated result set equals E's result set.
	// Under Implication Oracle: original ⊇ mutated (shrinking)
	FixMOrFalseL = "FixMOrFalseL"

	// Rule 4: E → CASE WHEN TRUE THEN E ELSE rand END (true branch wrapping)
	// Always evaluates to E. Under Implication Oracle: mutated ⊇ original (expanding)
	FixMCaseTrueU = "FixMCaseTrueU"

	// Rule 3: E → CASE WHEN FALSE THEN rand ELSE E END (false branch wrapping)
	// Always evaluates to E. Under Implication Oracle: original ⊇ mutated (shrinking)
	FixMCaseFalseL = "FixMCaseFalseL"

	// Rule 5/6: E → CASE WHEN rand THEN E ELSE E END (random branch, semantically equivalent)
	// Theoretically: both branches return E, so result should be identical.
	// If not identical → bug detected. This is an "equivalence" mutation.
	FixMCaseRandEq = "FixMCaseRandEq"
)

// 1. --------------------------------------------------
// *ast.BinaryOperationExpr:
//
// a {>|>=} b -> (a) + 1 {>|>=} (b) + 0
//
// a {<|<=} b -> (a) + 0 {<|<=} (b) + 1
// FixMCmpU = "FixMCmpU"
// 2. --------------------------------------------------
// *ast.BinaryOperationExpr:
//
// a {>|>=} b -> (a) + 0 {>|>=} (b) + 1
//
// a {<|<=} b -> (a) + 1 {<|<=} (b) + 0
//
// FixMCmpL = "FixMCmpL"
// 3. --------------------------------------------------
// *ast.CompareSubqueryExpr: ALL true -> false
// FixMCmpSubU = "FixMCmpSubU"
// 4. --------------------------------------------------
// *ast.CompareSubqueryExpr: ALL false -> true
// FixMCmpSubL = "FixMCmpSubL"
// 5. --------------------------------------------------
// *ast.BetweenExpr:
//   expr between l and r
//   ->
//   (expr) >= l and (expr) <= r
//   -> FixMCmpU, 1 and and (expr) <= r, (expr) >= l and 1 )
// RdMBetweenU = "RdMBetweenU"
// 6. --------------------------------------------------
// *ast.BetweenExpr:
//   expr between l and r
//   ->
//   (expr) >= l and (expr) <= r
//   -> FixMCmpOpL / FixMCmpL )
// RdMBetweenL = "RdMBetweenL"
// 7. --------------------------------------------------
// *ast.PatternInExpr: in(x,x,x) -> in(x,x,x,...)
// RdMInU = "RdMInU"
// 8. --------------------------------------------------
// *ast.PatternInExpr: in(x,x,x,...) -> in(x,x,x)
// RdMInL = "RdMInL"