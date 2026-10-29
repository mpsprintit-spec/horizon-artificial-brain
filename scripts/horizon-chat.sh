#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
if ! command -v go >/dev/null 2>&1; then
  printf 'Go belum terpasang. Di Termux: pkg install golang git\n' >&2
  exit 1
fi
cd "$repo_dir"
mkdir -p bin
go build -o bin/horizon-chat ./cmd/horizon-chat
exec bin/horizon-chat "$@"
