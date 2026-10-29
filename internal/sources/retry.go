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
	// requestTimeout bounds every request: http.DefaultClient has no timeout,
	// so one stalled connection would hang the whole tool forever.
	requestTimeout = 30 * time.Second
)

// retryDelay is a var so tests can shorten the backoff.
var retryDelay = 5 * time.Second

// httpClient is the shared HTTP client used by every source.
var httpClient = &http.Client{Timeout: requestTimeout}

// MakeRequestWithRetry performs req, retrying on network errors and on
// 429/5xx responses up to retryAttempts times. Permanent client errors
// (4xx other than 429) fail immediately. On success the returned response is
// 2xx/3xx; failing responses are closed before the error is returned.
func MakeRequestWithRetry(req *http.Request, silent bool, logChan chan<- logging.Message, source string) (*http.Response, error) {
	var lastErr error

	for i := 0; i < retryAttempts; i++ {
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server returned status: %s", resp.Status)
			_ = resp.Body.Close()
		} else if resp.StatusCode >= 400 {
			// Permanent client error (401/403/404/...): fail fast with a
			// clear message instead of letting the caller decode an error
			// body as data.
			_ = resp.Body.Close()
			return nil, fmt.Errorf("server returned status: %s", resp.Status)		} else {
			// 2xx/3xx — success
			return resp, nil
		}

		if i < retryAttempts-1 {
			if !silent {
				logChan <- logging.Message{Type: "WARNING", Source: source, Message: fmt.Sprintf("Request failed (attempt %d/%d): %v. Retrying in %v...", i+1, retryAttempts, lastErr, retryDelay)}
			}
			time.Sleep(retryDelay)
		}
	}
	return nil, fmt.Errorf("request failed after %d attempts: %w", retryAttempts, lastErr)
}
