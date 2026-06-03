# Pinolo 语法覆盖分析（修正版）——从蕴含 Oracle 核心方法论出发

## 一、方法论纠正

### 之前分析的错误

之前的分析以"等价变换"（EET）为主要维度，将 IFNULL→COALESCE、LEAST→CASE、= ANY→IN 等归为"遗漏的等价变换"。这是方向性错误：

- **Pinolo 的核心数学基础是蕴含关系（包含/非包含）**，不是等价关系
- 等价变换（DeMorgan、BETWEEN→比较、COALESCE→CASE 等）只是 SECONDARY 附加手段
- SQLancer 的所有 oracle（EET/TLP/NoREC/PQS/CERT）都是等价检测，**没有蕴含 oracle**
- 蕴含 Oracle 是 Pinolo 的独创方法论，应以此为主要分析维度

### Pinolo 蕴含 Oracle 的数学模型

```
Upper mutation: 放松谓词 → 预期 original ⊆ mutated
Lower mutation: 收紧谓词 → 预期 mutated ⊆ original
Bug检出条件:   包含关系被违反 → 逻辑 bug

Flag传播:      NOT/IS FALSE/NOT IN/NOT LIKE/NOT REGEXP/NOT EXISTS/ALL → flag ^= 1
IsUpper计算:   ((U ^ Flag)^1) == 1 → 变异结果扩大；== 0 → 变异结果缩小
```

### 从蕴含视角，增强的价值来源于三类

**A类：新的蕴含变异点**（新的放松/收紧谓词机会）
- 例：BETWEEN 放松上界 → x >= a（更多行预期）

**B类：新的自然包含关系**（不依赖谓词变换的结构性包含）
- 例：INTERSECT ⊆ Q1，LEFT JOIN ⊇ INNER JOIN

**C类：表达式多样性**（让现有变异触发不同优化器路径）
- 例：JSON/ENUM 列参与比较 → FixMCmpOpU/L 覆盖但触发 JSON 优化路径

---

## 二、从蕴含视角审视当前覆盖

### 2.1 当前蕴含变异清单（Pinolo 核心力量）

| 变异 | 放松/收紧 | 包含关系 | 覆盖范围 |
|------|-----------|---------|---------|
| FixMDistinctU/L | DISTINCT→非DISTINCT / 反向 | DISTINCT结果 ⊆ 非DISTINCT结果 | ✅ 全DBMS |
| FixMUnionAllU/L | UNION→UNION ALL / 反向 | UNION结果 ⊆ UNION ALL结果 | ✅ MySQL/PG/M/A |
| FixMCmpOpU/L | >→>=, <→<=, =→>= / 反向 | 更严格比较 ⊆ 更宽松比较 | ✅ 全DBMS |
| FixMInNullU | IN(list)→IN(list,NULL) | IN结果 ⊆ IN+NULL结果 | ✅ MySQL/M |
| FixMWhere1U/0L | WHERE E→TRUE/FALSE | 原结果 ⊆ WHERE TRUE / WHERE FALSE ⊆ 原结果 | ✅ 全DBMS |
| FixMHaving1U/0L | HAVING E→TRUE/FALSE | 同WHERE | ✅ 全DBMS |
| FixMOn1U/0L | ON E→TRUE/FALSE | 同WHERE | ✅ 全DBMS |
| FixMRmUnionAllL | 删除UNION ALL分支 | 子集 ⊆ 全集 | ✅ MySQL/PG |
| RdMLikeU/L | LIKE模式放松/收紧 | 更严格模式 ⊆ 更宽松模式 | ✅ MySQL/M |
| RdMRegExpU/L | REGEXP放松/收紧 | 同LIKE | ✅ MySQL/M |
| FixMAndTrueU | 永真式∧E | E ⊆ 永真∧E(预期等价) | ✅ 全DBMS |
| FixMOrFalseL | 永假式∨E | 永假∨E ⊆ E(预期等价) | ✅ 全DBMS |
| FixMCaseTrueU/L | CASE TRUE/FALSE包裹 | 同AndTrue/OrFalse | ✅ 全DBMS |

