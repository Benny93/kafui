package shared

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	logDir      = ".kafui"
	logBaseName = "kafui.log"
	maxLogSize  = 10 * 1024 * 1024 // 10 MB per file
	maxBackups  = 3                 // kafui.log.1, .2, .3
)

// Log is the application-wide structured logger. Available after InitLogger().
// Other packages should use this directly:
//
//	shared.Log.Info("message", "key", value)
//	shared.Log.Error("something failed", "err", err)
var Log *slog.Logger = slog.Default() // safe default until Init is called

// InitLogger sets up the rotating file logger at ~/.kafui/kafui.log.
// It is idempotent and safe to call multiple times.
func InitLogger() {
	dir, err := resolveLogDir()
	if err != nil {
		// Fall back to stderr — TUI hasn't started yet so this is visible.
		fmt.Fprintf(os.Stderr, "kafui: cannot set up log directory: %v\n", err)
		return
	}

	if err := os.MkdirAll(dir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "kafui: cannot create log directory %s: %v\n", dir, err)
		return
	}

	logPath := filepath.Join(dir, logBaseName)
	rw := newRotatingWriter(logPath, maxLogSize, maxBackups)

	handler := slog.NewTextHandler(rw, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Use a compact RFC3339 time format
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	})

	Log = slog.New(handler)
	slog.SetDefault(Log)

	Log.Info("logger initialised", "path", logPath, "maxSizeMB", maxLogSize/1024/1024, "maxBackups", maxBackups)
}

// DebugLog is a backward-compatible shim for code that uses the old API.
func DebugLog(format string, args ...interface{}) {
	Log.Debug(fmt.Sprintf(format, args...))
}

// resolveLogDir returns the path to ~/.kafui.
func resolveLogDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, logDir), nil
}

// rotatingWriter is a thread-safe io.Writer that rotates the underlying file
// when it exceeds maxSize bytes, keeping up to maxBackups numbered copies.
type rotatingWriter struct {
	mu         sync.Mutex
	path       string
	maxSize    int64
	maxBackups int
	file       *os.File
	size       int64
}

func newRotatingWriter(path string, maxSize int64, maxBackups int) *rotatingWriter {
	rw := &rotatingWriter{path: path, maxSize: maxSize, maxBackups: maxBackups}
	// Open (or create) the log file; measure its current size so the first
	// write doesn't immediately rotate a file that is almost full.
	if f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
		rw.file = f
		if fi, err := f.Stat(); err == nil {
			rw.size = fi.Size()
		}
	}
	return rw
}

func (rw *rotatingWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file == nil {
		if err := rw.open(); err != nil {
			return 0, err
		}
	}

	if rw.size+int64(len(p)) > rw.maxSize {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := rw.file.Write(p)
	rw.size += int64(n)
	return n, err
}

func (rw *rotatingWriter) open() error {
	f, err := os.OpenFile(rw.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	rw.file = f
	if fi, err := f.Stat(); err == nil {
		rw.size = fi.Size()
	}
	return nil
}

// rotate renames kafui.log → kafui.log.1, kafui.log.1 → kafui.log.2, etc.,
// dropping files beyond maxBackups, then opens a fresh kafui.log.
func (rw *rotatingWriter) rotate() error {
	if rw.file != nil {
		_ = rw.file.Close()
		rw.file = nil
	}

	// Shift existing backup files: .3 dropped, .2 → .3, .1 → .2
	for i := rw.maxBackups - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", rw.path, i)
		dst := fmt.Sprintf("%s.%d", rw.path, i+1)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dst)
		}
	}

	// Promote current log to .1
	_ = os.Rename(rw.path, rw.path+".1")

	return rw.open()
}

// Close flushes and closes the underlying file. Call on application exit.
func (rw *rotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.file != nil {
		err := rw.file.Close()
		rw.file = nil
		return err
	}
	return nil
}

// ensure rotatingWriter satisfies io.WriteCloser
var _ io.WriteCloser = (*rotatingWriter)(nil)

// SlogWriter is an io.Writer that routes each written line into a *slog.Logger
// at INFO level. Use it to redirect legacy log.Logger or fmt.Fprintf calls
// (e.g. sarama.Logger, errWriter) into the structured log file without
// touching the terminal.
type SlogWriter struct {
	logger *slog.Logger
	buf    []byte
	mu     sync.Mutex
}

// NewSlogWriter creates a new SlogWriter backed by the given logger.
func NewSlogWriter(logger *slog.Logger) *SlogWriter {
	return &SlogWriter{logger: logger}
}

// Write buffers bytes and emits a log line for each newline-terminated record.
func (w *SlogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		idx := -1
		for i, b := range w.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]
		if line != "" {
			w.logger.Info(line)
		}
	}
	return len(p), nil
}
