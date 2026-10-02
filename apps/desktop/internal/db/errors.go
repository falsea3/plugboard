package db

import (
	"errors"
	"fmt"
)

type ApplyError struct {
	Index int
	Err   error
}

func (e *ApplyError) Error() string { return fmt.Sprintf("change %d: %v", e.Index+1, e.Err) }
func (e *ApplyError) Unwrap() error { return e.Err }

type StatementError struct {
	Statement  string
	Index      int
	Err        error
	RolledBack bool
}

func (e *StatementError) Error() string {
	msg := e.Err.Error()
	if e.Index > 0 {
		msg = fmt.Sprintf("statement %d: %s", e.Index+1, msg)
	}
	if e.RolledBack {
		msg += " (the open transaction was rolled back)"
	}
	return msg
}

func (e *StatementError) Unwrap() error { return e.Err }

type ReadOnlyError struct{ Reason string }

func (e *ReadOnlyError) Error() string {
	return fmt.Sprintf("read-only connection: %s are not allowed. Turn off read-only for this connection first", e.Reason)
}

var ErrConnLost = errors.New("the connection to the database was lost; the next run opens a new one. Check whether the statement ran before running it again")

var ErrRowGone = errors.New("the row was changed or deleted by someone else — refresh and try again")

var ErrTableBusy = errors.New("another transaction is using the table — perhaps one left open in the SQL editor. Commit or roll it back, then try again")
