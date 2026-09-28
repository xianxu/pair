---
id: '000328'
status: done
started: 2026-09-25T10:12:32-07:00
created: 2026-09-25
updated: 2026-09-25
actual_hours: N/A
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
