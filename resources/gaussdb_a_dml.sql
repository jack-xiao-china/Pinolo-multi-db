-- GaussDB A mode test DML (SELECT statements)
-- Uses Oracle-style syntax including ROWNUM, (+) outer join

-- Basic SELECT (similar to MySQL/M mode)
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

-- NULL handling
SELECT * FROM orders WHERE customer_id IS NULL;
SELECT * FROM orders WHERE customer_id IS NOT NULL;

-- Subqueries
SELECT * FROM company WHERE id = (SELECT MAX(id) FROM company);
SELECT * FROM orders WHERE customer_id IN (SELECT id FROM company WHERE age > 25);

-- JOIN (standard syntax)
SELECT c.name, o.amount FROM company c JOIN orders o ON c.id = o.customer_id;

-- UNION
SELECT id FROM company UNION ALL SELECT customer_id FROM orders WHERE customer_id IS NOT NULL;

-- DISTINCT
SELECT DISTINCT city FROM company;

-- Oracle-specific: ROWNUM
SELECT * FROM company WHERE ROWNUM <= 3;
SELECT id, name FROM company WHERE age > 20 AND ROWNUM <= 2;

-- Oracle-specific: (+) outer join syntax
-- Note: This syntax will be preprocessed to LEFT JOIN
SELECT c.name, o.amount FROM company c, orders o WHERE c.id = o.customer_id(+);

-- Oracle-specific: NVL function
SELECT id, name, NVL(city, 'Unknown') as city FROM company WHERE id <= 5;

-- Oracle-specific: DUAL table
SELECT 1 FROM DUAL;
SELECT 'test' FROM DUAL;
SELECT SYSDATE FROM DUAL;

-- Oracle-specific: DECODE function
SELECT id, name, DECODE(status, 'completed', 'Done', 'pending', 'Waiting', 'Other') as status_desc
FROM orders WHERE order_id <= 4;

-- Simple queries
SELECT 1;
SELECT 'test';
SELECT NULL;