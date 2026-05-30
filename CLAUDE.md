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

4. **mutation/oracle** (`oracle.go`) - Implication Oracle that compares original vs mutated SQL results. A bug is detected when result containment relationship is violated.

5. **task** (`task.go`, `taskpool.go`) - Orchestrates testing workflow:
   - Loads config, connects to DBMS, runs mutations, reports bugs
   - TaskPool runs multiple tasks in parallel with thread control

### Mutation Types

See `mutation/stage2/allmutations.go` for all mutation names:
- `FixMDistinctU/L` - DISTINCT modifications
- `FixMUnionAllU/L` - UNION to UNION ALL changes
- `FixMCmpOpU/L` - Comparison operator changes (>, <, = to >=, <=)
- `FixMInNullU` - Add NULL to IN lists
- `FixMWhere1U/0L` - WHERE clause modifications
- `FixMHaving1U/0L` - HAVING clause modifications
- `FixMOn1U/0L` - JOIN ON clause modifications
- `RdMLikeU/L` - LIKE pattern modifications
- `RdMRegExpU/L` - REGEXP pattern modifications

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

## Important Notes

- Only SELECT statements are supported for mutation testing
- SQL must have no side effects (no SELECT INTO, assignment operations)
- Comments in SQL files cannot contain `;` character
- The `impo.yy` grammar file in `resources/` defines the AST patterns for mutation