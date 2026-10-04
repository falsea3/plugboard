package api

import (
	"context"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (a *App) ListTables(sessionID, schema string) ([]model.TableInfo, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.Tables(ctx, schema)
}

func (a *App) DescribeTable(sessionID, schema, table string) ([]model.Column, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.Columns(ctx, schema, table)
}

func (a *App) Diagram(sessionID, schema string) (model.Diagram, error) {
	r, err := a.schemaReader(sessionID)
	if err != nil {
		return model.Diagram{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return r.Diagram(ctx, schema)
}

func (a *App) Relations(sessionID, schema string) ([]model.Relation, error) {
	r, err := a.schemaReader(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return r.Relations(ctx, schema)
}

func (a *App) FetchTablePage(sessionID string, q model.TableQuery) (model.TablePage, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.TablePage{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.TablePage(ctx, q)
}

func (a *App) CountRows(sessionID, queryID string, q model.TableQuery, exact bool) (model.RowCount, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.RowCount{}, err
	}
	ctx, done := a.cancellable(queryID, 5*time.Minute)
	defer done()
	return s.CountRows(ctx, q, exact)
}

func (a *App) Indexes(sessionID, schema, table string) ([]model.Index, error) {
	r, err := a.schemaReader(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return r.Indexes(ctx, schema, table)
}
