package api

import (
	"context"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (a *App) keyStore(id string) (db.KeyStore, error) {
	s, err := a.session(id)
	if err != nil {
		return nil, err
	}
	if k, ok := s.(db.KeyStore); ok {
		return k, nil
	}
	return nil, db.ErrNotSupported
}

func (a *App) ScanKeys(sessionID, db, pattern, cursor string, count int) (model.KeyPage, error) {
	k, err := a.keyStore(sessionID)
	if err != nil {
		return model.KeyPage{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return k.ScanKeys(ctx, db, pattern, cursor, count)
}

func (a *App) ReadKey(sessionID, db, key, cursor string) (model.KeyValue, error) {
	k, err := a.keyStore(sessionID)
	if err != nil {
		return model.KeyValue{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return k.ReadKey(ctx, db, key, cursor)
}

func (a *App) KeyCounts(sessionID string) (map[string]int64, error) {
	k, err := a.keyStore(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return k.KeyCounts(ctx)
}

func (a *App) EditKey(sessionID, db string, e model.KeyEdit) error {
	k, err := a.keyStore(sessionID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return k.EditKey(ctx, db, e)
}