### 2.2 当前蕴含变异的覆盖盲点

| 表达式类型 | 当前处理 | 蕴含变异机会 | 评估 |
|-----------|---------|-------------|------|
| **BETWEEN** | 仅等价(BETWEEN→>=AND<=) | ❌ 缺蕴含：BETWEEN→>= (upper), BETWEEN→<= (upper) | **遗漏** |
| **NOT BETWEEN** | 无变异 | ❌ 缺蕴含：NOT BETWEEN→< (upper), →> (upper) | **遗漏** |
| **INTERSECT** | 无处理 | ❌ 缺自然包含：INTERSECT ⊆ Q1 | **遗漏** |
| **EXCEPT** | 无处理 | ❌ 缺自然包含：EXCEPT ⊆ Q1 | **遗漏** |
| **LEFT/RIGHT JOIN** | Stage1转INNER JOIN | ❌ 缺自然包含：INNER ⊆ LEFT | **遗漏** |
| **ANY/ALL跨量词** | 仅FixMCmpOpU/L | ❌ 缺蕴含：ALL(subq) ⊆ ANY(subq) | **遗漏** |
| **<=> (NULL-safe eq)** | skip，无变异 | ❌ 缺蕴含：a=b ⊆ a<=>b (lower) | **遗漏** |
| **IS NULL/IS NOT NULL** | visitor空实现 | ❌ 缺蕴含：IS NOT NULL→TRUE(upper) | **遗漏** |
| **!=/<>** | FixMCmpOpL: !=→< | ⚠️ !=→< 不一定成立 | **需修正** |

---

## 三、蕴含变异遗漏的可行性评估

### 3.1 BETWEEN 蕴含变异（A类：新的放松/收紧点）

**数学证明**：
```
x BETWEEN a AND b 的结果集 = {行 | x >= a AND x <= b}
x >= a 的结果集              = {行 | x >= a}

x BETWEEN a AND b ⊆ x >= a  (满足上下界的行必然满足下界)
x BETWEEN a AND b ⊆ x <= b  (满足上下界的行必然满足上界)

Upper mutation: x BETWEEN a AND b → x >= a  (预期 original ⊆ mutated)
Upper mutation: x BETWEEN a AND b → x <= b  (预期 original ⊆ mutated)
Lower mutation: x BETWEEN a AND b → x = a   (预期 mutated ⊆ original, 仅精确值)
```

| 维度 | MySQL | PG | GaussDB-M | GaussDB-A |
|------|-------|----|-----------|-----------|
| 语法支持 | ✅ | ✅ | ✅ | ✅ |
| Parser支持 | ✅ BetweenExpr | ✅ A_Expr BETWEEN | ✅ | ✅ |
| 蕴含关系 | ✅ 严格包含 | ✅ 严格包含 | ✅ | ✅ |
| NULL语义 | ⚠️ NULL BETWEEN → NULL, NULL >= → NULL | ⚠️ 同 | ⚠️ | ⚠️ |
| 实现难度 | **低** | **中** (需PG引擎) | **低** | **低** |

**实现方案**：
1. 新增常量：`FixMBetweenDropUpperBoundU`（BETWEEN→>=），`FixMBetweenDropLowerBoundU`（BETWEEN→<=）
2. MySQL: 在 BetweenExpr 上新增 mining 逻辑，构建 `BinaryOperationExpr{x >= a}` 替换 BetweenExpr
3. PG: 在 A_Expr BETWEEN kind 上新增类似逻辑
4. **NOT BETWEEN同理**：`x NOT BETWEEN a AND b` → `x < a`（upper, flag翻转后可能为lower）

**优势**：
- **蕴含关系严格成立**，无 NULL 边界问题（NULL BETWEEN → NULL，NULL >= → NULL，包含关系不违反）
- BETWEEN 的上下界分别放松，触发优化器对范围查询的索引选择差异（可能走不同索引）
- 实现简单，和 FixMBetweenToCmp（等价变换）在同一节点

