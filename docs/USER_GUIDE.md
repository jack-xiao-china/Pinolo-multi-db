# PINOLO 使用指导

## 1 项目概述

PINOLO (impomysql) 基于 **Implication Oracle** 方法检测数据库逻辑漏洞。核心原理：对 SELECT 语句 S 生成变异版本 S'，若 S' 为上变异（结果集扩大）则 `Result(S) ⊆ Result(S')` 应成立；若违反此包含关系则存在逻辑漏洞。

## 2 支持的数据库

| dbms 值 | 数据库 | Parser | 驱动 | 需要的数据库准备 |
|----------|--------|--------|------|------------------|
| `mysql` | MySQL | TiDB v5.4.2 | go-sql-driver/mysql | 无，自动创建 |
| `mariadb` | MariaDB | TiDB v5.4.2 | go-sql-driver/mysql | 无，自动创建 |
| `tidb` | TiDB | TiDB v5.4.2 | go-sql-driver/mysql | 无，自动创建 |
| `oceanbase` | OceanBase | TiDB v5.4.2 | go-sql-driver/mysql | 无，自动创建 |
| `postgresql` | PostgreSQL | pg_query_go v6 | pgx/v5 | 需手动创建数据库 |
| `opengauss_m` | openGauss M | TiDB v5.4.2 | openGauss-connector-go-pq | `CREATE DATABASE testm WITH DBCOMPATIBILITY 'M'` |
| `gaussdb_m` | GaussDB M | TiDB v5.4.2 | openGauss-connector-go-pq | `CREATE DATABASE testm WITH DBCOMPATIBILITY 'M'` |
| `opengauss_a` | openGauss A | pg_query_go v6 | openGauss-connector-go-pq | `CREATE DATABASE testa WITH DBCOMPATIBILITY 'A'` |
| `gaussdb_a` | GaussDB A | pg_query_go v6 | openGauss-connector-go-pq | `CREATE DATABASE testa WITH DBCOMPATIBILITY 'A'` |

## 3 编译

### 3.1 前提条件

- Go 1.20+
- **GCC 编译器**（pg_query_go 含 C 代码，必须 CGO 编译）
- 运行 setup_third_party.sh 初始化本地依赖（pg_query_go + openGauss-connector-go-pq）

### 3.2 初始化第三方依赖

```bash
# 首次编译前，运行初始化脚本下载 pg_query_go 和 openGauss-connector-go-pq
bash setup_third_party.sh
```

脚本会将依赖克隆到 `third_party/` 目录（go.mod replace 已指向此路径）。如脚本无法执行，可手动克隆：

```bash
mkdir -p third_party
git clone --depth 1 https://github.com/pganalyze/pg_query_go.git third_party/pg_query_go
git clone --depth 1 https://gitee.com/opengauss/openGauss-connector-go-pq.git third_party/openGauss-connector-go-pq
```

### 3.3 Linux 编译

```bash
# 安装 GCC（如未安装）
# Ubuntu/Debian: sudo apt install gcc libc6-dev
# CentOS/RHEL:   sudo yum install gcc glibc-devel

cd /path/to/Pinolo-main
bash setup_third_party.sh  # 首次编译
CGO_ENABLED=1 go build -o impomysql
```

### 3.4 Windows 编译

```bash
# 安装 MinGW-w64（64位），确保 gcc 在 PATH 中
# 方法1: scoop install mingw
# 方法2: 下载 https://github.com/niXman/mingw-builds-binaries/releases

set PATH=C:\mingw64\bin;%PATH%
set CGO_ENABLED=1
go build -o impomysql.exe
```

### 3.4 编译问题排查

| 错误信息 | 原因 | 解决 |
|----------|------|------|
| `C compiler "gcc" not found` | GCC 不在 PATH 中 | 安装 MinGW-w64 / gcc，加入 PATH |
| `64-bit mode not compiled in` | 使用了 32 位 MinGW | 安装 MinGW-w64（64 位版本） |
| `undefined: pgquery.Parse` | CGO_ENABLED=0 | 设置 `CGO_ENABLED=1` |
| `cannot find package pg_query_go` | go.mod replace 路径错误 | 检查 `third_party/` 目录是否存在 |

