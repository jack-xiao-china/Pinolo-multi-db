# Pinolo EET 增强可行性评估方案

## 架构约束前提

在评估每个增强项之前，必须明确 Pinolo 的核心架构约束：

### Parser 体系

| DBMS | Parser | Connector | Stage1 | Stage2 |
|------|--------|-----------|---------|---------|
| MySQL | TiDB parser v5.4.2 | go-sql-driver/mysql | InitAndExec() | MutateAllAndExec() |
| PostgreSQL | pg_query v6 (PG16) | pgx/v5 | InitForPostgreSQLAndExec() | MutateAllAndExecForPostgreSQL() |
| GaussDB-M | TiDB parser | openGauss-connector-go-pq | InitAndExecForMMode() | MutateAllAndExecForMMode() |
| GaussDB-A | TiDB parser (Oracle预处理后) | openGauss-connector-go-pq | InitForAMode() | MutateAllAndExec() |

**关键约束**：
- TiDB parser 无法解析 INTERSECT/EXCEPT、PG POSIX 正则(~)、Oracle 专有语法
- pg_query 无法解析 MySQL 专有函数(IF/IFNULL)
- TiDB parser 将 `opcode.LogicOr` 渲染为 `OR` 而非 `||`（已知限制）
- GaussDB-A 经 Oracle 预处理后用 TiDB parser 解析，限制了 A-mode 特有变异的可达性

### exprReplacer 机制约束

当前 `exprReplacer` 可替换的目标节点类型：

| 已支持 | 未支持 |
|--------|--------|
| SelectStmt.Where/Having.Expr | FuncCallExpr.FnName/Args |
| OnCondition.Expr | BetweenExpr.Expr/Left/Right |
| SelectField.Expr | CaseExpr.WhenClauses/ElseClause |
| BinaryOperationExpr.L/R | SubqueryExpr |
| UnaryOperationExpr.V | ValueExpr (常量) |
| ParenthesesExpr.Expr | **SetOperationStmt** (INTERSECT/EXCEPT) |
| IsNullExpr.Expr | RangeVar (表引用) |
| IsTruthExpr.Expr | |
| PatternInExpr.Expr | |
| PatternLikeExpr.Expr | |
| PatternRegexpExpr.Expr | |
| CompareSubqueryExpr.L | |
| HavingClause.Expr | |

**扩展 exprReplacer 是大部分增强的前提**——需要在 Leave() 中新增更多 case。

### Oracle 约束

- **Implication Oracle (Check)**：检测结果集包含关系违反，用于 upper/lower 变异
- **Equivalence Oracle (CheckEquivalence)**：检测结果集完全相等违反，用于语义等价变换
- **两者的 NULL 语义**：Result.Rows 中 NULL 表示为字符串 `"NULL"`，所以 `NULL ≠ NULL` 在 CMP 中被视为不相等

### Visitor 递归停止约束

当前 visitor 在以下节点停止递归（不继续访问子节点）：
- 数值运算 (+, -, *, /, DIV, MOD, |, &, ^ 等)
- 比较运算 (=, >=, >, <=, <, !=, <=>, IS NULL, IN, BETWEEN, LIKE, REGEXP)
- 子查询（无 ANY/ALL/SOME/IN/EXISTS）
- CASE/IF 控制流
- **所有函数调用**（除了 COALESCE/NULLIF 在 mining 中特殊处理）

**增强函数类 EET 变换需要修改此策略**：对目标函数不再停止递归，改为在 mining 函数中添加等价变换候选。

---

## 一、高优先级增强项可行性评估

### H3: IFNULL → COALESCE 等价变换

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 函数存在 | ✅ IFNULL(a,b) | ❌ 无IFNULL | ✅ IFNULL(a,b) | ❌ 用NVL(a,b) |
| Parser支持 | ✅ TiDB解析为FuncCallExpr | N/A | ✅ TiDB解析 | ✅ 预处理后 |
| 语义等价证明 | ✅ IFNULL(a,b) ≡ COALESCE(a,b) | N/A | ✅ | ✅ NVL(a,b) ≡ COALESCE(a,b) |
| NULL处理 | 两者均：a非NULL→a，a为NULL→b | N/A | ✅ 一致 | ✅ 一致 |
| 实现难度 | **低** | N/A | **低** | **低** (NVL→COALESCE) |

**等价性证明**：
```
IFNULL(a, b):
  a IS NOT NULL → 返回 a
  a IS NULL     → 返回 b

COALESCE(a, b):
  a IS NOT NULL → 返回 a
  a IS NULL     → 返回 b

完全等价，无 NULL 边界差异。
```

**实现方案**：
1. **Visitor 修改**：在 `visitFuncCallExpr` 中，当函数名为 `IFNULL` 且参数数量为 2 时，添加 `FixMIFNullToCoalesce` 候选
2. **Mutation 实现**：
   ```go
   // doFixMIFNullToCoalesce: IFNULL(a,b) → COALESCE(a,b)
   // 替换整个 FuncCallExpr 节点
   func doFixMIFNullToCoalesce(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
       expr := in.(*ast.FuncCallExpr)
       if expr.FnName.L != "IFNULL" || len(expr.Args) != 2 {
           return nil, errors.New("[FixMIFNullToCoalesce]expected IFNULL with 2 args")
       }
       newExpr := &ast.FuncCallExpr{
           FnName: ast.NewCIStr("COALESCE"),
           Args:   expr.Args,  // 共享参数节点
       }
       parenExpr := &ast.ParenthesesExpr{Expr: newExpr}
       replaceExprInRoot(rootNode, expr, parenExpr)
       sql, err := restore(rootNode)
       replaceExprInRoot(rootNode, parenExpr, expr)  // 恢复原始
       return sql, nil
   }
   ```
