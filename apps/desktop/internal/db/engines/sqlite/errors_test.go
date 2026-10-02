package sqlite_test

import (
	"errors"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/engines/sqlite"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/dialect"
)

func TestSyntaxErrors(t *testing.T) {
	d := sqlite.Dialect{}
	err := errors.New(`SQL logic error: near "fron": syntax error (1)`)
	if d.Classify(err) != dialect.BadSyntax || d.ErrorPosition(err, "select * fron t") != 9 || d.ErrorMessage(err) != `near "fron": syntax error` {
		t.Fatalf("near: %v %d %q", d.Classify(err), d.ErrorPosition(err, "select * fron t"), d.ErrorMessage(err))
	}
	end := errors.New("SQL logic error: incomplete input (1)")
	if d.Classify(end) != dialect.BadSyntax || d.ErrorPosition(end, "select") != 6 || d.ErrorMessage(end) != "the statement ends too early" {
		t.Fatal("incomplete input")
	}
	if d.Classify(errors.New("SQL logic error: no such table: t (1)")) == dialect.BadSyntax {
		t.Fatal("a missing table is no syntax error")
	}
}
