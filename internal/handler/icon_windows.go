package handler

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func icon2Png(iconPath string, pngPath string) error {
	iconPath = strings.TrimSpace(iconPath)
	if iconPath == "" {
		return fmt.Errorf("icon path is empty")
	}

	iconFile, iconIndex := splitIconLocation(iconPath)
	if iconFile == "" {
		return fmt.Errorf("icon path is invalid")
	}

	iconFile = os.ExpandEnv(iconFile)
	iconExt := strings.ToLower(filepath.Ext(iconFile))

	switch iconExt {
	case ".png":
		return copyFile(iconFile, pngPath)
	case ".jpg", ".jpeg", ".bmp", ".gif", ".tif", ".tiff", ".webp":
		return convertImageToPng(iconFile, pngPath)
	case ".ico":
		return convertIconToPng(iconFile, pngPath)
	case ".exe", ".dll", ".icl", ".cpl":
		return extractExecutableIconToPng(iconFile, iconIndex, pngPath)
	default:
		return extractAssociatedIconToPng(iconFile, pngPath)
	}
}

func splitIconLocation(value string) (string, int) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0
	}
	if strings.HasPrefix(value, "@") {
		value = strings.TrimPrefix(value, "@")
	}
	value = strings.Trim(value, "\"")

	if idx := strings.LastIndex(value, ","); idx > 0 {
		pathPart := strings.TrimSpace(value[:idx])
		indexPart := strings.TrimSpace(value[idx+1:])
		if pathPart != "" && indexPart != "" {
			if index, err := strconv.Atoi(indexPart); err == nil {
				return pathPart, index
			}
		}
	}
	return value, 0
}

func convertImageToPng(sourcePath, destPath string) error {
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Drawing;
$source = '%s';
$dest = '%s';
if (-not (Test-Path -LiteralPath $source)) { exit 1 };
$img = [System.Drawing.Image]::FromFile($source);
$img.Save($dest, [System.Drawing.Imaging.ImageFormat]::Png);
$img.Dispose()`, escapePowerShellSingleQuoted(sourcePath), escapePowerShellSingleQuoted(destPath))

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to convert image to png: %w\n%s", err, output)
	}
	return nil
}

func convertIconToPng(sourcePath, destPath string) error {
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Drawing;
$source = '%s';
$dest = '%s';
if (-not (Test-Path -LiteralPath $source)) { exit 1 };
$icon = New-Object System.Drawing.Icon($source);
$bmp = $null;
try { $bmp = $icon.ToBitmap() } catch { $bmp = [System.Drawing.Bitmap]::FromHicon($icon.Handle) };
if ($bmp -eq $null -or $bmp.Width -le 0 -or $bmp.Height -le 0) {
    $sizes = 256,128,64,48,32,16;
    foreach ($s in $sizes) {
        try {
            $tmp = New-Object System.Drawing.Icon($source, $s, $s);
            $bmp = $tmp.ToBitmap();
            if ($bmp -ne $null -and $bmp.Width -gt 0 -and $bmp.Height -gt 0) { break };
        } catch {}
    }
};
if ($bmp -eq $null) { exit 2 };
$bmp.Save($dest, [System.Drawing.Imaging.ImageFormat]::Png)`, escapePowerShellSingleQuoted(sourcePath), escapePowerShellSingleQuoted(destPath))

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to convert icon to png: %w\n%s", err, output)
	}
	return nil
}

func extractExecutableIconToPng(sourcePath string, iconIndex int, destPath string) error {
	if _, err := os.Stat(sourcePath); err != nil {
		return err
	}

	script := fmt.Sprintf(`Add-Type -AssemblyName System.Drawing;
Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition @'
[DllImport("shell32.dll", CharSet=CharSet.Unicode)]
public static extern uint ExtractIconEx(string szFileName, int nIconIndex, out IntPtr phiconLarge, out IntPtr phiconSmall, uint nIcons);
[DllImport("user32.dll", CharSet=CharSet.Unicode)]
public static extern bool DestroyIcon(IntPtr hIcon);
'@;
$source = '%s';
$dest = '%s';
$index = %d;
if (-not (Test-Path -LiteralPath $source)) { exit 1 };
$large = [IntPtr]::Zero; $small = [IntPtr]::Zero;
$null = [Win32.NativeMethods]::ExtractIconEx($source, $index, [ref]$large, [ref]$small, 1);
$handle = if ($large -ne [IntPtr]::Zero) { $large } else { $small };
if ($handle -eq [IntPtr]::Zero) { exit 2 };
$icon = [System.Drawing.Icon]::FromHandle($handle);
$bmp = $null;
try { $bmp = $icon.ToBitmap() } catch { $bmp = [System.Drawing.Bitmap]::FromHicon($icon.Handle) };
if ($bmp -eq $null -or $bmp.Width -le 0 -or $bmp.Height -le 0) { exit 3 };
$bmp.Save($dest, [System.Drawing.Imaging.ImageFormat]::Png);
[Win32.NativeMethods]::DestroyIcon($handle) | Out-Null`, escapePowerShellSingleQuoted(sourcePath), escapePowerShellSingleQuoted(destPath), iconIndex)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		if fallbackErr := extractAssociatedIconToPng(sourcePath, destPath); fallbackErr == nil {
			return nil
		}
		return fmt.Errorf("failed to extract icon to png: %w\n%s", err, output)
	}
	return nil
}

func extractAssociatedIconToPng(sourcePath, destPath string) error {
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Drawing;
$source = '%s';
$dest = '%s';
if (-not (Test-Path -LiteralPath $source)) { exit 1 };
$icon = [System.Drawing.Icon]::ExtractAssociatedIcon($source);
if ($icon -eq $null) { exit 2 };
$bmp = $icon.ToBitmap();
$bmp.Save($dest, [System.Drawing.Imaging.ImageFormat]::Png)`, escapePowerShellSingleQuoted(sourcePath), escapePowerShellSingleQuoted(destPath))

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to convert icon to png: %w\n%s", err, output)
	}
	return nil
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
