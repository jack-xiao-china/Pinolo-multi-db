# PINOLO - MySQL Logic Bug Detector

A portable tool for detecting logical bugs in MySQL and MySQL-compatible DBMSs through Implication Oracle testing.

## Features

- **Cross-Platform**: Windows, Linux, macOS (Intel & Apple Silicon)
- **Logic Bug Detection**: Uses Implication Oracle to find SQL query engine defects
- **Mutation Testing**: 17 mutation types for comprehensive coverage
- **Portable**: Single executable, no dependencies

## Quick Start

### Windows
```batch
run.bat task task_template.json
```

### Linux/macOS
```bash
chmod +x run.sh
./run.sh task task_template.json
```

### Direct Execution
```bash
# Windows
impomysql.exe task task_template.json

# Linux
./impomysql_linux task task_template.json

# macOS (Intel)
./impomysql_darwin_amd64 task task_template.json

# macOS (Apple Silicon)
./impomysql_darwin_arm64 task task_template.json
```

## Configuration

Edit `task_template.json`:

```json
{
  "outputPath": "./output",
  "dbms": "mysql",
  "taskId": 1,
  "host": "localhost",
  "port": 3306,
  "username": "your_user",
  "password": "your_password",
  "dbname": "TEST_PINOLO",
  "seed": 0,
  "ddlPath": "./test_ddl.sql",
  "dmlPath": "./test_dml.sql"
}
```

### Config Fields

| Field | Description | Required |
|-------|-------------|----------|
| outputPath | Output directory | No (default: ./output) |
| dbms | DBMS name (mysql/mariadb/tidb/oceanbase) | No (default: mysql) |
| taskId | Task identifier | Yes |
| host | Database host | Yes |
| port | Database port | Yes |
| username | Database user | Yes |
| password | Database password | Yes |
| dbname | Test database name | Yes |
| seed | Random seed (0 = use timestamp) | No |
| ddlPath | Path to DDL SQL file | Yes |
| dmlPath | Path to DML SQL file | Yes |

## Test SQL Files

### DDL (test_ddl.sql)
- Creates test tables
- Inserts sample data
- Executed before each test run

### DML (test_dml.sql)
- Contains SELECT statements to test
- Each statement is mutated and compared

**Important Restrictions:**
- Only SELECT statements supported
- Comments cannot contain semicolons (;)
- SQL must have no side effects (no SELECT INTO)

## Output Structure

```
output/mysql/task-<id>/
  bugs/               # Detected logical bugs
    bug-*.log         # Bug details
    bug-*.json        # Structured report
  result.json         # Test statistics
  task.log            # Execution log
```

## Result Statistics

| Field | Meaning |
|-------|---------|
| ddlSqlsNum | DDL statements executed |
| dmlSqlsNum | Test queries processed |
| stage1ExecErrNum | Preprocessing failures |
| stage2UnitNum | Mutations generated |
| impoBugsNum | **Logical bugs detected** |

## Mutation Types

| Mutation | Description |
|----------|-------------|
| FixMDistinctU/L | DISTINCT modifications |
| FixMCmpOpU/L | Comparison operator changes (>, <, = → >=, <=) |
| FixMUnionAllU/L | UNION/UNION ALL conversion |
| FixMInNullU | Add NULL to IN lists |
| FixMWhere1U/0L | WHERE clause replacement |
| FixMHaving1U/0L | HAVING clause replacement |
| FixMOn1U/0L | JOIN ON condition replacement |
| RdMLikeU/L | LIKE pattern modifications |
| RdMRegExpU/L | REGEXP pattern modifications |

## Advanced Commands

```bash
# Parallel testing pool
impomysql taskpool taskpool_template.json

# SQL simplification
impomysql sqlsim task task_template.json

# Bug stability verification (run 10 times)
impomysql ckstable task task_template.json 10

# Version verification
impomysql affversion task task_template.json 3306 "8.0.30"
```

## Supported DBMS

| DBMS | Status |
|------|--------|
| MySQL 5.7+ | ✅ Full support |
| MariaDB | ✅ Full support |
| TiDB | ✅ Full support |
| OceanBase (MySQL mode) | ✅ Full support |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Connection failed | Verify host/port/credentials in config |
| No bugs found | Normal result - DB implements SQL correctly |
| Stage1 errors | Some features not supported (aggregates, window functions) |
| Permission denied (Linux/Mac) | Run `chmod +x impomysql_*` |

## File Contents

| File | Description |
|------|-------------|
| impomysql.exe | Windows executable |
| impomysql_linux | Linux executable |
| impomysql_darwin_amd64 | macOS Intel executable |
| impomysql_darwin_arm64 | macOS Apple Silicon executable |
| run.bat | Windows launcher |
| run.sh | Linux/macOS launcher |
| task_template.json | Single task config template |
| taskpool_template.json | Parallel task pool config |
| test_ddl.sql | Database schema and test data |
| test_dml.sql | Test SELECT statements |
| README.md | This documentation |

## Architecture

```
SQL Input → Parse (TiDB Parser) → Stage1 (Preprocess)
           → Stage2 (Mutate) → Execute → Oracle (Compare)
           → Report Bugs
```

## License

This tool is based on PINOLO (impomysql) project.
Uses TiDB Parser (v5.4.2) for SQL AST manipulation.