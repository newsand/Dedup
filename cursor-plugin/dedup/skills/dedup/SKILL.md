---
name: dedup
description: Use the `dedup` CLI to find and handle byte-exact duplicates (BLAKE3).
---

## Quick usage

- Report duplicates in the current directory:

```bash
dedup report .
```

- Copy uniques + canonicals into a flattened output directory:

```bash
dedup copy-unique . --out ./deduped --dry-run=false
```

## Notes

- `dedup` must be installed and available in `PATH`.
- Start with `report` first (safe), then switch to a destructive mode only after reviewing the plan.

