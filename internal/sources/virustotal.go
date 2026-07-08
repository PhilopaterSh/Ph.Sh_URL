package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
)

type virusTotalResponse struct {
	UndetectedUrls [][]string `json:"undetected_urls"`
	ResponseCode   int        `json:"response_code"`
}

// FetchVirusTotalURLs queries VirusTotal's domain report for undetected URLs.
// It requires an API key; if none is configured it returns immediately.
func FetchVirusTotalURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- []string, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	if len(apiKeys) == 0 {
		return
	}
	apiKey := apiKeys[apiKeyIndex%len(apiKeys)]
	url := fmt.Sprintf("https://www.virustotal.com/vtapi/v2/domain/report?apikey=%s&domain=%s", apiKey, domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "VT", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
		ch <- nil
		return
	}

	resp, err := MakeRequestWithRetry(req, silent, logChan, "VT")
	if err != nil {
		if !silent {
			logChan <- logging.Message{Type: "ERROR", Source: "VT", Message: err.Error()}
		}
		ch <- nil
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var r virusTotalResponse
	json.Unmarshal(body, &r)
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
	ch <- urls
}
