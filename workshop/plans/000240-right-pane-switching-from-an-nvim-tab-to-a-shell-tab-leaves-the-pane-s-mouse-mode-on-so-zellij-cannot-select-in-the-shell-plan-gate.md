---
gate: plan-quality
issue: 240
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-12T19:01:40-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Diff against m.hostScan (what the pane was shown), not a new "outgoing child" record
          detail: applyTakeover (run.go:994) only receives the incoming child; hostScan is already fed exactly what the pane is shown (run.go:936-956, 1040). Read it before the reset at run.go:1000. The outgoing-child design also breaks at removeTab (child already Closed, run.go:1450) and clearTab (nil incoming, run.go:1757, which is what Alt+t new-tab does). ARCH-DRY, ARCH-ORDER.
          family: pane-state-source-of-truth
          round: 1
        - id: PQ-2
          severity: Important
          title: 1000/1002/1003 are one variable in xterm and zellij; model a tagged enum + SGR bool, not a set
          detail: ?1000l turns tracking off regardless of which mode was set, so a set-diff emits wrong bytes after ?1002h,?1000l. Promote screen.go:606's bool to {Off,Normal,Button,Any}; delta is one h or one l per axis. Clear it in the RIS arm at screen.go:510. ARCH-ORDER.
          family: mode-model-matches-terminal
          round: 1
        - id: PQ-3
          severity: Minor
          title: Name the pure delta function and its test strategy
          detail: e.g. mouseReconcile(held, want) []byte in termcmd; test by generating the full held x want product rather than hand-listing cases. ARCH-PURE.
          family: named-pure-seam
          round: 1
        - id: PQ-4
          severity: Minor
          title: The tab-switch mode probe is not in the tree; "re-run the probe" is not executable
          detail: probes/termsmoke checks content markers only. Land the probe under cmd/probes or extend termsmoke to log DECSET/DECRST, and say how the "ring no longer holds the startup DECSET" precondition is produced.
          family: probe-in-tree
          round: 1
        - id: PQ-5
          severity: Minor
          title: State the compose-time read vs ring-snapshot race in one sentence, and fix the hostty comment
          detail: Modes are read on the writer goroutine at compose time and later live chunks are written verbatim, so the pane converges. hostty/repaint.go:62-66 currently says mouse is deliberately not asserted; note termcmd's reconciliation there.
          family: ordering-stated
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-12T19:07:35-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: held is read from m.hostScan before the reset (run.go:1000-1004); removeTab and clearTab both reach applyTakeover.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: mouseTracking tagged enum + sgrMouse bit (screen.go:129-174); RIS arm clears it (screen.go:552).
          round: 2
        - id: PQ-3
          disposition: addressed
          note: mouseReconcile(held, want) at run.go:1767; full 8x8 product test with Screen as oracle.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: probes/mousemodesmoke in tree; the precondition question is moot if the asserted/replayed distinguisher is fixed (see new finding).
          round: 2
        - id: PQ-5
          disposition: addressed
          note: hostty/repaint.go:69-73 names termcmd's reconciliation; the hostScan-feed ordering is stated in Spec bullet 2.
          round: 2
      findings:
        - id: PQ-6
          severity: Important
          title: Done-when claims the reconcile prefix is a grouped DECSET, but the Spec's one-write-per-axis rule makes it byte-identical to nvim's replayed startup bytes
          detail: 'mouseReconcile (run.go:1774-1781) emits ?1002h then ?1006h separately, so the probe''s modes() output cannot tell asserted from replayed on the switch back to nvim, and probes/mousemodesmoke/main.go:9-11 repeats the false claim. The mux test already uses position before HomeAndClear (mousemode_test.go:96-106). Make plan, probe and test agree: have the probe report modes before vs after HomeAndClear per switch and rewrite the Done-when bullet to that oracle (or, if grouping is preferred, state it in the Spec as a change to the per-axis rule). ARCH-PURPOSE.'
          family: acceptance-oracle-matches-design
          round: 2
        - id: PQ-7
          severity: Minor
          title: hostScan comments predating the mouse prefix are now false
          detail: run.go:796 says hostScan is fed child bytes only; run.go:1053-1054 says the composed prefix is HomeAndClear alone and "changes no behaviour". Both are contradicted by the reconcile prefix; the block at run.go:1046-1050 already explains the suspension, so point the two stale sentences at it.
          family: comment-matches-code
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-12T19:09:12-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: addressed
          note: Done-when now uses a count oracle (DECSET twice vs once on the way back, DECRST vs nothing on the way there) that matches mouseReconcile's per-axis writes; probe header and mux test (position before HomeAndClear) agree with it.
          round: 3
        - id: PQ-7
          disposition: not-addressed
          note: run.go:796 ("CHILD bytes ONLY") and run.go:1053-1054 ("HomeAndClear alone ... changes no behaviour") are still in the tree and still contradicted by the reconcile prefix fed at run.go:1057. Minor; fix in the implementation commit.
          round: 3
      blocked: false
