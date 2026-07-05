package router

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/stretchr/testify/assert"
)

// mockDataSource implements api.KafkaDataSource for testing
type mockDataSource struct{}

func (m *mockDataSource) GetTopics() (map[string]api.Topic, error) {
	return map[string]api.Topic{
		"test-topic": {
			NumPartitions:     3,
			ReplicationFactor: 1,
			ReplicaAssignment: make(map[int32][]int32),
			ConfigEntries:     make(map[string]*string),
		},
	}, nil
}

func (m *mockDataSource) GetMessages(topicName string, partition int32, offset int64, limit int) ([]api.Message, error) {
	return []api.Message{
		{
			Key:       "test-key",
			Value:     "test-value",
			Offset:    offset,
			Partition: partition,
		},
	}, nil
}

func (m *mockDataSource) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handler api.MessageHandlerFunc, stopCallback func(interface{})) error {
	// Mock implementation - just call handler with test message
	message := api.Message{
		Key:       "test-key",
		Value:     "test-value",
		Offset:    0,
		Partition: 0,
	}
	handler(message)
	return nil
}

func (m *mockDataSource) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	return nil
}

func (m *mockDataSource) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	return []api.ConsumerGroup{}, nil
}

func (m *mockDataSource) Init(cfgOption string) {}

func (m *mockDataSource) GetContexts() ([]string, error) {
	return []string{"default"}, nil
}

func (m *mockDataSource) GetContext() string {
	return "default"
}

func (m *mockDataSource) SetContext(contextName string) error {
	return nil
}

func (m *mockDataSource) GetClusterDetails(clusterName string) (api.ClusterInfo, error) {
	return api.ClusterInfo{Name: clusterName}, nil
}

func (m *mockDataSource) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	return nil, nil
}

func (m *mockDataSource) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (m *mockDataSource) GetSchemas() ([]api.Schema, error) {
	return []api.Schema{}, nil
}

func (m *mockDataSource) GetSchemaDetails(subjects []string) ([]api.Schema, error) {
	return []api.Schema{}, nil
}

func (m *mockDataSource) GetSchemaContent(subject string, version int) (string, error) {
	return `{"type":"record","name":"Mock","fields":[]}`, nil
}

func (m *mockDataSource) GetSchemaVersions(subject string) ([]api.SchemaVersion, error) {
	return nil, nil
}
func (m *mockDataSource) GetGlobalCompatibility() (api.CompatibilityLevel, error) { return "", nil }
func (m *mockDataSource) GetSubjectCompatibility(subject string) (api.CompatibilityLevel, bool, error) {
	return "", false, nil
}
func (m *mockDataSource) RegisterSchema(subject, schemaText, schemaType string) (api.Schema, error) {
	return api.Schema{}, nil
}
func (m *mockDataSource) CheckSchemaCompatibility(subject, schemaText, schemaType string) (bool, []string, error) {
	return true, nil, nil
}
func (m *mockDataSource) DeleteSubject(subject string, permanent bool) ([]int, error) {
	return nil, nil
}
func (m *mockDataSource) DeleteSchemaVersion(subject string, version int, permanent bool) error {
	return nil
}
func (m *mockDataSource) SetGlobalCompatibility(level api.CompatibilityLevel) error { return nil }
func (m *mockDataSource) SetSubjectCompatibility(subject string, level api.CompatibilityLevel) error {
	return nil
}

func (m *mockDataSource) GetACLs() ([]api.ACLEntry, error) {
	return []api.ACLEntry{}, nil
}

func (m *mockDataSource) GetACLsFiltered(filter api.ACLFilter) ([]api.ACLEntry, error) {
	return []api.ACLEntry{}, nil
}

func (m *mockDataSource) CreateACL(entry api.ACLEntry) error { return nil }

func (m *mockDataSource) DeleteACL(entry api.ACLEntry) error { return nil }

func (m *mockDataSource) GetClientQuotas() ([]api.ClientQuotaEntry, error) {
	return []api.ClientQuotaEntry{}, nil
}

func (m *mockDataSource) AlterClientQuotas(entity api.ClientQuotaEntity, quotas map[string]float64) error {
	return nil
}

func (m *mockDataSource) GetTopicNames() ([]string, error) {
	topics, err := m.GetTopics()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(topics))
	for name := range topics {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockDataSource) DecodeMessage(_ context.Context, msg api.Message) (api.Message, error) {
	return msg, nil
}

