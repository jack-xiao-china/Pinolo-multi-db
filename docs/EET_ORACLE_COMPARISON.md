# Pinolo vs SQLancer EET Oracle 语法覆盖对比分析

## 概述

本文档对比分析 SQLancer 的 EET Oracle 与 Pinolo 当前实现的语法覆盖范围，找出遗漏的语法支持和变异规则，为后续开发提供优先级指导。

**分析日期**: 2026-06-01

**总体覆盖率估算**: **35-40%**

---

## 一、EET 变换规则对比

| 变换规则 | SQLancer MySQL | SQLancer PostgreSQL | Pinolo MySQL | Pinolo PostgreSQL | 状态 |
|---------|:--------------:|:-------------------:|:------------:|:-----------------:|:----:|
| **Tautology Wrapping** `(p OR NOT p OR p IS NULL) AND E` | ✅ (Rule 1) | ✅ | ✅ `FixMAndTrueU` | ✅ `FixMAndTrueU_Pg` | ✅ 已实现 |
| **Contradiction Wrapping** `(p AND NOT p AND p IS NOT NULL) OR E` | ✅ (Rule 2) | ✅ | ✅ `FixMOrFalseL` | ✅ `FixMOrFalseL_Pg` | ✅ 已实现 |
| **CASE WHEN TRUE** `CASE WHEN TRUE THEN E ELSE rand END` | ✅ (Rule 4) | ✅ | ✅ `FixMCaseTrueU` | ✅ `FixMCaseTrueU_Pg` | ✅ 已实现 |
| **CASE WHEN FALSE** `CASE WHEN FALSE THEN rand ELSE E END` | ✅ (Rule 3) | ✅ | ✅ `FixMCaseFalseL` | ✅ `FixMCaseFalseL_Pg` | ✅ 已实现 |
| **CASE WHEN rand** `CASE WHEN rand THEN E ELSE E END` | ✅ (Rule 5/6) | ✅ | ✅ `FixMCaseRandEq` | ✅ `FixMCaseRandEq_Pg` | ✅ 已实现 |
| **De Morgan's Law** `(A AND B) → NOT(NOT(A) OR NOT(B))` | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **BETWEEN → Comparison** `x BETWEEN a AND b → (x >= a) AND (x <= b)` | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **EXISTS → IN** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **IN → EXISTS** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **COALESCE → CASE** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **NULLIF → CASE** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **INTERSECT → EXISTS** | ❌ (MySQL不支持语法) | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **EXCEPT → NOT EXISTS** | ❌ (MySQL不支持语法) | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |

**结论**：Pinolo 仅实现了 EET 的 5 种**包装规则**，遗漏了 SQLancer 的 8 条**语义重写规则**。

**覆盖率**: **45%** (5/11 规则)

---

## 二、查询形状对比

| 查询形状 | SQLancer MySQL | SQLancer PostgreSQL | Pinolo Generator | Pinolo Stage1 | 状态 |
|---------|:--------------:|:-------------------:|:---------------:|:-------------:|:----:|
| **Plain SELECT** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **UNION / UNION ALL** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **INTERSECT / INTERSECT ALL** | ❌ (不支持语法) | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **EXCEPT / EXCEPT ALL** | ❌ (不支持语法) | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **WITH (CTE)** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **Derived Table (Subquery in FROM)** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **LATERAL Subquery** | ❌ | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |

**覆盖率**: MySQL **100%** (4/4), PostgreSQL **67%** (4/6)

---

## 三、DML 语句支持对比

| DML 类型 | SQLancer MySQL | SQLancer PostgreSQL | Pinolo | 状态 |
|---------|:--------------:|:-------------------:|:------:|:----:|
| **SELECT (主 EET Oracle)** | ✅ | ✅ | ✅ | ✅ 已实现 |
| **INSERT-SELECT** | ✅ `MySQLEETInsertSelectOracle` | ✅ | ❌ | ⚠️ **遗漏** |
| **UPDATE with WHERE** | ✅ `MySQLEETUpdateOracle` | ✅ | ❌ | ⚠️ **遗漏** |
| **DELETE with WHERE** | ✅ `MySQLEETDeleteOracle` | ✅ | ❌ | ⚠️ **遗漏** |

