package launch

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/jackmordaunt/icns/v3"
	"image/png"
	"os"
	"watools/pkg/models"
)

type WaLaunchApp struct {
	ctx     context.Context
	scanner AppScanner
}

func NewWaLaunchApp() *WaLaunchApp {
	return &WaLaunchApp{}
}

func (w *WaLaunchApp) Startup(ctx context.Context) {
	w.ctx = ctx
	w.scanner = NewAppScanner()
}

func (w *WaLaunchApp) GetApplication() []models.Command {
	commands, _ := w.scanner.GetApplication()
	return commands
}

func (w *WaLaunchApp) GetIconBase64(iconPath string) string {
	// TODO: clash verge rev icon 错误
	if iconPath == "" {
		return ""
	}
	file, err := os.Open(iconPath)
	if err != nil {
		return ""
	}
	defer file.Close()
	img, err := icns.Decode(file)
	if err != nil {
		return ""
	}
	var pngBuffer bytes.Buffer
	err = png.Encode(&pngBuffer, img)
	if err != nil {
		return ""
	}
	base64String := base64.StdEncoding.EncodeToString(pngBuffer.Bytes())
	return base64String
}
