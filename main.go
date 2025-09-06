package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"watools/config"
	"watools/internal/coordinator"
	"watools/internal/handler"
	"watools/pkg/logger"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsJson []byte

func init() {
	initLang()
	config.ParseProject(wailsJson)
	logger.InitWaLogger()
}

func initLang() {
	os.Setenv("LANG", "en_US.UTF-8")
	os.Setenv("LC_ALL", "en_US.UTF-8")
}

func main() {
	waAppCoordinator := coordinator.GetWaAppCoordinator()

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
			config.InitWithWailsContext(ctx)
			waAppCoordinator.Startup(ctx)
		},
		OnShutdown: waAppCoordinator.Shutdown,
		Bind:       []interface{}{waAppCoordinator},
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
