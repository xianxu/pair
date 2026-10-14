# Boundary Review — pair#550 (whole-issue close)

| field | value |
|-------|-------|
| issue | 550 — Qoder slots never connect to Couch: wrapper creates no message endpoint and never settles (pair:4, 10-14) |
| repo | pair |
| issue file | workshop/issues/000550-qoder-slots-never-connect-to-couch-wrapper-creates-no-message-endpoint-and-never-settles-pair-4-10-14.md |
| boundary | whole-issue close |
| milestone | — |
| window | 41861430868ce99ad949ce01b1154b61ec584077..7505a1357745ed1fd3c5392107f8d604fa3b3a5a |
| command | sdlc close --issue 550 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T09:27:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation addresses Qoder’s missing receiver admission and composer support. Recorded-screen, endpoint, broker-delivery, and full wrapper tests pass. No correctness defect was found in the changed code. Update README’s receiver-support documentation before closing the boundary.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-facing-surface-documentation
    title: |
      README update appears missing for Qoder peer-message support
    detail: |
      README.md:649–652 still lists only Claude Code and Codex CLI as peer-message receivers, while peer_runtime.go:28 enables Qoder and atlas/couch.md:464 documents it. Update the README support paragraph to include Qoder and its captured-version evidence. Family enumeration: this receiver-support paragraph is the sole affected README location; the corresponding atlas support statement is already updated.
```

1. **Strengths**

   - `composer_recognizers.go:319` shares Qoder’s ruled-box specification between Return recognition and peer parsing.
   - `peer_composer.go:185` requires captured placeholder wording, cursor position, painted background, and matching hint foreground before treating it as empty.
   - `peer_qoder_test.go:111` exercises production endpoint startup and live-tail transport; `peer_runtime_test.go:26` exercises Qoder through the real broker/socket delivery path.
   - Inventory-policy and captured-composer tests cover every enabled receiver.

2. **Critical findings:** None.

3. **Important findings:** README omission described above. Add Qoder to `README.md:649–652`, consistent with the receiver policy and atlas.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused Qoder, inventory, captured-composer, and broker tests passed.
   - `go test ./cmd/internal/wrapcmd -count=1` passed in 136.264 seconds.
   - `git diff --check` passed; both new fixture hashes match their metadata.
   - Coverage includes idle versus trust, empty versus occupied multiline input, placeholder mutations, permission/question pickers, operator interruption, and exactly one paste/submit.
   - Native live-smoke evidence was not independently reproduced; executable code is unchanged from the revision recorded for that smoke.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Shared geometry and existing delivery transitions are reused.
   - **ARCH-PURE: pass.** Composer classification remains snapshot-based; runtime IO stays in the existing boundary.
   - **ARCH-PURPOSE: pass.** Registration, live tail, settling, and message delivery are exercised. All six inventory agents have explicit policies, and all three enabled receivers have captured empty-composer coverage.

7. **Plan revision recommendation:** Add a `## Revisions` entry extending the documentation checklist to update README’s receiver-support paragraph alongside the atlas.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.899234417s
head: "7505a1357745ed1fd3c5392107f8d604fa3b3a5a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-14T09:31:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 550 — Qoder slots never connect to Couch: wrapper creates no message endpoint and never settles (pair:4, 10-14) |
| repo | pair |
| issue file | workshop/issues/000550-qoder-slots-never-connect-to-couch-wrapper-creates-no-message-endpoint-and-never-settles-pair-4-10-14.md |
| boundary | whole-issue close |
| milestone | — |
| window | 41861430868ce99ad949ce01b1154b61ec584077..50ce66473a08977b9bf3107a03c64b997d398ca8 |
| command | sdlc close --issue 550 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T09:31:55-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Qoder receiver admission and composer parsing satisfy the issue’s documented contracts. Focused tests and the full wrapper package pass. BR-1 is addressed by the README correction, and no new blocking findings were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:649–657 now includes Qoder, captured versions 1.1.71/1.1.66, and the current submission contract. This matches peer_runtime.go:27–30, the recorded-screen tests, and dispatchPeer/advancePeer. The correction changes prose only.
```

1. **Strengths**

   - `composer_recognizers.go:319` shares Qoder’s ruled-box specification between Return recognition and peer parsing.
   - `peer_composer.go:185` requires captured wording, cursor position, painted background, and matching hint foreground before admitting the placeholder as empty.
   - `peer_qoder_test.go:111` exercises production endpoint registration and live-tail transport; `peer_runtime_test.go:26` exercises broker/socket delivery.
   - Inventory tests enforce explicit policies for all six agents and captured empty-composer support for every enabled receiver.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused Qoder, inventory, captured-composer, and broker tests passed.
   - `go test ./cmd/internal/wrapcmd -count=1` passed in 54.996 seconds.
   - `git diff --check` passed; both fixture hashes match their metadata.
   - Coverage includes idle/trust, empty/occupied multiline input, placeholder mutations, picker refusal, operator interruption, settling, and exactly one bracketed paste plus bare-CR submit.
   - Native live smoke was not independently repeated. Inspection confirms executable code is unchanged from the recorded smoke revision.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Shared geometry and existing delivery transitions are reused.
   - **ARCH-PURE: pass.** Composer classification remains deterministic over snapshots; runtime IO stays at the existing boundary.
   - **ARCH-PURPOSE: pass.** Registration, live tail, settling, and delivery are exercised. Receiver-policy enforcement covers the complete inventory.

7. **Plan revision recommendations:** None. The existing `## Revisions` entry incorporates the README documentation sweep, and both README and atlas updates are present.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.084162167s
head: "50ce66473a08977b9bf3107a03c64b997d398ca8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
