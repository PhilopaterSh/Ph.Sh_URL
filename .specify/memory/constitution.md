# Ph.Sh_URL Constitution

## Core Principles

### I. Single-Binary CLI, No Runtime Dependencies
`Ph.Sh_url` ships as one self-contained Go binary. No external runtime (Docker, Python, Node)
is required to run it; the only dependency is `gopkg.in/yaml.v3` for config parsing. Keep it
this way — new features must not require a database, daemon, or additional installed tool to
run the core scan.

### II. Never Lose Collected Data
The tool's core promise is that a scan can be interrupted (Ctrl+C, crash, network loss) without
losing previously found URLs. Any change touching `finalUrlSet`, `failedDomains`, the log file,
or the interrupt handler MUST preserve: (a) atomic-enough writes so a partial write doesn't
corrupt `endpoints.txt`, and (b) safe concurrent access — no goroutine may read/write shared
scan state without holding `resultsMu`.

### III. Graceful Degradation Per Source
Each data source (VirusTotal, AlienVault OTX, Wayback Machine, Hudson Rock) must fail
independently: a missing API key, a source outage, or a rate limit on one source must never
abort the scan for the other sources or for other domains. Prefer logging a warning and
returning an empty result over `log.Fatal`.

### IV. Test-First for Pure Logic (NON-NEGOTIABLE)
Any function with no I/O side effects (domain validation/cleaning, key filtering, URL
deduplication logic, etc.) MUST have unit tests in the corresponding `_test.go` file before
being considered done. Functions that do network I/O are exempt from unit testing but must be
reviewed for correct error handling and resource cleanup (closed response bodies, no goroutine
leaks).

### V. CI Is the Merge Gate
`go build ./...`, `go vet ./...`, and `go test ./...` MUST pass in CI
(`.github/workflows/ci.yml`) before a change is considered mergeable to `main`. Do not bypass
CI failures by weakening tests or `vet` findings — fix the underlying issue.

## Versioning & Releases

- Version format: `MAJOR.MINOR.PATCH`, tracked in the `version` constant in `main.go` and
  mirrored in the README badge and "What's New" section.
- Every release gets an annotated git tag (`vX.Y.Z`) and a corresponding GitHub Release with
  notes summarizing user-visible changes and bug fixes.
- Breaking changes to CLI flags or `config.yaml` structure require a MAJOR bump.

## Development Workflow

- New features and non-trivial fixes go through spec-kit: `/speckit-specify` → `/speckit-plan`
  → `/speckit-tasks` → `/speckit-implement`, with specs stored under `specs/NNN-short-name/`.
- Small, obvious bug fixes (typos, off-by-one, clearly-scoped correctness fixes) may be applied
  directly without a full spec, but should still be covered by a test where the bug is in pure
  logic.
- Keep `main.go` as a single file only as long as it stays readable; if/when it grows
  significantly, splitting into packages (e.g. `sources/`, `config/`) is an explicit
  architectural decision that should go through `/speckit-specify`, not an incidental
  refactor bundled into an unrelated change.

## Governance

This constitution reflects how `Ph.Sh_URL` is actually maintained as of v1.2.0. Amendments
should be made when a real decision changes (e.g. adding a new data source, changing the
concurrency model) — update this file in the same change that makes the decision, not
separately.

**Version**: 1.0.0 | **Ratified**: 2026-07-08 | **Last Amended**: 2026-07-08
