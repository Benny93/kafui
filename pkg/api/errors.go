package api

import "fmt"

// ConnectionError represents a connection-related error
type ConnectionError struct {
	Message string
	Cause   error
}

func (e ConnectionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("connection error: %s (caused by: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("connection error: %s", e.Message)
}

func (e ConnectionError) Unwrap() error {
	return e.Cause
}

// NewConnectionError creates a new connection error
func NewConnectionError(message string) ConnectionError {
	return ConnectionError{Message: message}
}

// NewConnectionErrorWithCause creates a new connection error with a cause
func NewConnectionErrorWithCause(message string, cause error) ConnectionError {
	return ConnectionError{Message: message, Cause: cause}
}

// TimeoutError represents a timeout-related error
type TimeoutError struct {
	Message string
	Timeout string
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("timeout error: %s (timeout: %s)", e.Message, e.Timeout)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message, timeout string) TimeoutError {
	return TimeoutError{Message: message, Timeout: timeout}
}

// AuthenticationError represents an authentication-related error
type AuthenticationError struct {
	Message string
	Method  string
}

func (e AuthenticationError) Error() string {
	return fmt.Sprintf("authentication error: %s (method: %s)", e.Message, e.Method)
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message, method string) AuthenticationError {
	return AuthenticationError{Message: message, Method: method}
}

// AuthorizationError represents an authorization-related error
type AuthorizationError struct {
	Message  string
	Resource string
	Action   string
}

func (e AuthorizationError) Error() string {
	return fmt.Sprintf("authorization error: %s (resource: %s, action: %s)", e.Message, e.Resource, e.Action)
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message, resource, action string) AuthorizationError {
	return AuthorizationError{Message: message, Resource: resource, Action: action}
}

// TopicError represents a topic-related error
type TopicError struct {
	Message   string
	TopicName string
}

func (e TopicError) Error() string {
	return fmt.Sprintf("topic error: %s (topic: %s)", e.Message, e.TopicName)
}

// NewTopicError creates a new topic error
func NewTopicError(message, topicName string) TopicError {
	return TopicError{Message: message, TopicName: topicName}
}

// PartitionError represents a partition-related error
type PartitionError struct {
	Message     string
	TopicName   string
	PartitionID int32
}

func (e PartitionError) Error() string {
	return fmt.Sprintf("partition error: %s (topic: %s, partition: %d)", e.Message, e.TopicName, e.PartitionID)
}

// NewPartitionError creates a new partition error
func NewPartitionError(message, topicName string, partitionID int32) PartitionError {
	return PartitionError{Message: message, TopicName: topicName, PartitionID: partitionID}
}