3. **exprReplacer 需新增**：`*ast.FuncCallExpr` case（替换整个函数调用节点）
4. **GaussDB-A**：同理实现 `FixMNVLToCoalesce`，将 `NVL(a,b)` → `COALESCE(a,b)`
5. **常量注册**：`allmutations.go` 新增 `FixMIFNullToCoalesce`, `FixMNVLToCoalesce`
6. **等价性标记**：加入 `isEquivalenceMutation()` 列表

**优势**：
- 实现极简单，IFNULL/COALESCE 是 MySQL/GaussDB-M 的高频函数
- 等价性无争议，NULL 处理完全一致
- 可快速验证工具正确性（应发现 0 bug）

**劣势**：
- IFNULL→COALESCE 太"显而易见"，DBMS 实现通常共享代码路径，bug 概率较低
- 但在高版本 MySQL 中 IFNULL 可能走不同优化器路径（索引选择不同），仍有发现 bug 价值

**可行性结论**：✅ MySQL/GaussDB-M/GaussDB-A 均可实现，PostgreSQL 不适用。**推荐立即实施**。

---

### H2: INTERSECT → EXISTS / EXCEPT → NOT EXISTS 等价变换

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 语法支持 | ❌ 无INTERSECT/EXCEPT | ✅ | ❌ 无INTERSECT | ✅ (Oracle兼容) |
| Parser支持 | ❌ TiDB无法解析 | ✅ pg_query解析SetOperationStmt | ❌ | ❌ TiDB无法解析 |
| 语义等价证明 | N/A | ✅ (需NULL-safe列等值) | N/A | ❌ parser限制 |
| NULL处理 | N/A | ✅ 列等值需用 `a=b OR (a IS NULL AND b IS NULL)` | N/A | N/A |
| 实现难度 | N/A | **中** | N/A | ❌ 不可行 |

**等价性证明**（PostgreSQL）：
```sql
-- 原始：
SELECT c1, c2 FROM t1 WHERE p1 INTERSECT SELECT c1, c2 FROM t2 WHERE p2

-- 等价变换：
SELECT c1, c2 FROM t1 WHERE p1 AND EXISTS(
    SELECT 1 FROM t2 WHERE p2
      AND (t1.c1 = t2.c1 OR (t1.c1 IS NULL AND t2.c1 IS NULL))
      AND (t1.c2 = t2.c2 OR (t1.c2 IS NULL AND t2.c2 IS NULL))
)

-- EXCEPT → NOT EXISTS:
SELECT c1, c2 FROM t1 WHERE p1 AND NOT EXISTS(
    SELECT 1 FROM t2 WHERE p2
      AND (t1.c1 = t2.c1 OR (t1.c1 IS NULL AND t2.c1 IS NULL))
      AND (t1.c2 = t2.c2 OR (t1.c2 IS NULL AND t2.c2 IS NULL))
)
```

**NULL 处理关键**：INTERSECT/EXCEPT 中两个 NULL 被视为相等，但 `NULL = NULL` 返回 NULL（非 TRUE），所以必须用 `a=b OR (a IS NULL AND b IS NULL)` 实现 NULL-safe 等值比较。

**实现方案（PostgreSQL）**：
1. **pg_mutatevisitor 修改**：在 `visitSelectStmt` 中检测 `SetOperationStmt`，当 `op=INTERSECT` 或 `op=EXCEPT` 时添加候选
2. **Mutation 实现**：
   - 需要解析两个子查询的列名列表
   - 构建 NULL-safe 列等值条件
   - 用 EXISTS/NOT EXISTS 替换整个 SetOperationStmt
   - 需要新增 `pg_replaceExprInRoot` 的 PG 版本（当前 PG 变异用 JSON patch 而非 AST 替换）
3. **PG 变异引擎差异**：PG stage2 用 `pgquery.Deparse()` + JSON patch 模式，不像 MySQL 用 AST 替换+restore 模式。需要为 INTERSECT/EXCEPT 变换实现专门的 JSON patch 或采用类似 MySQL 的 AST 构建方式
4. **exprReplacer PG 版本需新增**：`SetOperationStmt` 处理

**优势**：
- INTERSECT/EXCEPT 是 PG/GaussDB-A 的核心集合操作，优化器处理路径不同于 EXISTS 子查询
- SQLancer 已验证此变换能发现 PG 真实 bug
- NULL-safe 列等值比较触发 IS NULL 优化路径，bug 概率较高

**劣势**：
- **PG 变异引擎需要较大改造**：当前 PG 用 JSON patch 而非 AST 替换，INTERSECT/EXCEPT 变换需要从 SetOperationStmt 拆解出两个子查询并构建全新的 SELECT 结构
- **GaussDB-A 因 TiDB parser 限制不可行**：A-mode 支持 INTERSECT/EXCEPT 语法，但 TiDB parser 无法解析，需改用 pg_query 或实现 Oracle→PG 语法预处理
- 实现复杂度中等偏高

