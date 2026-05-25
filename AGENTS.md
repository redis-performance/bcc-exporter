# Agent guidelines

Instructions for AI coding agents (Claude Code, Copilot, Cursor, etc.) working in this repo.

## Project overview

`bcc-exporter` is a minimal HTTP server written in Go (1.24, no external
dependencies) that exposes Linux CPU profiling data over HTTP. It provides two
endpoints: `/debug/pprof/profile` returns a binary pprof file produced by
`perf record` + `pprof` conversion (compatible with `go tool pprof`), and
`/debug/folded/profile` returns folded stack traces produced by
`profile-bpfcc` (suitable for Flamegraph). Both endpoints accept `?pid=<pid>&seconds=<n>`
query parameters and support a `&test=true` mode that returns deterministic
mock data without invoking any kernel-level tools. Optional HTTP basic
authentication (`-password` flag, username `admin`) is available for both
endpoints. The server is designed to run as root or with `CAP_SYS_ADMIN`
because both `perf` and BCC tools require elevated kernel access.

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

System dependencies for the two profiling back-ends (Linux only):

```bash
# perf + pprof (binary pprof endpoint)
sudo apt-get install linux-perf
go install github.com/google/pprof@latest

# BCC tools (folded stacks endpoint)
sudo apt-get install bpfcc-tools linux-headers-$(uname -r)
```

If neither back-end is available locally, use `&test=true` on any request to
exercise the HTTP layer without kernel access.

## Branch naming

Same as human contributors: `<type>/<short-description>` (e.g. `fix/off-by-one-in-pipeline`).

## Coding standards

- Match the style already in the file you are editing.
- Prefer clear, minimal changes over large refactors unless explicitly asked.
- Do not add comments that describe *what* the code does — only add comments when the *why* is non-obvious.
- Do not introduce new dependencies without checking with the maintainer. The
  module intentionally has zero external Go dependencies; keep it that way.

## Running tests

Run the full unit test suite (no root or BCC/perf tools required):

```bash
make test
# or equivalently
go test ./...
```

Integration tests (`TestPerfIntegration`, `TestCheckRequiredTools`) skip
automatically when the required tools are absent or the process is not root.
To run them explicitly:

```bash
sudo go test ./... -run TestPerfIntegration
```

Before declaring a task complete, also run:

```bash
make dev   # go fmt ./... && go vet ./... && go build
```

Always run tests before declaring a task complete.

## How to submit changes

1. Create a branch: `git checkout -b <type>/<description>`.
2. Commit with a clear message focused on *why*, not *what*.
3. Open a pull request against `main`.
4. Do **not** push directly to `main`.

## What to avoid

- Do not reformat files unrelated to your change.
- Do not remove error handling or tests.
- Do not commit secrets, credentials, or large binary files.
- Do not amend published commits.
- Do not add external Go module dependencies; the project has none by design.
- Do not call `perf`, `pprof`, or `profile-bpfcc` directly in tests — use
  `&test=true` mode or table-driven unit tests against the HTTP handlers.
- Do not run the server binary in CI or tests; it requires root and live kernel
  subsystems unavailable in most CI environments.
