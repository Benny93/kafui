package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUIState_String(t *testing.T) {
	tests := []struct {
		state    UIState
		expected string
	}{
		{StateNormal, "normal"},
		{StateHelp, "help"},
		{StateSearch, "search"},
		{StateModal, "modal"},
		{99, "unknown"}, // Unknown state
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFocusState_String(t *testing.T) {
	tests := []struct {
		state    FocusState
		expected string
	}{
		{FocusNone, "none"},
		{FocusMain, "main"},
		{FocusSidebar, "sidebar"},
		{FocusSearch, "search"},
		{FocusFooter, "footer"},
		{99, "unknown"}, // Unknown state
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadingState_Values(t *testing.T) {
	// Verify all loading states have unique values
	states := []LoadingState{
		LoadingIdle,
		LoadingInitial,
		LoadingRefresh,
		LoadingMore,
	}

	seen := make(map[LoadingState]bool)
	for _, state := range states {
		assert.False(t, seen[state], "Duplicate loading state value: %d", state)
		seen[state] = true
	}
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{ConnectionUnknown, "unknown"},
		{ConnectionConnected, "connected"},
		{ConnectionDisconnected, "disconnected"},
		{ConnectionReconnecting, "reconnecting"},
		{99, "unknown"}, // Unknown state
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectionState_Values(t *testing.T) {
	// Verify all connection states have unique values
	states := []ConnectionState{
		ConnectionUnknown,
		ConnectionConnected,
		ConnectionDisconnected,
		ConnectionReconnecting,
	}

	seen := make(map[ConnectionState]bool)
	for _, state := range states {
		assert.False(t, seen[state], "Duplicate connection state value: %d", state)
		seen[state] = true
	}
}

func TestStateTransitions_Valid(t *testing.T) {
	// Test valid state transitions
	validTransitions := []struct {
		from UIState
		to   UIState
	}{
		{StateNormal, StateHelp},
		{StateHelp, StateNormal},
		{StateNormal, StateSearch},
		{StateSearch, StateNormal},
		{StateNormal, StateModal},
		{StateModal, StateNormal},
		{StateSearch, StateModal},
		{StateModal, StateSearch},
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			// All transitions are technically valid in the type system
			// The application logic determines which are allowed
			assert.NotEqual(t, tt.from, tt.to, "Transition should be between different states")
		})
	}
}

func TestFocusStateTransitions(t *testing.T) {
	// Test focus state transitions
	transitions := []struct {
		from FocusState
		to   FocusState
	}{
		{FocusNone, FocusMain},
		{FocusMain, FocusSidebar},
		{FocusSidebar, FocusSearch},
		{FocusSearch, FocusFooter},
		{FocusFooter, FocusNone},
	}

	for _, tt := range transitions {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			// All transitions are technically valid
			assert.NotEqual(t, tt.from, tt.to, "Transition should be between different states")
		})
	}
}

func TestUIState_Comparison(t *testing.T) {
	// Test that different states are not equal
	states := []UIState{
		StateNormal,
		StateHelp,
		StateSearch,
		StateModal,
	}

	for i := 0; i < len(states); i++ {
		for j := i + 1; j < len(states); j++ {
			assert.NotEqual(t, states[i], states[j],
				"States %v and %v should be different", states[i], states[j])
		}
	}
}

func TestFocusState_Comparison(t *testing.T) {
	// Test that different focus states are not equal
	states := []FocusState{
		FocusNone,
		FocusMain,
		FocusSidebar,
		FocusSearch,
		FocusFooter,
	}

	for i := 0; i < len(states); i++ {
		for j := i + 1; j < len(states); j++ {
			assert.NotEqual(t, states[i], states[j],
				"Focus states %v and %v should be different", states[i], states[j])
		}
	}
}

func TestLoadingState_Sequence(t *testing.T) {
	// Verify loading states follow a logical sequence
	assert.Equal(t, LoadingState(0), LoadingIdle, "LoadingIdle should be 0")
	assert.Equal(t, LoadingState(1), LoadingInitial, "LoadingInitial should be 1")
	assert.Equal(t, LoadingState(2), LoadingRefresh, "LoadingRefresh should be 2")
	assert.Equal(t, LoadingState(3), LoadingMore, "LoadingMore should be 3")
}

func TestConnectionState_Sequence(t *testing.T) {
	// Verify connection states follow a logical sequence
	assert.Equal(t, ConnectionState(0), ConnectionUnknown, "ConnectionUnknown should be 0")
	assert.Equal(t, ConnectionState(1), ConnectionConnected, "ConnectionConnected should be 1")
	assert.Equal(t, ConnectionState(2), ConnectionDisconnected, "ConnectionDisconnected should be 2")
	assert.Equal(t, ConnectionState(3), ConnectionReconnecting, "ConnectionReconnecting should be 3")
}

func TestStateConstants(t *testing.T) {
	// Test that state constants are properly defined
	assert.GreaterOrEqual(t, int(StateNormal), 0)
	assert.GreaterOrEqual(t, int(FocusNone), 0)
	assert.GreaterOrEqual(t, int(LoadingIdle), 0)
	assert.GreaterOrEqual(t, int(ConnectionUnknown), 0)
}