**可行性结论**：
- ✅ PostgreSQL 可实现，但需改造 PG stage2 引擎
- ❌ MySQL/GaussDB-M 不适用（无 INTERSECT/EXCEPT 语法）
- ❌ GaussDB-A 当前不可行（TiDB parser 限制）
- **推荐作为 Phase 2 实施，优先解决 PG stage2 引擎改造**

---

### H4: ANY/ALL/SOME 子查询等价变换

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 语法支持 | ✅ x = ANY(subq) | ✅ x = ANY(subq) | ✅ | ✅ |
| Parser支持 | ✅ CompareSubqueryExpr | ✅ SubLink ANY_SUBLINK | ✅ | ✅ (预处理后) |
| 等价变换 | 部分 | 部分 | 部分 | 部分 |
| NULL处理 | ⚠️ 需CASE包装 | ⚠️ 需CASE包装 | ⚠️ | ⚠️ |
| 实现难度 | **中** | **中** | **中** | **中** |

**可实现的等价变换**：

| 变换 | 等价性 | NULL 影响 |
|------|--------|-----------|
| `x = ANY(subq)` → `x IN (subq)` | ✅ 严格等价 | IN 和 = ANY 对 NULL 处理一致（子查询含 NULL 时，若 x 不匹配任何非NULL值则返回 NULL） |
| `x <> ALL(subq)` → `x NOT IN (subq)` | ✅ 严格等价 | 同上 |
| `x > ANY(subq)` → `EXISTS(subq WHERE col > x)` | ⚠️ 近似等价 | x > ANY 返回 NULL 当子查询空或含 NULL；EXISTS 不受 NULL 影响。需 CASE 包装处理 NULL |
| `x > ALL(subq)` → `NOT EXISTS(subq WHERE col <= x)` | ⚠️ 近似等价 | 同上 |

**实现方案**：
1. **MySQL/GaussDB-M**：
   - visitor 在 `visitCompareSubqueryExpr` 中，当 `CompareSubqueryExpr` 的 `Op` 为 `=且 SubqueryOp 为 ANY/SOME` 时，添加 `FixMEqAnyToIn` 候选
   - 变换：`x = ANY(subq)` → `x IN (subq)`（简单替换运算符和 ALL/ANY 标记）
   - `x <> ALL(subq)` → `x NOT IN (subq)`同理
   
2. **PostgreSQL**：
   - pg_mutatevisitor 在 `visitSubLink` 中，当 `subLinkType=ANY_SUBLINK` 且运算符为 `=` 时添加候选
   - 变换：`x = ANY(subq)` → `x IN (subq)`
   
3. **exprReplacer 需新增**：`*ast.CompareSubqueryExpr` 的完整替换（当前只替换 .L 字段，需替换整个节点）

**优势**：
- = ANY → IN 是严格等价，不需 NULL 特殊处理
- ANY/ALL 子查询触发不同优化器路径（子查询执行策略：materialization vs semi-join）
- MySQL/MariaDB/TiDB 在 ANY/ALL 优化上有已知 bug 历史

**劣势**：
- `> ANY` / `> ALL` 变换需要 NULL-safe CASE 包装，实现更复杂
- 当前 visitor 对 CompareSubqueryExpr 只做 FixMCmpOpU/L 变换，需要扩展 mining 逻辑
- 子查询等价变换可能受 Stage1 子查询深度限制影响

**可行性结论**：✅ 四款 DBMS 均可实现 `= ANY → IN` 和 `<> ALL → NOT IN` 等价变换。`> ANY` / `> ALL` 变换需 NULL 包装，难度更高。**推荐先实施简单等价变换，后续扩展**。

---

### H1: JSON/JSONB 类型与函数

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 语法支持 | ✅ (MySQL 8.0+) | ✅ (PG 9.4+) | ✅ | ✅ |
| Parser支持 | ⚠️ TiDB解析JSON函数为FuncCallExpr | ⚠️ pg_query解析->/->>为OpExpr | ⚠️ | ⚠️ |
| EET变换 | ❌ 无简单等价 | ❌ 无简单等价 | ❌ | ❌ |
| 表达式生成增强 | ✅ | ✅ | ✅ | ✅ |
| NULL处理 | 复杂 | 复杂 | 复杂 | 复杂 |
| 实现难度 | **高** | **高** | **高** | **高** |

**关键发现**：SQLancer 并没有 JSON 等价变换规则。JSON 函数只是作为"可生成表达式"存在，通过已有的 tautology/contradiction/CASE wrapping 来测试。

**重新定义增强目标**：

JSON 增强不是 EET 变换，而是**表达式生成多样性增强**：
1. 在 generator 中增加 JSON 类型和 JSON 函数生成
2. 在 visitor 中对 JSON 函数调用不再停止递归，而是将其作为 wrapping 候选
3. 这样 JSON 表达式就能被 FixMAndTrueU/FixMOrFalseL/FixMCaseTrueU/L/RandEq 包裹测试

**JSON 函数潜在的等价变换**（需逐一验证 NULL 语义）：

