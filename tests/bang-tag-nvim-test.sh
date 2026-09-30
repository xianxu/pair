#!/usr/bin/env bash
# #337: a `!` draft line reaches the agent without the `!` and, inside couch,
# publishes the thread description through the real `couch` command line.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-bang-tag.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

mkdir -p "$RT/bin"
mkdir -p "$RT/empty-bin"
cat >"$RT/bin/couch" <<'EOF'
#!/bin/sh
printf 'scope=%s tag=%s %s\n' "${COUCH_THREAD_SCOPE:-}" "${COUCH_THREAD_TAG:-}" "$*" >>"$PAIR_TEST_COUCH_CALLS"
case "$PAIR_TEST_BANG_CASE" in
  nonzero|describe-nonzero) exit 17 ;;
  slow)
    # Only the test driver can release us, after submit returns. A bounded
    # fallback makes a synchronous-publisher regression fail without leaking.
    tries=0
    while [ ! -f "$PAIR_TEST_COUCH_CALLS.release" ]; do
      tries=$((tries + 1))
      if [ "$tries" -ge 100 ]; then
        : >"$PAIR_TEST_COUCH_CALLS.timed-out"
        exit 18
      fi
      /bin/sleep 0.05
    done
    : >"$PAIR_TEST_COUCH_CALLS.completed"
    ;;
esac
EOF
chmod +x "$RT/bin/couch"

for bang_case in couch standalone missing nonzero slow retry describe describe-standalone describe-nonzero describe-missing; do
  if [ "$bang_case" = standalone ] || [ "$bang_case" = describe-standalone ]; then scope='' tag=''; else scope=S1 tag=T1; fi
  run_headless --timeout 30 -- \
    env PAIR_DATA_DIR='' PAIR_TAG='' PAIR_SCOPE_KEY='' PAIR_RETENTION_PROTOCOL='' \
    PATH="$RT/bin:$PATH" COUCH_THREAD_SCOPE="$scope" COUCH_THREAD_TAG="$tag" \
    PAIR_TEST_BANG_CASE="$bang_case" PAIR_TEST_COUCH_CALLS="$RT/$bang_case.calls" \
    PAIR_TEST_EMPTY_PATH="$RT/empty-bin" \
    PAIR_LOG_PATH="$RT/$bang_case.md" \
    nvim --headless -u "$ROOT/nvim/init.lua" \
    -l "$ROOT/nvim/bang_tag_integration_test.lua"
  printf '  ok   bang tag %s\n' "$bang_case"
done

echo 'bang-tag-nvim-test: all passed'
