#!/usr/bin/env bash
set -euo pipefail
repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/pair-test-bootstrap-check.XXXXXX") || exit 1
trap 'rm -rf "${scratch:?}"' EXIT
mkdir "$scratch/caller"
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
# A public marker from an earlier implementation is ordinary user data now;
# it cannot suppress the empty-environment entry boundary.
[ "$1" = --pair-test-isolated-child ]
[ "$3" = 'argument with spaces' ]
exit 23
CHILD
status=0
HOME="$scratch/caller" PAIR_SCROLLBACK_EVENTS_PATH="$scratch/caller/capture" COUCH_THREAD_SCOPE=foreign ZELLIJ_SESSION_NAME=foreign \
  bash "$fixture" --pair-test-isolated-child "$scratch/home" 'argument with spaces' || status=$?
[ "$status" = 23 ]
child_home=$(cat "$scratch/home")
[ ! -e "$child_home" ] # outer bootstrap cleans even on child failure
[ "$(cat "$scratch/caller/capture")" = caller-owned ]
[ "$(find "$scratch/caller" -mindepth 1 -maxdepth 1 | wc -l | tr -d ' ')" = 1 ]
printf 'pair-test-bootstrap cleanup, argv, exit status and caller isolation PASS\n'
