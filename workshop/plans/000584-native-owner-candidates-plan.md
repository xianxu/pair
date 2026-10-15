# Native owner candidate and legacy proof implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Main session retains the measured proof context; SDLC owns the fresh-context boundary review.

**Goal:** Restore native Resume from disk on the existing fleet without admitting two live writers.

**Architecture:** Select possible wrappers before demanding argv. Preserve the modern exact-identity path. For a legacy ready record, combine exact owner argv, direct live-child kinship, readiness ordinal and owner ledger identity; a different ledger ID is a negative collision exclusion. A matching ID requires writable held transcript proof before the ordinary source exemption can apply. Never infer from cwd, activity or text.

**Tech Stack:** Go, native procutil snapshots, existing ledger and native filename codecs, bounded read-only lsof.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Candidate filter / observation retry | launcher/recovery_native_owner.go | modified |
| Legacy collision exclusion / source proof | launcher/recovery_native_legacy.go | new |
| Native transcript filename identity | sessioninventory/target.go | modified (reuse existing codecs) |
| Held writable file output parser | procutil/procutil.go | new |

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Recovery native owner observation | launcher/recovery_native_owner.go | modified | existing process-table and metadata IO |
| Legacy held-file reader | launcher/recovery_native_legacy.go | new | procutil lsof and native filename identity |
| Context-bounded writable-file probe | procutil/procutil.go | new | lsof; fake executable in tests |

## Proof and limits

- Candidate kernel names are pair, pair-wrap and unavailable names. Known unrelated names and kernel-confirmed zombies are not candidates. Missing candidate argv/identity triggers at most one full re-observation under the existing five-second total budget. Wrapper-set churn is retried once; persistent uncertainty refuses.
- Modern readiness continues to require exact native child birth and ancestry. The legacy exception applies only to missing agent_identity, not an identity mismatch, corrupt readiness or unreadable ledger.
- The observed wrapper's command names the exact validated Pair storage owner. The legacy readiness must name a non-zombie child whose direct PPID is that wrapper, with nonempty current birth; capture and recheck these exact generations and edges before accepting proof.
- Read the exact owner ledger generation selected by readiness.launch_ordinal, never its latest unrelated launch. A nonempty, unambiguous different native ID excludes collision for that owner only. Other agent kinds are already excluded. Missing/conflicting IDs remain uncertainty only where collision cannot be excluded.
- A matching native ID remains protected. To exempt the exact source during preflight, observe its native process holding that ID's root transcript writable under the registered native storage root. Reuse existing filename grammar. For Codex, the ready child is a node shim; inspect only that child and its immediate native Codex engine (validated argv), not arbitrary shell/tool descendants. There can be multiple held subagent files: match the exact owner ledger ID, never pick the newest or infer by cwd.
- lsof is read-only, context-bounded (at most two seconds and within the shared five-second budget), with bounded output. Failure/partial output is uncertainty, never no-owner proof. Tests use a stateful fake table and fake held-file seam; host conformance observes current descriptors without launch/kill/record writes.
- On the second ownership observation, recheck the proof's child/engine generations and edges and held owner ID. Changed observations retry once or refuse. No writes or new durable artifacts are introduced by this scan (ARCH-FUNERAL).
- The legacy error names agent, exact storage owner and child PID; Couch adds the friendly slot when it can resolve it by a preview read. The next action is relaunch to migrate; this does not relax collision protection.
- ARCH-DRY: reuse ledger projection, native filename grammar and procutil ownership of lsof. ARCH-PURE: keep candidate/exclusion/parser logic pure, IO injected. ARCH-SECURE/ORDER: missing evidence never establishes absence; preserve exact-generation fences. ARCH-CONSTRAINTS: foreground admission, existing five-second envelope, one retry, no background work. ARCH-MOCK: fake table/file-holder evidence models both observations and changing parent/generation/file sets. No input delivery, startup recovery, agent defaults or schemas change.

## Tasks

- [x] Candidate/zombie/churn regressions fail on original code; minimal candidate fix passes native-owner tests.
- [x] Add red legacy tests: different-ID exclusion, matching-ID collision/source exemption, direct parent and birth changes, wrong owner/ordinal/held ID, read-only paths, no proof, other-agent exclusion and modern identity mismatch.
- [x] Factor existing exact-ordinal ledger projection once for modern, legacy and orphan consumers; add the injected legacy held-file check without weakening modern readiness.
- [x] Add bounded writable-file IO and reuse native filename codecs; fake command/parser tests and read-only host conformance.
- [x] Add exact-owner migration diagnostics, with friendly Couch slot resolution through preview reads where available; tests for known and unresolved owners.
- [ ] Update atlas and Log with justification, the 18-wrapper census, host evidence and limitations; targeted tests and build.
- [ ] Record the full suite on the pushed final code head; close once with the mandatory SDLC review; land on SHIP and report merge to hub. No live migration or resume from this slot.

## Revisions

- 2026-10-15: Initial small candidate repair expanded by the TL's approved legacy proof because ordinary operational migration is circularly blocked and 18 of 21 live wrappers are legacy. Exact held writable transcript plus kinship is positive proof; an exact different ordinal-ledger ID is the approved negative collision exclusion.

### 2026-10-15 16:02 — plan-gate refinements and approved exclusion

