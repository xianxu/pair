# #672 BR-1 exact-head host qualification

Reviewed source head: 5e299f51362df4c72591502cca46971ec7df5a80. Registered checkout:
/Users/xianxu/workspace/worktree/pair-slot1/pair. All checks below returned exit0.
Source remained byte-identical to that head (git diff --quiet HEAD -- cmd internal).
No source, fixture, deadline, decoder or gate configuration changed for this evidence.

Commands run directly on the registered host:

- TMPDIR=/tmp go test ./cmd/internal/launcher ./cmd/internal/zellijreply ./cmd/internal/zellijsession -count=1
- TMPDIR=/tmp go test -race ./cmd/internal/launcher ./cmd/internal/zellijreply ./cmd/internal/zellijsession -count=1
- TMPDIR=/tmp go test ./internal/repolint -count=1

Normal PASS: launcher61.335s, reply0.132s, session0.042s.
Race PASS: launcher132.609s, reply1.194s, session1.066s.
Repository catalog PASS7.513s.

| Check | Retained log | SHA256 |
|---|---|---|
| normal | /tmp/pair672-host-normal.log | 48a5d8a374fe69ba372bcdcf42ca45050a9ec932a45e9e01da9a21e81681efcd |
| race | /tmp/pair672-host-race.log | 2a9570bd0267f13a06d8ac824ced3f169f2ca7e261f69b2a1ff5f4b4bf8170e0 |
| catalog | /tmp/pair672-host-catalog.log | d5528731e3da3868fb6572ffb26c4daef7a7403c52716c34faaf9443b8e58769 |

The review clone's default macOS TMPDIR produces overlong Unix socket addresses
in existing workbench_attach_invariant tests; a short owned temporary root passes
those same assertions, with no bound changes or skipped tests. The registered
main-slot1 branch tracks origin/refs/heads/main; origin/main was Git-verified
b115dc79533f5949b765ef9b31cf02a579727c4c. This is the authentic main authority
required by the existing sleep-debt ratchet, absent from the review clone.
No synthetic refs or weakened ratchet were introduced.

The reviewer independently passed the new normal/race regressions and verified
they fail against the base implementation. The finding explicitly requests this
host evidence and identifies no implementation defect. Review round2 should
explicitly dispose the unverifiable finding against these measured results.
The separate recorded full suite is still queued on code head5e299f51362df4c72591502cca46971ec7df5a80;
this evidence neither replaces it nor claims its result.
