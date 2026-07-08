# Phase 0 Research: Split main.go Into Packages

No `NEEDS CLARIFICATION` markers were left in the Technical Context — this is a refactor of an
existing, fully-understood codebase (not new technology adoption), so research here is about
package-layout decisions rather than technology unknowns.

## Decision: Use `internal/` for the new packages

**Rationale**: `Ph.Sh_url` is a CLI binary, not a library meant to be imported by other Go
modules (`go.mod` module path `github.com/PhilopaterSh/Ph.Sh_url` is used only to build the
binary). Go's `internal/` convention makes this explicit and compiler-enforced: nothing outside
this module can import `internal/sources`, etc. This matches Constitution Principle I
(single-binary CLI, no externally-consumed library surface).

**Alternatives considered**: Top-level packages (`config/`, `sources/`, ...) without the
`internal/` prefix — rejected because it would (misleadingly) suggest these packages are a
supported public API, when the project's only public surface is the CLI itself.

## Decision: One package per responsibility, one file per source inside `sources/`

**Rationale**: Directly satisfies User Story 1 (locate/change one source without touching
others) and User Story 3 (domain validation testable with zero network dependency). A single
`sources` package with 4 files (rather than 4 separate packages) was chosen over one-package-
per-source because the four fetchers share the same signature shape, the same retry helper
(`makeRequestWithRetry`), and the same `logMessage` channel protocol — splitting them into
separate packages would force that shared plumbing into a 5th package anyway, adding indirection
without adding testability (each file is already independently readable and touchable).

**Alternatives considered**:
- One package per source (`sources/virustotal`, `sources/otx`, ...) — rejected as over-fragmented
  for ~50-90 lines per source; would multiply import boilerplate in `main.go` for no testability
  gain over separate files in one package.
- Keep everything in `main` package, just split across multiple files (`main.go`,
  `sources.go`, `config.go`) — rejected because it doesn't satisfy FR-003 (domain validation
  must have no dependency on network/config code) as a compiler-enforced guarantee; same-package
  files can still accidentally couple.

## Decision: `resultsMu`, `finalUrlSet`, `failedDomains` stay in `main.go`

**Rationale**: This state is cross-source aggregation (shared by all 4 fetchers' results) and
is read by the Ctrl+C interrupt handler — it belongs to orchestration, not to any single
package. Moving it into e.g. `internal/sources` would create a dependency from `sources` back
into result-aggregation concerns it doesn't own, and would risk re-introducing the v1.2.0 data
race fix if the mutex and the state it guards are split across package boundaries.

**Alternatives considered**: A dedicated `internal/results` package wrapping `finalUrlSet`/
`failedDomains`/`resultsMu` behind an API (e.g. `Add(urls []string)`, `Snapshot()`) — considered
a reasonable follow-up but deferred: it's an orthogonal improvement (encapsulating mutable state
behind a type) rather than something this feature's user stories require, and the smaller change
is easier to verify against the "zero behavior change" constraint (FR-005).

## Decision: Shared log-fan-in plumbing goes in `internal/logging`

**Rationale**: `logMessage`, the color constants, and the pattern of draining a `chan
logMessage` into colored `log.Printf` calls are used identically by all four sources and by
`main.go`. Centralizing the type + constants avoids duplicating them across `sources/*.go`.
The actual per-domain fan-in goroutine (`for msg := range logChan { ... }`) stays in `main.go`
since it's part of the scan loop's lifecycle (one goroutine per domain iteration), not owned by
any package.

**Alternatives considered**: Put color constants directly in `main` and have `sources` take a
pre-formatted string channel — rejected because it would leak presentation formatting into the
orchestration loop instead of keeping `logMessage{Type, Source, Message}` as a structured value
sources can emit without knowing about ANSI colors.

**Output**: All `NEEDS CLARIFICATION` items resolved (none existed); package layout decisions
documented above feed directly into `data-model.md` and `tasks.md`.