func (m *mockDataSource) ListSerdes() []string { return []string{"string", "hex", "json"} }

func (m *mockDataSource) GetClusterStatistics(_ context.Context, _ string) (api.ClusterStatistics, error) {
	return api.ClusterStatistics{}, nil
}

func (m *mockDataSource) GetClusterCapabilities(_ context.Context, _ string) ([]api.Capability, error) {
	return nil, nil
}

func (m *mockDataSource) ValidateClusterConnection(_ context.Context, _ string) ([]api.ValidationResult, error) {
	return nil, nil
}
func (m *mockDataSource) GetBrokers() ([]api.BrokerInfo, error) { return nil, nil }
func (m *mockDataSource) GetBrokerStats() (map[int32]api.BrokerStats, api.BrokerSummary, error) {
	return nil, api.BrokerSummary{}, nil
}
func (m *mockDataSource) GetBrokerLogDirs(brokerIDs []int32) (map[int32][]api.BrokerLogDir, error) {
	return nil, nil
}
func (m *mockDataSource) GetBrokerConfig(brokerID int32) ([]api.BrokerConfigEntry, error) {
	return nil, nil
}
func (m *mockDataSource) AlterBrokerConfig(brokerID int32, key, value string) error { return nil }
func (m *mockDataSource) AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error {
	return nil
}
func (m *mockDataSource) GetBrokerMetrics(brokerID int32) (string, error) { return "", nil }

// Topic-administration + analysis stubs (TP-1..TP-11, TP-29/TP-30).
func (m *mockDataSource) GetTopicConfig(topicName string) ([]api.TopicConfigEntry, error) {
	return nil, nil
}
func (m *mockDataSource) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	return api.TopicDetails{}, nil
}
func (m *mockDataSource) GetTopicSizes(topicNames []string) (map[string]int64, error) {
	return nil, nil
}
func (m *mockDataSource) CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error {
	return nil
}
func (m *mockDataSource) DeleteTopic(name string) error         { return nil }
func (m *mockDataSource) IsTopicDeletionEnabled() (bool, error) { return true, nil }
func (m *mockDataSource) UpdateTopicConfig(name string, entries map[string]*string) error {
	return nil
}
func (m *mockDataSource) IncreasePartitions(name string, totalCount int32) error { return nil }
func (m *mockDataSource) PurgeTopicMessages(name string, partition int32) error  { return nil }
func (m *mockDataSource) RecreateTopic(name string) error                        { return nil }
func (m *mockDataSource) ChangeReplicationFactor(name string, newFactor int16) error {
	return nil
}
func (m *mockDataSource) StartTopicAnalysis(ctx context.Context, topicName string) error {
	return nil
}
func (m *mockDataSource) GetTopicAnalysis(topicName string) (*api.TopicAnalysis, error) {
	return nil, nil
}
func (m *mockDataSource) CancelTopicAnalysis(topicName string) error { return nil }

func (m *mockDataSource) GetConnectClusters(withStats bool) ([]api.ConnectCluster, error) {
	return nil, nil
}
func (m *mockDataSource) GetConnectorNames(connect string) ([]string, error) { return nil, nil }
func (m *mockDataSource) GetConnectors() ([]api.Connector, error)            { return nil, nil }
func (m *mockDataSource) GetConnectorDetails(connect, name string) (api.ConnectorDetails, error) {
	return api.ConnectorDetails{}, nil
}
func (m *mockDataSource) CreateConnector(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (m *mockDataSource) UpdateConnectorConfig(connect, name string, config map[string]string) (api.Connector, error) {
	return api.Connector{}, nil
}
func (m *mockDataSource) DeleteConnector(connect, name string) error            { return nil }
func (m *mockDataSource) PauseConnector(connect, name string) error             { return nil }
func (m *mockDataSource) ResumeConnector(connect, name string) error            { return nil }
func (m *mockDataSource) StopConnector(connect, name string) error              { return nil }
func (m *mockDataSource) RestartConnector(connect, name string) error           { return nil }
func (m *mockDataSource) RestartConnectorTask(connect, name string, taskID int) error {
	return nil
}
func (m *mockDataSource) ResetConnectorOffsets(connect, name string) error { return nil }
func (m *mockDataSource) GetConnectorPlugins(connect string) ([]api.ConnectorPlugin, error) {
	return nil, nil
}
func (m *mockDataSource) ValidateConnectorConfig(connect, pluginClass string, config map[string]string) (api.ConnectorValidationResult, error) {
	return api.ConnectorValidationResult{}, nil
}
func (m *mockDataSource) ListKsqlStreams() ([]api.KsqlStream, error) { return nil, nil }
func (m *mockDataSource) ListKsqlTables() ([]api.KsqlTable, error)   { return nil, nil }
func (m *mockDataSource) ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	return nil, nil
}
func (m *mockDataSource) GetConsumerGroupDetail(groupID string) (api.ConsumerGroupDetail, error) {
	return api.ConsumerGroupDetail{}, nil
}
func (m *mockDataSource) GetConsumerGroupDetails(groupIDs []string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (m *mockDataSource) GetConsumerGroupsForTopic(topic string) ([]api.ConsumerGroup, error) {
	return nil, nil
}
func (m *mockDataSource) DeleteConsumerGroup(groupID string) error               { return nil }
func (m *mockDataSource) DeleteConsumerGroupOffsets(groupID, topic string) error { return nil }
func (m *mockDataSource) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	return nil
}

