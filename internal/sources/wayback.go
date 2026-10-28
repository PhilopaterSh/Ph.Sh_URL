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
func FetchWaybackURLs(domain string, ch chan<- Result, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	url := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=*.%s/*&output=text&fl=original&collapse=urlkey", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "Wayback", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
		ch <- Result{URLs: nil, Err: err}
		return
	}

	resp, err := MakeRequestWithRetry(req, silent, logChan, "Wayback")
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "Wayback", Message: err.Error()}
		}
		ch <- Result{URLs: nil, Err: err}
		return
	}
	defer resp.Body.Close()

	var fetchErr error
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<20)) // cap huge CDX responses at 512MB
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "Wayback", Message: fmt.Sprintf("Failed to read response: %v", err)}
		}
		fetchErr = err
	}
	var urls []string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.TrimSpace(line) != "" {
			urls = append(urls, line)
		}
	}
	if !silent {
		logChan <- logging.Message{Type: "SUCCESS", Source: "Wayback", Message: fmt.Sprintf("Found %d URLs", len(urls))}
	}
	ch <- Result{URLs: urls, Err: fetchErr}
}
