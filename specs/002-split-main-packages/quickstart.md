# Quickstart: Validating the Package Split

Prerequisites: Go 1.25+ toolchain available (`go version`), repository checked out on branch
`002-split-main-packages` (or with these changes applied on top of `main`).

## 1. Build and static checks

```sh
go build ./...
go vet ./...
```

Both must succeed with no errors/warnings — confirms the new package boundaries compile and the
import graph has no cycles (config/domain/logging/sources/main).

## 2. Automated tests

```sh
go test ./... -v
```

Expected: the same three test cases that existed before the split
(`TestIsValidDomain`, `TestCleanDomainLine`, `TestFilterPlaceholderKeys`), now running from
`internal/domain` and `internal/config` respectively, all green.

## 3. Behavior-unchanged check (FR-005 / SC-003)

Run the tool against a small, known domain list both before and after the refactor and diff the
output:

```sh
# Before refactor (e.g. on main / v1.2.0)
./Ph.Sh_url -d testdomains.txt -o before.txt -silent

# After refactor (this branch)
go build -o Ph.Sh_url_new .
./Ph.Sh_url_new -d testdomains.txt -o after.txt -silent

diff before.txt after.txt   # expect no differences (same sources, same dedup logic)
```

Also confirm unchanged non-silent output format by eye: banner, `[INFO]`/`[SUCCESS]` colored
log lines, and the `-version` flag output (`Ph.Sh_url version: 1.2.0` or current version)
should read identically to before the split.

## 4. Package boundary check (SC-002 / SC-004)

```sh
go list -deps ./internal/domain/...
```

Expected: no `net/http` and no `.../internal/config` in the dependency list — confirms domain
validation has zero network/config coupling, as required by User Story 3 / FR-003.

## 5. Interrupt-safety spot check (Constitution Principle II)

Start a scan against a domain list with several entries, let it process at least one domain,
then send Ctrl+C mid-scan:

```sh
./Ph.Sh_url -d testdomains.txt
# ...press Ctrl+C after the first "[INFO] Processing domain 1/N" line...
```

Expected: `[WARNING] Interrupt signal received. Saving results...` followed by
`[SUCCESS] Results saved to endpoints.txt`, with no panic, no data race warning if run under
`go run -race`, and `endpoints.txt` containing whatever URLs were found before the interrupt.
