#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
# The nvim pid sidecar's lifecycle (#583).
#
# Every pair-spawned embedded nvim publishes its pid to $PAIR_NVIM_PID_FILE so a
# reaper can end it by exact identity rather than by an argv pattern. This pins
# both halves of that record's life in the real configs under real nvim:
#
#   published — the sidecar appears at startup and names a live process;
#   removed   — a clean `qa!` deletes it, so the next sweep is not handed a pid
#               that stopped being ours;
#   survives  — a SIGKILLed embed keeps it, because that record is the only name
#               the 22 GB orphan in #583 ever had;
#   not stolen— a clean exit deletes only a record that still names *this*
#               process. A restarted editor for the same tag republishes its own
#               pid to the same path first, and deleting that would leave the
#               survivor unnamed.
#
# All four publishing configs are covered, not one. The removal cannot be shared
# with the publish: review.lua republishes on every activation (its embed is
# restarted and rolled back, so a pid recorded once at VimEnter would go stale),
# and that leaves the two write shapes apart, so a copy of the rule sits beside
# each config's own write. Four copies of a reap-path rule is the shape that let
# `review` drift out of the kind set, so behaviour pins each copy here instead of
# the copies pinning each other.
#
# Run: bash tests/nvim-pidfile-lifecycle-test.sh   (wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-nvim-pidfile.XXXXXX")" || exit 1
survivors=()
cleanup() {
  local p
  for p in ${survivors[@]+"${survivors[@]}"}; do kill "$p" 2>/dev/null; wait "$p" 2>/dev/null; done
  rm -rf "${RT:?}"
}
trap cleanup EXIT

# init.lua/scrollback.lua/changelog.lua refuse to start without a `pair` on PATH
# (retention.lua: "cannot protect editor storage"), which would abort before
# VimEnter and read as "never published". Under `make test` this is a hard link to
# the suite's one build; standalone it builds.
mkdir -p "$RT/bin"
bash "$ROOT/tests/lib/pair-test-binary.sh" "$RT/bin/pair-go" || exit 1
ln -s pair-go "$RT/bin/pair"

printf 'draft text\n' > "$RT/draft-test.md"
printf 'scrollback output\n' > "$RT/scrollback-test-claude.ansi"
printf '# change log\n' > "$RT/changelog-test-claude.md"
printf '# reviewed doc\n' > "$RT/doc.md"

# The quit trigger is a file the harness acks, not a timer: every observation
# below has to happen *while* nvim is up, and a fixed delay would race a slow
# boot. Registered through --cmd, so it exists before the config under test is
# sourced, and it defers rather than quitting inline — the config's own VimEnter
# publish runs first. vim.wait pumps the event loop, so the ack is seen; the 30s
# bound plus run_headless's watchdog keep a broken config from hanging the suite
# (#60).
cat > "$RT/harness.lua" <<'LUA'
vim.api.nvim_create_autocmd('VimEnter', {
  callback = function()
    vim.defer_fn(function()
      vim.wait(30000, function() return vim.loop.fs_stat(vim.env.PAIR_PIDFILE_ACK) ~= nil end, 20)
      vim.cmd('qa!')
    end, 0)
  end,
})
LUA

fails=0
report() {
  if [ "$1" = ok ]; then printf '  ok   %s\n' "$2"
  else printf '  FAIL %s\n' "$2"; fails=$((fails + 1)); fi
}
log_on_fail() { [ "$1" = ok ] || sed 's/^/    nvim| /' "$2"; }

START_PID='' SIDECAR='' ACK='' LOG=''
start() { # start NAME CONFIG FILE
  local name=$1 config=$2 file=$3
  SIDECAR="$RT/pid-$name"; ACK="$RT/ack-$name"; LOG="$RT/$name.log"
  rm -f "$SIDECAR" "$ACK" "$LOG"
  PATH="$RT/bin:$PATH" \
  PAIR_NVIM_PID_FILE="$SIDECAR" PAIR_PIDFILE_ACK="$ACK" \
  PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
  PAIR_DRAFT_PATH="$RT/draft-test.md" \
  PAIR_SCROLLBACK_ANSI_PATH="$RT/scrollback-test-claude.ansi" \
  PAIR_CHANGELOG_LOG="$RT/changelog-test-claude.md" \
    run_headless --timeout 45 -- \
    nvim --headless --cmd "luafile $RT/harness.lua" -u "$config" "$file" >"$LOG" 2>&1 &
  START_PID=$!
}

