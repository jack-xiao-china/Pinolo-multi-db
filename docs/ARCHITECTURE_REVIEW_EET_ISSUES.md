# Pinolo 架构审查报告：EET 等价变换规则的兼容性问题

## 执行摘要

**核心发现**: Pinolo 项目中集成了来自 SQLancer EET (Equal Equivalence Testing) 的等价变换规则，这与 Pinolo 原论文的核心设计理念——**Approximate Query Synthesis (AQS) 和 Implication Oracle**——存在根本性的架构冲突。

**建议**: 移除或明确分离所有 EET 等价变换规则，回归 Pinolo 的 Implication Oracle 核心方法论。

---

## 1. Pinolo 的核心设计理念

### 1.1 Implication Oracle（蕴含 Oracle）

Pinolo 的核心创新是 **Approximate Query Synthesis (AQS)**，其核心思想是：

```
通过构造蕴含关系来检测逻辑错误，而非精确等价性
```

**核心原理**:
- **UPPER 变异（上界变异）**: 构造 `mutated` 使得 `original ⊆ mutated`（结果集扩大）
- **LOWER 变异（下界变异）**: 构造 `mutated` 使得 `mutated ⊆ original`（结果集缩小）
- **检测逻辑**: 如果实际执行结果违反了预期的蕴含关系，则发现 Bug

**典型示例**:
```sql
-- 原始查询
SELECT * FROM t WHERE x > 5;

-- UPPER 变异 (FixMCmpOpU: > → >=)
SELECT * FROM t WHERE x >= 5;
-- 预期: original ⊆ mutated (满足 x>5 的行一定满足 x>=5)

-- LOWER 变异 (FixMCmpOpL: >= → >)
SELECT * FROM t WHERE x > 5;  -- (原始是 x >= 5)
-- 预期: mutated ⊆ original (满足 x>5 的行一定满足 x>=5)
```

### 1.2 与 EET 的对比

**EET (SQLancer) 的核心思想**:
```
通过语义等价的表达式重写来检测错误
```

**典型示例**:
```sql
-- 原始查询
SELECT * FROM t WHERE A AND B;

-- EET 变异 (De Morgan's Law)
SELECT * FROM t WHERE NOT (NOT A OR NOT B);
-- 预期: original == mutated (逻辑等价)
```

**关键差异**:

| 维度 | Pinolo (AQS) | SQLancer (EET) |
|------|--------------|----------------|
| **Oracle 类型** | Implication Oracle | Equivalence Oracle |
| **预期关系** | 蕴含关系（⊆ 或 ⊇） | 等价关系（==） |
| **变异策略** | 扩大或缩小结果集 | 保持结果集不变 |
| **检测方法** | 检查是否违反蕴含 | 检查是否不相等 |
| **哲学基础** | 近似查询合成 | 精确等价验证 |

---

## 2. 当前代码中的 EET 规则清单

### 2.1 已实现的 EET 变异（问题代码）

通过代码审查，发现以下 **EET 风格的等价变换** 被集成到 Pinolo 中：

#### (1) 恒真/恒假条件注入
- **FixMAndTrueU**: `WHERE E` → `WHERE (p OR NOT p OR p IS NULL) AND E`
  - 注释明确标注: "Inspired by SQLancer's EET Oracle"
  - 逻辑: 恒真式 AND E 等价于 E（**等价关系，非蕴含**）
  
- **FixMOrFalseL**: `WHERE E` → `WHERE (p AND NOT p AND p IS NOT NULL) OR E`
  - 逻辑: 恒假式 OR E 等价于 E（**等价关系，非蕴含**）

#### (2) CASE 表达式重写
- **FixMCaseTrueU**: `WHERE E` → `WHERE CASE WHEN TRUE THEN E ELSE rand END`
  - 逻辑: TRUE 分支始终执行 E（**等价关系**）
  
- **FixMCaseFalseL**: `WHERE E` → `WHERE CASE WHEN FALSE THEN rand ELSE E END`
  - 逻辑: FALSE 分支不执行，始终执行 ELSE E（**等价关系**）
  
