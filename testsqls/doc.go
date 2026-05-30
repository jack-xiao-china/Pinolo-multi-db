// Package testsqls: sql benchmark for testing.
// Prepare (set environment variables before running tests):
//   export TEST_DB_PASSWORD=your_password
//   export TEST_DB_PORT_MYSQL=13306        # default: 13306
//   export TEST_DB_PORT_MARIADB=23306      # default: 23306
//   export TEST_DB_PORT_TIDB=4000          # default: 4000
//   export TEST_DB_PORT_OCEANBASE=2881     # default: 2881
//
// Database environment must be provided by the user.
// We will use database TEST for testing.
// Make sure there is no important data in TEST, as we will automatically clear it.
package testsqls