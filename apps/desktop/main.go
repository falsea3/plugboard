package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/relay-client/plugboard/apps/desktop/internal/api"
	"github.com/relay-client/plugboard/apps/desktop/internal/crash"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

const singleInstanceID = "com.relayclient.plugboard"

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-version", "version":
			info := api.NewApp().AppInfo()
			fmt.Printf("%s %s (%s)\n", info.Name, info.Version, info.GoVersion)
			os.Exit(0)
		}
	}

	dir := api.PrepareDataDir()
	app := api.NewApp()
	info := app.AppInfo()
	header := fmt.Sprintf("%s %s (%s, %s)", info.Name, info.Version, info.GoVersion, info.Platform)
	if path, err := crash.Watch(crash.Dir(dir), header); err != nil {
		fmt.Fprintln(os.Stderr, "crash log:", err)
	} else {
		api.NoteCrash(app, path)
	}

	err := wails.Run(&options.App{
		Title:     "Plugboard",
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
		Bind:                 []any{app},
		ErrorFormatter:       api.FormatError,
		DisablePanicRecovery: true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
