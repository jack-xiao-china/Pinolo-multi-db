## v0.7.2 | 2026-06-04
- 修复：PG numeric 格式转换支持大整数 — `formatPgNumeric()` 使用 float64 替代 int64 解析数字，支持超过 int64 容量（9.2e18）的 PostgreSQL numeric 值
- 集成测试验证（v0.7.2）：MySQL 0 bugs, PG 0 bugs, GaussDB-M 0 bugs, GaussDB-A 0 bugs, 全部 0 假阳性

## v0.7.1 | 2026-06-04
- 修复：Error Oracle 假阳性过滤 — 添加 `isExpectedMutationError()` 过滤预期错误（"Subquery returns more than 1 row"、磁盘满、超时等）
- 修复：PG `FixMWhere1U_Pg` / `FixMWhere0L_Pg` 使用 `TRUE`/`FALSE` 替代 `1`/`0`（PG 要求 boolean 类型）
- 修复：PG numeric 类型内部格式 `{digits exp NaN ...}` 自动转换为标准小数字符串
- 新增：聚合查询自动检测 — `isAggregateResult()` 自动识别单行数值结果，切换到 `CheckAggregate` 数值比较
- 新增：全局查询超时 — `DefaultQueryTimeout=60s`，全部 4 个 connector 使用 `QueryContext` 超时
- 优化：k=2 组合变异 maxK2Pairs 降至 0（避免磁盘空间不足导致超时）
- 集成测试验证：MySQL 0 bugs, PG 0 bugs, GaussDB-M 0 bugs, GaussDB-A 0 bugs, 全部 0 假阳性

## v0.7.0 | 2026-06-04
- 重构：Task Runner 代码去重 — 提取 `runner.go` 公共变异处理循环（`processMutationUnit` + `MutateUnitAdapter` + `MutationLoopContext`），消除 4 个 task runner 80% 重复代码
- 优化：k=2 组合变异 CalCandidates 缓存 — 按变异后 SQL 缓存解析结果，避免 O(N²) 重复解析，显著减少开销
- 优化：PG k=2 组合变异 — `MutateAllForPostgreSQL` 新增 Phase 2 同 AST 双变异
- 新增：覆盖率引导变异 — `mutation_stats.go` 维护全局变异成功率统计，`PrioritizeUnits` 按历史 bug 发现率排序变异优先级
- 新增：聚合函数近似支持 — `AggregateInitVisitor` 保留 GROUP BY/聚合函数，`CheckAggregate` 数值比较 Oracle，`AggregateMode` 配置开关
- 新增：NULL 安全标记 — `NullMarker`（`\x00NULL\x00`）替代纯字符串 `"NULL"`，消除 SQL NULL 与字面值 `"NULL"` 的歧义（全部 4 个 connector）
- 文档：P2/P3 优化方案 `docs/pinolo-p2p3-optimization-plan.md`

## v0.6.0 | 2026-06-04
- 新增：`FixMCmpOpULE` / `FixMCmpOpULE_Pg` — `= → <=` upper 变异，补全 `FixMCmpOpU` 的 `= → >=` 方向（MySQL + PG）
- 新增：函数调用内表达式变异 — `visitFuncCallExpr` 递归遍历参数，`visitFuncCastExpr` 递归遍历 CAST 表达式，覆盖 `YEAR(d)=2024`、`ABS(a-b)>0` 等高频模式（MySQL + PG）
- 新增：Error Oracle — upper 变异后执行报错且原始 SQL 成功时，重新执行确认后报告为逻辑 bug（全部 4 个 task runner）
- 新增：`BugReport.IsErrorOracle` / `ErrorMsg` 字段，区分 Error Oracle 和 Implication Oracle bug
- 新增：`TaskResult.ErrorOracleBugsNum` 统计 Error Oracle 检测到的 bug 数
- 文档：P2/P3 优化方案 `docs/pinolo-p2p3-optimization-plan.md`

## v0.5.0 | 2026-06-04
- 修复：BetweenExpr NOT flag 缺失 — `miningBetweenExpr` 现在正确检查 `in.Not` 并翻转 flag，消除 NOT BETWEEN 假阳性（MySQL + PG）
- 修复：TaskPool 主循环不等待 goroutine 完成 — 添加 `sync.WaitGroup`，确保结果完整性（MySQL + PG）
- 修复：Oracle 错误不再终止整个任务 — 改为 `logger.Warn` + `continue`（全部4个 task runner）
- 修复：`false_positive.go` 中 `string(rune())` 转换为 `strconv.Itoa()`，修复不可打印字符
- 新增：IS NULL / IS NOT NULL 蕴含变异（`FixMIsNullToFalseL`、`FixMIsNotNullToTrueU`），覆盖高频 SQL 模式
- 新增：k=2 组合变异 — `MutateAll` 在同方向变异间生成配对组合，搜索空间从 O(N) 扩展到 O(N²)
- 新增：False Positive 检测器移植到 PostgreSQL、GaussDB-M、GaussDB-A task runner
- 优化：`FalsePositiveDetector` 从 `*connector.Connector` 改为 `connector.SQLExecutor` 接口，支持所有 DBMS
- 优化：`normalizeNumeric()` 支持科学计数法（`1e10` → `10000000000`，`1.5E-3` → `0.0015`）
- 文档：深度架构分析报告 `docs/pinolo-deep-analysis.md`
- 文档：整改方案 `docs/pinolo-improvement-plan.md`

