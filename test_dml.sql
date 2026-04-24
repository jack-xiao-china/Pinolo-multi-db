-- Test DML queries for PINOLO
-- Basic SELECT with WHERE
SELECT * FROM company WHERE age > 25;

-- Comparison operators
SELECT id, name FROM company WHERE salary >= 5000 AND salary <= 10000;
SELECT * FROM company WHERE age < 30 OR city = 'Beijing';

-- IN clause
SELECT * FROM company WHERE id IN (1, 2, 3, 4);
SELECT * FROM company WHERE city NOT IN ('Beijing', 'Shanghai');

-- LIKE pattern
SELECT * FROM company WHERE name LIKE 'A%';
SELECT * FROM company WHERE name LIKE '_a%';

-- BETWEEN
SELECT * FROM orders WHERE amount BETWEEN 100 AND 500;

-- NULL handling
SELECT * FROM company WHERE city IS NULL;
SELECT * FROM orders WHERE customer_id IS NOT NULL;

-- Subqueries
SELECT * FROM company WHERE id = (SELECT MAX(id) FROM company);
SELECT * FROM orders WHERE customer_id IN (SELECT id FROM company WHERE age > 25);

-- UNION
SELECT id, name FROM company WHERE age > 25
UNION ALL
SELECT customer_id, 'order' FROM orders WHERE amount > 300;

-- JOIN
SELECT c.name, o.amount
FROM company c
JOIN orders o ON c.id = o.customer_id
WHERE o.status = 'completed';

-- DISTINCT
SELECT DISTINCT city FROM company WHERE age > 20;

-- ORDER BY (note: will be removed by stage1)
SELECT id, name FROM company ORDER BY age DESC LIMIT 10;

-- HAVING
SELECT city, COUNT(*) as cnt FROM company GROUP BY city HAVING cnt > 1;

-- Aggregate (note: will be removed by stage1)
SELECT city, AVG(salary) FROM company GROUP BY city;

-- Simple value queries
SELECT 1;
SELECT 'test';
SELECT NULL;
SELECT 1 + 2;