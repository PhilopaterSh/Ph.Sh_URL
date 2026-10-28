package main

import "testing"

func TestNormalizeDomains(t *testing.T) {
	lines := []string{
		"example.com",
		"  Example.com  ",        // duplicate after cleaning (case-insensitive input)
		"",                       // empty line
		"example.com/extra",      // cleaned to example.com → duplicate
		"sub.example.com,other",  // first token wins
		"not a domain!",          // cleaned to "not" → invalid → skipped
		"valid.example.xn--p1ai", // punycode TLD accepted
	}
	got := normalizeDomains(lines, true)

	want := []string{"example.com", "sub.example.com", "valid.example.xn--p1ai"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("target[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeDomainsAllInvalid(t *testing.T) {
	got := normalizeDomains([]string{"", "   ", "nope"}, true)
	if len(got) != 0 {
		t.Errorf("expected no targets, got %v", got)
	}
}
