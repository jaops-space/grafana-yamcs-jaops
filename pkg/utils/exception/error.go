package exception

import (
	"fmt"
	"time"
)

// PluginException represents a detailed error specific to the Yamcs system.
// It includes a message, an error code, a timestamp, and an optional underlying error.
type PluginException struct {
	// Message provides a description of the error.
	Message string
	// Code represents a specific error code.
	Code string
	// Timestamp represents when the error occurred.
	Timestamp time.Time
	// Cause holds an optional underlying error.
	Cause error
}

// It returns a detailed error string including the message, code, timestamp, and cause.
func (e *PluginException) Error() string {
	errorMessage := fmt.Sprintf("Yamcs-Grafana Plugin Error [Code: %s, Timestamp: %s]: %s", e.Code, e.Timestamp.Format(time.RFC3339), e.Message)
	if e.Cause != nil {
		errorMessage += fmt.Sprintf(" | Cause: %v", e.Cause)
	}
	return errorMessage
}

// New creates a new PluginException with the given message and error code.
// It returns a pointer to the newly created error.
func New(message string, code string) *PluginException {
	return &PluginException{
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
		Cause:     nil,
	}
}

// Wrap creates a new PluginException wrapping an existing error.
// It allows including an underlying cause while also providing a custom message and error code.
func Wrap(message string, code string, cause error) *PluginException {
	return &PluginException{
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
		Cause:     cause,
	}
}