**劣势**：
- `x >= a` 是非常简单的放松，可能太"明显"，优化器很少出错

**可行性结论**：✅ 四款 DBMS 均可实现。**推荐立即实施**。

---

### 3.2 INTERSECT/EXCEPT 自然包含关系（B类：结构性包含）

**数学证明**：
```
Q1 INTERSECT Q2 = {行 | 行∈Q1 AND 行∈Q2}
Q1 INTERSECT Q2 ⊆ Q1  (交集是子集)
Q1 INTERSECT Q2 ⊆ Q2  (交集是子集)
Q1 EXCEPT Q2 ⊆ Q1      (差集是子集)

Upper mutation: Q1 INTERSECT Q2 → Q1  (预期 INTERSECT ⊆ Q1)
Upper mutation: Q1 INTERSECT Q2 → Q2  (预期 INTERSECT ⊆ Q2)
Upper mutation: Q1 EXCEPT Q2 → Q1     (预期 EXCEPT ⊆ Q1)
```

| 维度 | MySQL | PG | GaussDB-M | GaussDB-A |
|------|-------|----|-----------|-----------|
| 语法支持 | ❌ 无INTERSECT/EXCEPT | ✅ | ❌ | ✅ (Oracle兼容) |
| Parser支持 | ❌ TiDB无法解析 | ✅ SetOperationStmt | ❌ | ❌ TiDB限制 |
| 蕴含关系 | N/A | ✅ 严格包含 | N/A | ❌ parser不可达 |
| NULL语义 | N/A | ⚠️ INTERSECT中NULL=NULL,但SQL中NULL≠NULL | N/A | N/A |
| 实现难度 | N/A | **中-高** | N/A | ❌ 不可行 |

**PostgreSQL NULL 处理关键**：
```sql
-- INTERSECT 中两个 NULL 被视为相等（SQL标准）
SELECT NULL INTERSECT SELECT NULL → {NULL}  (一行)

-- 但 CMP 中 NULL 作为字符串"NULL"比较：
"NULL" == "NULL" → true  (Pinolo Result中NULL表示为字符串)
```
所以 Pinolo 的 CMP 逻辑天然支持 INTERSECT 的 NULL 语义（因为 "NULL" 字符串相等）。

**实现方案（PostgreSQL）**：
1. pg_mutatevisitor 新增 `visitSetOperationStmt` 处理 INTERSECT/EXCEPT
2. 新增常量：`FixMIntersectToUpper_Pg`（INTERSECT→Q1），`FixMExceptToUpper_Pg`（EXCEPT→Q1）
3. 变换实现：将 `SetOperationStmt{op=INTERSECT}` 替换为第一个子查询的 SelectStmt
4. 用 `pg_replaceExprInRoot` 或直接 Deparse 第一个子查询

**优势**：
- **自然包含关系，无需构造复杂变换**
- INTERSECT→Q1 检测的是交集与原查询的包含关系违反，这是纯粹的优化器逻辑 bug（如 INTERSECT 实现错误地包含了不该有的行，或错误地排除了该有的行）
- SQLancer EET 的 INTERSECT→EXISTS 等价变换也能发现 bug，但那是等价维度；蕴含维度（INTERSECT ⊆ Q1）可能发现不同类型的 bug

**劣势**：
- PG 变异引擎需要改造（当前只处理 UNION）
- GaussDB-A 因 TiDB parser 限制不可行
- INTERSECT→Q1 的放松可能太"明显"（Q1总是比INTERSECT大），bug 概率可能低于等价变换

**可行性结论**：
- ✅ PostgreSQL 可实现（需 PG 引擎改造）
- ❌ MySQL/GaussDB-M 不适用
- ❌ GaussDB-A 当前不可行（TiDB parser 限制）
- **推荐作为 Phase 2，配合 PG 引擎改造**

---

### 3.3 LEFT/RIGHT JOIN → INNER JOIN 蕴含关系（B类：结构性包含）

