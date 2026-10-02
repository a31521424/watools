package logger

import (
	"fmt"
	"strings"
)

func Error(err error, args ...interface{}) {
	event := WaLogger.Error().Err(err)
	message := ""
	if len(args) > 0 {
		message = fmt.Sprint(args[0])
	}
	for i := 1; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		lowered := strings.ToLower(key)
		if strings.Contains(lowered, "secret") || strings.Contains(lowered, "token") || strings.Contains(lowered, "password") || strings.Contains(lowered, "authorization") || lowered == "storage" || lowered == "body" || lowered == "input" || lowered == "apikey" {
			event.Str(key, "[redacted]")
		} else {
			event.Interface(key, args[i+1])
		}
	}
	if message == "" {
		event.Send()
	} else {
		event.Msg(message)
	}
}

func Info(msg string) {
	WaLogger.Info().Msg(msg)
}

func Debug(msg string) {
	WaLogger.Debug().Msg(msg)
}

func Print(msg string) {
	WaLogger.Info().Msg(msg)
}

func Trace(msg string) {
	WaLogger.Trace().Msg(msg)
}

func Warning(msg string) {
	WaLogger.Warn().Msg(msg)
}

func Fatal(msg string) {
	WaLogger.Fatal().Msg(msg)
}

func ErrorWithPureString(msg string) {
	WaLogger.Error().Msg(msg)
}