**结论**：Pinolo 仅支持 SELECT 测试，遗漏了 INSERT-SELECT、UPDATE、DELETE 的 EET 测试能力。

**覆盖率**: **25%** (1/4)

---

## 四、聚合与窗口函数对比

| 功能 | SQLancer MySQL | SQLancer PostgreSQL | Pinolo Stage1 | Pinolo Generator | 状态 |
|------|:--------------:|:-------------------:|:-------------:|:---------------:|:----:|
| **聚合函数 (COUNT, SUM, AVG, MIN, MAX)** | ✅ 15种 | ✅ | ❌ **移除** | ❌ | ⚠️ **设计差异** |
| **窗口函数 (ROW_NUMBER, RANK, LAG, LEAD)** | ✅ 7种 | ✅ 11种 | ❌ **移除** | ❌ | ⚠️ **设计差异** |
| **GROUP BY / HAVING** | ✅ | ✅ | ❌ **移除** | ✅ (可选) | ⚠️ **设计差异** |

**说明**：这是 Pinolo 的**设计选择**而非遗漏。Stage1 移除聚合/窗口函数是为了简化 Implication Oracle 的结果集比较逻辑。SQLancer 的 EET Oracle 通过 PQS (Pivoted Query Synthesis) 技术处理这些复杂情况。

---

## 五、表达式类型对比

| 表达式类型 | SQLancer MySQL | SQLancer PostgreSQL | Pinolo Generator | Pinolo Stage2 Mutation | 状态 |
|-----------|:--------------:|:-------------------:|:---------------:|:----------------------:|:----:|
| **列引用** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **常量** | ✅ (9种类型) | ✅ (16种类型) | ✅ (基础类型) | ✅ | ⚠️ **部分支持** |
| **一元前缀 (NOT, +, -)** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **一元后缀 (IS NULL, IS TRUE)** | ✅ 6种 | ✅ | ✅ 2种 (IS NULL/IS NOT NULL) | ✅ | ⚠️ **部分支持** |
| **二元逻辑 (AND, OR)** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **二元比较 (6种操作符)** | ✅ + `<=>` | ✅ | ✅ 6种 | ✅ `FixMCmpOpU/L` | ✅ 已实现 |
| **算术运算 (+, -, *, /, %)** | ✅ | ✅ | ✅ | ❌ | ⚠️ **无变异** |
| **位运算 (&, \|, ^, <<, >>)** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **IN 列表/子查询** | ✅ | ✅ | ✅ 列表 | ✅ `FixMInNullU` | ⚠️ **子查询IN无变异** |
| **BETWEEN** | ✅ | ✅ 对称/非对称 | ✅ | ❌ | ⚠️ **无变异** |
| **EXISTS 子查询** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **CASE WHEN** | ✅ | ✅ | ✅ | ✅ (EET) | ✅ 已实现 |
| **CAST** | ✅ | ✅ `::type` | ✅ | ❌ | ⚠️ **无变异** |
| **COALESCE** | ✅ | ✅ | ✅ | ❌ | ⚠️ **无变异** |
| **NULLIF** | ✅ | ✅ | ✅ | ❌ | ⚠️ **无变异** |
| **LIKE 模式匹配** | ✅ | ✅ | ✅ | ✅ `RdMLikeU/L` | ✅ 已实现 |
| **REGEXP 正则** | ✅ | ✅ POSIX正则 | ✅ | ✅ `RdMRegExpU/L` | ✅ 已实现 |
| **字符串连接 (\|\|)** | ✅ | ✅ | ✅ CONCAT | ❌ | ⚠️ **无变异** |
| **JSON 操作** | ✅ 8种函数 | ✅ 操作符+函数 | ❌ | ❌ | ⚠️ **遗漏** |
| **Range 操作 (@>, <@, &&)** | - | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **时间函数** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **时间间隔运算 (+ INTERVAL)** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |

**覆盖率**: **~40%**

---

## 六、JOIN 类型对比

