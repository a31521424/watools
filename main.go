package main

import (
	"embed"
	"fmt"
	"os"
	"sync"
	"watools/config"
	"watools/internal/app"
	"watools/internal/app_menu"
	"watools/internal/coordinator"
	"watools/internal/handler"
	"watools/pkg/logger"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var projectJSON []byte

func init() {
	os.Setenv("LANG", "en_US.UTF-8")
	os.Setenv("LC_ALL", "en_US.UTF-8")
	config.ParseProject(projectJSON)
	logger.InitWaLogger()
}

func main() {
	service := coordinator.GetWaAppCoordinator()
	desktop := application.New(application.Options{
		Name:        config.ProjectName(),
		Server:      application.ServerOptions{Host: "127.0.0.1", Port: 9246},
		Description: fmt.Sprintf("Version: %s\nAuthor: %s", config.ProjectVersion(), config.ProjectAuthor()),
		Services:    []application.Service{application.NewService(service)},
		Assets: application.AssetOptions{
			Handler:        handler.WithAssets(application.AssetFileServerFS(assets)),
			DisableLogging: true,
		},
		Logger: logger.NewSlogAdapter(),
	})
	window := desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "launcher", Title: "watools", Width: 1280, Height: 64,
		Frameless:        true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Mac:              application.MacWindow{TitleBar: application.MacTitleBarHidden},
	})
	service.ConfigureDesktop(desktop, window)
	var ready sync.Once
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(event *application.WindowEvent) {
		ready.Do(app.GetWaApp().OnWindowReady)
	})
	desktop.Menu.SetApplicationMenu(app_menu.GetWatoolsMenu())
	if err := desktop.Run(); err != nil {
		logger.Error(err)
	}
}