### 3.5 Linux 一键部署脚本

```bash
#!/bin/bash
# install_deps.sh - Pinolo Linux 部署
set -e

# 安装 Go（如未安装）
if ! command -v go &>/dev/null; then
    wget -q https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
fi

# 安装 GCC
sudo apt install -y gcc libc6-dev || sudo yum install -y gcc glibc-devel

# 初始化依赖 + 编译
cd /path/to/Pinolo-main
bash setup_third_party.sh
CGO_ENABLED=1 go build -o impomysql
echo "编译成功: ./impomysql"
```

## 4 CLI 命令参考

所有命令格式为 `impomysql <子命令> <参数>`。

| 子命令 | 用法 | 说明 |
|--------|------|------|
| `task` | `task <config.json>` | 运行单个测试任务 |
| `taskpool` | `taskpool <config.json>` | 运行并行任务池 |
| `ckstable` | `ckstable task <config> <execNum>` | Bug 稳定性验证（单任务） |
| | `ckstable taskpool <config> <threadNum> <execNum>` | Bug 稳定性验证（任务池） |
| `sqlsim` | `sqlsim task <config>` | SQL 简化（单任务） |
| | `sqlsim taskpool <config> <threadNum>` | SQL 简化（任务池） |
| `affversion` | `affversion task <config> <port> <version> [status]` | 版本影响验证 |
| | `affversion taskpool <config> <threadNum> <port> <version> [status]` | 版本影响验证（任务池） |
| `affdbdeployer` | `affdbdeployer <dbdeployerPath> <dbJsonPath> <config> <threadNum> <port> <newestImage> <oldestImage>` | 自动化版本验证 |
| `affclassify` | `affclassify <dbDeployerPath> <dbJsonPath> <config>` | Bug 版本分类 |
| `sqlsimx` | `sqlsimx <"dml"|"ddl"> <inputDML> <inputDDL> <output> <host> <port> <user> <pass> <db> [func]` | SQL 简化工具 |

## 5 配置说明

### 5.1 单任务配置 (TaskConfig)

**MySQL/GaussDB M/PostgreSQL/GaussDB A 通用格式**：

