---
name: dedup
description: Use the `dedup` CLI to find and handle byte-exact duplicates (BLAKE3).
---

## Quick usage

- Report duplicates in the current directory:

```bash
dedup report .
```

- Move duplicates away (dry-run first):

```bash
dedup move-duplicates . --out ./duplicates --dry-run
dedup move-duplicates . --out ./duplicates --dry-run=false
```

## Requirements

- `dedup` installed in `PATH` (use this repo's `install.sh` if you want).

