---
collaboration-mode: discussion
---

# Diagnostic device rebind implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Execute with the executing-plans skill; main owns the warm recovery boundary.

**Goal:** Restore logger admission after a device-only remount without certifying unknown payload effects.

**Architecture:** One pure persisted-checkpoint classifier; explicit consumers under existing diagnostic lock. Strict live-FD checks stay separate. No persisted format change.

**Tech Stack:** Go, existing flock/atomic JSON publisher, disposable filesystem fixtures.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Persisted checkpoint comparison | cmd/internal/diagnosticlog/checkpoint_identity.go | new |

The classifier owns exact/device-only/mismatch for one observed identity/size/time and one persisted generation. It is policy-free about IO and does not grant append-tail authority; colocated table tests require no mocks. Strict matches remains the same-invocation authority.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Current checkpoint recovery | cmd/internal/diagnosticlog/append_reservation.go | modified | existing locked state save |
| Preview and collection | cmd/internal/diagnosticlog/pages.go; collect.go | modified | existing regular-file/proof/delete protocol |

Real disposable directories model metadata from a previous mount; no new external command or runtime dependency. Existing writers and collection own artifact cleanup (ARCH-FUNERAL). Unknown codec members remain subject to existing strict decoding; no codec or format change (ARCH-EVOLVE). No new persisted state or authority vocabulary.

## Approved design and sequence

# #774 proposed plan — device-only persisted checkpoint recovery

Measured case: device16777232 in state versus16777229 on disk; inode728006625, size26061670 and nanosecond mtime1792450564739652454 match exactly. The old append reservation describes bytes already checkpointed. #736 health closure is SHIP in PR526; #774 is independent and claimed. #773 remains held.

1. Add one pure classifier for exact / device-only / mismatch against persisted generation metadata. Device-only requires nonzero matching inode, identical size and nanosecond mtime; mismatches name inode, size, time or unsafe identity. Keep the existing strict live-file matcher, append-tail classifiers and persisted-record equality unchanged (ARCH-PURE, ARCH-DRY).
2. Under the existing diagnostic flock, recover a device-only CURRENT checkpoint only when no Creating, Pending, Deleting or legacy Appending intent exists. Save the observed device identity through the existing atomic checkpoint publisher before allowing append. Preserve all clocks, bytes and unverified-tail metadata. Never rewrite an old reservation to create tail authority. A failed checkpoint save returns unavailable; the next explicit Open reconciles the file, with no new retry loop.
3. Apply the same persisted comparison to current/retained read-only preview and retained collection's existing exact metadata/proof path. Collection still requires its writer/reader proof and locked deletion intent. Keep interrupted creation/rotation/deletion/legacy append conservative if remount leaves uncertain effects; report a specific transaction/identity refusal. Live FD comparisons after opening, syncing or reading remain strict, including a device change DURING one operation. No schema, ownership, proof or retention-policy change.
4. RED first: reproduce committed append plus stale reservation and device-only mismatch by editing only disposable metadata. Assert Open currently fails. GREEN: reopen, append a new fact, verify original bytes plus new bytes and a second reopen. Add classifier negatives for inode, size, nanosecond time, zero identity; integration controls for foreign/partial/full uncertain tails, unsafe path, pending transaction and save interruption. Qualify retained preview and collection using the same old-device fixture, and strict within-operation device changes. Tests use real temporary files and synthetic persisted metadata, never the live store.
5. Update README/atlas ownership from #736 to #774 and explain remount recovery and conservative refusals. Run the diagnosticlog suite/race controls, affected reader inventory, recorded affected check, close review and normal landing. Hub rebuild/restart owns live verification that resources reports written facts and last-write advancing. Dropped historical facts are never replayed.

Actions: restart on the landed repair; inspect the named refusal if metadata genuinely differs.
| Internal result | Visible state | One recovery action | Possible loss |
|---|---|---|---|
| Exact or device-only checkpoint, save succeeds | healthy; writes advance | none needed | earlier dropped facts stay lost |
| IO/checkpoint save fails | unavailable, exact cause | correct named storage fault, restart | new facts are dropped |
| Inode/size/time/tail or transaction uncertainty | unavailable, specific refusal | inspect named store inconsistency before restart | no payload deleted or certified |

Cost: O(1) metadata comparison per touched generation, no payload reread for device-only recovery; at most one existing bounded checkpoint write on current rebind. Preview/collection keep existing bounded page traversal; N/2N classifier tests count one decision per generation. Existing flock is the single authority; no new executor/cache/store/format or machine change.

Covered: measured current failure, retained same-class reads/collection, strict live identity and unknown tails. Not broadened: recovery of an interrupted transaction across a remount; conservative refusal is explicit, not a new reset mechanism.


## Checkable tasks

- [ ] Add the measured Open/reopen fixture in checkpoint_identity_test.go and observe its semantic RED with go test ./cmd/internal/diagnosticlog -run DeviceRebind -count=1.
- [ ] Add the pure classifier matrix and smallest current-rebind glue; run the same cohort GREEN. Save only under existing flock; no pending intents accepted.
- [ ] Add retained preview/collection RED controls, apply the shared persisted comparison, then run GREEN. Keep live matches and in-flight recovery strict.
- [ ] Add save-failure, unsafe-path, uncertain-tail and pending-intent negatives; run all diagnosticlog tests and targeted race tests.
- [ ] Update README/atlas, run make test-affected-inventory, push and record affected check. Close once with binary-owned review, land through sdlc merge --yes, report PR/merge to hub. Hub owns live fresh-write acceptance.
