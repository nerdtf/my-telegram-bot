package api

import (
	"fmt"
)

// Error wraps an error with additional context
type Error struct {
	Err     error
	Message string
	Details interface{}
}

// ValidationError represents an error that occurs during validation.
type ValidationError struct {
	Message string
	Errors  map[string][]string
}

// Error method returns the error message for the Error struct.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// ValidationError method returns the error message for the ValidationError struct.
func (ve *ValidationError) ValidationError() string {
	return ve.Message
}
