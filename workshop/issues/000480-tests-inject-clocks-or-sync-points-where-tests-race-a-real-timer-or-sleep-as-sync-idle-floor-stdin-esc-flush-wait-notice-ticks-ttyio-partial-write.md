---
id: 000480
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'abddf571615f7b00ea5d209d8bad29bec5de96fb' # card fields mirrored from issue-cards; edit via sdlc
---

# tests: inject clocks or sync points where tests race a real timer or sleep as sync (idle floor, stdin ESC flush, wait-notice ticks, ttyio partial write, ...)

## Problem

Found by the #476 sweep (fixed real-time windows asserted under load, the #473/#476 flake family). Ranks are a reading of the code, not measured failures. Paths are under `cmd/internal/`.

These tests race a real timer, or sleep and then assert that something already happened. A slow scheduler makes them fail even though the product is right:
- wrapcmd/idle_floor_test.go:125-131: 60 ms idle floor with 15 ms sleeps between outputs; one oversleep fires a real alert, so "no alert" falsely fails.
- wrapcmd/translate_stdin_test.go:151 (5 ms sleep vs a 20 ms flush, "flushed too early"), :181 and translate_test.go:306 (second write must beat a 100 ms / 50 ms ESC flush timer).
- couchcmd/wait_notice_test.go:264 `tickAt` (~17 callers, :274-425): sleeps 150 ms as sync on socket delivery, then asserts `len(delivered)`. High.
- ttyio/writer_test.go:45: an 80 ms ctx must expire only after the pty write made partial progress (n>0). High. :238 asserts elapsed <= 4 s on a 3 s ctx; :257 asserts a cancel at 200 ms returns within 1 s.
- couchcore/registration_phase_test.go:80: the injected fault fires only within 200 ms of pane birth; a slow poll skips it.
- couchcore/never_turned_test.go:50 (refusal under 200 ms), launcher/pane_birth_test.go:223 (returns within 1 s with the production 10 s bound in play), couchmessage/broker_actors_test.go:57 (infers "no probe" from 100 ms latency; `restingProbes` already counts probes).
- broadcast/pointer_test.go:297 (sleep 100 ms, then expects a 429), hostty/hostty_test.go:236 (sleep 200 ms, then expects at least one SIGWINCH wake).

## Spec

Replace each real timer or sleep with the event or count it stands for: an injected clock or timer seam (the #473 `Couch.after` pattern), a counter the fake already keeps (`restingProbes`), or a channel signalled when the state is reached (in-flight == Max, first wake). Fix product code if the race is real there. Prove each window first by widening it with a temporary sleep. Split the issue if one item needs a product seam big enough to review alone.

## Done when

- Each listed test asserts an event, count or injected clock instead of wall time, and passes 200 runs under `-race` and parallel load.
- Each cause is recorded in the Log, measured by widening.

## Plan

- [ ] Per test, prove the window by widening it with a temporary sleep; record the cause in the Log.
- [ ] Replace each real timer or sleep with an injected clock/timer, an existing counter, or a state signal; fix product code if the race is there.
- [ ] 200 `-race` runs of each touched test under parallel load; full `make -k test`.

## Log

### 2026-10-12
