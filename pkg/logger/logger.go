package logger

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
	"watools/config"
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
	logFilePath := filepath.Join(logDir, fmt.Sprintf("watools-%s.log", time.Now().Format(time.DateOnly)))
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("CRITICAL: Failed to open log file %s, error: %v", logFilePath, err)
		panic(err)
		return
	}
	writers = append(writers, logFile)
	multipleWriter := io.MultiWriter(writers...)
	WaLogger = zerolog.New(multipleWriter).With().Timestamp().Logger()
	logStr := ""
	if isatty.IsTerminal(os.Stdout.Fd()) {
		WaLogger = WaLogger.Level(zerolog.InfoLevel)
		logStr = "WaLogger Init for terminal (dev mode)"
	} else {
		WaLogger = WaLogger.Level(zerolog.ErrorLevel)
		logStr = "WaLogger Init for file (production mode)"
	}
	Info(logStr)
	// Redirects Log to Zero log
	log.SetFlags(0)
	log.SetOutput(multipleWriter)
}
