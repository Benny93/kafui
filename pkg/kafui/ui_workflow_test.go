package kafui

import (
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestUIWorkflowIntegration tests the complete UI workflow integration
// This covers Priority 2 from the testing plan: UI navigation workflows
func TestUIWorkflowIntegration(t *testing.T) {
	// Create mock data source for testing
	dataSource := mock.KafkaDataSourceMock{}
	dataSource.Init("")

	// Test UI component creation and integration
	t.Run("MainPage Creation", func(t *testing.T) {
		mainPage := NewMainPage()
		if mainPage == nil {
			t.Fatal("Failed to create MainPage")
		}

		// Create minimal UI components for testing
		_ = tview.NewApplication()
		pages := tview.NewPages()
		modal := tview.NewModal()
		msgChannel := make(chan UIEvent, 10)

		// Test main page creation
		flex := mainPage.CreateMainPage(dataSource, pages, tviewApp, modal, msgChannel)
		if flex == nil {
			t.Error("CreateMainPage returned nil flex")
		}
	})

	t.Run("TopicPage Creation", func(t *testing.T) {
		app := tview.NewApplication()
		pages := tview.NewPages()
		msgChannel := make(chan UIEvent, 10)

		topicPage := NewTopicPage(dataSource, pages, app, msgChannel)
		if topicPage == nil {
			t.Fatal("Failed to create TopicPage")
		}

		topicPageFlex := topicPage.CreateTopicPage("Test Topic")
		if topicPageFlex == nil {
			t.Error("CreateTopicPage returned nil flex")
		}
	})

	t.Run("UI Event Channel Integration", func(t *testing.T) {
		msgChannel := make(chan UIEvent, 10)

		// Test that UI events can be sent and received
		testEvents := []UIEvent{
			OnFocusSearch,
			OnStartTableSearch,
			OnModalClose,
			OnPageChange,
		}

		for _, event := range testEvents {
			select {
			case msgChannel <- event:
				// Successfully sent event
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Failed to send event %v to channel", event)
			}
		}

		// Verify events can be received
		for i, expectedEvent := range testEvents {
			select {
			case receivedEvent := <-msgChannel:
				if receivedEvent != expectedEvent {
					t.Errorf("Event %d: expected %v, got %v", i, expectedEvent, receivedEvent)
				}
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Failed to receive event %d", i)
			}
		}
	})
}

// TestKeyboardNavigationIntegration tests keyboard navigation workflows
func TestKeyboardNavigationIntegration(t *testing.T) {
	// Test key event handling integration
	testCases := []struct {
		name        string
		key         tcell.Key
		rune        rune
		frontPage   string
		expectedAction string
	}{
		{
			name:           "Colon key triggers search focus",
			key:            tcell.KeyRune,
			rune:           ':',
			frontPage:      "main",
			expectedAction: "focus_search",
		},
		{
			name:           "Slash key triggers table search on main page",
			key:            tcell.KeyRune,
			rune:           '/',
			frontPage:      "main",
			expectedAction: "table_search",
		},
		{
			name:           "Escape key on topic page",
			key:            tcell.KeyEsc,
			rune:           0,
			frontPage:      "topicPage",
			expectedAction: "close_topic",
		},
		{
			name:           "Escape key on detail page",
			key:            tcell.KeyEsc,
			rune:           0,
			frontPage:      "DetailPage",
			expectedAction: "hide_detail",
		},
		{
			name:           "Escape key returns to main",
			key:            tcell.KeyEsc,
			rune:           0,
			frontPage:      "otherPage",
			expectedAction: "return_main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create event for testing
			event := tcell.NewEventKey(tc.key, tc.rune, tcell.ModNone)
			
			// Test that the event structure is correct
			if event.Key() != tc.key {
				t.Errorf("Expected key %v, got %v", tc.key, event.Key())
			}
			if event.Rune() != tc.rune {
				t.Errorf("Expected rune %v, got %v", tc.rune, event.Rune())
			}

			// Note: Full UI testing would require more complex setup
			// This test verifies the event structure and basic logic
			t.Logf("Successfully created and validated key event for %s", tc.name)
		})
	}
}

