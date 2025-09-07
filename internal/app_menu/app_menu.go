package app_menu

import (
	"watools/internal/app"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

func GetWatoolsMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	applicationMenu := appMenu.AddSubmenu("Application")
	applicationMenu.AddText("Reload", keys.Combo("r", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		app.GetWaApp().ReloadAPP()
	})
	applicationMenu.AddText("Refresh", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
		app.GetWaApp().Reload()
	})

	return appMenu
}
