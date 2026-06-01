## v0.2.7 | 2026-06-01
- 新增：EET 变换规则移植到 mutation 引擎，实现 5 种语义等价变换：
  - `FixMAndTrueU`：WHERE E → WHERE (p OR NOT p OR p IS NULL) AND E（永真式包裹）
  - `FixMOrFalseL`：WHERE E → WHERE (p AND NOT p AND p IS NOT NULL) OR E（永假式包裹）
  - `FixMCaseTrueU`：WHERE E → WHERE CASE WHEN TRUE THEN E ELSE rand END
  - `FixMCaseFalseL`：WHERE E → WHERE CASE WHEN FALSE THEN rand ELSE E END
  - `FixMCaseRandEq`：WHERE E → WHERE CASE WHEN rand THEN E ELSE E END（等价变换）
- 新增：MySQL EET mutations（`mutation/stage2/eet_mutations.go` + `eet_mutations_test.go`）
- 新增：PostgreSQL EET mutations（`mutation/stage2/pg_eet_mutations.go` + `pg_eet_mutations_test.go`）
- 修复：`testsqls/testsqls.go` getEnvIntOrDefault 函数签名错误（key 参数类型应为 string）
- 修复：`pg_mutatevisitor.go` UNION 检测条件错误（SETOP_NONE vs SET_OPERATION_UNDEFINED）

## v0.2.6 | 2026-05-30
- 新增：随机 SQL 生成模式（`genMode: "eet_style"`），基于数据库 schema 发现 + scope-aware 生成，作为第三种运行模式加入（与现有 DDL/DML 文件模式和 go-randgen 模式并存）
- 新增：`connector/schema.go` SchemaInfo/TableInfo/ColumnInfo 结构体和各 DBMS 的 DiscoverSchema() 方法（MySQL INFORMATION_SCHEMA、PostgreSQL pg_catalog、GaussDB-M/A 适配）
- 新增：`generator/` 模块——Go 内置 SQL 随机生成器，借鉴 EET scope-based + SQLancer ExpressionGenerator 设计，支持 4 种 SELECT 形状（plain、UNION、CTE、derived table）、JOIN、子查询、GROUP BY/HAVING 等
- 新增：TaskConfig/TaskPoolConfig 支持 GenMode/GenDepth/GenQueries 等新配置字段，RunTask/PrepareAndRunTask 新增随机生成分支
- 新增：配置文件示例 `resources/task_gen_config.json` 和 `resources/taskpool_gen_config.json`

## v0.2.5 | 2026-05-30
- 新增：vendor 目录纳入版本控制，支持离线编译（`go build -mod=vendor`）
- 新增：third_party 目录纳入版本控制（pg_query_go、openGauss-connector-go-pq）
- 优化：USER_GUIDE.md 编译步骤简化为 vendor 模式，无需网络下载依赖
- 优化：移除 setup_third_party.sh 和 go mod download 步骤
- 优化：一键部署改为离线部署示例

## v0.2.4 | 2026-05-30
- 优化：移除所有 Docker 环境准备内容（USER_GUIDE.md、README.md、POSTGRESQL_SUPPORT.md、CLAUDE.md、testsqls/doc.go、connector/connector_test.go），项目定位为工具本身，被测数据库环境由用户自行指定
- 优化：USER_GUIDE.md Section 6 简化为"运行示例"，仅保留 GaussDB 兼容模式创建说明和运行命令
- 优化：README.md 移除所有 Docker 命令，改为"用户需自行准备 DBMS 环境"

## v0.2.3 | 2026-05-30
- 安全：`testsqls/testsqls.go` 和 `connector/connector_test.go` 硬编码密码改为环境变量读取（TEST_DB_HOST/TEST_DB_PASSWORD/TEST_DB_PORT 等），默认密码改为 `your_password`
- 安全：`testsqls/doc.go` Docker 注释中密码替换为 `your_password`

## v0.2.2 | 2026-05-30
- 优化：重写 USER_GUIDE.md Section 3 编译部分——完整 Linux 系统依赖清单（build-essential/glibc-devel）、CGO 编译原理说明、8 步完整编译流程、GOPROXY 配置、网络故障处理
- 优化：编译问题排查扩展 Linux CGO 错误（pthread、stdlib.h、gnu99、OOM、Go 模块超时等 6 种新增场景）
- 优化：一键部署脚本修正（Go 版本 URL 可配置、添加 GOPROXY、build-essential 替代 gcc、错误处理）
- 新增：Section 6 Docker 环境示例——MariaDB、TiDB、OceanBase、openGauss/GaussDB 完整覆盖
- 优化：Section 9 排查表补充编译阶段 Linux CGO 错误行，修正重复编号
- 优化：Section 11 版本信息新增 Go 和 CGO 行，更新文档日期

## v0.2.1 | 2026-05-30
- 安全：移除 docs 和 resources 中所有真实凭证信息（用户名、密码、IP 地址），替换为占位符 `your_username`/`your_password`/`your_host`

## v0.2.0 | 2026-05-21
- 优化：go.mod replace 依赖改为相对路径（`./third_party/`），支持 Linux 编译移植
- 新增：pg_query_go 和 openGauss-connector-go-pq 内置到 `third_party/` 目录，无需外部下载
- 新增：PostgreSQL TaskPool 支持（`PgConnectorPool` + `RunTaskPoolForPostgreSQL`）
- 新增：PostgreSQL TaskPool 配置示例 `resources/postgresql_taskpool.json`
- 修复：`PostgreSQLConnector.InitDB()` 实现了真实的表清理逻辑（DROP public schema tables CASCADE）
- 修复：`RunTaskPostgreSQL` 支持外部 connector 参数（TaskPool 共享连接）
- 优化：pgx/v5 从 indirect 升为 direct 依赖
- 优化：`TaskPoolConfig` 支持 PostgreSQL 的 DDLPath/DMLPath 配置模式
- 优化：`RunTaskPoolUniversal` 路由函数，统一管理不同数据库类型的 TaskPool
- 优化：重写 USER_GUIDE.md，包含完整编译命令、运行命令、问题定位方法
- 优化：go.mod replace 依赖改为相对路径（`./third_party/`），支持 Linux 编译移植
- 新增：pg_query_go 和 openGauss-connector-go-pq 内置到 `third_party/` 目录，无需外部下载
- 新增：PostgreSQL TaskPool 支持（`PgConnectorPool` + `RunTaskPoolForPostgreSQL`）
- 新增：PostgreSQL TaskPool 配置示例 `resources/postgresql_taskpool.json`
- 修复：`PostgreSQLConnector.InitDB()` 实现了真实的表清理逻辑（DROP public schema tables CASCADE）
- 修复：`RunTaskPostgreSQL` 支持外部 connector 参数（TaskPool 共享连接）
- 优化：pgx/v5 从 indirect 升为 direct 依赖
- 优化：`TaskPoolConfig` 支持 PostgreSQL 的 DDLPath/DMLPath 配置模式
- 优化：`RunTaskPoolUniversal` 路由函数，统一管理不同数据库类型的 TaskPool