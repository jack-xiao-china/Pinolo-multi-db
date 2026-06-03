# Pinolo vs SQLancer EET 语法覆盖对比分析

## 对比维度

| 维度 | SQLancer | Pinolo | 差距评估 |
|------|----------|--------|----------|

---

## 1. 数据类型 (Data Types)

### 1.1 数值类型

| 类型 | SQLancer | Pinolo | 评估 |
|------|:--------:|:------:|------|
| TINYINT | ✅ | ✅ (impo.yy) | 无差距 |
| SMALLINT | ✅ | ✅ (impo.yy) | 无差距 |
| MEDIUMINT | ✅ (MySQL) | ✅ (impo.yy) | 无差距 |
| INT/INTEGER | ✅ | ✅ | 无差距 |
| BIGINT | ✅ | ✅ (expr_gen) | 无差距 |
| FLOAT | ✅ | ✅ (expr_gen) | 无差距 |
| DOUBLE | ✅ | ✅ (expr_gen) | 无差距 |
| DECIMAL/NUMERIC | ✅ | ✅ (expr_gen, CAST) | 无差距 |
| BIT | ✅ (MySQL 1-64bit) | ❌ 不支持 | **遗漏** |
| INT2/INT4/INT8 | ✅ (PG别名) | ✅ (expr_gen PG) | 无差距 |
| REAL | ✅ (PG) | ❌ | **小遗漏** |
| MONEY | ✅ (PG) | ❌ | **小遗漏** |

### 1.2 字符串类型

| 类型 | SQLancer | Pinolo | 评估 |
|------|:--------:|:------:|------|
| VARCHAR | ✅ | ✅ | 无差距 |
| CHAR/BPCHAR | ✅ | ✅ | 无差距 |
| TEXT | ✅ | ✅ | 无差距 |
| LONGTEXT/MEDIUMTEXT/TINYTEXT | ✅ (MySQL) | ✅ (impo.yy) | 无差距 |
| BINARY/VARBINARY | ✅ (MySQL) | ❌ | **遗漏** |
| BLOB | ✅ (MySQL) | ❌ | **小遗漏** |
| ENUM | ✅ | ❌ | **遗漏** |
| SET | ✅ (MySQL) | ❌ | **遗漏** |

### 1.3 时间类型

| 类型 | SQLancer | Pinolo | 评估 |
|------|:--------:|:------:|------|
| DATE | ✅ | ✅ (CAST target) | 无差距 |
| TIME/TIMETZ | ✅ | ✅ (impo.yy) | TIMETZ遗漏 |
| DATETIME | ✅ | ✅ (impo.yy) | 无差距 |
| TIMESTAMP/TIMESTAMPTZ | ✅ | ✅ (CAST target) | TIMESTAMPTZ遗漏 |
| YEAR | ✅ (MySQL) | ✅ (impo.yy) | 无差距 |
| INTERVAL | ✅ (PG) | ❌ | **遗漏** |

### 1.4 特殊类型

| 类型 | SQLancer | Pinolo | 评估 |
|------|:--------:|:------:|------|
| BOOLEAN | ✅ (PG) | ✅ (expr_gen PG) | 无差距 |
| JSON/JSONB | ✅ (MySQL+PG) | ❌ | **重大遗漏** |
| UUID | ✅ (PG) | ❌ | **遗漏** |
| INET | ✅ (PG) | ❌ | **遗漏** |
| BYTEA | ✅ (PG) | ❌ | **小遗漏** |
| ARRAY | ✅ (PG) | ❌ | **遗漏** |
| RANGE (int4range等) | ✅ (PG) | ❌ | **遗漏** |

**小结**：Pinolo 在基础数值/字符串/时间类型上覆盖良好，但缺失 JSON/JSONB、ENUM、ARRAY、UUID、INET、BIT、RANGE 等现代数据库核心类型。JSON 类型在现代 DBMS 中使用广泛，缺失是最严重的。

---

## 2. 数据对象 (Data Objects)

