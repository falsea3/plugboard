CREATE SCHEMA app;
CREATE SCHEMA "Odd Schema";
CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;

CREATE TYPE app.mood AS ENUM ('sad', 'ok', 'happy', 'it''s fine');
CREATE TYPE app.pair AS (a integer, b text COLLATE "C");
CREATE TYPE app.floatrange AS RANGE (subtype = float8, subtype_diff = float8mi);
CREATE FUNCTION app.is_positive(n numeric) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$ SELECT n > 0 $$;
CREATE DOMAIN app.positive AS numeric(12,2) CHECK (app.is_positive(VALUE));
CREATE DOMAIN app.email AS text COLLATE "C" DEFAULT 'none@example.com' NOT NULL CONSTRAINT email_at CHECK (VALUE LIKE '%@%');
CREATE SEQUENCE app.ticket_seq START 1000 INCREMENT 7 CACHE 3;
CREATE FUNCTION app.next_ticket() RETURNS bigint LANGUAGE sql AS $$ SELECT nextval('app.ticket_seq') $$;

CREATE TABLE app.customers (
  id integer GENERATED ALWAYS AS IDENTITY (START WITH 100 INCREMENT BY 5),
  email app.email,
  name text NOT NULL,
  nick public.citext,
  mood app.mood DEFAULT 'ok',
  created timestamptz NOT NULL DEFAULT now(),
  ticket bigint DEFAULT app.next_ticket(),
  name_upper text GENERATED ALWAYS AS (upper(name)) STORED,
  PRIMARY KEY (id),
  CONSTRAINT name_not_blank CHECK (name <> ''),
  UNIQUE (email)
);

CREATE TABLE app.orders (
  id bigserial PRIMARY KEY,
  customer_id integer NOT NULL REFERENCES app.customers (id) ON DELETE CASCADE,
  total app.positive,
  items jsonb NOT NULL DEFAULT '[]',
  tags text[] DEFAULT '{}',
  note text,
  parent_id bigint REFERENCES app.orders (id) DEFERRABLE INITIALLY DEFERRED,
  pair app.pair,
  span app.floatrange,
  during tstzrange,
  EXCLUDE USING gist (during WITH &&)
);
CREATE INDEX orders_note_lower ON app.orders (lower(note)) WHERE note IS NOT NULL;
ALTER TABLE app.orders ADD CONSTRAINT note_short CHECK (length(note) < 1000) NOT VALID;

CREATE TABLE app.a (id int PRIMARY KEY, b_id int);
CREATE TABLE app.b (id int PRIMARY KEY, a_id int REFERENCES app.a);
ALTER TABLE app.a ADD FOREIGN KEY (b_id) REFERENCES app.b;

CREATE TABLE app.events (id bigint NOT NULL, at date NOT NULL, kind text, payload bytea, PRIMARY KEY (id, at, kind)) PARTITION BY RANGE (at);
CREATE TABLE app.events_2025 PARTITION OF app.events FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
CREATE TABLE app.events_2026 PARTITION OF app.events FOR VALUES FROM ('2026-01-01') TO ('2027-01-01') PARTITION BY LIST (kind);
CREATE TABLE app.events_2026_a PARTITION OF app.events_2026 FOR VALUES IN ('a');
CREATE TABLE app.events_2026_rest PARTITION OF app.events_2026 DEFAULT;
CREATE INDEX events_kind ON app.events (kind);

CREATE TABLE app.base_log (id int, msg text CHECK (msg <> ''));
CREATE TABLE app.child_log (level int DEFAULT 1) INHERITS (app.base_log);
CREATE UNLOGGED TABLE app.cache (k text PRIMARY KEY, v text) WITH (fillfactor = 70);
CREATE TABLE "Odd Schema"."Mixed Case" ("Select" int, "a""b" text);

CREATE TABLE public.misc (id int PRIMARY KEY, f float8, n numeric, b bytea, j json, x xml, i interval, u uuid, bits bit varying(8), arr int[][]);

INSERT INTO app.customers (email, name, nick, mood, created) VALUES
  ('a@x.io', 'Ann', 'ANN', 'happy', '2026-01-01 10:00:00+00'),
  ('b@x.io', 'Bob, "the" builder', 'bob', 'it''s fine', '2026-01-02 10:00:00.5+00'),
  ('c@x.io', E'Tab\there\nnewline\\backslash', NULL, NULL, '2026-01-03 10:00:00+00'),
  ('d@x.io', 'Émilie 日本語 🎉', 'émi', 'sad', 'infinity');
INSERT INTO app.orders (customer_id, total, items, tags, note, pair, span, during) VALUES
  (100, 150.50, '{"a": [1, 2, {"b": null}], "big": 12345678901234567890}', '{x,"y z",NULL}', NULL, ROW(1, 'one'), '[1.5,2.5)', '[2026-01-01,2026-02-01)'),
  (105, 99.99, '[]', '{}', E'line1\r\nline2', NULL, NULL, NULL),
  (110, 1000000000.01, '"str"', NULL, '\.', NULL, NULL, NULL);
