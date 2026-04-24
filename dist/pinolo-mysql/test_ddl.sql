-- PINOLO Test DDL
-- Database schema and test data for logic bug detection

DROP TABLE IF EXISTS company;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS employees;

CREATE TABLE company (
  id INT PRIMARY KEY,
  name VARCHAR(100),
  age INT,
  city VARCHAR(50),
  salary DECIMAL(10,2),
  active BOOLEAN DEFAULT TRUE,
  KEY idx_age (age),
  KEY idx_city (city)
);

CREATE TABLE orders (
  order_id INT PRIMARY KEY,
  customer_id INT,
  product_id INT,
  amount DECIMAL(10,2),
  order_date DATE,
  status VARCHAR(20),
  KEY idx_customer (customer_id),
  KEY idx_product (product_id)
);

CREATE TABLE products (
  product_id INT PRIMARY KEY,
  name VARCHAR(100),
  category VARCHAR(50),
  price DECIMAL(10,2),
  stock INT,
  KEY idx_category (category)
);

CREATE TABLE employees (
  emp_id INT PRIMARY KEY,
  name VARCHAR(100),
  department VARCHAR(50),
  manager_id INT,
  hire_date DATE,
  salary DECIMAL(10,2),
  KEY idx_dept (department),
  KEY idx_manager (manager_id)
);

-- Test data for company
INSERT INTO company VALUES
  (1, 'Alice', 25, 'Beijing', 5000.00, TRUE),
  (2, 'Bob', 30, 'Shanghai', 8000.00, TRUE),
  (3, 'Charlie', 22, 'Guangzhou', 4000.00, TRUE),
  (4, 'David', 35, 'Beijing', 12000.00, TRUE),
  (5, 'Eve', 28, 'Shanghai', 6000.00, TRUE),
  (6, 'Frank', 40, 'Hangzhou', 15000.00, FALSE),
  (7, 'Grace', 26, NULL, 5500.00, TRUE),
  (8, 'Henry', 32, 'Shenzhen', 7000.00, TRUE),
  (9, 'Ivy', 24, 'Beijing', 4800.00, TRUE),
  (10, 'Jack', 38, NULL, 9500.00, FALSE);

-- Test data for orders
INSERT INTO orders VALUES
  (1, 1, 101, 100.50, '2024-01-15', 'completed'),
  (2, 1, 102, 200.00, '2024-02-20', 'pending'),
  (3, 2, 103, 500.00, '2024-01-10', 'completed'),
  (4, 3, 101, 150.00, '2024-03-05', 'cancelled'),
  (5, 4, 104, 300.00, '2024-02-28', 'completed'),
  (6, 5, 102, 450.00, '2024-01-25', 'pending'),
  (7, NULL, 105, 800.00, '2024-03-10', 'completed'),
  (8, 2, 101, 250.00, '2024-04-01', 'pending'),
  (9, 6, 103, 600.00, '2024-03-15', 'completed'),
  (10, NULL, 104, 350.00, '2024-04-05', 'cancelled');

-- Test data for products
INSERT INTO products VALUES
  (101, 'Laptop', 'Electronics', 999.99, 50),
  (102, 'Phone', 'Electronics', 599.99, 100),
  (103, 'Table', 'Furniture', 199.99, 30),
  (104, 'Chair', 'Furniture', 89.99, 80),
  (105, 'Book', 'Books', 19.99, 200);

-- Test data for employees
INSERT INTO employees VALUES
  (1, 'CEO', 'Management', NULL, '2020-01-01', 20000.00),
  (2, 'Manager A', 'Sales', 1, '2021-03-15', 10000.00),
  (3, 'Manager B', 'Engineering', 1, '2021-06-20', 12000.00),
  (4, 'Employee A', 'Sales', 2, '2022-01-10', 5000.00),
  (5, 'Employee B', 'Sales', 2, '2022-05-01', 5500.00),
  (6, 'Employee C', 'Engineering', 3, '2022-08-15', 6000.00),
  (7, 'Employee D', 'Engineering', 3, '2023-01-20', 6500.00);