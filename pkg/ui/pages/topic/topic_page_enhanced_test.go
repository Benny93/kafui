package topic

import (
	"testing"

	"github.com/Benny93/kafui/pkg/ui/core"
)

// Test that Model implements the enhanced Page interface
func TestModelImplementsEnhancedPageInterface(t *testing.T) {
	// This will fail to compile if Model doesn't implement the enhanced Page interface
	var _ core.Page = &Model{}

	// Create a minimal model for testing
	model := &Model{}

	// Test that the new methods exist and can be called
	// Note: We're not testing the actual implementation here, just that the methods exist
	_ = model.GetTitle()
	_ = model.GetHelp()
	_, _ = model.HandleNavigation(nil)
	_ = model.OnFocus()
	_ = model.OnBlur()
}

// Test the GetTitle method
func TestModelGetTitle(t *testing.T) {
	model := &Model{
		topicName: "test-topic",
	}
	title := model.GetTitle()
	if title == "" {
		t.Error("Expected non-empty title")
	}
}

// Test the GetHelp method
func TestModelGetHelp(t *testing.T) {
	model := &Model{}
	help := model.GetHelp()
	// Help might be empty initially, but should not panic
	_ = help
}

// Test the HandleNavigation method
func TestModelHandleNavigation(t *testing.T) {
	model := &Model{}
	page, cmd := model.HandleNavigation(nil)
	// Should return the same model and nil command by default
	if page.GetID() != model.GetID() {
		t.Error("Expected same model from HandleNavigation")
	}
	if cmd != nil {
		t.Error("Expected nil command from HandleNavigation")
	}
}

// Test the OnFocus method
func TestModelOnFocus(t *testing.T) {
	model := &Model{}
	cmd := model.OnFocus()
	// Should return nil by default (no special focus handling for topic page)
	_ = cmd
}

// Test the OnBlur method
func TestModelOnBlur(t *testing.T) {
	model := &Model{}
	cmd := model.OnBlur()
	// Should return nil by default
	if cmd != nil {
		t.Error("Expected nil command from OnBlur")
	}
}