package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
)

type hudsonRockResponse struct {
	Data struct {
		AllURLs []struct {
			URL string `json:"url"`
		} `json:"all_urls"`
	} `json:"data"`
}

// FetchHudsonRockURLs queries Hudson Rock's OSINT search-by-domain endpoint.
// Works without an API key, but results may be redacted/encrypted.
func FetchHudsonRockURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- Result, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()

	url := fmt.Sprintf("https://cavalier.hudsonrock.com/api/json/v2/osint-tools/search-by-domain?domain=%s", url.QueryEscape(domain))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "HudsonRock", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
		ch <- Result{URLs: nil, Err: err}
		return
	}

	if len(apiKeys) > 0 {
		apiKey := apiKeys[apiKeyIndex%len(apiKeys)]
		req.Header.Add("key", apiKey) // Assuming the header is 'key', might need to be 'X-API-Key' or similar
	} else {
		if !silent {
			logChan <- logging.Message{Type: "WARNING", Source: "HudsonRock", Message: "Processing without API key (data may be redacted)..."}
		}
	}

	resp, err := MakeRequestWithRetry(req, silent, logChan, "HudsonRock")
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "HudsonRock", Message: err.Error()}
		}
		ch <- Result{URLs: nil, Err: err}
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var fetchErr error
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "HudsonRock", Message: fmt.Sprintf("Failed to read response: %v", err)}
		}
		fetchErr = err
	}
	var r hudsonRockResponse
	if decodeErr := json.Unmarshal(body, &r); decodeErr != nil && fetchErr == nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "HudsonRock", Message: fmt.Sprintf("Failed to decode response: %v", decodeErr)}
		}
		fetchErr = decodeErr
	}

	var urls []string
	for _, item := range r.Data.AllURLs {
		urls = append(urls, item.URL)
	}

	if !silent {
		logChan <- logging.Message{Type: "SUCCESS", Source: "HudsonRock", Message: fmt.Sprintf("Found %d URLs", len(urls))}
	}
	ch <- Result{URLs: urls, Err: fetchErr}
}
