package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// DatabaseType 数据库类型枚举
type DatabaseType string

const (
	MySQL     DatabaseType = "mysql"
	PostgreSQL DatabaseType = "postgresql"
	GaussDBM  DatabaseType = "gaussdb_m"
	GaussDBA  DatabaseType = "gaussdb_a"
)

// TypeMapping 数据库类型映射
type TypeMapping struct {
	MySQL      string
	PostgreSQL string
	GaussDBM   string
	GaussDBA   string
}

// 数据类型映射表
var typeMappings = map[string]TypeMapping{
	// 整数类型
	"tinyint":      {"TINYINT", "SMALLINT", "TINYINT", "SMALLINT"},
	"smallint":     {"SMALLINT", "SMALLINT", "SMALLINT", "SMALLINT"},
	"mediumint":    {"MEDIUMINT", "INTEGER", "MEDIUMINT", "INTEGER"},
	"int":          {"INT", "INTEGER", "INT", "INTEGER"},
	"bigint":       {"BIGINT", "BIGINT", "BIGINT", "BIGINT"},
	"serial":       {"INT AUTO_INCREMENT", "SERIAL", "INT AUTO_INCREMENT", "SERIAL"},

	// 浮点和定点类型
	"float":        {"FLOAT", "REAL", "FLOAT", "REAL"},
	"double":       {"DOUBLE", "DOUBLE PRECISION", "DOUBLE", "DOUBLE PRECISION"},
	"decimal":      {"DECIMAL(10,2)", "DECIMAL(10,2)", "DECIMAL(10,2)", "DECIMAL(10,2)"},
	"numeric":      {"NUMERIC(10,2)", "NUMERIC(10,2)", "NUMERIC(10,2)", "NUMERIC(10,2)"},

	// 字符串类型
	"char":         {"CHAR(50)", "CHAR(50)", "CHAR(50)", "CHAR(50)"},
	"varchar":      {"VARCHAR(100)", "VARCHAR(100)", "VARCHAR(100)", "VARCHAR(100)"},
	"text":         {"TEXT", "TEXT", "TEXT", "TEXT"},
	"mediumtext":   {"MEDIUMTEXT", "TEXT", "MEDIUMTEXT", "TEXT"},
	"longtext":     {"LONGTEXT", "TEXT", "LONGTEXT", "TEXT"},

	// 二进制类型
	"binary":       {"BINARY(20)", "BYTEA", "BINARY(20)", "BYTEA"},
	"varbinary":    {"VARBINARY(100)", "BYTEA", "VARBINARY(100)", "BYTEA"},
	"blob":         {"BLOB", "BYTEA", "BLOB", "BYTEA"},

	// 日期时间类型
	"date":         {"DATE", "DATE", "DATE", "DATE"},
	"time":         {"TIME", "TIME", "TIME", "TIME"},
	"datetime":     {"DATETIME", "TIMESTAMP", "DATETIME", "TIMESTAMP"},
	"timestamp":    {"TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE"},
	"year":         {"YEAR", "SMALLINT", "YEAR", "SMALLINT"},

	// 布尔类型
	"boolean":      {"BOOLEAN", "BOOLEAN", "BOOLEAN", "BOOLEAN"},

	// JSON 类型
	"json":         {"JSON", "JSON", "JSON", "JSON"},
	"jsonb":        {"JSON", "JSONB", "JSON", "JSONB"},

	// 特殊类型
	"enum":         {"ENUM('A','B','C','D','SPECIAL')", "VARCHAR(50)", "ENUM('A','B','C','D','SPECIAL')", "VARCHAR(50)"},
	"set":          {"SET('read','write','execute','delete')", "TEXT", "SET('read','write','execute','delete')", "TEXT"},
	"uuid":         {"CHAR(36)", "UUID", "CHAR(36)", "UUID"},
	"array":        {"TEXT", "TEXT[]", "TEXT", "TEXT[]"},
}

// IndexMapping 索引类型映射
type IndexMapping struct {
	MySQL      string
	PostgreSQL string
	GaussDBM   string
	GaussDBA   string
}

