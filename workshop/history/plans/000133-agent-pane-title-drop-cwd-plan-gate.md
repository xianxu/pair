---
gate: plan-quality
issue: 133
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-07-29T15:51:24-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: KDL printf edit has no mechanical guard; Done-when's cwd round-trip proof is false
          detail: |-
            Done-when credits contextcmd_test.go:21 / dispatcher_test.go:273 /
            helper_equivalence_test.go:98 with proving the raw cwd round-trip, but all
            three hand-write the pane JSON and never execute main-2.kdl:45 /
            main-3.kdl:54. Dropping the cwd_display %s without its argument makes printf
            recycle the format and emit two concatenated objects; Unmarshal then fails,
            the poller skips the pane and contextcmd.paneCwd returns "" — silently, with
            make test green. Name the guard (extract the agent-pane args "-c" line, run
            it under sh in a temp PAIR_DATA_DIR, assert pane_id+cwd decode), or at
            minimum assert the file parses at the live check. ARCH-MOCK: the producer
            sits outside any seam with no fake and no conformance check. Note
            tests/term-pane-shortcuts-test.sh:210-214 enforces layout2==layout3 but
            passes when both are equally wrong.
          round: 1
        - id: PQ-2
          severity: Minor
          title: PaneInfo.Cwd and its decode go dead with the fallback; plan names only opts.Home
          detail: |-
            pane.Cwd's only non-test reader is run.go:194, so deleting the cwdDisp block
            orphans PaneInfo.Cwd (run.go:31) and runtime.go:73/:84. Go does not error on
            unused struct fields, so nothing catches it. The Home thread also reaches
            Options.Home (run.go:15), runcli.go:34 and run_test.go:91. The JSON cwd key
            itself correctly stays.
          round: 1
        - id: PQ-3
          severity: Minor
          title: PaneTitle becomes an identity function — plan does not say whether it survives
          detail: |-
            After dropping the suffix, PaneTitle(agent, cwd, home) returns agent with two
            unused params, and TestPaneTitle asserts identity. The ARCH-DRY argument the
            Revisions used against TildeAbbrev applies: consider inlining
            rt.SetEnv("PAIR_PANE_TITLE", agent) at createflow.go:468 and deleting both.
          round: 1
        - id: PQ-4
          severity: Minor
          title: Atlas step should name the specific stale lines
          detail: |-
            architecture.md:274 lists "cwd abbrev" among the poller's pure decisions;
            :276 documents the frame as "<agent> (<count>) [<cwd>]", the record as
            {pane_id, cwd, cwd_display}, and still says main.kdl. :278's
            cmd/internal/titlefmt reference is pre-existing #130 drift — fix as a
            side-quest or leave, but say which.
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-07-29T15:56:45-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Plan item 1 is now an executable producer conformance test written to pass pre-edit; Done-when names it as the only accepted proof and pins the one-JSON-object invariant.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Item 3 now follows PaneInfo.Cwd, runtime.go:73-74/:84-85 and Options.Home/runcli.go:34/run_test.go:91, and notes Go will not flag the unused field.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Item 5 deletes PaneTitle plus its test and inlines rt.SetEnv at createflow.go:468, citing the same ARCH-DRY reasoning.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Atlas step names :274/:276/:278 with the titlefmt drift split out as an explicit side-quest bullet.
          round: 2
      findings:
        - id: PQ-5
          severity: Minor
          title: KDL edit has four files, not two — the generated runtimebundle copies must be regenerated and committed
          detail: zellij/layouts/main-{2,3}.kdl is the source; cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-{2,3}.kdl are committed generated copies (Makefile.local:92-95) embedded via embed.go:8, and they carry the same printf at the same line numbers. make test regenerates them via the bin/pair prereq, but nothing fails if they are left uncommitted — runtimebundle-drift-check only compares two fresh generations, and embed_test.go never diffs the embedded copy against the tree. Say in item 4 that the source is zellij/layouts/ and that the regenerated assets get committed, and widen the Done-when grep to cover cwd_display and PAIR_PANE_CWD tree-wide.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-07-29T16:03:38-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: addressed
          note: Source-vs-generated now named in item 4 and the Done-when grep covers the assets tree with --no-ignore; the plan correctly refuted my "committed" claim — .gitignore:34 blanket-ignores the dir and make build regenerates it via PAIR_GO_SRCS.
          round: 3
      blocked: false
