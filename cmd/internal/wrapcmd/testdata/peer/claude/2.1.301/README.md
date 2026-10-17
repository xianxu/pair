# Claude Code 2.1.301 PR footer

`pr-footer.raw` is a minimal re-rendering of archived native terminal cells,
not a standalone slice of original VT bytes. Replay at **94 columns × 39 rows**.
Only composer/footer rows 34–37 (one-based) remain; unrelated content is cleared.
The visible cursor is zero-based (2,34). The original PR #387 is OSC8-linked.
All 376 selected cells (content, width and style) and cursor were verified against
full-prefix replay using the repository-pinned x/vt emulator.

Source: parked-scrollback-1-pair-8-20261017T024328 in local repo scope
990458a6956a804b; native banner Claude Code v2.1.301, launch ordinal 67,
generation be0b6a7cd7ee87d0de3e0dfe87b70c25. Replay prefix [0,32210830);
last synchronized update [32207890,32210763) is incremental, not self-contained.
The event log bounds this idle screen to 2026-10-17 00:12:56–02:42:27 PDT,
consistent with pair:6's approximately 02:13 rendered-text observation.

SHA256:
- Original raw: 900257bb7be631dea64469939d864c297b3f1b20dc1aebc1dfd84df561eaaa3e
- Capture witness: 551abe760a835726e30e267c203dbd85ce9e4e50aa0724a4ddb331c99f499be9
- Event log: 27969c1621449b9611614994d65b13d1b86d53a450c6afb273c2f20c6af1c533
- Last synchronized update: f9ebc9efb5a667ecc9290235508baf9740c686056fc5a774d3bcfdc06edb1a53
- Re-rendered fixture: f009b62d4ab8d404a7a9017b44300457a3773255aaae7ca8da70362437c1c7e6

Qualified: permission field, PR #N first optional field, shell counter and
management chrome. PR alone does not qualify a no-shell state. Malformed,
truncated, duplicate or misplaced PR fields stay unknown. This is display-only
background evidence; delivery admission still independently reads the composer.
