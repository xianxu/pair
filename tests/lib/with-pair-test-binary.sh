#!/usr/bin/env bash
# Own one fresh plain Pair for every consumer in this suite invocation.
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/pair-test-suite.XXXXXX")" || exit 1
trap 'rm -rf "${scratch:?}"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
cd "$repo"
make runtimebundle-generate
PAIR_TEST_BINARY= bash tests/lib/pair-test-binary.sh "$scratch/pair"
export PAIR_TEST_BINARY="$scratch/pair"
"$@"
