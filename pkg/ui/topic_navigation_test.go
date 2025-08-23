package ui

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/stretchr/testify/assert"
)

// TestTopicNavigationAfterTableRefactor tests that topic navigation works after converting from list to table
func TestTopicNavigationAfterTableRefactor(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}

	// Create main UI model
	mainModel := NewMainPage(mockDS)
	uiModel := Model{
		dataSource:  mockDS,
		currentPage: mainPage,
		mainPage:    &mainModel,
		width:       120,
		height:      40,
	}

	// Load some mock topics
	topics := map[string]api.Topic{
		"test-topic-1": {
			NumPartitions:     3,
			ReplicationFactor: 1,
			ConfigEntries:     make(map[string]*string),
		},
		"test-topic-2": {
			NumPartitions:     5,
			ReplicationFactor: 2,
			ConfigEntries:     make(map[string]*string),
		},
	}

	// Simulate topic loading
	items := make([]interface{}, 0, len(topics))
	for name, topic := range topics {
		items = append(items, topicItem{
			name:  name,
			topic: topic,
		})
	}

	// Update main page with topics
	topicListMessage := topicListMsg(items)
	updatedModel, _ := uiModel.mainPage.Update(topicListMessage)
	uiModel.mainPage = updatedModel.(*MainPageModel)

	// Verify topics are loaded
	assert.Len(t, uiModel.mainPage.allItems, 2, "Should have 2 topics loaded")
	assert.Len(t, uiModel.mainPage.allRows, 2, "Should have 2 table rows")

	// Simulate selecting the first topic in the table
	uiModel.mainPage.resourcesTable.GotoTop()

	// Verify we can get the selected item
	selectedItem := uiModel.mainPage.getSelectedResourceItem()
	assert.NotNil(t, selectedItem, "Should be able to get selected item")

	// Check that the selected item is a topicItem with correct data
	if topicItem, ok := selectedItem.(topicItem); ok {
		assert.NotEmpty(t, topicItem.name, "Topic name should not be empty")
		assert.Greater(t, topicItem.topic.NumPartitions, int32(0), "Topic should have partitions")
		t.Logf("Selected topic: %s with %d partitions", topicItem.name, topicItem.topic.NumPartitions)
	} else {
		t.Errorf("Selected item should be a topicItem, got %T", selectedItem)
	}

	// Simulate pressing enter to navigate to topic page
	pageChangeMessage := pageChangeMsg(topicPage)
	updatedUIModel, _ := uiModel.Update(pageChangeMessage)
	uiModel = updatedUIModel.(Model)

	// Verify navigation worked
	assert.Equal(t, topicPage, uiModel.currentPage, "Should be on topic page")
	assert.NotNil(t, uiModel.topicPage, "Topic page should be initialized")

	if uiModel.topicPage != nil {
		assert.NotEmpty(t, uiModel.topicPage.topicName, "Topic page should have a topic name")
		assert.Greater(t, uiModel.topicPage.topicDetails.NumPartitions, int32(0), "Topic page should have topic details")
		t.Logf("Topic page initialized for: %s", uiModel.topicPage.topicName)
	}
}

// TestFilteredTopicNavigation tests navigation when topics are filtered
func TestFilteredTopicNavigation(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}

	// Create main UI model
	mainModel := NewMainPage(mockDS)

	// Load some mock topics
	topics := map[string]api.Topic{
		"user-events": {
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
		"order-processing": {
			NumPartitions:     5,
			ReplicationFactor: 2,
		},
		"user-analytics": {
			NumPartitions:     2,
			ReplicationFactor: 1,
		},
	}

	// Simulate topic loading
	items := make([]interface{}, 0, len(topics))
	for name, topic := range topics {
		items = append(items, topicItem{
			name:  name,
			topic: topic,
		})
	}

	// Update main page with topics
	topicListMessage := topicListMsg(items)
	updatedModel, _ := mainModel.Update(topicListMessage)
	mainModel = *updatedModel.(*MainPageModel)

	// Apply a filter to show only "user" topics
	searchMessage := searchTopicsMsg("user")
	updatedModel, _ = mainModel.Update(searchMessage)
	mainModel = *updatedModel.(*MainPageModel)

	// Verify filtering worked
	assert.True(t, mainModel.isFiltered, "Should be in filtered state")
	assert.Len(t, mainModel.filteredItems, 2, "Should have 2 filtered items (user-events, user-analytics)")
	assert.Len(t, mainModel.filteredRows, 2, "Should have 2 filtered rows")

	// Select first filtered topic
	mainModel.resourcesTable.GotoTop()
	selectedItem := mainModel.getSelectedResourceItem()
	assert.NotNil(t, selectedItem, "Should be able to get selected filtered item")

	// The selected item should be a highlighted topic item
	switch item := selectedItem.(type) {
	case HighlightedTopicItem:
		assert.Contains(t, item.name, "user", "Filtered item should contain 'user'")
		assert.Greater(t, item.topic.NumPartitions, int32(0), "Filtered topic should have partitions")
		t.Logf("Selected filtered topic: %s", item.name)
	case topicItem:
		assert.Contains(t, item.name, "user", "Filtered item should contain 'user'")
		t.Logf("Selected filtered topic: %s", item.name)
	default:
		t.Errorf("Selected filtered item should be a HighlightedTopicItem or topicItem, got %T", selectedItem)
	}
}
