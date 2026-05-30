// Package testsqls: sql benchmark for testing.
// Prepare (set environment variables before running tests):
//   export TEST_DB_PASSWORD=your_password
//   export TEST_DB_PORT_MYSQL=13306        # default: 13306
//   export TEST_DB_PORT_MARIADB=23306      # default: 23306
//   export TEST_DB_PORT_TIDB=4000          # default: 4000
//   export TEST_DB_PORT_OCEANBASE=2881     # default: 2881
//
// Or start databases with Docker:
//   docker run -itd --name mysqltest -p 13306:3306 -e MYSQL_ROOT_PASSWORD=your_password mysql:8.0.30
//   docker run -itd --name mariadbtest -p 23306:3306 -e MYSQL_ROOT_PASSWORD=your_password mariadb:11.4
//   docker run -itd --name tidbtest -p 4000:4000 pingcap/tidb:v6.5.0
//   + SET PASSWORD = 'your_password';
//   docker run -itd --name oceanbasetest -p 2881:2881 oceanbase/oceanbase-ce:4.2.1
//   + SET PASSWORD = PASSWORD('your_password');
// We will use database TEST for testing.
// Make sure there is no important data in TEST, as we will automatically clear it.
package testsqls