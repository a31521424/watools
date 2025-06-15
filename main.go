package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"watools/apps"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := apps.NewWaApp()

	err := wails.Run(&options.App{
		Title:     "watools",
		Width:     800,
		Height:    58,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			About: &mac.AboutInfo{
				Title:   "WaTools",
				Message: "© 2025 Banbxio",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
