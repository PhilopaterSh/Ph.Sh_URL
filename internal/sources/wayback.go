package sources

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
)

// FetchWaybackURLs queries the Wayback Machine's CDX API for archived URLs
// under a domain. Works without an API key.
func FetchWaybackURLs(domain string, ch chan<- []string, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	url := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=*.%s/*&output=text&fl=original&collapse=urlkey", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "Wayback", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
		ch <- nil
		return
	}

	resp, err := MakeRequestWithRetry(req, silent, logChan, "Wayback")
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "Wayback", Message: err.Error()}
		}
		ch <- nil
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	// Filter out empty lines
	var urls []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			urls = append(urls, line)
		}
	}
	if !silent {
		logChan <- logging.Message{Type: "SUCCESS", Source: "Wayback", Message: fmt.Sprintf("Found %d URLs", len(urls))}
	}
	ch <- urls
}
