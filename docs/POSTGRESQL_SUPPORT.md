# PostgreSQL 原生数据库支持特性设计

## 文档信息

- **版本**: v1.0
- **创建日期**: 2026-04-24
- **作者**: PINOLO 开发团队
- **状态**: 设计阶段

---

## 一、特性概述

### 1.1 背景

PINOLO 是一款基于 Implication Oracle 方法的数据库逻辑漏洞检测工具。当前已支持：

| 数据库类型 | 协议 | Parser | 状态 |
|------------|------|--------|------|
| MySQL | MySQL | TiDB Parser | ✅ 已支持 |
| MariaDB | MySQL | TiDB Parser | ✅ 已支持 |
| TiDB | MySQL | TiDB Parser | ✅ 已支持 |
| OceanBase | MySQL | TiDB Parser | ✅ 已支持 |
| GaussDB/openGauss M | PostgreSQL | TiDB Parser | ✅ 已支持 |
| GaussDB/openGauss A | PostgreSQL | pg_query_go | ✅ 已支持 |
| **PostgreSQL** | PostgreSQL | pg_query_go | 🔄 待开发 |

### 1.2 目标

为 PINOLO 添加原生 PostgreSQL 数据库支持，实现：
- 使用 PostgreSQL 社区官方 Go 驱动（pgx v5）
- 完整支持 PostgreSQL 语法特性
- 保持与现有架构的一致性

---

## 二、技术架构

### 2.1 核心依赖

```
┌─────────────────────────────────────────────────────────────────┐
│                      PostgreSQL Support Stack                     │
├─────────────────────────────────────────────────────────────────┤
│  pgx/v5 v5.5.5        ← PostgreSQL 官方 Go 驱动（连接池）          │
│  pg_query_go/v6 v6.0.0 ← PostgreSQL Parser（PostgreSQL 17 AST）  │
│  golang.org/x/crypto  ← PostgreSQL 认证加密                       │
├─────────────────────────────────────────────────────────────────┤
│  已在 go.mod 中引入，无需新增依赖                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        main.go (CLI)                              │
│  命令: impomysql task ./resources/postgresql_task.json           │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                 task/postgresql_task.go                          │
│  RunTaskPostgreSQL(config, logger)                               │
└────────────────────────────────┬────────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
        ▼                        ▼                        ▼
┌───────────────┐  ┌───────────────────┐  ┌───────────────────────┐
│ connector/    │  │ mutation/stage1/  │  │ mutation/stage2/      │
│ postgresql.go │  │ stage1_pg.go      │  │ pg_mutatevisitor.go   │
│               │  │                   │  │ pg_mutate_functions.go │
│ pgx/v5 连接池 │  │ pg_query预处理    │  │ pg_query AST变异      │
└───────────────┘  └───────────────────┘  └───────────────────────┘
        │                        │                        │
        ▼                        ▼                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                    parser/pgquery_adapter.go                      │
│  PgQueryParser.Parse(sql) → ASTNode                              │
│  PgQueryParser.Restore(node) → SQL                               │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                 github.com/pganalyze/pg_query_go/v6               │
│  PostgreSQL 17 官方 Parser Go 绑定                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 三、组件设计

### 3.1 PostgreSQL Connector

**文件**: `connector/postgresql.go`

```go
package connector

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/pkg/errors"
)

// PostgreSQLConnector: PostgreSQL 原生连接器
// 使用 pgx/v5 pgxpool 连接池
type PostgreSQLConnector struct {
    Host     string
    Port     int
    Username string
    Password string
    DbName   string
    pool     *pgxpool.Pool
}

// NewPostgreSQLConnector: 创建连接器
func NewPostgreSQLConnector(host, port, username, password, dbname string) (*PostgreSQLConnector, error) {
    connString := fmt.Sprintf(
        "postgres://%s:%s@%s:%d/%s?sslmode=disable",
        username, password, host, port, dbname)
    
    pool, err := pgxpool.New(context.Background(), connString)
    if err != nil {
        return nil, errors.Wrap(err, "pool creation error")
    }
    
    if err := pool.Ping(context.Background()); err != nil {
        return nil, errors.Wrap(err, "ping error")
    }
    
    return &PostgreSQLConnector{
        Host: host, Port: port, Username: username,
        Password: password, DbName: dbname, pool: pool,
    }, nil
}

// ExecSQL: 实现 SQLExecutor 接口
func (pg *PostgreSQLConnector) ExecSQL(sql string) *Result {
    rows, err := pg.pool.Query(context.Background(), sql)
    if err != nil {
        return &Result{Err: errors.Wrap(err, "query error")}
    }
    defer rows.Close()
    
    // 解析结果集...
    return result
}

// Close: 关闭连接池
func (pg *PostgreSQLConnector) Close() {
    pg.pool.Close()
}
```

**关键特性**:
- 使用 `pgxpool` 连接池，提升并发性能
- 支持 PostgreSQL 认证（SCRAM-SHA-256/md5）
- 实现 `SQLExecutor` 接口，与现有架构兼容

### 3.2 PostgreSQL Stage1

**文件**: `mutation/stage1/stage1_pg.go`

```go
package stage1

