PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE artists (id INTEGER PRIMARY KEY, name TEXT NOT NULL, country TEXT);
INSERT INTO artists VALUES(1,'Kino','RU');
INSERT INTO artists VALUES(2,'Radiohead','GB');
INSERT INTO artists VALUES(3,'Björk','IS');
INSERT INTO artists VALUES(4,'Daft Punk','FR');
INSERT INTO artists VALUES(5,'Nirvana','US');
CREATE TABLE albums (id INTEGER PRIMARY KEY, artist_id INTEGER NOT NULL REFERENCES artists(id), title TEXT NOT NULL, year INTEGER, rating REAL, cover BLOB);
WITH RECURSIVE s(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM s WHERE n < 800)
INSERT INTO albums (artist_id, title, year, rating, cover)
SELECT 1 + n % 5, 'Album #' || n, 1975 + n % 50,
       CASE WHEN n % 9 = 0 THEN NULL ELSE round(((n * 37) % 50) / 10.0, 1) END,
       CASE WHEN n % 4 = 0 THEN randomblob(8) END
FROM s;
CREATE VIEW top_albums AS SELECT a.title, ar.name AS artist, a.rating FROM albums a JOIN artists ar ON ar.id = a.artist_id WHERE a.rating >= 4;
COMMIT;
