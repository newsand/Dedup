#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
install.sh — install dedup on Linux

Installs `dedup` into PREFIX/bin (default: /usr/local/bin).

Sources:
  1) Local binary:
       ./install.sh --local ./bin/dedup

  2) GitHub release tarball:
       ./install.sh --repo your-org/Deduplicator --tag v1.0.0
     or
       ./install.sh --repo your-org/Deduplicator --version 1.0.0

Options:
  --local <path>      Install from an existing local binary (dedup)
  --repo <org/name>   GitHub repo (e.g. your-org/Deduplicator)
  --tag <vX.Y.Z>      Release tag (e.g. v1.0.0)
  --version <X.Y.Z>   Release version (e.g. 1.0.0). Implies tag=v<version>
  --prefix <dir>      Install prefix (default: /usr/local)
  --bin-dir <dir>     Install directory (overrides --prefix/bin)
  --dry-run           Print actions without changing anything
  -h, --help          Show help

Notes:
  - This script is Linux-only.
  - If installing to a system directory, run with sudo:
      sudo ./install.sh ...
EOF
}

die() {
  echo "error: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing dependency: $1"
}

is_linux() {
  [[ "$(uname -s | tr '[:upper:]' '[:lower:]')" == "linux" ]]
}

arch_go() {
  local m
  m="$(uname -m)"
  case "$m" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) die "unsupported architecture: $m (supported: amd64, arm64)" ;;
  esac
}

DRY_RUN=0
PREFIX="/usr/local"
BIN_DIR=""
LOCAL_BIN=""
REPO=""
TAG=""
VERSION=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; shift ;;
    --prefix) PREFIX="${2:-}"; shift 2 ;;
    --bin-dir) BIN_DIR="${2:-}"; shift 2 ;;
    --local) LOCAL_BIN="${2:-}"; shift 2 ;;
    --repo) REPO="${2:-}"; shift 2 ;;
    --tag) TAG="${2:-}"; shift 2 ;;
    --version) VERSION="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown argument: $1 (use --help)" ;;
  esac
done

is_linux || die "this installer is for Linux only"

if [[ -n "$VERSION" && -z "$TAG" ]]; then
  TAG="v${VERSION}"
fi

if [[ -z "$BIN_DIR" ]]; then
  BIN_DIR="${PREFIX%/}/bin"
fi

if [[ -n "$LOCAL_BIN" ]]; then
  [[ -f "$LOCAL_BIN" ]] || die "--local path does not exist: $LOCAL_BIN"
  [[ -x "$LOCAL_BIN" ]] || die "--local binary is not executable: $LOCAL_BIN"
  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "would install: $LOCAL_BIN -> ${BIN_DIR}/dedup"
    exit 0
  fi
  mkdir -p "$BIN_DIR"
  install -m 0755 "$LOCAL_BIN" "${BIN_DIR}/dedup"
  echo "installed: ${BIN_DIR}/dedup"
  exit 0
fi

[[ -n "$REPO" ]] || die "missing --repo (or use --local)"
[[ -n "$TAG" ]] || die "missing --tag/--version (or use --local)"

need_cmd curl
need_cmd tar
need_cmd install

ARCH="$(arch_go)"

# Matches .goreleaser.yaml `archives.name_template`:
#   dedup_<version>_linux_<arch>.tar.gz
# Where <version> is X.Y.Z (without the leading 'v').
VER="${TAG#v}"
ASSET="dedup_${VER}_linux_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"

if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "would download: $URL"
  echo "would install to: ${BIN_DIR}/dedup"
  exit 0
fi

TMPDIR="$(mktemp -d)"
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

echo "downloading: $URL"
curl -fsSL -o "${TMPDIR}/${ASSET}" "$URL"

tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"
[[ -f "${TMPDIR}/dedup" ]] || die "archive does not contain 'dedup' binary"

mkdir -p "$BIN_DIR"
install -m 0755 "${TMPDIR}/dedup" "${BIN_DIR}/dedup"

echo "installed: ${BIN_DIR}/dedup"
echo "verify:"
echo "  ${BIN_DIR}/dedup version"

