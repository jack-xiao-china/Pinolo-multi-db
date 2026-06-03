# 随机SQL生成模式测试流程详解

## 概述

随机SQL生成模式（`GenMode`）是 Pinolo 的核心测试模式之一，通过**从已存在的数据库schema中随机生成SELECT查询**来发现逻辑bug。与DDL/DML文件模式不同，它不需要预先编写测试用例，而是自动探索查询空间。

## 架构全景图

```
┌─────────────────────────────────────────────────────────────────┐
│                    随机SQL生成模式测试流程                        │
└─────────────────────────────────────────────────────────────────┘

阶段1: 数据准备（用户负责）
  ├─ 创建数据库和表结构
  ├─ 填充测试数据（TPC-H/TPC-DS/自定义）
  └─ 确保数据多样性和正确性

阶段2: Schema发现（自动）
  ├─ DiscoverSchema() 查询 INFORMATION_SCHEMA
  ├─ 提取表名、列名、列类型、主键、可空性
  └─ 构建 SchemaInfo 元数据结构

阶段3: 查询生成（自动）
  ├─ 基于 SchemaInfo 随机生成 SELECT 查询
  ├─ 支持4种查询形状：Plain/UNION/CTE/Derived
  ├─ 随机生成表达式（列引用、常量、比较、算术、函数、CASE等）
  └─ 写入 gen_ddl.sql 和 gen_dml.sql

阶段4: 测试执行（自动）
  ├─ Stage1: SQL简化（移除聚合、窗口函数等）
  ├─ Stage2: 找变异点 + 应用变异
  ├─ 执行原始SQL和变异SQL
  └─ Oracle比较结果集

阶段5: Bug检测（自动）
  ├─ 蕴含变异：检查子集/超集关系
  ├─ 等价变异：检查结果集相等
  └─ 违反预期关系 → 报告Bug
```

## 详细流程分析

### 阶段1: 数据准备（用户负责）

**关键约束**：随机模式**不会创建表或插入数据**，必须预先准备。

```go
// task.go:384-392
if config.GenMode != "" {
    // In random generation mode, tables already exist in the database.
    // DDL file is written for record only, not executed.
    logger.Info("genMode: skip DDL execution (tables already exist)")
}
```

**数据准备方式**：
- TPC-H/TPC-DS 基准测试数据（推荐）
- 业务真实数据（脱敏后）
- 手工构造的小规模测试数据
- 其他数据生成工具（如 mysql_random_data_load）

**数据质量要求**：
- 表结构多样化（不同数据类型、主键、外键）
- 数据值多样化（覆盖边界值、NULL值、特殊字符）
- 数据量适中（太少导致空结果集，太多影响性能）

### 阶段2: Schema发现（自动）

**核心代码**：`connector/schema_mysql.go`

```go
func (conn *Connector) DiscoverSchema() (*SchemaInfo, error) {
    // 1. 查询所有表名
    SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES 
    WHERE TABLE_SCHEMA = 'dbname' AND TABLE_TYPE = 'BASE TABLE'
    
    // 2. 查询每个表的列信息
    SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, COLUMN_KEY, IS_NULLABLE
    FROM INFORMATION_SCHEMA.COLUMNS 
    WHERE TABLE_SCHEMA = 'dbname' AND TABLE_NAME = 'tablename'
    
    // 3. 构建元数据结构
    SchemaInfo {
        Tables: []TableInfo {
            Name: "lineitem",
            Columns: []ColumnInfo {
                {Name: "l_orderkey", Type: "int", IsKey: true, Nullable: false},
                {Name: "l_quantity", Type: "decimal(15,2)", IsKey: false, Nullable: false},
                // ...
            }
        }
    }
}
```

**发现的元数据**：
- 表名
- 列名、列类型（完整类型如 `varchar(20)`、`decimal(15,2)`）
- 是否主键（`PRI`）或唯一键（`UNI`）
- 是否可空（`IS_NULLABLE`）

