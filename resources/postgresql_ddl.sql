-- PostgreSQL Test DDL
-- Simple test tables for mutation testing

DROP TABLE IF EXISTS company CASCADE;
DROP TABLE IF EXISTS employee CASCADE;
DROP TABLE IF EXISTS department CASCADE;

CREATE TABLE company (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    age INTEGER,
    salary DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE department (
    id SERIAL PRIMARY KEY,
    dept_name TEXT NOT NULL,
    location TEXT
);

CREATE TABLE employee (
    id SERIAL PRIMARY KEY,
    emp_name TEXT NOT NULL,
    company_id INTEGER REFERENCES company(id),
    dept_id INTEGER REFERENCES department(id),
    hire_date DATE,
    status TEXT DEFAULT 'active'
);

-- Create indexes
CREATE INDEX idx_company_age ON company(age);
CREATE INDEX idx_company_salary ON company(salary);
CREATE INDEX idx_employee_company ON employee(company_id);

-- Insert test data
INSERT INTO company (name, age, salary) VALUES
    ('Alice', 25, 50000.00),
    ('Bob', 30, 60000.00),
    ('Charlie', 35, 70000.00),
    ('David', 28, 55000.00),
    ('Eve', 32, 65000.00);

INSERT INTO department (dept_name, location) VALUES
    ('Engineering', 'Building A'),
    ('Sales', 'Building B'),
    ('HR', 'Building C');

INSERT INTO employee (emp_name, company_id, dept_id, hire_date, status) VALUES
    ('John', 1, 1, '2020-01-15', 'active'),
    ('Jane', 2, 2, '2021-03-20', 'active'),
    ('Mike', 3, 1, '2019-07-10', 'inactive');