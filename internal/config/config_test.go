package config

import "testing"

func TestFilterPlaceholderKeys(t *testing.T) {
	input := []string{"YOUR_VT_API_KEY_1", "", "  ", "real-key-123", "YOUR_OTX_API_KEY_1"}
	got := filterPlaceholderKeys(input)

	if len(got) != 1 || got[0] != "real-key-123" {
		t.Errorf("filterPlaceholderKeys(%v) = %v, want [real-key-123]", input, got)
	}
}
