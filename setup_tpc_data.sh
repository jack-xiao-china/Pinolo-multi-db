#!/bin/bash
# TPC-H 和 TPC-DS 数据生成工具脚本
# 自动下载、编译、生成最小数据量测试数据

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TPC_DIR="$SCRIPT_DIR/tpc-tools"
DATA_DIR="$SCRIPT_DIR/tpc-data"

echo "=========================================="
echo "TPC Benchmark Data Generation Tool"
echo "=========================================="

# 创建目录
mkdir -p "$TPC_DIR"
mkdir -p "$DATA_DIR/tpch"
mkdir -p "$DATA_DIR/tpcds"

# ========================================
# 1. TPC-H dbgen
# ========================================
echo ""
echo "[1/4] Checking TPC-H dbgen..."
cd "$TPC_DIR"

# 检查是否已存在 dbgen
if [ -f "tpch-dbgen/dbgen" ]; then
    echo "✓ dbgen already compiled"
else
    echo ""
    echo "TPC-H dbgen 需要从源码编译"
    echo ""
    echo "请执行以下步骤："
    echo "  1. 下载 TPC-H dbgen 源码："
    echo "     cd $TPC_DIR"
    echo "     git clone https://github.com/electrum/tpch-dbgen.git"
    echo "     或从 TPC 官网下载: http://www.tpc.org/tpch/"
    echo ""
    echo "  2. 编译："
    echo "     cd tpch-dbgen"
    echo "     make"
    echo ""
    echo "  3. 生成最小数据量 (Scale Factor = 0.01, 约 10MB)："
    echo "     ./dbgen -s 0.01 -f"
    echo ""
    echo "  4. 移动数据文件："
    echo "     mv *.tbl $DATA_DIR/tpch/"
    echo ""
    echo "完成后重新运行此脚本。"
    exit 1
fi

echo ""
echo "[2/4] Verifying TPC-H data..."
if [ -f "$DATA_DIR/tpch/lineitem.tbl" ]; then
    echo "✓ TPC-H data already exists"
    ls -lh "$DATA_DIR/tpch/"
else
    echo "✗ TPC-H data not found"
    echo "请生成数据："
    echo "  cd $TPC_DIR/tpch-dbgen"
    echo "  ./dbgen -s 0.01 -f"
    echo "  mv *.tbl $DATA_DIR/tpch/"
    exit 1
fi

# ========================================
# 2. TPC-DS dsdgen
# ========================================
echo ""
echo "[3/4] Checking TPC-DS dsdgen..."
if [ -f "$TPC_DIR/tpc-ds/tools/dsdgen" ]; then
    echo "✓ dsdgen already compiled"
else
    echo ""
    echo "TPC-DS dsdgen 需要从 TPC 官网下载"
    echo ""
    echo "请执行以下步骤："
    echo "  1. 从 http://www.tpc.org/tpcds/ 下载 TPC-DS v3.2.0"
    echo "  2. 解压到 $TPC_DIR/tpc-ds/"
    echo "  3. cd tools && make"
    echo "  4. 生成最小数据量 (Scale Factor = 1, 约 1GB)："
    echo "     ./dsdgen -scale 1 -dir $DATA_DIR/tpcds/"
    echo ""
    echo "注意：TPC-DS 最小数据量约 1GB，如果磁盘空间有限可跳过"
fi

echo ""
echo "[4/4] Verifying TPC-DS data..."
if [ -f "$DATA_DIR/tpcds/store_sales.dat" ]; then
    echo "✓ TPC-DS data already exists"
    ls -lh "$DATA_DIR/tpcds/" | head -10
else
    echo "⚠ TPC-DS data not found (optional)"
    echo "  如需生成 TPC-DS 数据，请按上述步骤操作"
fi

# ========================================
# 3. 创建数据加载脚本
# ========================================
echo ""
echo "Creating data loading scripts..."

