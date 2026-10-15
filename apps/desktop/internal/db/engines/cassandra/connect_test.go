package cassandra

import (
	"errors"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
)

func TestDescribeConnect(t *testing.T) {
	for _, tc := range []struct{ raw, code string }{
		{"gocql: unable to create session: unable to discover protocol version: EOF", "conn_closed"},
		{"gocql: unable to create session: unable to discover protocol version: read tcp: connection reset by peer", "conn_closed"},
		{"Provided username relay and/or password are incorrect", "auth"},
		{"Keyspace 'nope' does not exist", "bad_database"},
	} {
		coded, ok := errors.AsType[apperr.Coded](describeConnect(errors.New(tc.raw)))
		if !ok || coded.Code() != tc.code {
			t.Errorf("%q: got %v", tc.raw, coded)
		}
	}
	plain := errors.New("something else")
	if describeConnect(plain) != plain {
		t.Error("an unknown error should pass through")
	}
}
