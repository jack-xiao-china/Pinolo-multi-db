# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PINOLO (impomysql) is a tool for detecting logical bugs in MySQL and MySQL-compatible DBMSs (MariaDB, TiDB, OceanBase) through Implication Oracle testing. It uses the TiDB parser (v5.4.2) for SQL AST manipulation and go-randgen for test case generation.

## Build and Run Commands

```bash
# Build
go build

# Run a single task (needs a running MySQL-compatible DBMS)
./impomysql task ./resources/taskconfig.json

# Run task pool (parallel testing)
./impomysql taskpool ./resources/taskpoolconfig.json

# Run tests (requires MySQL on localhost:13306)
go test ./...
go test ./mutation/stage1/...
go test -run TestImpoMutateSelectValue ./mutation/stage2/
```

## CLI Commands

The main executable supports these subcommands:

| Command | Description |
|---------|-------------|
| `task <config.json>` | Run a single testing task |
| `taskpool <config.json>` | Run tasks in parallel |
| `ckstable task <config.json> <execNum>` | Check bug stability for a task |
| `ckstable taskpool <config.json> <threadNum> <execNum>` | Check bug stability for taskpool |
| `sqlsim task <config.json>` | Simplify SQL statements for bugs |
| `sqlsim taskpool <config.json> <threadNum>` | Simplify SQL for taskpool |
| `affversion task <config.json> <port> <version> [whereVersionStatus]` | Verify bug affects specific DBMS version |
| `affversion taskpool <config.json> <threadNum> <port> <version>` | Verify bugs for taskpool |
| `affdbdeployer <args>` | Automated version verification with dbdeployer |
| `affclassify <args>` | Classify bugs by affected versions |
| `sqlsimx <opt> <inputDML> <inputDDL> <output> <host> <port> <user> <pass> <db>` | SQL simplification tool |

## Architecture

### Core Components

1. **connector** (`connector/connector.go`) - MySQL database connection, SQL execution, result handling. Creates databases, executes raw SQL, returns structured results.

2. **mutation/stage1** - SQL preprocessing that removes unsupported features:
   - Aggregate functions and GROUP BY
   - Window functions
   - LEFT/RIGHT JOIN
   - LIMIT clauses
   - Uncertain functions (e.g., RAND())

3. **mutation/stage2** - Mutation engine that transforms SQL predicates:
   - Uses `MutateVisitor` to traverse AST and identify mutation points
   - Applies mutations like: `WHERE x → WHERE 1`, `HAVING x → HAVING 1`, `DISTINCT → non-DISTINCT`, comparison operators modifications
   - Mutation naming: `FixM` (fixed mutation), `RdM` (random mutation), suffix `U` (upper/expanding), `L` (lower/shrinking)

4. **mutation/oracle** (`oracle.go`) - Implication Oracle that compares original vs mutated SQL results. A bug is detected when result containment relationship is violated. Also provides `CheckEquivalence()` for EET semantic rewrite mutations.

5. **task** (`task.go`, `taskpool.go`) - Orchestrates testing workflow:
   - Loads config, connects to DBMS, runs mutations, reports bugs
   - TaskPool runs multiple tasks in parallel with thread control
   - Selects `oracle.Check()` for implication mutations and `oracle.CheckEquivalence()` for equivalence mutations based on `IsEquivalence` field

6. **generator** (`generator/`) - Random SQL generation engine for EET-style testing:
   - Schema discovery from live database via `connector.SchemaInfo`
   - Scope-aware expression generation with type constraints
   - Supports MySQL, PostgreSQL, GaussDB-A/M dialects
   - Configurable features: JOINs, self-joins, subqueries, CTEs, ENUM constants

### Mutation Types

See `mutation/stage2/allmutations.go` for all mutation names. Mutations are categorized as:

**Implication mutations** (use `oracle.Check()` with containment logic):
- `FixMDistinctU/L` - DISTINCT modifications
- `FixMUnionAllU/L` - UNION to UNION ALL changes
- `FixMCmpOpU/L` - Comparison operator changes (>, <, = → >=, <=; **excluding !=**)
- `FixMInNullU` - Add NULL to IN lists
- `FixMWhere1U/0L` - WHERE clause modifications
- `FixMHaving1U/0L` - HAVING clause modifications
- `FixMOn1U/0L` - JOIN ON clause modifications
- `RdMLikeU/L` - LIKE pattern modifications
- `RdMRegExpU/L` - REGEXP pattern modifications
- `FixMBetweenDropUpperU` - x BETWEEN a AND b → x >= a (upper: drop upper bound)
- `FixMBetweenDropLowerU` - x BETWEEN a AND b → x <= b (upper: drop lower bound)
- `FixMNullEqToLowerL` - a <=> b → a = b (lower: null-safe eq → normal eq)
- `FixMAllToAnyU` - ALL(subq) → ANY(subq) (upper: ALL ⊆ ANY)
- `FixMAnyToAllL` - ANY(subq) → ALL(subq) (lower: ANY ⊇ ALL)

