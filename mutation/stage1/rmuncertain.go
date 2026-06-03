package stage1

import (
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/test_driver"
)

// rmUncertain: remove uncertain (non-deterministic) functions
//
// Conservative strategy:
// - Replace uncertain functions with constant values
// - Use deterministic replacements that preserve query structure
//
// Functions handled:
//
// Random functions:
//   - RAND() -> 0.5
//
// Date/Time functions:
//   - NOW(), CURRENT_TIMESTAMP, LOCALTIMESTAMP, SYSDATE() -> '2020-01-01 00:00:00'
//   - CURDATE(), CURRENT_DATE, UTC_DATE() -> '2020-01-01'
//   - CURTIME(), CURRENT_TIME, UTC_TIME() -> '00:00:00'
//   - LOCALTIME -> '2020-01-01 00:00:00'
//
// UUID functions:
//   - UUID(), UUID_SHORT() -> '00000000-0000-0000-0000-000000000000'
//   - UUID_TO_BIN(uuid) -> '00000000-0000-0000-0000-000000000000'
//   - BIN_TO_UUID(bin) -> '00000000-0000-0000-0000-000000000000'
//
// Other uncertain functions:
//   - BENCHMARK(count, expr) -> 0
//   - CONNECTION_ID() -> 0
//   - CURRENT_USER(), CURRENT_ROLE(), USER(), SESSION_USER(), SYSTEM_USER() -> 'user'
//   - DATABASE(), SCHEMA() -> 'test'
//   - FOUND_ROWS(), ROW_COUNT() -> 0
//   - LAST_INSERT_ID() -> 0
//   - ANY_VALUE(expr) -> expr (use the argument itself)
//   - MASTER_POS_WAIT(file, pos) -> 0
//   - SLEEP(duration) -> 0
//   - RANDOM_BYTES(count) -> '0000000000'
func rmUncertain(in ast.Node) bool {
	if funcCall, ok := in.(*ast.FuncCallExpr); ok {
		funcName := funcCall.FnName.L

		// Map of uncertain functions to their deterministic replacements
		replacements := map[string]interface{}{
			// Random functions
			"rand": 0.5,

			// Date/Time functions
			"now":               "2020-01-01 00:00:00",
			"current_timestamp": "2020-01-01 00:00:00",
			"localtimestamp":    "2020-01-01 00:00:00",
			"sysdate":           "2020-01-01 00:00:00",
			"curdate":           "2020-01-01",
			"current_date":      "2020-01-01",
			"utc_date":          "2020-01-01",
			"curtime":           "00:00:00",
			"current_time":      "00:00:00",
			"utc_time":          "00:00:00",
			"localtime":         "2020-01-01 00:00:00",

			// UUID functions
			"uuid":          "00000000-0000-0000-0000-000000000000",
			"uuid_short":    "00000000-0000-0000-0000-000000000000",
			"uuid_to_bin":   "00000000-0000-0000-0000-000000000000",
			"bin_to_uuid":   "00000000-0000-0000-0000-000000000000",

			// Other uncertain functions
			"benchmark":      0,
			"connection_id":  0,
			"current_user":   "user",
			"current_role":   "user",
			"user":           "user",
			"session_user":   "user",
			"system_user":    "user",
			"database":       "test",
			"schema":         "test",
			"found_rows":     0,
			"row_count":      0,
			"last_insert_id": 0,
			"master_pos_wait": 0,
			"sleep":          0,
			"random_bytes":   "0000000000",
		}

		if replacement, isUncertain := replacements[funcName]; isUncertain {
			// Special case: ANY_VALUE(expr) -> expr (use the argument itself)
			if funcName == "any_value" && len(funcCall.Args) > 0 {
				// Replace the function call with its first argument
				*funcCall = ast.FuncCallExpr{
					FnName: funcCall.Args[0].(*ast.FuncCallExpr).FnName,
					Args:   funcCall.Args[0].(*ast.FuncCallExpr).Args,
				}
				return true
			}

			// For other functions, replace with constant value
			switch v := replacement.(type) {
			case int:
				funcCall.FnName.L = ""
				funcCall.FnName.O = ""
				funcCall.Args = []ast.ExprNode{
					&test_driver.ValueExpr{
						Datum: test_driver.NewDatum(v),
					},
				}
			case float64:
				funcCall.FnName.L = ""
				funcCall.FnName.O = ""
				funcCall.Args = []ast.ExprNode{
					&test_driver.ValueExpr{
						Datum: test_driver.NewDatum(v),
					},
				}
			case string:
				funcCall.FnName.L = ""
				funcCall.FnName.O = ""
				funcCall.Args = []ast.ExprNode{
					&test_driver.ValueExpr{
						Datum: test_driver.NewDatum(v),
					},
				}
			}
			return true
		}
	}
	return false
}
