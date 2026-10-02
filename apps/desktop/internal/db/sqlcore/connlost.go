package sqlcore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/dialect"
)

func (s *Session) connLost(err error) bool {
	if err == nil {
		return false
	}
	for _, target := range []error{
		driver.ErrBadConn, sql.ErrConnDone, io.EOF, io.ErrUnexpectedEOF,
		net.ErrClosed, syscall.ECONNRESET, syscall.EPIPE, syscall.ECONNREFUSED,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	for _, part := range []string{"unexpected eof", "connection reset", "broken pipe", "conn closed", "bad connection", "failed to connect"} {
		if strings.Contains(msg, part) {
			return true
		}
	}
	return s.Dialect.Classify(err) == dialect.ConnLost
}

func (s *Session) stalePlan(err error) bool {
	return err != nil && s.Dialect.Classify(err) == dialect.StalePlan
}

func retry[T any](ctx context.Context, s *Session, fn func() (T, error)) (T, error) {
	v, err := fn()
	if (s.connLost(err) || s.stalePlan(err)) && ctx.Err() == nil {
		return fn()
	}
	return v, err
}