| 对象 | SQLancer | Pinolo | 评估 |
|------|:--------:|:------:|------|
| 普通表 | ✅ | ✅ | 无差距 |
| 临时表 | ✅ | ❌ | **遗漏** |
| 分区表 | ✅ (RANGE/LIST/HASH) | ❌ | **遗漏** |
| 视图 (VIEW) | ✅ | ❌ (stage1跳过) | **需评估** |
| 物化视图 | ✅ (PG) | ❌ | **小遗漏** |
| 主键索引 | ✅ | ✅ (CREATE TABLE) | 无差距 |
| 唯一索引 | ✅ | ✅ | 无差距 |
| 部分索引 (Partial Index) | ✅ (PG) | ❌ | **遗漏** |
| 表达式索引 | ✅ (PG) | ❌ | **遗漏** |
| 外键约束 | ✅ (CASCADE/SET NULL等) | ❌ | **小遗漏** |
| CHECK约束 | ✅ | ❌ | **小遗漏** |
| 序列 (SEQUENCE) | ✅ (PG) | ❌ | **小遗漏** |
| CTE (WITH) | ✅ | ✅ (generator) | 无差距 |
| Derived Table | ✅ | ✅ (generator) | 无差距 |

**小结**：Pinolo 支持基本表对象，但缺少临时表、分区表、部分索引等。临时表和分区表的优化器行为可能与普通表不同，是潜在的 bug 来源。视图虽然 Pinolo 不支持，但这是 Implication Oracle 设计约束（视图语义需要等价推理）。

---

## 3. 表达式 (Expressions)

### 3.1 算术表达式

| 表达式 | SQLancer | Pinolo | 评估 |
|--------|:--------:|:------:|------|
| +, -, *, / | ✅ | ✅ (impo.yy) | 无差距 |
| % / MOD | ✅ | ✅ (impo.yy DIV/MOD) | 无差距 |
| 一元 +/- | ✅ | ✅ (impo.yy) | 无差距 |
| 位运算 AND/OR/XOR/NOT | ✅ | ✅ (impo.yy \|/&等) | 无差距 |
| 位移 << >> | ✅ | ❌ | **小遗漏** |
| 幂运算 ^ | ✅ (PG) | ✅ (post_num.yy) | 无差距 |

### 3.2 比较表达式

| 表达式 | SQLancer | Pinolo | 评估 |
|--------|:--------:|:------:|------|
| =, !=, <>, <, <=, >, >= | ✅ | ✅ | 无差距 |
| IS NULL / IS NOT NULL | ✅ | ✅ | 无差距 |
| IS TRUE / IS FALSE / IS NOT TRUE / IS NOT FALSE | ✅ | ✅ (visitor处理) | 无差距 |
| <=> (NULL-safe eq) | ✅ (MySQL) | ❌ | **遗漏** |
| BETWEEN / NOT BETWEEN | ✅ | ✅ (EET mutation) | 无差距 |
| LIKE / NOT LIKE | ✅ | ✅ (RdMLikeU/L) | 无差距 |
| REGEXP / NOT REGEXP | ✅ (MySQL) | ✅ (RdMRegExpU/L) | 无差距 |
| POSIX正则 ~ / !~ / ~* / !~* | ✅ (PG) | ❌ | **遗漏** |
| SIMILAR TO | ✅ (PG) | ❌ | **遗漏** |
| IN (列表) | ✅ | ✅ (FixMInNullU) | 无差距 |
| IN (子查询) | ✅ | ✅ (FixMInToExists) | 无差距 |
| ANY / ALL / SOME 子查询 | ✅ | ❌ | **遗漏** |

### 3.3 逻辑表达式

| 表达式 | SQLancer | Pinolo | 评估 |
|--------|:--------:|:------:|------|
| AND / OR | ✅ | ✅ (DeMorgan EET) | 无差距 |
| NOT | ✅ | ✅ (visitor) | 无差距 |
| XOR | ✅ (MySQL) | ✅ (visitor跳过) | **需关注** |

### 3.4 CASE/控制流

