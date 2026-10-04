package api

import (
	"context"
	"errors"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (a *App) RunQuery(sessionID, queryID, script string) model.QueryRun {
	s, err := a.session(sessionID)
	if err != nil {
		return model.QueryRun{Results: []model.ResultSet{}, Error: describe(err).Message, ErrorIndex: -1, ErrorPosition: -1}
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()

	results, err := s.Run(ctx, script)
	run := model.QueryRun{Results: results, ErrorIndex: -1, ErrorPosition: -1}
	if run.Results == nil {
		run.Results = []model.ResultSet{}
	}
	if err != nil {
		run.Error = describe(err).Message
		var se *db.StatementError
		if errors.As(err, &se) {
			run.ErrorIndex = se.Index
			run.RolledBack = se.RolledBack
			run.ErrorPosition = se.Position
		}
		run.Cancelled = errors.Is(ctx.Err(), context.Canceled)
	}
	return run
}

func (a *App) CheckSyntax(sessionID, script string) ([]model.SyntaxProblem, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	checker, ok := s.(db.SyntaxChecker)
	if !ok {
		return []model.SyntaxProblem{}, nil
	}
	ctx, cancel := context.WithTimeout(a.context(), 5*time.Second)
	defer cancel()
	return checker.CheckSyntax(ctx, script)
}

func (a *App) WriteStatements(sessionID, script string) ([]string, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	return s.WriteStatements(script), nil
}

func (a *App) RunMore(sessionID, queryID, statement string, offset int) (model.ResultSet, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.ResultSet{}, err
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()
	return s.RunMore(ctx, statement, offset)
}

func (a *App) cancellable(queryID string, timeout time.Duration) (context.Context, func()) {
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(a.context(), timeout)
	} else {
		ctx, cancel = context.WithCancel(a.context())
	}
	a.mu.Lock()
	a.queries[queryID] = cancel
	a.mu.Unlock()
	return ctx, func() {
		a.mu.Lock()
		delete(a.queries, queryID)
		a.mu.Unlock()
		cancel()
	}
}

func (a *App) CancelQuery(queryID string) {
	a.mu.Lock()
	cancel := a.queries[queryID]
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
