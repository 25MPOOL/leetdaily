package discordid_test

import (
	"testing"

	"github.com/nkoji21/leetdaily/internal/discordid"
)

func TestIsValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  bool
	}{
		{"123456789012345678", true},
		{"1", true},
		{"", false},
		{"   ", false},
		{"abc", false},
		{"123abc", false},
		{"12.34", false},
		{" 123456789012345678 ", false},
	}

	for _, tc := range cases {
		got := discordid.IsValid(tc.input)
		if got != tc.want {
			t.Errorf("IsValid(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