| JOIN 类型 | SQLancer MySQL | SQLancer PostgreSQL | Pinolo Generator | Pinolo Stage1 | 状态 |
|----------|:--------------:|:-------------------:|:---------------:|:-------------:|:----:|
| **INNER JOIN** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **LEFT [OUTER] JOIN** | ✅ | ✅ | ✅ | ❌ **转INNER** | ⚠️ **设计差异** |
| **RIGHT [OUTER] JOIN** | ✅ | ✅ | ✅ | ❌ **转INNER** | ⚠️ **设计差异** |
| **CROSS JOIN** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **FULL [OUTER] JOIN** | ❌ | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **NATURAL JOIN** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **STRAIGHT_JOIN** | ✅ MySQL特有 | - | ❌ | ❌ | ⚠️ **遗漏** |

**覆盖率**: MySQL **50%** (3/6), PostgreSQL **60%** (3/5)

---

## 七、数据类型对比

| 数据类型 | SQLancer MySQL | SQLancer PostgreSQL | Pinolo Generator | Pinolo Mutation | 状态 |
|---------|:--------------:|:-------------------:|:---------------:|:---------------:|:----:|
| **INT 系列** | ✅ INT, BIGINT, SMALLINT, TINYINT | ✅ INT2, INT4, INT8 | ✅ | ✅ | ✅ 已实现 |
| **FLOAT/DECIMAL** | ✅ FLOAT, DOUBLE, DECIMAL | ✅ FLOAT4, FLOAT8, NUMERIC | ✅ | ✅ | ✅ 已实现 |
| **VARCHAR/CHAR/TEXT** | ✅ | ✅ TEXT, BPCHAR, VARCHAR | ✅ | ✅ | ✅ 已实现 |
| **BOOLEAN** | ✅ | ✅ | ✅ | ✅ | ✅ 已实现 |
| **DATE/TIME/TIMESTAMP** | ✅ 5种 | ✅ 6种 + TIMESTAMPTZ | ✅ | ❌ | ⚠️ **无变异** |
| **INTERVAL** | ❌ | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **JSON/JSONB** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **BIT** | ✅ 1-64位 | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **ENUM** | ✅ | ✅ | ❌ | ❌ | ⚠️ **遗漏** |
| **SET** | ✅ MySQL特有 | - | ❌ | ❌ | ⚠️ **遗漏** |
| **ARRAY** | ❌ | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **UUID** | ❌ | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **RANGE** | ❌ | ✅ INT4RANGE等 | ❌ | ❌ | ⚠️ **遗漏 (PG)** |
| **INET/CIDR** | ❌ | ✅ | ❌ | ❌ | ⚠️ **遗漏 (PG)** |

**覆盖率**: **~35%**

---

## 八、遗漏清单汇总

### 🔴 高优先级遗漏

| 类别 | 遗漏项 | 影响 | 实施建议 |
|------|-------|------|----------|
| **EET 变换规则** | De Morgan's Law | 无法测试逻辑运算的等价变换 | Phase 1 |
| **EET 变换规则** | BETWEEN→Comparison | 无法测试 BETWEEN 条件的逻辑漏洞 | Phase 1 |
| **EET 变换规则** | EXISTS↔IN | 无法测试子查询相关逻辑漏洞 | Phase 1 |
| **EET 变换规则** | COALESCE→CASE | 无法测试 NULL 处理函数的逻辑漏洞 | Phase 1 |
| **EET 变换规则** | NULLIF→CASE | 无法测试 NULLIF 函数的逻辑漏洞 | Phase 1 |
| **DML 支持** | INSERT-SELECT Oracle | 无法测试写入语句的逻辑漏洞 | Phase 4 |
| **DML 支持** | UPDATE Oracle | 无法测试 UPDATE WHERE 条件的逻辑漏洞 | Phase 4 |
| **DML 支持** | DELETE Oracle | 无法测试 DELETE WHERE 条件的逻辑漏洞 | Phase 4 |
| **PostgreSQL 特有语法** | INTERSECT 查询形状 | PG 测试覆盖不全 | Phase 2 |
| **PostgreSQL 特有语法** | EXCEPT 查询形状 | PG 测试覆盖不全 | Phase 2 |
| **PostgreSQL 特有语法** | INTERSECT→EXISTS 变换 | 无法测试 INTERSECT 逻辑漏洞 | Phase 2 |
| **PostgreSQL 特有语法** | EXCEPT→NOT EXISTS 变换 | 无法测试 EXCEPT 逻辑漏洞 | Phase 2 |
| **PostgreSQL 特有语法** | LATERAL 子查询 | 无法测试 LATERAL 相关逻辑漏洞 | Phase 2 |
| **PostgreSQL 特有语法** | FULL JOIN | 无法测试 FULL JOIN 相关逻辑漏洞 | Phase 2 |
| **表达式** | EXISTS 子查询生成和变异 | 无法测试 EXISTS 相关逻辑漏洞 | Phase 1 |

