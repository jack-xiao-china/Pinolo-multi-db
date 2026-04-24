# PINOLO 用户使用指导

## 目录

- [项目概述](#项目概述)
- [支持的数据库类型](#支持的数据库类型)
- [安装与编译](#安装与编译)
- [配置说明](#配置说明)
- [使用示例](#使用示例)
- [输出结构](#输出结构)
- [变异策略说明](#变异策略说明)
- [注意事项](#注意事项)
- [常见问题](#常见问题)

---

## 项目概述

PINOLO (impomysql) 是一款基于 **Implication Oracle** 的数据库逻辑漏洞检测工具。通过 SQL 变异技术，自动检测数据库查询引擎中可能存在的逻辑缺陷。

### 核心原理

Implication Oracle 方法基于以下逻辑：
- 对于 SQL 语句 S 和其变异版本 S'
- 如果 S' 是 S 的"上变异"(结果集扩大)，则应满足 `Result(S) ⊆ Result(S')`
- 如果违反此包含关系，则存在逻辑漏洞

### 主要特性

- ✅ 支持 MySQL 及兼容数据库（MariaDB、TiDB、OceanBase）
- ✅ 支持 GaussDB/openGauss M 模式（MySQL 兼容）
- ✅ 支持 GaussDB/openGauss A 模式（Oracle 兼容）
- ✅ 17 种变异策略覆盖常见逻辑漏洞场景
- ✅ 自动化测试流程，支持批量任务执行

---

## 支持的数据库类型

### MySQL 系数据库

| 数据库 | 配置值 `dbms` | 连接协议 | 说明 |
|--------|--------------|----------|------|
| MySQL | `mysql` | MySQL | 原生支持 |
| MariaDB | `mariadb` | MySQL | 兼容 MySQL |
| TiDB | `tidb` | MySQL | 兼容 MySQL |
| OceanBase | `oceanbase` | MySQL | MySQL 模式 |

### GaussDB/openGauss M 模式

| 数据库 | 配置值 `dbms` | 连接协议 | 说明 |
|--------|--------------|----------|------|
| openGauss M | `opengauss_m` | PostgreSQL | MySQL 兼容模式 |
| GaussDB M | `gaussdb_m` | PostgreSQL | MySQL 兼容模式 |

**创建 M 模式数据库**：
```sql
CREATE DATABASE testm WITH DBCOMPATIBILITY 'M';
```

### GaussDB/openGauss A 模式

| 数据库 | 配置值 `dbms` | 连接协议 | 说明 |
|--------|--------------|----------|------|
| openGauss A | `opengauss_a` | PostgreSQL | Oracle 兼容模式 |
| GaussDB A | `gaussdb_a` | PostgreSQL | Oracle 兼容模式 |

**创建 A 模式数据库**：
```sql
CREATE DATABASE testa WITH DBCOMPATIBILITY 'A';
```

**A 模式支持的 Oracle 语法**：
- `ROWNUM` 伪列
- `(+)` 外连接语法
- `NVL()` 函数
- `DECODE()` 函数
- `SYSDATE` 函数
- `DUAL` 表
- `VARCHAR2` 数据类型

---

## 安装与编译

### 系统要求

- Go 1.20+
- GCC 编译器（用于 pg_query_go，仅 A 模式需要）
- 目标数据库连接

### Windows 编译

#### 基础编译（不含 A 模式）

```bash
cd D:\Jack.Xiao\dbtools\Pinolo-main\Pinolo-main
go build -o impomysql.exe
```

#### 完整编译（含 A 模式，需要 GCC）

1. 安装 MinGW-w64：
   ```powershell
   scoop install mingw
   ```
   或手动下载：https://github.com/niXman/mingw-builds-binaries/releases

2. 设置环境变量：
   ```bash
   set PATH=C:\mingw64\bin;%PATH%
   set CGO_ENABLED=1
   ```

3. 编译：
   ```bash
   go build -o impomysql.exe
   ```

### Linux 编译

```bash
cd /path/to/Pinolo-main
CGO_ENABLED=1 go build -o impomysql
```

### 获取 pg_query_go（A 模式必需）

从 GitHub 克隆：
```bash
git clone https://github.com/pganalyze/pg_query_go.git
```

配置 go.mod 替换路径：
```go
replace github.com/pganalyze/pg_query_go/v6 v6.0.0 => /path/to/pg_query_go
```

---

## 配置说明

### 任务配置文件 (task config)

```json
{
  "outputPath": "./output",
  "dbms": "gaussdb_a",
  "taskId": 1,
  "host": "192.168.95.195",
  "port": 8000,
  "username": "tpcc",
  "password": "Taurus@123",
  "dbname": "testa",
  "seed": 0,
  "ddlPath": "./resources/gaussdb_a_ddl.sql",
  "dmlPath": "./resources/gaussdb_a_dml.sql"
}
```

### 配置字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `outputPath` | string | 输出目录，默认 `./output` |
| `dbms` | string | 数据库类型，见上表 |
| `taskId` | int | 任务 ID，≥ 0 |
| `host` | string | 数据库主机地址 |
| `port` | int | 数据库端口 |
| `username` | string | 用户名 |
| `password` | string | 密码 |
| `dbname` | string | 数据库名 |
| `seed` | int64 | 随机种子，≤ 0 使用当前时间 |
| `ddlPath` | string | DDL 文件路径 |
| `dmlPath` | string | DML 文件路径（SELECT 语句） |

### 任务池配置文件 (taskpool config)

```json
{
  "outputPath": "./output",
  "taskPoolId": 1,
  "threadNum": 4,
  "taskConfigs": [
    "./resources/task1.json",
    "./resources/task2.json"
  ]
}
```

---

## 使用示例

### 单任务执行

```bash
# MySQL 任务
impomysql.exe task ./resources/taskconfig.json

# GaussDB M 模式任务
impomysql.exe task ./resources/gaussdb_m_task.json

# GaussDB A 模式任务
impomysql.exe task ./resources/gaussdb_a_task.json
```

### 任务池执行（并行）

```bash
impomysql.exe taskpool ./resources/taskpoolconfig.json
```

### Bug 稳定性检查

```bash
# 单任务稳定性检查（执行 10 次）
impomysql.exe ckstable task ./resources/taskconfig.json 10

# 任务池稳定性检查
impomysql.exe ckstable taskpool ./resources/taskpoolconfig.json 4 10
```

### SQL 简化

```bash
impomysql.exe sqlsim task ./resources/taskconfig.json
impomysql.exe sqlsim taskpool ./resources/taskpoolconfig.json 4
```

### 版本验证

```bash
impomysql.exe affversion task ./resources/taskconfig.json 13306 "8.0.30"
```

---

## 输出结构

### 目录结构

```
output/
├── mysql/
│   └── task-1/
│       ├── bugs/
│       │   ├── bug-0-5-FixMWhere1U.log
│       │   └── bug-0-5-FixMWhere1U.json
│       ├── result.json
│       └── task.log
├── gaussdb_m/
│   └── task-1/
│       └── ...
├── gaussdb_a/
│   └── task-1/
│       └── ...
```

### Bug 报告文件 (bug-*.json)

```json
{
  "reportTime": "2026-04-24 10:30:00",
  "bugId": 0,
  "sqlId": 5,
  "mutationName": "FixMWhere1U",
  "isUpper": true,
  "originalSql": "SELECT * FROM company WHERE age > 25",
  "originalResult": {
    "columnNames": ["id", "name", "age"],
    "rows": [["1", "Alice", "25"]]
  },
  "mutatedSql": "SELECT * FROM company WHERE 1",
  "mutatedResult": {
    "columnNames": ["id", "name", "age"],
    "rows": [["1", "Alice", "25"], ["2", "Bob", "30"]]
  }
}
```

### 任务结果文件 (result.json)

```json
{
  "startTime": "2026-04-24 10:00:00",
  "ddlSqlsNum": 7,
  "dmlSqlsNum": 19,
  "endInitTime": "2026-04-24 10:00:05",
  "stage1ErrNum": 2,
  "stage1ExecErrNum": 1,
  "stage1SkippedNum": 3,
  "stage2ErrNum": 0,
  "stage2UnitNum": 62,
  "stage2UnitErrNum": 5,
  "stage2UnitExecErrNum": 3,
  "impoBugsNum": 2,
  "saveBugErrNum": 0,
  "endTime": "2026-04-24 10:30:00"
}
```

---

## 变异策略说明

### Stage1 预处理

在变异前，对 SQL 进行预处理，移除不支持特性：

| 过滤项 | 说明 |
|--------|------|
| 聚合函数 | 移除 `SUM`, `COUNT`, `AVG`, `MAX`, `MIN` 及 `GROUP BY` |
| 窗口函数 | 移除窗口函数调用 |
| LEFT/RIGHT JOIN | 转为 INNER JOIN |
| LIMIT 子句 | 移除 LIMIT |
| 不确定函数 | 移除 `RAND()`, `UUID()` 等 |

### A 模式额外过滤

| 过滤项 | 说明 |
|--------|------|
| CONNECT BY | 层次查询，跳过 |
| PL/SQL 块 | 存储过程块，跳过 |
| DBMS_* 包 | Oracle 包调用，跳过 |

### Stage2 变异类型

#### 固定变异 (FixM)

| 变异名称 | 变异逻辑 | 方向 |
|----------|----------|------|
| `FixMDistinctU` | DISTINCT → 无 DISTINCT | 结果扩大 |
| `FixMDistinctL` | 无 DISTINCT → DISTINCT | 结果缩小 |
| `FixMUnionAllU` | UNION → UNION ALL | 结果扩大 |
| `FixMUnionAllL` | UNION ALL → UNION | 结果缩小 |
| `FixMCmpOpU` | `>` → `>=`, `<` → `<=`, `=` → `>=` | 结果扩大 |
| `FixMCmpOpL` | `>=` → `>`, `<=` → `<` | 结果缩小 |
| `FixMInNullU` | `IN(x,y)` → `IN(x,y,NULL)` | 结果扩大 |
| `FixMWhere1U` | `WHERE expr` → `WHERE 1` | 结果扩大 |
| `FixMWhere0L` | `WHERE expr` → `WHERE 0` | 结果缩小 |
| `FixMHaving1U` | `HAVING expr` → `HAVING 1` | 结果扩大 |
| `FixMHaving0L` | `HAVING expr` → `HAVING 0` | 结果缩小 |
| `FixMOn1U` | `ON expr` → `ON 1` | 结果扩大 |
| `FixMOn0L` | `ON expr` → `ON 0` | 结果缩小 |
| `FixMRmUnionAllL` | 移除 UNION ALL 后续分支 | 结果缩小 |

#### 随机变异 (RdM)

| 变异名称 | 变异逻辑 | 说明 |
|----------|----------|------|
| `RdMLikeU` | LIKE 模式扩展 | `_` → `%`, 正常字符 → `_` |
| `RdMLikeL` | LIKE 模式收缩 | `%` → `_` |
| `RdMRegExpU` | 正则模式扩展 | 添加匹配扩展符 |
| `RdMRegExpL` | 正则模式收缩 | 减少匹配范围 |

---

## 注意事项

### SQL 文件要求

1. **仅支持 SELECT 语句**
   - DML 文件中的 SQL 必须是 SELECT 或 UNION SELECT
   - INSERT/UPDATE/DELETE 语句将被过滤

2. **注释限制**
   - SQL 文件中的注释不能包含 `;` 字符
   - 否则会被错误分割为多条语句

3. **无副作用**
   - SQL 不应有副作用（如 SELECT INTO、变量赋值）

### A 模式特殊说明

1. **ROWNUM 处理**
   - ROWNUM 语法保留原样执行
   - GaussDB A 模式原生支持 ROWNUM

2. **(+) 外连接**
   - 预处理转换为 LEFT JOIN 语法
   - `WHERE t1.id = t2.id(+)` → `LEFT JOIN t2 ON t1.id = t2.id`

3. **CONNECT BY 跳过**
   - 含 CONNECT BY 的 SQL 将被过滤跳过
   - 不会参与变异测试

### 性能建议

1. **批量测试**
   - 使用 taskpool 进行并行测试
   - 设置适当的 threadNum（建议 4-8）

2. **SQL 数量**
   - 单任务 SQL 数量建议 ≤ 1000
   - 大量 SQL 可分多个任务执行

---

## 常见问题

### Q1: 编译报错 "gcc not found"

**解决**：安装 MinGW-w64 并设置 PATH：
```bash
scoop install mingw
set PATH=C:\mingw64\bin;%PATH%
```

### Q2: 编译报错 "64-bit mode not compiled in"

**原因**：使用了 32 位版本的 MinGW (MinGW.org)

**解决**：安装 64 位版本 MinGW-w64

### Q3: A 模式任务报错 "parse error"

**原因**：Oracle 特有语法不兼容

**解决**：
- 确认使用 GaussDB A 模式数据库
- 检查 SQL 是否包含 CONNECT BY（会被跳过）

### Q4: 连接 GaussDB 失败

**检查项**：
1. 数据库是否已创建并设置兼容模式
2. 网络连通性（ping 测试）
3. 用户名密码是否正确
4. 端口是否正确（A 模式通常 8000，M 模式可能不同）

### Q5: 输出目录无内容

**可能原因**：
1. 所有 SQL 被 Stage1 过滤
2. 任务执行出错（查看 task.log）

---

## 版本信息

- **项目版本**: 基于 impomysql 原版扩展
- **TiDB Parser**: v5.4.2
- **pg_query_go**: v6.0.0 (PostgreSQL 17)
- **MySQL Driver**: go-sql-driver/mysql v1.6.0
- **PostgreSQL Driver**: pgx/v5 v5.5.5

---

## 联系与反馈

如有问题或建议，请通过项目仓库提交 Issue。

---

*文档更新日期：2026-04-24*