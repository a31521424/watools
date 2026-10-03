package main

import (
	"embed"
	"fmt"
	"os"
	"sync"
	"watools/config"
	"watools/internal/api"
	"watools/internal/app"
	"watools/internal/app_menu"
	"watools/internal/command"
	"watools/internal/handler"
	"watools/internal/host"
	"watools/internal/plugin"
	"watools/internal/update"
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
	waApp := app.GetWaApp() // shared with the native hotkey manager
	plugins := plugin.NewWaPlugin()
	commands := command.NewWaLaunch()
	var desktop *application.App
	var window *application.WebviewWindow
	commandService := host.NewCommandService(commands, func() { desktop.Event.Emit("watools.applicationChanged") })
	pluginService := host.NewPluginService(plugins, func() ([]string, error) {
		return desktop.Dialog.OpenFile().SetTitle("Select Plugin .wt File(s)").AddFilter("WaTools Plugin Files", "*.wt").AttachToWindow(window).PromptForMultipleSelection()
	})
	desktop = application.New(application.Options{
		Name:         config.ProjectName(),
		Server:       application.ServerOptions{Host: "127.0.0.1", Port: 9246},
		Description:  fmt.Sprintf("Version: %s\nAuthor: %s", config.ProjectVersion(), config.ProjectAuthor()),
		MarshalError: host.MarshalError,
		Services: []application.Service{
			application.NewServiceWithOptions(host.NewAppService(waApp, func() bool { return desktop.Env.Info().Debug }, func() host.WindowGeometry { return window }), application.ServiceOptions{MarshalError: host.MarshalError}),
			application.NewServiceWithOptions(commandService, application.ServiceOptions{MarshalError: host.MarshalError}),
			application.NewServiceWithOptions(pluginService, application.ServiceOptions{MarshalError: host.MarshalError}),
			application.NewServiceWithOptions(host.NewStorageService(plugins), application.ServiceOptions{MarshalError: host.MarshalError}),
			application.NewServiceWithOptions(host.NewSystemService(api.NewWaApi()), application.ServiceOptions{MarshalError: host.MarshalError}),
			application.NewServiceWithOptions(host.NewUpdateService(update.NewService(), func() { desktop.Quit() }), application.ServiceOptions{MarshalError: host.MarshalError}),
			application.NewServiceWithOptions(&host.LogService{}, application.ServiceOptions{MarshalError: host.MarshalError}),
		},
		Assets: application.AssetOptions{
			Handler:        handler.WithAssets(application.AssetFileServerFS(assets), plugins.IsEnabled),
			DisableLogging: true,
		},
		Logger: logger.NewSlogAdapter(),
	})
	window = desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "launcher", Title: "watools", Width: 1280, Height: 64,
		Frameless:        true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Mac:              application.MacWindow{TitleBar: application.MacTitleBarHidden, Backdrop: application.MacBackdropTransparent},
	})
	waApp.ConfigureDesktop(desktop, window)
	var ready sync.Once
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(event *application.WindowEvent) {
		ready.Do(waApp.OnWindowReady)
	})
	desktop.Menu.SetApplicationMenu(app_menu.GetWatoolsMenu(waApp))
	if err := desktop.Run(); err != nil {
		logger.Error(err)
	}
}
