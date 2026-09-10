---
gate: boundary-review
issue: 223
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-10T10:00:51-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: zellijwrapmargin reports WRAP SCROLLS (fixed) when its DSR/rows reading fails to parse
          detail: wrapEscaped (main.go:202-213) ignores Sscanf failures and the switch default (main.go:190-197) prints the fixed verdict; scratch-verified for an empty CPR reply, a failed tput, a missing wrap line and a wrap row above the margin. Return (wrapRow, bottom, ok), route !ok or unexpected rows to PROBE-INCONCLUSIVE exit 2 as zellijscrollregion does, and add a table test.
          family: probe-unparsed-reading-is-a-verdict
          round: 1
        - id: BR-2
          severity: Important
          title: README Install omits the new zellij 0.45.0 floor that pair term's strip requires
          detail: atlas/architecture.md:582 and zellij/config.kdl:51 state the requirement, but README.md:255 only says brew installs zellij if absent; an existing 0.44.x zellij is kept and nothing warns (no doctor floor, and the probe exits 0).
          family: readme-install-surface-gate
          round: 1
        - id: BR-3
          severity: Important
          title: atlas cites repro/223-bottom-strip-zellij-wrap, a tag that exists only in the local checkout
          detail: git ls-remote --tags origin lists 48 tags and none match repro; push.followTags is unset. Push the tag (operator-confirmed) or cite commit 0baacfa7, which is on origin/main, at atlas/architecture.md:590-591.
          family: durable-doc-cites-unpublished-ref
          round: 1
        - id: BR-4
          severity: Minor
          title: reserve.go, reserve_test.go and atlas:569-572 still say DECOM arbitration is what a top strip needs
          detail: '#223 measured that DECSTBM params are absolute even under DECOM, so a top strip also needs the child''s DECSTBM rewritten in flight; the new atlas text says reserve.go records this, but reserve.go:34-36 and its error message at :67 carry only the older premise.'
          family: superseded-premise-sweep
          round: 1
        - id: BR-5
          severity: Minor
          title: No test pins pane_frame_style full in zellij/config.kdl
          detail: Removing it silently brings back the first-line-vanished bug on zellij 0.45; a sibling of TestEveryTerminalPaneRungIsBorderless would pin it cheaply.
          family: config-invariant-unpinned
          round: 1
        - id: BR-6
          severity: Minor
          title: The zellij 0.45 upgrade trips the DELETE THIS marker on couchtty stripWheelResizeModifier
          detail: zellij 0.45.1 dump-config carries mouse_scroll_resize (mouse.go:92-95 names it as the deletion trigger); deleting the filter depends on 0.45 becoming a real floor, so file a follow-up.
          family: version-scoped-workaround-tripwire
          round: 1
        - id: BR-7
          severity: Minor
          title: Stale probe comments - PAIR_PROBE_EDGE mode list, writeLayout reference, 0.45.1 region reading
          detail: main.go:99 lists only wrap, plain and decom modes; layout.kdl:4 names writeLayout, now zellijprobe.WriteLayout; the recorded 0.45.1 region 1..23 predates the full-frame config. Also probe.sh is mode 100644 while probe_top.sh is 100755.
          family: stale-comment
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-10T10:19:08-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-checked in a scratch copy: the old parser fails 6 of 8 subtests, a variant with no default branch fails 2; make test''s go test ./... reaches the test.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Floor stated at README.md:216 and :257; the Homebrew reason it gives is wrong, raised as the new prose-claim finding.
          round: 2
        - id: BR-3
          disposition: addressed
          note: 0baacfa7 is reachable from origin/main (the tag is still local-only, correctly not cited); the atlas's description of the commit is wrong, raised in the new finding.
          round: 2
        - id: BR-4
          disposition: addressed
          note: reserve.go comment, error message, test name and atlas:570-574 now give the DECSTBM-absolute reason; no stale DECOM-arbitration premise left outside workshop/.
          round: 2
        - id: BR-5
          disposition: addressed
          note: 'Mutation-checked: key removed, commented out, or missing from only the runtime-bundle copy - all three fail TestConfigStatesFullPaneFrames.'
          round: 2
        - id: BR-6
          disposition: addressed
          note: Filed as 226 with the blocker (0.45 not enforced) stated; grep finds no other DELETE THIS tripwire.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: 'Four named sites fixed; the copied sibling round 1 named, probes/zellijscrollregion/layout.kdl:3-4, still says see writeLayout there. Rule: when fixing a copied comment, grep the copied phrase across the tree in the same commit.'
          round: 2
      findings:
        - id: BR-8
          severity: Minor
          title: Fix-round prose states four checkable facts that git, the Homebrew source or the probe contradicts or does not support
          detail: '(1) README.md:257-258 says brew install skips a present dependency; for pair''s bottle-less formula Homebrew treats an outdated dep as unsatisfied (dependency.rb:82-84) and upgrades it (formula_installer.rb:951) - the real gaps are a non-brew zellij earlier on PATH and a pair that brew upgrade finds current. (2) atlas/architecture.md:594 calls 0baacfa7 the last commit before the frame-style change; 5fdaf32e^ is 4e62e04a. (3) reserve.go:40 (first half-page) and atlas:573/:599 (at once): the nvim mode takes one dump-screen after six keystroke groups (main.go:159-189), and the Log records only that end state. (4) paneframestyle_test.go:18-19 says TestEveryTerminalPaneRungIsBorderless reads the same files; it reads layouts/main-3.kdl. Rule: run the check (git, tool source, the probe''s actual reading) before committing a checkable claim.'
          family: prose-claim-unchecked-against-source
          round: 2
        - id: BR-9
          severity: Minor
          title: atlas states the top-edge DECSTBM-under-DECOM measurement twice, at architecture.md:570-574 and :596-601
          detail: BR-4's sweep added it to the Edge asymmetry paragraph, and ef8cdb34's strip-stays-at-the-BOTTOM paragraph already said nearly the same, including real nvim overwrote it at once, so correcting that phrase now needs two edits. Keep one copy and point to it (ARCH-DRY).
          family: doc-fact-single-home
          round: 2
      blocked: false
