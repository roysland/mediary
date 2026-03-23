#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v gosec >/dev/null 2>&1; then
    echo "gosec is not installed or not on PATH" >&2
    exit 127
fi

args=(
    -exclude-generated
    ./...
)

if [[ $# -gt 0 ]]; then
    output_path="$1"
    mkdir -p "$(dirname "$output_path")"
    gosec "${args[@]}" |& tee "$output_path"
    exit ${PIPESTATUS[0]}
fi

exec gosec "${args[@]}"