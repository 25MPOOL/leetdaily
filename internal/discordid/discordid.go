package discordid

import (
	"strings"
	"unicode"
)

// IsValid reports whether value is a valid Discord snowflake ID
// (a non-empty string of ASCII digits).
func IsValid(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}