content_hash: 051c1eef19cd6890ee051350a226ef8bffc8a24d81e49aaaf2a10cc38626bcb1
---

# Gate ledger — pair#240 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T19:01:40-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `pane-state-source-of-truth` Diff against m.hostScan (what the pane was shown), not a new "outgoing child" record
  applyTakeover (run.go:994) only receives the incoming child; hostScan is already fed exactly what the pane is shown (run.go:936-956, 1040). Read it before the reset at run.go:1000. The outgoing-child design also breaks at removeTab (child already Closed, run.go:1450) and clearTab (nil incoming, run.go:1757, which is what Alt+t new-tab does). ARCH-DRY, ARCH-ORDER.
- **PQ-2** [Important] `mode-model-matches-terminal` 1000/1002/1003 are one variable in xterm and zellij; model a tagged enum + SGR bool, not a set
  ?1000l turns tracking off regardless of which mode was set, so a set-diff emits wrong bytes after ?1002h,?1000l. Promote screen.go:606's bool to {Off,Normal,Button,Any}; delta is one h or one l per axis. Clear it in the RIS arm at screen.go:510. ARCH-ORDER.
- **PQ-3** [Minor] `named-pure-seam` Name the pure delta function and its test strategy
  e.g. mouseReconcile(held, want) []byte in termcmd; test by generating the full held x want product rather than hand-listing cases. ARCH-PURE.
- **PQ-4** [Minor] `probe-in-tree` The tab-switch mode probe is not in the tree; "re-run the probe" is not executable
  probes/termsmoke checks content markers only. Land the probe under cmd/probes or extend termsmoke to log DECSET/DECRST, and say how the "ring no longer holds the startup DECSET" precondition is produced.
- **PQ-5** [Minor] `ordering-stated` State the compose-time read vs ring-snapshot race in one sentence, and fix the hostty comment
  Modes are read on the writer goroutine at compose time and later live chunks are written verbatim, so the pane converges. hostty/repaint.go:62-66 currently says mouse is deliberately not asserted; note termcmd's reconciliation there.

## Round 2 — 2026-09-12T19:07:35-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — held is read from m.hostScan before the reset (run.go:1000-1004); removeTab and clearTab both reach applyTakeover.
- PQ-2 — addressed — mouseTracking tagged enum + sgrMouse bit (screen.go:129-174); RIS arm clears it (screen.go:552).
- PQ-3 — addressed — mouseReconcile(held, want) at run.go:1767; full 8x8 product test with Screen as oracle.
- PQ-4 — addressed — probes/mousemodesmoke in tree; the precondition question is moot if the asserted/replayed distinguisher is fixed (see new finding).
- PQ-5 — addressed — hostty/repaint.go:69-73 names termcmd's reconciliation; the hostScan-feed ordering is stated in Spec bullet 2.

### Raised

- **PQ-6** [Important] `acceptance-oracle-matches-design` Done-when claims the reconcile prefix is a grouped DECSET, but the Spec's one-write-per-axis rule makes it byte-identical to nvim's replayed startup bytes
  mouseReconcile (run.go:1774-1781) emits ?1002h then ?1006h separately, so the probe's modes() output cannot tell asserted from replayed on the switch back to nvim, and probes/mousemodesmoke/main.go:9-11 repeats the false claim. The mux test already uses position before HomeAndClear (mousemode_test.go:96-106). Make plan, probe and test agree: have the probe report modes before vs after HomeAndClear per switch and rewrite the Done-when bullet to that oracle (or, if grouping is preferred, state it in the Spec as a change to the per-axis rule). ARCH-PURPOSE.
- **PQ-7** [Minor] `comment-matches-code` hostScan comments predating the mouse prefix are now false
  run.go:796 says hostScan is fed child bytes only; run.go:1053-1054 says the composed prefix is HomeAndClear alone and "changes no behaviour". Both are contradicted by the reconcile prefix; the block at run.go:1046-1050 already explains the suspension, so point the two stale sentences at it.

## Round 3 — 2026-09-12T19:09:12-07:00 (claude) — passed

### Disposed

- PQ-6 — addressed — Done-when now uses a count oracle (DECSET twice vs once on the way back, DECRST vs nothing on the way there) that matches mouseReconcile's per-axis writes; probe header and mux test (position before HomeAndClear) agree with it.
- PQ-7 — not-addressed — run.go:796 ("CHILD bytes ONLY") and run.go:1053-1054 ("HomeAndClear alone ... changes no behaviour") are still in the tree and still contradicted by the reconcile prefix fed at run.go:1057. Minor; fix in the implementation commit.

## Open findings

- **PQ-7** [Minor] `comment-matches-code` hostScan comments predating the mouse prefix are now false
