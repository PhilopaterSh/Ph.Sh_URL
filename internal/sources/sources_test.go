package sources

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
)

// stubTransport serves canned responses without touching the network.
type stubTransport struct {
	handler func(req *http.Request, call int) (*http.Response, error)
	calls   int
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls++
	return s.handler(req, s.calls)
}

func stubResponse(status int, body string) (*http.Response, error) {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

// useStub swaps the package-level HTTP client for one wired to the stub and
// returns a restore function.
func useStub(t *testing.T, s *stubTransport) func() {
	t.Helper()
	original := httpClient
	originalDelay := retryDelay
	httpClient = &http.Client{Transport: s}
	retryDelay = time.Millisecond
	return func() {
		httpClient = original
		retryDelay = originalDelay
	}
}

func drainLog(logChan chan logging.Message) {
	for {
		select {
		case <-logChan:
		default:
			return
		}
	}
}

// --- MakeRequestWithRetry: the v1.2.1 P0 regression tests ---

func TestMakeRequestWithRetryRetriesOn5xxThenSucceeds(t *testing.T) {
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		if call < 3 {
			return stubResponse(http.StatusInternalServerError, "boom")
		}
		return stubResponse(http.StatusOK, `{"ok":true}`)
	}}
	restore := useStub(t, s)
	defer restore()

	logChan := make(chan logging.Message, 16)
	req, _ := http.NewRequest("GET", "http://stub/x", nil)
	resp, err := MakeRequestWithRetry(req, true, logChan, "T")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	resp.Body.Close()
	if s.calls != 3 {
		t.Errorf("expected 3 attempts, got %d", s.calls)
	}
}

func TestMakeRequestWithRetryFailsFastOn403(t *testing.T) {
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		return stubResponse(http.StatusForbidden, "forbidden")
	}}
	restore := useStub(t, s)
	defer restore()

	logChan := make(chan logging.Message, 16)
	req, _ := http.NewRequest("GET", "http://stub/x", nil)
	if _, err := MakeRequestWithRetry(req, true, logChan, "T"); err == nil {
		t.Fatal("expected error on 403, got nil")
	}
	if s.calls != 1 {
		t.Errorf("403 must not be retried, got %d attempts", s.calls)
	}
}

func TestMakeRequestWithRetryRetriesOn429And4xxBodyIsNotData(t *testing.T) {
	// Pre-v1.2.1 bug: a 429/5xx body was decoded as data and logged as
	// "SUCCESS Found 0 URLs". This test locks the fix in.
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		return stubResponse(http.StatusTooManyRequests, `{"error":"rate limited"}`)
	}}
	restore := useStub(t, s)
	defer restore()

	logChan := make(chan logging.Message, 16)
	req, _ := http.NewRequest("GET", "http://stub/x", nil)
	if _, err := MakeRequestWithRetry(req, true, logChan, "T"); err == nil {
		t.Fatal("expected error on persistent 429, got nil")
	}
	if s.calls != retryAttempts {
		t.Errorf("expected %d attempts on 429, got %d", retryAttempts, s.calls)
	}
}

func TestMakeRequestWithRetryRetriesOnNetworkError(t *testing.T) {
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		return nil, errors.New("connection reset")
	}}
	restore := useStub(t, s)
	defer restore()

	logChan := make(chan logging.Message, 16)
	req, _ := http.NewRequest("GET", "http://stub/x", nil)
	if _, err := MakeRequestWithRetry(req, true, logChan, "T"); err == nil {
		t.Fatal("expected error, got nil")
	}
	if s.calls != retryAttempts {
		t.Errorf("expected %d attempts, got %d", retryAttempts, s.calls)
	}
}

// --- Fetchers: Result contract (v1.2.2 regression tests) ---

func collectResult(t *testing.T, ch chan Result) Result {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("fetcher never sent a result")
		return Result{}
	}
}