**不获取的信息**：
- ❌ 实际数据值
- ❌ 数据分布统计
- ❌ 索引定义（除主键/唯一键）
- ❌ 外键约束

### 阶段3: 查询生成（自动）

**核心模块**：`generator/`

#### 3.1 查询形状选择

```go
// generator.go:84-106
func (g *QueryGenerator) GenerateSelect() string {
    shape := g.Rand.Intn(4)
    switch shape {
    case 0: return g.generatePlainSelect()      // SELECT ... FROM ... WHERE ...
    case 1: return g.generateUnionSelect()      // SELECT ... UNION SELECT ...
    case 2: return g.generateCTESelect()        // WITH cte AS (...) SELECT ...
    case 3: return g.generateDerivedSelect()    // SELECT * FROM (SELECT ...) AS t
    }
}
```

#### 3.2 Plain SELECT 生成流程

```go
// query_gen.go:12-72
func (g *QueryGenerator) generatePlainSelect() string {
    scope := NewScope(g.Schema, 0)
    
    // 1. 构建 FROM 子句（填充 scope 的表和列）
    fromClause, scope := g.generateFromClause(scope)
    // 例: "lineitem AS t0 INNER JOIN orders AS t1 ON t0.l_orderkey = t1.o_orderkey"
    
    // 2. 构建 SELECT 列表（1-3个列表达式）
    selectList := g.generateSelectList(scope)
    // 例: "t0.l_quantity AS ref0, t1.o_orderdate AS ref1"
    
    // 3. 构建 WHERE 子句（随机布尔表达式）
    whereClause := ""
    if g.randBool() && scope.NumColumns() > 0 {
        whereClause = g.generateBoolExpr(scope, 0)
        // 例: "(t0.l_quantity > 10 AND t1.o_orderstatus = 'O')"
    }
    
    // 4. 构建 GROUP BY + HAVING（可选）
    groupByClause := ""
    havingClause := ""
    if g.Config.EnableGroupBy && g.randBool() {
        groupByClause = g.generateGroupByClause(scope)
        // 例: "t0.l_returnflag, t1.o_orderpriority"
        if g.randBool() {
            havingClause = g.generateBoolExpr(scope, 0)
            // 例: "COUNT(*) > 5"
        }
    }
    
    // 5. 构建 ORDER BY（可选）
    orderByClause := ""
    if g.Config.EnableOrderBy && g.randBool() {
        orderByClause = g.generateOrderByClause(scope)
        // 例: "t0.l_quantity DESC, t1.o_orderdate ASC"
    }
    
    // 6. 构建 LIMIT（可选，需要 ORDER BY）
    limitClause := ""
    if g.Config.EnableLimit && orderByClause != "" && g.randBool() {
        limitClause = fmt.Sprintf("LIMIT %d", g.randInt(1, 50))
    }
    
    // 组装完整查询
    sql := "SELECT " + selectList + " FROM " + fromClause
    if whereClause != "" { sql += " WHERE " + whereClause }
    if groupByClause != "" { sql += " GROUP BY " + groupByClause }
    if havingClause != "" { sql += " HAVING " + havingClause }
    if orderByClause != "" { sql += " ORDER BY " + orderByClause }
    if limitClause != "" { sql += " " + limitClause }
    
    return sql
}
```

#### 3.3 表达式生成策略

**核心代码**：`expr_gen.go`

```go
func (g *QueryGenerator) generateExpression(scope *Scope, depth int, typeConstraint string) string {
    // 达到最大深度或没有可用列时，生成叶子节点
    if depth >= g.Config.MaxDepth || scope.NumColumns() == 0 {
        return g.generateLeaf(scope, typeConstraint)
    }
    
    // 随机选择表达式类型（20面骰子）
    choice := g.d20()
    switch {
    case choice <= 3:   return g.generateColumnRefExpr(scope, typeConstraint)   // 列引用
    case choice <= 6:   return g.generateConstantExpr(typeConstraint)           // 常量
    case choice <= 10:  return g.generateComparisonExpr(scope, depth)           // 比较
    case choice <= 14:  return g.generateBinaryArithExpr(scope, depth, typeConstraint) // 算术
    case choice <= 17:  return g.generateFunctionCallExpr(scope, depth, typeConstraint) // 函数
    case choice <= 19:  return g.generateCaseExpr(scope, depth, typeConstraint) // CASE
    default:            return g.generateInExpr(scope, depth)                   // IN
    }
}
```

