// Package logging holds the shared log-message shape and ANSI color codes
// used by main.go and every package under internal/sources.
package logging

// Message is a structured log event emitted by a data source, fanned into
// main.go's colored log.Printf output.
type Message struct {
	Type    string
	Source  string
	Message string
}

const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorReset  = "\033[0m"
)
