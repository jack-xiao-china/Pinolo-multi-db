-- Test DDL for PINOLO
DROP TABLE IF EXISTS company;
DROP TABLE IF EXISTS orders;

CREATE TABLE company (
  id INT PRIMARY KEY,
  name VARCHAR(100),
  age INT,
  city VARCHAR(50),
  salary DECIMAL(10,2),
  KEY idx_age (age),
  KEY idx_city (city)
);

CREATE TABLE orders (
  order_id INT PRIMARY KEY,
  customer_id INT,
  amount DECIMAL(10,2),
  order_date DATE,
  status VARCHAR(20),
  KEY idx_customer (customer_id)
);

INSERT INTO company VALUES
  (1, 'Alice', 25, 'Beijing', 5000.00),
  (2, 'Bob', 30, 'Shanghai', 8000.00),
  (3, 'Charlie', 22, 'Guangzhou', 4000.00),
  (4, 'David', 35, 'Beijing', 12000.00),
  (5, 'Eve', 28, 'Shanghai', 6000.00),
  (6, 'Frank', 40, 'Hangzhou', 15000.00),
  (7, 'Grace', 26, NULL, 5500.00);

INSERT INTO orders VALUES
  (1, 1, 100.50, '2024-01-15', 'completed'),
  (2, 1, 200.00, '2024-02-20', 'pending'),
  (3, 2, 500.00, '2024-01-10', 'completed'),
  (4, 3, 150.00, '2024-03-05', 'cancelled'),
  (5, 4, 300.00, '2024-02-28', 'completed'),
  (6, 5, 450.00, '2024-01-25', 'pending'),
  (7, NULL, 800.00, '2024-03-10', 'completed');