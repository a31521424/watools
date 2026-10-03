package host

import "watools/pkg/logger"

type LogService struct{}

func (s *LogService) GetLogDirectoryApi() string { return logger.GetLogDirectory() }
func (s *LogService) ListLogFilesApi() ([]logger.LogFileInfo, error) {
	v, e := logger.ListLogFiles()
	return v, failure("log.list", e)
}
func (s *LogService) QueryLogsApi(q logger.LogQuery) (logger.LogQueryResult, error) {
	if q.Limit < 0 || q.Limit > 1000 {
		return logger.LogQueryResult{}, invalid("limit 必须是 0 到 1000 的整数")
	}
	v, e := logger.QueryLogs(q)
	return v, failure("log.query", e)
}
func (s *LogService) LogFrontendApi(level, message string) error {
	switch level {
	case "debug":
		logger.Debug(message)
	case "info":
		logger.Info(message)
	case "warn":
		logger.Warning(message)
	case "error":
		logger.ErrorWithPureString(message)
	default:
		return invalid("日志级别必须是 debug、info、warn 或 error")
	}
	return nil
}
