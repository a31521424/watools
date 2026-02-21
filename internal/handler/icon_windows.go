package handler

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	_ "image/gif"
	_ "image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	ico "github.com/biessek/golang-ico"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"golang.org/x/sys/windows"
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
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	return writePng(destPath, img)
}

func convertIconToPng(sourcePath, destPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, err := ico.Decode(file)
	if err != nil {
		return err
	}
	return writePng(destPath, img)
}

func extractExecutableIconToPng(sourcePath string, iconIndex int, destPath string) error {
	if _, err := os.Stat(sourcePath); err != nil {
		return err
	}
	img, err := extractIconFromFile(sourcePath, int32(iconIndex))
	if err == nil {
		return writePng(destPath, img)
	}

	if fallbackErr := extractAssociatedIconToPng(sourcePath, destPath); fallbackErr == nil {
		return nil
	}

	return err
}

func extractAssociatedIconToPng(sourcePath, destPath string) error {
	if _, err := os.Stat(sourcePath); err != nil {
		return err
	}

	img, err := extractAssociatedIcon(sourcePath)
	if err != nil {
		return err
	}
	return writePng(destPath, img)
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

func writePng(destPath string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return png.Encode(out, img)
}

func extractIconFromFile(sourcePath string, iconIndex int32) (image.Image, error) {
	var large windows.Handle
	var small windows.Handle

	count, err := extractIconEx(sourcePath, iconIndex, &large, &small)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("no icon handle")
	}

	handle := large
	if handle == 0 {
		handle = small
	}
	if handle == 0 {
		return nil, errors.New("no icon handle")
	}
	defer destroyIcon(handle)

	return hiconToImage(handle)
}

func extractAssociatedIcon(sourcePath string) (image.Image, error) {
	handle, err := extractAssociatedIconHandle(sourcePath)
	if err != nil {
		return nil, err
	}
	defer destroyIcon(handle)

	return hiconToImage(handle)
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  windows.Handle
	hbmColor windows.Handle
}

type bitmap struct {
	bmType       int32
	bmWidth      int32
	bmHeight     int32
	bmWidthBytes int32
	bmPlanes     uint16
	bmBitsPixel  uint16
	bmBits       uintptr
}

type bitmapInfoHeader struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

type bitmapInfo struct {
	header bitmapInfoHeader
	colors [1]uint32
}

const (
	diNormal      = 0x0003
	biRGB         = 0
	dibRGBColors  = 0
	defaultIconDP = 96
)

func hiconToImage(hicon windows.Handle) (image.Image, error) {
	width, height, cleanup, err := iconSize(hicon)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid icon size")
	}

	hdc, err := createCompatibleDC()
	if err != nil {
		return nil, err
	}
	defer deleteDC(hdc)

	var bmi bitmapInfo
	bmi.header.size = uint32(unsafe.Sizeof(bmi.header))
	bmi.header.width = int32(width)
	bmi.header.height = int32(-height)
	bmi.header.planes = 1
	bmi.header.bitCount = 32
	bmi.header.compression = biRGB
	bmi.header.xPelsPerMeter = defaultIconDP
	bmi.header.yPelsPerMeter = defaultIconDP

	var bits unsafe.Pointer
	hbmp, err := createDIBSection(hdc, &bmi, &bits)
	if err != nil {
		return nil, err
	}
	defer deleteObject(windows.Handle(hbmp))

	old, err := selectObject(hdc, windows.Handle(hbmp))
	if err != nil {
		return nil, err
	}
	defer selectObject(hdc, old)

	if err := drawIconEx(hdc, hicon, width, height); err != nil {
		return nil, err
	}
	if bits == nil {
		return nil, errors.New("icon bitmap has no pixel data")
	}

	size := width * height * 4
	src := unsafe.Slice((*byte)(bits), size)
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))

	for i := 0; i < size; i += 4 {
		b := src[i]
		g := src[i+1]
		r := src[i+2]
		a := src[i+3]
		if a == 0 {
			dst.Pix[i] = 0
			dst.Pix[i+1] = 0
			dst.Pix[i+2] = 0
			dst.Pix[i+3] = 0
			continue
		}
		ra := uint32(a)
		dst.Pix[i] = uint8((uint32(r)*255 + ra/2) / ra)
		dst.Pix[i+1] = uint8((uint32(g)*255 + ra/2) / ra)
		dst.Pix[i+2] = uint8((uint32(b)*255 + ra/2) / ra)
		dst.Pix[i+3] = a
	}

	return dst, nil
}

func iconSize(hicon windows.Handle) (int, int, func(), error) {
	var info iconInfo
	if err := getIconInfo(hicon, &info); err != nil {
		return 0, 0, func() {}, err
	}

	cleanup := func() {
		if info.hbmColor != 0 {
			deleteObject(info.hbmColor)
		}
		if info.hbmMask != 0 {
			deleteObject(info.hbmMask)
		}
	}

	if info.hbmColor != 0 {
		width, height, err := bitmapSize(info.hbmColor)
		return width, height, cleanup, err
	}

	if info.hbmMask != 0 {
		width, height, err := bitmapSize(info.hbmMask)
		if err != nil {
			return 0, 0, cleanup, err
		}
		if height > 1 {
			height /= 2
		}
		return width, height, cleanup, nil
	}

	cleanup()
	return 0, 0, func() {}, errors.New("icon bitmap not found")
}

