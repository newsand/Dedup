# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-22

### Added

- First public release.
- CLI `dedup` with subcommands:
  - `scan`, `report`                — discover duplicates, no filesystem changes
  - `copy-unique`                   — copy uniques + canonicals to a flattened output
  - `move-duplicates`               — move duplicates to a quarantine directory
  - `delete-duplicates`             — delete duplicates (requires `--yes`)
  - `version`
- Exact duplicate detection based on **BLAKE3** content hashing (streamed).
- Deterministic canonical selection: oldest `mtime`, lexical path tiebreak.
- `snake_case` + flatten naming with `--suppressname` and deterministic
  collision suffixes (`_1`, `_2`, ...).
- Write-ahead JSONL audit log at `./.dedup-audit/<started_at>.jsonl`.
- YAML configuration with environment variables (`DEDUP_*`) and CLI flag
  layering; precedence: flag > env > YAML > default.
- Safety invariants enforced before executing any destructive action.
- Cross-platform support: Linux (amd64, arm64) and Windows (amd64).
- goreleaser packaging: tarballs, zip and `.deb`.
- Unit, integration, and opt-in E2E test suites (`-tags=e2e`).
- GitHub Actions CI matrix for Linux + Windows.

### Security

- Default dry-run for every destructive command.
- `delete-duplicates` refuses to run without an explicit `--yes`.
- Copy never overwrites a non-identical destination; mismatches fail loudly.
