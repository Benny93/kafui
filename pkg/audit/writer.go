package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Writer appends audit records somewhere durable.
type Writer interface {
	Write(Record) error
}

// FileWriter appends one JSON line per record to a file. It is safe for
// concurrent use.
//
// ponytail: no rotation here — audit volume for a single-user local tool is
// small. If it ever needs capping, reuse pkg/ui/shared's rotatingWriter rather
// than adding a size check inline.
type FileWriter struct {
	mu sync.Mutex
	f  *os.File
}

// NewFileWriter opens (creating parent dirs and the file if needed) path for
// append with 0600 permissions.
func NewFileWriter(path string) (*FileWriter, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &FileWriter{f: f}, nil
}

// Write appends rec as a single JSON line.
func (w *FileWriter) Write(rec Record) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.f.Write(data)
	return err
}

// Close closes the underlying file.
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}

// DefaultPath returns the default audit log path (~/.kafui/audit.log).
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "audit.log"
	}
	return filepath.Join(home, ".kafui", "audit.log")
}
