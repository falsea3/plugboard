SET SESSION cte_max_recursion_depth = 10000;
CREATE TABLE customers (
  id         BIGINT AUTO_INCREMENT PRIMARY KEY,
  email      VARCHAR(255) NOT NULL UNIQUE,
  name       VARCHAR(255) NOT NULL,
  country    CHAR(2),
  is_active  TINYINT(1) NOT NULL DEFAULT 1,
  meta       JSON,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
  id          BIGINT AUTO_INCREMENT PRIMARY KEY,
  customer_id BIGINT NOT NULL,
  status      ENUM('new', 'paid', 'shipped', 'cancelled') NOT NULL DEFAULT 'new',
  total       DECIMAL(12, 2) NOT NULL,
  placed_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (customer_id) REFERENCES customers(id)
);

INSERT INTO customers (email, name, country, is_active, meta, created_at)
WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM seq WHERE n < 1500)
SELECT CONCAT('user', n, '@example.com'),
       CONCAT(ELT(1 + n % 6, 'Ann', 'Bob', 'Chen', 'Dana', 'Eli', 'Fatima'), ' ', n),
       ELT(1 + n % 5, 'US', 'DE', 'RU', 'JP', NULL),
       n % 7 <> 0,
       IF(n % 3 = 0, JSON_OBJECT('plan', 'pro', 'seats', n % 10), NULL),
       NOW() - INTERVAL n HOUR
FROM seq;

INSERT INTO orders (customer_id, status, total, placed_at)
WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM seq WHERE n < 5000)
SELECT 1 + (n * 7) % 1500, ELT(1 + n % 4, 'new', 'paid', 'shipped', 'cancelled'),
       ROUND(RAND() * 900 + 5, 2), NOW() - INTERVAL n MINUTE
FROM seq;

CREATE VIEW order_summary AS
SELECT c.id AS customer_id, c.name, COUNT(o.id) AS orders, SUM(o.total) AS revenue
FROM customers c LEFT JOIN orders o ON o.customer_id = c.id
GROUP BY c.id, c.name;
