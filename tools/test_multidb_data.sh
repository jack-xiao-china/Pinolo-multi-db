#!/bin/bash
# 多数据库测试数据自动化测试脚本
# 用于验证 resources/multidb_test/ 中的 DDL 和 DML 文件

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$PROJECT_DIR/resources/multidb_test"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    echo -e "\n${YELLOW}=== Test: $1 ===${NC}"
}

# MySQL 测试
test_mysql() {
    log_test "MySQL"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    local HOST="${MYSQL_HOST:-127.0.0.1}"
    local PORT="${MYSQL_PORT:-3306}"
    local USER="${MYSQL_USER:-tpcc}"
    local PASS="${MYSQL_PASSWORD:-Taurus@123}"
    local DB="multidb_test_mysql"

    log_info "Connecting to MySQL at $HOST:$PORT as $USER"

    # 检查连接
    if ! mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" -e "SELECT 1;" >/dev/null 2>&1; then
        log_error "Cannot connect to MySQL"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 删除旧数据库
    log_info "Dropping old database if exists"
    mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" -e "DROP DATABASE IF EXISTS $DB;" 2>&1 | grep -v "Warning"

    # 创建数据库
    log_info "Creating database $DB"
    mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" -e "CREATE DATABASE $DB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>&1 | grep -v "Warning"

    # 加载 DDL
    log_info "Loading DDL from $RESOURCE_DIR/ddl_mysql.sql"
    if ! mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" "$DB" < "$RESOURCE_DIR/ddl_mysql.sql" 2>&1 | grep -v "Warning"; then
        log_error "Failed to load DDL"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 加载 DML
    log_info "Loading DML from $RESOURCE_DIR/dml_mysql.sql"
    if ! mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" "$DB" < "$RESOURCE_DIR/dml_mysql.sql" 2>&1 | grep -v "Warning"; then
        log_error "Failed to load DML"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 验证数据
    log_info "Verifying data..."
    local TABLE_COUNT=$(mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" "$DB" -N -e "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='$DB';" 2>&1 | grep -v "Warning")
    local ROW_COUNT=$(mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" "$DB" -N -e "SELECT SUM(TABLE_ROWS) FROM information_schema.TABLES WHERE TABLE_SCHEMA='$DB';" 2>&1 | grep -v "Warning")

    log_info "Tables: $TABLE_COUNT, Total rows: $ROW_COUNT"

    if [ "$TABLE_COUNT" -ge 9 ] && [ "$ROW_COUNT" -ge 4000 ]; then
        log_info "MySQL test PASSED"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "MySQL test FAILED (expected >= 9 tables and >= 4000 rows)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# PostgreSQL 测试
test_postgresql() {
    log_test "PostgreSQL"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    local HOST="${POSTGRES_HOST:-localhost}"
    local PORT="${POSTGRES_PORT:-5432}"
    local USER="${POSTGRES_USER:-tpcc}"
    local PASS="${POSTGRES_PASSWORD:-Taurus@123}"
    local DB="multidb_test_postgresql"

    export PGPASSWORD="$PASS"

    log_info "Connecting to PostgreSQL at $HOST:$PORT as $USER"

    # 检查连接
    if ! psql -h"$HOST" -p"$PORT" -U"$USER" -c "SELECT 1;" >/dev/null 2>&1; then
        log_error "Cannot connect to PostgreSQL"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 删除旧数据库
    log_info "Dropping old database if exists"
    psql -h"$HOST" -p"$PORT" -U"$USER" -c "DROP DATABASE IF EXISTS $DB;" 2>&1

    # 创建数据库
    log_info "Creating database $DB"
    psql -h"$HOST" -p"$PORT" -U"$USER" -c "CREATE DATABASE $DB ENCODING 'UTF8';" 2>&1

    # 加载 DDL
    log_info "Loading DDL from $RESOURCE_DIR/ddl_postgresql.sql"
    if ! psql -h"$HOST" -p"$PORT" -U"$USER" -d"$DB" < "$RESOURCE_DIR/ddl_postgresql.sql" 2>&1; then
        log_error "Failed to load DDL"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 加载 DML
    log_info "Loading DML from $RESOURCE_DIR/dml_postgresql.sql"
    if ! psql -h"$HOST" -p"$PORT" -U"$USER" -d"$DB" < "$RESOURCE_DIR/dml_postgresql.sql" 2>&1; then
        log_error "Failed to load DML"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 验证数据
    log_info "Verifying data..."
    local TABLE_COUNT=$(psql -h"$HOST" -p"$PORT" -U"$USER" -d"$DB" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE';" 2>&1 | tr -d ' ')
    local ROW_COUNT=$(psql -h"$HOST" -p"$PORT" -U"$USER" -d"$DB" -t -c "SELECT SUM(n_tup_ins) FROM pg_stat_user_tables;" 2>&1 | tr -d ' ')

    log_info "Tables: $TABLE_COUNT, Total rows: $ROW_COUNT"

    if [ "$TABLE_COUNT" -ge 9 ] && [ "$ROW_COUNT" -ge 4000 ]; then
        log_info "PostgreSQL test PASSED"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "PostgreSQL test FAILED (expected >= 9 tables and >= 4000 rows)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# GaussDB-M 测试（使用 MySQL 协议）
test_gaussdb_m() {
    log_test "GaussDB-M"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    local HOST="${GAUSSDBM_HOST:-121.37.186.131}"
    local PORT="${GAUSSDBM_PORT:-19995}"
    local USER="${GAUSSDBM_USER:-sqlbuilder1}"
    local PASS="${GAUSSDBM_PASSWORD:-huawei@123}"
    local DB="multidb_test_gaussdb_m"

    log_info "Connecting to GaussDB-M at $HOST:$PORT as $USER"

    # 检查连接
    if ! mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" -e "SELECT 1;" >/dev/null 2>&1; then
        log_warn "Cannot connect to GaussDB-M, skipping test"
        TOTAL_TESTS=$((TOTAL_TESTS - 1))
        return 0
    fi

    # 删除旧数据库
    log_info "Dropping old database if exists"
    mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" -e "DROP DATABASE IF EXISTS $DB;" 2>&1 | grep -v "Warning" || true

    # 创建数据库
    log_info "Creating database $DB"
    mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" -e "CREATE DATABASE $DB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>&1 | grep -v "Warning"

    # 加载 DDL
    log_info "Loading DDL from $RESOURCE_DIR/ddl_gaussdb_m.sql"
    if ! mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" "$DB" < "$RESOURCE_DIR/ddl_gaussdb_m.sql" 2>&1 | grep -v "Warning"; then
        log_error "Failed to load DDL"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 加载 DML
    log_info "Loading DML from $RESOURCE_DIR/dml_gaussdb_m.sql"
    if ! mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" "$DB" < "$RESOURCE_DIR/dml_gaussdb_m.sql" 2>&1 | grep -v "Warning"; then
        log_error "Failed to load DML"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    log_info "GaussDB-M test PASSED (data loaded successfully)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
    return 0
}

# GaussDB-A 测试（使用 PostgreSQL 协议）
test_gaussdb_a() {
    log_test "GaussDB-A"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    local HOST="${GAUSSDBA_HOST:-121.37.186.131}"
    local PORT="${GAUSSDBA_PORT:-19995}"
    local USER="${GAUSSDBA_USER:-sqlbuilder1}"
    local PASS="${GAUSSDBA_PASSWORD:-huawei@123}"
    local DB="multidb_test_gaussdb_a"

    export PGPASSWORD="$PASS"

    log_info "Connecting to GaussDB-A at $HOST:$PORT as $USER"

    # 检查连接
    if ! psql -h"$HOST" -p"$PORT" -U"$USER" -c "SELECT 1;" >/dev/null 2>&1; then
        log_warn "Cannot connect to GaussDB-A, skipping test"
        TOTAL_TESTS=$((TOTAL_TESTS - 1))
        return 0
    fi

    # 删除旧数据库
    log_info "Dropping old database if exists"
    psql -h"$HOST" -p"$PORT" -U"$USER" -c "DROP DATABASE IF EXISTS $DB;" 2>&1 || true

    # 创建数据库
    log_info "Creating database $DB"
    psql -h"$HOST" -p"$PORT" -U"$USER" -c "CREATE DATABASE $DB ENCODING 'UTF8';" 2>&1

    # 加载 DDL
    log_info "Loading DDL from $RESOURCE_DIR/ddl_gaussdb_a.sql"
    if ! psql -h"$HOST" -p"$PORT" -U"$USER" -d"$DB" < "$RESOURCE_DIR/ddl_gaussdb_a.sql" 2>&1; then
        log_error "Failed to load DDL"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # 加载 DML
    log_info "Loading DML from $RESOURCE_DIR/dml_gaussdb_a.sql"
    if ! psql -h"$HOST" -p"$PORT" -U"$USER" -d"$DB" < "$RESOURCE_DIR/dml_gaussdb_a.sql" 2>&1; then
        log_error "Failed to load DML"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    log_info "GaussDB-A test PASSED (data loaded successfully)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
    return 0
}

# 主函数
main() {
    echo "========================================="
    echo "多数据库测试数据自动化测试"
    echo "========================================="
    echo ""

    # 检查资源文件
    if [ ! -d "$RESOURCE_DIR" ]; then
        log_error "Resource directory not found: $RESOURCE_DIR"
        exit 1
    fi

    # 运行测试
    test_mysql || true
    test_postgresql || true
    test_gaussdb_m || true
    test_gaussdb_a || true

    # 输出结果
    echo ""
    echo "========================================="
    echo "测试总结"
    echo "========================================="
    echo -e "总测试数: $TOTAL_TESTS"
    echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
    echo -e "${RED}失败: $FAILED_TESTS${NC}"
    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        log_info "All tests passed!"
        exit 0
    else
        log_error "Some tests failed!"
        exit 1
    fi
}

# 运行主函数
main "$@"