import (
    "github.com/pganalyze/pg_query_go/v6"
    "github.com/qaqcatz/impomysql/parser"
)

// InitForPostgreSQL: PostgreSQL 专用 Stage1 预处理
func InitForPostgreSQL(sql string) *InitResult {
    result := &InitResult{}
    
    // 1. Parse with pg_query_go
    ast, err := parser.NewPgQueryParserForPostgreSQL().Parse(sql)
    if err != nil {
        result.Err = err
        return result
    }
    
    // 2. Remove unsupported features:
    //    - Aggregate functions (SUM, COUNT, AVG, MAX, MIN)
    //    - Window functions
    //    - Complex JOIN (preserve INNER JOIN)
    //    - LIMIT/OFFSET (保留，PostgreSQL 支持)
    
    // 3. Restore to SQL
    initSql, err := parser.Restore(ast)
    if err != nil {
        result.Err = err
        return result
    }
    
    result.InitSql = initSql
    return result
}
```

### 3.3 PostgreSQL Stage2

**文件**: `mutation/stage2/pg_mutatevisitor.go`

```go
package stage2

import (
    "github.com/pganalyze/pg_query_go/v6"
    "github.com/qaqcatz/impomysql/parser"
)

// PgMutateVisitor: PostgreSQL AST 变异遍历器
type PgMutateVisitor struct {
    Root      *pgquery.ParseResult
    Candidates map[string][]*parser.MutationCandidate
}

// FindCandidates: 寻找变异候选点
func (v *PgMutateVisitor) FindCandidates() map[string][]*parser.MutationCandidate {
    v.Candidates = make(map[string][]*parser.MutationCandidate)
    
    // 遍历 PostgreSQL AST (pg_query.ParseResult)
    // 寻找以下节点类型:
    // - SelectStmt.WhereClause → FixMWhere1U, FixMWhere0L
    // - SelectStmt.HavingClause → FixMHaving1U, FixMHaving0L
    // - JoinExpr.Quals → FixMOn1U, FixMOn0L
    // - SelectStmt.DistinctClause → FixMDistinctU, FixMDistinctL
    // - SetOperationStmt → FixMUnionAllU, FixMUnionAllL
    // - A_Expr (comparison ops) → FixMCmpOpU, FixMCmpOpL
    // - InClause → FixMInNullU
    // - LikeExpr (~~ operator) → RdMLikePgU, RdMLikePgL
    // - RegexpExpr (~ operator) → RdMRegExpPgU, RdMRegExpPgL
    
    return v.Candidates
}
```

---

## 四、变异策略映射

### 4.1 可复用策略

| 变异名称 | PostgreSQL AST 节点 | 实现方式 |
|----------|---------------------|----------|
| FixMWhere1U | `SelectStmt.whereClause` | `WHERE expr` → `WHERE TRUE` |
| FixMWhere0L | `SelectStmt.whereClause` | `WHERE expr` → `WHERE FALSE` |
| FixMHaving1U | `SelectStmt.havingClause` | `HAVING expr` → `HAVING TRUE` |
| FixMHaving0L | `SelectStmt.havingClause` | `HAVING expr` → `HAVING FALSE` |
| FixMOn1U | `JoinExpr.quals` | `ON expr` → `ON TRUE` |
| FixMOn0L | `JoinExpr.quals` | `ON expr` → `ON FALSE` |
| FixMDistinctU | `SelectStmt.distinctClause` | `DISTINCT` → 移除 |
| FixMDistinctL | `SelectStmt.distinctClause` | 无 `DISTINCT` → 添加 |
| FixMUnionAllU | `SetOperationStmt.op` | `UNION` → `UNION ALL` |
| FixMUnionAllL | `SetOperationStmt.op` | `UNION ALL` → `UNION` |
| FixMCmpOpU | `A_Expr` (comparison) | `>` → `>=`, `<` → `<=` |
| FixMCmpOpL | `A_Expr` (comparison) | `>=` → `>`, `<=` → `<` |
| FixMInNullU | `InClause` | `IN(a,b)` → `IN(a,b,NULL)` |

### 4.2 PostgreSQL 特有策略

| 变异名称 | PostgreSQL AST 节点 | 实现方式 |
|----------|---------------------|----------|
| FixMDistinctOnU | `SelectStmt.distinctClause` | `DISTINCT ON(col)` → `DISTINCT` |
| FixMDistinctOnL | `SelectStmt.distinctClause` | `DISTINCT` → `DISTINCT ON(col)` |
| RdMLikePgU | `A_Expr` (~~ operator) | LIKE 模式扩展 |
| RdMLikePgL | `A_Expr` (~~ operator) | LIKE 模式收缩 |
| RdMRegExpPgU | `A_Expr` (~ operator) | 正则扩展 |
| RdMRegExpPgL | `A_Expr` (~ operator) | 正则收缩 |
| RdMILikeU | `A_Expr` (~~* operator) | ILIKE 模式扩展 |

---

## 五、配置示例

### 5.1 任务配置文件

**文件**: `resources/postgresql_task.json`

```json
{
  "outputPath": "./output",
  "dbms": "postgresql",
  "taskId": 1,
  "host": "localhost",
  "port": 5432,
  "username": "your_username",
  "password": "your_password",
  "dbname": "testdb",
  "seed": 0,
  "ddlPath": "./resources/postgresql_ddl.sql",
  "dmlPath": "./resources/postgresql_dml.sql"
}
```

### 5.2 DDL 示例

**文件**: `resources/postgresql_ddl.sql`

```sql
DROP TABLE IF EXISTS company;
CREATE TABLE company (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    age INTEGER,
    salary DECIMAL(10, 2)
);

