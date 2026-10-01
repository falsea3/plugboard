CREATE SCHEMA analytics;

CREATE TABLE customers (
  id         bigserial PRIMARY KEY,
  email      text NOT NULL UNIQUE,
  name       text NOT NULL,
  country    char(2),
  is_active  boolean NOT NULL DEFAULT true,
  meta       jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE products (
  id    serial PRIMARY KEY,
  sku   text NOT NULL UNIQUE,
  title text NOT NULL,
  price numeric(10, 2) NOT NULL,
  tags  text[]
);

CREATE TABLE orders (
  id          bigserial PRIMARY KEY,
  customer_id bigint NOT NULL REFERENCES customers(id),
  status      text NOT NULL DEFAULT 'new',
  total       numeric(12, 2) NOT NULL,
  note        text,
  placed_at   timestamptz NOT NULL DEFAULT now()
);

INSERT INTO customers (email, name, country, is_active, meta, created_at)
SELECT 'user' || g || '@example.com',
       (ARRAY['Ann', 'Bob', 'Chen', 'Dana', 'Eli', 'Fatima', 'Goran', 'Hana'])[1 + g % 8] || ' ' || g,
       (ARRAY['US', 'DE', 'RU', 'JP', 'BR', NULL])[1 + g % 6],
       g % 7 <> 0,
       CASE WHEN g % 3 = 0 THEN jsonb_build_object('plan', 'pro', 'seats', g % 10) END,
       now() - (g || ' hours')::interval
FROM generate_series(1, 2500) g;

INSERT INTO products (sku, title, price, tags)
SELECT 'SKU-' || lpad(g::text, 4, '0'), 'Product ' || g, round((random() * 200 + 1)::numeric, 2),
       ARRAY['tag' || g % 5, 'tag' || g % 3]
FROM generate_series(1, 120) g;

INSERT INTO orders (customer_id, status, total, note, placed_at)
SELECT 1 + (g * 7) % 2500,
       (ARRAY['new', 'paid', 'shipped', 'cancelled'])[1 + g % 4],
       round((random() * 900 + 5)::numeric, 2),
       CASE WHEN g % 11 = 0 THEN E'Leave at the door.\nCall first.' END,
       now() - (g || ' minutes')::interval
FROM generate_series(1, 10000) g;

CREATE VIEW order_summary AS
SELECT c.id AS customer_id, c.name, count(o.id) AS orders, sum(o.total) AS revenue
FROM customers c LEFT JOIN orders o ON o.customer_id = c.id
GROUP BY c.id, c.name;

CREATE TABLE analytics.daily_revenue AS
SELECT date_trunc('day', placed_at)::date AS day, sum(total) AS revenue, count(*) AS orders
FROM orders GROUP BY 1 ORDER BY 1;