content_hash: 040fd28c97456b406b5c3d9cfed93a3325705da72d884b60447c36e446b4aa85
---

# Gate ledger — pair#133 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-07-29T15:51:24-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] KDL printf edit has no mechanical guard; Done-when's cwd round-trip proof is false
  Done-when credits contextcmd_test.go:21 / dispatcher_test.go:273 /
  helper_equivalence_test.go:98 with proving the raw cwd round-trip, but all
  three hand-write the pane JSON and never execute main-2.kdl:45 /
  main-3.kdl:54. Dropping the cwd_display %s without its argument makes printf
  recycle the format and emit two concatenated objects; Unmarshal then fails,
  the poller skips the pane and contextcmd.paneCwd returns "" — silently, with
  make test green. Name the guard (extract the agent-pane args "-c" line, run
  it under sh in a temp PAIR_DATA_DIR, assert pane_id+cwd decode), or at
  minimum assert the file parses at the live check. ARCH-MOCK: the producer
  sits outside any seam with no fake and no conformance check. Note
  tests/term-pane-shortcuts-test.sh:210-214 enforces layout2==layout3 but
  passes when both are equally wrong.
- **PQ-2** [Minor] PaneInfo.Cwd and its decode go dead with the fallback; plan names only opts.Home
  pane.Cwd's only non-test reader is run.go:194, so deleting the cwdDisp block
  orphans PaneInfo.Cwd (run.go:31) and runtime.go:73/:84. Go does not error on
  unused struct fields, so nothing catches it. The Home thread also reaches
  Options.Home (run.go:15), runcli.go:34 and run_test.go:91. The JSON cwd key
  itself correctly stays.
- **PQ-3** [Minor] PaneTitle becomes an identity function — plan does not say whether it survives
  After dropping the suffix, PaneTitle(agent, cwd, home) returns agent with two
  unused params, and TestPaneTitle asserts identity. The ARCH-DRY argument the
  Revisions used against TildeAbbrev applies: consider inlining
  rt.SetEnv("PAIR_PANE_TITLE", agent) at createflow.go:468 and deleting both.
- **PQ-4** [Minor] Atlas step should name the specific stale lines
  architecture.md:274 lists "cwd abbrev" among the poller's pure decisions;
  :276 documents the frame as "<agent> (<count>) [<cwd>]", the record as
  {pane_id, cwd, cwd_display}, and still says main.kdl. :278's
  cmd/internal/titlefmt reference is pre-existing #130 drift — fix as a
  side-quest or leave, but say which.

## Round 2 — 2026-07-29T15:56:45-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Plan item 1 is now an executable producer conformance test written to pass pre-edit; Done-when names it as the only accepted proof and pins the one-JSON-object invariant.
- PQ-2 — addressed — Item 3 now follows PaneInfo.Cwd, runtime.go:73-74/:84-85 and Options.Home/runcli.go:34/run_test.go:91, and notes Go will not flag the unused field.
- PQ-3 — addressed — Item 5 deletes PaneTitle plus its test and inlines rt.SetEnv at createflow.go:468, citing the same ARCH-DRY reasoning.
- PQ-4 — addressed — Atlas step names :274/:276/:278 with the titlefmt drift split out as an explicit side-quest bullet.

### Raised

- **PQ-5** [Minor] KDL edit has four files, not two — the generated runtimebundle copies must be regenerated and committed
  zellij/layouts/main-{2,3}.kdl is the source; cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-{2,3}.kdl are committed generated copies (Makefile.local:92-95) embedded via embed.go:8, and they carry the same printf at the same line numbers. make test regenerates them via the bin/pair prereq, but nothing fails if they are left uncommitted — runtimebundle-drift-check only compares two fresh generations, and embed_test.go never diffs the embedded copy against the tree. Say in item 4 that the source is zellij/layouts/ and that the regenerated assets get committed, and widen the Done-when grep to cover cwd_display and PAIR_PANE_CWD tree-wide.

## Round 3 — 2026-07-29T16:03:38-07:00 (claude) — passed

### Disposed

- PQ-5 — addressed — Source-vs-generated now named in item 4 and the Done-when grep covers the assets tree with --no-ignore; the plan correctly refuted my "committed" claim — .gitignore:34 blanket-ignores the dir and make build regenerates it via PAIR_GO_SRCS.

## Open findings

(none — every finding has been disposed)