```json
{
  "outputPath": "./output",
  "dbms": "postgresql",
  "taskId": 1,
  "host": "localhost",
  "port": 5432,
  "username": "tpcc",
  "password": "Taurus@123",
  "dbname": "postgres",
  "seed": 0,
  "ddlPath": "./resources/postgresql_ddl.sql",
  "dmlPath": "./resources/postgresql_dml.sql"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `outputPath` | string | 否 | 输出目录，默认 `./output`（自动转绝对路径） |
| `dbms` | string | 否 | 数据库类型，默认 `mysql`，见第 2 节表 |
| `taskId` | int | 是 | 任务 ID，≥ 0 |
| `host` | string | 是 | 数据库主机 |
| `port` | int | 是 | 数据库端口 |
| `username` | string | 是 | 用户名 |
| `password` | string | 是 | 密码 |
| `dbname` | string | 是 | 数据库名 |
| `seed` | int64 | 否 | 随机种子，≤ 0 使用当前时间戳 |
| `ddlPath` | string | 是* | DDL 文件路径（建表+灌数据） |
| `dmlPath` | string | 是* | DML 文件路径（SELECT 语句） |

*使用 randgen 模式时，用 `rdGenPath`/`zzPath`/`yyPath`/`queriesNum` 代替 `ddlPath`/`dmlPath`。

**现有配置文件**：

| 文件 | dbms | 用途 |
|------|------|------|
| `resources/taskconfig.json` | mysql | MySQL 单任务 |
| `resources/postgresql_task.json` | postgresql | PostgreSQL 单任务 |
| `resources/gaussdb_m_task.json` | gaussdb_m | GaussDB M 单任务 |
| `resources/gaussdb_a_task.json` | gaussdb_a | GaussDB A 单任务 |
| `resources/opengauss_m_task.json` | opengauss_m | openGauss M 单任务 |

### 5.2 任务池配置 (TaskPoolConfig)

**MySQL randgen 模式**：

```json
{
  "outputPath": "./output",
  "dbms": "mysql",
  "host": "127.0.0.1",
  "port": 13306,
  "username": "root",
  "password": "123456",
  "dbPrefix": "TEST",
  "seed": 123456,
  "randGenPath": "./resources/go-randgen",
  "zzPath": "./resources/impo.zz.lua",
  "yyPath": "./resources/impo.yy",
  "queriesNum": 100,
  "threadNum": 4,
  "maxTasks": 16,
  "maxTimeS": 60
}
```

**PostgreSQL/GaussDB DDL+DML 模式**：

```json
{
  "outputPath": "./output",
  "dbms": "postgresql",
  "host": "localhost",
  "port": 5432,
  "username": "tpcc",
  "password": "Taurus@123",
  "dbPrefix": "pgtest_",
  "seed": 0,
  "threadNum": 4,
  "maxTasks": 100,
  "maxTimeS": 3600,
  "ddlPath": "./resources/postgresql_ddl.sql",
  "dmlPath": "./resources/postgresql_dml.sql"
}
```

| 字段 | 类型 | 说明 | 适用范围 |
|------|------|------|----------|
| `dbPrefix` | string | 每个线程创建独立数据库：`dbPrefix + threadId` | 所有 |
| `threadNum` | int | 并行线程数 | 所有 |
| `maxTasks` | int | 最大任务数，≤ 0 无限制 | 所有 |
| `maxTimeS` | int | 最大运行秒数，≤ 0 无限制 | 所有 |
| `randGenPath` | string | go-randgen 可执行文件路径 | MySQL |
| `zzPath` | string | go-randgen ZZ 文件路径 | MySQL |
| `yyPath` | string | go-randgen YY 文件路径 | MySQL |
| `queriesNum` | int | 每次生成的 SQL 数量 | MySQL |
| `ddlPath` | string | DDL 文件路径 | PostgreSQL/GaussDB |
| `dmlPath` | string | DML 文件路径 | PostgreSQL/GaussDB |

**TaskPool 数据库隔离机制**：

| dbms | 机制 | 说明 |
|------|------|------|
| `mysql` | ConnectorPool | 每线程一个 MySQL 数据库 `dbPrefix + i`，自动 CREATE DATABASE |
| `postgresql` | PgConnectorPool | 每线程一个 PG 数据库 `dbPrefix + i`，通过 postgres 维护连接自动 CREATE/DROP |
| `gaussdb_m/a` | 需手动创建 | 数据库需预先创建好，TaskPool 尚未适配 |

### 5.3 SQL 文件要求

**DDL 文件**（建表+灌数据）：
- 包含 DROP TABLE + CREATE TABLE + CREATE INDEX + INSERT 语句
- 注释中不能包含 `;` 字符
- 每个语句用 `;` 分隔

**DML 文件**（测试 SELECT 语句）：
- 仅支持 SELECT / UNION SELECT 语句
- INSERT/UPDATE/DELETE 会被 Stage1 过滤
- SQL 不能有副作用（SELECT INTO、变量赋值等）
- 注释中不能包含 `;` 字符

## 6 运行示例

### 6.1 PostgreSQL 环境准备

```bash
# Docker 快速启动 PostgreSQL
docker run -d --name pgtest -p 5432:5432 \
    -e POSTGRES_USER=tpcc \
    -e POSTGRES_PASSWORD=Taurus@123 \
    -e POSTGRES_DB=postgres \
    postgres:17

# 验证连接
psql -h localhost -p 5432 -U tpcc -d postgres -c "SELECT 1"
```

### 6.2 MySQL 环境准备

```bash
docker run -d --name mysqltest -p 13306:3306 \
    -e MYSQL_ROOT_PASSWORD=123456 \
    mysql:8.0.30
```

### 6.3 GaussDB A/M 环境

需在 GaussDB 服务端预先创建兼容模式数据库：

```sql
-- A 模式（Oracle 兼容）
CREATE DATABASE testa WITH DBCOMPATIBILITY 'A';

-- M 模式（MySQL 兼容）
CREATE DATABASE testm WITH DBCOMPATIBILITY 'M';
```

### 6.4 运行单任务

```bash
# PostgreSQL
./impomysql task ./resources/postgresql_task.json

