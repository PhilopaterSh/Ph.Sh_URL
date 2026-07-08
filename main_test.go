package main

import "testing"

func TestIsValidDomain(t *testing.T) {
	cases := []struct {
		domain string
		want   bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"a.co", true},
		{"example", false},
		{"-example.com", false},
		{"example-.com", false},
		{"example..com", false},
		{"", false},
		{"exa mple.com", false},
		{"example.c", false},
	}

	for _, c := range cases {
		if got := isValidDomain(c.domain); got != c.want {
			t.Errorf("isValidDomain(%q) = %v, want %v", c.domain, got, c.want)
		}
	}
}

func TestCleanDomainLine(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"  example.com  ", "example.com"},
		{"example.com!", "example.com"},
		{"exa*mple.com", "example.com"},
		{"", ""},
	}

	for _, c := range cases {
		if got := cleanDomainLine(c.input); got != c.want {
			t.Errorf("cleanDomainLine(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFilterPlaceholderKeys(t *testing.T) {
	input := []string{"YOUR_VT_API_KEY_1", "", "  ", "real-key-123", "YOUR_OTX_API_KEY_1"}
	got := filterPlaceholderKeys(input)

	if len(got) != 1 || got[0] != "real-key-123" {
		t.Errorf("filterPlaceholderKeys(%v) = %v, want [real-key-123]", input, got)
	}
}
