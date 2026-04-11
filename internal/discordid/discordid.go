package discordid

// IsValid reports whether value is a valid Discord snowflake ID
// (a non-empty string of ASCII decimal digits with no surrounding whitespace).
func IsValid(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