var indexMappings = map[string]IndexMapping{
	"primary":    {"PRIMARY KEY", "PRIMARY KEY", "PRIMARY KEY", "PRIMARY KEY"},
	"unique":     {"UNIQUE", "UNIQUE", "UNIQUE", "UNIQUE"},
	"btree":      {"INDEX", "INDEX", "INDEX", "INDEX"},
	"fulltext":   {"FULLTEXT INDEX", "INDEX", "FULLTEXT INDEX", "INDEX"},
	"spatial":    {"SPATIAL INDEX", "INDEX", "SPATIAL INDEX", "INDEX"},
}

// ColumnDef 列定义
type ColumnDef struct {
	Name         string
	TypeKey      string
	IsPrimaryKey bool
	IsUnique     bool
	IsIndex      bool
	Nullable     bool
	IndexType    string // btree, fulltext, spatial
}

// TableDef 表定义
type TableDef struct {
	Name    string
	Columns []ColumnDef
}

// 表定义列表
var tables = []TableDef{
	{
		Name: "t_int_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_tiny", TypeKey: "tinyint", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_small", TypeKey: "smallint", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_medium", TypeKey: "mediumint", Nullable: true},
			{Name: "c_int", TypeKey: "int", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_big", TypeKey: "bigint", Nullable: true, IsIndex: true, IndexType: "btree"},
		},
	},
	{
		Name: "t_float_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_float", TypeKey: "float", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_double", TypeKey: "double", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_decimal", TypeKey: "decimal", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_numeric", TypeKey: "numeric", Nullable: true},
		},
	},
	{
		Name: "t_string_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_char", TypeKey: "char", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_varchar", TypeKey: "varchar", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_text", TypeKey: "text", Nullable: true, IsIndex: true, IndexType: "fulltext"},
			{Name: "c_mediumtext", TypeKey: "mediumtext", Nullable: true},
			{Name: "c_longtext", TypeKey: "longtext", Nullable: true},
		},
	},
	{
		Name: "t_binary_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_binary", TypeKey: "binary", Nullable: true},
			{Name: "c_varbinary", TypeKey: "varbinary", Nullable: true},
			{Name: "c_blob", TypeKey: "blob", Nullable: true},
		},
	},
	{
		Name: "t_datetime_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_date", TypeKey: "date", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_time", TypeKey: "time", Nullable: true},
			{Name: "c_datetime", TypeKey: "datetime", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_timestamp", TypeKey: "timestamp", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_year", TypeKey: "year", Nullable: true},
		},
	},
	{
		Name: "t_boolean_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_bool", TypeKey: "boolean", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_flag", TypeKey: "int", Nullable: true, IsIndex: true, IndexType: "btree"},
		},
	},
	{
		Name: "t_json_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_json", TypeKey: "json", Nullable: true},
			{Name: "c_jsonb", TypeKey: "jsonb", Nullable: true},
		},
	},
	{
		Name: "t_special_types",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_enum", TypeKey: "enum", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_set", TypeKey: "set", Nullable: true},
			{Name: "c_uuid", TypeKey: "uuid", Nullable: true, IsUnique: true},
			{Name: "c_array", TypeKey: "array", Nullable: true},
		},
	},
	{
		Name: "t_index_test",
		Columns: []ColumnDef{
			{Name: "id", TypeKey: "serial", IsPrimaryKey: true, Nullable: false},
			{Name: "c_unique", TypeKey: "varchar", Nullable: true, IsUnique: true},
			{Name: "c_btree", TypeKey: "varchar", Nullable: true, IsIndex: true, IndexType: "btree"},
			{Name: "c_fulltext", TypeKey: "text", Nullable: true, IsIndex: true, IndexType: "fulltext"},
			{Name: "c_composite1", TypeKey: "int", Nullable: true},
			{Name: "c_composite2", TypeKey: "varchar", Nullable: true},
		},
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gen_multidb_test_data <output_dir>")
		fmt.Println("  Generates test data for MySQL, PostgreSQL, GaussDB-M, GaussDB-A")
		os.Exit(1)
	}

	outputDir := os.Args[1]
	os.MkdirAll(outputDir, 0755)

	dbTypes := []DatabaseType{MySQL, PostgreSQL, GaussDBM, GaussDBA}

	for _, dbType := range dbTypes {
		fmt.Printf("Generating test data for %s...\n", dbType)

		// 生成 DDL
		ddlPath := filepath.Join(outputDir, fmt.Sprintf("ddl_%s.sql", dbType))
		generateDDL(ddlPath, dbType)

		// 生成 DML
		dmlPath := filepath.Join(outputDir, fmt.Sprintf("dml_%s.sql", dbType))
		generateDML(dmlPath, dbType)
	}

	fmt.Println("\n✓ Test data generated successfully!")
	fmt.Println("\nTo load data:")
	fmt.Println("  MySQL:       mysql -h127.0.0.1 -P3306 -utpcc -pTaurus@123 testdb < ddl_mysql.sql && mysql ... < dml_mysql.sql")
	fmt.Println("  PostgreSQL:  psql -hlocalhost -p5432 -Utpcc -dtestdb < ddl_postgresql.sql && psql ... < dml_postgresql.sql")
	fmt.Println("  GaussDB-M:   mysql -h<host> -P<port> -u<user> -p<pass> testdb < ddl_gaussdb_m.sql && mysql ... < dml_gaussdb_m.sql")
	fmt.Println("  GaussDB-A:   psql -h<host> -p<port> -U<user> -dtestdb < ddl_gaussdb_a.sql && psql ... < dml_gaussdb_a.sql")
}

