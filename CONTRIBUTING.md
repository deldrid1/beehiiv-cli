# Contributing

Thanks for taking the time to contribute.

## Getting Started

1. Install Go `1.26` or newer.
2. Clone the repository.
3. Run `make build` to compile the CLI.
4. Run `make test` to verify the local environment.

## Local Workflow

- `make build` builds `./dist/beehiiv`
- `make test` runs the Go test suite
- `make fmt` formats Go sources with `gofmt`
- `make fmt-check` verifies formatting without mutating files
- `make lint` runs `go vet ./...`

If you want the local pre-commit hook:

```bash
git config core.hooksPath .githooks
```

## Pull Requests

- Keep changes focused and well-scoped.
- Add or update tests for behavior changes.
- Update docs when CLI behavior or contributor workflow changes.
- Include enough context in the PR summary for reviewers to understand the user-facing impact.

## Reporting Security Issues

Please do not open public issues for security-sensitive bugs. Follow the process in `SECURITY.md`.
