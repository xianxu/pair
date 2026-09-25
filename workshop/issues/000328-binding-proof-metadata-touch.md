---
id: 000328
status: working
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
started: 2026-09-25T10:12:32-07:00
flow: {kind: quick, provenance: inferred, spec: "1e989743", done: "8c8d8ad9"}
---

# Relaunch rejects a binding after metadata-only change

## Problem

Operator repro (tools thread, 2026-09-25):
1. Relaunch a working thread with alt+n. The past transcript loads correctly.
2. Detach Couch, then start it again.
3. Relaunch the same thread. It's refused with
   `resume-binding-provisional: its agent has not completed a turn yet`.

The binding is there. The 10:02 relaunch wrote launch 7 and, in the same
step, binding 8 to `20d98988…`, with a proof. The proof snapshots the
transcript as it was when the launch was prepared: inode `195338098`, size
`2559906`, ctime `1790355748`. Three seconds later claude, now resuming,
changed the file's metadata but not its contents: same inode, same size, ctime
`1790355751`.

`ValidateBindingProof` then fails with `ErrArtifactChanged`:
- `AdvanceTargetValidation` treats a same-size fingerprint with a changed
  mutation token as a rewrite (`incremental_inventory.go:389`).
- The fallback that re-reads from byte zero runs only if the file grew
  (`proofAllowsFullGrowthRevalidation`, `query.go:241`, returns
  `grew && missingGeneration`).

`QuerySessionContext` then returns with its status still `provisional` and no
diagnostic (`query.go:138`). So "the proof check failed" reaches the operator
as "no turn yet". A turn "fixes" it only because growth unlocks the fallback.
Detach isn't involved: a relaunch straight after a relaunch has the same
exposure.

## Spec

- **Revalidate by content, not metadata.** When the proof-named transcript is
  still the same file (same stable file ID, generation rules as today) and is
  **not smaller** than the proof, a failed incremental advance falls back to
  the full from-byte-zero revalidation. That covers growth (as today) and
  equal size with changed metadata (new). The re-read still requires exactly
  one validated root with the proof's native ID, agent, role and scanner
  schema. So a same-size rewrite that changed the conversation still fails.
  Metadata stays a cheap "nothing changed, skip the read" shortcut, never a
  revocation.
- **Record why a proof failed.** When `ValidateBindingProof` errors, the query
  appends a `binding_stale` diagnostic naming the error. It goes in the
  structured query result only: the UI refusal text is unchanged (operator:
  "in debugging log, not on UI"). Nothing new is printed on the `--owner` CLI,
  because `nvim/init.lua:838` captures its stdout and stderr together.
- Artifacts (ARCH): this creates nothing durable. The fallback's successful
  validation is published to the existing session-inventory catalog the same
  way the growth fallback already is.

## Done when

- A test: a proof-backed binding whose transcript gets a metadata-only change
  (same bytes, new ctime) still queries as `established` with the right root.
  Its sibling test proves a same-size rewrite to a different conversation does
  NOT establish, and carries a `binding_stale` diagnostic.
- Before the fix, the metadata-touch test fails.
- `go test ./cmd/internal/sessioninventory/...` passes, and the full
  `make test` is green.
- Live check: the tools thread queries `established` again (the scratch probe
  that reproduced `artifact changed during observation`), and the operator
  confirms that alt+n relaunch works after relaunch → detach → reattach.

## Plan

- [ ] Tests first: metadata touch establishes; same-size different
      conversation refuses with a diagnostic.
- [ ] Relax `proofAllowsFullGrowthRevalidation` from grew-only to not-smaller,
      and rename it to match.
- [ ] Append a `binding_stale` diagnostic on proof validation failure in
      `QuerySessionContext`.
- [ ] Verify live against the tools thread, then run `make test`.

## Log

### 2026-09-25

- Diagnosed live. The scratch probe (QuerySession, then ValidateBindingProof,
  against `repos/434128d5ad68b26e`) gave `validate err: session inventory
  artifact changed during observation`, with the transcript's size unchanged
  and its ctime +3 s after the proof.
- Split out #329: detach kills an unbound session watcher. Same refusal text,
  different cause (the `ariadne` threads have no binding at all).
