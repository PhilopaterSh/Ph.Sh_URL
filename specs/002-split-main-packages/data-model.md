# Phase 1 Data Model: Split main.go Into Packages

This feature has no persistent/database entities — it relocates existing in-memory types and
file formats unchanged. This document maps each entity to its new package (per FR-001–FR-009).

## `internal/config`

### `Config` (struct, relocated as-is)

| Field            | Type       | Notes                                    |
|------------------|------------|-------------------------------------------|
| `VirusTotalKeys` | `[]string` | yaml tag `virustotal`, placeholder-filtered |
| `AlienVaultKeys` | `[]string` | yaml tag `alienvault`, placeholder-filtered |
| `HudsonRockKeys` | `[]string` | yaml tag `hudsonrock`, placeholder-filtered |

**Validation rules**: `filterPlaceholderKeys` strips empty strings and values with prefix
`YOUR_` (unedited template placeholders) — behavior unchanged from v1.2.0.

**Functions**: `LoadConfig(silent bool) (*Config, error)`, `CreateDefaultConfig(path string)
error` — exported (capitalized) versions of the current unexported `loadConfig`/
`createDefaultConfig`, since `main.go` now calls them across a package boundary.

## `internal/domain`

Pure functions, no struct state:

- `IsValidDomain(domain string) bool`
- `CleanDomainLine(line string) string`

No dependency on `net/http`, `internal/config`, or `internal/sources` (enforced by Go's import
graph — this is what makes SC-004 verifiable via `go list -deps`).

## `internal/logging`

### `Message` (struct, renamed from `logMessage` for the exported API)

| Field     | Type     | Notes                                 |
|-----------|----------|-----------------------------------------|
| `Type`    | `string` | `"INFO"`, `"WARNING"`, `"ERROR"`, `"SUCCESS"`, `"FATAL"` |
| `Source`  | `string` | `"VT"`, `"OTX"`, `"Wayback"`, `"HudsonRock"` |
| `Message` | `string` | Human-readable log text                 |

Color constants (`ColorRed`, `ColorGreen`, `ColorYellow`, `ColorBlue`, `ColorReset`) move here,
exported so `main.go` can still colorize its own `[INFO]`/`[FATAL]` lines.

## `internal/sources`

### Per-source response types (relocated as-is, exported only if needed outside the file)

- `VirusTotalResponse` (`UndetectedUrls [][]string`, `ResponseCode int`)
- `AlienVaultResponse` (`URLList []struct{URL string}`, `HasNext bool`)
- `HudsonRockResponse` (`Data struct{AllURLs []struct{URL string}}`)

These stay unexported (`virusTotalResponse`, etc.) — they're JSON-unmarshal targets internal to
each source file, never touched by `main.go` or other sources.

### Fetch functions (relocated, signature unchanged apart from `internal/logging.Message` type)

- `FetchVirusTotalURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- []string, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool)`
- `FetchAlienVaultURLs(...)` (same shape)
- `FetchWaybackURLs(domain string, ch chan<- []string, logChan chan<- logging.Message, wg *sync.WaitGroup, silent bool)`
- `FetchHudsonRockURLs(...)` (same shape as VirusTotal/AlienVault)

### Shared helper

- `MakeRequestWithRetry(req *http.Request, silent bool, logChan chan<- logging.Message, source
  string) (*http.Response, error)` — unchanged logic (3 retries, 5s delay, no retry on 4xx),
  relocated from `main.go`.

## `main` (orchestration — stays in `main.go`)

- CLI flags (`-o`, `-d`, `-silent`, `-e`, `-version`) — unchanged.
- `finalUrlSet map[string]struct{}`, `failedDomains []string`, `resultsMu sync.Mutex` — the
  v1.2.0 race-fix state, unchanged and still living together in one file.
- The per-domain scan loop, Ctrl+C signal handler, and final output-writing logic — unchanged,
  now calling into `internal/config`, `internal/domain`, `internal/sources`.

**State transitions**: None of this feature changes runtime state transitions — a domain still
goes through clean → validate → fetch (4 sources in parallel) → aggregate → (success | failed)
exactly as before; only the source file each step lives in changes.
