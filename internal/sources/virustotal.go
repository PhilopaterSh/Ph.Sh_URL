package sources

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
)

type virusTotalResponse struct {
	UndetectedUrls [][]string `json:"undetected_urls"`
	ResponseCode   int        `json:"response_code"`
}

// FetchVirusTotalURLs queries VirusTotal's domain report for undetected URLs.
// It requires an API key; without one it reports a failure so the domain is
// not treated as scanned.
func FetchVirusTotalURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- Result, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	if len(apiKeys) == 0 {
		if !silent {
			logChan <- logging.Message{Type: "INFO", Source: "VT", Message: "No API key configured — skipping source"}
		}
		ch <- Result{URLs: nil, Err: errors.New("no API key configured")}
		return
	}
	apiKey := apiKeys[apiKeyIndex%len(apiKeys)]
	url := fmt.Sprintf("https://www.virustotal.com/vtapi/v2/domain/report?apikey=%s&domain=%s", apiKey, url.QueryEscape(domain))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "VT", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
		ch <- Result{URLs: nil, Err: err}
		return
	}

	resp, err := MakeRequestWithRetry(req, silent, logChan, "VT")
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "VT", Message: err.Error()}
		}
		ch <- Result{URLs: nil, Err: err}
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var fetchErr error
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "VT", Message: fmt.Sprintf("Failed to read response: %v", err)}
		}
		fetchErr = err
	}
	var r virusTotalResponse
	if decodeErr := json.Unmarshal(body, &r); decodeErr != nil && fetchErr == nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "VT", Message: fmt.Sprintf("Failed to decode response: %v", decodeErr)}
		}
		fetchErr = decodeErr
	}

	var urls []string
	if r.ResponseCode == 1 {
		for _, item := range r.UndetectedUrls {
			if len(item) > 0 {
				urls = append(urls, item[0])
			}
		}
	}
	if !silent {
		logChan <- logging.Message{Type: "SUCCESS", Source: "VT", Message: fmt.Sprintf("Found %d URLs", len(urls))}
	}
	ch <- Result{URLs: urls, Err: fetchErr}
}
