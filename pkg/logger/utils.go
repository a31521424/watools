package logger

func Error(err error, args ...interface{}) {
	if len(args) == 0 {
		WaLogger.Error().Err(err).Send()
	} else if msg, ok := args[0].(string); ok {
		WaLogger.Error().Err(err).Msg(msg)
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
