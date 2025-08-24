package topic

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSearchModeKeyHandling(t *testing.T) {
	// Create mock data source
	mockDS := &mock.KafkaDataSourceMock{}
	mockDS.Init("")

	// Create test topic
	testTopic := api.Topic{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ReplicaAssignment: make(map[int32][]int32),
		ConfigEntries:     make(map[string]*string),
	}

	// Create new model
	model := NewModel(mockDS, "test-topic", testTopic)

	// Enable search mode
	model.searchMode = true
	model.searchInput.Focus()

	// Test that 'q' key is handled by search input, not as quit
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, _ := model.Update(qMsg)
	
	// Should stay in the same model
	assert.IsType(t, &Model{}, updatedModel)
	
	// Search input should contain 'q'
	updatedTopicModel := updatedModel.(*Model)
	assert.Equal(t, "q", updatedTopicModel.searchInput.Value())

	// Test that 'r' key is handled by search input, not as retry
	rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel2, _ := updatedModel.Update(rMsg)
	
	// Should stay in the same model
	assert.IsType(t, &Model{}, updatedModel2)
	
	// Search input should contain 'qr'
	updatedTopicModel2 := updatedModel2.(*Model)
	assert.Equal(t, "qr", updatedTopicModel2.searchInput.Value())

	// Test that Enter key confirms search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel3, cmd3 := updatedModel2.Update(enterMsg)
	
	// Should stay in the same model
	assert.IsType(t, &Model{}, updatedModel3)
	
	// Should not navigate to detail page
	assert.Nil(t, cmd3)
	
	// Should exit search mode
	updatedTopicModel3 := updatedModel3.(*Model)
	assert.False(t, updatedTopicModel3.searchMode)
	assert.Equal(t, "qr", updatedTopicModel3.searchInput.Value()) // Value should remain

	// Test that Esc key cancels search
	// Re-enable search mode first
	updatedTopicModel3.searchMode = true
	updatedTopicModel3.searchInput.Focus()
	updatedTopicModel3.searchInput.SetValue("test")

	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updatedModel4, cmd4 := updatedModel3.Update(escMsg)
	
	// Should stay in the same model
	assert.IsType(t, &Model{}, updatedModel4)
	
	// Should not navigate back
	assert.Nil(t, cmd4)
	
	// Should exit search mode and clear input
	updatedTopicModel4 := updatedModel4.(*Model)
	assert.False(t, updatedTopicModel4.searchMode)
	assert.Equal(t, "", updatedTopicModel4.searchInput.Value()) // Value should be cleared
}