package postgres_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/dialect"
)

func TestSyntaxErrors(t *testing.T) {
	d := postgres.Dialect{}
	stmt := "select 'шл' fron t"
	err := &pgconn.PgError{Code: "42601", Position: 13, Message: `syntax error at or near "fron"`}
	if d.Classify(err) != dialect.BadSyntax {
		t.Fatal("not a syntax error")
	}
	if pos := d.ErrorPosition(err, stmt); stmt[pos:pos+4] != "fron" {
		t.Fatalf("position %d points at %q", pos, stmt[pos:])
	}
	if pos := d.ErrorPosition(&pgconn.PgError{Code: "42601", Position: 6}, "where"); pos != 5 {
		t.Fatalf("end of input = %d", pos)
	}
	if d.ErrorMessage(err) != `syntax error at or near "fron"` || d.ErrorPosition(errors.New("x"), stmt) != -1 {
		t.Fatal("message or unknown position")
	}
	if d.Classify(&pgconn.PgError{Code: "42P01"}) == dialect.BadSyntax {
		t.Fatal("a missing table is no syntax error")
	}
}