**表达式类型分布**：
- 15% 列引用（`t0.col_int`）
- 15% 常量（`42`、`'str_123'`）
- 20% 比较表达式（`(a > b)`）
- 20% 算术表达式（`(a + b * c)`）
- 15% 函数调用（`COALESCE(a, b)`、`IF(cond, x, y)`）
- 10% CASE 表达式（`CASE WHEN cond THEN x ELSE y END`）
- 5% IN 表达式（`a IN (1, 2, 3)`）

**类型约束**：
- PostgreSQL/GaussDB-A 严格类型检查：生成类型兼容的表达式
- MySQL/GaussDB-M 宽松类型强制转换：允许混合类型

#### 3.4 Scope 管理

**核心概念**：Scope 跟踪当前查询上下文中可用的表和列。

```go
type Scope struct {
    Tables  []TableRef   // 当前可用的表（带别名）
    Columns []ColumnRef  // 当前可用的列（带表别名和类型信息）
    Schema  *connector.SchemaInfo
    Level   int          // 嵌套深度（子查询时递增）
}

// 构建 FROM 子句时填充 Scope
func (g *QueryGenerator) generateFromClause(scope *Scope) (string, *Scope) {
    // 选择1-3个随机表
    numTables := g.randInt(1, min(3, len(g.Schema.Tables)))
    selectedTables := pickRandomN(g.Rand, g.Schema.Tables, numTables)
    
    // 添加表引用到 Scope
    for i, table := range selectedTables {
        alias := g.nextTableAlias()  // t0, t1, t2
        scope.AddTable(table.Name, alias)
        // AddTable 会自动将该表的所有列添加到 scope.Columns
    }
    
    // 生成 JOIN 条件（如果启用）
    if g.Config.EnableJoin && len(selectedTables) > 1 {
        // INNER JOIN / LEFT JOIN / RIGHT JOIN
    }
    
    return fromClause, scope
}
```

**Scope 的作用**：
- 确保生成的表达式引用有效的表和列
- 支持类型约束（如只选择 `int` 类型的列）
- 支持子查询的嵌套作用域

### 阶段4: 测试执行（自动）

与 DDL/DML 文件模式完全相同：

```go
// task.go:450-529
for _, dmlSql := range dmlSqls {
    // 2.1 Stage1: SQL简化
    stage1Result := stage1.InitAndExec(dmlSql.Sql, conn)
    // 移除: 聚合函数、窗口函数、LEFT/RIGHT JOIN、LIMIT、不确定函数
    
    // 2.2 Stage2: 变异
    stage2Result := stage2.MutateAllAndExec(originalSql, seed, conn)
    // 找变异点 + 生成变异SQL + 执行
    
    // 2.3 Oracle: 比较
    for _, mutateUnit := range stage2Result.MutateUnits {
        if mutateUnit.IsEquivalence {
            check = oracle.CheckEquivalence(originalResult, mutatedResult)
        } else {
            check = oracle.Check(originalResult, mutatedResult, isUpper)
        }
        
        if !check {
            // Bug detected!
            bugReport.SaveBugReport(...)
        }
    }
}
```

### 阶段5: Bug检测（自动）

与 DDL/DML 文件模式完全相同：

**蕴含变异检测**：
- Upper 变异（`>` → `>=`）：期望 `original ⊆ mutated`
- Lower 变异（`>=` → `>`）：期望 `original ⊇ mutated`
- 违反包含关系 → Bug

