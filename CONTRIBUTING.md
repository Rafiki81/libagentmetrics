# Contributing

Thanks for contributing to **libagentmetrics**.

## Branch and PR workflow

- Never push directly to `main`/`master`.
- Create a branch per change:
  - `fix/<short-description>`
  - `feat/<short-description>`
  - `chore/<short-description>`
- Open a pull request to `main`.
- Keep PRs focused and small when possible.

## Commit style

Prefer Conventional Commits:

- `feat: ...`
- `fix: ...`
- `docs: ...`
- `perf: ...`
- `refactor: ...`
- `test: ...`
- `ci: ...`
- `chore: ...`

## Validation checklist

Before requesting review, run:

```bash
go test ./...
go test -race ./...
go vet ./...
```

For hot-path changes in `monitor`, also run benchmarks:

```bash
go test ./monitor -run '^$' -bench 'Benchmark(ParseCursorDBLines|FormatTokenCount|AlertMonitorCheckNoAlert|AlertMonitorCheckFleet)$' -benchmem
```

## Recommended branch protection settings

Configure this in GitHub repository settings for `main` (and `master` if used):

- Require a pull request before merging.
- Require at least 1 approval.
- Require status checks to pass before merging:
  - CI / Vet + Test + Race
  - CI / Bench regression (warning-only job can remain non-blocking)
- Dismiss stale approvals when new commits are pushed.
- Require conversation resolution before merging.
- Block force pushes and deletions.
- Optional: Require linear history + squash merges.

## Security and performance expectations

- Keep changes minimal and dependency-light.
- Avoid introducing long-running background work in monitoring hot paths.
- Preserve API stability for `v1.x` public surface.
