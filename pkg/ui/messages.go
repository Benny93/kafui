package ui

import (
	"time"
)

// Custom message types
type topicListMsg []interface{} // Changed from []list.Item to []interface{}
type errorMsg error
type pageChangeMsg page
type timerTickMsg time.Time