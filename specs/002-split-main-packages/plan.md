# Implementation Plan: Split main.go Into Packages

**Branch**: `002-split-main-packages` | **Date**: 2026-07-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-split-main-packages/spec.md`

## Summary

`main.go` (~700 lines) currently mixes CLI flag parsing, config loading, four independent
HTTP-based data-source fetchers, and scan orchestration in one file. This plan splits it into
four packages — `internal/config`, `internal/domain`, `internal/logging`, `internal/sources` —
while keeping `main.go` as pure orchestration (flags, scan loop, signal handling, output
writing), with zero change to CLI flags, file formats, or log message text.

## Technical Context

**Language/Version**: Go 1.25.1 (per `go.mod`)

**Primary Dependencies**: `gopkg.in/yaml.v3` (only third-party dependency; unchanged by this
feature) — standard library otherwise (`net/http`, `encoding/json`, `flag`, `bufio`, `sync`, etc.)

**Storage**: Flat files only — `config.yaml` (input), `endpoints.txt` (output, configurable via
`-o`), `Ph.Sh_URL.log` (resume marker), `failed_domains.txt` (failure list). No database.

**Testing**: Go's built-in `testing` package + `go test ./...`; existing `main_test.go` covers
`isValidDomain`, `cleanDomainLine`, `filterPlaceholderKeys` and will be relocated with the code
it tests.

**Target Platform**: Cross-platform CLI binary (Windows/Linux/macOS via `go build`), no OS-specific code.

**Project Type**: Single-module CLI tool (not a web service, not a library intended for import
by other modules).

**Performance Goals**: N/A — this is a structural refactor; per-domain scan timing (20s
inter-domain delay, retry/backoff) is unchanged.

**Constraints**: Zero change to CLI flags (`-o`, `-d`, `-silent`, `-e`, `-version`), output file
formats/names, or log message text (FR-005). No new external dependencies (Assumptions).

**Scale/Scope**: ~700 lines split across ~4 new packages; no change to concurrency model beyond
relocating existing goroutines/channels into the packages that own them.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Checked against `.specify/memory/constitution.md`:

- **I. Single-Binary CLI, No Runtime Dependencies** — PASS. No new dependency is introduced;
  packages are internal (`internal/...`), still compiled into one binary.
- **II. Never Lose Collected Data** — PASS (with care). `finalUrlSet`/`failedDomains` and the
  `resultsMu` mutex stay in `main.go` (they're orchestration state, not source-specific), so the
  interrupt-safety behavior added in v1.2.0 is unchanged. Task list must include an explicit
  verification step (interrupt behavior unchanged) rather than assuming it.
- **III. Graceful Degradation Per Source** — PASS (with care). Each source's fetch function
  keeps returning `(urls []string)` down a channel and logging its own errors; moving the code
  must not introduce a shared failure path where one source's error can abort another's.
- **IV. Test-First for Pure Logic** — PASS. `internal/domain` (validation/cleaning) and the
  config placeholder-filtering logic keep their existing unit tests, relocated alongside the code.
- **V. CI Is the Merge Gate** — PASS. `.github/workflows/ci.yml` already runs
  `go build ./... && go vet ./... && go test ./...`, which naturally covers the new package
  layout with no workflow changes needed.

No violations — Complexity Tracking section is not needed.

## Project Structure

### Documentation (this feature)

```text
specs/002-split-main-packages/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md         # Phase 1 output
├── quickstart.md         # Phase 1 output
└── tasks.md              # Phase 2 output (/speckit-tasks — not created by this command)
```

### Source Code (repository root)

```text
main.go                        # CLI flags, scan loop, signal handling, output writing (orchestration only)
main_test.go                   # Removed / replaced — tests move with their code (see below)

internal/
├── config/
│   ├── config.go               # Config struct, loadConfig, createDefaultConfig, filterPlaceholderKeys
│   └── config_test.go          # filterPlaceholderKeys tests (from main_test.go)
├── domain/
│   ├── domain.go                # isValidDomain, cleanDomainLine
│   └── domain_test.go           # isValidDomain/cleanDomainLine tests (from main_test.go)
├── logging/
│   └── logging.go               # logMessage type, color constants, shared log-fan-in helper
└── sources/
    ├── retry.go                 # makeRequestWithRetry (shared HTTP-with-retry helper)
    ├── virustotal.go             # getVirusTotalURLs + VirusTotalResponse
    ├── alienvault.go             # getAlienVaultURLs + AlienVaultResponse
    ├── wayback.go                 # getWaybackURLs
    └── hudsonrock.go              # getHudsonRockURLs + HudsonRockResponse
```

**Structure Decision**: Single Go module, no new external boundary. New code lives under
`internal/` (Go-enforced: not importable outside this module, which is correct since this is a
CLI binary, not a library other projects should depend on). Four packages, one per
responsibility identified in the spec's Key Entities: `config`, `domain`, `logging`, `sources`.
`main.go` keeps the scan loop, signal handling, and `resultsMu`-protected result aggregation,
since that state is orchestration-level, not owned by any single source.

## Complexity Tracking

*No violations — table intentionally omitted.*
