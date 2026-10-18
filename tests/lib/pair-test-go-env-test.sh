#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/pair-go-env-check.XXXXXX") || exit 1
trap 'find "${scratch:?}" -type d -exec chmod u+rwx {} +; rm -rf "${scratch:?}"' EXIT HUP INT TERM
mkdir -p "$scratch/tools" "$scratch/caller home"
cat > "$scratch/tools/go" <<'GO'
#!/bin/sh
set -eu
if [ "$1" = env ]; then
  [ ! -f "$HOME/fail" ] || exit 37
  case "$2" in
    GOCACHE) printf '%s\n' "$HOME/compiler cache" ;;
    GOMODCACHE) printf '%s\n' "$HOME/module cache" ;;
    GOTMPDIR) printf '\n' ;;
    *) exit 99 ;;
  esac
elif [ "$1" = build ]; then
  mkdir -p "$GOCACHE"
  if [ ! -f "$GOCACHE/payload" ]; then
    printf 'fixed compiler payload\n' > "$GOCACHE/payload"
    printf 'write\n' >> "$GOCACHE/writes"
  fi
  printf 'fresh executable\n' > "$HOME/output"
else exit 98
fi
GO
chmod +x "$scratch/tools/go"
printf '#!/usr/bin/env bash\n. "%s/tests/lib/pair-test-bootstrap.sh"\n' "$repo" > "$scratch/fixture"
cat >> "$scratch/fixture" <<'CHILD'
set -eu
[ "$GOCACHE" = "$1/compiler cache" ]
[ "$GOMODCACHE" = "$1/module cache" ]
[ "${GOTMPDIR+x}" = x ] && [ -z "$GOTMPDIR" ]
[ -z "${COUCH_TREE+x}" ] && [ -z "${PAIR_DATA_DIR+x}" ]
go build
[ -f "$HOME/output" ]
printf '%s\n' "$HOME" >> "$2"
CHILD
for boundary in bootstrap isolated; do
  i=0
  while [ "$i" -lt 16 ]; do
    if [ "$boundary" = bootstrap ]; then
      env -u GOCACHE -u GOMODCACHE -u GOTMPDIR HOME="$scratch/caller home" PATH="$scratch/tools:$PATH" COUCH_TREE=foreign PAIR_DATA_DIR=/foreign bash "$scratch/fixture" "$scratch/caller home" "$scratch/homes"
    else
      env -u GOCACHE -u GOMODCACHE -u GOTMPDIR HOME="$scratch/caller home" PATH="$scratch/tools:$PATH" COUCH_TREE=foreign PAIR_DATA_DIR=/foreign /bin/sh "$repo/tests/with-isolated-pair.sh" /bin/sh -ec '[ "$GOCACHE" = "$1/compiler cache" ]; [ "$GOMODCACHE" = "$1/module cache" ]; [ "${GOTMPDIR+x}" = x ]; [ -z "$GOTMPDIR" ]; [ -z "${COUCH_TREE+x}" ]; go build; printf "%s\n" "$HOME" >> "$2"' fixture "$scratch/caller home" "$scratch/homes"
    fi
    i=$((i + 1))
    if [ "$i" = 8 ] || [ "$i" = 16 ]; then
      [ "$(wc -l < "$scratch/caller home/compiler cache/writes" | tr -d ' ')" = 1 ]
    fi
  done
done
[ "$(sort -u "$scratch/homes" | wc -l | tr -d ' ')" = 32 ]
while IFS= read -r home; do [ ! -e "$home" ]; done < "$scratch/homes"
# Explicit tool settings are literal, including the Go off sentinel and empty temp.
HOME="$scratch/caller home" PATH="$scratch/tools:$PATH" GOCACHE=off GOMODCACHE='literal space $(never-run)' GOTMPDIR= /bin/sh "$repo/tests/with-isolated-pair.sh" /bin/sh -ec '[ "$GOCACHE" = off ] && [ "$GOMODCACHE" = '\''literal space $(never-run)'\'' ] && [ "${GOTMPDIR+x}" = x ] && [ -z "$GOTMPDIR" ]'
printf '#!/usr/bin/env bash\n. "%s/tests/lib/pair-test-bootstrap.sh"\n' "$repo" > "$scratch/explicit"
cat >> "$scratch/explicit" <<'CHILD'
set -eu
[ "$GOCACHE" = off ]
[ "$GOMODCACHE" = 'literal space $(never-run)' ]
[ "${GOTMPDIR+x}" = x ] && [ -z "$GOTMPDIR" ]
CHILD
HOME="$scratch/caller home" PATH="$scratch/tools:$PATH" GOCACHE=off GOMODCACHE='literal space $(never-run)' GOTMPDIR= bash "$scratch/explicit"
# A distinct intentional cache key receives its own fixed payload.
HOME="$scratch/caller home" PATH="$scratch/tools:$PATH" GOCACHE="$scratch/intentional cache" /bin/sh "$repo/tests/with-isolated-pair.sh" go build
[ "$(wc -l < "$scratch/intentional cache/writes" | tr -d ' ')" = 1 ]
# Resolution errors keep their status and prevent executing a fixture.
touch "$scratch/caller home/fail"
status=0
env -u GOCACHE -u GOMODCACHE -u GOTMPDIR HOME="$scratch/caller home" PATH="$scratch/tools:$PATH" /bin/sh "$repo/tests/with-isolated-pair.sh" /bin/sh -c 'exit 99' || status=$?
[ "$status" = 37 ]
status=0
env -u GOCACHE -u GOMODCACHE -u GOTMPDIR HOME="$scratch/caller home" PATH="$scratch/tools:$PATH" bash "$scratch/fixture" unused unused || status=$?
[ "$status" = 37 ]
# Unrelated shell commands do not acquire a Go prerequisite.
mkdir "$scratch/no-go"
for tool in dirname mktemp mkdir rm; do ln -s "$(command -v "$tool")" "$scratch/no-go/$tool"; done
env -u GOCACHE -u GOMODCACHE -u GOTMPDIR PATH="$scratch/no-go" /bin/sh "$repo/tests/with-isolated-pair.sh" /bin/sh -c 'exit 0'
env -u GOCACHE -u GOMODCACHE -u GOTMPDIR PATH="$scratch/no-go" /bin/sh -c '. "$1"; pair_test_resolve_go_env; [ -z "${GOCACHE+x}" ]; [ -z "${GOMODCACHE+x}" ]; [ -z "${GOTMPDIR+x}" ]' fixture "$repo/tests/lib/pair-test-go-env.sh"
printf 'Go tool cache boundary PASS: 32 fresh homes, one shared payload; distinct key one payload\n'
