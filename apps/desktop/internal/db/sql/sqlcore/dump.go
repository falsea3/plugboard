package sqlcore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqltext"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

const maxImportErrors = 200

var fromStdin = regexp.MustCompile(`(?i)\bFROM\s+STDIN\b`)

func (s *Session) Dump(ctx context.Context, opts model.DumpOptions, w io.Writer, progress func(model.DumpProgress)) error {
	d, ok := s.Dialect.(dialect.Dump)
	if !ok {
		return db.ErrNotSupported
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return d.Dump(ctx, conn, opts, w, progress)
}

func (s *Session) Import(ctx context.Context, r io.Reader, total int64, opts model.ImportOptions, progress func(model.ImportProgress)) (model.ImportResult, error) {
	res := model.ImportResult{Skipped: []string{}, Errors: []model.ImportError{}}
	if s.Conn.ReadOnly {
		return res, &db.ReadOnlyError{Reason: "imports"}
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Close()
	defer conn.Raw(func(any) error { return driver.ErrBadConn })
	inTx := opts.OneTransaction && s.Dialect.TransactionalDDL()
	if inTx {
		if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
			return res, err
		}
	}
	archived := false
	if a, ok := s.Dialect.(dialect.Archive); ok {
		ar, ok, err := a.ReadArchive(r)
		if err != nil {
			return res, err
		}
		if c, closer := ar.(io.Closer); ok && closer {
			defer c.Close()
		}
		archived, r = ok, ar
	}
	sc := sqltext.NewScanner(r, s.Dialect.Syntax())
	last := time.Time{}
	report := func(current string, force bool) {
		if progress == nil || (!force && time.Since(last) < 150*time.Millisecond) {
			return
		}
		last = time.Now()
		bytes := sc.Consumed()
		if archived {
			bytes = -1
		}
		progress(model.ImportProgress{Bytes: bytes, Total: total, Statements: res.Statements, Errors: len(res.Errors), Current: cut(current, 120)})
	}
	failed := false
	for index := 0; ; index++ {
		stmt, err := sc.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			failed = true
			res.Errors = append(res.Errors, model.ImportError{Index: index, Message: err.Error()})
			break
		}
		if stmt[0] == '\\' {
			if len(res.Skipped) < 50 {
				res.Skipped = append(res.Skipped, cut(stmt, 120))
			}
			continue
		}
		report(stmt, false)
		var runErr error
		if sqltext.FirstKeyword(stmt) == "COPY" && fromStdin.MatchString(stmt) {
			var n int64
			n, runErr = s.copyIn(ctx, conn, sc, stmt)
			res.Rows += n
		} else {
			_, runErr = conn.ExecContext(ctx, stmt)
		}
		res.Statements++
		if runErr == nil {
			continue
		}
		if ctx.Err() != nil {
			res.Stopped, failed = true, true
			break
		}
		if len(res.Errors) < maxImportErrors {
			res.Errors = append(res.Errors, model.ImportError{Index: index + 1, Statement: cut(stmt, 400), Message: s.Dialect.ErrorMessage(runErr)})
		}
		if inTx || opts.StopOnError {
			failed, res.Stopped = true, true
			break
		}
	}
	if ctx.Err() != nil {
		res.Stopped, failed = true, true
	}
	if inTx {
		end := "COMMIT"
		if failed {
			end, res.RolledBack = "ROLLBACK", true
		}
		finish, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(finish, end); err != nil && !failed {
			return res, err
		}
	}
	report("", true)
	return res, nil
}

func (s *Session) copyIn(ctx context.Context, conn *sql.Conn, sc *sqltext.Scanner, stmt string) (int64, error) {
	ci, ok := s.Dialect.(dialect.CopyIn)
	if !ok {
		_ = sc.CopyData(io.Discard)
		return 0, errors.New("this database can't load COPY … FROM stdin data")
	}
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := sc.CopyData(&drain{w: pw})
		pw.CloseWithError(err)
		done <- err
	}()
	n, err := ci.CopyIn(ctx, conn, stmt, pr)
	pr.Close()
	if dataErr := <-done; dataErr != nil && err == nil {
		err = dataErr
	}
	return n, err
}

type drain struct {
	w      io.Writer
	failed bool
}

func (d *drain) Write(p []byte) (int, error) {
	if !d.failed {
		if _, err := d.w.Write(p); err != nil {
			d.failed = true
		}
	}
	return len(p), nil
}

func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}