func generateDDL(path string, dbType DatabaseType) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintf(f, "-- Multi-database test data DDL\n")
	fmt.Fprintf(f, "-- Database: %s\n", dbType)
	fmt.Fprintf(f, "-- Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 删除旧表
	for _, table := range tables {
		fmt.Fprintf(f, "DROP TABLE IF EXISTS %s;\n", table.Name)
	}
	fmt.Fprintln(f)

	// 创建表
	for _, table := range tables {
		fmt.Fprintf(f, "CREATE TABLE %s (\n", table.Name)

		for i, col := range table.Columns {
			typeStr := getTypeString(col.TypeKey, dbType)
			nullable := ""
			if !col.Nullable {
				nullable = " NOT NULL"
			}

			fmt.Fprintf(f, "    %s %s%s", col.Name, typeStr, nullable)

			if i < len(table.Columns)-1 || hasIndexes(table) {
				fmt.Fprint(f, ",")
			}
			fmt.Fprintln(f)
		}

		// 添加索引
		indexLines := []string{}
		for _, col := range table.Columns {
			if col.IsPrimaryKey {
				pkStr := getIndexString("primary", dbType)
				indexLines = append(indexLines, fmt.Sprintf("    %s (%s)", pkStr, col.Name))
			}
			if col.IsUnique && !col.IsPrimaryKey {
				uniqStr := getIndexString("unique", dbType)
				indexLines = append(indexLines, fmt.Sprintf("    %s (%s)", uniqStr, col.Name))
			}
			if col.IsIndex && !col.IsPrimaryKey && !col.IsUnique {
				idxType := col.IndexType
				if idxType == "" {
					idxType = "btree"
				}
				idxStr := getIndexString(idxType, dbType)
				indexName := fmt.Sprintf("idx_%s_%s", table.Name, col.Name)

				// PostgreSQL/GaussDB-A 使用 CREATE INDEX 语句
				if dbType == PostgreSQL || dbType == GaussDBA {
					// 不在表内定义，稍后单独创建
				} else {
					indexLines = append(indexLines, fmt.Sprintf("    %s %s (%s)", idxStr, indexName, col.Name))
				}
			}
		}

		// 复合索引
		if table.Name == "t_index_test" {
			if dbType == PostgreSQL || dbType == GaussDBA {
				// 单独创建
			} else {
				indexLines = append(indexLines, "    INDEX idx_composite (c_composite1, c_composite2)")
			}
		}

		for i, line := range indexLines {
			fmt.Fprint(f, line)
			if i < len(indexLines)-1 {
				fmt.Fprint(f, ",")
			}
			fmt.Fprintln(f)
		}

		fmt.Fprintf(f, ");\n\n")

		// PostgreSQL/GaussDB-A 单独创建索引
		if dbType == PostgreSQL || dbType == GaussDBA {
			for _, col := range table.Columns {
				if col.IsIndex && !col.IsPrimaryKey && !col.IsUnique {
					idxType := col.IndexType
					if idxType == "" {
						idxType = "btree"
					}

					// PostgreSQL/GaussDB-A 不支持在 TEXT 类型上直接创建 GIN 索引
					// 跳过 fulltext 索引或使用 tsvector 转换
					if idxType == "fulltext" && (col.TypeKey == "text" || col.TypeKey == "mediumtext" || col.TypeKey == "longtext") {
						// 使用 to_tsvector 函数创建函数索引
						indexName := fmt.Sprintf("idx_%s_%s", table.Name, col.Name)
						fmt.Fprintf(f, "CREATE INDEX %s ON %s USING gin (to_tsvector('english', %s));\n", indexName, table.Name, col.Name)
						continue
					}

					idxStr := getIndexString(idxType, dbType)
					indexName := fmt.Sprintf("idx_%s_%s", table.Name, col.Name)

					// PostgreSQL/GaussDB-A 使用 USING 子句指定索引方法
					usingClause := ""
					if idxType == "fulltext" {
						usingClause = " USING gin"
					} else if idxType == "spatial" {
						usingClause = " USING gist"
					}

					fmt.Fprintf(f, "CREATE %s %s ON %s%s (%s);\n", idxStr, indexName, table.Name, usingClause, col.Name)
				}
			}
			// 复合索引
			if table.Name == "t_index_test" {
				fmt.Fprintf(f, "CREATE INDEX idx_composite ON %s (c_composite1, c_composite2);\n", table.Name)
			}
			fmt.Fprintln(f)
		}
	}
}