published() { # 0 once the sidecar holds something
  local i
  for i in $(seq 1 400); do
    [ -s "$SIDECAR" ] && return 0
    sleep 0.02
  done
  return 1
}

quit() { : > "$ACK"; wait "$START_PID" 2>/dev/null; }

# record_pid reads the record's first line. review.lua writes a second one (the
# file its embed serves), so a whole-file read is not a pid (#583).
record_pid() { head -n 1 "$SIDECAR"; }

never_published() { # NAME
  report FAIL "$1: publishes its embed pid"
  log_on_fail FAIL "$LOG"
  quit
}

# A clean exit: the record appears, names something alive, and is gone afterwards.
case_removes_on_clean_exit() {
  local name=$1 config=$2 file=$3 rec live
  start "$name" "$config" "$file"
  if ! published; then never_published "$name"; return; fi
  rec=$(record_pid)
  report ok "$name: publishes its embed pid ($rec)"
  if kill -0 "$rec" 2>/dev/null; then live=ok; else live=FAIL; fi
  report "$live" "$name: the record names a live process"
  log_on_fail "$live" "$LOG"
  # The review kind's second line is the reaper's owner-naming evidence: it must
  # be the file this embed was spawned with, byte for byte, because the reaper
  # compares it against the process's argv (#583). The other kinds write one line.
  local served
  served=$(sed -n 2p "$SIDECAR")
  if [ "$name" = review ]; then
    if [ "$served" = "$file" ]; then report ok "$name: the record binds the file it serves"
    else
      report FAIL "$name: the record binds the file it serves (line 2 is '$served', want '$file')"
      log_on_fail FAIL "$LOG"
    fi
  elif [ -n "$served" ]; then
    report FAIL "$name: writes only a pid (line 2 is '$served')"
    log_on_fail FAIL "$LOG"
  fi
  quit
  if [ ! -e "$SIDECAR" ]; then report ok "$name: a clean exit removes the record"
  else
    report FAIL "$name: a clean exit removes the record (still holds $(record_pid 2>/dev/null))"
    log_on_fail FAIL "$LOG"
  fi
}

# An unclean death: the record is the reaper's only name for the process, so it
# must survive. kill -9 the *recorded* pid, which is the statement under test.
case_survives_sigkill() {
  local name=$1 config=$2 file=$3 rec held
  start "$name" "$config" "$file"
  if ! published; then never_published "$name"; return; fi
  rec=$(record_pid)
  kill -9 "$rec" 2>/dev/null
  wait "$START_PID" 2>/dev/null
  held=$(record_pid 2>/dev/null)
  if [ "$held" = "$rec" ]; then report ok "$name: a SIGKILLed embed keeps its record"
  else
    report FAIL "$name: a SIGKILLed embed keeps its record (holds '$held', want '$rec')"
    log_on_fail FAIL "$LOG"
  fi
}

# A later embed for the same tag republished its own pid to the same path before
# this one exited. Removing that record would leave the survivor unnamed.
case_keeps_a_republished_record() {
  local name=$1 config=$2 file=$3 survivor held
  start "$name" "$config" "$file"
  if ! published; then never_published "$name"; return; fi
  sleep 60 &
  survivor=$!
  survivors+=("$survivor")
  echo "$survivor" > "$SIDECAR"
  quit
  held=$(record_pid 2>/dev/null)
  if [ "$held" = "$survivor" ]; then
    report ok "$name: a clean exit keeps a record a later embed republished"
  else
    report FAIL "$name: a clean exit keeps a record a later embed republished (holds '$held', want '$survivor')"
    log_on_fail FAIL "$LOG"
  fi
  kill "$survivor" 2>/dev/null
  wait "$survivor" 2>/dev/null
}

echo "nvim-pidfile-lifecycle-test:"
for spec in \
  "draft|$ROOT/nvim/init.lua|$RT/draft-test.md" \
  "scrollback|$ROOT/nvim/scrollback.lua|$RT/scrollback-test-claude.ansi" \
  "changelog|$ROOT/nvim/changelog.lua|$RT/changelog-test-claude.md" \
  "review|$ROOT/nvim/review.lua|$RT/doc.md"; do
  IFS='|' read -r name config file <<< "$spec"
  case_removes_on_clean_exit "$name" "$config" "$file"
  case_survives_sigkill "$name" "$config" "$file"
  case_keeps_a_republished_record "$name" "$config" "$file"
done

if [ "$fails" -ne 0 ]; then
  echo "nvim-pidfile-lifecycle-test: $fails failure(s)"
  exit 1
fi
echo "nvim-pidfile-lifecycle-test: all passed"
