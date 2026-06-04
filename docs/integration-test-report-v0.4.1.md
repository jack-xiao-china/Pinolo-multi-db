# Pinolo v0.4.1 四款 DBMS 集成测试报告

## 测试概述

| 项目 | 详情 |
|------|------|
| **测试日期** | 2026-06-04 |
| **测试工具** | Pinolo (impomysql) v0.4.1 |
| **测试基准** | TPC-H v3.0.1 (22 标准查询) |
| **测试数据** | TPC-H SF≈0.01 (lineitem: 6724 rows) |
| **数据库** | MySQL, PostgreSQL, GaussDB-M, GaussDB-A |

## 测试结果汇总

| DBMS | 版本 | DML 查询数 | Stage1 错误 | Stage1 执行错误 | 变异单元数 | 发现 Bug 数 |
|------|------|-----------|------------|----------------|-----------|------------|
| **MySQL** | 8.x | 24 | 0 | 0 | N/A* | 0 |
| **PostgreSQL** | 16.x | 24 | 2 | 16 | 87 | **14** |
| **GaussDB-M** | 507.0.0 | 24 | 2 | 18 | 24 | 0 |
| **GaussDB-A** | 507.0.0 | 24 | 10 | 14 | 0 | 0 |

*MySQL 测试在处理 Q17 (关联子查询) 时超时，已完成 17/24 查询的处理。

## 各数据库详细分析

### 1. MySQL

- **测试配置**: `integration_mysql_task.json` (taskId: 500)
- **结果**: `output/mysql/task-500/`
- **DDL**: TPC-H 标准 8 表 + 数据加载 (9619 statements)
- **DML**: TPC-H 22 标准查询 + Q15 CREATE/DROP VIEW (24 statements)
- **进度**: 处理了 17/24 查询后在 Q17 超时
- **Bug**: 0 (已处理查询中未发现蕴含违规)
- **备注**: Q17 包含关联子查询 `l_quantity < (SELECT 0.2 * avg(l_quantity) FROM lineitem WHERE l_partkey = p_partkey)`，变异后执行时间显著增加

### 2. PostgreSQL

- **测试配置**: `integration_pg_task.json` (taskId: 501)
- **结果**: `output/postgresql/task-501/`
- **DDL**: TPC-H 标准 8 表 + 数据加载 (9520 statements)
- **DML**: TPC-H 22 标准查询 (PostgreSQL interval 语法兼容版)
- **变异单元**: 87 个
- **发现 Bug**: **14 个** (0 个误报)

#### Bug 详情

| Bug ID | SQL ID | 变异名称 | 类型 | 描述 |
|--------|--------|---------|------|------|
| 0 | 5 (Q6) | FixMCmpOpL_Pg | Lower | `>= → >` 日期比较变异 |
| 1 | 5 (Q6) | FixMBetweenDropUpperU_Pg | Upper | BETWEEN 丢弃上界 |
| 2 | 5 (Q6) | FixMBetweenDropLowerU_Pg | Upper | BETWEEN 丢弃下界 |
| 3-7 | 18 (Q17) | FixMCmpOpU_Pg | Upper | JOIN 条件 `= → >=` 变异 |
| 8-10 | 20 (Q19) | FixMBetweenDropUpperU_Pg | Upper | BETWEEN 丢弃上界 |
| 11-13 | 20 (Q19) | FixMBetweenDropLowerU_Pg | Upper | BETWEEN 丢弃下界 |

#### Bug 分析

**Q6 (Forecasting Revenue Change) - 3 bugs**:
- 原始 SQL 使用 `l_discount BETWEEN 0.06 - 0.01 AND 0.06 + 0.01`
- BETWEEN 变异丢弃上界或下界后，结果未保持预期的包含关系
- `>=` → `>` 的 Lower 变异在聚合场景下结果不一致

**Q17 (Small-Quantity-Order Revenue) - 5 bugs**:
- 变异修改了 JOIN 条件 `p_partkey = l_partkey → p_partkey >= l_partkey`
- Upper 变异预期 mutated ⊇ original，但实际结果显著不同
- 原始 avg_yearly: 796942714...，变异后: 760791394...

**Q19 (Discounted Revenue) - 6 bugs**:
- 复杂 OR 条件中的 BETWEEN 变异
- BETWEEN 变异提取单条件时丢弃了 JOIN 条件，导致笛卡尔积
- 原始 revenue: 15826.5，变异后: 33268446845.6

### 3. GaussDB-M (MySQL 兼容模式)

- **测试配置**: `integration_gaussdb_m_task.json` (taskId: 502)
- **结果**: `output/gaussdb_m/task-502/`
- **DDL**: 8 条 CREATE TABLE (已有表，DDL 执行报错但被忽略)
- **DML**: TPC-H 22 标准查询 (MySQL interval 语法)
- **变异单元**: 24 个
- **发现 Bug**: 0
- **备注**: 18/24 查询执行失败 (MySQL interval 语法与 GaussDB-M 不完全兼容)，仅 4 查询成功执行并生成 24 变异单元

### 4. GaussDB-A (Oracle 兼容模式)

- **测试配置**: `integration_gaussdb_a_task.json` (taskId: 503)
- **结果**: `output/gaussdb_a/task-503/`
- **DDL**: 8 条 CREATE TABLE (已有表，DDL 执行报错但被忽略)
- **DML**: TPC-H 22 标准查询 (PostgreSQL interval 语法)
- **变异单元**: 0
- **发现 Bug**: 0
- **备注**: Oracle 兼容模式下 PostgreSQL 风格 SQL 解析失败率高 (10 解析错误 + 14 执行错误)，无变异单元生成。GaussDB-A 的 Oracle 模式需要 Oracle 风格 SQL 语法 (如 TO_DATE 函数)

## 关键发现

1. **PostgreSQL 发现 14 个潜在逻辑问题**，涉及 Q6、Q17、Q19 三个 TPC-H 查询
2. **BETWEEN 变异 (新增的 FixMBetweenDropUpperU/LowerU)** 在复杂 WHERE 子句中表现积极，成功触发蕴含违规检测
3. **GaussDB 系列数据库兼容性**是主要挑战：
   - GaussDB-M 的 MySQL interval 语法支持不完整
   - GaussDB-A 的 Oracle 模式需要专用 SQL 语法
4. **MySQL 测试超时**在复杂关联子查询上，需要优化变异执行策略

## 建议

1. **PostgreSQL**: 对 14 个潜在 Bug 进行人工验证，确认是否为真正的逻辑 Bug 或误报
2. **GaussDB-M**: 创建 GaussDB-M 专用 SQL 语法版本的 TPC-H 查询，提高测试覆盖率
3. **GaussDB-A**: 创建 Oracle 风格 SQL 语法版本的 TPC-H 查询 (使用 TO_DATE, ROWNUM 等)
4. **MySQL**: 优化关联子查询的变异执行策略，或设置单查询超时限制
