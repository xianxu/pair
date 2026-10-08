# #379 — Contain UTF-8 control-string payloads

## Contract and evidence

The captured brain:0 incident and minimal production Endpoint reproducer establish that x/ansi v0.11.7 mistakes the 0x9c continuation byte of ✻ for ST. Repair all affected streaming consumers and the related SOS/PM/APC transition errors. Keep existing control, parameter, payload and memory bounds. User authorized implementation and sibling-path audit.

## Core concepts

| Entity | Location | Status |
| --- | --- | --- |
| Streaming parser and bounded UTF-8 continuation state | third_party/vt/ansiparser | new, derived from licensed upstream parser |

The parser remains a deterministic per-instance state machine. A control-string payload retains its current OSC/DCS/SOS/PM/APC state while a valid UTF-8 prefix consumes continuation bytes; those bytes use PutAction, never screen PrintAction or C1 transitions. A malformed prefix resets continuation expectation and processes the current byte normally, preserving ESC/BEL/CAN/SUB recovery and standalone C1 ST. First-continuation ranges reject overlong, surrogate and out-of-range UTF-8. Reset clears the continuation state. Fixed counters add constant memory; existing SetDataSize still limits storage even while an oversized string is consumed.

| Integration | Location | Status | Wraps |
| --- | --- | --- | --- |
| Emulator, control observer and notification output boundary | third_party/vt/emulator.go, pair_parameters.go; cmd/internal/wrapcmd/terminal_model.go, notification_output.go | modified | one shared streaming parser |

Copy only upstream parser.go with license/provenance into the existing vt fork's public ansiparser subpackage; reuse upstream Handler/Cmd/Params types and immutable transition table. Avoid a full dependency fork, global table mutation, character substitution or notification-specific filtering. This shares the correction across all raw streaming consumers (ARCH-DRY, ARCH-PURPOSE). Local cmd/internal/ansi is separately audited; trusted ASCII SGR DecodeSequence paths are outside this fault. No new service, goroutine or durable runtime artifact (ARCH-PURE, ARCH-MOCK, ARCH-FUNERAL).

## Work and verification

- [ ] Add regression coverage for all five string families, whole/every-split/bytewise feeds, intact dispatched payloads and no screen leakage. Include BEL/ESC-ST/C1-ST termination, malformed prefixes, cancellation, parser reset, and overflow followed by visible sentinel. Run red against unchanged parser.
- [ ] Implement shared parser repair; migrate every raw streaming consumer and preserve parameter guard semantics. Bring upstream parser tests across to defend compatibility.
- [ ] Add production Endpoint incident regression with nvim-position cursor, preserve notification effect and unchanged frame. Audit local strip and framed output helpers with adversarial strings; repair same-class defects if found.
- [ ] Run shared parser, vt, terminal, notification, observer and output-boundary suites (race where appropriate), build Couch, replay captured brain endpoint through corrected production parser and check right-pane footer is absent. Use private fixture outside repository; commit only synthetic fixtures.
- [ ] Update atlas and provenance; inventory new production files. Close with binary-owned fresh-context review, publish through PR and merge; retain #379 evidence.

## Risks

Fork divergence is bounded by a documented upstream file/version and retained upstream tests. Existing Ground-state malformed UTF-8 handling is audited separately; do not change unrelated glyph rendering without evidence. A parser reset or canceled sequence must not leave continuation state alive. Invalid standalone bytes in ignored strings remain contained except intentional control transitions. Snapshot/replay remains accelerated; minimal regression establishes timing independence for this cause.
