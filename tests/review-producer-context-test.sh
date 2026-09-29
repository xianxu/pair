#!/usr/bin/env bash
# Real Git and producer process: contextual refusals preserve pending artifacts.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-producer-context.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
cat > "$RT/docflow" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$EFFECT_LOG"
case "$1" in
  start) git checkout -qb review/doc 2>/dev/null || true ;;
  round) git add -- doc.md; git commit -q --allow-empty -m "review(doc): $3 r1" ;;
  ship) git commit -q --allow-empty -m shipped ;;
esac
SH
chmod +x "$RT/docflow"
setup() {
  CASE="$RT/$1"; mkdir -p "$CASE/repo"
  cd "$CASE/repo"
  git init -q; git config user.name Tester; git config user.email test@example.com
  printf 'foo bar\nbaz qux\n' > doc.md
  git add doc.md; git commit -qm init
  git checkout -qb review/doc
  git branch review/other
  export EFFECT_LOG="$CASE/effects" DOCFLOW_BIN="$RT/docflow" XDG_DATA_HOME="$CASE/data"
  export PAIR_REVIEW_OPEN_PATH="$CASE/open" PAIR_REVIEW_HANDOFF_PATH="$CASE/handoff" PAIR_REVIEW_LANDED_PATH="$CASE/landed"
  export PAIR_REVIEW_REQUEST_CONTEXT
  PAIR_REVIEW_REQUEST_CONTEXT="$(jq -cn --arg repo "$(pwd -P)" '{repo:$repo,branch:"review/doc",file:"doc.md",activation:"token-a"}')"
  printf '123\npane\n%s\n' "$(jq -cn --argjson context "$PAIR_REVIEW_REQUEST_CONTEXT" '{version:1,context:$context}')" > "$PAIR_REVIEW_OPEN_PATH"
  jq -cn --argjson context "$PAIR_REVIEW_REQUEST_CONTEXT" '{context:$context,summary:"applied",body:"landed body",applied:2}' > "$PAIR_REVIEW_LANDED_PATH"
  unset PAIR_REVIEW_FAKE_PAUSE_PATH PAIR_REVIEW_FAKE_PAUSE_STAGE PAIR_REVIEW_FAKE_SHIP
}
refused() {
  if "$ROOT/tests/lib/fake-review-agent.sh" doc doc.md > "$CASE/output" 2>&1; then
    echo "FAIL producer accepted $1" >&2; exit 1
  fi
  test ! -e "$EFFECT_LOG" || { echo "FAIL Git effect on $1" >&2; exit 1; }
  test -f "$PAIR_REVIEW_LANDED_PATH"
}
setup branch-mismatch
git checkout -q review/other
refused branch-mismatch
setup stale-activation
PAIR_REVIEW_REQUEST_CONTEXT="$(jq -c '.activation="old-token"' <<< "$PAIR_REVIEW_REQUEST_CONTEXT")"
refused stale-activation
setup unsafe-file
PAIR_REVIEW_REQUEST_CONTEXT="$(jq -c '.file="../outside"' <<< "$PAIR_REVIEW_REQUEST_CONTEXT")"
refused unsafe-file
setup symlink-file
rm doc.md
ln -s ../outside doc.md
printf 'outside' > ../outside
git add doc.md; git commit -qm symlink
refused symlink-file
setup valid-ship
export PAIR_REVIEW_FAKE_SHIP=1
"$ROOT/tests/lib/fake-review-agent.sh" doc doc.md > "$CASE/output" 2>&1
test "$(git log -1 --format=%s)" = shipped
setup valid
"$ROOT/tests/lib/fake-review-agent.sh" doc doc.md > "$CASE/output" 2>&1
jq -e --argjson context "$PAIR_REVIEW_REQUEST_CONTEXT" '.context == $context and (.records | length == 2)' "$PAIR_REVIEW_HANDOFF_PATH" >/dev/null
test "$(git rev-list --count HEAD)" = 3
test ! -e "$PAIR_REVIEW_LANDED_PATH"
# Each pause occurs immediately before the corresponding effect's fresh guard.
for stage in human proposal agent ship; do
  setup "paused-$stage"
  export PAIR_REVIEW_FAKE_PAUSE_PATH="$CASE/pause" PAIR_REVIEW_FAKE_PAUSE_STAGE="$stage"
  if [ "$stage" = ship ]; then export PAIR_REVIEW_FAKE_SHIP=1; fi
  "$ROOT/tests/lib/fake-review-agent.sh" doc doc.md > "$CASE/output" 2>&1 &
  producer=$!
  for _ in $(seq 1 300); do
    [ -f "$CASE/pause.ready" ] && break
    if ! kill -0 "$producer" 2>/dev/null; then cat "$CASE/output"; echo 'FAIL missing pause'; exit 1; fi
    sleep 0.01
  done
  test -f "$CASE/pause.ready"
  git checkout -q review/other
  before="$(git rev-parse HEAD)"
  touch "$CASE/pause.release"
  if wait "$producer"; then echo "FAIL accepted branch switch before $stage"; exit 1; fi
  test "$(git rev-parse HEAD)" = "$before"
  test -f "$PAIR_REVIEW_LANDED_PATH"
  if [ "$stage" = agent ]; then test -f "$PAIR_REVIEW_HANDOFF_PATH"; fi
  ! git log --format=%s | grep -qE 'agent r1|shipped'
done
setup wrong-landed
jq '.context.activation="other"' "$PAIR_REVIEW_LANDED_PATH" > "$CASE/wrong"
mv "$CASE/wrong" "$PAIR_REVIEW_LANDED_PATH"
if "$ROOT/tests/lib/fake-review-agent.sh" doc doc.md > "$CASE/output" 2>&1; then echo 'FAIL wrong landed accepted'; exit 1; fi
test -f "$PAIR_REVIEW_LANDED_PATH"
! git log --format=%s | grep -q 'agent r1'
printf 'review producer context tests passed\n'
