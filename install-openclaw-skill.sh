#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
install-openclaw-skill.sh — install the "dedup" OpenClaw Skill

Installs the skill folder into:
  - ~/.openclaw/skills/dedup  (default)
or
  - ./skills/dedup            (workspace-local)

Usage:
  ./install-openclaw-skill.sh
  ./install-openclaw-skill.sh --workspace
  ./install-openclaw-skill.sh --dir /custom/skills

Options:
  --workspace     Install into ./skills (relative to current directory)
  --dir <path>    Install into <path>/dedup
  --dry-run       Print actions without changing anything
  -h, --help      Show help

After install, enable it in ~/.openclaw/openclaw.json (JSON5), e.g.:
  skills: { entries: { dedup: { enabled: true } } }
EOF
}

die() { echo "error: $*" >&2; exit 1; }
need_arg() { [[ -n "${2:-}" ]] || die "missing value for $1"; }

DRY_RUN=0
MODE="home"
BASE_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; shift ;;
    --workspace) MODE="workspace"; shift ;;
    --dir) need_arg "$1" "${2:-}"; BASE_DIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown arg: $1" ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="${SCRIPT_DIR}/openclaw-skill/dedup"
[[ -f "${SRC}/SKILL.md" ]] || die "missing skill source at ${SRC}/SKILL.md"

if [[ -n "$BASE_DIR" ]]; then
  DEST_BASE="$BASE_DIR"
elif [[ "$MODE" == "workspace" ]]; then
  DEST_BASE="$(pwd)/skills"
else
  DEST_BASE="${HOME}/.openclaw/skills"
fi

DEST="${DEST_BASE%/}/dedup"

if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "would install skill:"
  echo "  from: $SRC"
  echo "  to:   $DEST"
  exit 0
fi

mkdir -p "$DEST_BASE"
rm -rf "$DEST"
cp -R "$SRC" "$DEST"

# Make helper scripts executable (optional, but convenient)
chmod +x "$DEST/scripts/"*.sh 2>/dev/null || true

echo "installed OpenClaw skill: $DEST"
echo "next:"
echo "  - enable it in ~/.openclaw/openclaw.json under skills.entries.dedup"
echo "  - ensure 'dedup' is installed in PATH (see $DEST/SKILL.md)"

