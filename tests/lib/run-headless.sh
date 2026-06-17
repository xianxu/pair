# Shared timeout watchdog for the headless-nvim regression tests (issue #60).
#
# Why this exists: a driver that fails to quit cleanly — e.g. `vim.cmd('qall')`
# on a buffer the driver modified, which raises `E37: No write since last
# change` and so refuses to quit — leaves `nvim --headless` blocked in its main
# loop FOREVER. Unbounded, that one stuck boot hangs the entire `make test`
# suite (observed: a 12m54s hang on test-autopair, plus week-old leaked nvim
# corpses in the process table). This wrapper bounds every headless boot and
# FAILS LOUD on overrun: it kills the stuck nvim, prints a diagnostic naming
# #60, and returns 124 (the conventional timeout exit code) instead of hanging.
#
# It also stops swallowing output. The old call form was
# `nvim --headless … >/dev/null 2>&1 || true`, which hid both the hang and any
# boot/driver error. run_headless captures nvim's stdout+stderr and surfaces it
# (prefixed `nvim|`) on ANY failure — timeout or non-zero exit — while staying
# quiet on success so the suite's own output stays clean.
#
# Usage (source this file, then):
#   run_headless [--timeout SECS] -- <command…>
# stdin is fed from /dev/null. Returns the command's exit code, or 124 on
# timeout. SECS defaults to 30.
#
# Portability: prefers a real timeout(1) — GNU coreutils `timeout`, or
# `gtimeout` on macOS via `brew install coreutils`. When neither is on PATH
# (the common macOS-without-coreutils case, including this repo's dev box) it
# falls back to a background + 1s-poll + SIGKILL loop, so the watchdog itself
# is never a portability landmine.

run_headless() {
  local timeout_s=30
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --timeout) timeout_s="$2"; shift 2 ;;
      --)        shift; break ;;
      *)         break ;;
    esac
  done
  if [ "$#" -eq 0 ]; then
    echo "run_headless: no command given" >&2
    return 2
  fi

  local out rc
  out="$(mktemp "${TMPDIR:-/tmp}/run-headless.XXXXXX")"

  local to_bin=''
  if   command -v timeout  >/dev/null 2>&1; then to_bin='timeout'
  elif command -v gtimeout >/dev/null 2>&1; then to_bin='gtimeout'
  fi

  if [ -n "$to_bin" ]; then
    # -k 2: if the TERM at the deadline is ignored, SIGKILL 2s later.
    if "$to_bin" -k 2 "$timeout_s" "$@" >"$out" 2>&1 </dev/null; then
      rc=0
    else
      rc=$?
    fi
  else
    # Portable fallback: background + poll + kill. `set -e`-safe — every
    # non-zero exit is consumed by a condition, never a bare statement.
    "$@" >"$out" 2>&1 </dev/null &
    local pid=$! waited=0
    rc=''
    while kill -0 "$pid" 2>/dev/null; do
      if [ "$waited" -ge "$timeout_s" ]; then
        kill -9 "$pid" 2>/dev/null
        wait "$pid" 2>/dev/null || true
        rc=124
        break
      fi
      sleep 1
      waited=$((waited + 1))
    done
    if [ -z "$rc" ]; then
      if wait "$pid"; then rc=0; else rc=$?; fi
    fi
  fi

  if [ "$rc" -eq 124 ]; then
    {
      echo "run_headless: TIMEOUT after ${timeout_s}s — nvim did not exit."
      echo "run_headless: likely a driver that failed to quit (e.g. 'qall' on a"
      echo "run_headless: modified buffer → E37); see issue #60. nvim output:"
      sed 's/^/  nvim| /' "$out"
    } >&2
  elif [ "$rc" -ne 0 ]; then
    {
      echo "run_headless: nvim exited non-zero (rc=$rc). nvim output:"
      sed 's/^/  nvim| /' "$out"
    } >&2
  fi
  rm -f "$out"
  return "$rc"
}
