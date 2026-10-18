#!/usr/bin/env bash
# Source as the first executable fixture statement. Always cross the shared
# empty-environment boundary. The guard fixes this source statement at line 2;
# the clean child executes the remaining trusted local body with original $0.
pair_test_bootstrap_repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd) || exit 1
. "$pair_test_bootstrap_repo/tests/lib/pair-test-go-env.sh"
pair_test_resolve_go_env || exit $?
pair_test_bootstrap_root=$(mktemp -d "${TMPDIR:-/tmp}/pair-test-environment.XXXXXX") || exit 1
pair_test_bootstrap_args=()
# Explicit inputs are read-only candidate/tool paths or invocation-owned
# diagnostic fixtures. No operator session, artifact or credential paths.
for pair_test_bootstrap_key in PAIR_TEST_BINARY GOCACHE GOMODCACHE GOTMPDIR PAIR_TEST_REVIEW_INIT PAIR_TEST_DRAFT_INIT PAIR_TEST_OBSERVATION_WANTED_BIN PAIR_TEST_OBSERVATION_REPORT; do
    if [ "${!pair_test_bootstrap_key+x}" = x ]; then
        pair_test_bootstrap_args+=("$pair_test_bootstrap_key=${!pair_test_bootstrap_key}")
    fi
done
pair_test_bootstrap_status=0
/bin/sh "$pair_test_bootstrap_repo/tests/lib/pair-test-env.sh" "${pair_test_bootstrap_root:?}" \
    "${pair_test_bootstrap_args[@]}" -- bash -c "$(tail -n +3 "$0")" \
    "$0" "$@" || pair_test_bootstrap_status=$?
# Go module-cache directories are deliberately read-only. Only directories
# inside this owned root need write permission for unlink; find does not follow
# symlinks, so fixture links cannot grant authority over their caller targets.
find "${pair_test_bootstrap_root:?}" -type d -exec chmod u+rwx {} + || {
    [ "$pair_test_bootstrap_status" -ne 0 ] || pair_test_bootstrap_status=1
}
rm -rf "${pair_test_bootstrap_root:?}" || {
    [ "$pair_test_bootstrap_status" -ne 0 ] || pair_test_bootstrap_status=1
}
exit "$pair_test_bootstrap_status"