**数学证明**：
```
INNER JOIN on T1, T2 with ON cond:
  结果 = {行 | 行∈T1×T2 AND cond(行)=TRUE}

LEFT JOIN on T1, T2 with ON cond:
  结果 = INNER JOIN结果 ∪ {T1中不匹配的行, NULL补齐}

LEFT JOIN ⊇ INNER JOIN  (外连接包含内连接结果 + 不匹配行)
```

| 维度 | MySQL | PG | GaussDB-M | GaussDB-A |
|------|-------|----|-----------|-----------|
| 当前状态 | Stage1转INNER后只测ON变异 | 同 | 同 | 同 |
| 蕴含关系 | ✅ INNER ⊆ LEFT | ✅ | ✅ | ✅ |
| 实现难度 | **中** | **中** | **中** | **中** |

**当前问题**：Stage1 将 LEFT/RIGHT JOIN 转为 INNER JOIN，然后只测 FixMOn1U/0L。这意味着：
- INNER JOIN ON TRUE ⊆ LEFT JOIN ON TRUE（应该是 upper mutation）
- 但 Stage1 转换后，LEFT JOIN 信息丢失，无法做此变异

**实现方案**：
1. **方案A：保留 LEFT/RIGHT JOIN，不转 INNER JOIN**
   - 修改 Stage1 `rmlrjoin.go`，不再转换 LEFT/RIGHT JOIN
   - 修改 Stage2 `visitJoin`，不再跳过 LEFT/RIGHT JOIN
   - 新增蕴含变异：INNER JOIN → LEFT JOIN（upper），LEFT JOIN → INNER JOIN（lower）
   - 保留 FixMOn1U/0L 适用于所有 JOIN 类型
   
2. **方案B：在 Stage1 转换前做 LEFT/RIGHT JOIN 蕴含测试**
   - 在 Stage1 处理前，对原始 SQL（含 LEFT JOIN）执行获取原结果
   - 将 LEFT JOIN 转为 INNER JOIN 后执行获取变异结果
   - 检查：变异结果（INNER） ⊆ 原结果（LEFT）
   - 这需要修改 Stage1 流程，增加"转换前蕴含测试"步骤

**方案A vs B 分析**：

| 对比 | 方案A（保留LEFT JOIN） | 方案B（转换前测试） |
|------|----------------------|-------------------|
| 设计侵入性 | 高（修改Stage1+Stage2+Visitor） | 中（仅修改Stage1流程） |
| WHERE变异兼容性 | ⚠️ LEFT JOIN+WHERE 语义复杂 | ✅ Stage1转换后WHERE变异不受影响 |
| ON变异兼容性 | ✅ ON条件变异适用于所有JOIN | ✅ 同 |
| 实现难度 | 高 | 中 |

**推荐方案B**：在 Stage1 转换前增加一步 LEFT/RIGHT JOIN → INNER JOIN 蕴含测试。这样既保留了现有 Stage1/Stage2 对 INNER JOIN 的完整变异体系，又新增了 LEFT/RIGHT JOIN 的结构性蕴含测试。

**优势**：
- LEFT JOIN ⊇ INNER JOIN 是严格的自然蕴含关系，无需构造变换
- 外连接 vs 内连接的优化器路径差异显著（索引选择、NULL 填充策略）
- 数据库在 LEFT JOIN 实现上的 bug 较常见（如 MySQL 的某些版本在 LEFT JOIN + WHERE 条件组合上产生错误结果）

**劣势**：
- LEFT JOIN + WHERE 条件组合的语义需要仔细处理
  - `SELECT * FROM t1 LEFT JOIN t2 ON a WHERE b` 中 WHERE b 过滤的是 LEFT JOIN 后的结果
  - 所以 LEFT JOIN WHERE b ⊆ LEFT JOIN（WHERE 收紧了结果）
  - 但 LEFT JOIN WHERE b 和 INNER JOIN WHERE b 的包含关系更复杂
- 方案B 需要额外的执行步骤（增加测试时间）

**可行性结论**：✅ 四款 DBMS 均可实现（方案B）。**推荐 Phase 2 实施**。

---

