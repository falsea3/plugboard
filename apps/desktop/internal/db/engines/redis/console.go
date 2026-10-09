package redis

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func classify(err error) error {
	var ne net.Error
	if errors.Is(err, io.EOF) || errors.Is(err, goredis.ErrClosed) || (errors.As(err, &ne) && !ne.Timeout()) {
		var op *net.OpError
		if errors.As(err, &op) && op.Op == "dial" {
			return err
		}
		return db.ErrConnLost
	}
	return err
}

func (s *Session) Run(ctx context.Context, script string) ([]model.ResultSet, error) {
	results := []model.ResultSet{}
	cmds, err := parseScript(script)
	if err != nil {
		var le *lineError
		errors.As(err, &le)
		return results, &db.StatementError{Statement: le.text, Index: len(cmds), Err: le.err, Position: -1}
	}
	s.consoleMu.Lock()
	defer s.consoleMu.Unlock()
	for i, c := range cmds {
		fail := func(err error) ([]model.ResultSet, error) {
			return results, &db.StatementError{Statement: c.text, Index: i, Err: err, Position: -1}
		}
		if why, ok := refused[strings.ToLower(c.args[0])]; ok {
			return fail(apperr.New("not_supported", why))
		}
		if s.conn.ReadOnly && !s.isRead(ctx, c.args) {
			return fail(&db.ReadOnlyError{Reason: "commands that change data"})
		}
		conn, err := s.consoleConn()
		if err != nil {
			return fail(err)
		}
		start := time.Now()
		v, err := conn.Do(ctx, anyArgs(c.args)...).Result()
		if errors.Is(err, goredis.Nil) {
			v, err = nil, nil
		}
		if err != nil {
			err = classify(err)
			if errors.Is(err, db.ErrConnLost) || ctx.Err() != nil {
				s.resetConsole()
			}
			return fail(err)
		}
		if strings.EqualFold(c.args[0], "select") && len(c.args) == 2 {
			if n, err := strconv.Atoi(c.args[1]); err == nil {
				s.consoleDB = n
			}
		}
		rs := reply(c, v)
		rs.DurationMs = float64(time.Since(start).Microseconds()) / 1000
		results = append(results, rs)
	}
	return results, nil
}

func (s *Session) consoleConn() (*goredis.Conn, error) {
	if s.console != nil {
		return s.console, nil
	}
	n, err := dbIndex(s.conn.Database)
	if err != nil {
		return nil, err
	}
	if s.consoleDB >= 0 {
		n = s.consoleDB
	}
	c, err := s.client(n, true)
	if err != nil {
		return nil, err
	}
	s.console = c.Conn()
	return s.console, nil
}

func (s *Session) resetConsole() {
	if s.console != nil {
		s.console.Close()
		s.console = nil
	}
}

func anyArgs(args []string) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = a
	}
	return out
}
