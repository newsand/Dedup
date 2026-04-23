# Deduplicator

Exact file deduplicator (BLAKE3) — CLI in Go.

Duplicates are detected **only** by byte-identical content. No near-duplicate,
no content transformation, no OCR. By default every destructive command runs
in dry-run mode.

## Requirements

- Go **1.25+** (see `go.mod`)

## Run (from source)

No full `make` needed. You can run directly with `go run`:

```bash
go run ./cmd/dedup --help
go run ./cmd/dedup report ./some/dir
```

Or build a local binary:

```bash
go build -o bin/dedup ./cmd/dedup
./bin/dedup report ./some/dir
```

`make build` is just a convenience wrapper that produces `./bin/dedup`.

## Quick start (commands)

```bash
dedup report ./photos
dedup copy-unique ./photos --out ./deduped --dry-run=false
dedup move-duplicates ./photos --dest ./quarantine --dry-run=false
dedup delete-duplicates ./photos --yes --dry-run=false
```

## How to actually generate a “deduplicated folder”

`dedup` has **two kinds of commands**:

- **Report-only** (no filesystem changes): `scan`, `report`
- **Action commands** (create/move/delete files): `copy-unique`, `move-duplicates`, `delete-duplicates`

If what you want is a new directory containing the deduplicated result, use **`copy-unique`**:

```bash
# 1) Preview exactly what would be copied (default is dry-run)
dedup copy-unique ./photos --out ./deduped

# 2) Actually create/populate the output directory
dedup copy-unique ./photos --out ./deduped --dry-run=false
```

What you get in `--out`:

- **Uniq files** (files that have no duplicates)
- **Canonicals** (one chosen representative for each duplicate group)
- All written into a **single flattened directory** with normalized names (snake_case) + deterministic suffixes on collisions

If instead you want to keep the originals “as-is” and just move the extra copies away, use **`move-duplicates`** (quarantine):

```bash
dedup move-duplicates ./photos --dest ./quarantine
dedup move-duplicates ./photos --dest ./quarantine --dry-run=false
```

## Usage

All commands take one or more **roots**. To use the current directory as root, pass `.`:

```bash
dedup report .
```

### `dedup scan <roots...>`

Alias of `report` (dry-run, human-readable output).

### `dedup report <roots...>`

Scans roots and prints a duplicate report. Does **not** modify files.

### `dedup copy-unique <roots...> --out <dir>`

Copies **uniques + canonicals** to a single flattened output directory.

- **Safety**: defaults to `--dry-run=true` (set `--dry-run=false` to copy)
- **Naming**: uses snake_case + deterministic collision suffixes; `--suppressname` drops the original filename

### `dedup move-duplicates <roots...> [--dest <dir>]`

Moves **duplicates only** to a quarantine directory, keeping the canonical in place.

- **Safety**: defaults to `--dry-run=true` (set `--dry-run=false` to move)

### `dedup delete-duplicates <roots...> --yes`

Deletes **duplicates only**, keeping the canonical in place.

- **Safety**: defaults to `--dry-run=true`
- **Requires**: `--dry-run=false --yes` to actually delete

### `dedup version`

Prints version/build information.

### `dedup completion <shell>`

Generates shell autocompletion scripts for `bash|zsh|fish|powershell`.

## Bundle / release artifacts

This repo ships goreleaser config (`.goreleaser.yaml`) that produces:

- `linux/amd64`, `linux/arm64`: `.tar.gz` archives and `.deb` packages
- `windows/amd64`: `.zip` archive (`dedup.exe`)

Locally, you can build the same binary that goes into the bundles with:

```bash
make build
```

## Install (Linux)

Choose one of the options below.

### Install using `install.sh` (recommended)

From a **local build**:

```bash
make build
chmod +x install.sh
sudo ./install.sh --local ./bin/dedup
dedup version
```

Without `sudo` (installs for the current user):

```bash
make build
chmod +x install.sh
./install.sh --local ./bin/dedup --bin-dir "$HOME/.local/bin"
export PATH="$HOME/.local/bin:$PATH"
dedup version
```

From a **GitHub Release** (downloads the correct `linux/amd64` or `linux/arm64` bundle):

```bash
chmod +x install.sh
sudo ./install.sh --repo your-org/Deduplicator --tag v1.0.0
```

### Install from a `.tar.gz` bundle

```bash
tar -xzf dedup_<version>_linux_amd64.tar.gz
sudo install -m 0755 dedup /usr/local/bin/dedup
dedup version
```

### Install from a `.deb` package

```bash
sudo dpkg -i dedup_<version>_linux_amd64.deb
dedup version
```

### Install from source

```bash
git clone <this-repo>
cd Deduplicator
make build
sudo install -m 0755 bin/dedup /usr/local/bin/dedup
```

## Windows build status

- **CI**: the workflow runs tests on **`windows-latest`** (`.github/workflows/ci.yml`).
- **Release bundle**: goreleaser is configured to build **`windows/amd64`** zip artifacts (`.goreleaser.yaml`).

No MSI/installer is provided; Windows distribution is currently a zip containing `dedup.exe`.

## OpenClaw Skill (install)

This repo includes an OpenClaw skill folder at `openclaw-skill/dedup/`.

Install it to your default OpenClaw skills directory:

```bash
chmod +x install-openclaw-skill.sh
./install-openclaw-skill.sh
```

Or install it into the current workspace as `./skills/dedup`:

```bash
./install-openclaw-skill.sh --workspace
```

Then enable it in `~/.openclaw/openclaw.json` (JSON5):

```json5
{
  skills: {
    entries: {
      dedup: { enabled: true },
    },
  },
}
```

The skill expects the `dedup` binary to be available in `PATH`. See `openclaw-skill/dedup/SKILL.md` for install options.


## License

MIT — see [LICENSE](LICENSE).
