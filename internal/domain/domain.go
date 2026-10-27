// Package domain provides pure, network-free domain-name validation and
// cleaning used before a domain is scanned.
package domain

import (
	"regexp"
	"strings"
)

// isValidDomainPattern matches registrable domains, allowing punycode TLDs
// (e.g. example.xn--p1ai).
var isValidDomainPattern = regexp.MustCompile(`(?i)^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+([a-z]{2,}|xn--[a-z0-9-]+)$`)

// IsValidDomain checks if a string is a syntactically valid domain name.
func IsValidDomain(domain string) bool {
	return isValidDomainPattern.MatchString(domain)
}

// CleanDomainLine extracts the domain candidate from a raw input line. It
// keeps only the leading run of domain-legal characters, so lines like
// "example.com,other.com" or "example.com/extra" never fuse into a bogus
// domain such as "example.comextra" (which used to pass validation).
func CleanDomainLine(line string) string {
	line = strings.TrimSpace(line)
	for i, r := range line {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			continue
		}
		line = line[:i]
		break
	}
	return strings.Trim(line, ".-")
}