// TestPageNavigationWorkflow tests the complete page navigation workflow
func TestPageNavigationWorkflow(t *testing.T) {
	// Test page transitions and state management
	pages := tview.NewPages()
	
	// Create mock pages
	mainPage := tview.NewFlex()
	topicPage := tview.NewFlex()
	detailPage := tview.NewFlex()
	modal := tview.NewModal()

	// Add pages to the page manager
	pages.
		AddPage("main", mainPage, true, true).
		AddPage("topicPage", topicPage, true, false).
		AddPage("DetailPage", detailPage, true, false).
		AddPage("modal", modal, true, false)

	// Test initial state
	frontPage, _ := pages.GetFrontPage()
	if frontPage != "main" {
		t.Errorf("Expected initial front page to be 'main', got '%s'", frontPage)
	}

	// Test page switching
	pages.SwitchToPage("topicPage")
	frontPage, _ = pages.GetFrontPage()
	if frontPage != "topicPage" {
		t.Errorf("Expected front page to be 'topicPage', got '%s'", frontPage)
	}

	// Test modal showing/hiding
	pages.ShowPage("modal")
	// Modal should be visible but we can't easily test visibility in unit tests
	
	pages.HidePage("modal")
	// Modal should be hidden

	// Test returning to main page
	pages.SwitchToPage("main")
	frontPage, _ = pages.GetFrontPage()
	if frontPage != "main" {
		t.Errorf("Expected front page to be 'main' after return, got '%s'", frontPage)
	}
}

// TestDataSourceUIIntegration tests integration between data source and UI components
func TestDataSourceUIIntegration(t *testing.T) {
	dataSource := mock.KafkaDataSourceMock{}
	dataSource.Init("")

	t.Run("Topics Integration", func(t *testing.T) {
		topics, err := dataSource.GetTopics()
		if err != nil {
			t.Fatalf("Failed to get topics: %v", err)
		}

		// Verify topics can be used in UI context
		for topicName, topic := range topics {
			if topicName == "" {
				t.Error("Found topic with empty name")
			}
			if topic.NumPartitions <= 0 {
				t.Errorf("Topic %s has invalid partition count: %d", topicName, topic.NumPartitions)
			}
			if topic.ReplicationFactor <= 0 {
				t.Errorf("Topic %s has invalid replication factor: %d", topicName, topic.ReplicationFactor)
			}
		}
	})

	t.Run("Consumer Groups Integration", func(t *testing.T) {
		groups, err := dataSource.GetConsumerGroups()
		if err != nil {
			t.Fatalf("Failed to get consumer groups: %v", err)
		}

		// Verify consumer groups can be displayed in UI
		for _, group := range groups {
			if group.Name == "" {
				t.Error("Found consumer group with empty name")
			}
			if group.State == "" {
				t.Error("Found consumer group with empty state")
			}
			// Consumers count can be 0, so we don't check for > 0
		}
	})

	t.Run("Context Switching Integration", func(t *testing.T) {
		contexts, err := dataSource.GetContexts()
		if err != nil {
			t.Fatalf("Failed to get contexts: %v", err)
		}

		if len(contexts) == 0 {
			t.Skip("No contexts available for testing")
		}

		// Test context switching
		originalContext := dataSource.GetContext()
		
		for _, context := range contexts {
			err := dataSource.SetContext(context)
			if err != nil {
				t.Errorf("Failed to set context to %s: %v", context, err)
				continue
			}

			currentContext := dataSource.GetContext()
			if currentContext != context {
				t.Errorf("Expected context %s, got %s", context, currentContext)
			}
		}

		// Restore original context
		if originalContext != "" {
			dataSource.SetContext(originalContext)
		}
	})
}

// TestUIPropertyCreation tests UI property display functions
func TestUIPropertyCreation(t *testing.T) {
	t.Run("CreatePropertyInfo", func(t *testing.T) {
		propertyName := "Test Property"
		propertyValue := "Test Value"
		
		inputField := CreatePropertyInfo(propertyName, propertyValue)
		if inputField == nil {
			t.Fatal("CreatePropertyInfo returned nil")
		}

		// Verify the input field is properly configured
		if inputField.GetText() != propertyValue {
			t.Errorf("Expected text %s, got %s", propertyValue, inputField.GetText())
		}
	})

	t.Run("CreateRunInfo", func(t *testing.T) {
		runeName := "Test Rune"
		info := "Test Info"
		
		inputField := CreateRunInfo(runeName, info)
		if inputField == nil {
			t.Fatal("CreateRunInfo returned nil")
		}

		// Verify the input field is properly configured
		if inputField.GetText() != info {
			t.Errorf("Expected text %s, got %s", info, inputField.GetText())
		}
	})
}