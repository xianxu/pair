# Terminal ownership

The shared terminal implementation for #255 lives in `cmd/internal/terminal`.
M2 adds the library; Couch and Pair adoption belongs to M3. Live symptom acceptance
is held for the operator after M4.

An `Endpoint` owns one child's emulator and negotiated modes. Output updates that
state before a consumer can see the batch. Queries are answered from that endpoint
and enter the same bounded FIFO as encoded operator input. Immutable `Frame`
publications carry endpoint identity, generation and acknowledged geometry epoch.
Synchronized output retains the last complete frame, with a 150ms recovery bound.
Clipboard writes, notifications, title/directory changes and bells are ordered,
origin-bearing effects; snapshots cannot replay them. Clipboard reads are explicitly
unsupported and receive an empty response.

`Frame`, `View`, `Compose` and `Render` express screen composition and admission
without IO. `Presenter` owns the physical parent writer and commits input admission
only after presentation completes. A failed partial write invalidates that
connection's presentation state. Product code retains shortcut and notification
policy, while terminal negotiation and encoding belong to the shared connection.
Parent mouse capture differs by boundary: Couch needs motion for its chrome;
Pair's terminal pane must leave native Zellij selection available when its child
requests no mouse reporting. Child requests still determine forwarded events.

`ttyio.File` acquires input/output descriptor flags before pumps start. Its read and
write adapters share that lifetime, use nonblocking syscalls and bounded polling,
and restore flags after operations join. Duplicating a descriptor would not isolate
its nonblocking flag. `InputWriter` preserves packet order and accepted-prefix
failures, with 128 packets / 1MiB including the packet currently being written.

The child contract is `pair-vt-256color`, derived from the explicit capability table
in `terminal/profile_query.go`. `terminfo/` contains its source and audit.
Capabilities are not inferred from the outer terminal. The checked-in local module
fork in `third_party/vt` preserves upstream provenance and records its narrow
repairs and bounds in `PAIR_PATCHES.md`. Root Go tests do not traverse that nested
module; its tests must also run from `third_party/vt`.

Verification has three independent layers: literal terminal/frame fixtures,
`terminalqualify` backend and integration cases, and actual renderer wire interpreted
by pinned `@xterm/headless` under `tests/terminal-oracle`. Production composition and
sustained native terminal acceptance add evidence beyond those library checks.
