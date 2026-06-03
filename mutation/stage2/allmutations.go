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

	// EET semantic rewrite mutations (equivalence class)
	// Inspired by SQLancer's EET Oracle semantic rewrite rules

	// De Morgan's Law: (A AND B) → NOT(NOT(A) OR NOT(B))
	// Semantically equivalent. If not identical → bug detected.
	FixMDeMorganAnd = "FixMDeMorganAnd"

	// De Morgan's Law: (A OR B) → NOT(NOT(A) AND NOT(B))
	// Semantically equivalent. If not identical → bug detected.
	FixMDeMorganOr = "FixMDeMorganOr"

	// BETWEEN → Comparison: x BETWEEN a AND b → (x >= a) AND (x <= b)
	// Semantically equivalent. If not identical → bug detected.
	FixMBetweenToCmp = "FixMBetweenToCmp"

	// BETWEEN → Drop Upper Bound: x BETWEEN a AND b → x >= a
	// Implication (upper): satisfying both bounds ⊆ satisfying lower bound. If containment violated → bug detected.
	FixMBetweenDropUpperU = "FixMBetweenDropUpperU"

	// BETWEEN → Drop Lower Bound: x BETWEEN a AND b → x <= b
	// Implication (upper): satisfying both bounds ⊆ satisfying upper bound. If containment violated → bug detected.
	FixMBetweenDropLowerU = "FixMBetweenDropLowerU"

	// NULL-safe Equality → Normal Equality: a <=> b → a = b
	// Implication (lower): = result ⊆ <=> result (a=b TRUE ⊆ a<=>b TRUE). If containment violated → bug detected.
	FixMNullEqToLowerL = "FixMNullEqToLowerL"

	// ALL → ANY/SOME: x > ALL(subq) → x > ANY(subq)
	// Implication (upper): ALL result ⊆ ANY result (satisfying ALL values ⊆ satisfying SOME value)
	// Warning: NULL boundary may break containment (empty subquery: ALL→TRUE, ANY→FALSE). Accept false positive risk.
	FixMAllToAnyU = "FixMAllToAnyU"

	// ANY → ALL: x > ANY(subq) → x > ALL(subq)
	// Implication (lower): ANY result ⊇ ALL result (satisfying SOME value ⊇ satisfying ALL values)
	// Warning: NULL boundary may break containment. Accept false positive risk.
	FixMAnyToAllL = "FixMAnyToAllL"

	// COALESCE → CASE: COALESCE(a, b) → CASE WHEN a IS NOT NULL THEN a ELSE b END
	// Semantically equivalent. If not identical → bug detected.
	FixMCoalesceToCase = "FixMCoalesceToCase"

	// NULLIF → CASE: NULLIF(a, b) → CASE WHEN a = b THEN NULL ELSE a END
	// Semantically equivalent. If not identical → bug detected.
	FixMNullifToCase = "FixMNullifToCase"

	// EXISTS → IN: EXISTS(subquery) → lhs IN (subquery) with NULL-safe CASE wrapping
	// Semantically equivalent (with NULL safety). If result sets differ → bug detected.
	FixMExistsToIn = "FixMExistsToIn"

	// IN → EXISTS: lhs IN (subquery) → EXISTS(subquery WHERE lhs = col AND pred)
	// Semantically equivalent (with NULL safety). If result sets differ → bug detected.
	FixMInToExists = "FixMInToExists"

	// GaussDB-M specific EET mutations (equivalence class)
	// These exploit behavioral differences between GaussDB-M and standard MySQL

	// IF → CASE: IF(cond, a, b) → CASE WHEN cond THEN a ELSE b END
	// Semantically equivalent in M mode. If result sets differ → bug detected.
	FixMIfToCase = "FixMIfToCase"

	// CONCAT → Pipe: CONCAT(a, b) → a || b
	// Semantically equivalent in M mode (may differ in NULL handling). If result sets differ → bug detected.
	FixMConcatToPipe = "FixMConcatToPipe"
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