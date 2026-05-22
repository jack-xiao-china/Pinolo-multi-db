#!/bin/bash
# setup_third_party.sh - Initialize local dependencies for Pinolo
# Run this script before first compilation
# Usage: bash setup_third_party.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
THIRD_PARTY_DIR="$SCRIPT_DIR/third_party"

mkdir -p "$THIRD_PARTY_DIR"

# pg_query_go v6 (PostgreSQL 17 Parser) - not published on Go proxy
if [ ! -d "$THIRD_PARTY_DIR/pg_query_go" ]; then
    echo "Cloning pg_query_go..."
    git clone --depth 1 https://github.com/pganalyze/pg_query_go.git "$THIRD_PARTY_DIR/pg_query_go"
    cd "$THIRD_PARTY_DIR/pg_query_go"
    # Checkout v6 branch if available, otherwise use latest
    git checkout v6 2>/dev/null || echo "Using default branch (v6 may not be tagged yet)"
    cd "$SCRIPT_DIR"
else
    echo "pg_query_go already exists, skipping"
fi

# openGauss-connector-go-pq - from gitee
if [ ! -d "$THIRD_PARTY_DIR/openGauss-connector-go-pq" ]; then
    echo "Cloning openGauss-connector-go-pq..."
    git clone --depth 1 https://gitee.com/opengauss/openGauss-connector-go-pq.git "$THIRD_PARTY_DIR/openGauss-connector-go-pq"
    cd "$SCRIPT_DIR"
else
    echo "openGauss-connector-go-pq already exists, skipping"
fi

echo "Third-party dependencies initialized."
echo "Now run: CGO_ENABLED=1 go build -o impomysql"