# MySQL
./impomysql task ./resources/taskconfig.json

# GaussDB A
./impomysql task ./resources/gaussdb_a_task.json

# GaussDB M
./impomysql task ./resources/gaussdb_m_task.json
```

### 6.5 运行任务池

```bash
# PostgreSQL 并行测试（4线程）
./impomysql taskpool ./resources/postgresql_taskpool.json

# MySQL 并行测试（randgen 模式）
./impomysql taskpool ./resources/taskpoolconfig.json
```

### 6.6 Bug 稳定性验证

```bash
# 单任务重复执行 10 次验证 Bug 是否稳定复现
./impomysql ckstable task ./resources/postgresql_task.json 10

# 任务池验证（4线程，10次）
./impomysql ckstable taskpool ./resources/taskpoolconfig.json 4 10
```

### 6.7 SQL 简化

```bash
# 简化触发 Bug 的 SQL 语句，便于问题定位
./impomysql sqlsim task ./resources/postgresql_task.json
```

### 6.8 版本影响验证

```bash
# 验证 Bug 在特定版本是否受影响
./impomysql affversion task ./resources/taskconfig.json 13306 "8.0.30"
```

## 7 输出结构与结果解读

### 7.1 目录结构

```
output/<dbms>/task-<taskId>/
  bugs/
    bug-<bugId>-<sqlId>-<mutationName>.log    # Bug 详细日志
    bug-<bugId>-<sqlId>-<mutationName>.json   # Bug 结构化报告
  result.json                                  # 任务汇总统计
  task.log                                     # 任务运行日志
```

示例：

```
output/postgresql/task-1/
  bugs/bug-0-3-FixMWhere1U_Pg.log
  bugs/bug-0-3-FixMWhere1U_Pg.json
  result.json
  task.log
