package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

func loadLargeObjects(ctx context.Context, q queryer) ([]uint32, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (uint32, error) {
		var oid uint32
		err := r.Scan(&oid)
		return oid, err
	}, `SELECT oid FROM pg_largeobject_metadata ORDER BY oid`)
}

func (d Dialect) largeObjectACLs(ctx context.Context, q queryer) ([]pgACL, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (pgACL, error) {
		var x pgACL
		err := r.Scan(&x.alter, &x.on, &x.name, &x.owner, &x.set, &x.grants)
		return x, err
	}, `SELECT 'LARGE OBJECT', 'LARGE OBJECT', m.oid::text, quote_ident(pg_get_userbyid(m.lomowner)), m.lomacl IS NOT NULL, `+grants("m.lomacl")+`
		FROM pg_largeobject_metadata m ORDER BY m.oid`)
}

func (d Dialect) largeObjectData(ctx context.Context, q queryer, oid uint32, onlyMissing bool, o *dumpOut) error {
	create := fmt.Sprintf("SELECT pg_catalog.lo_create('%d')", oid)
	if onlyMissing {
		create += fmt.Sprintf(" WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_largeobject_metadata WHERE oid = '%d')", oid)
	}
	if onlyMissing {
		fmt.Fprintf(o, "%s;\n", create)
	}
	lw := &loWriter{w: o, oid: oid}
	for off := int64(0); ; off += loChunk {
		var chunk []byte
		if err := q.QueryRowContext(ctx, `SELECT lo_get($1::oid, $2, $3)`, oid, off, loChunk).Scan(&chunk); err != nil {
			return fmt.Errorf("large object %d: %w", oid, err)
		}
		if _, err := lw.Write(chunk); err != nil {
			return err
		}
		if len(chunk) < loChunk {
			break
		}
	}
	if err := lw.flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(o)
	return err
}
