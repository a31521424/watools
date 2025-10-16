package handler

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"watools/config"
	"watools/pkg/logger"
)

func pluginRoute(res http.ResponseWriter, req *http.Request) {
	relativePath := strings.TrimPrefix(req.URL.Path, "/api/plugin")
	pluginPath := path.Join(config.ProjectCacheDir(), "plugins", relativePath)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		logger.Error(err, fmt.Sprintf("Plugin file not found: %s", pluginPath))
		http.NotFound(res, req)
		return
	}

	// Set appropriate Content-Type based on file extension
	setContentType(res, pluginPath)

	http.ServeFile(res, req, pluginPath)
}

// setContentType sets the appropriate Content-Type header based on file extension
func setContentType(res http.ResponseWriter, filePath string) {
	ext := strings.ToLower(path.Ext(filePath))

	switch ext {
	// JavaScript and TypeScript
	case ".js":
		res.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".mjs":
		res.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".ts":
		res.Header().Set("Content-Type", "text/typescript; charset=utf-8")

	// Stylesheets
	case ".css":
		res.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".scss", ".sass":
		res.Header().Set("Content-Type", "text/x-scss; charset=utf-8")
	case ".less":
		res.Header().Set("Content-Type", "text/x-less; charset=utf-8")

	// HTML and XML
	case ".html", ".htm":
		res.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".xml":
		res.Header().Set("Content-Type", "application/xml; charset=utf-8")
	case ".xhtml":
		res.Header().Set("Content-Type", "application/xhtml+xml; charset=utf-8")

	// Data formats
	case ".json":
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".yaml", ".yml":
		res.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	case ".toml":
		res.Header().Set("Content-Type", "application/toml; charset=utf-8")
	case ".csv":
		res.Header().Set("Content-Type", "text/csv; charset=utf-8")

	// Images
	case ".png":
		res.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		res.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		res.Header().Set("Content-Type", "image/gif")
	case ".svg":
		res.Header().Set("Content-Type", "image/svg+xml")
	case ".webp":
		res.Header().Set("Content-Type", "image/webp")
	case ".ico":
		res.Header().Set("Content-Type", "image/x-icon")
	case ".bmp":
		res.Header().Set("Content-Type", "image/bmp")

	// Fonts
	case ".woff":
		res.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		res.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		res.Header().Set("Content-Type", "font/ttf")
	case ".otf":
		res.Header().Set("Content-Type", "font/otf")
	case ".eot":
		res.Header().Set("Content-Type", "application/vnd.ms-fontobject")

	// Audio
	case ".mp3":
		res.Header().Set("Content-Type", "audio/mpeg")
	case ".wav":
		res.Header().Set("Content-Type", "audio/wav")
	case ".ogg":
		res.Header().Set("Content-Type", "audio/ogg")
	case ".m4a":
		res.Header().Set("Content-Type", "audio/mp4")
	case ".flac":
		res.Header().Set("Content-Type", "audio/flac")

	// Video
	case ".mp4":
		res.Header().Set("Content-Type", "video/mp4")
	case ".webm":
		res.Header().Set("Content-Type", "video/webm")
	case ".avi":
		res.Header().Set("Content-Type", "video/x-msvideo")
	case ".mov":
		res.Header().Set("Content-Type", "video/quicktime")
	case ".wmv":
		res.Header().Set("Content-Type", "video/x-ms-wmv")

	// Archives
	case ".zip":
		res.Header().Set("Content-Type", "application/zip")
	case ".tar":
		res.Header().Set("Content-Type", "application/x-tar")
	case ".gz":
		res.Header().Set("Content-Type", "application/gzip")
	case ".rar":
		res.Header().Set("Content-Type", "application/vnd.rar")
	case ".7z":
		res.Header().Set("Content-Type", "application/x-7z-compressed")

	// Documents
	case ".pdf":
		res.Header().Set("Content-Type", "application/pdf")
	case ".doc":
		res.Header().Set("Content-Type", "application/msword")
	case ".docx":
		res.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	case ".xls":
		res.Header().Set("Content-Type", "application/vnd.ms-excel")
	case ".xlsx":
		res.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	case ".ppt":
		res.Header().Set("Content-Type", "application/vnd.ms-powerpoint")
	case ".pptx":
		res.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")

	// Text files
	case ".txt":
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case ".md", ".markdown":
		res.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	case ".rtf":
		res.Header().Set("Content-Type", "application/rtf")

	// Programming languages
	case ".py":
		res.Header().Set("Content-Type", "text/x-python; charset=utf-8")
	case ".go":
		res.Header().Set("Content-Type", "text/x-go; charset=utf-8")
	case ".java":
		res.Header().Set("Content-Type", "text/x-java-source; charset=utf-8")
	case ".c":
		res.Header().Set("Content-Type", "text/x-c; charset=utf-8")
	case ".cpp", ".cc", ".cxx":
		res.Header().Set("Content-Type", "text/x-c++; charset=utf-8")
	case ".h", ".hpp":
		res.Header().Set("Content-Type", "text/x-c; charset=utf-8")
	case ".rs":
		res.Header().Set("Content-Type", "text/x-rust; charset=utf-8")
	case ".php":
		res.Header().Set("Content-Type", "text/x-php; charset=utf-8")
	case ".rb":
		res.Header().Set("Content-Type", "text/x-ruby; charset=utf-8")
	case ".swift":
		res.Header().Set("Content-Type", "text/x-swift; charset=utf-8")
	case ".kt":
		res.Header().Set("Content-Type", "text/x-kotlin; charset=utf-8")
	case ".sh":
		res.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	case ".ps1":
		res.Header().Set("Content-Type", "text/x-powershell; charset=utf-8")

	// Config files
	case ".conf", ".config":
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case ".ini":
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case ".properties":
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case ".env":
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Binary executables (no charset for binary)
	case ".exe":
		res.Header().Set("Content-Type", "application/octet-stream")
	case ".bin":
		res.Header().Set("Content-Type", "application/octet-stream")
	case ".dmg":
		res.Header().Set("Content-Type", "application/x-apple-diskimage")
	case ".pkg":
		res.Header().Set("Content-Type", "application/octet-stream")
	case ".deb":
		res.Header().Set("Content-Type", "application/vnd.debian.binary-package")
	case ".rpm":
		res.Header().Set("Content-Type", "application/x-rpm")

	// Default case - let Go's http.ServeFile detect the content type
	default:
		// No explicit Content-Type set, let http.ServeFile handle it
	}
}
