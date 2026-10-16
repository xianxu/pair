---
id: 000591
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '49d994a1448385703258d79efb988b5c7c90241f' # card fields mirrored from issue-cards; edit via sdlc
---

# Diagnose legacy review resume reconstruction failure

## Problem

Recorded full suite on unchanged source74ea2d2aacf1232af3d0ae38f86353e596ae6208 failed in tests/review-resume-test.sh. The initial legacy empty-agent-round case returned placed=false hl=0 diag=0, producing three failed assertions. Later explicit resolved-identity and ambiguous-history cases passed. No identity status, diagnostic, command, stdout/stderr or timing was preserved for the first case, so the cause cannot be established from the failing artifact.

## Spec

Diagnose the implicit identity-resolution/reconstruction boundary without weakening assertions or increasing timeouts. The first case invokes reconstruct_on_open without identity; nvim/review/init.lua calls identity.resolve and, for missing history, retries resolution with the selected file/head. identity.lua chooses PAIR_HOME/bin/pair when inherited, otherwise the checkout binary, and waits2500ms. Later cases call the shell PAIR_BIN explicitly then supply the resolved identity. This is an observed asymmetry, not proof of an environment or timing cause. #586 concerns another review-observation fixture and may share this seam; keep separate until evidence links them.

Capture actionable evidence on the first failure: exact binary/root, resolver input/output/status/diagnostic, both fallback observations, reconstruction admission reason and Neovim messages/exit. Add a deterministic regression for the demonstrated cause and fix its owning boundary. No sleep, timeout increase, retry loop, skip or assertion weakening.

## Done when

- First-case failures retain enough evidence to distinguish wrong binary, incomplete/invalid observation and genuine unsupported history.
- A deterministic red/green regression verifies the demonstrated cause.
- Legacy empty-round reconstruction and scoped/ambiguous authorization assertions remain exact; isolated and loaded suite results are recorded.

## Plan

## Log

2026-10-15: observed while verifying #579; diagnostics IO source does not change nvim/review/init.lua, identity.lua or this fixture. Required run9c8da6fdf45067e5d6386cddc8344aea waited804s for quota then failed after3m26s before broad Go. Full log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/9c8da6fdf45067e5d6386cddc8344aea.log, SHA256 prefix1881fe7440d7. Exact first-case failures: no reconstruct (placed=false hl=0 diag=0), no change-highlights on resume, no diagnostics on resume. No isolated retry or source change made. Hub9a25379a-3879-4824-a314-9c396a4df10d requests filing unassigned and authorizes one unchanged recorded-suite rerun for #579; independent fixture fix waits for a free slot.