type mockResourceItem struct {
	id      string
	details map[string]string
}

func (m *mockResourceItem) GetID() string {
	return m.id
}

func (m *mockResourceItem) GetValues() []string {
	return []string{m.id}
}

func (m *mockResourceItem) GetDetails() map[string]string {
	return m.details
}

func TestNewRouter(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	// Note: We can't directly compare interfaces, so we'll skip this check

	if router.currentPage != "main" {
		t.Errorf("Expected initial page to be 'main', got '%s'", router.currentPage)
	}

	if len(router.pages) != 0 {
		t.Errorf("Expected empty pages map, got %d pages", len(router.pages))
	}

	if len(router.history) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(router.history))
	}
}

func TestNavigateTo(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	tests := []struct {
		name     string
		pageID   string
		data     interface{}
		expected string
	}{
		{
			name:     "Navigate to main page",
			pageID:   "main",
			data:     nil,
			expected: "main",
		},
		{
			name:   "Navigate to topic page with data",
			pageID: "topic",
			data: &NavigationData{
				TopicName: "test-topic",
				Topic: api.Topic{
					NumPartitions:     3,
					ReplicationFactor: 1,
				},
			},
			expected: "topic:test-topic",
		},
		{
			name:   "Navigate to message detail page",
			pageID: "message_detail",
			data: &NavigationData{
				TopicName: "test-topic",
				Message: api.Message{
					Partition: 0,
					Offset:    0,
					Key:       "test-key",
					Value:     "test-value",
				},
			},
			expected: "detail:test-topic:0:0",
		},
		{
			name:   "Navigate to resource detail page",
			pageID: "resource_detail",
			data: &NavigationData{
				ResourceItem: &mockResourceItem{
					id:      "test-resource",
					details: map[string]string{"Type": "Topic"},
				},
				ResourceType: "topic",
			},
			expected: "resource_detail:test-resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router.NavigateTo(tt.pageID, tt.data)

			// Router stores page by the pageID passed to NavigateTo
			// The page's GetID() may return a different dynamic ID
			if router.GetCurrentPageID() != tt.pageID {
				t.Errorf("Expected current page to be '%s', got '%s'", tt.pageID, router.GetCurrentPageID())
			}

			// Verify page was created
			page := router.GetCurrentPage()
			if page == nil {
				t.Error("Expected page to be created, got nil")
			}
		})
	}
}

func TestNavigationHistory(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate through several pages
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", &NavigationData{TopicName: "test-topic"})
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// Check history
	history := router.GetHistory()
	expectedHistory := []string{"main", "topic"}

	if len(history) != len(expectedHistory) {
		t.Errorf("Expected history length %d, got %d", len(expectedHistory), len(history))
	}

	for i, expected := range expectedHistory {
		if i >= len(history) || history[i] != expected {
			t.Errorf("Expected history[%d] to be '%s', got '%s'", i, expected, history[i])
		}
	}

	// Test back navigation
	cmd := router.Back()
	if cmd != nil {
		// Execute the command to complete navigation
		router.Update(cmd())
	}
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page after back to be 'topic', got '%s'", router.GetCurrentPageID())
	}

	cmd = router.Back()
	if cmd != nil {
		router.Update(cmd())
	}
	if router.GetCurrentPageID() != "main" {
		t.Errorf("Expected current page after second back to be 'main', got '%s'", router.GetCurrentPageID())
	}

	// Back from main should do nothing
	cmd = router.Back()
	if cmd != nil {
		router.Update(cmd())
	}
	if router.GetCurrentPageID() != "main" {
		t.Errorf("Expected current page to remain 'main', got '%s'", router.GetCurrentPageID())
	}
}

