#!/usr/bin/env bash
set -euo pipefail

# Wrapper so OpenClaw skill docs can install dedup.
# It simply delegates to the repository's install.sh.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

exec bash "${REPO_ROOT}/install.sh" "$@"