### 🟡 中优先级遗漏

| 类别 | 遗漏项 | 影响 | 实施建议 |
|------|-------|------|----------|
| **表达式变异** | BETWEEN 变异 | 表达式层测试覆盖不足 | Phase 3 |
| **表达式变异** | 算术运算变异 | 无法测试算术表达式的逻辑漏洞 | Phase 3 |
| **表达式变异** | 位运算变异 (MySQL) | 无法测试位运算的逻辑漏洞 | Phase 3 |
| **表达式变异** | CAST 变异 | 无法测试类型转换的逻辑漏洞 | Phase 3 |
| **JOIN 变异** | NATURAL JOIN 生成和变异 | JOIN 测试覆盖不全 | Phase 3 |
| **JOIN 变异** | STRAIGHT_JOIN 生成和变异 (MySQL) | MySQL 特有 JOIN 测试缺失 | Phase 3 |
| **数据类型** | JSON 支持 | 无法测试 JSON 类型的逻辑漏洞 | Phase 3 |
| **数据类型** | BIT 支持 | 无法测试 BIT 类型的逻辑漏洞 | Phase 3 |
| **数据类型** | ENUM 支持 | 无法测试 ENUM 类型的逻辑漏洞 | Phase 3 |
| **数据类型** | ARRAY 支持 (PG) | 无法测试 ARRAY 类型的逻辑漏洞 | Phase 3 |
| **数据类型** | UUID 支持 (PG) | 无法测试 UUID 类型的逻辑漏洞 | Phase 3 |
| **数据类型** | RANGE 支持 (PG) | 无法测试 RANGE 类型的逻辑漏洞 | Phase 3 |
| **子查询变异** | IN 子查询变异 | 子查询测试能力不足 | Phase 3 |

### 🟢 低优先级/设计差异

| 类别 | 项目 | 说明 |
|------|------|------|
| **聚合函数** | Stage1 移除 | 设计选择，需 PQS 技术才能支持（SQLancer 使用 PQS） |
| **窗口函数** | Stage1 移除 | 设计选择，需 PQS 技术才能支持 |
| **LEFT/RIGHT JOIN** | Stage1 转 INNER | 设计选择，简化 Oracle 逻辑 |
| **时间函数** | 暂未支持 | 可后续添加，需处理不确定性 |
| **不确定函数** | RAND, NOW, UUID 等 | Stage1 应移除，避免非确定性结果 |

---

## 九、建议实施优先级

### Phase 1: 补全核心 EET 变换规则 (预计工作量: 中等)

#### 1.1 De Morgan's Law 变换

**MySQL 实现**:
```
mutation/stage2/eet_demorgan.go

// FixMDeMorganAnd: (A AND B) → NOT(NOT(A) OR NOT(B))
// FixMDeMorganOr: (A OR B) → NOT(NOT(A) AND NOT(B))
```

**PostgreSQL 实现**:
```
mutation/stage2/pg_eet_demorgan.go

// FixMDeMorganAnd_Pg
// FixMDeMorganOr_Pg
```

#### 1.2 BETWEEN → Comparison 变换

**MySQL 实现**:
```
mutation/stage2/eet_between.go

// FixMBetweenToCmp: x BETWEEN a AND b → (x >= a) AND (x <= b)
```

**PostgreSQL 实现**:
```
mutation/stage2/pg_eet_between.go

// FixMBetweenToCmp_Pg: 支持对称和非对称 BETWEEN
```