func TestBreadcrumbsUpdateOnBack(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	router.NavigateTo("main", nil)
	router.NavigateTo("topic", &NavigationData{TopicName: "test-topic"})
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	if got := len(router.getBreadcrumbs()); got != 3 {
		t.Fatalf("expected 3 breadcrumbs after deep navigation, got %d", got)
	}

	// Back() must batch a breadcrumb update and shrink the trail.
	cmd := router.Back()
	if cmd == nil {
		t.Fatal("Back() returned nil command")
	}
	router.Update(cmd())
	if got := len(router.getBreadcrumbs()); got != 2 {
		t.Fatalf("expected 2 breadcrumbs after Back(), got %d", got)
	}
}

func TestSetDimensions(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Create a page first
	router.NavigateTo("main", nil)

	// Set dimensions
	width, height := 100, 50
	router.SetDimensions(width, height)

	if router.width != width {
		t.Errorf("Expected router width to be %d, got %d", width, router.width)
	}

	if router.height != height {
		t.Errorf("Expected router height to be %d, got %d", height, router.height)
	}

	// Verify dimensions were propagated to pages
	// Note: We can't easily test this without exposing page internals,
	// but we can verify the method doesn't panic
}

func TestClearHistory(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Build up some history
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", nil)
	router.NavigateTo("message_detail", nil)

	// Verify history exists
	if len(router.GetHistory()) == 0 {
		t.Error("Expected history to have entries before clearing")
	}

	// Clear history
	router.ClearHistory()

	// Verify history is empty
	if len(router.GetHistory()) != 0 {
		t.Errorf("Expected empty history after clearing, got %d entries", len(router.GetHistory()))
	}
}

func TestRouterUpdate(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Initialize router
	router.NavigateTo("main", nil)

	// Test PageChangeMsg handling
	pageChangeMsg := core.PageChangeMsg{
		PageID: "topic",
		Data: map[string]interface{}{
			"name": "test-topic",
			"topic": api.Topic{
				NumPartitions:     3,
				ReplicationFactor: 1,
			},
		},
	}

	// Update router with page change message
	updatedRouter, _ := router.Update(pageChangeMsg)

	// Verify router was updated
	if updatedRouter == nil {
		t.Error("Expected updated router, got nil")
	}

	// Note: cmd might be nil if navigation is immediate, which is fine

	// Verify current page changed
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page to be 'topic', got '%s'", router.GetCurrentPageID())
	}
}

func TestRouterView(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Test view with no current page
	view := router.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Navigate to main page and test view
	router.NavigateTo("main", nil)
	view = router.View()
	if view == "" {
		t.Error("Expected non-empty view after navigation")
	}
}

func TestCreatePageFallbacks(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	tests := []struct {
		name   string
		pageID string
		data   interface{}
	}{
		{
			name:   "Topic page with nil data",
			pageID: "topic",
			data:   nil,
		},
		{
			name:   "Message detail page with nil data",
			pageID: "message_detail",
			data:   nil,
		},
		{
			name:   "Resource detail page with nil data",
			pageID: "resource_detail",
			data:   nil,
		},
		{
			name:   "Unknown page ID",
			pageID: "unknown",
			data:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic and should create a page
			router.NavigateTo(tt.pageID, tt.data)

			page := router.GetCurrentPage()
			if page == nil {
				t.Error("Expected page to be created even with nil data")
			}
		})
	}
}

// TestPageChangeMsgWithNilDataDoesNotPanic guards against a nil-pointer-
// dereference crash: PageChangeMsg with nil Data used to make Router.Update
// box a typed-nil *NavigationData into the interface{} passed to NavigateTo.
// createPage's `data, ok := data.(*NavigationData)` assertions succeed on
// that (the dynamic type matches *NavigationData even though the pointer is
// nil), so every case dereferencing a field on it (e.g. "ksql_query"'s
// navData.TopicName) panicked. Reproduced live via `K` then `e` (open the
// ksqlDB query editor with no topic seed).
func TestPageChangeMsgWithNilDataDoesNotPanic(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))
	router.NavigateTo("main", nil)

	for _, pageID := range []string{"topic", "message_detail", "resource_detail", "schema_detail", "ksql_query", "broker:1"} {
		assert.NotPanics(t, func() {
			router.Update(core.NewPageChangeMsg(pageID, nil)())
		}, "PageChangeMsg(%q, nil) should not panic", pageID)
	}
}

