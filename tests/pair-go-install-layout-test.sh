#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_home="$(mktemp -d "${TMPDIR:-/tmp}/pair-go-install-layout.XXXXXX")"
trap 'rm -rf "$tmp_home"' EXIT
tmp_home="$(cd "$tmp_home" && pwd -P)"
# Exercise the real install recipe without compiling or touching the user prefix.
# A checkout and prefix with spaces also protect the shell quoting contract.
fixture="$tmp_home/source tree"
prefix="$tmp_home/install prefix"
mkdir -p "$fixture/bin" "$prefix"
cp "$repo_root/Makefile.local" "$fixture/Makefile"
for name in pair couch pair-launch-helper pair-dev couch-dev; do
    printf '#!/bin/sh\nprintf "first build\\n"\n' > "$fixture/bin/$name"
    chmod +x "$fixture/bin/$name"
done
# Both old copies and old links must be replaced without modifying their targets.
printf 'stale copy\n' > "$prefix/couch"
printf 'old target\n' > "$tmp_home/old-target"
ln -s "$tmp_home/old-target" "$prefix/pair"
make -s -C "$fixture" -o build install "INSTALL_DIR=$prefix" >/dev/null
for name in pair couch pair-launch-helper pair-dev couch-dev; do
    if [[ ! -L "$prefix/$name" ]]; then
        printf 'FAIL: installed %s must link to the checkout\n' "$name" >&2
        exit 1
    fi
    test "$(readlink "$prefix/$name")" = "$fixture/bin/$name"
done
test "$(cat "$tmp_home/old-target")" = 'old target'
test "$(readlink "$prefix/pair-slug")" = pair
# Build tools replace binaries by rename. The installed path must follow it.
printf '#!/bin/sh\nprintf "second build\\n"\n' > "$fixture/bin/couch.next"
chmod +x "$fixture/bin/couch.next"
mv "$fixture/bin/couch.next" "$fixture/bin/couch"
test "$("$prefix/couch")" = 'second build'
make -s -C "$fixture" -o build install "INSTALL_DIR=$prefix" >/dev/null
test "$("$prefix/couch")" = 'second build'
if [[ "${1:-}" == --links-only ]]; then
    printf 'install link regression passed\n'
    exit 0
fi

gomodcache="$(go env GOMODCACHE)"
gocache="$(go env GOCACHE)"

install_bin="$tmp_home/.local/bin"
old_bin="$tmp_home/old-bin"
mkdir -p "$install_bin" "$old_bin"
printf '#!/usr/bin/env bash\nprintf old-pair-shell\\n\n' > "$old_bin/pair"
chmod +x "$old_bin/pair"
ln -s "$old_bin/pair" "$install_bin/pair"

HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null

test -x "$install_bin/pair"
test -L "$install_bin/pair"
test -L "$install_bin/couch"
test -L "$install_bin/pair-launch-helper"
test -L "$install_bin/pair-dev"
test -L "$install_bin/couch-dev"
# #104 M3: the single binary — no separate pair-go; the only busybox symlink is
# pair-slug (external Stop hook), pointing at pair.
test ! -e "$install_bin/pair-go"
test -L "$install_bin/pair-slug"
test "$(readlink "$install_bin/pair-slug")" = pair

out="$("$install_bin/pair" --help)"
case "$out" in
    pair\ —*) ;;
    *)
        printf 'pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
        exit 1
        ;;
esac

# The pair-slug busybox symlink routes to `pair slug` (env-less → tolerant no-op,
# exit 0) — proves argv[0] dispatch resolves the external Stop-hook name.
PAIR_TAG="" PAIR_DATA_DIR="" "$install_bin/pair-slug"

out="$(PAIR_HOME="$repo_root" "$install_bin/pair" --help)"
case "$out" in
    pair\ —*) ;;
    *)
        printf 'PAIR_HOME pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
        exit 1
        ;;
esac

printf 'pair-go install layout test passed\n'
