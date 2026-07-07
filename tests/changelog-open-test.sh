#!/bin/sh
# End-to-end smoke test for bin/pair changelog open (#53/#58).
#
# The orchestrator launches render+distill as a DETACHED background process
# (survives the viewer closing) and opens nvim as a watcher. This test fakes the
# captured scrollback, the model (claude on PATH), and nvim (records its args +
# exits), runs the orchestrator, waits for the detached distiller to finish, then
# asserts the log was distilled, the anchor written, and nvim opened on the log.
set -eu

PAIR_HOME=$(cd "$(dirname "$0")/.." && pwd)
export PAIR_HOME

if [ ! -x "$PAIR_HOME/bin/pair" ]; then
    echo "SKIP changelog-open-test: build the pair binary first (make build)"
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

# Fake model + fake nvim on PATH. The fake nvim just records its args and exits
# (the real distiller runs detached, not under nvim).
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

"$PAIR_HOME/bin/pair" changelog open

# Wait for the detached distiller (its PID is in distill.lock) to finish.
DLOCK="$PAIR_DATA_DIR/changelog-t-claude.distill.lock"
LOG="$PAIR_DATA_DIR/changelog-t-claude.md"
ANCHOR="$PAIR_DATA_DIR/changelog-t-claude.anchor"
i=0
while [ $i -lt 60 ]; do
    if [ -f "$DLOCK" ]; then
        p=$(cat "$DLOCK" 2>/dev/null || true)
        if [ -n "${p:-}" ] && kill -0 "$p" 2>/dev/null; then i=$((i + 1)); sleep 1; continue; fi
    fi
    break
done

fail=0
grep -q 'M1 done for #53' "$LOG" 2>/dev/null || { echo "FAIL: distilled entry missing from log:"; cat "$LOG" 2>/dev/null; fail=1; }
grep -q '^## ' "$LOG" 2>/dev/null && { echo "FAIL: stale date header in log (dates were removed in #58)"; fail=1; }
grep -q '^turns:' "$ANCHOR" 2>/dev/null || { echo "FAIL: anchor missing turns header: $(cat "$ANCHOR" 2>/dev/null)"; fail=1; }
grep -q 'changelog-t-claude.md' "$tmp/nvim-args" 2>/dev/null || { echo "FAIL: nvim not opened on the log; args: $(cat "$tmp/nvim-args" 2>/dev/null)"; fail=1; }
[ -f "$PAIR_DATA_DIR/changelog-t-claude.openlock" ] && { echo "FAIL: openlock not cleared on viewer exit"; fail=1; }

if [ "$fail" -eq 0 ]; then
    echo "PASS changelog-open-test"
else
    exit 1
fi
