#!/bin/sh
# Focused keying test for bin/pair-changelog-open (#63): the change-log base is
# keyed on the resolved session id (PAIR_SESSION_ID -> config -> none). No model
# or distiller runs -- $RAW is empty, so the orchestrator's distiller block is
# skipped (its `[ -s "$RAW" ]` guard); the opener only resolves the base, touches
# <base>.md, and opens the (fake) viewer on it. We assert the path nvim was handed.
set -eu

PAIR_HOME=$(cd "$(dirname "$0")/.." && pwd); export PAIR_HOME
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-changelog-key.XXXXXX"); trap 'rm -rf "$tmp"' EXIT
export PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t PAIR_AGENT=claude
mkdir -p "$PAIR_DATA_DIR"

# Fake nvim records the file path it was opened on (the last positional arg).
fakebin="$tmp/bin"; mkdir -p "$fakebin"
cat > "$fakebin/nvim" <<EOF
#!/bin/sh
for a in "\$@"; do :; done
printf '%s\n' "\$a" > "$tmp/nvim-arg"
EOF
chmod +x "$fakebin/nvim"; export PATH="$fakebin:$PATH"

fail=0
opened() { cat "$tmp/nvim-arg" 2>/dev/null; }
run() { rm -f "$tmp/nvim-arg"; "$PAIR_HOME/bin/pair-changelog-open"; }

A=aaaa1111-2222-3333-4444-555566667777
B=bbbb1111-2222-3333-4444-555566667777
C=cccc1111-2222-3333-4444-555566667777

# (a) PAIR_SESSION_ID set -> keyed base
PAIR_SESSION_ID="$A" run
case "$(opened)" in *"changelog-t-claude-$A.md") ;;
  *) echo "FAIL (a) env-keyed base: $(opened)"; fail=1 ;; esac

# (b) fresh session = different id -> different, empty file (does not see A's log)
printf 'old log\n' > "$PAIR_DATA_DIR/changelog-t-claude-$A.md"
PAIR_SESSION_ID="$B" run
case "$(opened)" in *"changelog-t-claude-$B.md") ;;
  *) echo "FAIL (b) fresh id base: $(opened)"; fail=1 ;; esac
[ -s "$PAIR_DATA_DIR/changelog-t-claude-$B.md" ] \
  && { echo "FAIL (b) fresh log not empty"; fail=1; }

# (c) resume = same id -> same file, prior content intact
PAIR_SESSION_ID="$A" run
grep -q 'old log' "$PAIR_DATA_DIR/changelog-t-claude-$A.md" \
  || { echo "FAIL (c) resume lost prior content"; fail=1; }

# (d) env unset -> fall back to config.session_id
unset PAIR_SESSION_ID
printf '{"agent":"claude","args":[],"session_id":"%s"}' "$C" \
  > "$PAIR_DATA_DIR/config-t-claude.json"
run
case "$(opened)" in *"changelog-t-claude-$C.md") ;;
  *) echo "FAIL (d) config-fallback base: $(opened)"; fail=1 ;; esac

# (e) no env, no config session_id -> legacy unsuffixed base (backward compat)
rm -f "$PAIR_DATA_DIR/config-t-claude.json"
run
case "$(opened)" in *"changelog-t-claude.md") ;;
  *) echo "FAIL (e) legacy base: $(opened)"; fail=1 ;; esac

if [ "$fail" = 0 ]; then
  echo "PASS changelog-session-key-test"
else
  exit 1
fi
