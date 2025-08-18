package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
)

// Custom message types
type topicListMsg []list.Item
type errorMsg error
type pageChangeMsg page
type timerTickMsg time.Time