func generateDML(path string, dbType DatabaseType) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintf(f, "-- Multi-database test data DML\n")
	fmt.Fprintf(f, "-- Database: %s\n", dbType)
	fmt.Fprintf(f, "-- Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 添加字符集设置
	if dbType == MySQL || dbType == GaussDBM {
		fmt.Fprintf(f, "SET NAMES utf8mb4;\n")
		fmt.Fprintf(f, "SET CHARACTER SET utf8mb4;\n\n")
	} else if dbType == PostgreSQL || dbType == GaussDBA {
		fmt.Fprintf(f, "SET client_encoding TO 'UTF8';\n\n")
	}

	// 生成每张表的数据
	for _, table := range tables {
		fmt.Fprintf(f, "-- ============================================================\n")
		fmt.Fprintf(f, "-- Table: %s\n", table.Name)
		fmt.Fprintf(f, "-- ============================================================\n")

		// 生成 100-1000 行数据
		numRows := 100 + r.Intn(901)
		fmt.Fprintf(f, "-- Rows: %d\n\n", numRows)

		for i := 0; i < numRows; i++ {
			generateRow(f, table, dbType, i)
		}
		fmt.Fprintln(f)
	}
}

func generateRow(f *os.File, table TableDef, dbType DatabaseType, rowIndex int) {
	values := []string{}

	for _, col := range table.Columns {
		if col.IsPrimaryKey && col.TypeKey == "serial" {
			// 自增主键，跳过（让数据库自动生成）
			continue
		}

		// 10% 概率生成 NULL（但 UNIQUE 列减少 NULL 概率）
		if col.Nullable && !col.IsUnique && r.Intn(10) == 0 {
			values = append(values, "NULL")
			continue
		}

		// UNIQUE 列需要特殊处理，确保唯一性
		if col.IsUnique {
			value := generateUniqueValue(col.TypeKey, dbType, rowIndex)
			values = append(values, value)
			continue
		}

		// 生成测试数据
		value := generateValue(col.TypeKey, dbType, rowIndex)
		values = append(values, value)
	}

	// 构建 INSERT 语句
	colNames := []string{}
	for _, col := range table.Columns {
		if !(col.IsPrimaryKey && col.TypeKey == "serial") {
			colNames = append(colNames, col.Name)
		}
	}

	if len(colNames) > 0 {
		fmt.Fprintf(f, "INSERT INTO %s (%s) VALUES (%s);\n",
			table.Name,
			joinStrings(colNames, ", "),
			joinStrings(values, ", "))
	} else {
		fmt.Fprintf(f, "INSERT INTO %s DEFAULT VALUES;\n", table.Name)
	}
}

func generateUniqueValue(typeKey string, dbType DatabaseType, rowIndex int) string {
	// 为 UNIQUE 列生成唯一值，使用 rowIndex 确保不重复
	switch typeKey {
	case "varchar":
		return fmt.Sprintf("'unique_varchar_%d'", rowIndex)
	case "char":
		return fmt.Sprintf("'unique_char_%d'", rowIndex)
	case "int":
		return fmt.Sprintf("%d", rowIndex*1000+1)
	case "bigint":
		return fmt.Sprintf("%d", int64(rowIndex)*1000000+1)
	case "uuid":
		return generateUuidValue(dbType, rowIndex)
	default:
		return generateValue(typeKey, dbType, rowIndex)
	}
}

