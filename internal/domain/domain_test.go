package domain

import "testing"

func TestIsValidDomain(t *testing.T) {
	cases := []struct {
		domain string
		want   bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"a.co", true},
		{"example.xn--p1ai", true},
		{"example", false},
		{"-example.com", false},
		{"example-.com", false},
		{"example..com", false},
		{"", false},
		{"exa mple.com", false},
		{"example.c", false},
	}

	for _, c := range cases {
		if got := IsValidDomain(c.domain); got != c.want {
			t.Errorf("IsValidDomain(%q) = %v, want %v", c.domain, got, c.want)
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
		{"example.com.", "example.com"},
		{"example.com/extra", "example.com"},
		{"a.com,b.com", "a.com"},
		{"exa*mple.com", "exa"}, // cut at '*' — must NOT fuse into a bogus "example.com"
		{"", ""},
	}

	for _, c := range cases {
		if got := CleanDomainLine(c.input); got != c.want {
			t.Errorf("CleanDomainLine(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