## v0.4.1 | 2026-06-04
- 新增：四款 DBMS 集成测试（MySQL, PostgreSQL, GaussDB-M, GaussDB-A）使用 TPC-H v3.0.1 基准
- 新增：PostgreSQL 专用 TPC-H DML 语法（interval '90 days' 替代 interval '90' day）
- 新增：GaussDB-A 专用数据加载脚本（ALTER SESSION SET nls_date_format = 'YYYY-MM-DD'）
- 新增：假阳性检测机制（FalsePositiveDetector，3轮验证，0.67阈值）
- 测试：PostgreSQL 发现 14 个蕴含 Oracle 违规（Q6/Q17/Q19，0 误报）
- 测试：GaussDB-M 24 变异单元 0 bugs（18/24 查询因 interval 语法执行失败）
- 测试：GaussDB-A 0 变异单元（Oracle 兼容模式 SQL 语法不兼容）
- 测试：MySQL 17/24 查询处理完成（Q17 关联子查询超时），0 bugs
- 文档：集成测试报告 `docs/integration-test-report-v0.4.1.md`

## v0.4.0 | 2026-06-03
- 架构重构：移除所有 EET (Equivalent Expression Testing) 等价变换规则，回归 Pinolo 论文核心 Implication Oracle 方法论
- 删除：12 个 EET 等价变异（FixMAndTrueU/OrFalseL/CaseTrueU/CaseFalseL/CaseRandEq/DeMorganAnd/DeMorganOr/BetweenToCmp/CoalesceToCase/NullifToCase/ExistsToIn/InToExists）及其 PG/GaussDB-M 变体
- 删除：`oracle.CheckEquivalence()` 函数、`IsEquivalence` 字段、`isEquivalenceMutation()` 函数
- 保留：所有 Implication 变异（FixMCmpOpU/L, FixMWhere1U/0L, FixMBetweenDropUpperU/LowerU, FixMNullEqToLowerL, FixMAllToAnyU/AnyToAllL 等）
- 新增：`ast_replace.go` 通用 AST 替换工具（从 eet_demorgan.go 提取）
- 新增：`between_drop.go`/`pg_between_drop.go` BETWEEN 丢界蕴含变异独立文件
- 简化：GaussDB-M visitor/stage2 不再包含 EET 变异分发
- 简化：task runner 统一使用 `oracle.Check()` 蕴含 Oracle
- 验证：TPC-H 24 查询 207 变异单元 0 假阳性，TPC-DS 12 查询 0 假阳性

## v0.3.1 | 2026-06-03
- 修复：FixMAndTrueU/FixMOrFalseL tautology/contradiction 包裹添加 `ParenthesesExpr`，解决 AND/OR 优先级问题（`p OR NOT p OR p IS NULL AND E` 被错误解析）
- 修复：FixMBetweenToCmp NOT BETWEEN 使用 De Morgan 定律替代 NOT 包裹，避免括号不平衡（`NOT(x>=a AND x<=b)` → `(x<a) OR (x>b)`）
- 优化：`Result.CMP` 数值归一化（`normalizeNumeric`），解决 `"0"` vs `"0.0000"` 字符串比较误判
- 优化：生成器类型感知统一（MySQL + PG 均使用 `generateTypeCompatibleValue`），减少跨类型比较导致的执行错误
- 优化：WHERE 子句生成概率从 50% 提升至 83%，新增 IN 谓词生成（`generateInPredicate`），增加变异触发覆盖率
- 优化：DISTINCT 概率 17%、HAVING 概率 27%、BETWEEN/IN/LIKE 各 8%，覆盖更多变异类型
- 新增：`generateTypeCompatibleBound`/`generateTypeCompatibleValue` 扩展支持 date/time/datetime/year/bit/enum 等类型
- 测试效果：假阳性从 39 降至 0，变异单元从 597 增至 844（+41%），解析错误下降 44%

## v0.3.0 | 2026-06-02
- 修复：移除 FixMCmpOpL 对 `!=`/`<>` 的无效蕴含变异（MySQL + PG），`!=→<` 无包含关系
- 修复：将 tautology/contradiction/CASE wrapping 变异（FixMAndTrueU/FixMOrFalseL/FixMCaseTrueU/FixMCaseFalseL）修正为等价分类，使用 `CheckEquivalence()` 而非 `Check()`
- 修复：GaussDB-A task runner 根据 `IsEquivalence` 选择 equivalence oracle（之前只用 `Check()`）
- 修复：MySQL task runner 根据 `IsEquivalence` 选择 equivalence oracle
- 修复：移除 `addFixMExistsToIn` 的 `!in.Not` 限制，NOT EXISTS 也支持等价变换
- 新增：2 条 BETWEEN 蕴含变异（MySQL）：
  - `FixMBetweenDropUpperU`：x BETWEEN a AND b → x >= a（满足上下界 ⊆ 满足下界）
  - `FixMBetweenDropLowerU`：x BETWEEN a AND b → x <= b（满足上下界 ⊆ 满足上界）
