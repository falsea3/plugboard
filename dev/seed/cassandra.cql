CREATE ROLE IF NOT EXISTS relay WITH PASSWORD = 'relay' AND LOGIN = true AND SUPERUSER = true;

CREATE KEYSPACE IF NOT EXISTS shop WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TYPE IF NOT EXISTS shop.address (street text, city text, country text);

CREATE TABLE IF NOT EXISTS shop.customers (
  id int PRIMARY KEY,
  email text,
  name text,
  country text,
  plan text,
  is_active boolean,
  tags set<text>,
  address frozen<address>,
  created_at timestamp
);

CREATE TABLE IF NOT EXISTS shop.orders_by_customer (
  customer_id int,
  created_at timestamp,
  order_id uuid,
  total decimal,
  status text,
  items map<text, int>,
  PRIMARY KEY ((customer_id), created_at, order_id)
) WITH CLUSTERING ORDER BY (created_at DESC, order_id ASC);

CREATE INDEX IF NOT EXISTS customers_country ON shop.customers (country);

CREATE MATERIALIZED VIEW IF NOT EXISTS shop.customers_by_plan AS
  SELECT plan, id, name FROM shop.customers WHERE plan IS NOT NULL AND id IS NOT NULL
  PRIMARY KEY (plan, id);

INSERT INTO shop.customers (id, email, name, country, plan, is_active, tags, address, created_at) VALUES (1, 'ann.novak@example.com', 'Ann Novak', 'US', 'pro', true, {'beta', 'vip'}, {street: '1 Main St', city: 'Austin', country: 'US'}, '2026-01-03 10:00:00+0000');
INSERT INTO shop.customers (id, email, name, country, plan, is_active, tags, created_at) VALUES (2, 'bob.ito@example.com', 'Bob Ito', 'JP', 'free', true, {}, '2026-01-04 11:30:00+0000');
INSERT INTO shop.customers (id, email, name, country, plan, is_active, created_at) VALUES (3, 'chen.garcia@example.com', 'Chen Garcia', 'DE', 'team', false, '2026-01-05 09:15:00+0000');
INSERT INTO shop.customers (id, email, name, plan, is_active, created_at) VALUES (4, 'dana.berg@example.com', 'Dana Berg', 'free', true, '2026-01-06 08:00:00+0000');
INSERT INTO shop.customers (id, email, name, country, plan, is_active, created_at) VALUES (5, 'eli.okafor@example.com', 'Eli Okafor', 'NL', 'pro', true, '2026-01-07 18:45:00+0000');

INSERT INTO shop.orders_by_customer (customer_id, created_at, order_id, total, status, items) VALUES (1, '2026-02-01 10:00:00+0000', 5f1e9c1a-0b6e-4d3c-9b8e-1a2b3c4d5e01, 129.00, 'paid', {'keyboard': 1});
INSERT INTO shop.orders_by_customer (customer_id, created_at, order_id, total, status, items) VALUES (1, '2026-02-05 12:30:00+0000', 5f1e9c1a-0b6e-4d3c-9b8e-1a2b3c4d5e02, 488.90, 'shipped', {'monitor': 1, 'hub': 1});
INSERT INTO shop.orders_by_customer (customer_id, created_at, order_id, total, status, items) VALUES (3, '2026-02-07 09:00:00+0000', 5f1e9c1a-0b6e-4d3c-9b8e-1a2b3c4d5e03, 39.90, 'refunded', {'hub': 1});

CREATE TABLE IF NOT EXISTS shop.seeded (id text PRIMARY KEY);
INSERT INTO shop.seeded (id) VALUES ('done');
