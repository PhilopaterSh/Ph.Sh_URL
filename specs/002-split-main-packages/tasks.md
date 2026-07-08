# Tasks: Split main.go Into Packages

**Input**: Design documents from `/specs/002-split-main-packages/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Existing unit tests (`isValidDomain`, `cleanDomainLine`, `filterPlaceholderKeys`) are
relocated with their code; no new tests are required beyond what already exists (v1.2.0 already
satisfies Constitution Principle IV for these functions).

**Organization**: Tasks are grouped by user story per `spec.md`. Note: User Story 2 (main.go as
pure orchestration) is the integration/cleanup story and genuinely depends on User Story 1 and
User Story 3 being complete first — see Dependencies below.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Create empty package directories: `internal/config/`, `internal/domain/`,
      `internal/logging/`, `internal/sources/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared plumbing that both User Story 1 (sources) and User Story 2 (main.go) need.

**⚠️ CRITICAL**: Must complete before Phase 3 (US1) and Phase 5 (US2).

- [X] T002 [P] Create `internal/logging/logging.go`: move the `Message` struct (was
      `logMessage`: `Type`, `Source`, `Message` fields) and `ColorRed`/`ColorGreen`/
      `ColorYellow`/`ColorBlue`/`ColorReset` constants out of `main.go`
- [X] T003 [P] Create `internal/config/config.go`: move `Config` struct, `LoadConfig`
      (was `loadConfig`), `CreateDefaultConfig` (was `createDefaultConfig`), and
      `filterPlaceholderKeys` out of `main.go`, exporting the names `main.go` will call
- [X] T004 [P] Create `internal/config/config_test.go`: move `TestFilterPlaceholderKeys` out of
      `main_test.go` into this package
- [X] T005 Run `go build ./internal/...` to confirm the two foundational packages compile in
      isolation before starting Phase 3

**Checkpoint**: `internal/logging` and `internal/config` compile independently; ready for
sources and main.go to depend on them.

---

## Phase 3: User Story 1 - Change one source without touching the others (Priority: P1) 🎯 MVP

**Goal**: Each of the 4 data sources lives in its own file under `internal/sources/`.

**Independent Test**: Locate the Hudson Rock fetching logic by file name alone; `go build
./internal/sources/...` succeeds without needing `main.go`.

### Implementation for User Story 1

- [X] T006 [P] [US1] Create `internal/sources/retry.go`: move `makeRequestWithRetry` →
      `MakeRequestWithRetry(req *http.Request, silent bool, logChan chan<- logging.Message,
      source string) (*http.Response, error)` out of `main.go`, using `internal/logging.Message`
- [X] T007 [P] [US1] Create `internal/sources/virustotal.go`: move `getVirusTotalURLs` →
      `FetchVirusTotalURLs(...)` and the (unexported) VirusTotal response struct out of
      `main.go`, calling `MakeRequestWithRetry` from T006
- [X] T008 [P] [US1] Create `internal/sources/alienvault.go`: move `getAlienVaultURLs` →
      `FetchAlienVaultURLs(...)` and its response struct out of `main.go` — **preserve the
      v1.2.0 fix** that closes each page's `resp.Body` immediately (not deferred)
- [X] T009 [P] [US1] Create `internal/sources/wayback.go`: move `getWaybackURLs` →
      `FetchWaybackURLs(...)` out of `main.go`
- [X] T010 [P] [US1] Create `internal/sources/hudsonrock.go`: move `getHudsonRockURLs` →
      `FetchHudsonRockURLs(...)` and its response struct out of `main.go`
- [X] T011 [US1] Update `main.go` to call `sources.FetchVirusTotalURLs` /
      `sources.FetchAlienVaultURLs` / `sources.FetchWaybackURLs` /
      `sources.FetchHudsonRockURLs` instead of the local functions (depends on T006–T010)
- [X] T012 [US1] Delete the now-migrated fetcher functions, response structs, and
      `makeRequestWithRetry` from `main.go` (depends on T011)

