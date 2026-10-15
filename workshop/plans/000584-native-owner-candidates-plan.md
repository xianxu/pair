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
- [ ] Add red legacy tests: different-ID exclusion, matching-ID collision/source exemption, direct parent and birth changes, wrong owner/ordinal/held ID, read-only paths, no proof, other-agent exclusion and modern identity mismatch.
- [ ] Factor existing exact-ordinal ledger projection once for modern, legacy and orphan consumers; add the injected legacy held-file check without weakening modern readiness.
- [ ] Add bounded writable-file IO and reuse native filename codecs; fake command/parser tests and read-only host conformance.
- [ ] Add exact-owner migration diagnostics, with friendly Couch slot resolution through preview reads where available; tests for known and unresolved owners.
- [ ] Update atlas and Log with justification, the 18-wrapper census, host evidence and limitations; targeted tests and build.
- [ ] Record the full suite on the pushed final code head; close once with the mandatory SDLC review; land on SHIP and report merge to hub. No live migration or resume from this slot.

## Revisions

- 2026-10-15: Initial small candidate repair expanded by the TL's approved legacy proof because ordinary operational migration is circularly blocked and 18 of 21 live wrappers are legacy. Exact held writable transcript plus kinship is positive proof; an exact different ordinal-ledger ID is the approved negative collision exclusion.
