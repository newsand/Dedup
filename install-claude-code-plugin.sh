#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
install-claude-code-plugin.sh — install the "dedup" Claude Code plugin (local)

This installs the plugin from this repo into a local Claude Code plugins folder.
It only places files/symlinks; it does not require Claude Code to be installed.

Default destination:
  ~/.claude/plugins/local/dedup

Usage:
  ./install-claude-code-plugin.sh
  ./install-claude-code-plugin.sh --copy
  ./install-claude-code-plugin.sh --dir /custom/plugins

Options:
  --dir <path>   Destination base directory. Plugin will be placed at <path>/dedup
  --copy         Copy files instead of symlinking
  --dry-run      Print actions without changing anything
  -h, --help     Show help
EOF
}

die() { echo "error: $*" >&2; exit 1; }
need_arg() { [[ -n "${2:-}" ]] || die "missing value for $1"; }

DRY_RUN=0
COPY=0
DEST_BASE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; shift ;;
    --copy) COPY=1; shift ;;
    --dir) need_arg "$1" "${2:-}"; DEST_BASE="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown arg: $1" ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="${SCRIPT_DIR}/claude-code-plugin/dedup"
[[ -f "${SRC}/.claude-plugin/plugin.json" ]] || die "missing plugin source at ${SRC}"

if [[ -z "$DEST_BASE" ]]; then
  DEST_BASE="${HOME}/.claude/plugins/local"
fi

DEST="${DEST_BASE%/}/dedup"

if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "would install Claude Code plugin:"
  echo "  from: $SRC"
  echo "  to:   $DEST"
  echo "  mode: $([[ $COPY -eq 1 ]] && echo copy || echo symlink)"
  exit 0
fi

mkdir -p "$DEST_BASE"
rm -rf "$DEST"

if [[ "$COPY" -eq 1 ]]; then
  cp -R "$SRC" "$DEST"
else
  ln -s "$SRC" "$DEST"
fi

echo "installed Claude Code plugin: $DEST"
echo "next:"
echo "  - restart Claude Code (or reload plugins) if needed"

