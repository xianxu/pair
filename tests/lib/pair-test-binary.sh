#!/usr/bin/env bash
# The only builder of full Pair test executables. Private aliases retain argv[0]
# and os.Executable directory behavior without another link by the Go toolchain.
set -euo pipefail
[ "$#" = 1 ] || { echo 'usage: pair-test-binary.sh OUTPUT' >&2; exit 2; }
if [ -n "${PAIR_TEST_BINARY:-}" ]; then
    exec python3 - "$PAIR_TEST_BINARY" "$1" <<'PY'
import errno, os, sys
source, destination = sys.argv[1:]
if not os.path.isabs(source) or not os.path.isfile(source) or not os.access(source, os.X_OK):
    sys.exit('PAIR_TEST_BINARY must be an absolute executable file: ' + source)
if source == destination:
    sys.exit(0)
try:
    os.link(source, destination)
except OSError as error:
    if error.errno != errno.EXDEV:
        raise
    # #749: keep executing the shared image across volumes too. A fresh byte
    # copy triggers another macOS security assessment before the CLI can run.
    # Symlink creation, like link, refuses an existing destination.
    os.symlink(source, destination)
PY
fi
# Resolve relative outputs before selecting the checkout, independently of cwd.
output="$1"
case "$output" in /*) ;; *) output="$PWD/$output" ;; esac
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
# exec retains the caller's CommandContext cancellation semantics.
exec go build -o "$output" ./cmd/pair-go
