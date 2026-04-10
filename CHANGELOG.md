# Changelog

All notable changes to this project will be documented in this file.

## [v1.1.1] - 2026-04-10

### Fixed

- Restored fleet budget and burn-rate fields in `config.AlertConfig`, aligning configuration, tests, and the example with `AlertMonitor`.
- Made fleet budget alert evaluation deterministic by using an internal clock inside `AlertMonitor`.
- Improved `SecurityMonitor` matching so regex-like rules such as `curl.*|.*sh` are honored without misclassifying safe commands that only contain literal pipe syntax.
- Updated `examples/basic` to call `LocalModelMonitor.Collect()` before reading local model results.

### Docs

- Corrected README release metadata, config API signatures, default config path, `HistoryStore` constructor signature, runtime dependency notes, and the Codex CLI agent ID.
- Documented this patch release and linked the README to the changelog.

## [v1.1.0] - 2026-02-15

### Added

- Token signal confidence in `TokenMetrics.Confidence` so consumers can weigh metric reliability.
- Burn-rate guardrails with warning and critical thresholds (`burn_rate_warning`, `burn_rate_critical`).
- Fleet budget alerts with daily and monthly limits via aggregated `CheckFleet` checks.

### Improved

- Operational hardening in monitors, including safer defaults, pruning, and more resilient fallback paths.
- Release and CI maturity with stable `v1.x` API policy and automated release/check workflows.