| 变换 | 等价性验证 | 结论 |
|------|-----------|------|
| JSON_EXTRACT(col, '$.key') → col->'$.key' | MySQL: -> 返回 JSON 类型，JSON_EXTRACT 返回 JSON 类型 | ✅ 等价（但 TiDB parser 不解析 ->） |
| JSON_EXTRACT(col, '$.key') → col->>'$.key' | ->> 返回字符串并去引号，-> 返回 JSON 类型 | ❌ 不等价（类型不同） |
| JSON_VALID('null') → 1 | 常量等价，无意义 | ❌ 无测试价值 |
| JSON_TYPE(JSON_OBJECT('k', v)) → 'OBJECT' | 常量等价 | ❌ 无测试价值 |

**结论**：JSON 函数之间**几乎没有严格等价变换**。JSON 增强主要是**生成多样性**，而非等价变换。

**实现方案**：
1. **Generator 增强**（expr_gen.go）：
   ```go
   jsonFuncs := []string{"JSON_TYPE", "JSON_VALID", "JSON_EXTRACT", 
                          "JSON_ARRAY", "JSON_OBJECT", "JSON_CONTAINS"}
   // 生成 JSON 常量值：null, 1, "str", {"k":v}, [1,2]
   ```
2. **DDL 增强**（ddl_gen.go）：新增 JSON 类型列
   ```go
   // MySQL: col JSON
   // PG: col JSONB
   ```
3. **Visitor 修改**：在 `visitFuncCallExpr` 中，当函数名不是 COALESCE/NULLIF/IFNULL 时，如果函数返回布尔类型或可以参与布尔表达式，允许作为 wrapping 候选（而非停止递归）

**优势**：
- JSON 是现代 DBMS 核心功能，优化器有独立 JSON 处理路径
- JSON 表达式通过 wrapping 测试可以暴露 JSON 优化器 bug
- MySQL 8.0+ JSON 函数有已知 bug 历史

**劣势**：
- **TiDB parser 不支持 -> 和 ->> 操作符解析**，无法实现 JSON_EXTRACT→-> 等价变换
- **pg_query 解析 JSON 操作符为 OpExpr**，但 Deparse 可能改变 JSON 操作符语法
- JSON 函数 NULL 语义复杂（JSON null vs SQL NULL），可能导致等价 oracle 误报
- 实现工作量高（需要新增整个 JSON 生成体系）

**可行性结论**：
- ✅ 四款 DBMS 均可实现**表达式生成增强**（中等难度）
- ❌ 四款 DBMS 均难以实现**JSON 等价变换**（parser 限制 + NULL 语义复杂）
- **推荐先做表达式生成增强，等价变换作为远期目标**

---

### H5: ENUM 类型

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 语法支持 | ✅ ENUM('a','b','c') | ✅ (需CREATE TYPE) | ✅ | ❌ Oracle无ENUM |
| Parser支持 | ✅ TiDB解析ENUM | ✅ pg_query解析 | ✅ | ❌ |
| 生成增强 | ✅ 简单 | ⚠️ 中等(需CREATE TYPE) | ✅ | ❌ |
| EET变换 | ❌ 无简单等价 | ❌ | ❌ | ❌ |
| 实现难度 | **低**(MySQL) / **中**(PG) | **中** | **低** | ❌ |

**分析**：ENUM 值没有语义等价变换规则。增强是**生成多样性**：ENUM 列参与 WHERE 条件比较时，触发优化器对 ENUM 索引的选择路径。

**MySQL/GaussDB-M 实现方案**：
1. Generator 中新增 ENUM 列类型：`ENUM('a','b','c','d','e')`
2. 表达式生成中 ENUM 列的常量值从 ENUM 值列表中随机选取
3. 无需修改 visitor（ENUM 列参与比较时已由现有比较变异覆盖）

**PostgreSQL 实现方案**：
1. DDL 生成需先 `CREATE TYPE mood AS ENUM ('sad','ok','happy')`
2. 表达式生成中 PG ENUM 需类型标注 `'ok'::mood`
3. 更复杂，但 PG ENUM 优化器路径不同于普通字符串，有 bug 潜力

**优势**：MySQL ENUM 是高频数据类型，优化器对 ENUM 索引有专门处理路径。

**劣势**：PG ENUM 需要 CREATE TYPE，增加 DDL 生成复杂度；无等价变换。

**可行性结论**：✅ MySQL/GaussDB-M 低难度可实施。⚠️ PostgreSQL 中等难度。❌ GaussDB-A 不适用。**推荐作为生成增强在 Phase 1 实施**。

---

## 二、快速增强项可行性评估

### M1: LEAST/GREATEST → CASE 等价变换

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 函数支持 | ✅ LEAST/GREATEST | ✅ | ✅ | ✅ |
| Parser支持 | ✅ FuncCallExpr | ✅ FuncCall | ✅ | ✅ |
| 等价性 | ⚠️ **不完全等价** | ⚠️ **不完全等价** | ⚠️ | ⚠️ |
| NULL处理 | ❌ LEAST(NULL,1)=NULL ≠ CASE结果 | ❌ | ❌ | ❌ |
| 实现难度 | **中** (需NULL包装) | **中** | **中** | **中** |

