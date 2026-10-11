#!/usr/bin/env bash
# tests/host-preflight.sh — can this host run `make test` at all? (pair#450)
#
# The suite needs unix sockets (couch, notify, the review pane's restore server),
# PTYs (wrap, term, ptychild) and exec of ps (process identity in launcher,
# procutil, pair-go). An agent sandbox refuses all three, and the suite then
# reports ~150 failures that read like code bugs; workers kept recording them as
# "pre-existing on main". Probe each once and name the refusal. A refusal this
# script does not probe still fails as an ordinary test; add its probe here.
#
# Run: bash tests/host-preflight.sh
set -uo pipefail
if ! command -v python3 >/dev/null 2>&1; then
  echo "host-preflight FAILED: python3 not found (the test suite needs it)" >&2
  exit 1
fi
out="$(python3 -I - <<'PY' 2>&1
import os, pty, socket, subprocess, tempfile
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
    subprocess.run(["ps", "-p", str(os.getpid()), "-o", "pid="], check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
except (OSError, subprocess.CalledProcessError) as e:
    refused.append("exec ps: %s" % e)
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
