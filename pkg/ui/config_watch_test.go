package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

type fakeSender struct{ ch chan tea.Msg }

func (f *fakeSender) Send(msg tea.Msg) { f.ch <- msg }

func writeConfig(t *testing.T, path, body string, mod time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestWatchConfigFile_ReloadsOnChange(t *testing.T) {
	shared.InitLogger()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := time.Now().Add(-time.Hour)
	writeConfig(t, path, "ui:\n  theme: dark\n", base)

	sender := &fakeSender{ch: make(chan tea.Msg, 4)}
	done := make(chan struct{})
	defer close(done)
	go watchConfigFile(sender, path, 20*time.Millisecond, done)

	// Change the file with a newer mtime to trigger a reload.
	writeConfig(t, path, "ui:\n  theme: light\n", base.Add(time.Minute))

	select {
	case msg := <-sender.ch:
		if _, ok := msg.(core.ConfigReloadedMsg); !ok {
			t.Fatalf("expected ConfigReloadedMsg, got %T", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for config reload")
	}
}

func TestWatchConfigFile_ParseErrorSurfacesUIError(t *testing.T) {
	shared.InitLogger()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := time.Now().Add(-time.Hour)
	writeConfig(t, path, "ui:\n  theme: dark\n", base)

	sender := &fakeSender{ch: make(chan tea.Msg, 4)}
	done := make(chan struct{})
	defer close(done)
	go watchConfigFile(sender, path, 20*time.Millisecond, done)

	// Write invalid YAML with a newer mtime.
	writeConfig(t, path, "ui: [ this is: not valid", base.Add(time.Minute))

	select {
	case msg := <-sender.ch:
		if _, ok := msg.(shared.UIError); !ok {
			t.Fatalf("expected UIError on parse failure, got %T", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parse-error notification")
	}
}