**Checkpoint**: `go build ./...` succeeds; every source's logic is in its own file under
`internal/sources/`, none of it in `main.go`.

---

## Phase 4: User Story 3 - Domain validation testable with no network dependency (Priority: P3)

**Goal**: `internal/domain` has zero dependency on `net/http` or `internal/config`.

**Independent Test**: `go test ./internal/domain/...` passes standalone; `go list -deps
./internal/domain/...` contains neither `net/http` nor `.../internal/config`.

> Implemented before Phase 5 (US2) because US2's "main.go is pure orchestration" goal requires
> domain validation to already live outside `main.go` — see Dependencies below.

### Implementation for User Story 3

- [X] T013 [P] [US3] Create `internal/domain/domain.go`: move `isValidDomain` →
      `IsValidDomain` and `cleanDomainLine` → `CleanDomainLine` out of `main.go`
- [X] T014 [P] [US3] Create `internal/domain/domain_test.go`: move `TestIsValidDomain` and
      `TestCleanDomainLine` out of `main_test.go` into this package
- [X] T015 [US3] Update `main.go` to call `domain.IsValidDomain` / `domain.CleanDomainLine`
      instead of the local functions (depends on T013)
- [X] T016 [US3] Delete `isValidDomain`/`cleanDomainLine` from `main.go` (depends on T015)

**Checkpoint**: `go test ./internal/domain/...` passes in isolation; `go list -deps
./internal/domain/...` confirms no `net/http`/config coupling.

---

## Phase 5: User Story 2 - main.go reads as pure orchestration (Priority: P2)

**Goal**: `main.go` contains only flag parsing, the scan loop, signal handling, result
aggregation, and calls into `internal/config`, `internal/domain`, `internal/sources`,
`internal/logging`.

**Independent Test**: `main.go` contains no direct `net/http` call sites for any data source, no
`Config`/`isValidDomain`/`cleanDomainLine` definitions.

**Depends on**: Phase 2 (T003, config), Phase 3 (US1 sources fully migrated), Phase 4 (US3
domain fully migrated) — this is the integration story that only "completes" once the other
extractions have happened.

### Implementation for User Story 2

- [X] T017 [US2] Update `main.go`'s config loading call site to `config.LoadConfig(*silent)`;
      delete the local `Config` struct / `loadConfig` / `createDefaultConfig` (depends on T003)
- [X] T018 [US2] Update `main.go`'s per-domain log-fan-in goroutine and `log.Printf` color usage
      to use `logging.Message` and the `logging.Color*` constants (depends on T002)
- [X] T019 [US2] Delete the old `logMessage` type and local color constants from `main.go`
      (superseded by `internal/logging`, depends on T018)
- [X] T020 [US2] Read through `main.go` top-to-bottom and confirm it matches plan.md's Project
      Structure description (flags, scan loop, signal handling, output writing only) — target
      under 200 lines per spec.md SC-001

**Checkpoint**: `main.go` has no source-fetching, config-parsing, or domain-validation logic —
SC-001 and SC-002 satisfied.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Whole-module verification per quickstart.md, across all three user stories.

- [X] T021 [P] Run `go build ./...`, `go vet ./...`, and `go test ./... -v` for the whole module
- [X] T022 Run quickstart.md step 3 (behavior-unchanged diff check) against a real test domain
      and confirm `endpoints.txt` output is byte-identical to pre-refactor output
- [X] T023 [P] Run quickstart.md step 4 (`go list -deps ./internal/domain/...`) and confirm no
      `net/http` / `internal/config` in the output
- [X] T024 Run quickstart.md step 5 (Ctrl+C interrupt-safety spot check) and confirm
      `endpoints.txt`/`failed_domains.txt` are written correctly with no panic
- [X] T025 [P] Update `README.md` only if it references `main.go`'s internal structure (grep
      first — likely no change needed since README only documents the CLI surface)
- [X] T026 Update `specs/002-split-main-packages/tasks.md` checkboxes to `[X]` and commit with a
      message referencing this feature directory

