package helpers

import (
	"time"
)

// StringToPtr returns the pointer to a string
func StringToPtr(str string) *string {
	return &str
}

// boolToPtr returns the pointer to a boolean
func BoolToPtr(b bool) *bool {
	return &b
}

// IntToPtr returns the pointer to an int
func IntToPtr(i int) *int {
	return &i
}

// DurationToPtr return the pointer to a duration
func DurationToPtr(d time.Duration) *time.Duration {
	return &d
}
