module github.com/qaqcatz/impomysql

go 1.25.0

require (
	gitee.com/opengauss/openGauss-connector-go-pq v0.0.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/jackc/pgx/v5 v5.9.2
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/pganalyze/pg_query_go/v6 v6.0.0
	github.com/pingcap/tidb/parser v0.0.0-20220627062839-d6be9105e6c4
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.9.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/pingcap/errors v0.11.5-0.20210425183316-da1aaba5fb63 // indirect
	github.com/pingcap/log v0.0.0-20210625125904-98ed8e2eb1c7 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.18.1 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace github.com/pganalyze/pg_query_go/v6 v6.0.0 => ./third_party/pg_query_go

replace gitee.com/opengauss/openGauss-connector-go-pq v0.0.0 => ./third_party/openGauss-connector-go-pq
