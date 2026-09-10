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

## Open findings

- **BR-1** [Important] `probe-unparsed-reading-is-a-verdict` zellijwrapmargin reports WRAP SCROLLS (fixed) when its DSR/rows reading fails to parse
- **BR-2** [Important] `readme-install-surface-gate` README Install omits the new zellij 0.45.0 floor that pair term's strip requires
- **BR-3** [Important] `durable-doc-cites-unpublished-ref` atlas cites repro/223-bottom-strip-zellij-wrap, a tag that exists only in the local checkout
- **BR-4** [Minor] `superseded-premise-sweep` reserve.go, reserve_test.go and atlas:569-572 still say DECOM arbitration is what a top strip needs
- **BR-5** [Minor] `config-invariant-unpinned` No test pins pane_frame_style full in zellij/config.kdl
- **BR-6** [Minor] `version-scoped-workaround-tripwire` The zellij 0.45 upgrade trips the DELETE THIS marker on couchtty stripWheelResizeModifier
- **BR-7** [Minor] `stale-comment` Stale probe comments - PAIR_PROBE_EDGE mode list, writeLayout reference, 0.45.1 region reading
