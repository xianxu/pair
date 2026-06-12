#!/bin/sh
# End-to-end smoke test for bin/pair-changelog-open (#53 M2).
#
# The orchestrator is now thin: it opens nvim immediately and hands off the
# render+distill to nvim's background job (nvim/changelog.lua). This test fakes
# the captured scrollback, the model (claude on PATH), and nvim — where the fake
# nvim *simulates the background job* by running the render+distill from the
# PAIR_CHANGELOG_* env the orchestrator exports. It then asserts the log was
# distilled, the anchor written, nvim was opened on the log, and the lock cleared.
set -eu

PAIR_HOME=$(cd "$(dirname "$0")/.." && pwd)
export PAIR_HOME

if [ ! -x "$PAIR_HOME/bin/pair-changelog" ] || [ ! -x "$PAIR_HOME/bin/pair-scrollback-render" ]; then
    echo "SKIP changelog-open-test: build the binaries first (make pair-changelog pair-scrollback-render)"
    exit 0
fi

# Explicit template path: macOS mktemp -d otherwise ignores TMPDIR.
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-changelog-test.XXXXXX")
trap 'rm -rf "$tmp"' EXIT

export PAIR_DATA_DIR="$tmp/data"
export PAIR_TAG="t"
export PAIR_AGENT="claude"
mkdir -p "$PAIR_DATA_DIR"

# Fake captured scrollback (a ❯ prompt line → one completed turn).
printf '%s\n' '❯ do a thing' 'agent did some work' 'finished M1' \
    > "$PAIR_DATA_DIR/scrollback-t-claude.raw"
printf '{"type":"resize","offset":0,"cols":80,"rows":24}\n' \
    > "$PAIR_DATA_DIR/scrollback-t-claude.events.jsonl"

# Fake model + fake nvim on PATH. The fake nvim records its args, then simulates
# changelog.lua's background refresh job using the exported env.
fakebin="$tmp/bin"
mkdir -p "$fakebin"
cat > "$fakebin/claude" <<'EOF'
#!/bin/sh
cat >/dev/null
printf '%s\n' '- M1 done for #53; distiller wired up'
EOF
cat > "$fakebin/nvim" <<EOF
#!/bin/sh
printf '%s\n' "\$@" > "$tmp/nvim-args"
if [ "\${PAIR_CHANGELOG_REFRESH:-0}" = "1" ]; then
    "\$PAIR_CHANGELOG_RENDER" --plain --max-lines 0 \\
        "\$PAIR_CHANGELOG_RAW" "\$PAIR_CHANGELOG_EVENTS" "\$PAIR_CHANGELOG_CLEANED" \\
      && "\$PAIR_CHANGELOG_DISTILL" --cleaned "\$PAIR_CHANGELOG_CLEANED" \\
        --log "\$PAIR_CHANGELOG_LOG" --anchor "\$PAIR_CHANGELOG_ANCHOR" \\
        --agent "\$PAIR_CHANGELOG_AGENT" --today "\$PAIR_CHANGELOG_TODAY"
fi
EOF
chmod +x "$fakebin/claude" "$fakebin/nvim"
export PATH="$fakebin:$PATH"

"$PAIR_HOME/bin/pair-changelog-open"

fail=0
LOG="$PAIR_DATA_DIR/changelog-t-claude.md"
ANCHOR="$PAIR_DATA_DIR/changelog-t-claude.anchor"

grep -q 'M1 done for #53' "$LOG" 2>/dev/null || { echo "FAIL: distilled entry missing from log:"; cat "$LOG" 2>/dev/null; fail=1; }
grep -q '^## ' "$LOG" 2>/dev/null || { echo "FAIL: no date header in log"; fail=1; }
grep -q '^turns:' "$ANCHOR" 2>/dev/null || { echo "FAIL: anchor missing turns header: $(cat "$ANCHOR" 2>/dev/null)"; fail=1; }
grep -q 'changelog-t-claude.md' "$tmp/nvim-args" 2>/dev/null || { echo "FAIL: nvim not opened on the log; args: $(cat "$tmp/nvim-args" 2>/dev/null)"; fail=1; }
[ -f "$PAIR_DATA_DIR/changelog-t-claude.openlock" ] && { echo "FAIL: openlock not cleared on exit"; fail=1; }

if [ "$fail" -eq 0 ]; then
    echo "PASS changelog-open-test"
else
    exit 1
fi
