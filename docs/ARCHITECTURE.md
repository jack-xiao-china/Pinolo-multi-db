# Pinolo 架构文档

**版本**: v0.4.0  
**最后更新**: 2026-06-03  
**状态**: 生产就绪

---

## 1. 概述

Pinolo 是一个基于 **Approximate Query Synthesis (AQS)** 和 **Implication Oracle** 的数据库逻辑错误检测工具。本项目是原始 Pinolo 论文（ATC'23）的开源实现，支持 MySQL、PostgreSQL、GaussDB-M 和 GaussDB-A 四种数据库。

### 1.1 核心设计理念

Pinolo 的核心创新是 **近似查询合成**（Approximate Query Synthesis）：

```
给定原始查询 Q，合成一个新查询 Q'，使得：
- Q ⊆ Q' (UPPER approximation，结果集扩大)
- Q' ⊆ Q (LOWER approximation，结果集缩小)
```

**关键洞察**：
- 使用**近似**（添加/移除约束）而非**等价变换**
- 通过结构化的查询合成保证蕴含关系
- 不依赖语义等价的复杂证明

### 1.2 Implication Oracle

```
执行 Q 和 Q'，检查结果集是否满足预期的蕴含关系：
- UPPER 变异：检查 result(Q) ⊆ result(Q')
- LOWER 变异：检查 result(Q') ⊆ result(Q)
- 违反蕴含关系 → 发现逻辑错误
```

### 1.3 与 SQLancer 方法的对比

| 维度 | Pinolo (AQS) | SQLancer (TLP/PQS) |
|------|-------------|-------------------|
| **核心思想** | 近似查询合成 | 三元逻辑划分 / 分区查询合成 |
| **变异策略** | 添加/移除约束 | 基于逻辑等价性划分 |
| **Oracle 类型** | Implication Oracle | 多种 Oracle |
| **健全性保证** | 结构化合成保证蕴含 | 依赖逻辑等价性证明 |
| **实现复杂度** | 中等 | 较高 |

---

## 2. 系统架构

### 2.1 整体流程图

```
原始 SQL
    ↓
Stage1: 查询简化
    - 移除聚合函数和 GROUP BY
    - 移除窗口函数
    - LEFT/RIGHT JOIN → INNER JOIN
    - 移除 LIMIT
    - 移除不确定函数（如 RAND()）
    ↓
简化后的 SQL
    ↓
Stage2: 变异生成
    - 遍历 AST，识别变异点
    - 应用 Implication 变异规则
    - 生成 UPPER/LOWER 变异查询
    ↓
变异查询集合
    ↓
执行引擎
    - 执行原始查询和变异查询
    - 收集结果集
    ↓
Implication Oracle
    - 检查结果集蕴含关系
    - 识别逻辑错误
    ↓
错误报告
```

### 2.2 核心组件

#### 2.2.1 Stage1: 查询简化 (`mutation/stage1/`)

**职责**：简化查询，使其更适合变异测试

**转换规则**：
- 移除聚合函数（SUM, COUNT, AVG 等）和 GROUP BY
- 移除窗口函数（ROW_NUMBER, RANK 等）
- LEFT/RIGHT JOIN 转换为 INNER JOIN
- 移除 LIMIT 子句
- 移除不确定函数（RAND(), UUID() 等）

**设计意图**：
- 简化查询结构，减少变异复杂度
- 避免聚合和窗口函数引入的非确定性
- 使结果集比较更可靠

