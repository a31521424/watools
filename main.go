package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"watools/config"
	"watools/internal"
	"watools/internal/app"
	"watools/internal/handler"
	"watools/internal/launch"
	"watools/pkg/logger"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsJson []byte

func init() {
	config.ParseProject(wailsJson)
	logger.InitWaLogger()
}

func initApp(ctx context.Context, apps []interface{}) {
	config.InitWailsContext(ctx)
	config.InitDevMode()

	for _, waApp := range apps {
		baseApp := waApp.(internal.BaseApp)
		baseApp.Startup(ctx)
	}
}

func main() {
	waApps := []interface{}{
		app.GetWaApp(),
		launch.GetWaLaunch(),
	}

	err := wails.Run(&options.App{
		Title:     "watools",
		Width:     800,
		Height:    58,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: handler.NewWaHandler(),
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup: func(ctx context.Context) {
			initApp(ctx, waApps)
		},
		Bind: waApps,
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			About: &mac.AboutInfo{
				Title:   config.ProjectName(),
				Message: fmt.Sprintf("Version: %s\nAuthor: %s", config.ProjectVersion(), config.ProjectAuthor()),
			},
		},
		Logger: logger.NewAdapter(),
	})

	if err != nil {
		logger.Error(err)
	}
}
