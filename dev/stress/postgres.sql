-- Stress data for the sample PostgreSQL, in its own database "stress".
-- About 2 GB on disk. Load with `make db-stress`.

CREATE SCHEMA IF NOT EXISTS stress;
SET search_path = stress;

-- 300 rows of ~3.5 MB jsonb each: what a table page of huge documents does.
DROP TABLE IF EXISTS fat_json;
CREATE TABLE fat_json (id int PRIMARY KEY, title text, doc jsonb);
INSERT INTO fat_json
SELECT g, 'document ' || g,
       (SELECT jsonb_agg(jsonb_build_object('i', i, 'hash', md5(i::text || g::text), 'tags', jsonb_build_array('a', 'b', i % 7)))
          FROM generate_series(1, 40000) i)
  FROM generate_series(1, 300) g;

-- One 100 MB text cell, one 30 MB bytea cell, one JSON nested 50 000 levels deep.
DROP TABLE IF EXISTS huge_cells;
CREATE TABLE huge_cells (id int PRIMARY KEY, kind text, body text, blob bytea);
INSERT INTO huge_cells VALUES
  (1, 'text 100 MB', repeat('Plugboard stress line with some unicode — ✓ ü 汉字 ', 2000000), NULL),
  (2, 'bytea 30 MB', NULL, decode(repeat(md5('x'), 1000000), 'hex') || decode(repeat(md5('y'), 875000), 'hex')),
  (3, 'deep json', repeat('[', 50000) || repeat(']', 50000), NULL),
  (4, 'small', 'just a normal row', '\xdeadbeef');

-- 5 million narrow rows: paging, counts, sorting without an index.
DROP TABLE IF EXISTS big;
CREATE TABLE big (id bigint PRIMARY KEY, n int, label text, at timestamptz);
INSERT INTO big SELECT g, ((g * 7919) % 100000)::int, 'row ' || g, '2020-01-01'::timestamptz + g * interval '1 minute'
  FROM generate_series(1::bigint, 5000000) g;

-- 1500 columns.
DO $$
DECLARE cols text;
BEGIN
  SELECT string_agg(format('c%s text DEFAULT %L', i, 'v'), ', ') INTO cols FROM generate_series(1, 1500) i;
  EXECUTE 'DROP TABLE IF EXISTS wide';
  EXECUTE 'CREATE TABLE wide (id int PRIMARY KEY, ' || cols || ')';
  EXECUTE 'INSERT INTO wide (id) SELECT g FROM generate_series(1, 500) g';
END $$;

-- 10 000 tables with foreign keys between neighbours (sidebar, diagram, catalog).
CREATE SCHEMA IF NOT EXISTS many;
SELECT format('CREATE TABLE IF NOT EXISTS many.t%s (id int PRIMARY KEY, parent int %s, name text, %s text)',
         i, CASE WHEN i > 1 THEN format('REFERENCES many.t%s', i - 1) ELSE '' END,
         quote_ident('a rather long column name that goes on and on ' || i))
  FROM generate_series(1, 10000) i
\gexec

-- Awkward names.
DROP TABLE IF EXISTS "Ünïcødé ✓ table with a really long name that hits the limit";
CREATE TABLE "Ünïcødé ✓ table with a really long name that hits the limit" ("id" int PRIMARY KEY, "select" text, "weird ""quoted"" col" text, "😀" text);
INSERT INTO "Ünïcødé ✓ table with a really long name that hits the limit" VALUES (1, 'a', 'b', 'c');

ANALYZE;

-- Open on the stress schema, not the empty public one.
ALTER DATABASE stress SET search_path = stress, public;