This revision supersedes the earlier test-case inventory and makes both admission consumers explicit (PQ-1/PQ-2).

- `observeLegacyNativeOrphans` runs first (`recovery_native_owner.go`, before wrapper ledger processing). Move exact attached-wrapper detection before its missing-AgentIdentity refusal. An attached legacy direct child proceeds to the shared wrapper proof path, not automatic admission. A true orphan still reads the exact readiness ordinal through shared `readRecoveryNativeID`; an exact different native ID excludes collision, while a same/unknown ID with no wrapper association refuses with owner migration advice. Modern identities and retained-generation comparisons keep their existing fences. No catch-and-ignore of unknown evidence.
- `recoveryNativeProofForWrapper` combines `recoveryWrapperReady` with the extracted `readRecoveryNativeID`. Normal readiness needs exact identity/ancestry as today. Legacy readiness needs a non-zombie direct child and wrapper generation from exact owner argv. The TL's 61032867 explicitly approves exclusion of a different requested native ID by that exact owner/ordinal ledger; matching IDs additionally require `checkLegacyHeldNative` before an exact source exemption. Proof carries the holder generation for the final observation comparison. Both the initial and final native checks use this same function.
- `checkRecoveryNativeOwnersWithDiscovery` / `observeLegacyNativeOrphans` / `recoveryNativeProofForWrapper`: adversarial stateful process-table and held-file sequences through the real gate pin allowed exclusions, collision refusals, exact source exemption, unknown evidence and generation/parent changes; assert no executable launch/kill effects.
- `parseLsofWritableNames` / `LsofWritableNames`: fuzz malformed/truncated field streams and use a fake executable to assert all-or-error results, requested-PID ownership, writable regular descriptors and hard context/output limits.
- `NativeIDForArtifact` (export the existing `observationNativeID` decision, without a parallel filename parser): table-driven malformed/other-root artifact grammar asserts identity only for the registered provider's root transcript filenames. `checkLegacyHeldNative` uses these identities only for exact owner-ledger matching, never a whole native inventory or latest scan.
- Legacy error formatting / Couch friendly-slot advice: pin exact owner fallback and preview-only friendly resolution; a broken catalogue must preserve the original refusal, not invent a slot.

Measured limitations remain explicit: the ready PID can be a node shim, so a positively identified immediate Codex engine is inspected too; arbitrary grandchildren/tools are not. Idle Claude holds no transcript, hence a different exact ledger ID excludes collision, while a matching/unknown one remains guarded. New wrappers do not use this legacy exception.

### 2026-10-15 — approved unbound legacy set and finite ledger read

TL receipt 5d2dcac0 authorizes an exact legacy launch with absent (not conflicting/malformed) ID to use the complete engine-held writable native ID set. This is a positive writer set from the proven wrapper/direct-child/native-engine lineage, decoded by the existing filename grammar. The same holder births, parent edges and sorted ID set are rechecked; a matching requested ID remains blocked except the exact source exemption, different IDs exclude that target, and partial/empty/unrecognized evidence remains unknown. No state is persisted and no cwd/tail evidence is used. The proof carries child birth even for ledger-based negative exclusions.

The host's valid 21 MiB ledger also requires #508's existing contract: no reader-only whole-ledger cutoff below the writer's contract. `readNativeOwnerMetadata` reads a finite size-observed ledger snapshot through the existing no-symlink, regular-file, cancellation and unchanged-generation reader; small readiness files retain their fixed bound. A >8 MiB actual-file regression pins this path.

### 2026-10-15 — modern absent-only held proof

TL approved fcda041b: apply the complete held-ID-set fallback to modern owners only after full readiness identity and ancestry checks. Conflicting/malformed IDs and identity mismatches remain refusals. The host cold-history decoding cost currently exceeds the existing admission budget; measure a codec repair before any validation-scope or timing change.

### 2026-10-15 — approved equivalent codec performance repair

TL approved bc20aff9 after repeated cold-host checks exposed insufficient margin (codec-only passes 4.321/4.868 s, two runs exhausted 5 s). Keep the exact whole-ledger validation scope and five-second budget. Replace Decoder.Token duplicate traversal in shared strictjson with json.Valid plus key-only duplicate checks; preserve float-overflow rejection, Unicode/escaped key equality, nested duplicates and trailing-data refusal. Differential tests compare the old scanner against the new scanner on adversarial cases, fuzz input and a read-only corpus of real ledger rows. Also remove private ledger fields' redundant duplicate traversals after the strict whole-row check, preserving unknown/null/trailing rejection. Require measured host margin under load before closing. ARCH-DRY: repair the shared codec rather than introducing a weaker ownership-only parser.

### 2026-10-15 — approved bounded proof scheduling

TL approved dcac37b0: prove at most four owners concurrently in each before/after phase. Consume results and business checks in stable wrapper order regardless of completion order; a stable-order failure cancels and joins every worker before returning. Context cancellation also joins. Fresh snapshot IO, whole-ledger validation and both process/evidence fences remain unchanged. The pure memo has a short mutex around map access only; never hold it over IO, hashing or decode. Add deterministic shuffled completion and cancellation/join tests, and run the real multi-owner gate under the race detector. Same five-second total budget and no persistent cache.
