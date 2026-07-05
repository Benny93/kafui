package components

import (
	"strings"
	"testing"
)

func TestBreadcrumbActionsSlot(t *testing.T) {
	b := NewBreadcrumb()
	b.SetWidth(80)
	b.SetItems([]string{"Kafka UI", "Topics"})
	b.SetActions("d delete • e edit")

	view := b.View()
	if !strings.Contains(view, "Topics") {
		t.Fatalf("breadcrumb missing item text: %q", view)
	}
	if !strings.Contains(view, "delete") {
		t.Fatalf("breadcrumb missing page-actions text: %q", view)
	}
}

func TestBreadcrumbEmptyHidden(t *testing.T) {
	b := NewBreadcrumb()
	b.SetActions("x action")
	if b.View() != "" {
		t.Fatal("breadcrumb with no items should render empty even with actions set")
	}
}
