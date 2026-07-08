// Package sources implements the per-source URL-discovery fetchers
// (VirusTotal, AlienVault OTX, Wayback Machine, Hudson Rock).
package sources

import (
	"fmt"
	"net/http"
	"time"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
)

const (
	retryAttempts = 3
	retryDelay    = 5 * time.Second
)

// MakeRequestWithRetry performs req, retrying on network/server errors up to
// retryAttempts times. Client errors (4xx) are returned immediately without
// retrying.
func MakeRequestWithRetry(req *http.Request, silent bool, logChan chan<- logging.Message, source string) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < retryAttempts; i++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			// Success, return response
			return resp, nil
		}

		if !silent {
			logChan <- logging.Message{Type: "WARNING", Source: source, Message: fmt.Sprintf("Request failed (attempt %d/%d): %v. Retrying in %v...", i+1, retryAttempts, err, retryDelay)}
		}

		// Don't retry on client-side errors, but do on server-side or network errors.
		if resp != nil && (resp.StatusCode >= 400 && resp.StatusCode < 500) {
			if !silent {
				logChan <- logging.Message{Type: "ERROR", Source: source, Message: fmt.Sprintf("Client error %d, not retrying.", resp.StatusCode)}
			}
			return resp, err // Return response as-is for client errors
		}

		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("request failed after %d attempts: %w", retryAttempts, err)
}
