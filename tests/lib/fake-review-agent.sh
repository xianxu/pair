#!/usr/bin/env bash
# tests/lib/fake-review-agent.sh — process-level fake of the review agent
# (#66; M4a = fake-agent-v2). A real agent (ariadne #000121) recognizes
# review-mode, owns ALL git, and computes records via memory + a SKILL. This fake
# mirrors the PROTOCOL with fixed records so the loop test is deterministic AND
# faithful to "the agent owns git" (invariant #1):
#   1. create the review/<slug> branch (the nvim no longer does);
#   2. commit the human round (the nvim already SAVED the incoming edits);
#   3. propose records via the handoff (the nvim watches it);
#   4. wait for the nvim's landed-artifact and commit the agent round VERBATIM
#      from it (body == what actually landed — invariant #3).
# Records target 'foo'/'baz' — the doc fixture must contain them.
#
# Runs in the doc's repo (cwd), with DOCFLOW_BIN + XDG_DATA_HOME from the caller.
# Usage: fake-review-agent.sh <tag> [file]
# Scoped requests set PAIR_REVIEW_REQUEST_CONTEXT to the captured context JSON
# and PAIR_REVIEW_OPEN_PATH to the pane state. Fake pause hooks are test-only:
# PAIR_REVIEW_FAKE_PAUSE_PATH publishes .ready and waits for .release at
# PAIR_REVIEW_FAKE_PAUSE_STAGE (human, proposal, agent, ship; default agent).
# PAIR_REVIEW_FAKE_SHIP=1 exercises the ship effect instead of a review round.
set -euo pipefail
tag="${1:?usage: fake-review-agent.sh <tag> [file]}"
file="${2:-doc.md}"
dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
mkdir -p "$dir"
handoff="${PAIR_REVIEW_HANDOFF_PATH:-$dir/review-handoff-$tag.json}"
landed="${PAIR_REVIEW_LANDED_PATH:-$dir/review-landed-$tag.json}"
mkdir -p "$(dirname "$handoff")" "$(dirname "$landed")"
docflow="${DOCFLOW_BIN:-docflow}"

# Capture the request's identity once. Re-read pane/checkout state before every
# effect so a response to an old request cannot acquire a newer activation.
context="${PAIR_REVIEW_REQUEST_CONTEXT:-}"
refuse() { echo "fake-review-agent: context mismatch: $*; pending artifacts preserved" >&2; exit 1; }
pause_before() {
  local stage="$1" hook="${PAIR_REVIEW_FAKE_PAUSE_PATH:-}"
  [ -n "$hook" ] && [ "${PAIR_REVIEW_FAKE_PAUSE_STAGE:-agent}" = "$stage" ] || return 0
  : > "$hook.ready"
  for _ in $(seq 1 1000); do [ -f "$hook.release" ] && return 0; sleep 0.01; done
  refuse "pause timed out before $stage"
}
validate_context() {
  [ -n "$context" ] || return 0
  jq -e 'type == "object" and ([.repo,.branch,.file,.activation] | all(type == "string" and length > 0))' <<< "$context" >/dev/null || refuse 'invalid request'
  [ -n "${PAIR_REVIEW_OPEN_PATH:-}" ] && [ -f "$PAIR_REVIEW_OPEN_PATH" ] || refuse 'pane metadata missing'
  local metadata repo branch requested_file
  metadata="$(sed -n '3p' "$PAIR_REVIEW_OPEN_PATH")"
  jq -e --argjson expected "$context" '.context == $expected' <<< "$metadata" >/dev/null || refuse 'pane activation changed'
  repo="$(git rev-parse --show-toplevel)" || refuse 'not a repository'
  repo="$(cd "$repo" && pwd -P)"
  branch="$(git symbolic-ref --quiet --short HEAD)" || refuse 'detached checkout'
  jq -e --arg repo "$repo" --arg branch "$branch" '.repo == $repo and .branch == $branch' <<< "$context" >/dev/null || refuse 'checkout changed'
  requested_file="$(jq -r '.file' <<< "$context")"
  [ "$file" = "$requested_file" ] || refuse 'document differs from request'
  python3 - "$repo" "$requested_file" <<'PYSAFE' || refuse 'unsafe document path'
import pathlib, sys
root, name = pathlib.Path(sys.argv[1]), sys.argv[2]
p = pathlib.PurePosixPath(name)
if p.is_absolute() or any(part in ('', '.', '..') for part in name.split('/')) or '\x00' in name:
    sys.exit(1)
file = root
for part in p.parts:
    file = file / part
    if file.is_symlink():
        sys.exit(1)
if not file.is_file() or not file.resolve().is_relative_to(root):
    sys.exit(1)
PYSAFE
  git -C "$repo" ls-files --error-unmatch -- "$requested_file" >/dev/null 2>&1 || refuse 'document is not tracked'
}
if [ "${PAIR_REVIEW_FAKE_SHIP:-}" = 1 ]; then
  pause_before ship
  validate_context
  "$docflow" ship
  exit 0
fi

# (1) branch + (2) human round (the nvim saved the incoming edits).
if [ -z "$context" ]; then
  "$docflow" start "$file" 2>/dev/null || true # legacy first-open branch preparation
fi
pause_before human
validate_context
if [ -n "$context" ]; then "$docflow" round --side human -m incoming
else "$docflow" round --side human -m incoming || true; fi

# (3) propose records — the handoff the nvim watches.
pause_before proposal
validate_context
cat > "$handoff.tmp" <<'JSON'
[{"old":"foo","occurrence":1,"new":"FOO","explain":"caps foo"},{"old":"baz","occurrence":1,"new":"BAZ","explain":"caps baz"}]
JSON
if [ -n "$context" ]; then
  jq --argjson context "$context" '{context:$context,records:.}' "$handoff.tmp" > "$handoff.envelope.tmp"
  mv "$handoff.envelope.tmp" "$handoff.tmp"
fi
mv "$handoff.tmp" "$handoff"

# (4) wait for the nvim to apply + save + write the landed-artifact, then commit
# the agent round verbatim from it (this is what keeps the no-intelligence fake
# faithful — it commits exactly what landed, not its own proposal).
for _ in $(seq 1 200); do [ -f "$landed" ] && break; sleep 0.05; done
[ -f "$landed" ] || { echo "fake-review-agent: no landed-artifact at $landed" >&2; exit 1; }
pause_before agent
validate_context
if [ -n "$context" ]; then
  jq -e --argjson context "$context" '.context == $context' "$landed" >/dev/null || refuse 'landed activation differs'
fi
summary="$(jq -r '.summary' "$landed")"
body="$(jq -r '.body' "$landed")"
validate_context
"$docflow" round --side agent -m "$summary" --body "$body"
rm -f "$landed"
