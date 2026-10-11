#!/usr/bin/env bash
# tests/host-preflight.sh — can this host run `make test` at all? (pair#450)
#
# The suite needs unix sockets (couch, notify, the review pane's restore server)
# and PTYs (wrap, term, ptychild). An agent sandbox refuses both, and the suite
# then reports ~150 failures that read like code bugs. Workers kept recording
# them as "pre-existing on main". Fail first, once, and name the cause.
#
# Run: bash tests/host-preflight.sh
set -uo pipefail
out="$(python3 -I - <<'PY' 2>&1
import os, pty, socket, tempfile
d = tempfile.mkdtemp()
refused = []
try:
    s = socket.socket(socket.AF_UNIX)
    s.bind(os.path.join(d, "probe.sock"))
    s.close()
except OSError as e:
    refused.append("unix socket bind: %s" % e)
try:
    m, s = pty.openpty()
    os.close(m); os.close(s)
except OSError as e:
    refused.append("pty open: %s" % e)
try:
    os.unlink(os.path.join(d, "probe.sock"))
except OSError:
    pass
os.rmdir(d)
print("\n".join(refused))
raise SystemExit(1 if refused else 0)
PY
)"
if [ $? -eq 0 ]; then
  echo "host-preflight ok"
  exit 0
fi
printf 'host-preflight FAILED: this host refuses what the test suite needs:\n' >&2
printf '%s\n' "$out" | sed 's/^/  /' >&2
printf 'An agent sandbox does this. Rerun make test outside the sandbox;\n' >&2
printf 'failures from this shell are not evidence about the code.\n' >&2
exit 1