# TPC-H MySQL 加载脚本
cat > "$SCRIPT_DIR/load_tpch_mysql.sh" << 'EOF'
#!/bin/bash
# 加载 TPC-H 数据到 MySQL
# 用法: ./load_tpch_mysql.sh <host> <port> <user> <password>

HOST=${1:-127.0.0.1}
PORT=${2:-3306}
USER=${3:-tpcc}
PASS=${4:-Taurus@123}
DB="tpch"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$SCRIPT_DIR/tpc-data/tpch"

echo "Loading TPC-H data to MySQL ($HOST:$PORT, database: $DB)..."

# 创建数据库
mysql -h$HOST -P$PORT -u$USER -p$PASS -e "CREATE DATABASE IF NOT EXISTS $DB;"

# 执行 DDL
mysql -h$HOST -P$PORT -u$USER -p$PASS $DB < "$SCRIPT_DIR/resources/tpch_ddl.sql"

# 加载数据
for table in region nation supplier customer part partsupp orders lineitem; do
    echo "Loading $table..."
    mysql -h$HOST -P$PORT -u$USER -p$PASS $DB -e \
        "LOAD DATA LOCAL INFILE '$DATA_DIR/${table}.tbl' INTO TABLE $table FIELDS TERMINATED BY '|' LINES TERMINATED BY '|\n';"
done

echo "TPC-H data loaded successfully!"
EOF

chmod +x "$SCRIPT_DIR/load_tpch_mysql.sh"

# TPC-H PostgreSQL 加载脚本
cat > "$SCRIPT_DIR/load_tpch_pg.sh" << 'EOF'
#!/bin/bash
# 加载 TPC-H 数据到 PostgreSQL
# 用法: ./load_tpch_pg.sh <host> <port> <user> <password>

HOST=${1:-localhost}
PORT=${2:-5432}
USER=${3:-tpcc}
PASS=${4:-Taurus@123}
DB="tpch"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$SCRIPT_DIR/tpc-data/tpch"

export PGPASSWORD=$PASS

echo "Loading TPC-H data to PostgreSQL ($HOST:$PORT, database: $DB)..."

# 创建数据库
psql -h$HOST -p$PORT -U$USER -c "CREATE DATABASE $DB;" 2>/dev/null || true

# 执行 DDL
psql -h$HOST -p$PORT -U$USER -d$DB -f "$SCRIPT_DIR/resources/tpch_ddl.sql"

# 加载数据
for table in region nation supplier customer part partsupp orders lineitem; do
    echo "Loading $table..."
    psql -h$HOST -p$PORT -U$USER -d$DB -c \
        "\\COPY $table FROM '$DATA_DIR/${table}.tbl' DELIMITER '|' CSV;"
done

echo "TPC-H data loaded successfully!"
EOF

chmod +x "$SCRIPT_DIR/load_tpch_pg.sh"

# ========================================
# 4. 使用说明
# ========================================
echo ""
echo "=========================================="
echo "Data Generation Status"
echo "=========================================="
echo ""
if [ -f "$DATA_DIR/tpch/lineitem.tbl" ]; then
    echo "✓ TPC-H data: $DATA_DIR/tpch/"
    echo "  - 8 tables, ~10MB total"
    echo "  - Scale Factor: 0.01"
    echo ""
fi
if [ -f "$DATA_DIR/tpcds/store_sales.dat" ]; then
    echo "✓ TPC-DS data: $DATA_DIR/tpcds/"
    echo "  - 25 tables, ~1GB total"
    echo "  - Scale Factor: 1"
    echo ""
fi
echo "Loading scripts created:"
echo "  - load_tpch_mysql.sh"
echo "  - load_tpch_pg.sh"
echo ""
echo "Usage examples:"
echo "  # MySQL"
echo "  ./load_tpch_mysql.sh 127.0.0.1 3306 tpcc Taurus@123"
echo ""
echo "  # PostgreSQL"
echo "  ./load_tpch_pg.sh localhost 5432 tpcc Taurus@123"
echo ""
echo "  # Run Pinolo TPC-H test"
echo "  ./impomysql task resources/tpch_task.json"
echo ""
