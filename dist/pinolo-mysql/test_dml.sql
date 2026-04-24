-- PINOLO Test DML
-- SELECT statements for logic bug detection

-- ============================================
-- Basic SELECT with WHERE conditions
-- ============================================
SELECT * FROM company WHERE age > 25;
SELECT id, name FROM company WHERE salary >= 5000 AND salary <= 10000;
SELECT * FROM company WHERE age < 30 OR city = 'Beijing';
SELECT * FROM company WHERE NOT (age > 30);
SELECT * FROM company WHERE active = TRUE;

-- ============================================
-- Comparison operators
-- ============================================
SELECT * FROM company WHERE salary > 5000;
SELECT * FROM company WHERE salary >= 5000;
SELECT * FROM company WHERE salary < 10000;
SELECT * FROM company WHERE salary <= 10000;
SELECT * FROM company WHERE age = 25;
SELECT * FROM company WHERE age != 30;
SELECT * FROM company WHERE id <> 5;

-- ============================================
-- IN clause
-- ============================================
SELECT * FROM company WHERE id IN (1, 2, 3, 4);
SELECT * FROM company WHERE city IN ('Beijing', 'Shanghai');
SELECT * FROM company WHERE id NOT IN (5, 6, 7);
SELECT * FROM company WHERE age IN (25, 30, 35);
SELECT 1 IN (1, 2, 3);
SELECT NULL IN (1, 2, 3);

-- ============================================
-- LIKE pattern matching
-- ============================================
SELECT * FROM company WHERE name LIKE 'A%';
SELECT * FROM company WHERE name LIKE '%e';
SELECT * FROM company WHERE name LIKE '_a%';
SELECT * FROM company WHERE name LIKE '%li%';
SELECT * FROM company WHERE name NOT LIKE 'B%';
SELECT * FROM company WHERE name LIKE 'A__%';

-- ============================================
-- BETWEEN
-- ============================================
SELECT * FROM company WHERE age BETWEEN 25 AND 35;
SELECT * FROM company WHERE salary BETWEEN 5000 AND 10000;
SELECT * FROM company WHERE id NOT BETWEEN 3 AND 7;
SELECT * FROM orders WHERE amount BETWEEN 100 AND 500;

-- ============================================
-- NULL handling
-- ============================================
SELECT * FROM company WHERE city IS NULL;
SELECT * FROM company WHERE city IS NOT NULL;
SELECT * FROM orders WHERE customer_id IS NULL;
SELECT * FROM orders WHERE customer_id IS NOT NULL;
SELECT * FROM company WHERE salary IS NULL OR age IS NULL;

-- ============================================
-- Subqueries
-- ============================================
SELECT * FROM company WHERE id = (SELECT MAX(id) FROM company);
SELECT * FROM company WHERE id > (SELECT AVG(id) FROM company);
SELECT * FROM company WHERE age > ALL (SELECT age FROM company WHERE city = 'Shanghai');
SELECT * FROM company WHERE age > ANY (SELECT age FROM company WHERE city = 'Beijing');
SELECT * FROM company WHERE id IN (SELECT customer_id FROM orders WHERE status = 'completed');
SELECT * FROM company WHERE EXISTS (SELECT 1 FROM orders WHERE orders.customer_id = company.id);
SELECT * FROM company WHERE NOT EXISTS (SELECT 1 FROM orders WHERE orders.customer_id = company.id AND orders.amount > 500);

-- ============================================
-- JOIN operations
-- ============================================
SELECT c.name, o.amount FROM company c JOIN orders o ON c.id = o.customer_id;
SELECT c.name, o.amount FROM company c JOIN orders o ON c.id = o.customer_id WHERE o.status = 'completed';
SELECT c.name, p.name, o.amount FROM company c JOIN orders o ON c.id = o.customer_id JOIN products p ON o.product_id = p.product_id;
SELECT e1.name, e2.name AS manager FROM employees e1 JOIN employees e2 ON e1.manager_id = e2.emp_id;

-- ============================================
-- UNION
-- ============================================
SELECT id, name FROM company WHERE age > 30 UNION ALL SELECT customer_id, 'order' FROM orders WHERE amount > 300;
SELECT city FROM company UNION SELECT 'Unknown' FROM company WHERE city IS NULL;
SELECT id FROM company WHERE age > 25 UNION SELECT customer_id FROM orders WHERE status = 'completed';

-- ============================================
-- DISTINCT
-- ============================================
SELECT DISTINCT city FROM company;
SELECT DISTINCT age FROM company WHERE age > 25;
SELECT DISTINCT city, age FROM company;

-- ============================================
-- HAVING clause (will be processed by stage1)
-- ============================================
SELECT city, COUNT(*) as cnt FROM company GROUP BY city HAVING cnt > 1;
SELECT department, AVG(salary) FROM employees GROUP BY department HAVING AVG(salary) > 5000;

-- ============================================
-- WITH clause (CTE)
-- ============================================
WITH active_users AS (SELECT * FROM company WHERE active = TRUE) SELECT * FROM active_users WHERE age > 25;
WITH high_salary AS (SELECT id, name, salary FROM company WHERE salary > 7000) SELECT * FROM high_salary WHERE age < 40;
WITH beijing_users AS (SELECT id, name FROM company WHERE city = 'Beijing') SELECT b.name, o.amount FROM beijing_users b JOIN orders o ON b.id = o.customer_id;

-- ============================================
-- Boolean expressions
-- ============================================
SELECT * FROM company WHERE (age > 25 AND salary > 5000) OR city = 'Beijing';
SELECT * FROM company WHERE age > 25 AND (salary > 5000 OR city = 'Shanghai');
SELECT * FROM company WHERE NOT (age > 30 AND salary < 8000);
SELECT * FROM company WHERE (age > 20) IS TRUE;
SELECT * FROM company WHERE (salary > 10000) IS FALSE;
SELECT * FROM company WHERE (city IS NULL) IS NOT TRUE;

-- ============================================
-- Simple value queries
-- ============================================
SELECT 1;
SELECT 'test';
SELECT NULL;
SELECT 1 + 2;
SELECT 10 - 5;
SELECT 3 * 4;
SELECT 100 / 10;
SELECT TRUE;
SELECT FALSE;

-- ============================================
-- Parentheses
-- ============================================
SELECT * FROM company WHERE ((age > 25) AND (salary > 5000));
SELECT * FROM company WHERE (age > 25) OR ((salary > 8000) AND (city = 'Beijing'));
SELECT * FROM company WHERE NOT ((age > 30) OR (salary < 5000));

-- ============================================
-- Order comparisons with expressions
-- ============================================
SELECT * FROM company WHERE id + 1 > 5;
SELECT * FROM company WHERE salary * 12 > 50000;
SELECT * FROM company WHERE age - 5 < 20;