**已知限制**：
- 转换过于激进，可能丢失原始查询语义
- 无法发现与被移除特性相关的逻辑错误
- 详见 [§5.1 Stage1 转换问题](#51-stage1-转换问题)

#### 2.2.2 Stage2: 变异生成 (`mutation/stage2/`)

**职责**：基于规则生成 Implication 变异查询

**核心机制**：
- 遍历 AST，识别变异点
- 应用预定义的变异规则
- 跟踪 Flag（正/负上下文）
- 生成 UPPER/LOWER 变异查询

**变异规则**：详见 [§3 变异策略](#3-变异策略)

**Flag 传播机制**：详见 [§4 Flag 传播机制](#4-flag-传播机制)

#### 2.2.3 Implication Oracle (`mutation/oracle/`)

**职责**：检查结果集是否满足预期的蕴含关系

**核心逻辑**：
```go
func Check(originResult, mutatedResult, isUpper) {
    cmp := CMP(originResult, mutatedResult)
    
    // cmp = -1: original ⊂ mutated
    // cmp = 0:  original = mutated
    // cmp = 1:  original ⊃ mutated
    
    if isUpper {
        // 期望 original ⊆ mutated (cmp = -1 or 0)
        return cmp == -1 || cmp == 0
    } else {
        // 期望 mutated ⊆ original (cmp = 1 or 0)
        return cmp == 1 || cmp == 0
    }
}
```

**已知限制**：
- 简单的结果集比较，缺乏多次执行取交集
- 浮点数精度、NULL 处理可能不够健壮
- 详见 [§5.3 Oracle 比较逻辑问题](#53-oracle-比较逻辑问题)

#### 2.2.4 执行引擎 (`task/`)

**职责**：协调查询执行和错误检测

**核心流程**：
1. 加载测试配置
2. 连接数据库
3. 执行 Stage1 简化
4. 执行 Stage2 变异生成
5. 执行所有查询
6. 调用 Oracle 检查结果
7. 生成错误报告

**测试模式**：
- **基于给定查询**：使用 `dmlPath` 指定的查询文件
- **随机生成查询**：使用 `generator` 模块随机生成查询

详见 [USER_GUIDE.md](USER_GUIDE.md) 的测试模式说明。

---

## 3. 变异策略

### 3.1 当前变异清单（v0.4.0）

EET 等价变换已在 v0.4.0 中移除。当前保留的所有变异都是 **Implication 变异**（近似变换）。

#### 3.1.1 MySQL 变异（11 个）

| 变异名称 | 类型 | 变换规则 | 蕴含关系 | 健全性 |
|---------|------|---------|---------|--------|
| FixMCmpOpU | UPPER | `> → >=`, `< → <=`, `= → >=` | original ⊆ mutated | ✅ |
| FixMCmpOpL | LOWER | `>= → >`, `<= → <` | mutated ⊆ original | ✅ |
| FixMWhere1U | UPPER | `WHERE E → WHERE TRUE` | original ⊆ mutated | ✅ |
| FixMWhere0L | LOWER | `WHERE E → WHERE FALSE` | mutated ⊆ original | ✅ |
| FixMHaving1U | UPPER | `HAVING E → HAVING TRUE` | original ⊆ mutated | ✅ |
| FixMHaving0L | LOWER | `HAVING E → HAVING FALSE` | mutated ⊆ original | ✅ |
| FixMOn1U | UPPER | `ON E → ON TRUE` | original ⊆ mutated | ✅ |
| FixMOn0L | LOWER | `ON E → ON FALSE` | mutated ⊆ original | ✅ |
| FixMInNullU | UPPER | `IN (v1, v2) → IN (v1, v2, NULL)` | original ⊆ mutated | ✅ |
| FixMBetweenDropUpperU | UPPER | `x BETWEEN a AND b → x >= a` | original ⊆ mutated | ✅ |
| FixMBetweenDropLowerU | UPPER | `x BETWEEN a AND b → x <= b` | original ⊆ mutated | ✅ |

#### 3.1.2 额外变异（3 个）

| 变异名称 | 类型 | 变换规则 | 蕴含关系 | 健全性 | 备注 |
|---------|------|---------|---------|--------|------|
| FixMNullEqToLowerL | LOWER | `a <=> b → a = b` | mutated ⊆ original | ⚠️ | NULL 处理差异可能导致假阳性 |
| FixMAllToAnyU | UPPER | `ALL → ANY` | original ⊆ mutated | ⚠️ | 空子查询边界情况需注意 |
| FixMAnyToAllL | LOWER | `ANY → ALL` | mutated ⊆ original | ⚠️ | 空子查询边界情况需注意 |

#### 3.1.3 PostgreSQL 变异（15 个）

- 上述 MySQL 变异的 `_Pg` 后缀版本
- 额外：`FixMIsNotDistinctFromToLowerL_Pg`（IS NOT DISTINCT FROM → =）

### 3.2 变异健全性分析

#### 3.2.1 FixMWhere1U/0L

**变换规则**：
```sql
-- 原始查询
SELECT * FROM T WHERE E

-- FixMWhere1U (UPPER)
SELECT * FROM T WHERE TRUE
-- 移除约束，结果集扩大 ✅

-- FixMWhere0L (LOWER)
SELECT * FROM T WHERE FALSE
-- 添加约束，结果集缩小 ✅
```

**健全性证明**：
- `WHERE TRUE` 返回所有行，包含原始结果集
- `WHERE FALSE` 返回空集，被原始结果集包含
- 在所有上下文中都成立

**评估**：✅ **完全健全**

#### 3.2.2 FixMCmpOpU/L

**变换规则**：
```sql
-- 原始查询
SELECT * FROM T WHERE X > 5

-- FixMCmpOpU (UPPER): > → >=
SELECT * FROM T WHERE X >= 5
-- 放宽条件，结果集扩大 ✅

-- FixMCmpOpL (LOWER): >= → >
SELECT * FROM T WHERE X > 5
-- 收紧条件，结果集缩小 ✅
```

**健全性证明**：
- 在正上下文中：放宽条件 → 结果集扩大
- 在负上下文中：放宽条件 → 结果集缩小（Flag 反转）
- Flag 传播机制正确处理上下文

**评估**：✅ **完全健全**（依赖 Flag 传播正确性）

#### 3.2.3 FixMBetweenDropUpperU/LowerU

**变换规则**：
```sql
-- 原始查询
SELECT * FROM T WHERE X BETWEEN 10 AND 20

-- FixMBetweenDropUpperU (UPPER): 移除上界
SELECT * FROM T WHERE X >= 10
-- 移除上界约束，结果集扩大 ✅

-- FixMBetweenDropLowerU (UPPER): 移除下界
SELECT * FROM T WHERE X <= 20
-- 移除下界约束，结果集扩大 ✅
```

**健全性证明**：
- 移除上界：满足 `X >= 10 AND X <= 20` 的行一定满足 `X >= 10`
- 移除下界：满足 `X >= 10 AND X <= 20` 的行一定满足 `X <= 20`
- 在所有上下文中都成立

**评估**：✅ **完全健全**

#### 3.2.4 FixMAllToAnyU/AnyToAllL

**变换规则**：
```sql
-- 原始查询
SELECT * FROM T WHERE X > ALL (SELECT Y FROM S)

-- FixMAllToAnyU (UPPER): ALL → ANY
SELECT * FROM T WHERE X > ANY (SELECT Y FROM S)
-- 放宽量词约束，结果集扩大 ✅
```

**健全性分析**：
- **一般情况**：`X > ALL (S)` 意味着 X 大于 S 中所有元素，`X > ANY (S)` 意味着 X 大于 S 中至少一个元素
- 满足 `ALL` 的行一定满足 `ANY`，因此 original ⊆ mutated ✅

**边界情况**：
```sql
-- 如果 S 为空集：
-- ALL (空集) = TRUE（vacuous truth，所有元素都满足条件）
-- ANY (空集) = FALSE（没有元素满足条件）

-- 原始查询返回所有行（TRUE）
-- 变异查询返回空集（FALSE）
-- 违反 UPPER 蕴含关系（original ⊆ mutated）❌
```

**评估**：⚠️ **在空子查询情况下不健全**

**建议**：
- 在文档中明确标注此边界情况
- 考虑在 Oracle 中添加空子查询检测逻辑
- 或接受此限制，监控假阳性率

#### 3.2.5 FixMNullEqToLowerL

**变换规则**：
```sql
-- 原始查询
SELECT * FROM T WHERE A <=> B

-- FixMNullEqToLowerL (LOWER): <=> → =
SELECT * FROM T WHERE A = B
-- 收紧条件，结果集缩小 ✅
```

**健全性分析**：
- **一般情况**：`A = B` 为 TRUE 时，`A <=> B` 也为 TRUE
- 满足 `=` 的行一定满足 `<=>`，因此 mutated ⊆ original ✅

**边界情况**：
```sql
-- A = NULL, B = NULL:
-- <=> 返回 TRUE（NULL-safe 等于）
-- = 返回 NULL（不是 TRUE）

-- 原始查询包含此行，变异查询不包含
-- 符合 LOWER 蕴含关系（mutated ⊆ original）✅
```

**评估**：✅ **技术上健全，但可能产生假阳性**

**建议**：
- 保持现状
- 监控假阳性率
- 如果假阳性率过高，考虑禁用此变异

---

## 4. Flag 传播机制

### 4.1 概述

Flag 传播机制用于跟踪 AST 遍历过程中的**正/负上下文**，确保变异方向的正确性。

```
flag = 1: 正上下文（蕴含方向不变）
flag = 0: 负上下文（蕴含方向反转）
```

### 4.2 传播规则

#### 4.2.1 正上下文传播

以下节点保持 flag 不变：
- `AND` 操作符
- `OR` 操作符
- `SELECT` 语句的 WHERE 子句
- `HAVING` 子句
- `JOIN ON` 条件

#### 4.2.2 负上下文传播

以下节点反转 flag：
- `NOT` 操作符
- `IS FALSE` 操作符
- `IS NOT TRUE` 操作符

#### 4.2.3 停止传播

以下节点停止递归遍历（保守策略）：
- 比较操作符（`=`, `>`, `<`, `>=`, `<=`, `!=`, `<>`）
- `IS NULL`, `IS NOT NULL`
- `IN`, `BETWEEN`, `LIKE`, `REGEXP`
- 子查询（除 `ANY`, `ALL`, `SOME`, `IN`, `EXISTS` 外）
- 控制流语句（`CASE`, `IF`）
- 函数调用
- 未知特性

### 4.3 示例

```sql
-- 示例 1: 正上下文
SELECT * FROM T WHERE X > 5
-- X > 5 在正上下文中
-- > → >= 是 UPPER 变异
-- isUpper = (U ^ Flag) ^ 1 = (1 ^ 1) ^ 1 = 1 ✅

-- 示例 2: 负上下文
SELECT * FROM T WHERE (X > 5) IS FALSE
-- X > 5 在 IS FALSE 下是负上下文
-- > → >= 是 UPPER 变异，但在负上下文中结果集缩小
-- isUpper = (U ^ Flag) ^ 1 = (1 ^ 0) ^ 1 = 0 ✅
```

### 4.4 正确性保证

Flag 传播机制的正确性依赖于：
1. **完整的 AST 节点分析**：所有可能反转上下文的节点都被正确处理
2. **保守的停止策略**：遇到未知节点时停止递归，避免错误传播

**已知限制**：
- 某些复杂表达式可能无法正确分析
- 保守策略可能导致遗漏部分变异点

**评估**：✅ **设计合理，文档清晰**

---

## 5. 已知问题和限制

### 5.1 Stage1 转换问题

**问题描述**：
Stage1 移除了大量查询特性（聚合、窗口函数、LEFT/RIGHT JOIN、LIMIT），这可能：
- 丢失原始查询的语义
- 无法发现与被移除特性相关的逻辑错误
- 降低测试覆盖率

**与论文设计的差异**：
论文中并未明确提到 Stage1 这样的预处理步骤。Pinolo 的核心思想是直接对原始查询进行近似合成。

**潜在影响**：
- 如果原始查询包含聚合函数，Stage1 会将其移除，导致无法测试聚合相关的逻辑错误
- LEFT/RIGHT JOIN 转换为 INNER JOIN 会改变查询语义，可能掩盖 JOIN 相关的逻辑错误

**建议改进**：
- **短期**：保持现状，Stage1 简化了实现
- **长期**：考虑更细粒度的转换策略，保留部分查询特性
- **配置化**：添加配置选项，允许用户控制 Stage1 的转换级别

### 5.2 部分变异的边界情况

#### 5.2.1 FixMAllToAnyU/AnyToAllL 的空子查询问题

详见 [§3.2.4 FixMAllToAnyU/AnyToAllL](#324-fixmlltoanyuanytoall)

**建议**：
- 在文档中明确标注此边界情况
- 考虑在 Oracle 中添加空子查询检测逻辑

#### 5.2.2 FixMNullEqToLowerL 的 NULL 处理

详见 [§3.2.5 FixMNullEqToLowerL](#325-fixmnulleqtolowerl)

**建议**：
- 保持现状，但监控假阳性率

### 5.3 Oracle 比较逻辑问题

**当前实现**：
```go
func CMP(result1, result2) int {
    // 简单的结果集比较
    // 返回 -1, 0, 1 表示子集关系
}
```

**潜在问题**：
1. **浮点数精度**：浮点数比较可能因精度问题导致误判
2. **NULL 处理**：NULL 值的比较逻辑可能不够健壮
3. **排序敏感性**：如果结果集未排序，比较可能不稳定

**与论文设计的差异**：
论文中提到了更复杂的结果比较策略，包括：
- 多次执行取交集（减少非确定性影响）
- 更精细的 NULL 处理逻辑

**建议改进**：
- **短期**：保持现状，当前的简单比较在大多数情况下足够
- **长期**：考虑添加多次执行和更精细的 NULL 处理

### 5.4 缺乏系统化的查询合成框架

**当前实现**：
当前实现更像是**基于规则的变异**，而非系统化的查询合成。每个变异都是独立的规则，缺乏统一的合成框架。

**与论文设计的差异**：
论文中描述了一个系统化的查询合成框架，包括：
- 查询模式识别
- 约束提取
- 近似策略选择
- 蕴含关系验证

**影响**：
- 当前实现更简单，易于理解和维护
- 但缺乏论文中描述的系统化方法
- 可能限制了变异策略的扩展性

**建议改进**：
- **短期**：保持现状，当前实现已经有效
- **长期**：考虑引入更系统化的查询合成框架

### 5.5 随机 SQL 生成器的对齐问题

**当前实现**：
```go
// generator 模块生成随机 SQL 查询
// 使用预定义的多数据库测试数据集
```

**潜在问题**：
1. **数据生成与查询生成的分离**：当前实现中，数据生成和查询生成是独立的，可能导致生成的查询无法充分利用数据特性
2. **缺乏反馈机制**：生成器无法根据历史测试结果调整生成策略

**与论文设计的差异**：
论文中的 AQS 是基于**给定查询**进行近似合成，而非随机生成查询。

**建议改进**：
- 明确区分两种测试模式：
  1. **基于给定查询的 AQS**（论文核心方法）
  2. **随机查询生成 + AQS**（扩展方法）
- 在文档中说明两种模式的差异和适用场景

---

## 6. 与论文设计的一致性评估

### 6.1 总体评估

**当前实现与论文设计的一致性**：**75-80%**

| 维度 | 评分 | 说明 |
|------|------|------|
| **方法论一致性** | 90% | 核心方法论与论文一致（AQS + Implication Oracle） |
| **变异策略健全性** | 85% | 大部分变异健全，少数边界情况需注意 |
| **代码质量** | 90% | EET 移除后代码更清晰 |
| **架构一致性** | 70% | 基于规则 vs 系统化框架有差异 |
| **Oracle 鲁棒性** | 75% | 相对简单，可改进空间 |

### 6.2 符合论文设计的方面

✅ **核心方法论一致**：
- 所有变异都是近似变换（添加/移除约束）
- 每个变异都有明确的蕴含关系（UPPER 或 LOWER）
- 使用 Implication Oracle 检查蕴含关系

✅ **变异策略合理**：
- 基于规则的近似变换，简单易懂
- 大部分变异在数学上是健全的
- Flag 传播机制正确处理正/负上下文

✅ **EET 移除成功**：
- v0.4.0 移除了所有等价变换
- 回归核心方法论
- 代码更清晰，维护成本降低

### 6.3 与论文设计的差异

⚠️ **Stage1 转换过于激进**：
- 移除大量查询特性，可能丢失语义
- 论文中并未明确提到 Stage1

⚠️ **缺乏系统化的查询合成框架**：
- 基于规则的变异 vs 系统化框架
- 架构层面的主要差异

⚠️ **Oracle 比较逻辑相对简单**：
- 缺乏多次执行取交集
- 浮点数精度、NULL 处理可能不够健壮

---

## 7. 未来改进方向

### 7.1 高优先级（1-3 个月）

1. **完善文档**：明确当前实现与论文设计的差异
2. **添加回归测试**：确保 EET 移除后系统功能正常
3. **监控假阳性率**：跟踪系统的假阳性率，识别问题变异

### 7.2 中优先级（3-6 个月）

4. **改进 Stage1 转换策略**：减少激进转换，保留更多查询特性
5. **增强 Oracle 比较逻辑**：提高鲁棒性，减少误判
6. **优化随机 SQL 生成器**：提高生成查询的质量和相关性

### 7.3 低优先级（6-12 个月）

7. **引入系统化的查询合成框架**：实现论文中描述的系统化方法
8. **扩展测试覆盖范围**：支持更多数据库和查询特性

详见 [ARCHITECTURE_REVIEW_v0.4.0.md](ARCHITECTURE_REVIEW_v0.4.0.md)

---

## 8. 参考资料

1. **原始论文**：
   - Hao, et al. "Detecting Logical Bugs in Database Management Systems with Approximate Query Synthesis." ATC'23.
   - 论文 PDF：[atc23-hao-pinolo.pdf](atc23-hao-pinolo%20Detecting%20Logical%20Bugs%20in%20Database%20Management%20Systems%20with%20Approximate%20Query%20Synthesis.pdf)

2. **相关工具**：
   - SQLancer：https://github.com/sqlancer/sqlancer
   - Squirrel：https://github.com/s3team/Squirrel

3. **项目文档**：
   - [USER_GUIDE.md](USER_GUIDE.md)：用户指南
   - [release_notes.md](release_notes.md)：发布说明
   - [ARCHITECTURE_REVIEW_v0.4.0.md](ARCHITECTURE_REVIEW_v0.4.0.md)：架构审查报告

---

**文档版本历史**：
- v1.0 (2026-06-03)：初始版本，基于 v0.4.0 代码库
