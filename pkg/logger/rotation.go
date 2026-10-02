package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Rotation also happens during a long-running session. Old files are bounded by
// count, so a burst of logging cannot consume disk indefinitely.
type rotatingWriter struct {
	mu      sync.Mutex
	dir     string
	file    *os.File
	day     string
	size    int64
	maxSize int64
	keep    int
	now     func() time.Time
}

func (w *rotatingWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.maxSize == 0 {
		w.maxSize = 8 << 20
	}
	if w.keep == 0 {
		w.keep = 14
	}
	if w.now == nil {
		w.now = time.Now
	}
	if int64(len(data)) > w.maxSize {
		return 0, fmt.Errorf("log record exceeds file size limit")
	}
	day := w.now().Format(time.DateOnly)
	if w.file == nil || day != w.day || w.size+int64(len(data)) > w.maxSize {
		if w.file != nil {
			if err := w.file.Close(); err != nil {
				return 0, err
			}
			w.file = nil
		}
		entries, err := os.ReadDir(w.dir)
		if err != nil {
			return 0, err
		}
		index := -1
		prefix := "watools-" + day + "-"
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".log") {
				value, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".log"))
				if err == nil && value > index {
					index = value
				}
			}
		}
		if index < 0 {
			index = 0
		}
		path := filepath.Join(w.dir, fmt.Sprintf("watools-%s-%04d.log", day, index))
		if stat, err := os.Stat(path); err == nil && stat.Size()+int64(len(data)) > w.maxSize {
			index++
			path = filepath.Join(w.dir, fmt.Sprintf("watools-%s-%04d.log", day, index))
		} else if err != nil && !os.IsNotExist(err) {
			return 0, err
		}

		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return 0, err
		}
		stat, err := file.Stat()
		if err != nil {
			file.Close()
			return 0, err
		}
		w.file = file
		w.day = day
		w.size = stat.Size()
		entries, err = os.ReadDir(w.dir)
		if err != nil {
			return 0, err
		}
		var names []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), "watools-") && strings.HasSuffix(entry.Name(), ".log") {
				names = append(names, entry.Name())
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		remaining := w.keep - 1
		for _, name := range names {
			candidate := filepath.Join(w.dir, name)
			if candidate == path {
				continue
			}
			if remaining > 0 {
				remaining--
				continue
			}
			if err := os.Remove(candidate); err != nil {
				return 0, err
			}
		}
	}
	n, err := w.file.Write(data)
	w.size += int64(n)
	return n, err
}
func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
