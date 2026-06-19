#!/usr/bin/env bash
# tests/lib/fake-review-agent.sh — process-level fake of the review agent
# (#66 M1). Writes a deterministic records handoff (atomically) to the path
# review.handoff watches. A real agent would compute records via memory
# discovery + a SKILL.md (M4); this emits fixed ones so the loop test is
# deterministic. Records target 'foo'/'baz' — the doc fixture must contain them.
#
# Usage: fake-review-agent.sh <tag>
set -euo pipefail
tag="${1:?usage: fake-review-agent.sh <tag>}"
dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
mkdir -p "$dir"
p="$dir/review-handoff-$tag.json"
cat > "$p.tmp" <<'JSON'
[{"old":"foo","occurrence":1,"new":"FOO","explain":"caps foo"},{"old":"baz","occurrence":1,"new":"BAZ","explain":"caps baz"}]
JSON
mv "$p.tmp" "$p"