func bitmapSize(handle windows.Handle) (int, int, error) {
	var bm bitmap
	if err := getObject(handle, unsafe.Sizeof(bm), unsafe.Pointer(&bm)); err != nil {
		return 0, 0, err
	}
	return int(bm.bmWidth), int(bm.bmHeight), nil
}

var (
	modShell32                  = windows.NewLazySystemDLL("shell32.dll")
	modUser32                   = windows.NewLazySystemDLL("user32.dll")
	modGdi32                    = windows.NewLazySystemDLL("gdi32.dll")
	procExtractIconExW          = modShell32.NewProc("ExtractIconExW")
	procExtractAssociatedIconW  = modShell32.NewProc("ExtractAssociatedIconW")
	procDestroyIcon             = modUser32.NewProc("DestroyIcon")
	procGetIconInfo             = modUser32.NewProc("GetIconInfo")
	procDrawIconEx              = modUser32.NewProc("DrawIconEx")
	procCreateCompatibleDC      = modGdi32.NewProc("CreateCompatibleDC")
	procDeleteDC                = modGdi32.NewProc("DeleteDC")
	procCreateDIBSection        = modGdi32.NewProc("CreateDIBSection")
	procSelectObject            = modGdi32.NewProc("SelectObject")
	procDeleteObject            = modGdi32.NewProc("DeleteObject")
	procGetObjectW              = modGdi32.NewProc("GetObjectW")
)

func extractIconEx(path string, index int32, large, small *windows.Handle) (uint32, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	ret, _, callErr := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(index),
		uintptr(unsafe.Pointer(large)),
		uintptr(unsafe.Pointer(small)),
		1,
	)
	if ret == 0 && callErr != nil && callErr != windows.ERROR_SUCCESS {
		return 0, callErr
	}
	return uint32(ret), nil
}

func extractAssociatedIconHandle(path string) (windows.Handle, error) {
	path16, err := windows.UTF16FromString(path)
	if err != nil {
		return 0, err
	}

	var index uint16
	ret, _, callErr := procExtractAssociatedIconW.Call(
		0,
		uintptr(unsafe.Pointer(&path16[0])),
		uintptr(unsafe.Pointer(&index)),
	)
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return 0, callErr
		}
		return 0, errors.New("no associated icon")
	}
	return windows.Handle(ret), nil
}

func destroyIcon(handle windows.Handle) {
	if handle == 0 {
		return
	}
	_, _, _ = procDestroyIcon.Call(uintptr(handle))
}

func getIconInfo(hicon windows.Handle, info *iconInfo) error {
	ret, _, callErr := procGetIconInfo.Call(uintptr(hicon), uintptr(unsafe.Pointer(info)))
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("GetIconInfo failed")
	}
	return nil
}

func createCompatibleDC() (windows.Handle, error) {
	ret, _, callErr := procCreateCompatibleDC.Call(0)
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return 0, callErr
		}
		return 0, errors.New("CreateCompatibleDC failed")
	}
	return windows.Handle(ret), nil
}

func deleteDC(handle windows.Handle) {
	if handle == 0 {
		return
	}
	_, _, _ = procDeleteDC.Call(uintptr(handle))
}

func createDIBSection(hdc windows.Handle, bmi *bitmapInfo, bits *unsafe.Pointer) (windows.Handle, error) {
	ret, _, callErr := procCreateDIBSection.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(bmi)),
		dibRGBColors,
		uintptr(unsafe.Pointer(bits)),
		0,
		0,
	)
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return 0, callErr
		}
		return 0, errors.New("CreateDIBSection failed")
	}
	return windows.Handle(ret), nil
}

func selectObject(hdc windows.Handle, obj windows.Handle) (windows.Handle, error) {
	ret, _, callErr := procSelectObject.Call(uintptr(hdc), uintptr(obj))
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return 0, callErr
		}
		return 0, errors.New("SelectObject failed")
	}
	return windows.Handle(ret), nil
}

func deleteObject(handle windows.Handle) {
	if handle == 0 {
		return
	}
	_, _, _ = procDeleteObject.Call(uintptr(handle))
}

func drawIconEx(hdc windows.Handle, hicon windows.Handle, width, height int) error {
	ret, _, callErr := procDrawIconEx.Call(
		uintptr(hdc),
		0,
		0,
		uintptr(hicon),
		uintptr(width),
		uintptr(height),
		0,
		0,
		diNormal,
	)
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("DrawIconEx failed")
	}
	return nil
}

func getObject(handle windows.Handle, size uintptr, dest unsafe.Pointer) error {
	ret, _, callErr := procGetObjectW.Call(uintptr(handle), size, uintptr(dest))
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("GetObjectW failed")
	}
	return nil
}