| 表达式 | SQLancer | Pinolo | 评估 |
|--------|:--------:|:------:|------|
| CASE WHEN | ✅ | ✅ (FixMCaseTrueU/L/Rand) | 无差距 |
| IF(cond,t,f) | ✅ (MySQL) | ✅ (FixMIfToCase M-mode) | 无差距 |
| IFNULL(a,b) | ✅ (MySQL) | ❌ 无等价变换 | **遗漏** |

### 3.5 字符串表达式

| 表达式 | SQLancer | Pinolo | 评估 |
|--------|:--------:|:------:|------|
| CONCAT | ✅ | ✅ (FixMConcatToPipe M-mode) | 无差距 |
| CONCAT_WS | ✅ | ❌ 无等价变换 | **遗漏** |
| || 运算符 | ✅ (PG) | ❌ (TiDB parser限制) | **已知限制** |
| LENGTH/CHAR_LENGTH | ✅ | ✅ (impo.yy) | 无差距 |
| UPPER/LOWER | ✅ | ✅ (impo.yy) | 无差距 |
| TRIM/LTRIM/RTRIM | ✅ | ✅ (impo.yy) | 无差距 |
| LEFT/RIGHT/SUBSTRING | ✅ | ✅ (impo.yy) | 无差距 |
| REPLACE | ✅ | ✅ (impo.yy) | 无差距 |
| REVERSE/REPEAT/SPACE | ✅ | ✅ (impo.yy) | 无差距 |
| LPAD/RPAD | ✅ | ✅ (impo.yy) | 无差距 |
| ASCII | ✅ | ✅ (impo.yy) | 无差距 |
| LOCATE/INSTR | ✅ | ✅ (impo.yy) | 无差距 |
| HEX/UNHEX | ✅ | ❌ | **小遗漏** |

### 3.6 JSON 表达式

| 表达式 | SQLancer | Pinolo | 评估 |
|--------|:--------:|:------:|------|
| JSON_TYPE | ✅ | ❌ | **重大遗漏** |
| JSON_VALID | ✅ | ❌ | **重大遗漏** |
| JSON_EXTRACT/-> /->> | ✅ | ❌ | **重大遗漏** |
| JSON_CONTAINS/@> | ✅ (PG) | ❌ | **重大遗漏** |
| JSON_ARRAY | ✅ | ❌ | **重大遗漏** |
| JSON_OBJECT | ✅ | ❌ | **重大遗漏** |
| JSON_REMOVE | ✅ | ❌ | **重大遗漏** |
| JSON_KEYS | ✅ | ❌ | **重大遗漏** |

### 3.7 时间表达式

| 表达式 | SQLancer | Pinolo | 评估 |
|--------|:--------:|:------:|------|
| YEAR/MONTH/DAY/HOUR/MINUTE/SECOND | ✅ | ✅ (impo.yy) | 无差距 |
| DAYOFWEEK/DAYOFMONTH/DAYOFYEAR/WEEK/QUARTER | ✅ | ✅ (impo.yy) | 无差距 |
| DATEDIFF/LAST_DAY/TO_DAYS/FROM_DAYS | ✅ | ✅ (impo.yy) | 无差距 |
| DATE_ADD/DATE_SUB + INTERVAL | ✅ | ✅ (impo.yy) | 无差距 |
| DATE_TRUNC | ✅ (PG) | ❌ | **遗漏** |
| EXTRACT/DATE_PART | ✅ (PG) | ❌ | **小遗漏** |
| MAKE_INTERVAL | ✅ (PG) | ❌ | **小遗漏** |

**小结**：Pinolo 在基础比较/逻辑/算术表达式上覆盖全面，但 JSON 表达式完全缺失（重大遗漏），ANY/ALL/SOME 子查询、NULL-safe 等值(`<=>`)、PG POSIX正则也缺失。IFNULL 缺少等价变换规则。

---

## 4. 函数等价变换 (Function Equivalence Rules)

### 4.1 已实现的等价变换