### 3.4 ANY/ALL 跨量词蕴含变异（A类：新的放松/收紧点）

**数学证明**：
```
x > ALL(subq): x 大于子查询的所有值 → x 大于子查询的任意值
x > ALL(subq) ⊆ x > ANY(subq)  (满足"大于全部"必然满足"大于某个")

x = ALL(subq): x 等于子查询的所有值 → x 等于子查询的某个值
x = ALL(subq) ⊆ x = ANY(subq)  (满足"等于全部"必然满足"等于某个")

Upper mutation: x > ALL(subq) → x > ANY(subq)  (预期 ALL结果 ⊆ ANY结果)
Upper mutation: x = ALL(subq) → x = ANY(subq)  (预期 ALL结果 ⊆ ANY结果)

Lower mutation: x > ANY(subq) → x > ALL(subq)  (预期 ALL结果 ⊆ ANY结果，反向即 ANY ⊇ ALL)
```

| 维度 | MySQL | PG | GaussDB-M | GaussDB-A |
|------|-------|----|-----------|-----------|
| 语法支持 | ✅ ANY/ALL/SOME | ✅ | ✅ | ✅ |
| Parser支持 | ✅ CompareSubqueryExpr | ✅ SubLink | ✅ | ✅ |
| 蕴含关系 | ✅ 严格包含(无NULL边界) | ✅ | ✅ | ✅ |
| NULL语义 | ⚠️ 需评估 | ⚠️ | ⚠️ | ⚠️ |
| 实现难度 | **中** | **中** | **中** | **中** |

**NULL 语义关键**：
```sql
-- 当子查询返回空集：
x > ALL(empty) → TRUE  (空集的"全部"条件成立)
x > ANY(empty) → FALSE (空集的"某个"条件不成立)
ALL ⊆ ANY? FALSE ⊆ TRUE? 不成立！ALL结果包含ALL行，ANY结果为空

-- 当子查询含 NULL：
x > ALL(subq_with_NULL) → NULL (NULL不满足>，所以ALL包含NULL时结果为NULL)
x > ANY(subq_with_NULL) → x > some_non_null OR NULL
蕴含关系在含 NULL 时可能被打破！
```

**修正**：ALL→ANY 蕴含关系在**子查询不含 NULL 且非空**时成立。在含 NULL 或空子查询时可能打破。

**实现方案**：
1. 新增常量：`FixMAllToAnyU`（ALL→ANY/SOME），`FixMAnyToAllL`（ANY→ALL）
2. MySQL: 在 CompareSubqueryExpr 上，当 `in.All=true` 时，将 `All` 改为 `false`（ANY），反之亦然
3. PG: 在 SubLink 上，当 `subLinkType=ANY_SUBLINK` 时，改为 ALL_SUBLINK

**优势**：
- ALL→ANY 是自然的蕴含关系，不需要构造复杂变换
- ANY/ALL 子查询的优化器实现差异大（materialization vs semi-join vs anti-join）
- MySQL/MariaDB/TiDB 在 ANY/ALL 优化上有已知 bug 历史

**劣势**：
- NULL 语义可能打破蕴含关系，导致误报
- 需要在 oracle 检查时处理 NULL 误报（或记录误报率，接受一定误报）
- ALL→ANY 的放松可能太"明显"

**可行性结论**：✅ 四款 DBMS 均可实现，但需接受 NULL 误报。**推荐 Phase 1 实施，标注 NULL 误报风险**。

---

### 3.5 <=> (NULL-safe eq) 蕴含变异（A类：新的放松/收紧点）

**数学证明**：
```
a <=> b: NULL-safe 等值
  a=NULL, b=NULL → TRUE
  a=NULL, b≠NULL → FALSE
  a≠NULL, b≠NULL → a=b的结果

a = b: 普通等值
  a=NULL, b=NULL → NULL (非TRUE)
  a=NULL, b≠NULL → NULL (非FALSE)
  a≠NULL, b≠NULL → a=b的结果

a = b 的TRUE结果 ⊆ a <=> b 的TRUE结果
  (a=b为TRUE的行，<=>也为TRUE；但<=>还包含NULL=NULL为TRUE的行)

Lower mutation: a <=> b → a = b  (预期 =结果 ⊆ <=>结果)
```

