package operator

import (
	"os/exec"
	"os/user"
	"path/filepath"
	"watools/pkg/models"
)

func GetOperations() []*models.OperationCommand {
	return []*models.OperationCommand{
		// 无需特殊权限的操作
		models.NewOperationCommand("System Sleep", "Put your Mac to sleep", "moon", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to sleep").Run()
		}),
		models.NewOperationCommand("Lock Screen", "Lock the screen", "lock", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to keystroke \"q\" using {control down, command down}").Run()
		}),
		models.NewOperationCommand("Empty Trash", "Empty the Trash", "trash-2", func() error {
			return exec.Command("osascript", "-e", "tell application \"Finder\" to empty trash").Run()
		}),
		models.NewOperationCommand("Show Desktop", "Show desktop by hiding all windows", "monitor", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to key code 103 using {command down}").Run()
		}),
		models.NewOperationCommand("Toggle Dark Mode", "Switch between light and dark mode", "sun-moon", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to tell appearance preferences to set dark mode to not dark mode").Run()
		}),
		models.NewOperationCommand("Open Activity Monitor", "Launch Activity Monitor", "activity", func() error {
			return exec.Command("open", "/Applications/Utilities/Activity Monitor.app").Run()
		}),
		models.NewOperationCommand("Open Terminal", "Launch Terminal", "terminal", func() error {
			return exec.Command("open", "/Applications/Utilities/Terminal.app").Run()
		}),
		models.NewOperationCommand("Open System Preferences", "Launch System Preferences", "settings", func() error {
			return exec.Command("open", "/System/Applications/System Preferences.app").Run()
		}),
		models.NewOperationCommand("Take Screenshot", "Take a screenshot of the entire screen", "camera", func() error {
			currentUser, err := user.Current()
			if err != nil {
				return err
			}
			desktopPath := filepath.Join(currentUser.HomeDir, "Desktop", "screenshot.png")
			return exec.Command("screencapture", "-C", desktopPath).Run()
		}),
		models.NewOperationCommand("Force Quit Front App", "Force quit the frontmost application", "x-circle", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to keystroke \"q\" using {command down, option down}").Run()
		}),
		models.NewOperationCommand("Toggle Mission Control", "Show Mission Control", "layout-grid", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to key code 160").Run()
		}),
		models.NewOperationCommand("Open Spotlight", "Open Spotlight search", "search", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to keystroke \" \" using command down").Run()
		}),
		models.NewOperationCommand("Minimize All Windows", "Minimize all windows of the front app", "minimize-2", func() error {
			return exec.Command("osascript", "-e", "tell application \"System Events\" to keystroke \"m\" using {command down, option down}").Run()
		}),
		models.NewOperationCommand("New Finder Window", "Open a new Finder window", "folder", func() error {
			return exec.Command("osascript", "-e", "tell application \"Finder\" to make new Finder window").Run()
		}),
		models.NewOperationCommand("Eject All Volumes", "Eject all removable volumes", "eject", func() error {
			return exec.Command("osascript", "-e", "tell application \"Finder\" to eject (every disk whose ejectable is true)").Run()
		}),
	}
}
