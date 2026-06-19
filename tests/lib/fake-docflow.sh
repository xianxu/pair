#!/usr/bin/env bash
# tests/lib/fake-docflow.sh — hermetic stand-in for ariadne's scripts/docflow.sh
# used by the review tests (#66 M1). Makes REAL git commits matching docflow's
# shape (subject, --author for agent rounds, body) so tests assert author/
# subject/body without an ariadne checkout, and appends its argv (space-joined)
# to $DOCFLOW_ARGLOG for arg-forwarding assertions. Faithful to the real verbs
# we use: start <file>, round --side h|a [-m s] [--body b], status, ship.
set -euo pipefail

AGENT_AUTHOR="${DOCFLOW_AGENT_AUTHOR:-Claude <noreply@anthropic.com>}"
[ -n "${DOCFLOW_ARGLOG:-}" ] && printf '%s\n' "$*" >> "$DOCFLOW_ARGLOG"

slugify() {
  local s="${1##*/}"; s="${s%.*}"
  s="$(printf '%s' "$s" | tr '[:upper:]' '[:lower:]' | tr -c '[:alnum:]' '-')"
  printf '%s' "$s" | sed -E 's/-+/-/g; s/^-//; s/-$//'
}

verb="${1:-}"; shift || true
case "$verb" in
  start)
    f="${1:?start needs a file}"
    [ -f "$f" ] || { echo "fake-docflow: $f not found" >&2; exit 1; }
    git checkout -q -b "review/$(slugify "$f")"
    # record the in-scope file — the real docflow stages in-scope-only, never -A
    printf '%s\n' "$f" > "$(git rev-parse --git-dir)/fake-docflow-inscope"
    ;;
  round)
    side=""; summary=""; body=""
    while [ $# -gt 0 ]; do
      case "$1" in
        --side) side="$2"; shift 2;;
        -m|--summary) summary="$2"; shift 2;;
        --body) body="$2"; shift 2;;
        --) shift; break;;
        *) shift;;
      esac
    done
    [ "$side" = human ] || [ "$side" = agent ] || { echo "round needs --side human|agent" >&2; exit 1; }
    cur="$(git rev-parse --abbrev-ref HEAD)"
    slug="${cur#review/}"
    # stage in-scope files only (matches the real docflow); -A only as fallback
    inscope="$(git rev-parse --git-dir)/fake-docflow-inscope"
    if [ -s "$inscope" ]; then
      while IFS= read -r insf; do [ -n "$insf" ] && git add -- "$insf"; done < "$inscope"
    else
      git add -A
    fi
    if git diff --cached --quiet; then
      echo "fake-docflow: no changes for $side round — skipping" >&2
      exit 0
    fi
    n=$(( $(git log --oneline --grep="review($slug): $side r" 2>/dev/null | wc -l | tr -d ' ') + 1 ))
    [ -n "$summary" ] || summary="$side round $n"
    subject="review($slug): $side r$n — $summary"
    args=(commit -q -m "$subject")
    [ -n "$body" ] && args+=(-m "$body")
    if [ "$side" = agent ]; then
      args+=(-m "Co-Authored-By: $AGENT_AUTHOR")
      git "${args[@]}" --author="$AGENT_AUTHOR"
    else
      git "${args[@]}"
    fi
    ;;
  status) git rev-parse --abbrev-ref HEAD ;;
  ship)   : ;;  # no-op for tests
  *) echo "fake-docflow: unknown verb '$verb'" >&2; exit 1 ;;
esac
