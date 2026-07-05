package errorpage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorPageVariants(t *testing.T) {
	m := New(nil, NotFound, "", "")
	m.SetDimensions(80, 24)
	assert.Equal(t, "error", m.GetID())
	assert.Contains(t, m.View(), "Not found")

	assert.Contains(t, New(nil, AccessDenied, "", "").View(), "Access denied")
	assert.Contains(t, New(nil, Generic, "Boom", "details here").View(), "Boom")
	assert.Contains(t, New(nil, Generic, "Boom", "details here").View(), "details here")
	assert.Contains(t, New(nil, NotFound, "", "").View(), "esc to go back")
}
