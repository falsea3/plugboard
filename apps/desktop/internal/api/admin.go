package api

import (
	"context"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func sessionAs[T any](a *App, id string) (T, error) {
	var none T
	s, err := a.session(id)
	if err != nil {
		return none, err
	}
	if v, ok := s.(T); ok {
		return v, nil
	}
	return none, db.ErrNotSupported
}

func (a *App) catalogCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(a.context(), catalogTimeout)
}

func (a *App) ConfigFiles(sessionID string) ([]model.ConfigFile, error) {
	c, err := sessionAs[db.ServerConfig](a, sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return c.ConfigFiles(ctx)
}

func (a *App) ReadConfig(sessionID, path string) (model.ConfigText, error) {
	c, err := sessionAs[db.ServerConfig](a, sessionID)
	if err != nil {
		return model.ConfigText{}, err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return c.ReadConfig(ctx, path)
}

func (a *App) WriteConfig(sessionID, path, text string) (model.ConfigSaved, error) {
	c, err := sessionAs[db.ServerConfig](a, sessionID)
	if err != nil {
		return model.ConfigSaved{}, err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return c.WriteConfig(ctx, path, text)
}

func (a *App) Roles(sessionID string) (model.RoleList, error) {
	r, err := sessionAs[db.RoleManager](a, sessionID)
	if err != nil {
		return model.RoleList{}, err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return r.Roles(ctx)
}

func (a *App) RoleSQL(sessionID, from string, to model.RoleSpec) ([]string, error) {
	r, err := sessionAs[db.RoleManager](a, sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return r.RoleSQL(ctx, from, to)
}

func (a *App) SaveRole(sessionID, from string, to model.RoleSpec) (string, error) {
	r, err := sessionAs[db.RoleManager](a, sessionID)
	if err != nil {
		return "", err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return r.SaveRole(ctx, from, to)
}

func (a *App) DropRole(sessionID, id string) error {
	r, err := sessionAs[db.RoleManager](a, sessionID)
	if err != nil {
		return err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return r.DropRole(ctx, id)
}
