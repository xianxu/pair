#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../lib/pair-test-bootstrap.sh"
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
d="$(mktemp -d "${TMPDIR:-/tmp}/pair-binary-contract.XXXXXX")" || exit 1
trap 'rm -rf "${d:?}"' EXIT
printf '#!/bin/sh\nexit 0\n' > "$d/source"
chmod +x "$d/source"
PAIR_TEST_BINARY="$d/source" bash "$root/tests/lib/pair-test-binary.sh" "$d/alias"
python3 - "$d/source" "$d/alias" <<'PY'
import os,sys
assert os.path.samefile(*sys.argv[1:]), 'private placement must preserve inode'
PY
# #749: a cross-volume fixture must also execute the shared image. Copying a
# fresh Mach-O made AMFI assessment consume copy-on-select's entire deadline.
mkdir "$d/exdev-bin"
cat > "$d/exdev-bin/python3" <<'SH'
#!/bin/sh
shift # The helper's Python source arrives on stdin (python3 - SOURCE DEST).
exec "$PAIR_TEST_REAL_PYTHON" -c '
import errno, os, sys
def cross_device(*args, **kwargs):
    raise OSError(errno.EXDEV, "forced cross-device link")
os.link = cross_device
exec(compile(sys.stdin.read(), "pair-test-binary.sh", "exec"))
' "$@"
SH
chmod +x "$d/exdev-bin/python3"
PATH="$d/exdev-bin:$PATH" PAIR_TEST_REAL_PYTHON="$(command -v python3)" PAIR_TEST_BINARY="$d/source" \
  bash "$root/tests/lib/pair-test-binary.sh" "$d/cross-volume-alias"
python3 - "$d/source" "$d/cross-volume-alias" <<'PY'
import os,sys
assert os.path.samefile(*sys.argv[1:]), 'cross-volume placement must not copy the candidate'
assert os.path.islink(sys.argv[2]), 'cross-volume alias must point to the shared image'
PY
if PAIR_TEST_BINARY=relative bash "$root/tests/lib/pair-test-binary.sh" "$d/invalid"; then exit 1; fi
if PAIR_TEST_BINARY="$d/source" bash "$root/tests/lib/pair-test-binary.sh" "$d/alias"; then exit 1; fi
mkdir "$d/fake"
cat > "$d/fake/go" <<'GO'
#!/bin/sh
set -eu
printf 'build\n' >> "${CALLS:?}"
[ "$PWD" = "${ROOT_EXPECTED:?}" ]
[ "$1" = build ] && [ "$2" = -o ]
printf '#!/bin/sh\nexit 0\n' > "$3"
chmod +x "$3"
GO
cat > "$d/fake/make" <<'MAKE'
#!/bin/sh
exit 0
MAKE
chmod +x "$d/fake/"*
# Caller cwd must not choose the source tree or reinterpret relative outputs.
(cd "$root/tests"; PATH="$d/fake:$PATH" CALLS="$d/subcalls" ROOT_EXPECTED="$root" PAIR_TEST_BINARY= bash "$root/tests/lib/pair-test-binary.sh" "$d/from-subdir")
(cd "$d"; PATH="$d/fake:$PATH" CALLS="$d/subcalls" ROOT_EXPECTED="$root" PAIR_TEST_BINARY= bash "$root/tests/lib/pair-test-binary.sh" relative-output)
[ -x "$d/from-subdir" ] && [ -x "$d/relative-output" ]
PATH="$d/fake:$PATH" CALLS="$d/calls" ROOT_EXPECTED="$root" PAIR_TEST_BINARY= bash "$root/tests/lib/with-pair-test-binary.sh" sh -c 'test -x "$PAIR_TEST_BINARY"; printf "%s" "$PAIR_TEST_BINARY" > "$1"; exit 7' sh "$d/path" && exit 1 || code=$?
[ "$code" = 7 ]
[ "$(wc -l < "$d/calls" | tr -d ' ')" = 1 ]
[ ! -e "$(cat "$d/path")" ]
printf 'shared binary shell contracts passed\n'