```

### 7.2 result.json 字段解读

| 字段 | 含义 | 正常范围 |
|------|------|----------|
| `ddlSqlsNum` | DDL 语句总数 | 与 DDL 文件行数匹配 |
| `dmlSqlsNum` | DML 语句总数 | 与 DML 文件行数匹配 |
| `stage1ErrNum` | Stage1 解析失败数 | 应较少 |
| `stage1ExecErrNum` | Stage1 执行失败数 | 应较少 |
| `stage1SkippedNum` | A 模式跳过的 SQL 数 | 仅 A 模式有 |
| `stage2ErrNum` | Stage2 变异失败数 | 应为 0 或极少 |
| `stage2UnitNum` | 变异单元总数 | 应远大于 dmlSqlsNum |
| `stage2UnitErrNum` | 变异单元解析失败数 | 应较少 |
| `stage2UnitExecErrNum` | 变异单元执行失败数 | 应较少 |
| `impoBugsNum` | **检测到的逻辑漏洞数** | **核心指标** |
| `saveBugErrNum` | Bug 报告保存失败数 | 应为 0 |

### 7.3 Bug 报告 (bug-*.json) 字段

| 字段 | 含义 |
|------|------|
| `bugId` | Bug 序号 |
| `sqlId` | 原 SQL 在 DML 文件中的序号 |
| `mutationName` | 变异策略名称 |
| `isUpper` | true=上变异(扩大)，false=下变异(缩小) |
| `originalSql` | 原始 SQL（Stage1 预处理后） |
| `originalResult` | 原始 SQL 执行结果（行集） |
| `mutatedSql` | 变异后的 SQL |
| `mutatedResult` | 变异 SQL 执行结果（行集） |

**判定逻辑**：
- 上变异 (`isUpper=true`)：`originalResult.Rows` 应是 `mutatedResult.Rows` 的子集，违反则为 Bug
- 下变异 (`isUpper=false`)：`mutatedResult.Rows` 应是 `originalResult.Rows` 的子集，违反则为 Bug

## 8 变异策略说明

### 8.1 Stage1 预处理（过滤不支持特性）

**MySQL/GaussDB M 通用预处理**：

| 过滤项 | 操作 |
|--------|------|
| 聚合函数 + GROUP BY | 移除 |
| 窗口函数 | 移除 |
| LEFT/RIGHT JOIN | 转 INNER JOIN |
| LIMIT | 移除 |
| 不确定函数 (RAND, UUID 等) | 移除 |

**PostgreSQL 额外预处理**：

| 过滤项 | 操作 |
|--------|------|
| DISTINCT ON | 移除 ON 列，保留 DISTINCT |
| ORDER BY | 移除 |
| FOR UPDATE/SHARE | 移除 |
| LIMIT/OFFSET | 移除 |
| INTO clause | 移除 |

**GaussDB A 额外预处理**：

| 过滤项 | 操作 |
|--------|------|
| CONNECT BY | 跳过整个 SQL |
| PL/SQL 块 | 跳过 |
| ROWNUM | 保留原样 |
| `(+)` 外连接 | 转 LEFT JOIN |

### 8.2 Stage2 变异类型

**MySQL/GaussDB M 变异**：

| 名称 | 变异 | 方向 |
|------|------|------|
| FixMWhere1U / FixMWhere0L | WHERE → 1 / 0 | 上/下 |
| FixMHaving1U / FixMHaving0L | HAVING → 1 / 0 | 上/下 |
| FixMOn1U / FixMOn0L | ON → 1 / 0 | 上/下 |
| FixMDistinctU / FixMDistinctL | 删除/添加 DISTINCT | 上/下 |
| FixMUnionAllU / FixMUnionAllL | UNION → UNION ALL / 反向 | 上/下 |
| FixMCmpOpU / FixMCmpOpL | >→>=,<→<=,=→>= / >=→>,<=→< | 上/下 |
| FixMInNullU | IN 列表添加 NULL | 上 |
| RdMLikeU / RdMLikeL | LIKE 模式扩展/收缩 | 上/下 |
| RdMRegExpU / RdMRegExpL | 正则模式扩展/收缩 | 上/下 |

**PostgreSQL 变异（16 种）**：

| 名称 | 变异 | 方向 |
|------|------|------|
| FixMWhere1U_Pg / FixMWhere0L_Pg | WHERE → TRUE / FALSE | 上/下 |
| FixMHaving1U_Pg / FixMHaving0L_Pg | HAVING → TRUE / FALSE | 上/下 |
| FixMOn1U_Pg / FixMOn0L_Pg | ON → TRUE / FALSE | 上/下 |
| FixMDistinctU_Pg / FixMDistinctL_Pg | 删除/添加 DISTINCT | 上/下 |
| FixMUnionAllU_Pg / FixMUnionAllL_Pg | UNION → UNION ALL / 反向 | 上/下 |
| FixMCmpOpU_Pg / FixMCmpOpL_Pg | 比较运算符变换 | 上/下 |
| FixMInNullU_Pg | IN 列表添加 NULL | 上 |
| RdMLikePgU / RdMLikePgL | LIKE 模式扩展/收缩 | 上/下 |
| RdMRegExpPgU / RdMRegExpPgL | 正则 `~` 模式扩展/收缩 | 上/下 |

## 9 问题定位方法

### 9.1 通用排查流程

```
运行报错 → 查看 task.log → 定位错误类型 → 按下表排查
```

### 9.2 编译阶段问题

| 现象 | 排查步骤 |
|------|----------|
| `gcc not found` | 1. `gcc --version` 确认已安装<br>2. 确认 PATH 包含 gcc<br>3. 确认 `CGO_ENABLED=1` |
| `undefined: pgquery.*` | 1. 确认 `CGO_ENABLED=1`（pg_query_go 含 C 代码）<br>2. 确认 `third_party/pg_query_go/` 存在<br>3. 确认 go.mod replace 指向 `./third_party/pg_query_go` |
| `cannot find package` | 1. 运行 `go mod tidy`<br>2. 检查 go.mod replace 路径是否正确<br>3. 检查网络（go proxy 访问） |
| `cgo: C compiler "gcc"` | Windows: 安装 MinGW-w64 并加入 PATH<br>Linux: `apt install gcc` |

### 9.2 运行阶段问题

| 现象 | 排查步骤 |
|------|----------|
| 程序立即退出 `len(args) <= 1` | 命令缺少子命令，检查 CLI 格式 |
| `new task config error` | 1. 检查 JSON 文件路径是否正确<br>2. 检查 JSON 格式是否合法<br>3. 检查字段完整性 |
| 连接失败 | 1. 检查 host/port 是否可达<br>2. 检查用户名密码<br>3. 检查 dbname 是否存在<br>4. PostgreSQL: 检查 `pg_hba.conf`<br>5. GaussDB: 检查 DBCOMPATIBILITY 模式 |
| `SELECT 1 failed, database may crash` | 1. 数据库服务可能崩溃<br>2. 连接超时/断开<br>3. 查看数据库服务日志 |
| 所有 SQL 被 Stage1 过滤 | 1. 检查 DML 文件是否全是 SELECT<br>2. 检查 `stage1ErrNum` 和 `stage1ExecErrNum`<br>3. 查看 task.log 中 `[Stage1 Error]` 详情 |
| 无 Bug 检出 | 1. 检查 `stage2UnitNum` 是否 > 0<br>2. 检查 `stage2UnitExecErrNum` 是否过高<br>3. 可能数据库确实无此类漏洞 |

### 9.3 PostgreSQL 特定问题

| 现象 | 排查步骤 |
|------|----------|
| `database "xxx" does not exist` | 单任务：检查 dbname 配置<br>TaskPool：检查 dbPrefix，系统会自动创建 `dbPrefix+0`, `dbPrefix+1`, ... |
| `password authentication failed` | 1. 检查 pg_hba.conf 认证方式<br>2. 确认密码正确<br>3. SCRAM-SHA-256 需要 pgx/v5 支持 |
| `connection refused` | 1. `systemctl status postgresql`<br>2. 检查端口 5432 是否监听 |
| TaskPool `CREATE DATABASE` 失败 | 1. 确认连接 postgres 维护库的权限<br>2. 确认 `CREATEDB` 权限已授予用户<br>3. 手动测试: `psql -U tpcc -d postgres -c "CREATE DATABASE pgtest_0"` |
| 正则变异报错 | PostgreSQL `~` 操作符与 MySQL `REGEXP` 语法不同，使用 pg_query_go 解析 |

### 9.4 GaussDB A 模式特定问题

| 现象 | 排查步骤 |
|------|----------|
| `parse error` | 1. 确认数据库是 A 模式（`WITH DBCOMPATIBILITY 'A'`）<br>2. 检查 SQL 是否含 CONNECT BY（会被跳过）<br>3. Oracle 特有函数可能不支持 |
| ROWNUM 相关错误 | ROWNUM 保留原样执行，需 GaussDB A 原生支持 |

### 9.5 查看详细日志

```bash
# 查看任务日志
cat output/postgresql/task-1/task.log

