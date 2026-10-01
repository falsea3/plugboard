package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/relay-client/relay-db/apps/desktop/internal/api"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

const singleInstanceID = "com.relayclient.relaydb"

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-version", "version":
			info := api.NewApp().AppInfo()
			fmt.Printf("%s %s (%s)\n", info.Name, info.Version, info.GoVersion)
			os.Exit(0)
		}
	}

	app := api.NewApp()
	err := wails.Run(&options.App{
		Title:     "Relay DB",
		Width:     1320,
		Height:    840,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: options.NewRGBA(17, 17, 19, 255),
		Menu:             api.BuildMenu(app),
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               singleInstanceID,
			OnSecondInstanceLaunch: func(options.SecondInstanceData) { api.ShowWindow(app) },
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				FullSizeContent:            true,
			},
			WebviewIsTransparent: true,
		},
		Bind: []any{app},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