**等价性反证**：
```sql
-- MySQL:
SELECT LEAST(NULL, 1)     → NULL
SELECT CASE WHEN NULL <= 1 THEN NULL ELSE 1 END  → 1 (CASE将NULL视为FALSE)

-- PostgreSQL:
SELECT LEAST(NULL, 1)     → NULL
SELECT CASE WHEN NULL <= 1 THEN NULL ELSE 1 END  → 1 (同MySQL)
```

**严格等价变换**（需 NULL 包装）：
```sql
-- LEAST(a, b) → 
CASE 
  WHEN a IS NULL THEN NULL
  WHEN b IS NULL THEN NULL
  WHEN a <= b THEN a 
  ELSE b 
END

-- GREATEST(a, b) →
CASE
  WHEN a IS NULL THEN NULL
  WHEN b IS NULL THEN NULL  
  WHEN a >= b THEN a
  ELSE b
END
```

**实现方案**：
1. Visitor 在 `visitFuncCallExpr` 中识别 LEAST/GREATEST
2. 构建含 NULL 检查的 CASE 表达式
3. exprReplacer 新增 `*ast.FuncCallExpr` 整体替换

**优势**：
- LEAST/GREATEST 是 MySQL/PG 高频函数
- 含 NULL 检查的 CASE 变换比简单 LEAST 调用触发更多优化器分支
- NULL 包装本身暴露 IS NULL 优化路径的 bug

**劣势**：
- 必须包含 NULL 检查才能等价，否则 oracle 会误报
- LEAST 可接受 >2 个参数（variadic），>2 参数的 CASE 变换更复杂
- 当前只支持 2 参数版本，>2 参数需递归 CASE

**可行性结论**：✅ 四款 DBMS 均可实现（需 NULL-safe CASE 包装），但**等价性需严格验证**。2 参数版本推荐实施，>2 参数版本作为扩展。

---

### M8: IS TRUE → = TRUE 等价变换

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 等价性 | ❌ **不等价** | ✅ 等价 | ❌ 不等价 | ✅ 等价(布尔类型) |
| 原因 | MySQL TRUE=1, 2 IS TRUE ≠ 2=1 | PG TRUE是布尔值 | 同MySQL | A-mode有BOOLEAN类型 |

**不等价证明**（MySQL）：
```sql
-- MySQL:
SELECT 2 IS TRUE    → 1 (TRUE，任何非零非NULL值)
SELECT 2 = TRUE     → 0 (FALSE，2 ≠ 1)
SELECT -1 IS TRUE   → 1 (TRUE)
SELECT -1 = TRUE    → 0 (FALSE)
```

**等价证明**（PostgreSQL）：
```sql
-- PostgreSQL:
SELECT TRUE IS TRUE  → TRUE
SELECT TRUE = TRUE   → TRUE
SELECT FALSE IS TRUE → FALSE
SELECT FALSE = TRUE  → FALSE
SELECT NULL::boolean IS TRUE → NULL
SELECT NULL::boolean = TRUE   → NULL
-- 完全一致
```

**实现方案（仅 PG/GaussDB-A）**：
1. pg_mutatevisitor 在处理 BooleanTest（IS TRUE/IS FALSE）节点时添加 `FixMIsTrueToEqTrue_Pg` 候选
2. 变换：`expr IS TRUE` → `expr = TRUE`，`expr IS FALSE` → `expr = FALSE`
3. 同理：`expr IS NOT TRUE` → `expr <> TRUE`（PG 中 `<>` 等价于 `!=`）

**优势**：PG 中 IS TRUE 和 = TRUE 虽语义等价，但可能走不同优化器路径（IS TRUE 可能用 BooleanTest 优化，= TRUE 用普通等值比较优化）

**劣势**：
- MySQL/GaussDB-M **不可实施**（不等价）
- PG BooleanTest 节点在 pg_query 中的 JSON 表示可能需要额外处理

**可行性结论**：✅ PostgreSQL 和 GaussDB-A 可实施。❌ MySQL 和 GaussDB-M 不可实施。

---

### M9: NOT ISNULL(x) → x IS NOT NULL

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| ISNULL函数 | ✅ ISNULL(x)返回0/1 | ❌ 无ISNULL函数 | ✅ | ❌ 用NVL |
| 等价性 | ✅ 在布尔上下文等价 | N/A | ✅ | N/A |
| Parser支持 | ✅ FuncCallExpr | N/A | ✅ | N/A |
| 实现难度 | **低** | N/A | **低** | N/A |

**等价性证明**（MySQL 布尔上下文）：
```sql
-- MySQL:
SELECT NOT ISNULL(NULL)    → 0 (ISNULL(NULL)=1, NOT 1=0)
SELECT NULL IS NOT NULL    → 0 (FALSE)
SELECT NOT ISNULL(1)       → 1 (ISNULL(1)=0, NOT 0=1)
SELECT 1 IS NOT NULL       → 1 (TRUE)
-- 布尔上下文中等价
```

**注意**：在非布尔上下文中不完全等价：
```sql
SELECT NOT ISNULL(1)    → 1 (整数1)
SELECT 1 IS NOT NULL    → TRUE (布尔TRUE，但MySQL中TRUE=1)
-- MySQL 中布尔和整数互通，实际值相同
```

