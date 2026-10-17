#!/bin/sh
# Single environment owner for Go, shell and Python Pair subprocess fixtures.
# Usage: pair-test-env.sh ROOT [KEY=VALUE ...] -- COMMAND [ARG ...]
#        pair-test-env.sh --print0 ROOT [KEY=VALUE ...]
set -eu
mode=run
if [ "${1-}" = --print0 ]; then mode=print; shift; fi
root=${1-}
case "$root" in /*) ;; *) echo 'pair-test-env: disposable absolute root required' >&2; exit 2 ;; esac
[ -d "$root" ] || { echo 'pair-test-env: root must exist' >&2; exit 2; }
shift
if [ "$mode" = print ]; then
  printf '%s\000' "PATH=${PATH:-/usr/bin:/bin}" "HOME=$root" "PAIR_TEST_ENV_SCRIPT=$0"
  for entry do
    case "$entry" in [A-Za-z_]*=*) printf '%s\000' "$entry" ;; *) echo 'pair-test-env: expected explicit KEY=VALUE' >&2; exit 2 ;; esac
  done
  exit 0
fi
# Rotate explicit assignments past the command, drop the separator, then
# rotate the command back. Positional parameters preserve spaces/newlines.
assignments=0
while [ "$1" != -- ]; do
  entry=$1
  case "$entry" in [A-Za-z_]*=*) ;; *) echo 'pair-test-env: expected KEY=VALUE' >&2; exit 2 ;; esac
  shift
  set -- "$@" "$entry"
  assignments=$((assignments + 1))
  [ "$assignments" -lt "$#" ] || { echo 'pair-test-env: missing -- command separator' >&2; exit 2; }
done
shift
commands=$(($# - assignments))
[ "$commands" -gt 0 ] || { echo 'pair-test-env: missing command' >&2; exit 2; }
while [ "$commands" -gt 0 ]; do
  entry=$1; shift
  set -- "$@" "$entry"
  commands=$((commands - 1))
done
exec /usr/bin/env -i "PATH=${PATH:-/usr/bin:/bin}" "HOME=$root" "PAIR_TEST_ENV_SCRIPT=$0" "$@"