UPDATE app.orders SET parent_id = 1 WHERE id = 2;
INSERT INTO app.a VALUES (1, NULL);
INSERT INTO app.b VALUES (1, 1);
UPDATE app.a SET b_id = 1;
INSERT INTO app.events VALUES (1, '2025-05-01', 'a', '\xdead'), (2, '2026-03-01', 'a', NULL), (3, '2026-04-01', 'b', '\x');
INSERT INTO app.child_log VALUES (1, 'base msg', 3);
INSERT INTO app.base_log VALUES (2, 'only base');
INSERT INTO app.cache VALUES ('pinned', 'x'), ('other', 'y');
INSERT INTO "Odd Schema"."Mixed Case" VALUES (1, 'q"uote');
INSERT INTO public.misc VALUES
  (1, 'NaN', 'NaN', '\x00ff10', '{"keep":  "spacing"}', '<a>1</a>', '1 year 2 mons', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', B'101', '{{1,2},{3,4}}'),
  (2, 'Infinity', 123456789012345678901234567890.123456789, '', NULL, NULL, '-1 day', NULL, NULL, NULL),
  (3, '-Infinity', -0.0, NULL, '[]', NULL, NULL, NULL, NULL, NULL);
SELECT nextval('app.ticket_seq');

CREATE VIEW app.big_orders AS SELECT * FROM app.orders WHERE total > 100;
CREATE VIEW app.big_order_customers AS SELECT c.name, o.total FROM app.big_orders o JOIN app.customers c ON c.id = o.customer_id;
CREATE MATERIALIZED VIEW app.mood_counts AS SELECT mood, count(*) AS n FROM app.customers GROUP BY mood;
CREATE UNIQUE INDEX ON app.mood_counts (mood);
CREATE MATERIALIZED VIEW app.empty_mv AS SELECT 1 AS x WITH NO DATA;
CREATE VIEW app.secure WITH (security_barrier) AS SELECT id FROM app.customers;

CREATE FUNCTION app.touch() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.note := coalesce(NEW.note, 'touched');
  RETURN NEW;
END
$$;
CREATE TRIGGER orders_touch BEFORE INSERT OR UPDATE ON app.orders FOR EACH ROW EXECUTE FUNCTION app.touch();
CREATE FUNCTION app.order_count(cid int) RETURNS bigint LANGUAGE sql STABLE
BEGIN ATOMIC
  SELECT count(*) FROM app.orders WHERE customer_id = cid;
END;
CREATE PROCEDURE app.reset_cache() LANGUAGE sql AS $$ DELETE FROM app.cache $$;
CREATE FUNCTION app.add_int(state int, v int) RETURNS int LANGUAGE sql IMMUTABLE AS 'SELECT state + v';
CREATE AGGREGATE app.my_sum(int) (SFUNC = app.add_int, STYPE = int, INITCOND = '0');
CREATE FUNCTION app.big_orders_of(cid int) RETURNS SETOF app.big_orders LANGUAGE sql AS $$ SELECT * FROM app.big_orders WHERE customer_id = cid $$;

CREATE RULE no_delete_pinned AS ON DELETE TO app.cache WHERE old.k = 'pinned' DO INSTEAD NOTHING;
ALTER TABLE app.customers ENABLE ROW LEVEL SECURITY;
CREATE POLICY own_rows ON app.customers FOR SELECT TO PUBLIC USING (name <> 'hidden');

COMMENT ON SCHEMA app IS 'Application data';
COMMENT ON TABLE app.customers IS 'People who buy things';
COMMENT ON COLUMN app.customers.email IS 'Contact, unique';
COMMENT ON FUNCTION app.touch() IS 'Fills note';
COMMENT ON TYPE app.mood IS 'How it went';
COMMENT ON CONSTRAINT name_not_blank ON app.customers IS 'No empty names';
COMMENT ON VIEW app.big_orders IS 'Over 100';
COMMENT ON INDEX app.events_kind IS 'by kind';
COMMENT ON DOMAIN app.email IS 'Has an @';
COMMENT ON CONSTRAINT email_at ON DOMAIN app.email IS 'Needs an at sign';
COMMENT ON TRIGGER orders_touch ON app.orders IS 'it''s a trigger';
COMMENT ON RULE no_delete_pinned ON app.cache IS 'Keep pinned';
COMMENT ON POLICY own_rows ON app.customers IS 'Hide hidden';
COMMENT ON SEQUENCE app.ticket_seq IS 'Tickets';
COMMENT ON COLUMN "Odd Schema"."Mixed Case"."a""b" IS 'odd column';

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'plugboard_owner') THEN CREATE ROLE plugboard_owner; END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'plugboard_reader') THEN CREATE ROLE plugboard_reader; END IF;
END $$;
ALTER TABLE app.cache OWNER TO plugboard_owner;
ALTER SCHEMA "Odd Schema" OWNER TO plugboard_owner;
ALTER FUNCTION app.touch() OWNER TO plugboard_owner;
ALTER TYPE app.mood OWNER TO plugboard_owner;
ALTER VIEW app.secure OWNER TO plugboard_owner;
GRANT USAGE ON SCHEMA app TO plugboard_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA app TO plugboard_reader;
GRANT SELECT (id, name), UPDATE (name) ON app.customers TO plugboard_reader;
GRANT INSERT ON app.orders TO plugboard_reader WITH GRANT OPTION;
GRANT USAGE ON SEQUENCE app.ticket_seq TO plugboard_reader;
REVOKE EXECUTE ON FUNCTION app.next_ticket() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.next_ticket() TO plugboard_reader;
REVOKE USAGE ON TYPE app.pair FROM PUBLIC;
ALTER DEFAULT PRIVILEGES IN SCHEMA app GRANT SELECT ON TABLES TO plugboard_reader;
ALTER DEFAULT PRIVILEGES REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
SELECT lo_from_bytea(424242, decode(repeat('00ff10', 100000), 'hex'));
SELECT lo_create(424243);
COMMENT ON LARGE OBJECT 424242 IS 'a blob';
ALTER LARGE OBJECT 424242 OWNER TO plugboard_owner;
GRANT SELECT ON LARGE OBJECT 424243 TO plugboard_reader;
