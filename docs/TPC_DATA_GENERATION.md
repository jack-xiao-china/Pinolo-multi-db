# TPC-H/TPC-DS 测试数据生成指南

本文档说明如何使用 TPC 官方工具生成最小数据量的测试数据。

## 概述

- **TPC-H**: Scale Factor (SF) = 0.01，约 **10MB** 数据
- **TPC-DS**: Scale Factor (SF) = 1，约 **1GB** 数据

## TPC-H 数据生成（推荐）

### 步骤 1：下载 dbgen

从 GitHub 下载 TPC-H dbgen 工具：

```bash
cd tpc-tools
git clone https://github.com/electrum/tpch-dbgen.git
```

**备用方案**：从 TPC 官网下载
- 访问: http://www.tpc.org/tpch/
- 下载 "TPC-H Specification and Program"
- 解压到 `tpc-tools/tpch-dbgen/`

### 步骤 2：编译 dbgen

```bash
cd tpch-dbgen
make
```

### 步骤 3：生成最小数据量

```bash
# 生成 Scale Factor 0.01 的数据（约 10MB）
./dbgen -s 0.01 -f

# 移动数据文件到指定目录
mkdir -p ../tpc-data/tpch
mv *.tbl ../tpc-data/tpch/
```

生成的文件：
- `region.tbl` - 5 行
- `nation.tbl` - 25 行
- `supplier.tbl` - 100 行
- `customer.tbl` - 1,500 行
- `part.tbl` - 200 行
- `partsupp.tbl` - 800 行
- `orders.tbl` - 15,000 行
- `lineitem.tbl` - 60,057 行（最大表）

### 步骤 4：加载数据到数据库

#### MySQL

```bash
./load_tpch_mysql.sh 127.0.0.1 3306 tpcc Taurus@123
```

#### PostgreSQL

```bash
./load_tpch_pg.sh localhost 5432 tpcc Taurus@123
```

### 步骤 5：运行 Pinolo 测试

```bash
# 测试 MySQL
./impomysql task resources/tpch_task.json

# 测试 PostgreSQL
./impomysql task resources/tpch_pg_task.json

# 测试 GaussDB-M
./impomysql task resources/tpch_gaussdb_m_task.json
```

## TPC-DS 数据生成（可选）

### 步骤 1：下载 dsdgen

从 TPC 官网下载 TPC-DS 工具：
- 访问: http://www.tpc.org/tpcds/
- 下载 "TPC-DS v3.2.0 Specification and Program"
- 解压到 `tpc-tools/tpc-ds/`

### 步骤 2：编译 dsdgen

```bash
cd tpc-ds/tools
make
```

### 步骤 3：生成数据

```bash
# 生成 Scale Factor 1 的数据（约 1GB）
mkdir -p ../../tpc-data/tpcds
./dsdgen -scale 1 -dir ../../tpc-data/tpcds/
```

**注意**：TPC-DS 最小数据量约 1GB，如果磁盘空间有限可以跳过。

### 步骤 4：加载数据

```bash
# MySQL
./load_tpcds_mysql.sh 127.0.0.1 3306 tpcc Taurus@123

# PostgreSQL
./load_tpcds_pg.sh localhost 5432 tpcc Taurus@123
```

## 验证数据

### 检查 TPC-H 数据

```bash
# MySQL
mysql -h127.0.0.1 -P3306 -utpcc -pTaurus@123 tpch -e "SELECT COUNT(*) FROM lineitem;"
# 预期输出: 60057

# PostgreSQL
psql -hlocalhost -p5432 -Utpcc -dtpch -c "SELECT COUNT(*) FROM lineitem;"
# 预期输出: 60057
```

### 检查表结构

```bash
# MySQL
mysql -h127.0.0.1 -P3306 -utpcc -pTaurus@123 tpch -e "SHOW TABLES;"

# PostgreSQL
psql -hlocalhost -p5432 -Utpcc -dtpch -c "\dt"
```

## 目录结构

```
Pinolo-main/
├── resources/
│   ├── tpch_ddl.sql          # TPC-H 表结构
│   ├── tpch_dml.sql          # TPC-H 22 个查询
│   ├── tpcds_ddl.sql         # TPC-DS 表结构
│   ├── tpcds_dml.sql         # TPC-DS 查询
│   ├── tpch_task.json        # MySQL 任务配置
│   └── tpch_pg_task.json     # PostgreSQL 任务配置
├── tpc-tools/                # TPC 工具源码（需手动下载）
│   ├── tpch-dbgen/           # TPC-H dbgen
│   └── tpc-ds/               # TPC-DS dsdgen
└── tpc-data/                 # 生成的测试数据
    ├── tpch/                 # TPC-H 数据文件
    │   ├── region.tbl
    │   ├── nation.tbl
    │   ├── supplier.tbl
    │   ├── customer.tbl
    │   ├── part.tbl
    │   ├── partsupp.tbl
    │   ├── orders.tbl
    │   └── lineitem.tbl
    └── tpcds/                # TPC-DS 数据文件（可选）
```

## 常见问题

### Q1: 编译 dbgen 时出现错误？

**解决方案**：确保安装了 build-essential 和 gcc

```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# CentOS/RHEL
sudo yum install gcc make
```

### Q2: 加载数据时出现权限错误？

**MySQL**：需要开启 `local_infile` 功能

```bash
# 编辑 my.cnf
[mysqld]
local-infile=1

# 重启 MySQL
sudo systemctl restart mysql
```

**PostgreSQL**：确保用户有 `COPY` 权限

```sql
ALTER USER tpcc WITH SUPERUSER;
```

### Q3: 数据量太小，想生成更大的数据？

调整 Scale Factor：

```bash
# TPC-H: SF=1 约 1GB
./dbgen -s 1 -f

# TPC-H: SF=10 约 10GB
./dbgen -s 10 -f
```

### Q4: 如何清理已生成的数据？

```bash
# 删除数据文件
rm -rf tpc-data/

# 删除数据库
mysql -h127.0.0.1 -P3306 -utpcc -pTaurus@123 -e "DROP DATABASE tpch;"
psql -hlocalhost -p5432 -Utpcc -c "DROP DATABASE tpch;"
```

## 性能基准

### TPC-H (SF=0.01) 数据量

| 表名 | 行数 | 大小 |
|------|------|------|
| region | 5 | ~1KB |
| nation | 25 | ~2KB |
| supplier | 100 | ~8KB |
| customer | 1,500 | ~120KB |
| part | 200 | ~16KB |
| partsupp | 800 | ~64KB |
| orders | 15,000 | ~1.2MB |
| lineitem | 60,057 | ~5MB |
| **总计** | **77,687** | **~10MB** |

### TPC-DS (SF=1) 数据量

| 事实表 | 行数 | 大小 |
|--------|------|------|
| store_sales | 2,750,473 | ~200MB |
| catalog_sales | 1,441,548 | ~100MB |
| web_sales | 719,384 | ~50MB |
| store_returns | 287,514 | ~20MB |
| catalog_returns | 144,067 | ~10MB |
| web_returns | 71,763 | ~5MB |
| inventory | 11,745,000 | ~800MB |
| **总计** | ~17M | **~1GB** |

## 参考链接

- TPC-H 官网: http://www.tpc.org/tpch/
- TPC-DS 官网: http://www.tpc.org/tpcds/
- dbgen GitHub: https://github.com/electrum/tpch-dbgen