| 维度 | MySQL | PG | GaussDB-M | GaussDB-A |
|------|-------|----|-----------|-----------|
| <=>语法 | ✅ | ❌ (用IS NOT DISTINCT FROM) | ✅ | ❌ |
| Parser支持 | ✅ opcode.NullEQ | ❌ | ✅ | ❌ |
| 蕴含关系 | ✅ a=b ⊆ a<=>b | ✅ (IS NOT DISTINCT FROM版) | ✅ | ✅ (NVL相关?) |
| NULL语义 | ✅ 严格成立 | ✅ | ✅ | N/A |
| 实现难度 | **低** | **中** (不同语法) | **低** | ❌ |

**实现方案（MySQL/GaussDB-M）**：
1. 新增常量：`FixMNullEqToLowerL`（<=>→=）
2. 在 visitBinaryOperationExpr 中，不再 skip `opcode.NullEQ`
3. 变换：将 `a <=> b` 的 `Op` 从 `NullEQ` 改为 `EQ`（=）
4. 用 replaceExprInRoot 替换整个 BinaryOperationExpr

**PG 版本**：`a IS NOT DISTINCT FROM b` → `a = b`（同样蕴含，但语法不同）

**优势**：
- <=>→= 是严格的蕴含关系（a=b ⊆ a<=>b），无 NULL 误报
- <=> 和 = 在 MySQL 中触发不同优化路径（<=> 可能不走索引，= 走索引）
- 实现极简单

**劣势**：
- 只有 lower mutation（<=>→=），没有自然 upper mutation
- <=> 在实际 SQL 中出现频率不高

**可行性结论**：✅ MySQL/GaussDB-M 可实施。✅ PG/GaussDB-A 用 IS NOT DISTINCT FROM 版本可实施。**推荐 Phase 1**。

---

### 3.6 IS NULL/IS NOT NULL 蕴含变异（A类：新的放松/收紧点）

**数学证明**：
```
IS NOT NULL:
  原结果 = {行 | x IS NOT NULL 为TRUE}
  WHERE TRUE = {所有行}
  原结果 ⊆ WHERE TRUE

Upper mutation: WHERE x IS NOT NULL → WHERE TRUE  (预期 IS NOT NULL ⊆ TRUE)
  但这已被 FixMWhere1U 覆盖（整个 WHERE 替换为 TRUE）

Lower mutation: WHERE x IS NOT NULL → WHERE FALSE  (预期 FALSE ⊆ IS NOT NULL)
  这也被 FixMWhere0L 覆盖

所以 IS NOT NULL 的整个 WHERE 替换已被覆盖。
但 IS NOT NULL 的部分收紧不被覆盖：
  WHERE x IS NOT NULL → WHERE x > 0  (预期 x>0 ⊆ x IS NOT NULL，如果所有非NULL值为正)

⚠️ 这需要语义假设（非NULL值>0），不严格成立。
```

**重新审视**：IS NULL/IS NOT NULL 的蕴含变异其实**大部分已被 FixMWhere1U/0L 覆盖**（因为 WHERE→TRUE/FALSE 包含了所有谓词替换）。单独做 IS NOT NULL 变异不会增加新的蕴含点。

**但有价值的遗漏**：
```sql
-- 混合谓词中的 IS NOT NULL:
WHERE x IS NOT NULL AND y > 5
  → FixMWhere1U: WHERE TRUE (upper, 全覆盖)
  → FixMWhere0L: WHERE FALSE (lower, 全覆盖)

  但缺少：
  WHERE x IS NOT NULL AND y > 5
  → WHERE y > 5  (upper, 部分放松：去掉IS NOT NULL约束)
  预期: (x IS NOT NULL AND y>5) ⊆ (y>5)

  这是有价值的蕴含变异！去掉WHERE中的某个子条件（部分放松）。
```

