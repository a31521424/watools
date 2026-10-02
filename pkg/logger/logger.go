package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
	"watools/config"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

var WaLogger zerolog.Logger

func getLogDir() string {
	logDir := filepath.Join(config.ProjectCacheDir(), "logs")
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		log.Printf("Failed to get log dir: %v", err)
		panic(err)
	}
	return logDir
}

func InitWaLogger() {
	var writers []io.Writer

	logDir := getLogDir()

	if isatty.IsTerminal(os.Stdout.Fd()) {
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime, NoColor: false}
		writers = append(writers, consoleWriter)
	}
	logFile := &rotatingWriter{dir: logDir}

	writers = append(writers, logFile)
	multipleWriter := io.MultiWriter(writers...)
	WaLogger = zerolog.New(multipleWriter).With().Timestamp().Logger()
	logStr := ""
	if isatty.IsTerminal(os.Stdout.Fd()) {
		logStr = "WaLogger Init for terminal (dev mode)"
	} else {
		logStr = "WaLogger Init for file (production mode)"
	}
	setLogLevel()
	Info(logStr)
	// Redirects Log to Zero log
	log.SetFlags(0)
	log.SetOutput(multipleWriter)
}

func setLogLevel() {
	lvl, _ := zerolog.ParseLevel(config.LogLevel())
	zerolog.SetGlobalLevel(lvl)
}
