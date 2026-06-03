-- Multi-database test data DDL
-- Database: postgresql
-- Generated: 2026-06-03 15:18:18

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
    id SERIAL NOT NULL,
    c_tiny SMALLINT,
    c_small SMALLINT,
    c_medium INTEGER,
    c_int INTEGER,
    c_big BIGINT,
    PRIMARY KEY (id)
);

CREATE INDEX idx_t_int_types_c_tiny ON t_int_types (c_tiny);
CREATE INDEX idx_t_int_types_c_small ON t_int_types (c_small);
CREATE INDEX idx_t_int_types_c_int ON t_int_types (c_int);
CREATE INDEX idx_t_int_types_c_big ON t_int_types (c_big);

CREATE TABLE t_float_types (
    id SERIAL NOT NULL,
    c_float REAL,
    c_double DOUBLE PRECISION,
    c_decimal DECIMAL(10,2),
    c_numeric NUMERIC(10,2),
    PRIMARY KEY (id)
);

CREATE INDEX idx_t_float_types_c_float ON t_float_types (c_float);
CREATE INDEX idx_t_float_types_c_double ON t_float_types (c_double);
CREATE INDEX idx_t_float_types_c_decimal ON t_float_types (c_decimal);

CREATE TABLE t_string_types (
    id SERIAL NOT NULL,
    c_char CHAR(50),
    c_varchar VARCHAR(100),
    c_text TEXT,
    c_mediumtext TEXT,
    c_longtext TEXT,
    PRIMARY KEY (id)
);

CREATE INDEX idx_t_string_types_c_char ON t_string_types (c_char);
CREATE INDEX idx_t_string_types_c_varchar ON t_string_types (c_varchar);
CREATE INDEX idx_t_string_types_c_text ON t_string_types USING gin (to_tsvector('english', c_text));

CREATE TABLE t_binary_types (
    id SERIAL NOT NULL,
    c_binary BYTEA,
    c_varbinary BYTEA,
    c_blob BYTEA,
    PRIMARY KEY (id)
);


CREATE TABLE t_datetime_types (
    id SERIAL NOT NULL,
    c_date DATE,
    c_time TIME,
    c_datetime TIMESTAMP,
    c_timestamp TIMESTAMP WITH TIME ZONE,
    c_year SMALLINT,
    PRIMARY KEY (id)
);

CREATE INDEX idx_t_datetime_types_c_date ON t_datetime_types (c_date);
CREATE INDEX idx_t_datetime_types_c_datetime ON t_datetime_types (c_datetime);
CREATE INDEX idx_t_datetime_types_c_timestamp ON t_datetime_types (c_timestamp);

CREATE TABLE t_boolean_types (
    id SERIAL NOT NULL,
    c_bool BOOLEAN,
    c_flag INTEGER,
    PRIMARY KEY (id)
);

CREATE INDEX idx_t_boolean_types_c_bool ON t_boolean_types (c_bool);
CREATE INDEX idx_t_boolean_types_c_flag ON t_boolean_types (c_flag);

CREATE TABLE t_json_types (
    id SERIAL NOT NULL,
    c_json JSON,
    c_jsonb JSONB,
    PRIMARY KEY (id)
);


CREATE TABLE t_special_types (
    id SERIAL NOT NULL,
    c_enum VARCHAR(50),
    c_set TEXT,
    c_uuid UUID,
    c_array TEXT[],
    PRIMARY KEY (id),
    UNIQUE (c_uuid)
);

CREATE INDEX idx_t_special_types_c_enum ON t_special_types (c_enum);

CREATE TABLE t_index_test (
    id SERIAL NOT NULL,
    c_unique VARCHAR(100),
    c_btree VARCHAR(100),
    c_fulltext TEXT,
    c_composite1 INTEGER,
    c_composite2 VARCHAR(100),
    PRIMARY KEY (id),
    UNIQUE (c_unique)
);

CREATE INDEX idx_t_index_test_c_btree ON t_index_test (c_btree);
CREATE INDEX idx_t_index_test_c_fulltext ON t_index_test USING gin (to_tsvector('english', c_fulltext));
CREATE INDEX idx_composite ON t_index_test (c_composite1, c_composite2);