| 变换规则 | SQLancer | Pinolo | 一致性 |
|----------|:--------:|:------:|--------|
| DeMorgan AND/OR | ✅ | ✅ | ✅ 一致 |
| BETWEEN → 比较 | ✅ | ✅ | ✅ 一致 |
| COALESCE → CASE | ✅ | ✅ | ✅ 一致 |
| NULLIF → CASE | ✅ | ✅ | ✅ 一致 |
| EXISTS → IN | ✅ | ✅ | ✅ 一致 |
| IN → EXISTS | ✅ | ✅ | ✅ 一致 |
| IF → CASE | ✅ | ✅ (M-mode) | ✅ 一致 |
| Tautology wrapping (AND TRUE) | ✅ | ✅ | ✅ 一致 |
| Contradiction wrapping (OR FALSE) | ✅ | ✅ | ✅ 一致 |
| CASE TRUE/FALSE/RAND wrapping | ✅ | ✅ | ✅ 一致 |

### 4.2 SQLancer 有但 Pinolo 缺失的等价变换

| 变换规则 | SQLancer | Pinolo | 评估 |
|----------|:--------:|:------:|------|
| INTERSECT → EXISTS | ✅ (PG) | ❌ | **遗漏** (PG特有) |
| EXCEPT → NOT EXISTS | ✅ (PG) | ❌ | **遗漏** (PG特有) |
| IFNULL → COALESCE | ✅ | ❌ | **遗漏** (MySQL) |
| LEAST/GREATEST → CASE | ✅ | ❌ | **遗漏** |
| IS TRUE → = TRUE | ✅ | ❌ | **小遗漏** |
| NOT ISNULL(x) → x IS NOT NULL | ✅ | ❌ | **小遗漏** |
| CAST 类型等价 | ✅ | ❌ | **遗漏** |

### 4.3 Pinolo 独有的变换

| 变换规则 | Pinolo | 说明 |
|----------|:------:|------|
| CONCAT → || | ✅ (M-mode) | SQLancer 未对 CONCAT 做等价变换 |
| Implication oracle 变换族 | ✅ | Pinolo 独有的 FixMWhere1U/0L 等 implication 变换，SQLancer 无此机制 |

**小结**：核心等价变换（DeMorgan、BETWEEN、COALESCE、NULLIF、EXISTS/IN）两家一致。Pinolo 缺失的主要是：INTERSECT/EXCEPT→EXISTS 的 PG 特有变换、IFNULL→COALESCE、LEAST/GREATEST→CASE。Implication oracle 变换族是 Pinolo 独有优势。

---

## 5. DQL 方式 (Query Patterns)

### 5.1 SELECT 查询形态

| 查询形态 | SQLancer | Pinolo | 评估 |
|----------|:--------:|:------:|------|
| Plain SELECT | ✅ | ✅ | 无差距 |
| UNION / UNION ALL | ✅ | ✅ (FixMUnionAllU/L) | 无差距 |
| INTERSECT | ✅ (PG) | ❌ | **遗漏** |
| EXCEPT | ✅ (PG) | ❌ | **遗漏** |
| WITH (CTE) | ✅ | ✅ (generator) | 无差距 |
| Derived Table (子查询FROM) | ✅ | ✅ (generator) | 无差距 |
| JOIN (INNER) | ✅ | ✅ | 无差距 |
| LEFT/RIGHT JOIN | ✅ | ✅ (stage1转为INNER) | 设计差异 |
| CROSS JOIN | ✅ | ✅ (post_num.yy) | 无差距 |
| NATURAL JOIN | ✅ | ✅ (post_num.yy) | 无差距 |
| STRAIGHT JOIN | ✅ (MySQL) | ❌ | **小遗漏** |
| SELF JOIN | ✅ | ❌ | **遗漏** |
| GROUP BY | ✅ (可禁用) | ✅ (stage1移除) | 设计差异 |
| HAVING | ✅ | ✅ (FixMHaving) | 无差距 |
| ORDER BY | ✅ | ❌ (stage1移除) | 设计差异 |
| LIMIT | ✅ | ❌ (stage1移除) | 设计差异 |
| DISTINCT | ✅ | ✅ (FixMDistinctU/L) | 无差距 |
| Aggregate functions | ✅ (COUNT/SUM/AVG/MIN/MAX) | ❌ (stage1移除) | **设计差异** |
| Window functions | ✅ (PG) | ❌ (stage1移除) | **设计差异** |
| FOR UPDATE/SHARE | ✅ (PG) | ❌ | **小遗漏** |