func generateValue(typeKey string, dbType DatabaseType, rowIndex int) string {
	switch typeKey {
	case "tinyint":
		return generateTinyIntValue(rowIndex)
	case "smallint":
		return generateSmallIntValue(rowIndex)
	case "mediumint":
		return generateMediumIntValue(rowIndex)
	case "int":
		return generateIntValue(rowIndex)
	case "bigint":
		return generateBigIntValue(rowIndex)
	case "float":
		return generateFloatValue(rowIndex)
	case "double":
		return generateDoubleValue(rowIndex)
	case "decimal", "numeric":
		return generateDecimalValue(rowIndex)
	case "char":
		return generateCharValue(dbType, rowIndex)
	case "varchar":
		return generateVarcharValue(dbType, rowIndex)
	case "text", "mediumtext", "longtext":
		return generateTextValue(dbType, rowIndex)
	case "binary", "varbinary":
		return generateBinaryValue(dbType, rowIndex)
	case "blob":
		return generateBlobValue(dbType, rowIndex)
	case "date":
		return generateDateValue(rowIndex)
	case "time":
		return generateTimeValue(rowIndex)
	case "datetime":
		return generateDatetimeValue(dbType, rowIndex)
	case "timestamp":
		return generateTimestampValue(dbType, rowIndex)
	case "year":
		return generateYearValue(dbType, rowIndex)
	case "boolean":
		return generateBooleanValue(dbType, rowIndex)
	case "json", "jsonb":
		return generateJsonValue(dbType, rowIndex)
	case "enum":
		return generateEnumValue(dbType, rowIndex)
	case "set":
		return generateSetValue(dbType, rowIndex)
	case "uuid":
		return generateUuidValue(dbType, rowIndex)
	case "array":
		return generateArrayValue(dbType, rowIndex)
	default:
		return "NULL"
	}
}

// ============================================================
// 值生成函数
// ============================================================

func generateTinyIntValue(rowIndex int) string {
	// 边界值
	if rowIndex < 5 {
		vals := []string{"-128", "-1", "0", "1", "127"}
		return vals[rowIndex]
	}
	// 重复值测试
	if rowIndex < 15 {
		return fmt.Sprintf("%d", (rowIndex%5))
	}
	// 随机值
	return fmt.Sprintf("%d", r.Intn(256)-128)
}

func generateSmallIntValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"-32768", "-1", "0", "1", "32767"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("%d", (rowIndex%10))
	}
	return fmt.Sprintf("%d", r.Intn(65536)-32768)
}

func generateMediumIntValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"-8388608", "-1", "0", "1", "8388607"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("%d", (rowIndex%10))
	}
	return fmt.Sprintf("%d", r.Intn(16777216)-8388608)
}

func generateIntValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"-2147483648", "-1", "0", "1", "2147483647"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("%d", (rowIndex%10))
	}
	return fmt.Sprintf("%d", r.Intn(2147483647))
}

func generateBigIntValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"-9223372036854775808", "-1", "0", "1", "9223372036854775807"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("%d", (rowIndex%10))
	}
	return fmt.Sprintf("%d", r.Int63n(9223372036854775807))
}

func generateFloatValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"-3.402823466E+38", "-1.0", "0.0", "1.0", "3.402823466E+38"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("%.2f", float64(rowIndex%10)+0.5)
	}
	return fmt.Sprintf("%.2f", r.Float64()*1000-500)
}

func generateDoubleValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"-1.7976931348623157E+308", "-1.0", "0.0", "1.0", "1.7976931348623157E+308"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("%.4f", float64(rowIndex%10)+0.1234)
	}
	return fmt.Sprintf("%.4f", r.Float64()*10000-5000)
}

func generateDecimalValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"-99999999.99", "-1.00", "0.00", "1.00", "99999999.99"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("%.2f", float64(rowIndex%10)+0.5)
	}
	return fmt.Sprintf("%.2f", r.Float64()*100000-50000)
}