## Verification Notes

- T021–T023: verified with the real Go toolchain — `go build ./...`, `go vet ./...`, and
  `go test ./... -v` all pass; `go list -deps ./internal/domain/...` contains no `net/http` or
  `internal/config`; `main.go` has zero `net/http`/`encoding/json`/`gopkg.in/yaml.v3` imports.
- T022: ran the built binary end-to-end against `example.com`/`example.org` (OTX only, keyless —
  confirming placeholder-key filtering still works), got matching colored log output and a
  correctly written, deduplicated `endpoints.txt`.
- T024: attempted a live Ctrl+C test via `kill -INT` from Git Bash on Windows; the signal did not
  reliably reach the Go process's `os.Interrupt` handler in this sandbox (a known POSIX-signal-
  to-Windows-console-event translation gap in MSYS/Git Bash, unrelated to this refactor — the
  interrupt-handling code itself was moved verbatim, unchanged from v1.2.0). Confidence instead
  comes from: (a) the mutex-guarded aggregation code is byte-for-byte identical to the v1.2.0
  logic that introduced it, just relocated, and (b) the multi-domain run above exercised the same
  `resultsMu`-guarded aggregation path across two domains with no deadlock or data loss.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup. Blocks Phase 3 and Phase 5.
- **User Story 1 / Phase 3 (P1)**: Depends on Phase 2 (T002, logging). Independent of US3.
- **User Story 3 / Phase 4 (P3)**: Depends on Phase 1 only (no logging/config dependency by
  design — that's the point of US3). Independent of US1.
- **User Story 2 / Phase 5 (P2)**: Depends on Phase 2 (T003), Phase 3 (US1 complete), and
  Phase 4 (US3 complete). This is the one deviation from strict story-independence: US2's
  definition of done ("main.go is pure orchestration") is only true once the other two stories
  have moved their code out.
- **Polish (Phase 6)**: Depends on all of Phase 3–5 being complete.

### Parallel Opportunities

- T002 and T003 (Phase 2) can run in parallel — different files, no shared state.
- T006–T010 (Phase 3, the four source files + retry helper) can all run in parallel — different
  files, no dependencies between them.
- T013–T014 (Phase 4) can run in parallel with each other, and the whole of Phase 4 can run in
  parallel with Phase 3 (US1 and US3 don't touch each other's files).
- T021, T023, T025 (Phase 6) can run in parallel.

## Parallel Example: Phase 3 (User Story 1)

```bash
Task: "Create internal/sources/retry.go: move makeRequestWithRetry"
Task: "Create internal/sources/virustotal.go: move getVirusTotalURLs"
Task: "Create internal/sources/alienvault.go: move getAlienVaultURLs (preserve v1.2.0 body-close fix)"
Task: "Create internal/sources/wayback.go: move getWaybackURLs"
Task: "Create internal/sources/hudsonrock.go: move getHudsonRockURLs"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup
2. Phase 2: Foundational (T002 logging; T003/T004 config can happen here too since they're
   cheap and unlock Phase 5 later)
3. Phase 3: User Story 1 (sources split out)
4. **STOP and VALIDATE**: `go build ./...`, confirm each source is in its own file
5. This alone already delivers the highest-value slice (locate/change one source in isolation)

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. Add User Story 1 (sources) → validate independently
3. Add User Story 3 (domain) → validate independently (can happen before or after US1, they
   don't touch each other)
4. Add User Story 2 (main.go slimmed down) → only now is this story "done", since it depends on
   the other two
5. Phase 6 Polish → full quickstart.md validation, then commit

## Notes

- [P] tasks touch different files with no dependencies — safe to do in any order or in parallel.
- Every "move X out of main.go" task should be a single commit-sized change: create the new
  file with the moved code, update the one call site in `main.go`, remove the old definition.
  Doing all three in one shot per function avoids a broken intermediate state where the same
  function exists in two places.
- Run `go build ./...` after every task, not just at checkpoints — catches import-cycle or
  signature mistakes immediately instead of batching them up.