func TestFetchWaybackURLsParsesLines(t *testing.T) {
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		return stubResponse(http.StatusOK, "http://a.example.com/\nhttp://b.example.com/\n\n")
	}}
	restore := useStub(t, s)
	defer restore()

	var wg sync.WaitGroup
	logChan := make(chan logging.Message, 16)
	ch := make(chan Result, 1)
	wg.Add(1)
	go FetchWaybackURLs("example.com", ch, logChan, &wg, true)
	wg.Wait()

	res := collectResult(t, ch)
	if res.Err != nil {
		t.Fatalf("expected success, got error: %v", res.Err)
	}
	if len(res.URLs) != 2 {
		t.Errorf("expected 2 URLs, got %d: %v", len(res.URLs), res.URLs)
	}
}

func TestFetchAlienVaultURLsPaginates(t *testing.T) {
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		if call == 1 {
			return stubResponse(http.StatusOK, `{"url_list":[{"url":"http://p1.example.com/"}],"has_next":true}`)
		}
		return stubResponse(http.StatusOK, `{"url_list":[{"url":"http://p2.example.com/"}],"has_next":false}`)
	}}
	restore := useStub(t, s)
	defer restore()

	var wg sync.WaitGroup
	logChan := make(chan logging.Message, 16)
	ch := make(chan Result, 1)
	wg.Add(1)
	go FetchAlienVaultURLs("example.com", nil, 0, ch, logChan, &wg, true)
	wg.Wait()

	res := collectResult(t, ch)
	if res.Err != nil {
		t.Fatalf("expected success, got error: %v", res.Err)
	}
	if len(res.URLs) != 2 {
		t.Errorf("expected URLs from both pages, got %v", res.URLs)
	}
}

func TestFetchAlienVaultURLsReportsErrorOn5xx(t *testing.T) {
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		return stubResponse(http.StatusForbidden, "nope")
	}}
	restore := useStub(t, s)
	defer restore()

	var wg sync.WaitGroup
	logChan := make(chan logging.Message, 16)
	ch := make(chan Result, 1)
	wg.Add(1)
	go FetchAlienVaultURLs("example.com", nil, 0, ch, logChan, &wg, true)
	wg.Wait()

	res := collectResult(t, ch)
	if res.Err == nil {
		t.Fatal("expected error on persistent 403, got nil")
	}
	// Partial data may exist, but the failure must be reported so the domain
	// is not marked as scanned.
}

func TestFetchVirusTotalURLsWithoutKeyIsExplicitFailure(t *testing.T) {
	var wg sync.WaitGroup
	logChan := make(chan logging.Message, 16)
	ch := make(chan Result, 1)
	wg.Add(1)
	go FetchVirusTotalURLs("example.com", nil, 0, ch, logChan, &wg, true)
	wg.Wait()

	res := collectResult(t, ch)
	if res.Err == nil {
		t.Fatal("expected explicit 'no API key' error, got nil")
	}
	if !strings.Contains(res.Err.Error(), "no API key") {
		t.Errorf("unexpected error: %v", res.Err)
	}
}

func TestFetchHudsonRockURLsMalformedJSONIsError(t *testing.T) {
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		return stubResponse(http.StatusOK, `<html>gateway error page</html>`)
	}}
	restore := useStub(t, s)
	defer restore()

	var wg sync.WaitGroup
	logChan := make(chan logging.Message, 16)
	ch := make(chan Result, 1)
	wg.Add(1)
	go FetchHudsonRockURLs("example.com", nil, 0, ch, logChan, &wg, true)
	wg.Wait()

	res := collectResult(t, ch)
	if res.Err == nil {
		t.Fatal("expected decode error for malformed JSON, got nil")
	}
}

func TestFetchZeroURLsIsSuccessNotFailure(t *testing.T) {
	// The v1.2.2 core fix: a completed scan with zero results is a success.
	s := &stubTransport{handler: func(req *http.Request, call int) (*http.Response, error) {
		return stubResponse(http.StatusOK, `{"url_list":[],"has_next":false}`)
	}}
	restore := useStub(t, s)
	defer restore()

	var wg sync.WaitGroup
	logChan := make(chan logging.Message, 16)
	ch := make(chan Result, 1)
	wg.Add(1)
	go FetchAlienVaultURLs("empty.example.com", nil, 0, ch, logChan, &wg, true)
	wg.Wait()

	res := collectResult(t, ch)
	if res.Err != nil {
		t.Fatalf("zero-result scan must be a success, got error: %v", res.Err)
	}
	if res.URLs != nil {
		t.Errorf("expected no URLs, got %v", res.URLs)
	}
}