#### 1.3 EXISTS ↔ IN 变换

**MySQL 实现**:
```
mutation/stage2/eet_subquery.go

// FixMExistsToIn: EXISTS(subquery) → TRUE IN (CASE WHEN subquery_pred IS NULL THEN FALSE ELSE subquery_pred END)
// FixMInToExists: lhs IN (subquery) → CASE WHEN (lhs IN subquery) IS NOT NULL THEN EXISTS(...) ELSE NULL END
```

**PostgreSQL 实现**:
```
mutation/stage2/pg_eet_subquery.go

// FixMExistsToIn_Pg
// FixMInToExists_Pg
```

#### 1.4 COALESCE → CASE 变换

```
mutation/stage2/eet_functions.go

// FixMCoalesceToCase: COALESCE(a, b) → CASE WHEN a IS NOT NULL THEN a ELSE b END
// 支持多参数 COALESCE (递归处理)
```

#### 1.5 NULLIF → CASE 变换

```
mutation/stage2/eet_functions.go

// FixMNullifToCase: NULLIF(a, b) → CASE WHEN a = b THEN NULL ELSE a END
```

---

### Phase 2: 补全 PostgreSQL 特有语法 (预计工作量: 较大)

#### 2.1 INTERSECT / EXCEPT 查询形状

**生成器扩展**:
```go
// generator/query_gen.go

func (g *QueryGenerator) generateIntersectSelect() string
func (g *QueryGenerator) generateExceptSelect() string
```

**候选检测扩展**:
```go
// mutation/stage2/pg_mutatevisitor.go

func (v *PgMutateVisitor) miningIntersectSelectStmt(sel *pgquery.SelectStmt, flag int)
func (v *PgMutateVisitor) miningExceptSelectStmt(sel *pgquery.SelectStmt, flag int)
```

#### 2.2 INTERSECT → EXISTS 变换

```
mutation/stage2/pg_eet_intersect.go

// FixMIntersectToExists_Pg: Q1 INTERSECT Q2 → Q1 WHERE EXISTS(Q2 WHERE col_equality AND Q2_pred)
// NULL 安全列等值比较: CASE WHEN t1.col IS NOT NULL AND t2.col IS NOT NULL AND t1.col = t2.col THEN TRUE ELSE FALSE END
```

#### 2.3 EXCEPT → NOT EXISTS 变换

```
mutation/stage2/pg_eet_except.go

// FixMExceptToNotExists_Pg: Q1 EXCEPT Q2 → Q1 WHERE NOT EXISTS(Q2 WHERE col_equality AND Q2_pred)
```

#### 2.4 FULL JOIN 支持

**生成器**:
```go
// generator/from_gen.go: 增加 FULL JOIN 生成概率
```

**Stage1 调整**:
```go
// mutation/stage1/stage1_pg.go: 保留 FULL JOIN（不转 INNER）
```

#### 2.5 LATERAL 子查询

```go
// generator/query_gen.go

func (g *QueryGenerator) generateLateralSubquery() string
```

---

### Phase 3: 补全表达式变异 (预计工作量: 中等)

#### 3.1 BETWEEN 变异

```
// mutation/stage2/between_mutations.go

// FixMBetweenU: 扩展 BETWEEN 范围
// FixMBetweenL: 收缩 BETWEEN 范围
```

#### 3.2 算术运算变异

```
// mutation/stage2/arith_mutations.go

// RdMArithOpU: a + b → a + b + rand_int(0, 10)
// RdMArithOpL: a + b → a + b - rand_int(0, 10)
```

#### 3.3 位运算变异 (MySQL)

```
// mutation/stage2/bit_mutations.go

// RdMBitOpU: a & b → a & b | rand_bit
// RdMBitOpL: a & b → a & b & rand_bit
```

#### 3.4 CAST 变异

```
// mutation/stage2/cast_mutations.go

// RdMCastType: CAST(x AS SIGNED) → CAST(x AS UNSIGNED)
// RdMCastType_Pg: CAST(x AS integer) → CAST(x AS bigint)
```

#### 3.5 JOIN 变异扩展