---

# Gate ledger — pair#223 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T10:00:51-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `probe-unparsed-reading-is-a-verdict` zellijwrapmargin reports WRAP SCROLLS (fixed) when its DSR/rows reading fails to parse
  wrapEscaped (main.go:202-213) ignores Sscanf failures and the switch default (main.go:190-197) prints the fixed verdict; scratch-verified for an empty CPR reply, a failed tput, a missing wrap line and a wrap row above the margin. Return (wrapRow, bottom, ok), route !ok or unexpected rows to PROBE-INCONCLUSIVE exit 2 as zellijscrollregion does, and add a table test.
- **BR-2** [Important] `readme-install-surface-gate` README Install omits the new zellij 0.45.0 floor that pair term's strip requires
  atlas/architecture.md:582 and zellij/config.kdl:51 state the requirement, but README.md:255 only says brew installs zellij if absent; an existing 0.44.x zellij is kept and nothing warns (no doctor floor, and the probe exits 0).
- **BR-3** [Important] `durable-doc-cites-unpublished-ref` atlas cites repro/223-bottom-strip-zellij-wrap, a tag that exists only in the local checkout
  git ls-remote --tags origin lists 48 tags and none match repro; push.followTags is unset. Push the tag (operator-confirmed) or cite commit 0baacfa7, which is on origin/main, at atlas/architecture.md:590-591.
- **BR-4** [Minor] `superseded-premise-sweep` reserve.go, reserve_test.go and atlas:569-572 still say DECOM arbitration is what a top strip needs
  #223 measured that DECSTBM params are absolute even under DECOM, so a top strip also needs the child's DECSTBM rewritten in flight; the new atlas text says reserve.go records this, but reserve.go:34-36 and its error message at :67 carry only the older premise.
- **BR-5** [Minor] `config-invariant-unpinned` No test pins pane_frame_style full in zellij/config.kdl
  Removing it silently brings back the first-line-vanished bug on zellij 0.45; a sibling of TestEveryTerminalPaneRungIsBorderless would pin it cheaply.
