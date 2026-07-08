# Feature Specification: Reliability Fixes & Spec-Kit Adoption (v1.2.0)

**Feature Branch**: `001-reliability-fixes-speckit`

**Created**: 2026-07-08

**Status**: Completed (documented retroactively — implemented directly, then recorded here for traceability)

**Input**: User description: "Fix concurrency bugs, harden config loading, add tests and CI, adopt spec-kit for v1.2.0"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scan runs without leaking or racing under load (Priority: P1)

As someone running `Ph.Sh_url` against many domains with OTX enabled, I need long scans to
run without leaking HTTP connections or corrupting results, so that large batches complete
reliably and Ctrl+C always saves a consistent result set.

**Why this priority**: These were real correctness bugs (a response-body leak and a data
race on shared state) that could cause resource exhaustion or corrupted/lost output on
interrupt — the core reliability promise of the tool.

**Independent Test**: Run the tool against a domain with many OTX result pages and confirm
response bodies are closed per page (not deferred to end of domain); trigger Ctrl+C mid-scan
and confirm `endpoints.txt` / `failed_domains.txt` are written from a consistent, lock-protected
snapshot.

**Acceptance Scenarios**:

1. **Given** a domain with multiple OTX pagination pages, **When** the tool processes it,
   **Then** each page's HTTP response body is closed immediately after being read instead of
   accumulating open bodies until the domain finishes.
2. **Given** a scan in progress, **When** the user sends Ctrl+C, **Then** the interrupt handler
   and the main loop never read/write `finalUrlSet`/`failedDomains` concurrently without holding
   the shared mutex.

---

### User Story 2 - First run with an unedited config doesn't waste requests (Priority: P2)

As a first-time user who hasn't edited `config.yaml` yet, I need the tool to treat the
placeholder values (`YOUR_VT_API_KEY_1`, etc.) as "no key provided" rather than as real keys,
so sources fall back to keyless mode instead of failing every request with an invalid key.

**Why this priority**: Silent, repeated failed API calls on every domain waste the 20s
per-domain request budget and produce misleading error logs for new users.

**Independent Test**: Run with a freshly generated default `config.yaml` and confirm VT/OTX/HR
requests are made without an API key (or skipped, for VT) instead of using the literal
placeholder string.

**Acceptance Scenarios**:

1. **Given** `config.yaml` contains only the default placeholder keys, **When** the config is
   loaded, **Then** `filterPlaceholderKeys` removes them and each source proceeds in keyless mode.
2. **Given** `config.yaml` contains a mix of a placeholder and a real key, **When** the config is
   loaded, **Then** only the real key is retained.

---

### User Story 3 - Changes are verifiable and repeatable (Priority: P3)

As a maintainer, I need a minimal automated test suite and CI so that future changes to domain
validation/cleaning/config parsing are checked automatically instead of relying on manual review.

**Why this priority**: The project had zero tests and no CI before this change, so regressions
in the pure helper functions could ship silently.

**Independent Test**: Run `go test ./...` locally and confirm the GitHub Actions workflow runs
`go build`, `go vet`, and `go test` on every push/PR.

**Acceptance Scenarios**:

1. **Given** a pull request, **When** CI runs, **Then** `go build ./...`, `go vet ./...`, and
   `go test ./...` all execute and must pass before merge is expected.

### Edge Cases

- Empty or whitespace-only API key entries in `config.yaml` are treated the same as unedited
  placeholders (filtered out).
- A domain with zero OTX pages (single-page response with `has_next: false`) must still close
  its one response body.
- Ctrl+C arriving between domains (during the inter-domain sleep) must still produce a
  consistent, mutex-protected snapshot of results.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The AlienVault OTX fetcher MUST close each paginated response body immediately
  after reading it, not defer closing until the domain's pagination loop exits.
- **FR-002**: All reads and writes of `finalUrlSet` and `failedDomains` from both the main scan
  loop and the Ctrl+C interrupt handler MUST be protected by a shared mutex.
- **FR-003**: `loadConfig` MUST strip empty and unedited placeholder API keys (values starting
  with `YOUR_`) from `VirusTotalKeys`, `AlienVaultKeys`, and `HudsonRockKeys` before they are used.
- **FR-004**: The repository MUST have an automated test suite covering `isValidDomain`,
  `cleanDomainLine`, and `filterPlaceholderKeys`.
- **FR-005**: The repository MUST run `go build`, `go vet`, and `go test` automatically via CI
  on every push and pull request.
- **FR-006**: The project MUST be scaffolded for spec-driven development via spec-kit
  (`.specify/`, Claude Code `speckit-*` skills) so future features go through
  `/speckit-specify` → `/speckit-plan` → `/speckit-tasks` → `/speckit-implement`.

### Key Entities

- **Config**: parsed `config.yaml`, holds per-source API key lists (VirusTotal, AlienVault,
  Hudson Rock); now filtered to exclude placeholder/empty entries.
- **finalUrlSet / failedDomains**: shared, mutex-protected state accumulated across the main
  scan loop and read by the interrupt handler for graceful-shutdown writes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `go vet ./...` reports no warnings on the shared result state.
- **SC-002**: `go test ./...` passes with the new test suite (3 test functions, all green).
- **SC-003**: A scan against a domain with an unedited `config.yaml` makes zero requests using
  a literal `YOUR_..._API_KEY_1` value.
- **SC-004**: CI (`.github/workflows/ci.yml`) runs build/vet/test on every push and PR to `main`.

## Assumptions

- No Go toolchain was available in the working environment at implementation time; verification
  was initially done by manual code review, then re-verified with `go build`/`vet`/`test` once
  the toolchain was located at `C:\Program Files\Go\bin\go.exe`.
- The GitHub repository (`github.com/PhilopaterSh/Ph.Sh_URL`) already had real history (tags
  v1.1.6–v1.1.9); this feature's commit was cherry-picked onto that real `main` rather than
  replacing it, and released as `v1.2.0`.
- Larger architectural changes (e.g. splitting `main.go` into packages) are out of scope for
  this feature and are left for a future spec-kit feature.
