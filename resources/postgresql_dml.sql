-- PostgreSQL Test DML (SELECT statements for mutation testing)

-- Basic SELECT with WHERE
SELECT * FROM company WHERE age > 25;

-- SELECT with comparison operators
SELECT name, salary FROM company WHERE salary >= 55000 ORDER BY salary DESC;

-- SELECT with IN clause
SELECT * FROM company WHERE age IN (25, 30, 35);

-- SELECT with LIKE
SELECT * FROM company WHERE name LIKE 'A%';

-- SELECT with DISTINCT
SELECT DISTINCT age FROM company;

-- SELECT with JOIN
SELECT c.name, e.emp_name FROM company c JOIN employee e ON c.id = e.company_id;

-- SELECT with LEFT JOIN (will be converted to INNER JOIN in Stage1)
SELECT c.name, d.dept_name FROM company c LEFT JOIN department d ON c.id = d.id;

-- SELECT with multiple conditions
SELECT * FROM company WHERE age > 25 AND salary < 70000;

-- SELECT with UNION
SELECT name FROM company WHERE age > 30 UNION SELECT emp_name FROM employee WHERE status = 'active';

-- SELECT with subquery
SELECT * FROM company WHERE id IN (SELECT company_id FROM employee);

-- SELECT with GROUP BY (will be removed in Stage1)
SELECT age, COUNT(*) FROM company GROUP BY age;

-- SELECT with HAVING
SELECT age, AVG(salary) FROM company GROUP BY age HAVING AVG(salary) > 55000;

-- SELECT with ORDER BY
SELECT * FROM company ORDER BY age ASC, salary DESC;

-- SELECT with LIMIT (will be removed in Stage1)
SELECT * FROM company LIMIT 5;

-- SELECT with BETWEEN
SELECT * FROM company WHERE age BETWEEN 25 AND 35;

-- SELECT with IS NULL
SELECT * FROM employee WHERE hire_date IS NULL;

-- SELECT with NOT
SELECT * FROM company WHERE NOT (age > 30);

-- Complex nested query
SELECT c.name, e.emp_name, d.dept_name
FROM company c
JOIN employee e ON c.id = e.company_id
JOIN department d ON e.dept_id = d.id
WHERE c.salary > 50000 AND e.status = 'active';