#!/usr/bin/env bash
# tests/lib/fail-docflow.sh — a docflow stand-in that always fails, for the
# milestone-review I3 test (a failed round must surface, not silently leave an
# edited+saved buffer with no commit).
echo "fail-docflow: simulated failure ($*)" >&2
exit 1