# 查看任务池日志
cat output/postgresql/taskpool.log

# 查看 Bug 详细报告
cat output/postgresql/task-1/bugs/bug-0-3-FixMWhere1U_Pg.json

# 实时监控日志
tail -f output/postgresql/task-1/task.log
```

### 9.6 结果快速诊断脚本

```bash
#!/bin/bash
# diagnose.sh - 快速诊断 Pinolo 任务结果
RESULT=$1  # result.json 路径

echo "=== 任务结果诊断 ==="
bugs=$(python3 -c "import json; d=json.load(open('$RESULT')); print(d['impoBugsNum'])")
units=$(python3 -c "import json; d=json.load(open('$RESULT')); print(d['stage2UnitNum'])")
s1err=$(python3 -c "import json; d=json.load(open('$RESULT')); print(d['stage1ErrNum'])")
s2err=$(python3 -c "import json; d=json.load(open('$RESULT')); print(d['stage2ErrNum'])")

echo "逻辑漏洞数: $bugs"
echo "变异单元数: $units"
echo "Stage1 解析失败: $s1err"
echo "Stage2 变异失败: $s2err"

if [ "$bugs" -gt 0 ]; then
    echo "发现 Bug！查看 bugs/ 目录获取详情"
elif [ "$units" -eq 0 ]; then
    echo "无变异单元执行，检查 Stage1/Stage2 是否正常"
else
    echo "未发现逻辑漏洞"
