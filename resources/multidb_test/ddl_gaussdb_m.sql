-- Multi-database test data DDL
-- Database: gaussdb_m
-- Generated: 2026-06-03 14:58:13

DROP TABLE IF EXISTS t_int_types;
DROP TABLE IF EXISTS t_float_types;
DROP TABLE IF EXISTS t_string_types;
DROP TABLE IF EXISTS t_binary_types;
DROP TABLE IF EXISTS t_datetime_types;
DROP TABLE IF EXISTS t_boolean_types;
DROP TABLE IF EXISTS t_json_types;
DROP TABLE IF EXISTS t_special_types;
DROP TABLE IF EXISTS t_index_test;

CREATE TABLE t_int_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_tiny TINYINT,
    c_small SMALLINT,
    c_medium MEDIUMINT,
    c_int INT,
    c_big BIGINT,
    PRIMARY KEY (id),
    INDEX idx_t_int_types_c_tiny (c_tiny),
    INDEX idx_t_int_types_c_small (c_small),
    INDEX idx_t_int_types_c_int (c_int),
    INDEX idx_t_int_types_c_big (c_big)
);

CREATE TABLE t_float_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_float FLOAT,
    c_double DOUBLE,
    c_decimal DECIMAL(10,2),
    c_numeric NUMERIC(10,2),
    PRIMARY KEY (id),
    INDEX idx_t_float_types_c_float (c_float),
    INDEX idx_t_float_types_c_double (c_double),
    INDEX idx_t_float_types_c_decimal (c_decimal)
);

CREATE TABLE t_string_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_char CHAR(50),
    c_varchar VARCHAR(100),
    c_text TEXT,
    c_mediumtext MEDIUMTEXT,
    c_longtext LONGTEXT,
    PRIMARY KEY (id),
    INDEX idx_t_string_types_c_char (c_char),
    INDEX idx_t_string_types_c_varchar (c_varchar),
    FULLTEXT INDEX idx_t_string_types_c_text (c_text)
);

CREATE TABLE t_binary_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_binary BINARY(20),
    c_varbinary VARBINARY(100),
    c_blob BLOB,
    PRIMARY KEY (id)
);

CREATE TABLE t_datetime_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_date DATE,
    c_time TIME,
    c_datetime DATETIME,
    c_timestamp TIMESTAMP,
    c_year YEAR,
    PRIMARY KEY (id),
    INDEX idx_t_datetime_types_c_date (c_date),
    INDEX idx_t_datetime_types_c_datetime (c_datetime),
    INDEX idx_t_datetime_types_c_timestamp (c_timestamp)
);

CREATE TABLE t_boolean_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_bool BOOLEAN,
    c_flag INT,
    PRIMARY KEY (id),
    INDEX idx_t_boolean_types_c_bool (c_bool),
    INDEX idx_t_boolean_types_c_flag (c_flag)
);

CREATE TABLE t_json_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_json JSON,
    c_jsonb JSON,
    PRIMARY KEY (id)
);

CREATE TABLE t_special_types (
    id INT AUTO_INCREMENT NOT NULL,
    c_enum ENUM('A','B','C','D','SPECIAL'),
    c_set SET('read','write','execute','delete'),
    c_uuid CHAR(36),
    c_array TEXT,
    PRIMARY KEY (id),
    INDEX idx_t_special_types_c_enum (c_enum),
    UNIQUE (c_uuid)
);

CREATE TABLE t_index_test (
    id INT AUTO_INCREMENT NOT NULL,
    c_unique VARCHAR(100),
    c_btree VARCHAR(100),
    c_fulltext TEXT,
    c_composite1 INT,
    c_composite2 VARCHAR(100),
    PRIMARY KEY (id),
    UNIQUE (c_unique),
    INDEX idx_t_index_test_c_btree (c_btree),
    FULLTEXT INDEX idx_t_index_test_c_fulltext (c_fulltext),
    INDEX idx_composite (c_composite1, c_composite2)
);

