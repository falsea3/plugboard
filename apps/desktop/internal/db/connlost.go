package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/go-sql-driver/mysql"
)

// isConnLost reports errors that mean "the connection under this query died"
// (server restart, SSH tunnel drop, network blip) rather than "the query was
// wrong". Drivers wrap these inconsistently, so the text is checked too.
func isConnLost(err error) bool {
	if err == nil {
		return false
	}
	for _, target := range []error{
		driver.ErrBadConn, sql.ErrConnDone, io.EOF, io.ErrUnexpectedEOF,
		net.ErrClosed, syscall.ECONNRESET, syscall.EPIPE, syscall.ECONNREFUSED,
		mysql.ErrInvalidConn,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{"unexpected eof", "connection reset", "broken pipe", "conn closed", "bad connection", "failed to connect"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// ErrConnLost is returned by the SQL editor when its connection died under a
// statement it can't safely run again: one that may have written something,
// or any statement inside a transaction. The user decides what to re-run.
var ErrConnLost = errors.New("the connection to the database was lost; the next run opens a new one. Check whether the statement ran before running it again")

// retry runs fn, and once more if the first attempt failed only because a
// pooled connection had died. For reads and for writes that never reached COMMIT.
func retry[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	v, err := fn()
	if err != nil && isConnLost(err) && ctx.Err() == nil {
		return fn()
	}
	return v, err
}
