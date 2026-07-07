#!/bin/sh
# End-to-end smoke for bin/pair scrollback open (#93 M2, the Go port of the
# floating-pane Alt+/ launcher). Fakes the captured PTY scrollback and nvim, runs
# the launcher, and asserts it rendered the .ansi and opened the viewer on it.
# No zellij on PATH ⇒ the viewport overlay is skipped gracefully (best-effort).
set -eu

PAIR_HOME=$(cd "$(dirname "$0")/.." && pwd)
export PAIR_HOME

if [ ! -x "$PAIR_HOME/bin/pair" ]; then
    echo "SKIP scrollback-open-test: build the binaries first (make build)"
    exit 0
fi

tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-scrollback-test.XXXXXX")
trap 'rm -rf "$tmp"' EXIT

export PAIR_DATA_DIR="$tmp/data"
export PAIR_TAG="t"
export PAIR_AGENT="claude"
mkdir -p "$PAIR_DATA_DIR"

# Fake captured PTY scrollback (raw bytes + a minimal resize sidecar so the
# in-process renderer produces an .ansi).
printf '%s\n' 'hello from the agent' 'a second line of output' \
    > "$PAIR_DATA_DIR/scrollback-t-claude.raw"
printf '{"type":"resize","offset":0,"cols":80,"rows":24}\n' \
    > "$PAIR_DATA_DIR/scrollback-t-claude.events.jsonl"

# Fake nvim records the file path it was opened on (last positional arg). No
# zellij fake ⇒ AgentPaneID() returns "" and the viewport overlay no-ops.
fakebin="$tmp/bin"
mkdir -p "$fakebin"
cat > "$fakebin/nvim" <<EOF
#!/bin/sh
for a in "\$@"; do :; done
printf '%s\n' "\$a" > "$tmp/nvim-arg"
EOF
chmod +x "$fakebin/nvim"
export PATH="$fakebin:$PATH"

"$PAIR_HOME/bin/pair" scrollback open

ANSI="$PAIR_DATA_DIR/scrollback-t-claude.ansi"
fail=0
[ -s "$ANSI" ] || { echo "FAIL: renderer did not write .ansi"; fail=1; }
case "$(cat "$tmp/nvim-arg" 2>/dev/null)" in
    *scrollback-t-claude.ansi) ;;
    *) echo "FAIL: nvim not opened on the .ansi: $(cat "$tmp/nvim-arg" 2>/dev/null)"; fail=1 ;;
esac
# The re-entrancy openlock is cleared on a clean exit.
[ -f "$PAIR_DATA_DIR/scrollback-t-claude.openlock" ] \
    && { echo "FAIL: openlock left behind"; fail=1; }

if [ "$fail" = 0 ]; then
    echo "PASS scrollback-open-test"
else
    exit 1
fi
