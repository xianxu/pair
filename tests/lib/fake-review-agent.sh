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
set -euo pipefail
tag="${1:?usage: fake-review-agent.sh <tag> [file]}"
file="${2:-doc.md}"
dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
mkdir -p "$dir"
handoff="$dir/review-handoff-$tag.json"
landed="$dir/review-landed-$tag.json"
docflow="${DOCFLOW_BIN:-docflow}"

# (1) branch + (2) human round (the nvim saved the incoming edits).
"$docflow" start "$file" 2>/dev/null || true   # tolerate "branch exists" on re-run
"$docflow" round --side human -m incoming || true

# (3) propose records — the handoff the nvim watches.
cat > "$handoff.tmp" <<'JSON'
[{"old":"foo","occurrence":1,"new":"FOO","explain":"caps foo"},{"old":"baz","occurrence":1,"new":"BAZ","explain":"caps baz"}]
JSON
mv "$handoff.tmp" "$handoff"

# (4) wait for the nvim to apply + save + write the landed-artifact, then commit
# the agent round verbatim from it (this is what keeps the no-intelligence fake
# faithful — it commits exactly what landed, not its own proposal).
for _ in $(seq 1 200); do [ -f "$landed" ] && break; sleep 0.05; done
[ -f "$landed" ] || { echo "fake-review-agent: no landed-artifact at $landed" >&2; exit 1; }
summary="$(jq -r '.summary' "$landed")"
body="$(jq -r '.body' "$landed")"
"$docflow" round --side agent -m "$summary" --body "$body"
rm -f "$landed"
