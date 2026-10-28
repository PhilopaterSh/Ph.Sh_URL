package sources

// Result is what every source sends back on its result channel: the URLs it
// found and, separately, whether the source actually completed its scan.
// A nil Err with zero URLs means "scanned successfully, nothing found" —
// which is not a failure. A non-nil Err means the scan itself failed (after
// retries); any partial URLs collected before the failure are still included.
type Result struct {
	URLs []string
	Err  error
}