```
// generator/from_gen.go: 增加 NATURAL JOIN、STRAIGHT_JOIN 生成
// mutation/stage2/join_mutations.go: 对应变异函数
```

#### 3.6 数据类型扩展

```
// generator/expr_gen.go: 增加 JSON、BIT、ENUM、ARRAY 类型支持
// connector/schema*.go: DiscoverSchema 时识别这些类型
```

---

### Phase 4: DML Oracle 支持 (预计工作量: 较大)

#### 4.1 INSERT-SELECT Oracle

```go
// task/insert_select_task.go

func RunTaskInsertSelect(config *TaskConfig, ...) (*TaskResult, error)

// 对 INSERT 子查询的 WHERE 条件进行 EET 变换
// 比较变换前后插入的行数差异
```

#### 4.2 UPDATE Oracle

```go
// task/update_task.go

func RunTaskUpdate(config *TaskConfig, ...) (*TaskResult, error)

// 对 UPDATE WHERE 条件进行 EET 变换
// 比较变换前后更新的行数差异
```

#### 4.3 DELETE Oracle

```go
// task/delete_task.go

func RunTaskDelete(config *TaskConfig, ...) (*TaskResult, error)

// 对 DELETE WHERE 条件进行 EET 变换
// 比较变换前后删除的行数差异
```

---

## 十、总体评估

| 维度 | SQLancer EET Oracle | Pinolo 当前 | 覆盖率 |
|------|--------------------|-------------|:------:|
| **EET 变换规则** | 11 条 | 5 条 | **45%** |
| **查询形状** | MySQL 4种 / PG 6种 | 4 种 | MySQL **100%** / PG **67%** |
| **DML 支持** | SELECT + INSERT-SELECT + UPDATE + DELETE | SELECT | **25%** |
| **表达式类型** | MySQL 15类 / PG 48+类 | 基础类型 | **~30%** |
| **表达式变异** | 全表达式覆盖 | WHERE/HAVING/ON/DISTINCT/UNION/比较/LIKE/REGEXP | **~40%** |
| **数据类型** | MySQL 12种 / PG 16种 | 基础数值/字符串/时间 | **~35%** |
| **JOIN 类型** | MySQL 6种 / PG 5种 | 3 种 | MySQL **50%** / PG **60%** |

### 覆盖率矩阵

```
EET 规则覆盖:     ████████░░░░░░░░░░░░ 45%
查询形状覆盖:     ████████████████░░░░ MySQL 100%, PG 67%
DML 支持:         █████░░░░░░░░░░░░░░░ 25%
表达式类型:       ██████░░░░░░░░░░░░░░ 30%
表达式变异:       ████████░░░░░░░░░░░░ 40%
数据类型:         ███████░░░░░░░░░░░░░ 35%
JOIN 类型:        ██████████░░░░░░░░░ 55%
```

---

## 十一、结论与建议

### 主要差距

1. **EET 变换规则**：仅实现 5 种包装规则，遗漏 8 条语义重写规则
2. **DML 支持**：仅 SELECT，遗漏 INSERT-SELECT、UPDATE、DELETE
3. **PostgreSQL 特有语法**：遗漏 INTERSECT、EXCEPT、LATERAL、FULL JOIN、Range 操作
4. **表达式变异**：覆盖率约 40%，遗漏 EXISTS 子查询、BETWEEN、算术/位运算变异等

### 建议实施路径

```
Phase 1 (核心): EET 语义重写规则 → 2-3 周
Phase 2 (PG特有): PostgreSQL INTERSECT/EXCEPT/LATERAL → 1-2 周  
Phase 3 (表达式): 补全表达式变异 → 1-2 周
Phase 4 (DML): INSERT-SELECT/UPDATE/DELETE Oracle → 1-2 周
```

**总预计工作量**: 5-9 周

### 参考

- SQLancer EET Oracle 原论文: "Detecting Logical Bugs in SQL Optimizers"
- SQLancer 项目地址: `D:\Jack.Xiao\dbtools\sqlancer-main\sqlancer-main`
- Pinolo 项目地址: `D:\Jack.Xiao\dbtools\Pinolo-main\Pinolo-main`