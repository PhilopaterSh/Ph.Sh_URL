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
func FetchAlienVaultURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- []string, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	var allUrls []string
	page := 1
	for {
		url := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/url_list?limit=500&page=%d", domain, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			if !silent {
				logChan <- logging.Message{Type: "ERROR", Source: "OTX", Message: fmt.Sprintf("Failed to create request: %v", err)}
			}
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
			break
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var r alienVaultResponse
		json.Unmarshal(body, &r)
		for _, item := range r.URLList {
			allUrls = append(allUrls, item.URL)
		}
		if !r.HasNext {
			break
		}
		page++
	}
	if !silent {
		logChan <- logging.Message{Type: "SUCCESS", Source: "OTX", Message: fmt.Sprintf("Found %d URLs", len(allUrls))}
	}
	ch <- allUrls
}
