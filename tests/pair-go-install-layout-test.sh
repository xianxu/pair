#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_home="$(mktemp -d "${TMPDIR:-/tmp}/pair-go-install-layout.XXXXXX")"
trap 'rm -rf "$tmp_home"' EXIT
gomodcache="$(go env GOMODCACHE)"
gocache="$(go env GOCACHE)"

HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null

install_bin="$tmp_home/.local/bin"
test -x "$install_bin/pair-go"
test -L "$install_bin/pair"
test -L "$install_bin/pair-dev"

out="$("$install_bin/pair-go" launch --help)"
case "$out" in
    pair\ —*) ;;
    *)
        printf 'pair-go launch --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
        exit 1
        ;;
esac

printf 'pair-go install layout test passed\n'
