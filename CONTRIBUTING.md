# Contributing

We treat this repo as "Open Source" within Redis: anyone who clears the bar below is welcome to contribute.

## Local setup

This repo is a Go project. You need Go 1.24+ installed.

```bash
git clone git@github.com:redis-performance/bcc-exporter.git
cd bcc-exporter

# Download and tidy Go module dependencies
make deps

# Build the binary
make build
```

### System dependencies

The server wraps two profiling back-ends and each requires separate system packages on Linux:

**perf + pprof (binary pprof endpoint):**
```bash
sudo apt-get install linux-perf
go install github.com/google/pprof@latest
```

**BCC tools (folded stacks endpoint):**
```bash
sudo apt-get install bpfcc-tools linux-headers-$(uname -r)
```

Both back-ends require Linux kernel 4.9+ with BPF support. If the system
packages are unavailable you can still exercise the HTTP layer by appending
`&test=true` to any request; this returns deterministic mock data without
invoking perf or BCC.

Running the server itself needs elevated privileges because `perf record` and
`profile-bpfcc` both require kernel access:

```bash
sudo ./bcc-exporter          # default port 8080
sudo ./bcc-exporter -port 9090
sudo ./bcc-exporter -password mysecretpassword
```

## Branch naming

```
<type>/<short-description>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Example: `feat/add-pipeline-mode`

## Coding standards

- Keep changes focused; one logical change per PR.
- Follow the conventions already present in the codebase (formatting, naming, error handling).
- No dead code, no commented-out blocks.

## Submitting changes

1. Fork or create a branch from `main`.
2. Make your changes with clear, atomic commits.
3. Open a pull request against `main` with a descriptive title and summary.
4. Address review comments promptly; force-push to the same branch to update.

## Testing

- All new behaviour must be covered by tests.
- Existing tests must pass: run the test suite locally before opening a PR.
- Coverage should not decrease.

Run the full unit test suite (no root, no BCC/perf required):

```bash
make test
# or equivalently
go test ./...
```

Integration tests that exercise real `perf` + `pprof` calls are skipped
automatically when those tools are absent or the process is not running as
root. To run them explicitly:

```bash
sudo go test ./... -run TestPerfIntegration
```

To additionally run format and vet checks before submitting:

```bash
make dev   # runs: go fmt ./... && go vet ./... && go build
```

## Review process

- At least one maintainer approval is required before merge.
- CI must be green.
- Maintainers may request changes or close PRs that do not meet the bar — this is normal and not personal.
