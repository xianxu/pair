#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_home="$(mktemp -d "${TMPDIR:-/tmp}/pair-go-install-layout.XXXXXX")"
trap 'rm -rf "$tmp_home"' EXIT
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
test ! -L "$install_bin/pair"
test -L "$install_bin/pair-dev"
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