**但这不是 IS NULL 特有问题**——这是"谓词部分放松"的通用缺失。当前 FixMWhere1U 是"全部放松"（WHERE→TRUE），缺少"部分放松"（去掉WHERE中的某一项）。

**可行性结论**：⚠️ IS NULL 单独的蕴含变异已被 FixMWhere1U/0L 间接覆盖。"谓词部分放松"是一个更大的设计方向，需要拆解 WHERE 的 AND 组合结构，**实现复杂度高，暂不推荐**。

---

### 3.7 !=/<> 蕴含变异修正（A类：已有实现的修正）

**当前实现问题**：
```go
// FixMCmpOpL: != → < (arbitrary choice)
// 这不是严格蕴含！
// a != b → a < b: 如果 a=5, b=5, != 为 FALSE, < 也为 FALSE → 包含成立
//               如果 a=5, b=3, != 为 TRUE, < 为 FALSE → 包含不成立！
// a != b 为TRUE的行不一定满足 a < b

// 所以 FixMCmpOpL: != → < 不保证蕴含关系！
```

**修正方案**：
1. `!= → TRUE` (upper: !=结果 ⊆ TRUE结果... 但这不是"放松"，是"完全替换")
2. 或：移除 != 的 FixMCmpOpL 变异（因为不保证蕴含）
3. 或：将 != 视为 `(a < b) OR (a > b)`，放松为 `< OR >=` → `TRUE`

**可行性结论**：⚠️ 当前 !=→< 的蕴含关系不严格成立，应**修正或移除**。但这不是新增强，而是现有实现的 bug 修正。**推荐立即修正**。

---

## 四、C类：表达式多样性增强（让现有蕴含变异触发新优化路径）

### 4.1 表达式多样性如何服务于蕴含 Oracle

当前蕴含变异（FixMCmpOpU/L, FixMWhere1U/0L, FixMOn1U/0L 等）的变异逻辑已全面覆盖基本的比较/逻辑运算。但**同样的变异在不同表达式上触发不同的优化器路径**：

```
WHERE age > 25 → WHERE age >= 25  (整数比较，简单优化路径)
WHERE JSON_EXTRACT(data, '$.age') > 25 → WHERE JSON_EXTRACT(data, '$.age') >= 25  (JSON函数比较，触发JSON优化路径)
```

所以增加表达式多样性（JSON、ENUM、SELF JOIN）不是为了做等价变换，而是为了让**现有蕴含变异覆盖更多优化器路径**。

### 4.2 表达式多样性评估

| 增强项 | 优化器新路径 | 蕴含变异覆盖 | 实现难度 | Bug潜力 |
|--------|-------------|-------------|---------|---------|
| **JSON函数列** | JSON索引选择、JSON函数计算 | FixMCmpOpU/L 自动覆盖 | 高(需生成体系) | 高 |
| **ENUM列** | ENUM索引选择、ENUM值比较 | FixMCmpOpU/L 自动覆盖 | 低(MySQL)/中(PG) | 中 |
| **SELF JOIN** | 自引用别名解析、同表索引策略 | FixMOn1U/0L 自动覆盖 | 低 | 中 |
| **ARRAY列(PG)** | 数组包含运算符优化 | 需新增ARRAY蕴含变异 | 高 | 中-高 |
| **UUID列(PG)** | UUID索引比较优化 | FixMCmpOpU/L 自动覆盖 | 低 | 低 |
| **IFNULL/IF函数** | 函数实现路径差异 | FixMCmpOpU/L(比较)覆盖 | 低(等价)+低(生成) | 低-中 |

**关键发现**：JSON 和 ENUM 的价值不在等价变换，而在让 FixMCmpOpU/L 等蕴含变异触发 JSON/ENUM 优化器路径。这与之前的分析方向一致，但动机不同。

---

## 五、修正后的优先级排序

### 从蕴含 Oracle 核心方法论出发

### Phase 0（立即修正）

| 序号 | 增强项 | 类型 | 说明 |
|------|--------|------|------|
| **P0** | !=→< 蕴含关系修正 | 修正 | !=→< 不保证蕴含，应修正或移除 |

