#!/usr/bin/env bash
set -euo pipefail
repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/pair-test-bootstrap-check.XXXXXX") || exit 1
trap 'find "${scratch:?}" -type d -exec chmod u+rwx {} +; rm -rf "${scratch:?}"' EXIT
mkdir -p "$scratch/caller/readonly"
chmod 555 "$scratch/caller/readonly"
printf 'caller-owned\n' > "$scratch/caller/capture"
fixture="$scratch/fixture.sh"
printf '#!/usr/bin/env bash\n. %q\n' "$repo/tests/lib/pair-test-bootstrap.sh" > "$fixture"
cat >> "$fixture" <<'CHILD'
set -euo pipefail
[ -z "${PAIR_SCROLLBACK_EVENTS_PATH+x}" ]
[ -z "${COUCH_THREAD_SCOPE+x}" ]
[ -z "${ZELLIJ_SESSION_NAME+x}" ]
[ -d "$HOME" ]
[ -f "$0" ]
. /dev/null
[ -f "$0" ]
printf '%s\n' "$HOME" > "$2"
# Go module caches create read-only trees inside the disposable HOME.
mkdir -p "$HOME/go/pkg/mod/fixture"
printf 'module fixture\n' > "$HOME/go/pkg/mod/fixture/go.mod"
chmod 444 "$HOME/go/pkg/mod/fixture/go.mod"
chmod 555 "$HOME/go/pkg/mod/fixture"
ln -s "$4" "$HOME/foreign"
# A public marker from an earlier implementation is ordinary user data now;
# it cannot suppress the empty-environment entry boundary.
[ "$1" = --pair-test-isolated-child ]
[ "$3" = 'argument with spaces' ]
exit 23
CHILD
status=0
# This cleanup-only fixture has no Go work; avoid Go telemetry/config writes
# in the synthetic caller HOME while the cache contract tests cover resolution.
GOCACHE=off GOMODCACHE="$scratch/modules" GOTMPDIR= TMPDIR="$scratch" HOME="$scratch/caller" PAIR_SCROLLBACK_EVENTS_PATH="$scratch/caller/capture" COUCH_THREAD_SCOPE=foreign ZELLIJ_SESSION_NAME=foreign \
  bash "$fixture" --pair-test-isolated-child "$scratch/home" 'argument with spaces' "$scratch/caller/readonly" || status=$?
[ "$status" = 23 ]
child_home=$(cat "$scratch/home")
[ ! -e "$child_home" ] # outer bootstrap cleans even on child failure
[ "$(cat "$scratch/caller/capture")" = caller-owned ]
[ "$(find "$scratch/caller" -mindepth 1 -maxdepth 1 | wc -l | tr -d ' ')" = 2 ]
[ ! -w "$scratch/caller/readonly" ] # cleanup never follows fixture symlinks
printf 'pair-test-bootstrap cleanup, argv, exit status and caller isolation PASS\n'
