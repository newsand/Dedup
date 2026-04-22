# Fixtures

Tiny, versioned inputs used by the test suite. Each subtree maps to the
cases described in [`Docs/14-testing-strategy.md`](../../Docs/14-testing-strategy.md):

- `images/exact/`                          — two byte-identical "PNG"s.
- `images/same_name_different_content/`    — same basename, different content.
- `images/same_size_different_hash/`       — 16 bytes each, different content.
- `pdfs/exact/`                            — two byte-identical "PDF"s.
- `pdfs/same_name_different_content/`      — same basename, different content.
- `naming/suppress_name/`                  — feeds the `--suppressname` test.
- `naming/collisions/`                     — feeds the flatten-collision test.
- `canonical/oldest_wins/`                 — canonical-by-mtime test.
- `canonical/lexical_tiebreak/`            — canonical-by-lexical test.
- `pipeline/mixed_tree/`                   — end-to-end smoke test:
  1 image dup + 1 PDF dup + 1 unique.

All files are under 256 bytes and use the minimal magic-byte prefix needed by
the file-type detector. They are **not** valid PNG/PDF — the tool only cares
about byte identity.
