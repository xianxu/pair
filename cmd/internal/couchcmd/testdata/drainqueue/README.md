# Real SDLC queue conformance

Run this retained check before first drain enablement, after changing the queue
argv/schema, adapter, decoder, or fake, and after updating the deployed SDLC queue
implementation. Keep the passing binary/source revision in the issue or release
record. The ordinary suite skips these tests unless both paths are explicitly
provided; it continues to exercise the stateful fake and strict decoder.

From the Pair checkout, select a clean Ariadne source checkout and its matching,
clean Go-built SDLC executable:

```sh
PAIR_DRAIN_SDLC_BINARY=/absolute/ariadne/bin/sdlc \
PAIR_DRAIN_SDLC_SOURCE=/absolute/ariadne \
go test ./cmd/internal/couchcmd -run '^TestPair722RealQueue' -count=1 -v
```

The test refuses partial/relative paths, missing Go provenance, dirty builds or
tracked source changes, and source/binary revision mismatches before any queue
command. Run from a Couch worker to also assert that the real worker identity
cannot mutate even the private queue. Outside Couch, that one assertion is
skipped. A wrong executable (for example `/usr/bin/true`) is a useful negative
control: it must fail the build-provenance check before any mutation.

Every test creates a fresh `t.TempDir()` with its own Git fixture, queue state,
fleet discovery root and machine capacity lock. `TMPDIR` selects scratch storage;
`sdlc local-check` supplies the RAM disk when available. No paths from an earlier
probe are reused. The harness clears `COUCH_TREE` in the test environment for fixture subprocesses;
the selected binary always runs in that fixture with all three private root
overrides set.
Never remove those overrides to investigate a failure. Test cleanup releases its
owners and deletes only the newly created temporary tree.

The tests invoke `sdlcDrainQueue` (production argv, timeout/output bounds and strict
JSON decoder), rather than parsing independently. They cover literal reasons,
replay/idempotency, owner isolation, all three active kinds, pending admission,
graceful versus fast cancellation, retained interrupted outcomes, lost lifetime
and unreadable observations, held legacy tokens/tickets, verified waiters, and
lost replies after successful pause/resume effects. Loss of a reply is injected
by a private wrapper around the selected real executable; subsequent real status
must reconcile its effect while preserving other owners.

`lease_helper.go.txt` is built once per test process in a suite-owned disposable
Go module against the selected Ariadne source revision. Every private fixture
aliases that read-only executable; `TestMain` removes it after all tests and
consumer cleanup finish. Its internal queuegate API creates real lifetime leases
and admission/cancellation/retirement evidence without running a suite, merge or
landing. It is a fixture driver, not a duplicate queue implementation. An API
change is expected to fail this build and require updating conformance alongside
the dependency. These checks establish adapter/protocol conformance; they do not
replace the operator-present live fleet board and checkpoint/ack acceptance.
