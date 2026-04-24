-- openGauss M mode test DML (SELECT statements)
-- Uses MySQL-style syntax

-- Basic SELECT
SELECT * FROM company WHERE age > 25;
SELECT id, name FROM company WHERE salary >= 5000;

-- Comparison operators
SELECT * FROM company WHERE salary > 5000 AND salary < 10000;
SELECT * FROM company WHERE age = 25 OR age = 30;

-- IN clause
SELECT * FROM company WHERE id IN (1, 2, 3);
SELECT * FROM company WHERE city NOT IN ('Beijing');

-- LIKE pattern
SELECT * FROM company WHERE name LIKE 'A%';
SELECT * FROM company WHERE name LIKE '_a%';

-- BETWEEN
SELECT * FROM orders WHERE amount BETWEEN 100 AND 500;

-- NULL handling
SELECT * FROM orders WHERE customer_id IS NULL;
SELECT * FROM orders WHERE customer_id IS NOT NULL;

-- Subqueries
SELECT * FROM company WHERE id = (SELECT MAX(id) FROM company);
SELECT * FROM orders WHERE customer_id IN (SELECT id FROM company WHERE age > 25);

-- JOIN
SELECT c.name, o.amount FROM company c JOIN orders o ON c.id = o.customer_id;

-- UNION
SELECT id FROM company UNION ALL SELECT customer_id FROM orders WHERE customer_id IS NOT NULL;

-- DISTINCT
SELECT DISTINCT city FROM company;

-- Simple queries
SELECT 1;
SELECT 'test';
SELECT NULL;