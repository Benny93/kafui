package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	templatestyles "github.com/Benny93/kafui/pkg/ui/template/ui/styles"
)

func TestNextThemeModeCycle(t *testing.T) {
	want := map[string]string{"auto": "dark", "dark": "light", "light": "auto", "": "auto"}
	for cur, exp := range want {
		if got := nextThemeMode(cur); got != exp {
			t.Errorf("nextThemeMode(%q) = %q, want %q", cur, got, exp)
		}
	}
}

func TestApplyThemeModeSyncsBothSystems(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	m := NewUIModel(ds)

	m.applyThemeMode("light")
	if m.common.Styles.GetTheme() != stylesPkg.LightTheme {
		t.Fatal("core styles not switched to light")
	}
	if templatestyles.IsDark() {
		t.Fatal("template theme should be light after applying light mode")
	}
	if m.common.AppConfig.UI.Theme != "light" {
		t.Fatalf("expected persisted mode 'light', got %q", m.common.AppConfig.UI.Theme)
	}

	m.applyThemeMode("dark")
	if m.common.Styles.GetTheme() != stylesPkg.DarkTheme {
		t.Fatal("core styles not switched to dark")
	}
	if !templatestyles.IsDark() {
		t.Fatal("template theme should be dark after applying dark mode")
	}
}
