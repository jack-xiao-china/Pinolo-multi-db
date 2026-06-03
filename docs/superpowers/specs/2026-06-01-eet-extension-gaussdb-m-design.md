# Pinolo EET 变换规则扩展与 GaussDB-M 深度支持设计

**日期**: 2026-06-01
**状态**: Draft
**参考**: `docs/EET_ORACLE_COMPARISON.md`

---

## Context

Pinolo 当前仅实现了 EET Oracle 的 5 种包装规则（tautology/contradiction/CASE WHEN），覆盖率约 45%。SQLancer 的 EET Oracle 另有 8 条语义重写规则（De Morgan、BETWEEN→Comparison、EXISTS↔IN、COALESCE→CASE、NULLIF→CASE、INTERSECT/EXCEPT 变换）未被覆盖。

同时，GaussDB-M（MySQL 兼容模式）虽有基本的 connector/task 支持，但直接复用 MySQL mutation 逻辑，无法发现 M 模式特有的行为差异漏洞（NULL 处理、类型转换、函数映射等）。

本设计解决三个问题：
1. 将 SQLancer 的 5 条核心语义重写规则移植为 Pinolo Stage2 的新 mutation 类型
2. 为 GaussDB-M 增加独立的 stage1 预处理和 stage2 mutation 体系
3. 复用 SQLancer EET 规则逻辑，以 Pinolo mutation 引擎模式实现

---

## 核心结论：对已有功能零影响

所有新增内容均为**追加式**——追加常量、追加 mining 函数调用、追加 switch case 分支。不删除、不重构、不覆盖已有逻辑。原因：Pinolo 的 Stage2 架构天然支持扩展，每个 mutation 类型是独立的 `(addCandidate → doMutation → ImpoMutate switch case)` 三段式实现。

| 已有组件 | 是否修改 | 说明 |
|---------|:---:|------|
| `allmutations.go` | 追加 | 新增 mutation 名称常量 |
| `mutatevisitor.go` | 追加 | miningSelectStmt 中追加新 mining 函数调用 |
| `stage2.go` (MySQL) | 追加 | ImpoMutate switch 追加新 case |
| `pg_mutatevisitor.go` | 追加 | miningWhereClause 追加新规则 |
| `pg_stage2.go` | 追加 | PgImpoMutate switch 追加新 case |
| `oracle.go` | 扩展 | 新增等价判断逻辑（不改已有 upper/lower 逻辑） |
| MySQL `stage1.go` | ❌ 不改 | M 模式有独立文件 |
| `connector/*.go` | ❌ 不改 | 已有完整支持 |
| `task/task.go` (MySQL) | ❌ 不改 | M 模式有独立 task 流程 |
| `task/postgresql_task.go` | ❌ 不改 | PG 流程独立 |

---

## 设计细节

### 1. EET 语义重写规则移植

#### 1.1 新增 5 条规则

| 规则 | MySQL 名称 | PG 名称 | Oracle 关系 | 变换描述 |
|------|-----------|---------|------------|---------|
| De Morgan AND | `FixMDeMorganAnd` | `FixMDeMorganAnd_Pg` | 等价 | `(A AND B) → NOT(NOT(A) OR NOT(B))` |
| De Morgan OR | `FixMDeMorganOr` | `FixMDeMorganOr_Pg` | 等价 | `(A OR B) → NOT(NOT(A) AND NOT(B))` |
| BETWEEN→Cmp | `FixMBetweenToCmp` | `FixMBetweenToCmp_Pg` | 等价 | `x BETWEEN a AND b → (x>=a) AND (x<=b)` |
| COALESCE→CASE | `FixMCoalesceToCase` | `FixMCoalesceToCase_Pg` | 等价 | `COALESCE(a,b) → CASE WHEN a IS NOT NULL THEN a ELSE b END` |
| NULLIF→CASE | `FixMNullifToCase` | `FixMNullifToCase_Pg` | 等价 | `NULLIF(a,b) → CASE WHEN a=b THEN NULL ELSE a END` |

**EXISTS↔IN** 规则因涉及子查询重写，实现复杂度高，单独在 Phase 1C 处理。

#### 1.2 等价类 mutation 的 Oracle 判断

这 5 条规则是语义等价变换——变换前后结果集应完全相同。不同于已有的 upper/lower mutation，需要一个 **equivalence** 判断模式。

在 `oracle.go` 中新增 `CheckEquivalence` 函数：
- 行数必须相同
- 每行内容必须完全匹配（逐行逐列比较）
- 如果不等价 → 漏洞

#### 1.3 实现模式（三段式）

每条规则遵循现有模式：

```
1. addCandidate: 在 miningSelectStmt 或 miningExprNode 中调用
2. doMutation: 实际变换 AST，生成新 SQL
3. ImpoMutate/PgImpoMutate switch: 追加 case 分支
```

#### 1.4 De Morgan 规则的 mining 位置

De Morgan 规则针对 `BinaryOperationExpr`（AND/OR），应添加在 `miningBinaryOperationExpr` 中。

#### 1.5 BETWEEN/COALESCE/NULLIF 规则的 mining 位置