func generateCharValue(dbType DatabaseType, rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"''", "'a'", "'特殊!@#'", "'中文测试'", "'A'"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		// 重复值
		return fmt.Sprintf("'char_%d'", rowIndex%3)
	}
	return fmt.Sprintf("'char_%d'", r.Intn(1000))
}

func generateVarcharValue(dbType DatabaseType, rowIndex int) string {
	if rowIndex < 10 {
		vals := []string{
			"''",
			"'a'",
			"'特殊字符!@#$%^&*()'",
			"'中文字符测试'",
			"'日本語テスト'",
			"'한국어 테스트'",
			"'émojis: 🎉🔥💯'",
			"'tab\\there'",
			"'newline\\nhere'",
			"'quote''here'",
		}
		return vals[rowIndex]
	}
	if rowIndex < 20 {
		// 重复值（但避免与 UNIQUE 列冲突）
		return fmt.Sprintf("'varchar_%d'", rowIndex%5)
	}
	// 使用 rowIndex 确保唯一性
	return fmt.Sprintf("'varchar_%d_unique'", rowIndex)
}

func generateTextValue(dbType DatabaseType, rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{
			"''",
			"'Short text'",
			"'这是一段中文文本，用于测试多字节字符的处理'",
			"'Special chars: !@#$%^&*()_+-=[]{}|;:,.<>?'",
			"'Mixed: 中英文 mixed text with émojis 🎉'",
		}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		// 重复值
		return fmt.Sprintf("'text_%d'", rowIndex%5)
	}
	return fmt.Sprintf("'text_%d_with_some_longer_content'", r.Intn(1000))
}

func generateBinaryValue(dbType DatabaseType, rowIndex int) string {
	if dbType == PostgreSQL || dbType == GaussDBA {
		// PostgreSQL 使用 BYTEA
		if rowIndex < 5 {
			return "'\\\\x'"
		}
		return fmt.Sprintf("'\\\\x%02x%02x%02x'", r.Intn(256), r.Intn(256), r.Intn(256))
	}
	// MySQL 使用 HEX
	if rowIndex < 5 {
		return "X''"
	}
	return fmt.Sprintf("X'%02X%02X%02X'", r.Intn(256), r.Intn(256), r.Intn(256))
}

func generateBlobValue(dbType DatabaseType, rowIndex int) string {
	return generateBinaryValue(dbType, rowIndex)
}

func generateDateValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"'1000-01-01'", "'1970-01-01'", "'2000-01-01'", "'2038-01-19'", "'9999-12-31'"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		// 重复值
		return fmt.Sprintf("'2020-%02d-15'", (rowIndex%12)+1)
	}
	year := 2000 + r.Intn(25)
	month := r.Intn(12) + 1
	day := r.Intn(28) + 1
	return fmt.Sprintf("'%04d-%02d-%02d'", year, month, day)
}

func generateTimeValue(rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"'00:00:00'", "'12:00:00'", "'23:59:59'", "'08:30:45'", "'16:45:30'"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("'%02d:00:00'", rowIndex%24)
	}
	hour := r.Intn(24)
	minute := r.Intn(60)
	second := r.Intn(60)
	return fmt.Sprintf("'%02d:%02d:%02d'", hour, minute, second)
}

