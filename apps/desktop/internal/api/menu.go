package api

import (
	"context"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const MenuEvent = "menu:command"

type appMenu struct {
	themes map[model.Theme]*menu.MenuItem
}

func (m *appMenu) setTheme(ctx context.Context, theme model.Theme) {
	if m == nil || ctx == nil {
		return
	}
	for t, item := range m.themes {
		item.Checked = t == theme
	}
	runtime.MenuUpdateApplicationMenu(ctx)
}

func BuildMenu(a *App) *menu.Menu {
	emit := func(cmd string) menu.Callback {
		return func(*menu.CallbackData) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, MenuEvent, cmd)
			}
		}
	}
	root := menu.NewMenu()

	app := root.AddSubmenu("Relay DB")
	app.AddText("About Relay DB", nil, emit("about"))
	app.AddSeparator()
	app.AddText("Settings…", keys.CmdOrCtrl(","), emit("settings"))
	app.AddSeparator()
	app.AddText("Hide Relay DB", keys.CmdOrCtrl("h"), func(*menu.CallbackData) { runtime.Hide(a.ctx) })
	app.AddSeparator()
	app.AddText("Quit Relay DB", keys.CmdOrCtrl("q"), func(*menu.CallbackData) { runtime.Quit(a.ctx) })

	file := root.AddSubmenu("File")
	file.AddText("New Connection…", keys.CmdOrCtrl("n"), emit("new-connection"))
	file.AddText("Open SQLite File…", keys.CmdOrCtrl("o"), emit("open-sqlite"))
	file.AddText("Switch Connection…", keys.CmdOrCtrl("k"), emit("switch-connection"))
	file.AddText("All Connections", keys.Combo("h", keys.CmdOrCtrlKey, keys.ShiftKey), emit("home"))
	file.AddSeparator()
	file.AddText("New Query", keys.CmdOrCtrl("t"), emit("new-query"))
	file.AddText("Add Row", keys.CmdOrCtrl("i"), emit("add-row"))
	file.AddText("Commit Changes", keys.CmdOrCtrl("s"), emit("commit"))
	file.AddText("Close Tab", keys.CmdOrCtrl("w"), emit("close-tab"))
	file.AddText("Close Connection", keys.Combo("w", keys.CmdOrCtrlKey, keys.ShiftKey), emit("close-connection"))

	root.Append(menu.EditMenu())

	view := root.AddSubmenu("View")
	theme := view.AddSubmenu("Theme")
	current := a.settings.Load().Theme
	m := &appMenu{themes: map[model.Theme]*menu.MenuItem{}}
	for _, t := range []struct {
		theme model.Theme
		label string
	}{{model.ThemeSystem, "Match System"}, {model.ThemeLight, "Light"}, {model.ThemeDark, "Dark"}} {
		m.themes[t.theme] = theme.AddRadio(t.label, t.theme == current, nil, emit("theme:"+string(t.theme)))
	}
	view.AddSeparator()
	view.AddText("Refresh", keys.CmdOrCtrl("r"), emit("refresh"))
	view.AddText("Filter Rows", keys.CmdOrCtrl("f"), emit("filter"))
	view.AddText("Filter Tables", keys.Combo("f", keys.CmdOrCtrlKey, keys.ShiftKey), emit("filter-tables"))
	a.menu = m

	win := root.AddSubmenu("Window")
	win.AddText("Minimize", keys.CmdOrCtrl("m"), func(*menu.CallbackData) { runtime.WindowMinimise(a.ctx) })
	win.AddText("Zoom", nil, func(*menu.CallbackData) { runtime.WindowToggleMaximise(a.ctx) })
	win.AddSeparator()
	win.AddText("Next Tab", keys.Combo("]", keys.CmdOrCtrlKey, keys.ShiftKey), emit("next-tab"))
	win.AddText("Previous Tab", keys.Combo("[", keys.CmdOrCtrlKey, keys.ShiftKey), emit("prev-tab"))

	return root
}
