# Feature Specification: Split main.go Into Packages

**Feature Branch**: `002-split-main-packages`

**Created**: 2026-07-08

**Status**: Completed — implemented via `/speckit-plan` → `/speckit-tasks` →
`/speckit-implement` workflow; see `tasks.md` for verification notes.

**Input**: User description: "Split the single main.go file into well-organized Go packages (config, sources, urlstore) so the codebase is easier to navigate, test, and extend with new data sources"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find and change one data source without reading the whole file (Priority: P1)

As a maintainer of `Ph.Sh_url`, I want each data source (VirusTotal, AlienVault OTX, Wayback
Machine, Hudson Rock) to live in its own file/package, so that fixing or extending one source
doesn't require reading or risking changes to the other three, to config loading, or to CLI
orchestration.

**Why this priority**: This is the direct pain point that motivated the refactor — `main.go`
currently mixes CLI flags, config loading, HTTP fetching for 4 unrelated sources, and the
scan/output orchestration in one ~700-line file. It's the highest-value, most independently
useful slice of the split.

**Independent Test**: Open the repository and locate the Hudson Rock fetching logic by
directory/file name alone (no need to search inside `main.go`); modify it and run
`go test ./sources/...` (or equivalent) without touching any other package.

**Acceptance Scenarios**:

1. **Given** the refactored codebase, **When** a maintainer needs to change how OTX pagination
   works, **Then** they only need to open the OTX source file and do not need to edit `main.go`.
2. **Given** the refactored codebase, **When** `go build ./...` is run, **Then** it produces the
   same CLI binary with unchanged flags and behavior as before the split.

---

### User Story 2 - main.go reads as pure orchestration (Priority: P2)

As a maintainer, I want `main.go` reduced to flag parsing, the scan loop, signal handling, and
wiring the config/sources/output packages together, so the entry point can be read top-to-bottom
in a couple of minutes instead of scrolling past HTTP-fetching code for four unrelated sources.

**Why this priority**: Directly enables User Story 1 — without shrinking `main.go`, moving
source code out doesn't actually make the entry point easier to read.

**Independent Test**: Read `main.go` alone and confirm it contains no direct `net/http` calls
for any of the four data sources.

**Acceptance Scenarios**:

1. **Given** the refactored codebase, **When** a new contributor opens `main.go`, **Then** they
   see only flag definitions, the scan loop, signal handling, and calls into other packages —
   no inline HTTP request construction for any data source.

---

### User Story 3 - Domain validation is independently testable (Priority: P3)

As a maintainer, I want domain validation/cleaning logic in its own package with no network
dependencies, so its unit tests run instantly and don't need mocking of HTTP clients.

**Why this priority**: Lowest risk, smallest slice — it formalizes what already exists
(`main_test.go` already unit-tests these functions) into a package boundary that makes the
"no network dependency" property explicit and enforced by the Go compiler (import graph).

**Independent Test**: Run the domain-validation package's tests in isolation
(e.g. `go test ./domain/...`) and confirm they complete without any network access.

**Acceptance Scenarios**:

1. **Given** the refactored codebase, **When** `isValidDomain`/`cleanDomainLine` tests run,
   **Then** they run from a package that imports no HTTP or YAML-config packages.

---

### Edge Cases

- Config and sources must not form an import cycle: sources need API keys, but must receive
  them as parameters/arguments rather than importing the config package directly.
- Shared constants used across sources (color codes, retry attempts/delay, the `logMessage`
  type used for the log-fan-in channel) need one shared home so all source files can use them
  without duplicating definitions.
- The refactor must not change the on-disk `config.yaml` format, output file format, log file
  name/format, or any CLI flag — this is a structure-only change, not a behavior change.
- Existing tests (`isValidDomain`, `cleanDomainLine`, `filterPlaceholderKeys`) must keep passing
  after being relocated to their new package(s).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST move configuration loading and parsing (`Config` struct,
  `loadConfig`, `createDefaultConfig`, `filterPlaceholderKeys`) into a dedicated package.
- **FR-002**: The system MUST move each data-source fetcher (VirusTotal, AlienVault OTX,
  Wayback Machine, Hudson Rock) and its response-struct types into a dedicated package, with
  one file per source.
- **FR-003**: The system MUST move domain validation/cleaning (`isValidDomain`,
  `cleanDomainLine`) into a package with no dependency on network or config packages.
- **FR-004**: `main.go` MUST be limited to: flag parsing, the domain scan loop, signal handling,
  result aggregation/output, and wiring the above packages together.
- **FR-005**: The refactor MUST NOT change any user-visible CLI behavior: flag names/semantics,
  `endpoints.txt`/`failed_domains.txt`/`Ph.Sh_URL.log` formats and names, or log message text.
- **FR-006**: All existing tests MUST continue to pass after being relocated to their new
  package, with no reduction in coverage.
- **FR-007**: The retry/HTTP-client helper (`makeRequestWithRetry`) MUST live alongside the
  sources package(s) that use it, without duplicating it per source.
- **FR-008**: The shared logging plumbing (`logMessage` type, color constants, the log-fan-in
  goroutine pattern) MUST have a single defined home usable by every source package.
- **FR-009**: The module MUST continue to build as `github.com/PhilopaterSh/Ph.Sh_url` with
  internal packages importable under that module path (e.g. `.../config`, `.../sources`).

### Key Entities

- **config package**: `Config` struct (VirusTotal/AlienVault/HudsonRock key lists), config file
  path resolution, default config file creation, placeholder-key filtering.
- **sources package**: one fetcher per external source, each taking a domain + keys + shared
  log/HTTP-retry plumbing and returning found URLs; owns the per-source response JSON structs.
- **domain package**: pure functions for validating and cleaning a domain string; no
  dependencies outside the standard library.
- **main (orchestrator)**: CLI flags, the per-domain scan loop, Ctrl+C signal handling,
  mutex-protected result aggregation, and final output writing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `main.go` shrinks from ~700 lines to CLI wiring only, with zero direct
  `net/http`/`encoding/json`/`gopkg.in/yaml.v3` imports (measured result: 360 lines, a 49%
  reduction — the original 200-line target undercounted how much space flag parsing, the resume
  log check, signal handling, and output writing legitimately take).
- **SC-002**: Each of the 4 data sources is locatable by file name alone, without opening
  `main.go` or any other source's file.
- **SC-003**: `go build ./...`, `go vet ./...`, and `go test ./...` all pass after the refactor,
  with zero change to CLI flags, output file formats, or log message text (verified by running
  the tool against a known domain before and after and diffing the output).
- **SC-004**: Package-level tests for domain validation run without importing `net/http` or the
  config package (verifiable via `go list -deps`).

## Assumptions

- Package layout follows standard Go convention: `main` package stays at the repo root
  (`main.go`), with new packages under subdirectories (e.g. `config/`, `sources/`, `domain/`).
- No new external dependencies are introduced; the only third-party dependency remains
  `gopkg.in/yaml.v3`.
- This is a behavior-preserving internal refactor — no new CLI flags, sources, or output formats
  are added as part of this feature.
- Splitting is done in one pass rather than incrementally across multiple releases, since the
  codebase is small enough (~700 lines) that partial states would add more churn than value.