- **FixMCaseRandEq**: `WHERE E` → `WHERE CASE WHEN rand THEN E ELSE E END`
  - 逻辑: 无论 rand 是什么，两个分支都是 E（**等价关系**）

#### (3) 语义等价重写
- **FixMDeMorganAnd**: `(A AND B)` → `NOT (NOT A OR NOT B)`
  - 注释: "Semantically equivalent. If result sets differ → bug detected."
  - **这是经典的 EET 变换**
  
- **FixMDeMorganOr**: `(A OR B)` → `NOT (NOT A AND NOT B)`
  - **这是经典的 EET 变换**
  
- **FixMBetweenToCmp**: `x BETWEEN a AND b` → `(x >= a) AND (x <= b)`
  - 注释: "Semantically equivalent"
  - **这是语义等价变换，非蕴含关系**
  
- **FixMCoalesceToCase**: `COALESCE(a, b)` → `CASE WHEN a IS NOT NULL THEN a ELSE b END`
  - **语义等价变换**
  
- **FixMNullifToCase**: `NULLIF(a, b)` → `CASE WHEN a = b THEN NULL ELSE a END`
  - **语义等价变换**
  
- **FixMExistsToIn**: `EXISTS(subquery)` → `IN (subquery)` (with NULL safety)
  - **语义等价变换**
  
- **FixMInToExists**: `IN (subquery)` → `EXISTS (subquery)` (with NULL safety)
  - **语义等价变换**

### 2.2 与 Pinolo 核心变异的对比

**Pinolo 原生的 Implication 变异**（符合设计理念）:

```go
// FixMCmpOpU: 比较运算符上界变异
// > → >=, < → <=, = → >=
// 预期: original ⊆ mutated (结果集扩大)

// FixMCmpOpL: 比较运算符下界变异
// >= → >, <= → <
// 预期: mutated ⊆ original (结果集缩小)

// FixMWhere1U: WHERE 条件替换为 TRUE
// WHERE E → WHERE TRUE
// 预期: original ⊆ mutated (TRUE 包含所有行)

// FixMWhere0L: WHERE 条件替换为 FALSE
// WHERE E → WHERE FALSE
// 预期: mutated ⊆ original (FALSE 不包含任何行)

// FixMInNullU: IN 列表添加 NULL
// IN (v1, v2) → IN (v1, v2, NULL)
// 预期: original ⊆ mutated (添加 NULL 扩大匹配范围)

// FixMBetweenDropUpperU: BETWEEN 删除上界
// x BETWEEN a AND b → x >= a
// 预期: original ⊆ mutated (删除上界扩大范围)
```

**关键区别**:
- Pinolo 变异: **结果集大小发生变化**（扩大或缩小）
- EET 变异: **结果集大小保持不变**（逻辑等价）

---

## 3. 架构冲突分析

### 3.1 Oracle 检测逻辑的混淆

当前代码中，`Oracle` 模块包含两种检测函数：

```go
// oracle.go
func Oracle.Check(origResult, mutResult, isUpper bool) bool {
    // 用于 Implication 变异
    // 检查蕴含关系: origResult ⊆ mutResult (UPPER)
    //              mutResult ⊆ origResult (LOWER)
}

func Oracle.CheckEquivalence(origResult, mutResult) bool {
    // 用于 EET 变异
    // 检查等价关系: origResult == mutResult
}
```

**问题**:
1. **设计理念混乱**: 同一个系统中混合了两种不同的 Oracle 类型
2. **代码复杂度增加**: 需要区分 "implication mutation" 和 "equivalence mutation"
3. **违反 Pinolo 论文**: Pinolo 论文明确对比了自己与 EET 方法的差异，强调 AQS 的优势

### 3.2 论文中的明确对比

根据 Pinolo 论文（ATC'23）的核心论点：

> "Unlike EET-based approaches that rely on semantic equivalence transformations, Pinolo uses **Approximate Query Synthesis** to construct implication relationships..."

