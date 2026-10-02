-- Runs on MySQL 8.0+ and MariaDB 10.4+. Rows are made from a table of numbers
-- rather than a recursive CTE, whose depth limit the two name differently.
CREATE TEMPORARY TABLE seq (n INT PRIMARY KEY);
INSERT INTO seq
SELECT 1 + a.d + 10 * b.d + 100 * c.d + 1000 * e.d
FROM (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4
      UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) a
CROSS JOIN (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4
      UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) b
CROSS JOIN (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4
      UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) c
CROSS JOIN (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4
      UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) e;

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
SELECT CONCAT('user', n, '@example.com'),
       CONCAT(ELT(1 + n % 6, 'Ann', 'Bob', 'Chen', 'Dana', 'Eli', 'Fatima'), ' ', n),
       ELT(1 + n % 5, 'US', 'DE', 'RU', 'JP', NULL),
       n % 7 <> 0,
       IF(n % 3 = 0, JSON_OBJECT('plan', 'pro', 'seats', n % 10), NULL),
       NOW() - INTERVAL n HOUR
FROM seq WHERE n <= 1500 ORDER BY n;

INSERT INTO orders (customer_id, status, total, placed_at)
SELECT 1 + (n * 7) % 1500, ELT(1 + n % 4, 'new', 'paid', 'shipped', 'cancelled'),
       ROUND(RAND() * 900 + 5, 2), NOW() - INTERVAL n MINUTE
FROM seq WHERE n <= 5000 ORDER BY n;

DROP TEMPORARY TABLE seq;

CREATE VIEW order_summary AS
SELECT c.id AS customer_id, c.name, COUNT(o.id) AS orders, SUM(o.total) AS revenue
FROM customers c LEFT JOIN orders o ON o.customer_id = c.id
GROUP BY c.id, c.name;