**实现方案**：
1. Visitor 在遇到 `UnaryOperationExpr{Op: NOT}` 包裹 `FuncCallExpr{FnName: "ISNULL"}` 时，添加 `FixMNotIsnullToIsNotNull` 倒选
2. 变换：`NOT ISNULL(x)` → `x IS NOT NULL`
3. 构建新 AST：`&ast.IsNullExpr{Expr: x, Not: true}`

**优势**：极简变换，ISNULL 函数触发不同优化路径

**劣势**：MySQL 中 ISNULL 是简单函数，优化器差异可能不大

**可行性结论**：✅ MySQL/GaussDB-M 可实施。PostgreSQL/GaussDB-A 不适用。

---

### M5: SELF JOIN 查询形态

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 语法支持 | ✅ | ✅ | ✅ | ✅ |
| Parser支持 | ✅ TiDB解析 | ✅ pg_query解析 | ✅ | ✅ |
| 生成增强 | ✅ | ✅ | ✅ | ✅ |
| 实现难度 | **低** | **低** | **低** | **低** |

**实现方案**：
1. Generator 中 `from_gen.go` 新增 self join 生成：
   ```go
   // FROM t1 AS a JOIN t1 AS b ON a.col = b.col
   ```
2. 无需修改 visitor（self join 的 WHERE/ON 条件已由现有变异覆盖）
3. Stage1 需注意：LEFT JOIN self join 会被转为 INNER JOIN（这是现有行为）

**优势**：
- Self join 暴露优化器对同表引用的处理（别名解析、索引选择策略）
- 实现极简单

**劣势**：
- Self join 查询在现有变异中已自动覆盖 WHERE/ON 条件
- 纯生成增强，不是等价变换

**可行性结论**：✅ 四款 DBMS 均可实施，**极低难度**。

---

### M2: NULL-safe 等值 <=> → IS NOT DISTINCT FROM

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| <=> 支持 | ✅ | ❌ (用IS NOT DISTINCT FROM) | ✅ | ❌ |
| Parser支持 | ✅ TiDB解析为opcode.NullEQ | ❌ | ✅ | ❌ |
| 等价变换 | ⚠️ 需评估 | ⚠️ 需评估 | ⚠️ | ❌ |
| 实现难度 | **中** | **中** | **中** | ❌ |

**潜在等价变换**：
```sql
-- MySQL:
a <=> b → CASE WHEN a IS NULL AND b IS NULL THEN TRUE
           WHEN a IS NULL OR b IS NULL THEN FALSE
           WHEN a = b THEN TRUE ELSE FALSE END

-- 但此 CASE 在 MySQL 中类型为整数(1/0)，<=> 也是整数(1/0)
-- 严格等价 ✅
```

**更简化的等价变换**（PG）：
```sql
-- PostgreSQL:
a IS NOT DISTINCT FROM b → (a = b OR (a IS NULL AND b IS NULL))
-- 但 (a = b OR ...) 在 a,b 均为 NULL 时:
--   NULL = NULL → NULL, NULL IS NULL → TRUE, TRUE IS NULL → TRUE
--   NULL OR TRUE → TRUE ✅
-- 当 a NULL, b 非 NULL:
--   NULL = b → NULL, NULL IS NULL → TRUE, b IS NULL → FALSE
--   NULL OR (TRUE AND FALSE) → NULL OR FALSE → NULL ❌ 
--   而 IS NOT DISTINCT FROM 返回 FALSE
-- 所以 (a = b OR (a IS NULL AND b IS NULL)) 不完全等价！

-- 正确等价：
a IS NOT DISTINCT FROM b → 
  CASE WHEN a IS NULL AND b IS NULL THEN TRUE
       WHEN a IS NULL OR b IS NULL THEN FALSE
       WHEN a = b THEN TRUE ELSE FALSE END
```

**实现方案**：
1. MySQL visitor 在 `visitBinaryOperationExpr` 中，当 Op 为 `opcode.NullEQ` 时，添加 `FixMNullEqToCase` 候选
2. 变换：`a <=> b` → 含 NULL 检查的 CASE 表达式

**优势**：`<=>` 是 MySQL 专有运算符，优化器可能有专门处理路径

**劣势**：
- CASE 变换复杂，实现难度中等
- PG 的 IS NOT DISTINCT FROM 需类似但不同的 CASE 变换
- 两个变换都需要 NULL 检查包装

**可行性结论**：✅ MySQL/GaussDB-M 可实施 <=> → CASE。✅ PostgreSQL/GaussDB-A 可实施 IS NOT DISTINCT FROM → CASE。**中等难度**。

---

### M3: PG POSIX 正则 ~ / !~

| 维度 | MySQL | PostgreSQL | GaussDB-M | GaussDB-A |
|------|-------|------------|-----------|-----------|
| 语法支持 | ❌ MySQL用REGEXP | ✅ | ❌ | ✅ |
| Parser支持 | ❌ | ✅ pg_query解析为OpExpr | ❌ | ⚠️ 可能 |
| 等价变换 | N/A | ⚠️ 需评估 | N/A | ⚠️ |
| 实现难度 | N/A | **中** | N/A | **中** |

**潜在等价变换**（PostgreSQL）：
```sql
-- ~ (正则匹配) 无简单等价变换
-- 但可以做:
str ~ 'pattern' → str REGEXP 'pattern' (PG中 REGEXP 是 ~ 的别名？不，PG用SIMILAR TO)

-- 实际上 PG 中:
-- ~ 是 POSIX 正则匹配（区分大小写）
-- ~* 是 POSIX 正则匹配（不区分大小写）
-- SIMILAR TO 是 SQL 标准正则（类LIKE语法）
-- 这三者不等价
```

