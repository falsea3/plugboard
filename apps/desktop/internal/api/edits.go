package api

import (
	"context"
	"errors"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (a *App) ApplyChanges(sessionID string, cs model.ChangeSet) model.ApplyResult {
	s, err := a.rowEditor(sessionID)
	if err != nil {
		return model.ApplyResult{Error: describe(err).Message, FailedIndex: -1}
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	n, err := s.ApplyChanges(ctx, cs)
	if err != nil {
		res := model.ApplyResult{Error: describe(err).Message, FailedIndex: -1}
		var ae *db.ApplyError
		if errors.As(err, &ae) {
			res.FailedIndex = ae.Index
			res.Error = describe(ae.Err).Message
		}
		return res
	}
	return model.ApplyResult{Applied: n, FailedIndex: -1}
}

func (a *App) ApplyStructure(sessionID, queryID string, sc model.StructureChange) model.ApplyResult {
	s, err := a.structureEditor(sessionID)
	if err != nil {
		return model.ApplyResult{Error: describe(err).Message, FailedIndex: -1}
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()
	n, partial, err := s.ApplyStructure(ctx, sc)
	if err != nil {
		res := model.ApplyResult{Applied: n, Partial: partial, Error: describe(err).Message, FailedIndex: -1, Cancelled: errors.Is(ctx.Err(), context.Canceled)}
		var ae *db.ApplyError
		if errors.As(err, &ae) {
			res.FailedIndex = ae.Index
			res.Error = describe(ae.Err).Message
		}
		return res
	}
	return model.ApplyResult{Applied: n, FailedIndex: -1}
}

func (a *App) PreviewStructure(sessionID string, sc model.StructureChange) ([]string, error) {
	s, err := a.structureEditor(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.PreviewStructure(ctx, sc)
}

func (a *App) PreviewChanges(sessionID string, cs model.ChangeSet) ([]string, error) {
	s, err := a.rowEditor(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.PreviewChanges(ctx, cs)
}

func (a *App) RenameTableSQL(sessionID, schema, from, to string) (string, error) {
	s, err := a.structureEditor(sessionID)
	if err != nil {
		return "", err
	}
	return s.RenameTableSQL(schema, from, to) + ";", nil
}

func (a *App) CreateIndexSQL(sessionID, schema, table string, idx model.NewIndex) (string, error) {
	s, err := a.structureEditor(sessionID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(idx.Name) == "" || len(idx.Columns) == 0 {
		return "", apperr.New("bad_index", "an index needs a name and at least one column")
	}
	return s.CreateIndexSQL(schema, table, idx) + ";", nil
}

func (a *App) DropIndexSQL(sessionID, schema, table, name string) (string, error) {
	s, err := a.structureEditor(sessionID)
	if err != nil {
		return "", err
	}
	return s.DropIndexSQL(schema, table, name) + ";", nil
}
