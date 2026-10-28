package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
)

type alienVaultResponse struct {
	URLList []struct {
		URL string `json:"url"`
	} `json:"url_list"`
	HasNext bool `json:"has_next"`
}

// FetchAlienVaultURLs queries AlienVault OTX's URL list for a domain,
// paginating until has_next is false. Works without an API key, though one
// is preferred for better results.
func FetchAlienVaultURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- Result, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	var allUrls []string
	var fetchErr error
	page := 1
	for {
		url := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/url_list?limit=500&page=%d", domain, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			if !silent {
				logChan <- logging.Message{Type: "ERROR", Source: "OTX", Message: fmt.Sprintf("Failed to create request: %v", err)}
			}
			fetchErr = err
			break
		}

		if len(apiKeys) > 0 {
			apiKey := apiKeys[apiKeyIndex%len(apiKeys)]
			req.Header.Add("X-OTX-API-KEY", apiKey)
		}

		resp, err := MakeRequestWithRetry(req, silent, logChan, "OTX")
		if err != nil {
			if !silent {
				logChan <- logging.Message{Type: "ERROR", Source: "OTX", Message: err.Error()}
			}
			fetchErr = err
			break
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if !silent {
				logChan <- logging.Message{Type: "ERROR", Source: "OTX", Message: fmt.Sprintf("Failed to read response: %v", err)}
			}
			fetchErr = err
			break
		}
		var r alienVaultResponse
		if err := json.Unmarshal(body, &r); err != nil {
			if !silent {
				logChan <- logging.Message{Type: "ERROR", Source: "OTX", Message: fmt.Sprintf("Failed to decode response: %v", err)}
			}
			fetchErr = err
			break
		}
		for _, item := range r.URLList {
			allUrls = append(allUrls, item.URL)
		}
		if !r.HasNext {
			break
		}
		page++
	}
	ch <- Result{URLs: allUrls, Err: fetchErr}
}
