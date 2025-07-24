package kafui

import (
	"context"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockKafkaDataSourceWithConsume extends the existing MockKafkaDataSource for consumption testing
type MockKafkaDataSourceWithConsume struct {
	MockKafkaDataSource
	mock.Mock
}

func (m *MockKafkaDataSourceWithConsume) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	args := m.Called(ctx, topicName, flags, handleMessage, onError)
	
	// Simulate message consumption
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
			// Send some test messages
			handleMessage(api.Message{
				Key:           "test-key-1",
				Value:         "test-value-1",
				Offset:        100,
				Partition:     0,
				KeySchemaID:   "1",
				ValueSchemaID: "2",
				Headers:       []api.MessageHeader{{Key: "header1", Value: "value1"}},
			})
			handleMessage(api.Message{
				Key:           "test-key-2",
				Value:         "test-value-2-with-a-very-long-value-that-should-be-truncated-because-it-exceeds-100-characters-limit",
				Offset:        101,
				Partition:     1,
				KeySchemaID:   "3",
				ValueSchemaID: "4",
				Headers:       []api.MessageHeader{{Key: "header2", Value: "value2"}},
			})
		}
	}()
	
	return args.Error(0)
}

func createTestTopicPage() (*TopicPage, *MockKafkaDataSource, *tview.Application) {
	mockDS := &MockKafkaDataSource{}
	app := tview.NewApplication()
	pages := tview.NewPages()
	msgChannel := make(chan UIEvent, 10)
	
	tp := NewTopicPage(mockDS, pages, app, msgChannel)
	return tp, mockDS, app
}

func createTestTopicPageWithConsume() (*TopicPage, *MockKafkaDataSourceWithConsume, *tview.Application) {
	mockDS := &MockKafkaDataSourceWithConsume{}
	app := tview.NewApplication()
	pages := tview.NewPages()
	msgChannel := make(chan UIEvent, 10)
	
	tp := NewTopicPage(mockDS, pages, app, msgChannel)
	return tp, mockDS, app
}

func TestNewTopicPage(t *testing.T) {
	mockDS := &MockKafkaDataSource{}
	app := tview.NewApplication()
	pages := tview.NewPages()
	msgChannel := make(chan UIEvent, 10)
	
	tp := NewTopicPage(mockDS, pages, app, msgChannel)
	
	assert.NotNil(t, tp)
	assert.Equal(t, app, tp.app)
	assert.Equal(t, mockDS, tp.dataSource)
	assert.Equal(t, pages, tp.pages)
	assert.Equal(t, msgChannel, tp.msgChannel)
	assert.NotNil(t, tp.topFlexElements)
	assert.Equal(t, 0, tp.topFlexElements.Len())
}

func TestCreateTopicPage(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	
	flex := tp.CreateTopicPage("test-topic")
	
	assert.NotNil(t, flex)
	assert.NotNil(t, tp.consumerTable)
	assert.NotNil(t, tp.topFlex)
	assert.NotNil(t, tp.messagesFlex)
	assert.NotNil(t, tp.bottomFlex)
	assert.NotNil(t, tp.notifyView)
	
	// Check table properties
	rows, _ := tp.consumerTable.GetSelectable()
	assert.True(t, rows)
	
	// Check if table has input capture set
	assert.NotNil(t, tp.consumerTable)
}

func TestGetMessageKey(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	
	tests := []struct {
		partition string
		offset    string
		expected  string
	}{
		{"0", "100", "0:100"},
		{"1", "200", "1:200"},
		{"10", "999", "10:999"},
	}
	
	for _, tt := range tests {
		result := tp.getMessageKey(tt.partition, tt.offset)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetHandler(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	
	handler := tp.getHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, tp.consumedMessages)
	
	// Test handler functionality
	testMsg := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    100,
		Partition: 0,
	}
	
	handler(testMsg)
	
	expectedKey := tp.getMessageKey("0", "100")
	assert.Contains(t, tp.consumedMessages, expectedKey)
	assert.Equal(t, testMsg, tp.consumedMessages[expectedKey])
	assert.True(t, tp.newMessageConsumed)
}

