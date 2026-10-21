---
id: 000748
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line

## Problem

`couchcore.TestRecoveryRealHelperAndSessionConformanceLive/warm` in the required `couch-zellij-conformance` job fails
intermittently on GitHub macOS: "recovery session state could not be checked; retry: observe live session owner:
zellij action list-panes --json --command returned no reply" (helper exited before recovery; the final query got no
reply). It failed on pair#706's heads twice on 10-20 (run 39501347923 attempt 1; run 39521409698 job 118965862227,
log SHA256 cd823c3894b2ef291a8c4ae4be3b0c70c062aa9ada5fa3d8b436538e393c4f69) and passed on rerun once; #706 touches
no couchcore, launcher or zellij code. Registered in pair#636. Because conformance is required, each occurrence
holds the pair landing line.