**小结**：Pinolo 通过 stage1 移除 GROUP BY/LIMIT/聚合函数等，这是 Implication Oracle 的设计约束（结果集包含关系需要确定性语义）。但 INTERSECT/EXCEPT 是 PG 核心集合操作，且有等价变换规则可做 EET 测试，属于重要遗漏。SELF JOIN 也没有覆盖。

---

## 6. 综合差距评估与增强建议

### 6.1 高优先级增强（重大遗漏，bug 发现潜力大）

| # | 遗漏项 | 影响范围 | 增强建议 | 实现难度 |
|---|--------|---------|----------|---------|
| **H1** | **JSON/JSONB 类型与函数** | MySQL 8.0+, PG 9.4+, GaussDB | 新增 JSON 类型生成 + JSON 函数等价变换规则（JSON_EXTRACT→->, JSON_CONTAINS等价） | **高** (需AST支持) |
| **H2** | **INTERSECT/EXCEPT → EXISTS 变换** | PG, GaussDB-A | 新增 PG EET 变换：INTERSECT→EXISTS, EXCEPT→NOT EXISTS | **中** (PG parser已有) |
| **H3** | **IFNULL → COALESCE 变换** | MySQL, GaussDB-M | 新增：IFNULL(a,b) → COALESCE(a,b) 等价 | **低** (简单函数替换) |
| **H4** | **ANY/ALL/SOME 子查询** | MySQL, PG, GaussDB | 新增 visitor 识别 ANY/ALL 子查询，配合等价变换 | **中** |
| **H5** | **ENUM 类型** | MySQL, GaussDB-M | 新增 ENUM 类型列生成，ENUM 值的等价比较 | **低** |

### 6.2 中优先级增强（有 bug 发现潜力）

| # | 遗漏项 | 影响范围 | 增强建议 | 实现难度 |
|---|--------|---------|----------|---------|
| **M1** | **LEAST/GREATEST → CASE** | MySQL, PG | 新增等价变换：LEAST(a,b)→CASE WHEN a<=b THEN a ELSE b END | **低** |
| **M2** | **NULL-safe 等值 <=>** | MySQL | 新增 visitor 处理 <=> 运算符，等价变换 <=> → IS NOT DISTINCT FROM | **中** |
| **M3** | **PG POSIX 正则 ~ / !~** | PG | 新增 PG visitor 识别 POSIX 正则表达式节点 | **中** |
| **M4** | **临时表** | MySQL, PG | 新增临时表生成和测试（优化器可能不同路径） | **中** |
| **M5** | **SELF JOIN** | 所有DBMS | 新增 self join 查询形态生成 | **低** |
| **M6** | **ARRAY 类型** | PG, GaussDB-A | 新增 ARRAY 列生成和数组运算等价 | **高** |
| **M7** | **UUID 类型** | PG, GaussDB-A | 新增 UUID 列和运算生成 | **低** |
| **M8** | **IS TRUE → = TRUE 等价** | MySQL, PG | 新增：expr IS TRUE → expr = TRUE 等价变换 | **低** |
| **M9** | **NOT ISNULL(x) → x IS NOT NULL** | MySQL | 新增：NOT ISNULL(x) → x IS NOT NULL 等价 | **低** |

### 6.3 低优先级增强（边际效益较小）

