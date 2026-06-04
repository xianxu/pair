# emitter-health.sh — probe whether pair's Go adapt emitters were built WITH
# logging code (#000047). Sourced by doctor.sh; functions kept thin + injectable
# so the binary-selection logic is unit-testable without live processes.
#
# Why this exists: a pair-wrap / pair-slug binary built before #000045 has no
# adaptation-logging code at all, so it emits nothing and the flight recorder
# goes silent WITH NO ERROR. The doctor would otherwise show a thin log and be
# unable to name the cause — the exact confusion that produced #000046/#000047.
# (#000046's `pair-dev` PREVENTS staleness at launch; this DIAGNOSES it.)
#
# The probe greps each binary for an adapt *signal string* compiled in by its
# emitter. Keep these markers in sync with the emitter call sites — a rename
# there must be mirrored here or the probe false-flags a fresh binary as stale:
#   pair-wrap → cmd/pair-wrap/main.go  ("return-remap" / "overlay-detect" / "output-filter")
#   pair-slug → cmd/pair-slug/main.go  ("slug-parse")

# _pid_exe PID — print the executable path backing PID (portable, best-effort).
# Empty when unresolvable (dead pid, no tool, sandbox) so the caller falls back.
_pid_exe() {
    local pid="$1" exe=""
    [ -n "$pid" ] || return 0
    if [ -r "/proc/$pid/exe" ]; then                      # Linux
        exe="$(readlink "/proc/$pid/exe" 2>/dev/null)"
    fi
    if [ -z "$exe" ] && command -v lsof >/dev/null 2>&1; then   # macOS/full path
        # FD column ($4) == "txt" is the executable text image; $NF is its path.
        # (Must filter on txt — the first plain `n` record is the cwd, not the exe.)
        exe="$(lsof -p "$pid" 2>/dev/null | awk '$4=="txt"{print $NF; exit}')"
    fi
    if [ -z "$exe" ]; then                                 # last resort (may truncate)
        exe="$(ps -p "$pid" -o comm= 2>/dev/null | head -n1)"
    fi
    # if/fi (not `&& printf`) so the function returns 0 even when exe is empty —
    # the caller runs under `set -e` (doctor.sh) and must not abort here.
    if [ -n "$exe" ]; then printf '%s\n' "$exe"; fi
}

# _resolve_emitter NAME DATADIR TAG — print the binary path to check. Prefers the
# *actually running* binary (via the pair-wrap-pid-<tag> pidfile) since that's
# what is (or isn't) emitting — checking the PATH binary can miss the case where
# a fresh repo/bin coexists with a stale running ~/.local/bin one (the original
# bug). Only pair-wrap is long-running with a pidfile; pair-slug is on-demand,
# so it resolves via PATH only.
_resolve_emitter() {
    local name="$1" datadir="$2" tag="$3" pidfile pid exe
    if [ "$name" = "pair-wrap" ] && [ -n "$tag" ]; then
        pidfile="$datadir/pair-wrap-pid-$tag"
        if [ -r "$pidfile" ]; then
            pid="$(head -n1 "$pidfile" 2>/dev/null | tr -dc '0-9')"
            exe="$(_pid_exe "$pid")"
            if [ -n "$exe" ] && [ -e "$exe" ]; then printf '%s\n' "$exe"; return 0; fi
        fi
    fi
    command -v "$name" 2>/dev/null || true
}

# _binary_has_marker PATH MARKER — 0 present, 1 absent, 2 can't-tell (no file or
# no `strings`). A binary with adapt logging carries MARKER in its string table.
_binary_has_marker() {
    local path="$1" marker="$2" n
    [ -n "$path" ] && [ -e "$path" ] || return 2
    command -v strings >/dev/null 2>&1 || return 2
    # `grep -c` (count, consumes all input) NOT `grep -q` (early-exits on first
    # match): under `set -o pipefail` (doctor.sh), grep -q's early exit SIGPIPEs
    # `strings`, so the pipeline reports non-zero even on a match → false STALE.
    n="$(strings "$path" 2>/dev/null | grep -c -- "$marker")" || true
    [ "${n:-0}" -gt 0 ] && return 0
    return 1
}

# emitter_health_report DATADIR TAG — print the "emitter health" section.
# Always returns 0 (diagnostic; staleness is signalled in the OUTPUT, not the
# exit code, so it's safe under the launcher's / doctor.sh's `set -e`).
emitter_health_report() {
    local datadir="$1" tag="${2:-}" spec name marker bin
    echo "-- emitter health (are the Go adapt emitters built with logging?) --"
    for spec in "pair-wrap:return-remap" "pair-slug:slug-parse"; do
        name="${spec%%:*}"; marker="${spec##*:}"
        bin="$(_resolve_emitter "$name" "$datadir" "$tag")"
        if [ -z "$bin" ]; then
            echo "  [?]     $name — not found (PATH or pidfile); can't check"
            continue
        fi
        # `|| rc=$?` so a non-zero verdict (1/2) doesn't trip the caller's
        # `set -e`; the verdict is then dispatched by the case.
        local rc=0
        _binary_has_marker "$bin" "$marker" || rc=$?
        case $rc in
            0) echo "  [ok]    $name  ($bin)" ;;
            1) echo "  [STALE] $name  ($bin) — no adapt logging; its aspects can't log."
               echo "          Fix: make install (or launch via pair-dev). See atlas \"Binary freshness\"." ;;
            2) echo "  [?]     $name  ($bin) — strings unavailable; can't check" ;;
        esac
    done
    return 0
}