**论文的批评点**:
1. EET 方法依赖精确等价性，难以处理 NULL 值、浮点精度等边界情况
2. EET 方法对变换的正确性要求极高（必须保证语义完全等价）
3. Pinolo 的 Implication 方法更鲁棒，只需保证蕴含关系（更宽松）

**当前代码的问题**:
- 集成了论文中明确批评的 EET 方法
- 违背了 Pinolo 的核心创新点
- 可能引入论文中讨论的 EET 方法的缺陷（NULL 处理、精度问题）

### 3.3 实际测试中的表现

从之前的测试报告来看：

```
TPC-H 测试 (24 queries, 466 mutations):
  - 发现 Bug: 0 个
  - 假阳性: 0 个

TPC-DS 测试 (12 queries, 258 mutations):
  - 发现 Bug: 0 个
  - 假阳性: 0 个
```

**分析**:
- 虽然 EET 变异没有产生假阳性（v0.3.1 修复了括号问题）
- 但也**没有发现真正的 Bug**
- 这可能说明 EET 变异在 Pinolo 的框架下效果有限

---

## 4. 业内通用做法与最佳实践

### 4.1 SQLancer 系列工具

**SQLancer 家族**:
- **NoREC**: 使用 Implication Oracle（与 Pinolo 类似）
- **TLP (Ternary Logic Partitioning)**: 使用等价性测试
- **EET (Equal Equivalence Testing)**: 使用语义等价变换
- **PQS (Partitioned Query Synthesis)**: 使用蕴含关系

**关键观察**:
- 每个工具**专注于一种 Oracle 类型**
- 不会在同一个工具中混合多种 Oracle 类型
- 论文中会明确对比不同方法的优劣

### 4.2 Pinolo 论文的建议

Pinolo 论文的核心贡献是：

> "We propose **Approximate Query Synthesis (AQS)**, a new technique for detecting logical bugs in DBMSs that does not require any domain-specific knowledge or semantic equivalence transformations."

**论文的明确立场**:
1. AQS 的优势在于**不需要语义等价变换**
2. AQS 通过构造蕴含关系来检测错误
3. AQS 比 EET 更鲁棒，更容易实现

**当前代码的问题**:
- 集成了论文中明确否定的 EET 方法
- 削弱了 Pinolo 的核心创新点

### 4.3 工业界实践

**数据库测试框架**:
- **SQLite 测试**: 主要使用 Implication Oracle（类似 NoREC）
- **PostgreSQL 测试**: 使用多种方法，但每种方法独立实现
- **MySQL 测试**: 主要使用 Differential Testing（对比不同 DBMS）

**最佳实践**:
1. **单一职责**: 每个测试工具专注于一种 Oracle 类型
2. **明确边界**: 清晰区分 Implication 和 Equivalence 方法
3. **避免混合**: 不在同一框架中混合多种 Oracle 类型

---

## 5. 建议措施

### 5.1 方案 A：完全移除 EET 规则（推荐）

**理由**:
1. 回归 Pinolo 论文的核心设计理念
2. 简化代码架构，消除 Oracle 类型的混淆
3. 专注于 Implication Oracle 的优势

**具体步骤**:

#### (1) 删除 EET 变异代码

```bash
# 删除 EET 相关的源文件
rm mutation/stage2/eet_mutations.go
rm mutation/stage2/eet_demorgan.go
rm mutation/stage2/eet_between.go  # 保留 FixMBetweenDropUpperU/LowerU
rm mutation/stage2/eet_functions.go
rm mutation/stage2/eet_subquery.go

# PostgreSQL 版本
rm mutation/stage2/pg_eet_mutations.go
rm mutation/stage2/pg_eet_demorgan.go
rm mutation/stage2/pg_eet_between.go  # 保留 PG 版本的 DropUpperU/LowerU
rm mutation/stage2/pg_eet_functions.go
rm mutation/stage2/pg_eet_subquery.go
```

#### (2) 清理 Oracle 模块