| # | 遗漏项 | 影响范围 | 增强建议 | 实现难度 |
|---|--------|---------|----------|---------|
| **L1** | BIT 类型 | MySQL | 新增 BIT 列生成 | **低** |
| **L2** | BINARY/VARBINARY | MySQL | 新增二进制字符串类型 | **低** |
| **L3** | INTERVAL 类型 | PG | 新增时间间隔运算 | **中** |
| **L4** | RANGE 类型 | PG | 新增范围类型运算 | **高** |
| **L5** | 分区表 | MySQL, PG | 新增分区表 DDL 生成 | **中** |
| **L6** | 部分索引 | PG | 新增 CREATE INDEX WHERE 条件 | **低** |
| **L7** | HEX/UNHEX 函数 | MySQL | 新增字符串函数 | **低** |
| **L8** | FOR UPDATE/SHARE | PG | 新增锁子句 | **低** |

---

## 7. Pinolo 独有优势（SQLancer 无此机制）

| 优势 | 说明 |
|------|------|
| **Implication Oracle 变换族** | FixMWhere1U/0L, FixMHaving1U/0L, FixMOn1U/0L 等 "替换为常量" 变换，检测结果集包含关系违反。SQLancer 无此机制 |
| **DISTINCT 变换** | FixMDistinctU/L 检测 DISTINCT 与非 DISTINCT 的等价性违反 |
| **UNION/UNION ALL 变换** | FixMUnionAllU/L 检测 UNION 与 UNION ALL 的等价性违反 |
| **LIKE/REGEXP 随机变异** | RdMLikeU/L, RdMRegExpU/L 对模式字符串的随机变形 |
| **IN + NULL 变换** | FixMInNullU 检测 IN 列表含 NULL 时的语义差异 |
| **Stage1 预处理** | 自动移除不确定因素（聚合、窗口函数、LEFT JOIN、LIMIT），保证 oracle 比较的确定性 |

这些是 Pinolo 的核心竞争力，不应丢弃。

---

## 8. 增强实施路线图

### Phase 1：低成本高收益（1-2天）

- **H3**: IFNULL → COALESCE 等价变换（MySQL/GaussDB-M）
- **M1**: LEAST/GREATEST → CASE 等价变换
- **M8**: IS TRUE → = TRUE 等价变换
- **M9**: NOT ISNULL → IS NOT NULL 等价变换
- **M5**: SELF JOIN 查询形态

### Phase 2：中等投入（3-5天）

- **H2**: INTERSECT/EXCEPT → EXISTS 变换（PG EET）
- **H4**: ANY/ALL/SOME 子查询 visitor + 等价变换
- **M2**: NULL-safe 等值 <=> visitor
- **M3**: PG POSIX 正则 ~ / !~ visitor
- **H5**: ENUM 类型列生成

### Phase 3：高投入高潜力（5-10天）

- **H1**: JSON/JSONB 类型与函数等价变换体系
- **M6**: ARRAY 类型与运算等价
- **M4**: 临时表测试

### 按需推进

- **L1-L8**: 低优先级，根据实际 bug 发现率决定是否投入

---

## 9. 设计约束说明

Pinolo 的以下"缺失"是 Implication Oracle 设计约束导致的，不应简单对照 SQLancer 补齐：

| 约束 | 说明 | SQLancer 处理 |
|------|------|---------------|
| 聚合函数移除 | Implication Oracle 需确定性结果集包含关系 | SQLancer 用 TLP oracle 分离聚合 |
| GROUP BY 移除 | 同上 | SQLancer EET 也禁用 GROUP BY |
| LEFT/RIGHT JOIN 转为 INNER | 结果集包含关系不适用于外连接 | SQLancer 保留外连接 |
| LIMIT 移除 | LIMIT 导致结果集截断，包含关系不成立 | SQLancer EET 也有限制 |
| ORDER BY 移除 | 排序不影响结果集内容 | SQLancer EET 保留 ORDER BY |

**建议**：LEFT/RIGHT JOIN 的保留值得评估。SQLancer EET 允许外连接，说明等价变换在外连接 WHERE 条件上仍然成立（WHERE 过滤的是连接后结果）。Pinolo 可考虑在 stage1 中保留外连接但标记 ON 条件变异为 implication 模式（ON 1 → 笛卡尔积 ⊇ 外连接结果）。