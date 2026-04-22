---
name: dedup
description: Use the Deduplicator CLI (BLAKE3 exact duplicates) from OpenClaw agents.
user-invocable: true
metadata: {"openclaw":{"emoji":"🧹","requires":{"bins":["dedup"]},"install":[{"kind":"download","label":"Install dedup (uses {baseDir}/scripts/install-dedup.sh)"}]}}
---

## What this skill does

This skill teaches the agent how to use the `dedup` CLI to find and handle **byte-exact** duplicates (BLAKE3).

## Quick usage

- Report duplicates in the current directory:

```bash
dedup report .
```

- Copy uniques + canonicals into a flattened directory:

```bash
dedup copy-unique . --out ./deduped --dry-run=false
```

## Installation (binary)

This skill expects the `dedup` binary to exist in PATH.

From this repository (local build), install via:

```bash
make build
bash "{baseDir}/scripts/install-dedup.sh" --local "{baseDir}/../../bin/dedup"
```

Or install from a GitHub Release (recommended for user machines):

```bash
bash "{baseDir}/scripts/install-dedup.sh" --repo your-org/Deduplicator --tag v1.0.0
```

## Agent instructions

When the user asks to deduplicate files:

1. Confirm which action they want:
   - `report` (safe, no filesystem changes)
   - `copy-unique` (writes output dir)
   - `move-duplicates` (moves duplicates to quarantine)
   - `delete-duplicates` (irreversible; requires `--yes` and `--dry-run=false`)
2. Prefer starting with `dedup report <root>` in **dry-run** to show the plan.
3. Always use explicit roots (e.g. `.`) and explicit `--out` / `--dest`.
4. For destructive actions, ensure the user understands:
   - default is `--dry-run=true`
   - `delete-duplicates` requires `--dry-run=false --yes`

