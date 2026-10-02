package mysql_test

import (
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/engines/mysql"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/dialect"
)

func TestSyntaxErrors(t *testing.T) {
	d := mysql.Dialect{}
	manual := "You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use "
	stmt := "select\n  a,\n  b c d e\nfrom t"
	err := &mysqldriver.MySQLError{Number: 1064, Message: manual + "near 'd e\nfrom t' at line 3"}
	if d.Classify(err) != dialect.BadSyntax {
		t.Fatal("not a syntax error")
	}
	if pos := d.ErrorPosition(err, stmt); stmt[pos:pos+3] != "d e" {
		t.Fatalf("position %d points at %q", pos, stmt[pos:])
	}
	if d.ErrorMessage(err) != "syntax error near 'd e'" {
		t.Fatalf("message = %q", d.ErrorMessage(err))
	}
	end := &mysqldriver.MySQLError{Number: 1064, Message: manual + "near '' at line 1"}
	if d.ErrorPosition(end, "select 1 from t where") != 21 || d.ErrorMessage(end) != "syntax error at the end of the statement" {
		t.Fatal("end of the statement")
	}
	if d.Classify(&mysqldriver.MySQLError{Number: 1146}) == dialect.BadSyntax {
		t.Fatal("a missing table is no syntax error")
	}
}
