package api

import "github.com/relay-client/plugboard/apps/desktop/internal/model"

func (a *App) ListScripts(connID string) ([]string, error) {
	return a.scripts.List(connID)
}

func (a *App) ReadScript(connID, name string) (string, error) {
	return a.scripts.Read(connID, name)
}

func (a *App) CreateScript(connID, name, sql string) error {
	return a.scripts.Create(connID, name, sql)
}

func (a *App) WriteScript(connID, name, sql string) error {
	return a.scripts.Write(connID, name, sql)
}

func (a *App) RenameScript(connID, from, to string) error {
	return a.scripts.Rename(connID, from, to)
}

func (a *App) DeleteScript(connID, name string) error {
	return a.scripts.Delete(connID, name)
}

func (a *App) QueryTabs(connID string) ([]model.QueryTab, error) {
	return a.scripts.Tabs(connID)
}

func (a *App) SaveQueryTabs(connID string, tabs []model.QueryTab) error {
	return a.scripts.SaveTabs(connID, tabs)
}
