// Package domain provides pure, network-free domain-name validation and
// cleaning used before a domain is scanned.
package domain

import (
	"regexp"
	"strings"
)

// IsValidDomain checks if a string is a syntactically valid domain name.
func IsValidDomain(domain string) bool {
	match, _ := regexp.MatchString(`(?i)^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`, domain)
	return match
}

// CleanDomainLine removes invalid characters from a domain string.
func CleanDomainLine(line string) string {
	// This regex removes anything that is not a letter, number, dot, or hyphen.
	reg := regexp.MustCompile(`[^a-zA-Z0-9.-]`)
	cleaned := reg.ReplaceAllString(line, "")
	return strings.TrimSpace(cleaned)
}