### Phase 1（低成本，严格蕴含，低误报）

| 序号 | 增强项 | 类型 | MySQL | PG | GaussDB-M | GaussDB-A | 优势 |
|------|--------|------|-------|----|-----------|-----------|------|
| **P1-1** | BETWEEN→>= / BETWEEN→<= | A类蕴含 | ✅ | ✅ | ✅ | ✅ | 严格包含，新增放松点 |
| **P1-2** | <=>→= (lower) | A类蕴含 | ✅ | ✅(IS NOT DISTINCT FROM) | ✅ | ❌ | 严格包含，新增收紧点 |
| **P1-3** | ALL→ANY (upper) | A类蕴含 | ✅ | ✅ | ✅ | ✅ | 自然包含，⚠️ NULL误报 |
| **P1-4** | SELF JOIN生成 | C类多样性 | ✅ | ✅ | ✅ | ✅ | 让FixMOn触发新路径 |
| **P1-5** | ENUM列生成 | C类多样性 | ✅ | ⚠️ | ✅ | ❌ | 让FixMCmpOp触发新路径 |

### Phase 2（中等投入，需架构改造或 NULL 处理）

| 序号 | 增强项 | 类型 | MySQL | PG | GaussDB-M | GaussDB-A | 需改造 |
|------|--------|------|-------|----|-----------|-----------|--------|
| **P2-1** | INTERSECT→Q1 (蕴含) | B类结构性 | ❌ | ✅ | ❌ | ❌ | PG引擎改造 |
| **P2-2** | EXCEPT→Q1 (蕴含) | B类结构性 | ❌ | ✅ | ❌ | ❌ | PG引擎改造 |
| **P2-3** | LEFT→INNER (蕴含) | B类结构性 | ✅ | ✅ | ✅ | ✅ | Stage1流程改造 |
| **P2-4** | JSON表达式生成 | C类多样性 | ✅ | ✅ | ✅ | ✅ | 新增生成体系 |
| **P2-5** | IFNULL→COALESCE (等价) | EET等价 | ✅ | ❌ | ✅ | ✅(NVL) | 低(仅exprReplacer扩展) |

### Phase 3（高投入，远期目标）

| 序号 | 增强项 | 类型 | 说明 |
|------|--------|------|------|
| **P3-1** | 谓词部分放松(去掉WHERE某子条件) | A类蕴含 | 需拆解AND组合结构，复杂 |
| **P3-2** | GaussDB-A改用pg_query | 架构改造 | 使INTERSECT/EXCEPT可达 |
| **P3-3** | ARRAY蕴含变异(PG) | A类蕴含 | 数组包含运算符蕴含 |
| **P3-4** | JSON等价变换 | EET等价 | Parser限制+NULL语义 |

---

## 六、总结：蕴含视角 vs 等价视角的差异

| 维度 | 等价视角（之前分析） | 蕴含视角（修正分析） |
|------|---------------------|---------------------|
| 核心方法论 | EET等价变换是主要遗漏 | 蕴含变异是核心力量，等价变换是补充 |
| 最高优先级遗漏 | IFNULL→COALESCE等价 | BETWEEN蕴含(放松上界/下界) |
| 最大结构性遗漏 | JSON等价变换 | INTERSECT⊆Q1, LEFT⊇INNER |
| 表达式多样性动机 | 做"新的等价变换" | 让现有蕴含变异触发新优化路径 |
| !=/<>处理 | 未发现问题 | !=→<蕴含不成立，需修正 |
| ALL→ANY | "= ANY→IN等价变换" | "ALL⊆ANY蕴含变异"（同一变换，不同理解） |
| <=>处理 | "<=>→CASE等价变换" | "<=>→=蕴含变异（lower）"（同一变换，不同理解） |

**核心结论**：Pinolo 的增强应以**蕴含变异扩展**为主要方向，等价变换作为补充。蕴含 Oracle 是 Pinolo 的独创方法论和核心竞争力，增强应围绕"新的包含关系"而非"新的等价关系"展开。