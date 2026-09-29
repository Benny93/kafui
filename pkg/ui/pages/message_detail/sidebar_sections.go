package messagedetail

import (
	"fmt"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
)

// MessageInfoSection implements SidebarSection for message information
type MessageInfoSection struct {
	model *Model
}

// NewMessageInfoSection creates a new message info sidebar section
func NewMessageInfoSection(model *Model) *MessageInfoSection {
	return &MessageInfoSection{
		model: model,
	}
}

// GetTitle returns the section title
func (m *MessageInfoSection) GetTitle() string {
	return "Message Info"
}

// RenderItems returns the items to display in this section
func (m *MessageInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	if m.model == nil {
		return []providers.SidebarItem{}
	}

	items := []providers.SidebarItem{
		{
			Icon:   "●",
			Text:   "Topic",
			Value:  m.model.topicName,
			Status: "info",
		},
		{
			Icon:   "●",
			Text:   "Partition",
			Value:  fmt.Sprintf("%d", m.model.message.Partition),
			Status: "info",
		},
		{
			Icon:   "●",
			Text:   "Offset",
			Value:  fmt.Sprintf("%d", m.model.message.Offset),
			Status: "info",
		},
		{
			Icon:   "●",
			Text:   "Key Size",
			Value:  fmt.Sprintf("%d bytes", len(m.model.message.Key)),
			Status: "muted",
		},
		{
			Icon:   "●",
			Text:   "Value Size",
			Value:  fmt.Sprintf("%d bytes", len(m.model.message.Value)),
			Status: "muted",
		},
	}

	if len(m.model.message.Headers) > 0 {
		items = append(items, providers.SidebarItem{
			Icon:   "●",
			Text:   "Headers",
			Value:  fmt.Sprintf("%d", len(m.model.message.Headers)),
			Status: "info",
		})
	}

	// Add current time as viewing time since message timestamp may not be available
	items = append(items, providers.SidebarItem{
		Icon:   "●",
		Text:   "Viewed At",
		Value:  time.Now().Format("15:04:05"),
		Status: "muted",
	})

	// Limit to maxItems
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	return items
}

// HandleSectionUpdate handles section updates
func (m *MessageInfoSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

// InitSection initializes the section
func (m *MessageInfoSection) InitSection() tea.Cmd {
	return nil
}

// RefreshSection refreshes the section data
func (m *MessageInfoSection) RefreshSection() tea.Cmd {
	return nil
}

// SchemaInfoSection implements SidebarSection for schema information
type SchemaInfoSection struct {
	model *Model
}

// NewSchemaInfoSection creates a new schema info sidebar section
func NewSchemaInfoSection(model *Model) *SchemaInfoSection {
	return &SchemaInfoSection{
		model: model,
	}
}

// GetTitle returns the section title
func (s *SchemaInfoSection) GetTitle() string {
	return "Schema Info"
}

// RenderItems returns the items to display in this section
func (s *SchemaInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	if s.model == nil {
		return []providers.SidebarItem{}
	}

	// One row per side: the schema id, plus the record name once loaded.
	var keyInfo, valueInfo *api.SchemaInfo
	if info := s.model.GetSchemaInfo(); info != nil {
		keyInfo, valueInfo = info.KeySchema, info.ValueSchema
	}
	items := []providers.SidebarItem{}
	for _, side := range []struct {
		label, id string
		info      *api.SchemaInfo
	}{
		{"Key Schema", s.model.message.KeySchemaID, keyInfo},
		{"Value Schema", s.model.message.ValueSchemaID, valueInfo},
	} {
		if side.id == "" && side.info == nil {
			continue
		}
		value := side.id
		if name := schemaDisplayName(side.info, 20); name != "" {
			value = strings.TrimPrefix(value+" · "+name, " · ")
		}
		items = append(items, providers.SidebarItem{Icon: "●", Text: side.label, Value: value, Status: "info"})
	}

	if len(items) == 0 {
		items = append(items, providers.SidebarItem{Icon: "○", Text: "Schema", Value: "None", Status: "muted"})
	}

	// Limit to maxItems
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	return items
}

// HandleSectionUpdate handles section updates
func (s *SchemaInfoSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

// InitSection initializes the section
func (s *SchemaInfoSection) InitSection() tea.Cmd {
	return nil
}

// RefreshSection refreshes the section data
func (s *SchemaInfoSection) RefreshSection() tea.Cmd {
	return nil
}