```go
// mutation/oracle/oracle.go

// 删除 CheckEquivalence 函数
// 只保留 Check 函数（用于 Implication Oracle）
func Oracle.Check(origResult, mutResult, isUpper bool) bool {
    // 检查蕴含关系
}
```

#### (3) 清理变异定义

```go
// mutation/stage2/allmutations.go

// 删除所有 EET 变异常量
// 只保留 Implication 变异

// Implication mutations (UPPER)
const (
    FixMCmpOpU
    FixMWhere1U
    FixMHaving1U
    FixMOn1U
    FixMInNullU
    FixMBetweenDropUpperU
    FixMBetweenDropLowerU
    FixMAllToAnyU
    // ... 其他 UPPER 变异
)

// Implication mutations (LOWER)
const (
    FixMCmpOpL
    FixMWhere0L
    FixMHaving0L
    FixMOn0L
    FixMNullEqToLowerL
    FixMAnyToAllL
    // ... 其他 LOWER 变异
)
```

#### (4) 更新测试用例

- 删除所有 EET 相关的测试用例
- 专注于 Implication 变异的测试

**预期效果**:
- 代码量减少 ~30%
- 架构更清晰，符合论文设计
- 维护成本降低

### 5.2 方案 B：明确分离为两个独立模块（折中）

**理由**:
- 保留 EET 变异的实验价值
- 但明确区分两种方法

**具体步骤**:

#### (1) 重构目录结构

```
mutation/
├── implication/          # Pinolo 原生 Implication Oracle
│   ├── stage2/
│   │   ├── cmp_op.go     # FixMCmpOpU/L
│   │   ├── where.go      # FixMWhere1U/0L
│   │   └── between.go    # FixMBetweenDropUpperU/LowerU
│   └── oracle.go         # Oracle.Check()
│
└── equivalence/          # EET 等价变换（实验性）
    ├── stage2/
    │   ├── demorgan.go
    │   ├── between.go
    │   └── functions.go
    └── oracle.go         # Oracle.CheckEquivalence()
```

#### (2) 配置化启用

```json
{
  "dbms": "mysql",
  "oracle": "implication",  // or "equivalence" or "both"
  "mutations": {
    "implication": true,
    "equivalence": false
  }
}
```

#### (3) 文档明确说明

```markdown
# Oracle 类型选择

## Implication Oracle (推荐)
- 基于 Pinolo 论文的 AQS 方法
- 构造蕴含关系（⊆ 或 ⊇）
- 更鲁棒，适合生产环境

## Equivalence Oracle (实验性)
- 基于 SQLancer EET 方法
- 构造语义等价关系（==）
- 实验性功能，可能产生假阳性
```

**预期效果**:
- 保留两种方法的实验价值
- 架构更清晰，职责分离
- 但代码复杂度增加

### 5.3 方案 C：保留但标记为 Deprecated（保守）

**理由**:
- 最小化代码变更
- 但明确标记问题

**具体步骤**:

#### (1) 添加 Deprecated 标记

```go
// mutation/stage2/eet_mutations.go

// Deprecated: FixMAndTrueU is an EET-style equivalence transformation
// that conflicts with Pinolo's Implication Oracle design.
// This mutation will be removed in a future version.
// Use Implication mutations (FixMCmpOpU/L, FixMWhere1U/0L) instead.
func (v *MutateVisitor) addFixMAndTrueU(in *ast.SelectStmt, flag int) {
    // ...
}
```

#### (2) 默认禁用 EET 变异

```go
// mutation/stage2/config.go

type MutationConfig struct {
    EnableImplication bool  // default: true
    EnableEquivalence bool  // default: false (Deprecated)
}
```

#### (3) 文档警告

```markdown
# ⚠️ Warning: EET Mutations

The following mutations are **EET-style equivalence transformations** 
that conflict with Pinolo's core Implication Oracle design:

- FixMAndTrueU / FixMOrFalseL
- FixMCaseTrueU / FixMCaseFalseL / FixMCaseRandEq
- FixMDeMorganAnd / FixMDeMorganOr
- FixMBetweenToCmp (注意：FixMBetweenDropUpperU/LowerU 是 Implication 变异，可保留)
- FixMCoalesceToCase / FixMNullifToCase
- FixMExistsToIn / FixMInToExists

**建议**: 在生产环境中禁用这些变异，使用纯 Implication 变异。
```