- 新增：1 条 NullEq 蕴含变异（MySQL）：
  - `FixMNullEqToLowerL`：a <=> b → a = b（= 的 TRUE 集合 ⊆ <=> 的 TRUE 集合）
- 新增：2 条 ALL/ANY 跨量词蕴含变异（MySQL）：
  - `FixMAllToAnyU`：x > ALL(subq) → x > ANY(subq)（满足所有值 ⊆ 满足某个值）
  - `FixMAnyToAllL`：x > ANY(subq) → x > ALL(subq)（满足某个值 ⊇ 满足所有值）
- 新增：`exprReplacer` 支持 `*ast.FuncCallExpr` 整体替换（遍历 Args slice）
- 新增：随机 SQL 生成 SELF JOIN 支持（`EnableSelfJoin` 配置，d6≤2 时添加自连接）
- 新增：随机 SQL 生成 ENUM 类型支持（`parseEnumValues`、`PickEnumValue`、typeMatch 扩展）
- 新增：3 组 PG 版本蕴含变异补齐：
  - `FixMBetweenDropUpperU_Pg`/`FixMBetweenDropLowerU_Pg`：PG BETWEEN 放松上界/下界
  - `FixMAllToAnyU_Pg`/`FixMAnyToAllL_Pg`：PG ALL↔ANY 跨量词
  - `FixMIsNotDistinctFromToLowerL_Pg`：PG IS NOT DISTINCT FROM → =（`AEXPR_NOT_DISTINCT` Kind）

## v0.2.9 | 2026-06-01
- 修复：`RunTaskGaussDB()` 调用标准 MySQL stage1/stage2 函数改为 M-mode 专用函数（`InitAndExecForMMode`/`MutateAllAndExecForMMode`），使 `FixMIfToCase`/`FixMConcatToPipe` 变异可达
- 修复：`RunTaskGaussDB()` 新增 Stage1 Skipped 处理逻辑（M-mode 不支持的语法跳过）
- 新增：`MutateUnit`/`PgMutateUnit` `IsEquivalence` 字段，区分 implication oracle 与 equivalence oracle
- 新增：`isEquivalenceMutation()`/`isEquivalenceMutationPg()` helper 函数
- 新增：task runner 根据 `IsEquivalence` 选择 `oracle.Check()` 或 `oracle.CheckEquivalence()`
- 新增：GaussDB-M EET 集成测试（`gaussdb_m_eet_mutations_test.go`）

## v0.2.8 | 2026-06-01
- 新增：5 条 EET 语义重写规则移植到 MySQL mutation 引擎：
  - `FixMDeMorganAnd`：(A AND B) → NOT(NOT(A) OR NOT(B))（De Morgan 定律 AND→OR）
  - `FixMDeMorganOr`：(A OR B) → NOT(NOT(A) AND NOT(B))（De Morgan 定律 OR→AND）
  - `FixMBetweenToCmp`：x BETWEEN a AND b → (x >= a) AND (x <= b)
  - `FixMCoalesceToCase`：COALESCE(a, b) → CASE WHEN a IS NOT NULL THEN a ELSE b END
  - `FixMNullifToCase`：NULLIF(a, b) → CASE WHEN a = b THEN NULL ELSE a END
- 新增：5 条 EET 语义重写规则移植到 PostgreSQL mutation 引擎：
  - `FixMDeMorganAnd_Pg`、`FixMDeMorganOr_Pg`、`FixMBetweenToCmp_Pg`、`FixMCoalesceToCase_Pg`、`FixMNullifToCase_Pg`
- 新增：Oracle 等价判断函数 `CheckEquivalence`（`mutation/oracle/oracle.go`）
- 新增：MySQL 版文件 `eet_demorgan.go`、`eet_between.go`、`eet_functions.go`
- 新增：PG 版文件 `pg_eet_demorgan.go`、`pg_eet_between.go`、`pg_eet_functions.go`、`pg_eet_subquery.go`
- 新增：2 条 EET 子查询变换规则（MySQL + PG 双版本）：
  - `FixMExistsToIn` / `FixMExistsToIn_Pg`：EXISTS(subquery) → NULL-safe IN 等价变换
  - `FixMInToExists` / `FixMInToExists_Pg`：IN(subquery) → NULL-safe EXISTS 等价变换
- 新增：`mutatevisitor.go` 中 BetweenExpr 和 FuncCallExpr 的 EET mining 支持
- 新增：`pg_mutatevisitor.go` 中 BoolExpr De Morgan mining、AExpr BETWEEN mining、FuncCall/CaseExpr mining 支持
- 新增：`exprReplacer` AST 替换辅助工具、`findFuncCallInWhere`/`findBetweenExprInWhere` PG AST 搜索辅助
- 新增：设计文档 `docs/superpowers/specs/2026-06-01-eet-extension-gaussdb-m-design.md`

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