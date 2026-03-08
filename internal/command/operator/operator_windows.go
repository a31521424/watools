package operator

import (
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"watools/pkg/models"
)

func GetOperations() []*models.OperationCommand {
	return []*models.OperationCommand{
		models.NewOperationCommand("System Sleep", "Put your PC to sleep", "moon", func() error {
			return exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0").Run()
		}),
		models.NewOperationCommand("Lock Screen", "Lock the screen", "lock", func() error {
			return exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
		}),
		models.NewOperationCommand("Empty Recycle Bin", "Empty the Recycle Bin", "trash-2", func() error {
			return exec.Command("powershell", "-NoProfile", "-Command", "Clear-RecycleBin -Force").Run()
		}),
		models.NewOperationCommand("Show Desktop", "Show desktop by hiding all windows", "monitor", func() error {
			return exec.Command("powershell", "-NoProfile", "-Command", "(New-Object -ComObject Shell.Application).ToggleDesktop()").Run()
		}),
		models.NewOperationCommand("Toggle Dark Mode", "Switch between light and dark mode", "sun-moon", func() error {
			script := "$path='HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize'; " +
				"$val=(Get-ItemProperty -Path $path -Name AppsUseLightTheme -ErrorAction SilentlyContinue).AppsUseLightTheme; " +
				"if ($val -eq 1) { $next=0 } else { $next=1 }; " +
				"Set-ItemProperty -Path $path -Name AppsUseLightTheme -Value $next; " +
				"Set-ItemProperty -Path $path -Name SystemUsesLightTheme -Value $next"
			return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
		}),
		models.NewOperationCommand("Take Screenshot", "Take a screenshot of the entire screen", "camera", func() error {
			currentUser, err := user.Current()
			if err != nil {
				return err
			}
			desktopPath := filepath.Join(currentUser.HomeDir, "Desktop", "screenshot.png")
			desktopPath = strings.ReplaceAll(desktopPath, "'", "''")
			script := "Add-Type -AssemblyName System.Windows.Forms; " +
				"Add-Type -AssemblyName System.Drawing; " +
				"$bounds=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds; " +
				"$bmp=New-Object System.Drawing.Bitmap $bounds.Width, $bounds.Height; " +
				"$graphics=[System.Drawing.Graphics]::FromImage($bmp); " +
				"$graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size); " +
				"$bmp.Save('" + desktopPath + "', [System.Drawing.Imaging.ImageFormat]::Png); " +
				"$graphics.Dispose(); $bmp.Dispose()"
			return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
		}),
		models.NewOperationCommand("Task View", "Open task view", "layout-grid", func() error {
			return exec.Command("explorer", "shell:::{3080F90E-D7AD-11D9-BD98-0000947B0257}").Run()
		}),
		models.NewOperationCommand("Eject All Volumes", "Eject all removable volumes", "book-up", func() error {
			script := "$shell = New-Object -ComObject Shell.Application; " +
				"Get-WmiObject Win32_LogicalDisk -Filter \"DriveType=2\" | ForEach-Object { " +
				"$drive = $_.DeviceID; " +
				"$shell.Namespace(17).ParseName($drive).InvokeVerb('Eject') }"
			return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
		}),
	}
}