fi
```

## 10 项目架构

```
Pinolo-main/
├── main.go                          # CLI 入口，8 个子命令
├── connector/                        # 数据库连接层
│   ├── connector.go                  # MySQL Connector（go-sql-driver/mysql）
│   ├── connectorpool.go              # MySQL 连接池（TaskPool 用）
│   ├── postgresql.go                 # PostgreSQL Connector（pgx/v5 pgxpool）
│   ├── pg_connectorpool.go           # PostgreSQL 连接池（TaskPool 用）
│   ├── opengauss.go                  # GaussDB M Connector（openGauss-connector-go-pq）
│   ├── gaussdb_a.go                  # GaussDB A Connector（openGauss-connector-go-pq）
│   ├── universal_connector.go        # 统一连接器路由
│   ├── interface.go                  # SQLExecutor 接口定义
│   ├── result.go                     # Result/EachSql 结构
│   └── extractsqls.go                # SQL 文件解析（按 ; 分割）
├── parser/                           # Parser 适配层
│   ├── tidb_adapter.go               # TiDB Parser 适配（MySQL/M 模式）
│   ├── pgquery_adapter.go            # pg_query_go 适配（PostgreSQL/A 模式）
│   └── oracle_preprocessor.go        # Oracle 语法预处理（A 模式）
├── mutation/
│   ├── stage1/                       # SQL 预处理
│   │   ├── stage1.go                 # MySQL Stage1 入口
│   │   ├── stage1_pg.go              # PostgreSQL Stage1 入口
│   │   ├── stage1_a.go               # A 模式 Stage1 入口
│   │   └── initvisitor.go            # AST 遍历（移除聚合/窗口/LIMIT 等）
│   ├── stage2/                       # 变异引擎
│   │   ├── stage2.go                 # MySQL Stage2 入口
│   │   ├── pg_stage2.go              # PostgreSQL Stage2 入口
│   │   ├── mutatevisitor.go          # MySQL AST 变异遍历器
│   │   ├── pg_mutatevisitor.go       # PostgreSQL AST 变异遍历器
│   │   ├── pg_mutate_functions.go    # PostgreSQL 16 种变异实现
│   │   └── fixm*.go / rdm*.go        # MySQL 各变异实现
│   └── oracle/oracle.go              # Implication Oracle（结果包含关系检查）
├── task/                             # 任务编排层
│   ├── task.go                       # MySQL 单任务
│   ├── postgresql_task.go            # PostgreSQL 单任务
│   ├── gaussdb_task.go               # GaussDB M 单任务 + RunTaskUniversal 路由
│   ├── gaussdb_a_task.go             # GaussDB A 单任务
│   └── taskpool.go                   # TaskPool（MySQL + PostgreSQL）
├── resources/                        # 测试资源
│   ├── postgresql_task.json          # PostgreSQL 单任务配置
│   ├── postgresql_taskpool.json      # PostgreSQL 任务池配置
│   ├── postgresql_ddl.sql / dml.sql  # PostgreSQL 测试 SQL
│   ├── gaussdb_a_task.json           # GaussDB A 配置
│   ├── gaussdb_m_task.json           # GaussDB M 配置
│   └── ...                           # 其他数据库配置和 SQL 文件
├── third_party/                      # 内置依赖（go.mod replace 指向此处）
│   ├── pg_query_go/                  # PostgreSQL 17 Parser（含 C 代码）
│   └── openGauss-connector-go-pq/    # openGauss Go 驱动
└── docs/
    ├── release_notes.md              # 版本变更记录
    └── USER_GUIDE.md                 # 本文档
```

## 11 版本信息

| 组件 | 版本 | 适用范围 |
|------|------|----------|
| TiDB Parser | v5.4.2 (commit d6be9105e6c4) | MySQL/GaussDB M |
| pg_query_go | v6.0.0 (PostgreSQL 17) | PostgreSQL/GaussDB A |
| go-sql-driver/mysql | v1.6.0 | MySQL 系列 |
| pgx/v5 | v5.9.2 | PostgreSQL |
| openGauss-connector-go-pq | v0.0.0 | GaussDB M/A |

---

*文档更新日期：2026-05-21*