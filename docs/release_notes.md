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