-- GaussDB A mode test DDL (Oracle compatibility)
-- Note: Create database with A mode compatibility first:
-- CREATE DATABASE testa WITH DBCOMPATIBILITY 'A';

-- A mode uses Oracle-style syntax

DROP TABLE IF EXISTS company;
DROP TABLE IF EXISTS orders;

CREATE TABLE company (
  id INT PRIMARY KEY,
  name VARCHAR2(100),
  age INT,
  city VARCHAR2(50),
  salary DECIMAL(10,2)
);

CREATE TABLE orders (
  order_id INT PRIMARY KEY,
  customer_id INT,
  amount DECIMAL(10,2),
  order_date DATE,
  status VARCHAR2(20)
);

INSERT INTO company VALUES
  (1, 'Alice', 25, 'Beijing', 5000.00),
  (2, 'Bob', 30, 'Shanghai', 8000.00),
  (3, 'Charlie', 22, 'Guangzhou', 4000.00),
  (4, 'David', 35, 'Beijing', 12000.00),
  (5, 'Eve', 28, 'Shanghai', 6000.00);

INSERT INTO orders VALUES
  (1, 1, 100.50, '2024-01-15', 'completed'),
  (2, 1, 200.00, '2024-02-20', 'pending'),
  (3, 2, 500.00, '2024-01-10', 'completed'),
  (4, NULL, 800.00, '2024-03-10', 'completed');