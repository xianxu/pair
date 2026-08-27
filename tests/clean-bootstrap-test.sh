#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd -P)"
base_link="$(readlink "$root/Makefile")"
base_dir="$(cd "$root/$(dirname "$base_link")" && pwd -P)"
clean_root="$(mktemp -d "${TMPDIR:-/tmp}/pair-clean-bootstrap.XXXXXX")"
trap 'rm -rf "$clean_root"' EXIT

git -C "$root" archive HEAD | tar -x -C "$clean_root"
if ! git -C "$root" diff --quiet HEAD -- \
  . \
  ':(exclude)workshop/plans/*-close-gate.md' \
  ':(exclude)workshop/plans/*-review.md'; then
  git -C "$root" diff HEAD --binary -- \
    . \
    ':(exclude)workshop/plans/*-close-gate.md' \
    ':(exclude)workshop/plans/*-review.md' |
    git -C "$clean_root" apply --binary
fi

test ! -e "$clean_root/.git"
test ! -e "$clean_root/cmd/internal/runtimebundle/assets/runtime"
ln -snf "$base_dir/Makefile" "$clean_root/Makefile"
ln -snf "$base_dir/Makefile.workflow" "$clean_root/Makefile.workflow"

plan="$(cd "$clean_root" && make -f Makefile.local -n test)"
first_command="$(printf '%s\n' "$plan" | sed -n '1p')"
case "$first_command" in
  *runtimebundle/generatecmd*) ;;
  *)
    echo "make test does not generate the runtime bundle first: $first_command" >&2
    exit 1
    ;;
esac

clean_cache="$(mktemp -d "${TMPDIR:-/tmp}/pair-clean-bootstrap-cache.XXXXXX")"
trap 'rm -rf "$clean_root" "$clean_cache"' EXIT
(
  cd "$clean_root"
  GOCACHE="$clean_cache" make -f Makefile.local test
)

echo "clean bootstrap test passed"