CREATE INDEX idx_company_age ON company(age);
```

### 5.3 DML 示例

**文件**: `resources/postgresql_dml.sql`

```sql
SELECT * FROM company WHERE age > 25;
SELECT name, salary FROM company WHERE salary >= 50000 ORDER BY salary DESC;
SELECT DISTINCT age FROM company;
SELECT c1.name, c2.name FROM company c1 JOIN company c2 ON c1.age = c2.age;
```

---

## 六、开发计划

### 6.1 阶段与工时

| 阶段 | 任务 | 工时估算 | 前置依赖 |
|------|------|----------|----------|
| **P1: Connector** | 创建 `connector/postgresql.go` | 8-12h | - |
| **P2: Parser 扩展** | 扩展 `parser/pgquery_adapter.go` | 4-8h | - |
| **P3: Stage1** | 创建 `mutation/stage1/stage1_pg.go` | 16-24h | P2 |
| **P4: Stage2** | 创建 `pg_mutatevisitor.go` + `pg_mutate_functions.go` | 72-108h | P2 |
| **P5: Task** | 创建 `task/postgresql_task.go` | 16-24h | P1-P4 |
| **P6: 测试** | 单元测试 + 集成测试 | 24-40h | P5 |
| **P7: 文档** | 更新 USER_GUIDE.md | 4-6h | P6 |
| **总计** | | **144-182h** | |

### 6.2 开发周期

- **预计周期**: 6 周（包含测试验证）
- **人员配置**: 1-2 名开发人员

---

## 七、验证计划

### 7.1 单元测试

```go
// connector/postgresql_test.go
func TestPostgreSQLConnectorExecSQL(t *testing.T) {
    pg, err := NewPostgreSQLConnector("localhost", 5432, "tpcc", "pass", "testdb")
    assert.NoError(t, err)
    
    result := pg.ExecSQL("SELECT 1+2")
    assert.NoError(t, result.Err)
    assert.Equal(t, "3", result.Rows[0][0])
}

// mutation/stage2/pg_mutatevisitor_test.go
func TestPgMutateVisitorFindCandidates(t *testing.T) {
    sql := "SELECT * FROM t WHERE a > 1"
    candidates := FindCandidatesForPostgreSQL(sql)
    assert.Contains(t, candidates, "FixMWhere1U")
    assert.Contains(t, candidates, "FixMCmpOpU")
}
```

### 7.2 集成测试

```bash
# 启动 PostgreSQL 测试环境
docker run -d --name pgtest -p 5432:5432 \
    -e POSTGRES_USER=your_username \
    -e POSTGRES_PASSWORD=your_password \
    -e POSTGRES_DB=testdb \
    postgres:17

# 运行集成测试
./impomysql task ./resources/postgresql_task.json

# 检查结果
cat output/postgresql/task-1/result.json
```

---

## 八、风险评估

| 风险项 | 风险等级 | 影响 | 应对措施 |
|--------|----------|------|----------|
| pg_query AST 结构复杂 | 中 | 变异实现难度增加 | 参考官方文档，逐步适配 |
| PostgreSQL 语义与 MySQL 差异 | 中 | Bug 判断逻辑差异 | 详细语义分析，调整 IsUpper |
| pgx 连接池配置 | 低 | 性能问题 | 使用默认配置，必要时调整 |
| 测试环境搭建 | 低 | 验证受阻 | 使用 Docker 快速搭建 |

---

## 九、附录

### 9.1 pg_query AST 参考

- [pg_query_go GitHub](https://github.com/pganalyze/pg_query_go)
- [PostgreSQL Parser 文档](https://www.postgresql.org/docs/current/parser.html)

### 9.2 pgx 驱动参考

- [pgx GitHub](https://github.com/jackc/pgx)
- [pgx 文档](https://github.com/jackc/pgx/wiki)

---

## 十、版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| v1.0 | 2026-04-24 | 初版设计文档 |

---

*文档维护: PINOLO 开发团队*