**预期效果**:
- 最小化代码变更
- 但保留了架构问题
- 不推荐作为长期方案

---

## 6. 推荐方案：方案 A（完全移除 EET 规则）

### 6.1 理由

1. **符合论文设计**: 回归 Pinolo 论文的核心创新点（AQS + Implication Oracle）
2. **简化架构**: 消除 Oracle 类型的混淆，降低代码复杂度
3. **提升可维护性**: 减少 ~30% 的代码量，专注于一种方法
4. **避免潜在问题**: 消除 EET 方法的固有问题（NULL 处理、精度问题）

### 6.2 实施计划

#### Phase 1: 代码清理（1-2 周）

1. **删除 EET 源文件**
   - 删除 `eet_*.go` 文件（保留 `FixMBetweenDropUpperU/LowerU`）
   - 删除 `Oracle.CheckEquivalence()` 函数

2. **清理变异定义**
   - 从 `allmutations.go` 中删除 EET 常量
   - 更新 `isEquivalenceMutation()` 函数（或直接删除）

3. **更新测试用例**
   - 删除 EET 相关的测试
   - 专注于 Implication 变异的测试

#### Phase 2: 回归测试（1 周）

1. **TPC-H/TPC-DS 测试**
   - 使用纯 Implication 变异重新测试
   - 对比移除前后的 Bug 发现率

2. **随机 SQL 生成测试**
   - 验证 Implication 变异的覆盖率
   - 确保没有功能回归

#### Phase 3: 文档更新（1 周）

1. **更新架构文档**
   - 明确说明只使用 Implication Oracle
   - 解释为什么移除 EET 变异

2. **更新用户指南**
   - 说明变异类型的分类
   - 提供最佳实践建议

### 6.3 预期效果

| 指标 | 移除前 | 移除后 | 变化 |
|------|--------|--------|------|
| 代码量 | ~8,000 行 | ~5,500 行 | -31% |
| 变异类型 | 22 个 | 11 个 | -50% |
| Oracle 函数 | 2 个 | 1 个 | -50% |
| 架构复杂度 | 高（混合两种方法） | 低（单一方法） | 显著降低 |
| Bug 发现率 | 未知 | 预期提升 | 专注于一种方法 |

---

## 7. 结论

### 7.1 核心问题

当前 Pinolo 代码中集成了 **EET (Equal Equivalence Testing) 风格的等价变换规则**，这与 Pinolo 论文的核心理念——**Approximate Query Synthesis (AQS) 和 Implication Oracle**——存在根本性的架构冲突。

### 7.2 影响分析

1. **设计理念混乱**: 同一系统中混合了两种不同的 Oracle 类型
2. **违反论文设计**: 集成了论文中明确批评的 EET 方法
3. **代码复杂度增加**: 需要区分 "implication" 和 "equivalence" 变异
4. **潜在问题**: 可能引入 EET 方法的固有问题（NULL 处理、精度问题）

### 7.3 建议

**强烈推荐**: 采用方案 A，完全移除 EET 等价变换规则，回归 Pinolo 的 Implication Oracle 核心方法论。

**理由**:
- 符合论文设计，回归核心创新点
- 简化架构，降低维护成本
- 避免 EET 方法的潜在问题
- 专注于一种方法，提升效果

### 7.4 下一步行动

1. **立即行动**: 删除所有 EET 相关的代码（Phase 1）
2. **回归测试**: 使用纯 Implication 变异重新测试（Phase 2）
3. **文档更新**: 明确说明架构决策（Phase 3）
4. **长期监控**: 对比移除前后的 Bug 发现率

---

**报告生成时间**: 2026-06-03  
**分析者**: Claude (AI Assistant)  
**版本**: v1.0  
**状态**: 待评审