**结论**：POSIX 正则(~)没有简单等价变换。它只是**表达式生成多样性增强**——在 PG generator 中生成 POSIX 正则表达式，让现有 LIKE/REGEXP 变异覆盖它。

**但是**：当前 PG visitor 中 `visitAExpr` 只处理 LIKE(ILIKE) 和 comparison 运算符。需要新增对 ~ (POSIX 正则) 运算符的识别和 RdM 正则变异。

**实现方案**：
1. pg_mutatevisitor 在 `visitAExpr` 中新增 POSIX 正则运算符识别
2. pg_query 中 POSIX 正则是 OpExpr 节点，运算符名为 "~", "~*", "!~", "!~*"
3. 新增 `RdMPOSIXRegExpPgU/L` 变异：对正则模式字符串做随机变形
4. Generator 新增 POSIX 正则表达式生成

**可行性结论**：⚠️ 无等价变换可做。✅ 可做**随机正则变异**（类似 RdMRegExpU/L），但这是 implication 变异而非 EET 变异。**中等难度**。

---

## 三、综合优先级排序

### Phase 1（1-2天，低成本高确定性）

| 序号 | 增强项 | 类型 | MySQL | PG | GaussDB-M | GaussDB-A | 难度 | Bug潜力 |
|------|--------|------|-------|----|-----------|-----------|------|---------|
| **P1-1** | IFNULL→COALESCE | EET等价 | ✅ | ❌ | ✅ | ✅(NVL) | 低 | 中 |
| **P1-2** | NVL→COALESCE (A-mode) | EET等价 | ❌ | ❌ | ❌ | ✅ | 低 | 中 |
| **P1-3** | NOT ISNULL→IS NOT NULL | EET等价 | ✅ | ❌ | ✅ | ❌ | 低 | 低-中 |
| **P1-4** | = ANY→IN | EET等价 | ✅ | ✅ | ✅ | ✅ | 中 | 高 |
| **P1-5** | SELF JOIN 生成 | 生成增强 | ✅ | ✅ | ✅ | ✅ | 低 | 低-中 |
| **P1-6** | IS TRUE→=TRUE (PG/A) | EET等价 | ❌ | ✅ | ❌ | ✅ | 低 | 低-中 |

### Phase 2（3-5天，中等投入）

| 序号 | 增强项 | 类型 | MySQL | PG | GaussDB-M | GaussDB-A | 障碍 | Bug潜力 |
|------|--------|------|-------|----|-----------|-----------|------|---------|
| **P2-1** | INTERSECT→EXISTS (PG) | EET等价 | ❌ | ✅ | ❌ | ❌(parser) | PG引擎改造 | 高 |
| **P2-2** | EXCEPT→NOT EXISTS (PG) | EET等价 | ❌ | ✅ | ❌ | ❌(parser) | PG引擎改造 | 高 |
| **P2-3** | LEAST/GREATEST→CASE | EET等价 | ✅ | ✅ | ✅ | ✅ | NULL包装 | 中-高 |
| **P2-4** | <=>→CASE | EET等价 | ✅ | ❌ | ✅ | ❌ | NULL包装 | 中 |
| **P2-5** | <> ALL→NOT IN | EET等价 | ✅ | ✅ | ✅ | ✅ | 同P1-4 | 中 |
| **P2-6** | ENUM类型生成 | 生成增强 | ✅ | ⚠️ | ✅ | ❌ | PG需CREATE TYPE | 低-中 |

### Phase 3（5-10天，高投入）

| 序号 | 增强项 | 类型 | MySQL | PG | GaussDB-M | GaussDB-A | 障碍 | Bug潜力 |
|------|--------|------|-------|----|-----------|-----------|------|---------|
| **P3-1** | JSON表达式生成 | 生成增强 | ✅ | ✅ | ✅ | ✅ | 生成体系 | 中-高 |
| **P3-2** | JSON等价变换 | EET等价 | ⚠️ | ⚠️ | ⚠️ | ⚠️ | parser+NULL | 低-中 |
| **P3-3** | IS NOT DISTINCT FROM→CASE | EET等价 | ❌ | ✅ | ❌ | ✅ | NULL包装 | 中 |
| **P3-4** | GaussDB-A改用pg_query | 架构改造 | ❌ | ❌ | ❌ | ✅ | 大改造 | 高 |
| **P3-5** | POSIX正则变异 | Implication | ❌ | ✅ | ❌ | ✅ | PG引擎 | 低-中 |

---

## 四、架构改造需求汇总

### 必须的公共改造（Phase 1 前置）

1. **exprReplacer 新增 FuncCallExpr 整体替换**
   - 当前只能替换函数参数位置，不能替换整个函数调用节点
   - IFNULL→COALESCE、LEAST→CASE 等变换都需要替换整个 FuncCallExpr
   - 改造：在 `exprReplacer.Leave()` 中新增 `case *ast.FuncCallExpr` 处理
   - **注意**：FuncCallExpr 可能出现在 SelectField、WHERE、HAVING、ON、BinaryOperationExpr.L/R 等多种父节点中，需逐一验证

