package api

import (
	"context"
	"errors"

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