**等价变异检测**：
- 语义等价变换（De Morgan、BETWEEN→Cmp 等）
- 期望 `original == mutated`
- 结果集不同 → Bug

## 与 DDL/DML 文件模式的对比

| 维度 | DDL/DML 文件模式 | 随机SQL生成模式 |
|------|------------------|-----------------|
| **数据准备** | DDL 文件创建表，DML 文件提供查询 | 用户预先创建表和数据 |
| **查询来源** | 手工编写或 TPC 基准 | 自动随机生成 |
| **覆盖范围** | 有限（取决于文件内容） | 理论上无限（随机探索） |
| **可重复性** | 高（固定文件） | 中（取决于 seed） |
| **适用场景** | 回归测试、特定场景 | 探索性测试、大规模测试 |
| **数据依赖** | DDL 自动创建表结构 | 依赖已有数据的质量和多样性 |

## 关键配置参数

```json
{
  "genMode": "eet_style",        // 启用随机生成模式
  "genDepth": 3,                 // 表达式最大嵌套深度
  "genQueries": 100,             // 生成的查询数量
  "genSeed": 123456,             // 随机种子（0=当前时间）
  "genJoin": true,               // 启用 JOIN
  "genSelfJoin": true,           // 启用自连接
  "genSubquery": true,           // 启用子查询
  "genUnion": true,              // 启用 UNION
  "genCTE": true,                // 启用 CTE
  "genGroupBy": true,            // 启用 GROUP BY
  "genOrderBy": true,            // 启用 ORDER BY
  "genLimit": true               // 启用 LIMIT
}
```

## 实践建议

### 1. 数据准备

**推荐**：使用 TPC-H/TPC-DS 基准测试数据
```bash
# 生成 TPC-H 数据（Scale Factor 0.01，约 10MB）
cd tpc-tools/tpch-dbgen
./dbgen -s 0.01 -f
mv *.tbl ../../tpc-data/tpch/

# 加载到 MySQL
./load_tpch_mysql.sh 127.0.0.1 3306 tpcc Taurus@123
```

**数据质量检查**：
```sql
-- 检查表是否有数据
SELECT table_name, table_rows 
FROM information_schema.tables 
WHERE table_schema = 'tpch';

-- 检查结果集分布
SELECT l_returnflag, COUNT(*) 
FROM lineitem 
GROUP BY l_returnflag;
```

### 2. 参数调优

**小规模快速测试**：
```json
{
  "genQueries": 10,
  "genDepth": 2,
  "genJoin": true,
  "genSubquery": false,
  "genUnion": false,
  "genCTE": false
}
```

**大规模深度测试**：
```json
{
  "genQueries": 1000,
  "genDepth": 5,
  "genJoin": true,
  "genSelfJoin": true,
  "genSubquery": true,
  "genUnion": true,
  "genCTE": true,
  "genGroupBy": true,
  "genOrderBy": true,
  "genLimit": true
}
```

### 3. Bug 复现

随机生成的查询会写入 `gen_dml.sql`，可以手动复现：

```bash
# 查看生成的查询
cat output/mysql/task-1/gen_dml.sql

# 手动执行特定查询
mysql -h127.0.0.1 -P3306 -utpcc -pTaurus@123 tpch < output/mysql/task-1/gen_dml.sql
```

## 总结

随机SQL生成模式的核心优势：
1. **自动化程度高**：无需手工编写测试用例
2. **覆盖范围广**：理论上可以探索无限查询空间
3. **发现意外bug**：可能发现人工难以想到的边界情况

关键约束：
1. **依赖已有数据**：必须预先准备高质量的测试数据
2. **不可完全重复**：相同的 seed 生成相同的查询，但数据变化会影响结果
3. **性能开销**：生成和执行大量查询需要时间

最佳实践：
- 使用 TPC-H/TPC-DS 基准数据作为测试基础
- 从小规模测试开始，逐步增加查询数量和复杂度
- 结合 DDL/DML 文件模式进行回归测试