func generateDatetimeValue(dbType DatabaseType, rowIndex int) string {
	if rowIndex < 5 {
		vals := []string{"'1000-01-01 00:00:00'", "'1970-01-01 00:00:00'", "'2000-01-01 12:00:00'", "'2038-01-19 03:14:07'", "'9999-12-31 23:59:59'"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("'2020-%02d-15 12:00:00'", (rowIndex%12)+1)
	}
	year := 2000 + r.Intn(25)
	month := r.Intn(12) + 1
	day := r.Intn(28) + 1
	hour := r.Intn(24)
	minute := r.Intn(60)
	second := r.Intn(60)
	return fmt.Sprintf("'%04d-%02d-%02d %02d:%02d:%02d'", year, month, day, hour, minute, second)
}

func generateTimestampValue(dbType DatabaseType, rowIndex int) string {
	if dbType == PostgreSQL || dbType == GaussDBA {
		// PostgreSQL TIMESTAMP 范围更广
		if rowIndex < 5 {
			vals := []string{"'1000-01-01 00:00:00'", "'1970-01-01 00:00:00'", "'2000-01-01 12:00:00'", "'2038-01-19 03:14:07'", "'9999-12-31 23:59:59'"}
			return vals[rowIndex]
		}
		if rowIndex < 15 {
			return fmt.Sprintf("'2020-%02d-15 12:00:00'", (rowIndex%12)+1)
		}
		year := 2000 + r.Intn(25)
		month := r.Intn(12) + 1
		day := r.Intn(28) + 1
		hour := r.Intn(24)
		minute := r.Intn(60)
		second := r.Intn(60)
		return fmt.Sprintf("'%04d-%02d-%02d %02d:%02d:%02d'", year, month, day, hour, minute, second)
	}
	// MySQL TIMESTAMP 范围: 1970-01-01 00:00:01 到 2038-01-19 03:14:07
	// 使用时区安全的范围: 1970-01-02 到 2038-01-18
	if rowIndex < 5 {
		vals := []string{"'1970-01-02 00:00:00'", "'1970-01-01 12:00:00'", "'2000-01-01 12:00:00'", "'2038-01-18 03:14:07'", "'2020-01-01 00:00:00'"}
		return vals[rowIndex]
	}
	if rowIndex < 15 {
		return fmt.Sprintf("'2020-%02d-15 12:00:00'", (rowIndex%12)+1)
	}
	year := 1970 + r.Intn(68)  // 1970-2037
	month := r.Intn(12) + 1
	day := r.Intn(28) + 1
	hour := r.Intn(24)
	minute := r.Intn(60)
	second := r.Intn(60)
	return fmt.Sprintf("'%04d-%02d-%02d %02d:%02d:%02d'", year, month, day, hour, minute, second)
}

func generateYearValue(dbType DatabaseType, rowIndex int) string {
	if dbType == PostgreSQL || dbType == GaussDBA {
		// PostgreSQL 使用 SMALLINT
		if rowIndex < 5 {
			vals := []string{"1901", "1970", "2000", "2023", "2155"}
			return vals[rowIndex]
		}
		return fmt.Sprintf("%d", 1901+r.Intn(255))
	}
	// MySQL YEAR 类型
	if rowIndex < 5 {
		vals := []string{"1901", "1970", "2000", "2023", "2155"}
		return vals[rowIndex]
	}
	return fmt.Sprintf("%d", 1901+r.Intn(255))
}

func generateBooleanValue(dbType DatabaseType, rowIndex int) string {
	if rowIndex < 5 {
		if dbType == PostgreSQL || dbType == GaussDBA {
			vals := []string{"FALSE", "TRUE", "FALSE", "TRUE", "FALSE"}
			return vals[rowIndex]
		}
		vals := []string{"0", "1", "0", "1", "0"}
		return vals[rowIndex]
	}
	if r.Intn(2) == 0 {
		if dbType == PostgreSQL || dbType == GaussDBA {
			return "FALSE"
		}
		return "0"
	}
	if dbType == PostgreSQL || dbType == GaussDBA {
		return "TRUE"
	}
	return "1"
}

func generateJsonValue(dbType DatabaseType, rowIndex int) string {
	if rowIndex < 10 {
		vals := []string{
			"'{}'",
			"'{\"key\": \"value\"}'",
			"'{\"number\": 123}'",
			"'{\"bool\": true}'",
			"'{\"null\": null}'",
			"'{\"array\": [1,2,3]}'",
			"'{\"nested\": {\"key\": \"value\"}}'",
			"'{\"special\": \"!@#$%\"}'",
			"'{\"chinese\": \"中文\"}'",
			"'{\"emoji\": \"🎉🔥\"}'",
		}
		return vals[rowIndex]
	}
	if rowIndex < 20 {
		// 重复值
		return fmt.Sprintf("'{\"id\": %d}'", rowIndex%5)
	}
	return fmt.Sprintf("'{\"id\": %d, \"value\": %d}'", r.Intn(1000), r.Intn(10000))
}

func generateEnumValue(dbType DatabaseType, rowIndex int) string {
	if dbType == PostgreSQL || dbType == GaussDBA {
		// PostgreSQL 使用 VARCHAR
		vals := []string{"'A'", "'B'", "'C'", "'D'", "'SPECIAL'"}
		if rowIndex < 5 {
			return vals[rowIndex]
		}
		return vals[r.Intn(len(vals))]
	}
	// MySQL ENUM
	vals := []string{"'A'", "'B'", "'C'", "'D'", "'SPECIAL'"}
	if rowIndex < 5 {
		return vals[rowIndex]
	}
	return vals[r.Intn(len(vals))]
}

func generateSetValue(dbType DatabaseType, rowIndex int) string {
	if dbType == PostgreSQL || dbType == GaussDBA {
		// PostgreSQL 使用 TEXT
		vals := []string{"''", "'read'", "'write'", "'read,write'", "'execute'"}
		if rowIndex < 5 {
			return vals[rowIndex]
		}
		return vals[r.Intn(len(vals))]
	}
	// MySQL SET
	vals := []string{"''", "'read'", "'write'", "'read,write'", "'execute'"}
	if rowIndex < 5 {
		return vals[rowIndex]
	}
	return vals[r.Intn(len(vals))]
}

func generateUuidValue(dbType DatabaseType, rowIndex int) string {
	if dbType == PostgreSQL || dbType == GaussDBA {
		// PostgreSQL UUID
		if rowIndex < 5 {
			vals := []string{
				"'00000000-0000-0000-0000-000000000000'",
				"'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'",
				"'ffffffff-ffff-ffff-ffff-ffffffffffff'",
				"'12345678-1234-1234-1234-123456789012'",
				"'87654321-4321-4321-4321-210987654321'",
			}
			return vals[rowIndex]
		}
		return fmt.Sprintf("'%08x-%04x-%04x-%04x-%012x'",
			r.Uint32(), r.Intn(65536), r.Intn(65536), r.Intn(65536), r.Int63n(281474976710656))
	}
	// MySQL CHAR(36)
	if rowIndex < 5 {
		vals := []string{
			"'00000000-0000-0000-0000-000000000000'",
			"'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'",
			"'ffffffff-ffff-ffff-ffff-ffffffffffff'",
			"'12345678-1234-1234-1234-123456789012'",
			"'87654321-4321-4321-4321-210987654321'",
		}
		return vals[rowIndex]
	}
	return fmt.Sprintf("'%08x-%04x-%04x-%04x-%012x'",
		r.Uint32(), r.Intn(65536), r.Intn(65536), r.Intn(65536), r.Int63n(281474976710656))
}

func generateArrayValue(dbType DatabaseType, rowIndex int) string {
	if dbType == PostgreSQL || dbType == GaussDBA {
		// PostgreSQL TEXT[]
		if rowIndex < 5 {
			vals := []string{"'{}'", "'{a}'", "'{a,b}'", "'{a,b,c}'", "'{特殊,中文,English}'"}
			return vals[rowIndex]
		}
		return fmt.Sprintf("'{{item%d,item%d}}'", r.Intn(100), r.Intn(100))
	}
	// MySQL 使用 TEXT
	if rowIndex < 5 {
		vals := []string{"'[]'", "'[\"a\"]'", "'[\"a\",\"b\"]'", "'[\"a\",\"b\",\"c\"]'", "'[\"特殊\",\"中文\",\"English\"]'"}
		return vals[rowIndex]
	}
	return fmt.Sprintf("'[\"item%d\",\"item%d\"]'", r.Intn(100), r.Intn(100))
}

// ============================================================
// 辅助函数
// ============================================================

func getTypeString(typeKey string, dbType DatabaseType) string {
	mapping, ok := typeMappings[typeKey]
	if !ok {
		return "TEXT"
	}

	switch dbType {
	case MySQL:
		return mapping.MySQL
	case PostgreSQL:
		return mapping.PostgreSQL
	case GaussDBM:
		return mapping.GaussDBM
	case GaussDBA:
		return mapping.GaussDBA
	default:
		return mapping.MySQL
	}
}

func getIndexString(indexType string, dbType DatabaseType) string {
	mapping, ok := indexMappings[indexType]
	if !ok {
		return "INDEX"
	}

	switch dbType {
	case MySQL:
		return mapping.MySQL
	case PostgreSQL:
		return mapping.PostgreSQL
	case GaussDBM:
		return mapping.GaussDBM
	case GaussDBA:
		return mapping.GaussDBA
	default:
		return mapping.MySQL
	}
}

func hasIndexes(table TableDef) bool {
	for _, col := range table.Columns {
		if col.IsPrimaryKey || col.IsUnique || col.IsIndex {
			return true
		}
	}
	return false
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