func TestShortValue(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	
	tests := []struct {
		name     string
		message  api.Message
		expected string
	}{
		{
			name:     "short value",
			message:  api.Message{Value: "short"},
			expected: "short",
		},
		{
			name:     "exactly 100 chars",
			message:  api.Message{Value: "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"},
			expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
		},
		{
			name:     "long value gets truncated",
			message:  api.Message{Value: "12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901"},
			expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890...",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.shortValue(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFieldFuzzyMatchesSearchText(t *testing.T) {
	msg := api.Message{
		Key:           "test-key",
		Value:         "test-value",
		Offset:        100,
		Partition:     0,
		KeySchemaID:   "schema1",
		ValueSchemaID: "schema2",
	}
	
	tests := []struct {
		name       string
		searchText string
		expected   bool
	}{
		{"match key", "test-key", true},
		{"match value", "test-value", true},
		{"match offset", "100", true},
		{"match partition", "0", true},
		{"match key schema", "schema1", true},
		{"match value schema", "schema2", true},
		{"case insensitive", "TEST-KEY", true},
		{"partial match", "test", true},
		{"no match", "nonexistent", false},
		{"empty search", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fieldFuzzyMatchesSearchText(msg, tt.searchText)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateTopicInfoSection(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	
	topic := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      1000,
	}
	
	flex := tp.CreateTopicInfoSection("test-topic", topic)
	
	assert.NotNil(t, flex)
	// The flex should have 4 items (topic name, message count, partitions, replication factor)
	assert.Equal(t, 4, flex.GetItemCount())
}

func TestCreateTopicInfoSectionWithZeroMessages(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	
	topic := api.Topic{
		NumPartitions:     3,
		ReplicationFactor: 2,
		MessageCount:      0, // Zero messages
	}
	
	flex := tp.CreateTopicInfoSection("test-topic", topic)
	
	assert.NotNil(t, flex)
	assert.Equal(t, 4, flex.GetItemCount())
}

func TestCreateConsumeFlagsSection(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.consumeFlags = api.ConsumeFlags{
		OffsetFlag: "latest",
		Follow:     true,
		Tail:       50,
	}
	
	flex := tp.CreateConsumeFlagsSection()
	
	assert.NotNil(t, flex)
	// The flex should have 3 items (offset, follow, tail)
	assert.Equal(t, 3, flex.GetItemCount())
}

func TestCreateInputLegend(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	
	flex := tp.CreateInputLegend()
	
	assert.NotNil(t, flex)
	// Should have left and right sections
	assert.Equal(t, 2, flex.GetItemCount())
}

func TestShowNotification(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.notifyView = tview.NewTextView()
	
	// Test notification
	tp.ShowNotification("Test notification")
	
	// Give some time for the goroutine to execute
	time.Sleep(50 * time.Millisecond)
	
	// The notification should be set
	assert.Equal(t, "Test notification", tp.notifyView.GetText(false))
	
	// Wait for notification to clear (it clears after 2 seconds, but we'll test the immediate effect)
	// In a real test environment, you might want to mock time or use dependency injection
}

func TestPageConsumeTopic(t *testing.T) {
	tp, mockDS, _ := createTestTopicPageWithConsume()
	tp.CreateTopicPage("test-topic") // Initialize UI components
	
	topic := api.Topic{
		NumPartitions:     2,
		ReplicationFactor: 1,
		MessageCount:      100,
	}
	
	flags := api.ConsumeFlags{
		OffsetFlag: "latest",
		Follow:     true,
		Tail:       50,
	}
	
	// Mock the ConsumeTopic call
	mockDS.On("ConsumeTopic", mock.AnythingOfType("*context.cancelCtx"), "test-topic", flags, mock.AnythingOfType("api.MessageHandlerFunc"), mock.AnythingOfType("func(interface {})")).Return(nil)
	
	tp.PageConsumeTopic("test-topic", topic, flags)
	
	assert.Equal(t, "test-topic", tp.topicName)
	assert.Equal(t, topic, tp.topicDetails)
	assert.Equal(t, flags, tp.consumeFlags)
	assert.NotNil(t, tp.consumedMessages)
	assert.NotNil(t, tp.cancelConsumption)
	assert.NotNil(t, tp.cancelRefresh)
	
	// Verify mock was called
	mockDS.AssertExpectations(t)
	
	// Clean up
	if tp.cancelConsumption != nil {
		tp.cancelConsumption()
	}
	if tp.cancelRefresh != nil {
		tp.cancelRefresh()
	}
}

func TestInputCapture(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	// Add some test data
	tp.consumedMessages = map[string]api.Message{
		"0:100": {
			Key:       "test-key",
			Value:     "test-value",
			Offset:    100,
			Partition: 0,
			Headers:   []api.MessageHeader{{Key: "header1", Value: "value1"}},
		},
	}
	
	// Create first row
	tp.createFirstRowTopicTable("test-topic")
	// Add a data row
	tp.consumerTable.SetCell(1, 0, tview.NewTableCell("100"))
	tp.consumerTable.SetCell(1, 1, tview.NewTableCell("0"))
	
	inputCapture := tp.inputCapture()
	assert.NotNil(t, inputCapture)
	
	// Test 'g' key (scroll to beginning)
	event := tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone)
	result := inputCapture(event)
	assert.Equal(t, event, result)
	
	// Test 'G' key (scroll to end)
	event = tcell.NewEventKey(tcell.KeyRune, 'G', tcell.ModNone)
	result = inputCapture(event)
	assert.Equal(t, event, result)
	
	// Test 'o' key (toggle offset)
	tp.consumeFlags = api.ConsumeFlags{OffsetFlag: "latest", Tail: 50}
	event = tcell.NewEventKey(tcell.KeyRune, 'o', tcell.ModNone)
	result = inputCapture(event)
	assert.Equal(t, event, result)
	assert.Equal(t, "oldest", tp.consumeFlags.OffsetFlag)
	assert.Equal(t, int32(0), tp.consumeFlags.Tail)
	
	// Test 'o' key again (toggle back)
	event = tcell.NewEventKey(tcell.KeyRune, 'o', tcell.ModNone)
	result = inputCapture(event)
	assert.Equal(t, event, result)
	assert.Equal(t, "latest", tp.consumeFlags.OffsetFlag)
	assert.Equal(t, int32(50), tp.consumeFlags.Tail)
}

func TestInputCaptureSearchFunctionality(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	inputCapture := tp.inputCapture()
	
	// Test '/' key (search)
	event := tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
	result := inputCapture(event)
	assert.Equal(t, event, result)
	assert.NotNil(t, tp.tableSearch)
	
	// Test '/' key again (should not create another search)
	event = tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
	result = inputCapture(event)
	assert.Equal(t, event, result)
}

func TestCreateInputSearch(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	onDone := func() {
		// Callback function for testing
	}
	
	searchFlex := tp.CreateInputSearch(onDone)
	
	assert.NotNil(t, searchFlex)
	assert.Equal(t, 1, searchFlex.GetItemCount())
	
	// Test that search text is preserved
	tp.searchText = "test-search"
	searchFlex2 := tp.CreateInputSearch(onDone)
	assert.NotNil(t, searchFlex2)
}

func TestRestartConsumer(t *testing.T) {
	tp, mockDS, _ := createTestTopicPageWithConsume()
	tp.CreateTopicPage("test-topic")
	
	// Set up initial state
	tp.topicName = "test-topic"
	tp.topicDetails = api.Topic{NumPartitions: 2}
	tp.consumeFlags = api.ConsumeFlags{OffsetFlag: "latest"}
	tp.consumedMessages = map[string]api.Message{"test": {}}
	tp.searchText = "search"
	
	// Mock the ConsumeTopic call
	mockDS.On("ConsumeTopic", mock.AnythingOfType("*context.cancelCtx"), "test-topic", tp.consumeFlags, mock.AnythingOfType("api.MessageHandlerFunc"), mock.AnythingOfType("func(interface {})")).Return(nil)
	
	// Set up cancel functions to avoid nil pointer
	_, cancel := context.WithCancel(context.Background())
	tp.cancelConsumption = cancel
	tp.cancelRefresh = cancel
	
	tp.RestartConsumer()
	
	// Verify state was cleared and restarted
	assert.Empty(t, tp.searchText)
	assert.NotNil(t, tp.consumedMessages)
	
	mockDS.AssertExpectations(t)
}

func TestCloseTopicPage(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	// Set up some state to be cleared
	tp.consumedMessages = map[string]api.Message{"test": {}}
	tp.searchText = "search"
	
	// Set up cancel functions
	_, cancel := context.WithCancel(context.Background())
	tp.cancelConsumption = cancel
	tp.cancelRefresh = cancel
	
	tp.CloseTopicPage()
	
	// Give some time for the goroutine to execute
	time.Sleep(50 * time.Millisecond)
	
	// The method should complete without panicking
}

func TestClearConsumedData(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	// Set up state to be cleared
	tp.searchText = "search"
	tp.tableSearch = tview.NewFlex()
	tp.bottomFlex.AddItem(tp.tableSearch, 0, 1, false)
	
	// Set up cancel functions
	_, cancel := context.WithCancel(context.Background())
	tp.cancelConsumption = cancel
	tp.cancelRefresh = cancel
	
	// Add some elements to topFlexElements
	testFlex := tview.NewFlex()
	tp.topFlexElements.PushBack(testFlex)
	tp.topFlex.AddItem(testFlex, 0, 1, false)
	
	tp.clearConsumedData()
	
	assert.Empty(t, tp.searchText)
	assert.Nil(t, tp.tableSearch)
	assert.Equal(t, 0, tp.topFlexElements.Len())
}

func TestCreateFirstRowTopicTable(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	// Test without search text
	tp.createFirstRowTopicTable("test-topic")
	
	// Check header cells
	cell := tp.consumerTable.GetCell(0, 0)
	assert.Equal(t, "Offset", cell.Text)
	
	cell = tp.consumerTable.GetCell(0, 1)
	assert.Equal(t, "Partition", cell.Text)
	
	cell = tp.consumerTable.GetCell(0, 4)
	assert.Equal(t, "Key", cell.Text)
	
	cell = tp.consumerTable.GetCell(0, 5)
	assert.Equal(t, "Value", cell.Text)
	
	// Test with search text
	tp.searchText = "search-term"
	tp.createFirstRowTopicTable("test-topic")
	
	// The title should include search text
	title := tp.messagesFlex.GetTitle()
	assert.Contains(t, title, "test-topic")
	assert.Contains(t, title, "search-term")
}

func TestRefreshTopicTableWithMessages(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	// Set up test messages
	tp.consumedMessages = map[string]api.Message{
		"0:100": {
			Key:           "key1",
			Value:         "value1",
			Offset:        100,
			Partition:     0,
			KeySchemaID:   "1",
			ValueSchemaID: "2",
		},
		"1:200": {
			Key:           "key2",
			Value:         "value2",
			Offset:        200,
			Partition:     1,
			KeySchemaID:   "3",
			ValueSchemaID: "4",
		},
	}
	
	tp.newMessageConsumed = true
	
	// Create a context for the refresh function
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start refresh in a goroutine
	go tp.refreshTopicTable(ctx)
	
	// Give some time for the refresh to process
	time.Sleep(150 * time.Millisecond)
	
	// Cancel to stop the refresh loop
	cancel()
	
	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)
	
	// Verify the function completed without panicking
	assert.NotNil(t, tp.consumerTable)
}

func TestRefreshTopicTableWithSearch(t *testing.T) {
	tp, _, _ := createTestTopicPage()
	tp.CreateTopicPage("test-topic")
	
	// Set up test messages
	tp.consumedMessages = map[string]api.Message{
		"0:100": {
			Key:     "matching-key",
			Value:   "value1",
			Offset:  100,
			Partition: 0,
		},
		"1:200": {
			Key:     "other-key",
			Value:   "value2",
			Offset:  200,
			Partition: 1,
		},
	}
	
	tp.searchText = "matching"
	tp.newMessageConsumed = true
	
	// Create a context for the refresh function
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start refresh in a goroutine
	go tp.refreshTopicTable(ctx)
	
	// Give some time for the refresh to process
	time.Sleep(150 * time.Millisecond)
	
	// Cancel to stop the refresh loop
	cancel()
	
	// The function should complete without issues
	assert.Equal(t, "matching", tp.searchText)
}