2. **PG 变异引擎改造**
   - 当前 PG 用 JSON patch + Deparse 模式
   - INTERSECT/EXCEPT、IS TRUE→=TRUE 等变换需要构建全新 SQL 结构
   - 两种方案：
     - (a) 继续用 JSON patch：构建完整 SelectStmt JSON 结构，patch 替换
     - (b) 改用类似 MySQL 的 AST 替换模式：`pg_replaceExprInRoot(rootNode, target, replacement)` + `pgquery.Deparse()`
   - **推荐方案 (b)**：更灵活，且 MySQL 体系已验证可行

3. **visitor 函数递归策略修改**
   - 当前所有函数调用（除 COALESCE/NULLIF）都停止递归
   - 需要改为：对有等价变换的函数（IFNULL、LEAST、GREATEST）添加 mining 逻辑
   - 对无等价变换但有 bug 潜力的函数（JSON 函数），允许作为 wrapping 候选（不停递归，但只添加 tautology/contradiction/CASE wrapping 候选）

4. **isEquivalenceMutation 列表更新**
   - 每个新增 EET 变异常量都需要加入 `isEquivalenceMutation()` 和 `isEquivalenceMutationPg()` 列表

### GaussDB-A 架构改造（Phase 3）

当前 GaussDB-A 用 TiDB parser 处理 Oracle 预处理后的 SQL。这限制了：
- 无法测试 INTERSECT/EXCEPT（TiDB parser 不解析）
- 无法测试 PG 特有运算符（POSIX 正则等）
- 无法测试 Oracle 专有语法（CONNECT BY, ROWNUM 等）

**改造方案**：GaussDB-A 改用 pg_query parser（因为 GaussDB-A 的 Oracle 兼容模式底层基于 openGauss，其 SQL 解析与 PostgreSQL 共享基础语法）

**改造难度**：高（需要重写 A-mode Stage1/Stage2/Visitor 整套体系）

**收益**：
- 可实施 INTERSECT→EXISTS / EXCEPT→NOT EXISTS
- 可实施 IS TRUE→=TRUE 等价变换
- 可测试 Oracle 特有语法（DECODE→CASE, NVL→COALESCE 等）
- 可发现更多 A-mode 优化器 bug

---

## 五、风险评估

### EET 等价变换的通用风险

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| **等价性误判** | oracle 误报 bug | 每个变换必须用 NULL 测试用例验证等价性 |
| **NULL 语义差异** | "等价"变换实际不等价 | 3VL(三值逻辑)分析：测试 NULL 输入在所有分支的行为 |
| **类型差异** | 返回类型不同导致 CMP 不匹配 | 确保变换前后返回类型一致（如 LEAST→CASE 都返回同一类型） |
| **Parser 渲染差异** | 生成的 SQL 语法不正确 | 每个 doFix 函数需验证 restore() 输出可被目标 DBMS 执行 |
| **exprReplacer 遗漏** | 替换失败导致变异不生效 | 每个变换需覆盖所有可能的父节点类型 |

### NULL 语义验证方法论

对每个新增 EET 变换，必须验证以下 NULL 边界情况：

```
输入a=NULL, 输入b=NULL  → 变换前后结果是否一致？
输入a=NULL, 输入b=非NULL → 变换前后结果是否一致？
输入a=非NULL, 输入b=NULL → 变换前后结果是否一致？
输入a=非NULL, 输入b=非NULL → 变换前后结果是否一致？
子查询返回空集 → 变换前后结果是否一致？
子查询返回含NULL行 → 变换前后结果是否一致？
```

---

## 六、最终推荐路线

### 立即可实施（确定性高，风险低）

1. ✅ **IFNULL→COALESCE** (MySQL/GaussDB-M/GaussDB-A) — 严格等价，实现简单
2. ✅ **= ANY → IN** (四款 DBMS) — 严格等价，bug 潜力高
3. ✅ **NOT ISNULL→IS NOT NULL** (MySQL/GaussDB-M) — 布尔上下文等价
4. ✅ **SELF JOIN 生成** (四款 DBMS) — 纯生成增强
5. ✅ **IS TRUE→=TRUE** (PostgreSQL/GaussDB-A) — PG 布尔语义等价

### 近期可实施（需 NULL 包装，中等风险）

6. ⚠️ **LEAST/GREATEST→CASE** (四款 DBMS) — 需 NULL 检查包装才能等价
7. ⚠️ **<=>→CASE** (MySQL/GaussDB-M) — 需 NULL 检查包装
8. ⚠️ **<> ALL→NOT IN** (四款 DBMS) — 严格等价但需扩展 ANY/ALL 体系

### 需架构改造后实施

9. 🔧 **INTERSECT→EXISTS** (PostgreSQL) — 需 PG stage2 引擎改造
10. 🔧 **EXCEPT→NOT EXISTS** (PostgreSQL) — 同上
11. 🔧 **GaussDB-A 改用 pg_query** — 需重写 A-mode 整套体系

### 远期目标

12. 📋 **JSON 表达式生成** — 需新增 JSON 生成体系
13. 📋 **JSON 等价变换** — parser 限制，远期目标
14. 📋 **PG POSIX 正则变异** — 非 EET 变换，是 implication 变异