**Equivalence mutations** (use `oracle.CheckEquivalence()` for exact equality):
- `FixMAndTrueU` - E → (p OR NOT p OR p IS NULL) AND E (tautology wrapping)
- `FixMOrFalseL` - E → (p AND NOT p AND p IS NOT NULL) OR E (contradiction wrapping)
- `FixMCaseTrueU` - E → CASE WHEN TRUE THEN E ELSE rand END
- `FixMCaseFalseL` - E → CASE WHEN FALSE THEN rand ELSE E END
- `FixMCaseRandEq` - E → CASE WHEN rand THEN E ELSE E END (random branch)
- `FixMDeMorganAnd` - (A AND B) → NOT(NOT(A) OR NOT(B)) (De Morgan AND)
- `FixMDeMorganOr` - (A OR B) → NOT(NOT(A) AND NOT(B)) (De Morgan OR)
- `FixMBetweenToCmp` - x BETWEEN a AND b → (x >= a) AND (x <= b)
- `FixMCoalesceToCase` - COALESCE(a, b) → CASE WHEN a IS NOT NULL THEN a ELSE b END
- `FixMNullifToCase` - NULLIF(a, b) → CASE WHEN a = b THEN NULL ELSE a END
- `FixMExistsToIn` - EXISTS(subq) → NULL-safe IN equivalent
- `FixMInToExists` - IN(subq) → NULL-safe EXISTS equivalent
- `FixMIfToCase` - IF(cond, a, b) → CASE WHEN cond THEN a ELSE b END (GaussDB-M)
- `FixMConcatToPipe` - CONCAT(a, b) → a || b (GaussDB-M)

**PG mutations** follow the same categories with `_Pg` suffix, plus `FixMIsNotDistinctFromToLowerL_Pg` (IS NOT DISTINCT FROM → =).

### Test Configuration

Task config JSON (see `resources/taskconfig.json`):
- `outputPath`, `dbms`, `taskId` - Output directory naming
- `host`, `port`, `username`, `password`, `dbname` - DBMS connection
- `seed` - Random seed (<=0 uses current time)
- `ddlPath`, `dmlPath` - SQL test files

For go-randgen mode, use `rdGenPath`, `zzPath`, `yyPath`, `queriesNum` instead of ddl/dml paths.

### Output Structure

```
output/<dbms>/task-<id>/
  bugs/
    bug-<bugId>-<sqlId>-<mutationName>.log
    bug-<bugId>-<sqlId>-<mutationName>.json
  result.json
  task.log
```

## Key Dependencies

- `github.com/pingcap/tidb/parser` - SQL parser (v5.4.2, commit `d6be9105e6c4`)
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/mattn/go-sqlite3` - SQLite for affversion tracking
- `github.com/sirupsen/logrus` - Logging

## Test Database Setup

Tests require a MySQL-compatible database running. The `testsqls` package uses hardcoded connection params:
- Host: `127.0.0.1`
- MySQL port: `13306`
- MariaDB port: `23306`
- TiDB port: `4000`
- OceanBase port: `2881`
- Credentials: `root/your_password`, database `TEST`

Users must provide their own DBMS test environment. The tool itself does not depend on Docker.

## TPC Benchmark Test Suites

The project includes complete TPC-H and TPC-DS benchmark test suites for logical bug detection:

### TPC-H (Decision Support Benchmark)
- **Schema**: `resources/tpch_ddl.sql` - 8 tables (nation, region, part, supplier, partsupp, customer, orders, lineitem)
- **Queries**: `resources/tpch_dml.sql` - 22 standard analytical queries
- **Task Configs**:
  - `tpch_task.json` - MySQL (port 3306, database: tpch)
  - `tpch_pg_task.json` - PostgreSQL (port 5432, database: tpch)
  - `tpch_gaussdb_m_task.json` - GaussDB-M (MySQL compatibility mode)
  - `tpch_gaussdb_a_task.json` - GaussDB-A (Oracle compatibility mode)

### TPC-DS (Decision Support Benchmark - Advanced)
- **Schema**: `resources/tpcds_ddl.sql` - 25 tables (7 fact tables + 18 dimension tables)
- **Queries**: `resources/tpcds_dml.sql` - 20 representative queries (simplified from full 99-query suite)
- **Task Configs**:
  - `tpcds_task.json` - MySQL (port 3306, database: tpcds)
  - `tpcds_pg_task.json` - PostgreSQL (port 5432, database: tpcds)

### Usage Example
```bash
# Run TPC-H on MySQL
./impomysql task resources/tpch_task.json

# Run TPC-DS on PostgreSQL
./impomysql task resources/tpcds_pg_task.json

# Run TPC-H on GaussDB-M
./impomysql task resources/tpch_gaussdb_m_task.json
```

**Note**: These benchmarks require pre-populated test data. Use TPC-H/TPC-DS data generators (e.g., `dbgen` for TPC-H, `dsdgen` for TPC-DS) to load data before running tests.

## Important Notes

- Only SELECT statements are supported for mutation testing
- SQL must have no side effects (no SELECT INTO, assignment operations)
- Comments in SQL files cannot contain `;` character
- The `impo.yy` grammar file in `resources/` defines the AST patterns for mutation