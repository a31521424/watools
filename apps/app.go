package apps

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/jackmordaunt/icns"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"image/png"
	"os"
	"watools/schemas"
)

type WaApp struct {
	ctx context.Context
}

func NewWaApp() *WaApp {
	return &WaApp{}
}

func (a *WaApp) InitWindowSize(ctx context.Context) {
	screen, err := runtime.ScreenGetAll(ctx)
	if err != nil {
		println(err.Error())
	}
	width := 800
	height := 56
	if len(screen) > 0 {
		width = screen[0].Size.Width / 3
	}
	runtime.WindowSetSize(ctx, width, height)
}

func (a *WaApp) Startup(ctx context.Context) {
	a.ctx = ctx
	a.InitWindowSize(ctx)
}

func (a *WaApp) GetSystemApplication() schemas.CommandGroup {
	platform, err := NewPlatform(a.ctx)
	if err != nil {
		return schemas.CommandGroup{}
	}
	if application, err := platform.GetApplication(); err == nil {
		return application
	}

	return schemas.CommandGroup{}
}

func (a *WaApp) GetIconBase64(iconPath string) string {
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