// TestBackMsgHandling verifies that BackMsg doesn't add to history
func TestBackMsgHandling(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate: main -> topic -> message_detail
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", &NavigationData{TopicName: "test-topic"})
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// Verify initial history
	initialHistory := router.GetHistory()
	expectedInitialLen := 2 // [main, topic]
	if len(initialHistory) != expectedInitialLen {
		t.Errorf("Expected initial history length %d, got %d", expectedInitialLen, len(initialHistory))
	}

	// Simulate BackMsg (like pressing Esc on message detail)
	backMsg := core.BackMsg{}
	router.Update(backMsg)

	// After back, should be on topic page
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page to be 'topic' after BackMsg, got '%s'", router.GetCurrentPageID())
	}

	// History should NOT have grown - it should have one less entry
	historyAfterBack := router.GetHistory()
	expectedAfterBackLen := 1 // [main]
	if len(historyAfterBack) != expectedAfterBackLen {
		t.Errorf("Expected history length %d after back, got %d. History: %v", expectedAfterBackLen, len(historyAfterBack), historyAfterBack)
	}

	// Now simulate going forward again: topic -> message_detail
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// History should be: [main, topic]
	historyAfterForward := router.GetHistory()
	expectedAfterForwardLen := 2 // [main, topic]
	if len(historyAfterForward) != expectedAfterForwardLen {
		t.Errorf("Expected history length %d after forward, got %d. History: %v", expectedAfterForwardLen, len(historyAfterForward), historyAfterForward)
	}

	// Press Esc again (BackMsg)
	router.Update(backMsg)

	// Should be back on topic
	if router.GetCurrentPageID() != "topic" {
		t.Errorf("Expected current page to be 'topic' after second BackMsg, got '%s'", router.GetCurrentPageID())
	}

	// History should still be: [main] (not growing)
	finalHistory := router.GetHistory()
	expectedFinalLen := 1 // [main]
	if len(finalHistory) != expectedFinalLen {
		t.Errorf("Expected final history length %d, got %d. History: %v", expectedFinalLen, len(finalHistory), finalHistory)
	}
}

// TestHistoryDoesNotGrow verifies that back-and-forth navigation doesn't create history loops
func TestHistoryDoesNotGrow(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate: main -> topic -> message_detail
	router.NavigateTo("main", nil)
	router.NavigateTo("topic", &NavigationData{TopicName: "test-topic"})
	router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})

	// Record baseline history length
	baselineHistoryLen := len(router.GetHistory())

	// Simulate multiple back-and-forth navigations
	for i := 0; i < 5; i++ {
		// Go back (Esc)
		router.Update(core.BackMsg{})
		if router.GetCurrentPageID() != "topic" {
			t.Errorf("Iteration %d: Expected 'topic' after back, got '%s'", i, router.GetCurrentPageID())
		}

		// Go forward again
		router.NavigateTo("message_detail", &NavigationData{TopicName: "test-topic"})
		if router.GetCurrentPageID() != "message_detail" {
			t.Errorf("Iteration %d: Expected 'message_detail' after forward, got '%s'", i, router.GetCurrentPageID())
		}
	}

	// History should not have grown beyond the baseline
	finalHistoryLen := len(router.GetHistory())
	if finalHistoryLen != baselineHistoryLen {
		t.Errorf("History grew from %d to %d after back-and-forth navigation. History: %v", baselineHistoryLen, finalHistoryLen, router.GetHistory())
	}
}

// TestHistoryDoesNotGrowOnForwardReturnNavigation guards against bug #8
// regressing: a page that returns to its caller via forward navigation
// (PageChangeMsg) instead of Router.Back() — e.g. the Clusters dashboard
// switching context and sending the user back to "main" — must not leave a
// dangling history entry. Repeating that round trip used to grow history (and
// the breadcrumb bar) without bound.
func TestHistoryDoesNotGrowOnForwardReturnNavigation(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))
	router.NavigateTo("main", nil)

	for i := 0; i < 5; i++ {
		router.NavigateTo("clusters", nil)
		// Simulate clusters_page.go's openSelected(): return to main via a
		// PageChangeMsg (forward navigation), not router.Back().
		router.Update(core.NewPageChangeMsg("main", nil)())
	}

	history := router.GetHistory()
	if len(history) > 1 {
		t.Errorf("history grew to %v after 5 open/return cycles, want length <= 1", history)
	}
	if got := len(router.getBreadcrumbs()); got > 2 {
		t.Errorf("breadcrumb has %d items after 5 open/return cycles, want <= 2", got)
	}
}