- **BR-6** [Minor] `version-scoped-workaround-tripwire` The zellij 0.45 upgrade trips the DELETE THIS marker on couchtty stripWheelResizeModifier
  zellij 0.45.1 dump-config carries mouse_scroll_resize (mouse.go:92-95 names it as the deletion trigger); deleting the filter depends on 0.45 becoming a real floor, so file a follow-up.
- **BR-7** [Minor] `stale-comment` Stale probe comments - PAIR_PROBE_EDGE mode list, writeLayout reference, 0.45.1 region reading
  main.go:99 lists only wrap, plain and decom modes; layout.kdl:4 names writeLayout, now zellijprobe.WriteLayout; the recorded 0.45.1 region 1..23 predates the full-frame config. Also probe.sh is mode 100644 while probe_top.sh is 100755.

## Round 2 — 2026-09-10T10:19:08-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Mutation-checked in a scratch copy: the old parser fails 6 of 8 subtests, a variant with no default branch fails 2; make test's go test ./... reaches the test.
- BR-2 — addressed — Floor stated at README.md:216 and :257; the Homebrew reason it gives is wrong, raised as the new prose-claim finding.
- BR-3 — addressed — 0baacfa7 is reachable from origin/main (the tag is still local-only, correctly not cited); the atlas's description of the commit is wrong, raised in the new finding.
- BR-4 — addressed — reserve.go comment, error message, test name and atlas:570-574 now give the DECSTBM-absolute reason; no stale DECOM-arbitration premise left outside workshop/.
- BR-5 — addressed — Mutation-checked: key removed, commented out, or missing from only the runtime-bundle copy - all three fail TestConfigStatesFullPaneFrames.
- BR-6 — addressed — Filed as 226 with the blocker (0.45 not enforced) stated; grep finds no other DELETE THIS tripwire.
- BR-7 — not-addressed — Four named sites fixed; the copied sibling round 1 named, probes/zellijscrollregion/layout.kdl:3-4, still says see writeLayout there. Rule: when fixing a copied comment, grep the copied phrase across the tree in the same commit.

### Raised

- **BR-8** [Minor] `prose-claim-unchecked-against-source` Fix-round prose states four checkable facts that git, the Homebrew source or the probe contradicts or does not support
  (1) README.md:257-258 says brew install skips a present dependency; for pair's bottle-less formula Homebrew treats an outdated dep as unsatisfied (dependency.rb:82-84) and upgrades it (formula_installer.rb:951) - the real gaps are a non-brew zellij earlier on PATH and a pair that brew upgrade finds current. (2) atlas/architecture.md:594 calls 0baacfa7 the last commit before the frame-style change; 5fdaf32e^ is 4e62e04a. (3) reserve.go:40 (first half-page) and atlas:573/:599 (at once): the nvim mode takes one dump-screen after six keystroke groups (main.go:159-189), and the Log records only that end state. (4) paneframestyle_test.go:18-19 says TestEveryTerminalPaneRungIsBorderless reads the same files; it reads layouts/main-3.kdl. Rule: run the check (git, tool source, the probe's actual reading) before committing a checkable claim.
- **BR-9** [Minor] `doc-fact-single-home` atlas states the top-edge DECSTBM-under-DECOM measurement twice, at architecture.md:570-574 and :596-601
  BR-4's sweep added it to the Edge asymmetry paragraph, and ef8cdb34's strip-stays-at-the-BOTTOM paragraph already said nearly the same, including real nvim overwrote it at once, so correcting that phrase now needs two edits. Keep one copy and point to it (ARCH-DRY).

## Open findings

- **BR-7** [Minor] `stale-comment` Stale probe comments - PAIR_PROBE_EDGE mode list, writeLayout reference, 0.45.1 region reading
- **BR-8** [Minor] `prose-claim-unchecked-against-source` Fix-round prose states four checkable facts that git, the Homebrew source or the probe contradicts or does not support
- **BR-9** [Minor] `doc-fact-single-home` atlas states the top-edge DECSTBM-under-DECOM measurement twice, at architecture.md:570-574 and :596-601
