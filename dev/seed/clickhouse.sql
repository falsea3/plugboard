CREATE TABLE shop.customers
(
    id UInt64,
    email String,
    name String,
    country LowCardinality(Nullable(FixedString(2))),
    plan Enum8('free' = 1, 'team' = 2, 'pro' = 3) DEFAULT 'free',
    is_active Bool DEFAULT true,
    tags Array(String),
    created_at DateTime64(3, 'UTC') DEFAULT now64(3)
)
ENGINE = MergeTree
ORDER BY id;

INSERT INTO shop.customers (id, email, name, country, plan, is_active, tags, created_at)
SELECT
    number + 1,
    concat('customer', toString(number + 1), '@example.com'),
    concat(['Ann', 'Bob', 'Chen', 'Dana', 'Eli', 'Fatima'][number % 6 + 1], ' ', ['Novak', 'Ito', 'Garcia', 'Berg'][number % 4 + 1]),
    if(number % 7 = 0, NULL, ['US', 'DE', 'JP', 'BR', 'NL'][number % 5 + 1]),
    ['free', 'team', 'pro'][number % 3 + 1],
    number % 9 != 0,
    if(number % 4 = 0, ['beta'], []),
    toDateTime64('2026-01-01 00:00:00', 3, 'UTC') + toIntervalMinute(number * 37)
FROM numbers(2500);

CREATE TABLE shop.orders
(
    id UInt64,
    customer_id UInt64,
    total Decimal(12, 2),
    status LowCardinality(String),
    meta Map(String, String),
    created_at DateTime('UTC'),
    INDEX status_idx status TYPE set(16) GRANULARITY 4
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (customer_id, created_at);

INSERT INTO shop.orders
SELECT
    number + 1,
    number % 2500 + 1,
    toDecimal64(round(5 + (number * 7919 % 50000) / 100, 2), 2),
    ['paid', 'shipped', 'refunded', 'pending'][number % 4 + 1],
    map('source', ['ads', 'blog', 'direct'][number % 3 + 1]),
    toDateTime('2026-01-01 00:00:00', 'UTC') + toIntervalMinute(number * 11)
FROM numbers(20000);

CREATE VIEW shop.revenue_by_country AS
SELECT c.country, count() AS orders, sum(o.total) AS revenue
FROM shop.orders AS o
INNER JOIN shop.customers AS c ON c.id = o.customer_id
GROUP BY c.country;

CREATE FUNCTION with_tax AS (amount) -> amount * 1.2;
