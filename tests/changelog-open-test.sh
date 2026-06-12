#!/bin/sh
# End-to-end smoke test for bin/pair-changelog-open (#53 M2).
#
# Fakes the captured scrollback, the model (claude on PATH), and nvim (a stub
# that records the file it was told to open), then asserts the orchestrator
# cleans the TTY → distills → writes the log + anchor → opens the viewer on the
# log, and clears its lock on exit. The shell glue otherwise has no coverage.
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

# Fake captured scrollback (plain text renders to clean lines via the VT100 path).
printf 'intro line\nagent did some work\nfinished M1\n' > "$PAIR_DATA_DIR/scrollback-t-claude.raw"
printf '{"type":"resize","offset":0,"cols":80,"rows":24}\n' > "$PAIR_DATA_DIR/scrollback-t-claude.events.jsonl"

# Fake model + fake nvim on PATH.
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
EOF
chmod +x "$fakebin/claude" "$fakebin/nvim"
export PATH="$fakebin:$PATH"

"$PAIR_HOME/bin/pair-changelog-open"

fail=0
LOG="$PAIR_DATA_DIR/changelog-t-claude.md"
ANCHOR="$PAIR_DATA_DIR/changelog-t-claude.anchor"

grep -q 'M1 done for #53' "$LOG" 2>/dev/null || { echo "FAIL: distilled entry missing from log:"; cat "$LOG" 2>/dev/null; fail=1; }
grep -q '^## ' "$LOG" 2>/dev/null || { echo "FAIL: no date header in log"; fail=1; }
[ -s "$ANCHOR" ] || { echo "FAIL: anchor not written"; fail=1; }
grep -q 'changelog-t-claude.md' "$tmp/nvim-args" 2>/dev/null || { echo "FAIL: nvim not opened on the log; args: $(cat "$tmp/nvim-args" 2>/dev/null)"; fail=1; }
[ -f "$PAIR_DATA_DIR/changelog-t-claude.openlock" ] && { echo "FAIL: openlock not cleared on exit"; fail=1; }

if [ "$fail" -eq 0 ]; then
    echo "PASS changelog-open-test"
else
    exit 1
fi
