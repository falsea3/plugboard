package api

import (
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (a *App) GetSettings() model.Settings {
	return a.settings.Load()
}

func (a *App) SaveSettings(v model.Settings) (model.Settings, error) {
	saved, err := a.settings.Save(v)
	if err != nil {
		return saved, err
	}
	a.menu.setTheme(a.ctx, saved.Theme)
	return saved, nil
}
