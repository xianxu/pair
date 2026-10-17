#!/usr/bin/env bash
# Publish a Go executable only after a complete successful build.
set -euo pipefail
output="${1:?usage: build-go-atomic.sh OUTPUT [go build arguments...]}"
shift
parent="$(dirname "$output")"
mkdir -p "$parent"
if [ -d "$output" ]; then
    printf 'build output is a directory: %s\n' "$output" >&2
    exit 1
fi
# The same parent guarantees rename stays on this filesystem. Each invocation
# owns its directory, including concurrent builds of the same executable.
staging="$(mktemp -d "$parent/.go-build.XXXXXX")" || exit 1
trap 'rm -rf "${staging:?}"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
go build -o "$staging/binary" "$@"
mv -f "$staging/binary" "$output"