- BETWEEN: 需要在 `visitExprNode` 中将 `*ast.BetweenExpr` 从 skip 改为 mining
- COALESCE/NULLIF: 需要在 `visitExprNode` 中将特定 `*ast.FuncCallExpr` 从 skip 改为 mining

这仅影响 EET 规则的候选发现，不影响已有 mutation 的候选发现逻辑。

#### 1.6 文件分布

```
mutation/stage2/
  eet_demorgan.go          # De Morgan (MySQL)
  eet_between.go           # BETWEEN→Cmp (MySQL)
  eet_functions.go         # COALESCE→CASE, NULLIF→CASE (MySQL)
  eet_subquery.go          # EXISTS↔IN (MySQL, Phase 1C)
  pg_eet_demorgan.go       # De Morgan (PG)
  pg_eet_between.go        # BETWEEN→Cmp (PG)
  pg_eet_functions.go      # COALESCE→CASE, NULLIF→CASE (PG)
  pg_eet_subquery.go       # EXISTS↔IN (PG, Phase 1C)
```

---

### 2. GaussDB-M 深度支持

#### 2.1 Parser 选择：TiDB parser

GaussDB-M 使用 TiDB parser（MySQL 语法解析器），理由：
- 输入 SQL 仍为 MySQL 语法风格（反引号、LIMIT、IF() 等）
- M 模式与 MySQL 的差异在于**行为/语义**而非**语法格式**
- 用户测试用例会以 MySQL 语法编写

#### 2.2 Stage1：M 模式专属预处理

文件：`mutation/stage1/stage1_gaussdb_m.go`

基于 `stage1.InitAndExec` 扩展，追加 M 模式特有处理：
1. 标准 MySQL Stage1 逻辑
2. M 模式特有移除（TOP n 子句、M 特有不确定函数、隐式类型转换风险）
3. 执行并返回结果

#### 2.3 Stage2：M 模式 MutateVisitor

文件：`mutation/stage2/gaussdb_m_mutatevisitor.go`

基于 `MutateVisitor`（TiDB parser 版本）扩展：
- 复用 MySQL 的基础 mutation
- + 5 条 EET 语义重写规则（复用 eet_demorgan.go 等的 doMutation 函数）
- + M 模式特有 EET 规则

#### 2.4 M 模式特有 EET 规则

| 规则 | 名称 | Oracle 关系 |
|------|------|------------|
| `TOP n → LIMIT n` | `FixMTopToLimit` | 等价 |
| `IF(cond,a,b) → CASE WHEN` | `FixMIfToCase` | 等价 |
| `CONCAT(a,b) → a || b` | `FixMConcatToPipe` | 等价（需验证 NULL） |

#### 2.5 Task 层修改

修改 `RunTaskGaussDB`：改用 M 专属的 stage1/stage2 流程。

#### 2.6 Schema 发现

新增 `connector/schema_gaussdb_m.go`。

#### 2.7 文件分布

```
mutation/stage1/
  stage1_gaussdb_m.go        # M 模式预处理

mutation/stage2/
  gaussdb_m_mutatevisitor.go  # M 模式 visitor
  gaussdb_m_stage2.go         # M 模式 stage2 入口
  gaussdb_m_eet_mutations.go  # M 模式特有 EET 规则

connector/
  schema_gaussdb_m.go         # M 模式 schema 发现

task/
  gaussdb_task.go             # 修改 RunTaskGaussDB
```

---

### 3. 复用 SQLancer EET 规则策略

#### 3.1 三套并行实现体系

| DBMS | Parser | Visitor | Stage2 入口 |
|------|--------|---------|------------|
| MySQL | TiDB parser | `MutateVisitor` | `stage2.go` |
| PostgreSQL | pg_query | `PgMutateVisitor` | `pg_stage2.go` |
| GaussDB-M | TiDB parser | `GaussDBMMutateVisitor` | `gaussdb_m_stage2.go` |

MySQL 和 GaussDB-M 共享 TiDB parser 下的 EET 实现（`eet_demorgan.go` 等）。

#### 3.2 复用方式：移植逻辑而非代码

变换逻辑从 Java AST manipulation 移植为 Go AST manipulation，NULL 安全性用 Go 中手动构建 IS NULL / IS NOT NULL 表达式实现，等价判断使用 Implication Oracle + equivalence 检查。

---

## 实施优先级

```
Phase 1A: MySQL EET 语义重写规则 (无依赖)
Phase 1B: PostgreSQL EET 语义重写规则 (依赖 1A)
Phase 1C: EXISTS↔IN (依赖 1A/B)
Phase 2:  GaussDB-M 深度支持 (依赖 1A)
Phase 3:  GaussDB-M 特有 EET 规则 (依赖 2)
```

---

## 验证方法

1. 单元测试：每条 EET 规则的 doMutation 函数
2. 等价验证：变换后 SQL 与原 SQL 在目标 DBMS 上结果一致
3. 回归测试：`go test ./mutation/stage2/...` 确保已有 mutation 不受影响
4. GaussDB-M 连接测试：完整 task 流程运行
5. 对比测试：同 SQL 分别走 MySQL 和 M 流程