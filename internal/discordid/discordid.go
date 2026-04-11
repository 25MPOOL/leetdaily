package discordid

import "unicode"

// IsValid reports whether value is a valid Discord snowflake ID
// (a non-empty string of ASCII digits with no surrounding whitespace).
func IsValid(value string) bool {
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
