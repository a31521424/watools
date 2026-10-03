package app_menu

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"watools/internal/app"
)

func GetWatoolsMenu() *application.Menu {
	menu := application.NewMenu()
	menu.AddRole(application.AppMenu)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.WindowMenu)
	view := menu.AddSubmenu("View")
	view.Add("Reload").SetAccelerator("CmdOrCtrl+Shift+r").OnClick(func(_ *application.Context) {
		app.GetWaApp().ReloadAPP()
	})
	view.Add("Refresh").SetAccelerator("CmdOrCtrl+r").OnClick(func(_ *application.Context) {
		app.GetWaApp().Reload()
	})
	return menu
}
