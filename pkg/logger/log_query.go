package logger

import (
	"bufio"
	"container/heap"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLogQueryLimit = 200
	maxLogQueryLimit     = 1000
)

var pluginIDPattern = regexp.MustCompile(`watools\.plugin(?:\.[a-zA-Z0-9_-]+)+`)

type LogFileInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updatedAt"`
}

type LogQuery struct {
	Files     []string `json:"files"`
	Keyword   string   `json:"keyword"`
	Levels    []string `json:"levels"`
	Sources   []string `json:"sources"`
	PluginIDs []string `json:"pluginIds"`
	Cursor    string   `json:"cursor"`
	Limit     int      `json:"limit"`
}

type LogRecord struct {
	ID          string            `json:"id"`
	File        string            `json:"file"`
	FilePath    string            `json:"filePath"`
	Line        int               `json:"line"`
	Raw         string            `json:"raw"`
	ParseStatus string            `json:"parseStatus"`
	Timestamp   string            `json:"timestamp,omitempty"`
	Level       string            `json:"level,omitempty"`
	Message     string            `json:"message"`
	Source      string            `json:"source,omitempty"`
	PluginID    string            `json:"pluginId,omitempty"`
	Error       string            `json:"error,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
}

type LogQueryResult struct {
	Records            []LogRecord `json:"records"`
	NextCursor         string      `json:"nextCursor,omitempty"`
	AvailableSources   []string    `json:"availableSources"`
	AvailablePluginIDs []string    `json:"availablePluginIds"`
	TotalMatched       int         `json:"totalMatched"`
}

func GetLogDirectory() string {
	return getLogDir()
}

func ListLogFiles() ([]LogFileInfo, error) {
	logDir := getLogDir()

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	files := make([]LogFileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".log" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to stat log file %s: %w", entry.Name(), err)
		}

		files = append(files, LogFileInfo{
			Name:      entry.Name(),
			Path:      filepath.Join(logDir, entry.Name()),
			Size:      info.Size(),
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].UpdatedAt == files[j].UpdatedAt {
			return files[i].Name > files[j].Name
		}
		return files[i].UpdatedAt > files[j].UpdatedAt
	})

	return files, nil
}

func QueryLogs(query LogQuery) (LogQueryResult, error) {
	files, err := resolveLogFiles(query.Files)
	if err != nil {
		return LogQueryResult{}, err
	}

	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	levelFilter := normalizeFilter(query.Levels)
	sourceFilter := normalizeFilter(query.Sources)
	pluginFilter := normalizeFilter(query.PluginIDs)

	start := parseCursor(query.Cursor)
	if start > 10000 {
		return LogQueryResult{}, fmt.Errorf("log cursor exceeds 10000; narrow the filter")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = defaultLogQueryLimit
	}
	if limit > maxLogQueryLimit {
		limit = maxLogQueryLimit
	}
	retained := &recordHeap{}
	availableSources := make(map[string]struct{})
	availablePluginIDs := make(map[string]struct{})
	total := 0
	capacity := start + limit + 1
	for _, file := range files {
		err := scanLogFile(file, func(record LogRecord) {
			if record.Source != "" && len(availableSources) < 1024 {
				availableSources[record.Source] = struct{}{}
			}
			if record.PluginID != "" && len(availablePluginIDs) < 1024 {
				availablePluginIDs[record.PluginID] = struct{}{}
			}
			if !matchesLogFilters(record, keyword, levelFilter, sourceFilter, pluginFilter) {
				return
			}
			total++
			if retained.Len() < capacity {
				heap.Push(retained, record)
			} else if newerRecord(record, (*retained)[0]) {
				heap.Pop(retained)
				heap.Push(retained, record)
			}
		})
		if err != nil {
			return LogQueryResult{}, err
		}
	}
	matchedRecords := []LogRecord(*retained)
	sort.Slice(matchedRecords, func(i, j int) bool { return newerRecord(matchedRecords[i], matchedRecords[j]) })
	start = min(start, len(matchedRecords))
	end := min(start+limit, len(matchedRecords))
	result := LogQueryResult{Records: matchedRecords[start:end], AvailableSources: setToSortedSlice(availableSources), AvailablePluginIDs: setToSortedSlice(availablePluginIDs), TotalMatched: total}
	if end < total {
		result.NextCursor = strconv.Itoa(end)
	}

	return result, nil
}

func resolveLogFiles(requestedFiles []string) ([]LogFileInfo, error) {
	files, err := ListLogFiles()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return []LogFileInfo{}, nil
	}

	if len(requestedFiles) == 0 {
		return []LogFileInfo{files[0]}, nil
	}

	fileMap := make(map[string]LogFileInfo, len(files))
	for _, file := range files {
		fileMap[file.Name] = file
		fileMap[file.Path] = file
	}

	resolved := make([]LogFileInfo, 0, len(requestedFiles))
	seen := make(map[string]struct{}, len(requestedFiles))
	for _, requested := range requestedFiles {
		requested = strings.TrimSpace(requested)
		if requested == "" {
			continue
		}

		file, ok := fileMap[requested]
		if !ok {
			continue
		}
		if _, exists := seen[file.Path]; exists {
			continue
		}

		seen[file.Path] = struct{}{}
		resolved = append(resolved, file)
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("no matching log files found")
	}

	return resolved, nil
}

func scanLogFile(file LogFileInfo, consume func(LogRecord)) error {
	handle, err := os.Open(file.Path)
	if err != nil {
		return err
	}
	defer handle.Close()
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		consume(parseLogLine(file, line, scanner.Text()))
	}
	return scanner.Err()
}
func newerRecord(a, b LogRecord) bool {
	left, right := parseLogTimestamp(a.Timestamp), parseLogTimestamp(b.Timestamp)
	if !left.Equal(right) {
		return left.After(right)
	}
	if a.File != b.File {
		return a.File > b.File
	}
	return a.Line > b.Line
}

type recordHeap []LogRecord

func (h recordHeap) Len() int            { return len(h) }
func (h recordHeap) Less(i, j int) bool  { return newerRecord(h[j], h[i]) }
func (h recordHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *recordHeap) Push(v interface{}) { *h = append(*h, v.(LogRecord)) }
func (h *recordHeap) Pop() interface{} {
	old := *h
	v := old[len(old)-1]
	*h = old[:len(old)-1]
	return v
}

func parseLogLine(file LogFileInfo, lineNumber int, rawLine string) LogRecord {
	record := LogRecord{
		ID:          fmt.Sprintf("%s:%d", file.Name, lineNumber),
		File:        file.Name,
		FilePath:    file.Path,
		Line:        lineNumber,
		Raw:         rawLine,
		ParseStatus: "raw",
		Message:     rawLine,
	}

	rawLine = strings.TrimSpace(rawLine)
	if rawLine == "" {
		record.Source = "unknown"
		return record
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(rawLine), &payload); err != nil {
		record.PluginID = extractPluginID(rawLine)
		record.Source = inferLogSource(rawLine, rawLine, record.PluginID)
		return record
	}

	record.ParseStatus = "structured"
	record.Timestamp = stringValue(payload["time"])
	record.Level = strings.ToLower(stringValue(payload["level"]))
	record.Message = firstNonEmptyString(stringValue(payload["message"]), rawLine)
	record.Error = stringValue(payload["error"])
	record.Fields = extractAuxiliaryFields(payload)
	record.PluginID = firstNonEmptyString(
		stringValue(payload["pluginId"]),
		stringValue(payload["packageId"]),
		stringValue(payload["pluginID"]),
		stringValue(payload["plugin"]),
		extractPluginID(record.Message),
		extractPluginID(record.Error),
		extractPluginID(rawLine),
	)
	record.Source = firstNonEmptyString(
		normalizeSource(stringValue(payload["source"])),
		normalizeSource(stringValue(payload["sourceType"])),
		inferLogSource(record.Message, rawLine, record.PluginID),
	)

	if len(record.Fields) == 0 {
		record.Fields = nil
	}

	return record
}

func matchesLogFilters(
	record LogRecord,
	keyword string,
	levelFilter map[string]struct{},
	sourceFilter map[string]struct{},
	pluginFilter map[string]struct{},
) bool {
	if keyword != "" {
		candidate := strings.ToLower(strings.Join([]string{
			record.Raw,
			record.Message,
			record.Source,
			record.PluginID,
			record.Error,
		}, "\n"))
		if !strings.Contains(candidate, keyword) {
			return false
		}
	}

	if len(levelFilter) > 0 {
		if _, ok := levelFilter[strings.ToLower(record.Level)]; !ok {
			return false
		}
	}

	if len(sourceFilter) > 0 {
		if _, ok := sourceFilter[strings.ToLower(record.Source)]; !ok {
			return false
		}
	}

	if len(pluginFilter) > 0 {
		if _, ok := pluginFilter[strings.ToLower(record.PluginID)]; !ok {
			return false
		}
	}

	return true
}

func extractAuxiliaryFields(payload map[string]interface{}) map[string]string {
	fields := make(map[string]string)
	for key, value := range payload {
		switch key {
		case "level", "time", "message", "source", "sourceType", "pluginId", "pluginID", "packageId":
			continue
		}

		if rendered := renderFieldValue(value); rendered != "" {
			fields[key] = rendered
		}
	}
	return fields
}

func renderFieldValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(encoded)
	}
}

func inferLogSource(message string, raw string, pluginID string) string {
	messageLower := strings.ToLower(message)
	rawLower := strings.ToLower(raw)

	switch {
	case strings.HasPrefix(message, "[Fronted]"), strings.HasPrefix(message, "[Frontend]"):
		return "app-frontend"
	case strings.Contains(messageLower, "[fronted]"), strings.Contains(messageLower, "[frontend]"):
		return "app-frontend"
	case pluginID != "":
		return "plugin"
	case strings.Contains(messageLower, "plugin"), strings.Contains(rawLower, "plugin"):
		return "plugin"
	default:
		return "app-backend"
	}
}

func extractPluginID(value string) string {
	return pluginIDPattern.FindString(value)
}

func normalizeSource(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "unknown":
		return ""
	case "app-backend", "backend", "main", "host":
		return "app-backend"
	case "app-frontend", "frontend", "renderer":
		return "app-frontend"
	case "plugin", "plugins":
		return "plugin"
	default:
		return value
	}
}

func normalizeFilter(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}

	normalized := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" || value == "all" {
			continue
		}
		normalized[value] = struct{}{}
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func mergeStringSet(target map[string]struct{}, source map[string]struct{}) {
	for value := range source {
		target[value] = struct{}{}
	}
}

func setToSortedSlice(values map[string]struct{}) []string {
	if len(values) == 0 {
		return []string{}
	}

	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func parseCursor(cursor string) int {
	if cursor == "" {
		return 0
	}

	value, err := strconv.Atoi(cursor)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func parseLogTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