// TestRouter_DynamicPageIDs tests that the router correctly handles dynamic page IDs
func TestRouter_DynamicPageIDs(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Test topic page with dynamic ID
	router.NavigateTo("topic:my-topic", &NavigationData{TopicName: "my-topic"})
	assert.Equal(t, "topic:my-topic", router.GetCurrentPageID())

	// Test message detail page with dynamic ID
	router.NavigateTo("detail:my-topic:0:123", &NavigationData{
		TopicName: "my-topic",
		Message: api.Message{
			Partition: 0,
			Offset:    123,
		},
	})
	assert.Equal(t, "detail:my-topic:0:123", router.GetCurrentPageID())

	// Test resource detail page with dynamic ID
	router.NavigateTo("resource_detail:group-1", &NavigationData{
		ResourceType: "consumer-group",
	})
	assert.Equal(t, "resource_detail:group-1", router.GetCurrentPageID())
}

// TestRouter_BaseIDExtraction tests that the router correctly extracts base IDs
func TestRouter_BaseIDExtraction(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	testCases := []struct {
		dynamicID string
		baseID    string
		pageType  string
	}{
		{"topic:my-topic", "topic", "topic"},
		{"topic:another-topic", "topic", "topic"},
		{"detail:topic:0:123", "detail", "message_detail"},
		{"detail:topic:1:456", "detail", "message_detail"},
		{"resource_detail:group-1", "resource_detail", "resource_detail"},
		{"main", "main", "main"},
	}

	for _, tc := range testCases {
		t.Run(tc.dynamicID, func(t *testing.T) {
			router.NavigateTo(tc.dynamicID, nil)

			// Verify current page ID is preserved (full dynamic ID)
			assert.Equal(t, tc.dynamicID, router.GetCurrentPageID())

			// Verify page was created (not nil)
			currentPage := router.GetCurrentPage()
			assert.NotNil(t, currentPage)
		})
	}
}

// TestRouter_DifferentMessageIDs tests that different message IDs create different pages
func TestRouter_DifferentMessageIDs(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Navigate to first message
	router.NavigateTo("detail:topic1:0:100", &NavigationData{
		TopicName: "topic1",
		Message: api.Message{
			Partition: 0,
			Offset:    100,
		},
	})

	page1 := router.GetCurrentPage()
	page1ID := router.GetCurrentPageID()

	// Navigate to different message
	router.NavigateTo("detail:topic1:0:200", &NavigationData{
		TopicName: "topic1",
		Message: api.Message{
			Partition: 0,
			Offset:    200,
		},
	})

	page2 := router.GetCurrentPage()
	page2ID := router.GetCurrentPageID()

	// Should have different page IDs
	assert.NotEqual(t, page1ID, page2ID)
	assert.Contains(t, page1ID, "100")
	assert.Contains(t, page2ID, "200")

	// Both should be message detail pages (same type, different instances)
	assert.NotNil(t, page1)
	assert.NotNil(t, page2)
}

// TestRouter_NavigationWithUniqueTopicIDs tests navigation between different topics
func TestRouter_NavigationWithUniqueTopicIDs(t *testing.T) {
	dataSource := &mockDataSource{}
	router := NewRouter(core.NewCommon(dataSource))

	// Start at main
	router.NavigateTo("main", nil)
	assert.Equal(t, "main", router.GetCurrentPageID())

	// Navigate to first topic
	router.NavigateTo("topic:topic-1", &NavigationData{TopicName: "topic-1"})
	assert.Equal(t, "topic:topic-1", router.GetCurrentPageID())

	// Navigate to second topic
	router.NavigateTo("topic:topic-2", &NavigationData{TopicName: "topic-2"})
	assert.Equal(t, "topic:topic-2", router.GetCurrentPageID())

	// Navigate back
	router.Update(core.BackMsg{})
	assert.Equal(t, "topic:topic-1", router.GetCurrentPageID())

	// Navigate back again
	router.Update(core.BackMsg{})
	assert.Equal(t, "main", router.GetCurrentPageID())
}
