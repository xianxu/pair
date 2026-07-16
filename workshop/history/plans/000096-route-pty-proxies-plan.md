---
issue: 000096
created: 2026-07-01
---

# Plan — route pair-wrap through the dispatcher; retire pair-scribe

Mechanical repackaging (no logic change) that puts the two remaining PTY-proxy
entrypoints on #92's streaming dispatch seam. `pair-wrap` is extracted to a
reusable `wrapcmd` package behind a thin shim and a `pair wrap` route;
`pair-scribe` is confirmed dead and retired.

## Decisions (settled in exploration)

- **pair-scribe is live user tooling → route it (not retire).** My first read
  ("orphaned → retire") was wrong: an in-tree grep can't see the user's
  `~/.zshrc`, and `atlas/architecture.md:44-46,730-732` + `cmd/pair-scribe/README.md`
  document scribe as a deliberate `script(1)` replacement installed to
  `~/.local/bin/pair-scribe` by `make install` and wired into the user's shell
  startup. The binary is in fact installed. The atlas **already lists `pair
  scribe`** as a target dispatcher subcommand. User confirmed: route it. So mirror
  the pair-wrap treatment — extract `cmd/internal/scribecmd`, add the `pair scribe`
  streaming route, and keep `cmd/pair-scribe` as a thin shim so
  `~/.local/bin/pair-scribe` keeps installing and the user's `~/.zshrc` `exec`
  line is untouched (non-destructive; `GO_BINS`/install stay). scribe is NOT in
  the runtime bundle (it's shell tooling, not runtime) — no manifest change.
- **stdio threading via type-assertion.** `wrapcmd.Run(args []string, stdin
  io.Reader, stdout, stderr io.Writer) int` is the #76 runner shape. pair-wrap
  needs `*os.File` for raw-mode / winsize / `pty.GetsizeFull`, which `io.Reader`
  can't give — so `Run` type-asserts `stdin`/`stdout` back to `*os.File` and
  stores both the interface and the file on the `proxy`. In production the shim
  and the streaming seam both pass the real `os.Stdin`/`os.Stdout`, so the
  assertion always succeeds and behavior is **byte-for-byte identical**; a
  non-file reader (a test) simply skips the terminal ops (already `isTTY`-guarded).
- **child-exit propagation becomes a return code.** `run()`'s mid-flight
  `os.Exit(exitErr.ExitCode())` becomes `return code, nil`; `run` returns
  `(int, error)`, and `Run` maps a non-nil error to the `pair-wrap: %v` stderr +
  exit 1, else returns the child code.

## Steps

- [ ] **Extract `cmd/internal/wrapcmd`.** `git mv cmd/pair-wrap/main.go →
      cmd/internal/wrapcmd/wrap.go` and `git mv` all `cmd/pair-wrap/*_test.go`
      into `cmd/internal/wrapcmd/`. Rewrite `package main` → `package wrapcmd`
      (source + every test). Delete `func main()`. Add exported
      `Run(args, stdin, stdout, stderr) int`; convert `run()` → `run(args
      []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)`.
- [ ] **Thread stdio through the `proxy`.** Add fields `stdin io.Reader`,
      `stdinFile *os.File`, `stdout io.Writer`, `stdoutFile *os.File`,
      `stderr io.Writer`; set them at proxy construction. Repoint the ~10 direct
      `os.Std*`/`os.Args` sites: `os.Args[1:]`→`args`; `translateStdinFrom(os.Stdin…)`
      and the stdin goroutine `Read`/`io.Copy`→`p.stdin`; raw-mode
      `isTTY`/`MakeRaw`/`Restore`→`p.stdinFile` (nil-guarded);
      `setWinsize`/banner `pty.GetsizeFull(os.Stdin|os.Stdout)`→`p.stdinFile`/
      `p.stdoutFile` (nil-guarded); `writeStartupBanner`→method on `p.stdout`;
      `newStdoutPump(os.Stdout)`→`newStdoutPump(p.stdout)` (both sites).
- [ ] **Thin shim.** `cmd/pair-wrap/main.go` = `func main() {
      os.Exit(wrapcmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }`.
- [ ] **Extract `cmd/internal/scribecmd`** (mirror wrapcmd). `git mv
      cmd/pair-scribe/main.go → cmd/internal/scribecmd/scribecmd.go`; move the
      README. `package main`→`package scribecmd`. `Run(args, stdin, stdout,
      stderr) int`. Switch the global `flag` usage to a local
      `flag.NewFlagSet("scribe", flag.ContinueOnError)` so `Run` doesn't consume
      `os.Args` and is re-callable; thread `os.Stdin`/`os.Stdout` and the
      `os.Exit`/`fatalf` paths into return codes (raw-mode/`InheritSize` need the
      `*os.File`, same type-assert seam as wrap).
- [ ] **Scribe thin shim.** `cmd/pair-scribe/main.go` = `func main() {
      os.Exit(scribecmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }`.
- [ ] **Wire the dispatcher.** `dispatcher.go`: flip `wrap` **and** `scribe`
      Status `planned`→`implemented` (keep `Streaming: true`).
      `cmd/pair-go/main.go`: add `case "wrap"`/`case "scribe"` to
      `runStreamingSubcommand`, each calling its `*cmd.Run(args[1:], stdin,
      stdout, stderr)`.
- [ ] **Makefile deps.** pair-wrap rule prereqs → `cmd/pair-wrap/main.go
      $(wildcard cmd/internal/wrapcmd/*.go) go.mod`; pair-scribe rule prereqs →
      `cmd/pair-scribe/main.go $(wildcard cmd/internal/scribecmd/*.go) go.mod`.
      `PAIR_GO_SRCS`: add the `cmd/internal/wrapcmd/*.go` and
      `cmd/internal/scribecmd/*.go` non-test files (pair-go imports both).
      `GO_BINS`/install/`.PHONY` for `pair-scribe` stay unchanged (shim survives).
- [ ] **Update tests for the status flip.**
      `dispatcher_test.go`: move `wrap` **and** `scribe` into the
      implemented/present set (only `launch` stays in the "absent" handoff list);
      add both to `IsStreaming` expectations; repoint
      `TestDispatchPlannedCommandReturnsUnsupported` to assert the streaming-seam
      guard message for `Dispatch(["wrap"])`.
      `main_test.go`: `TestRunWritesStderrAndReturnsDispatcherCode` uses an
      unknown command (not `wrap`) and asserts `"unknown command"`.
- [ ] **New parity tests** (the load-bearing verification). In `wrapcmd` and
      `scribecmd`: exit-code parity (child exit code propagates through `Run`),
      arg passthrough / unknown-flag + usage errors → exit codes, and a PTY-level
      behavior check driving `Run` through a pty pair (scribe currently has **no**
      tests — this is its first coverage). In `cmd/pair-go`: route tests proving
      `pair wrap` / `pair scribe` reach their `Run` via the streaming seam
      (mirrors the continuation/session-watch route tests).
- [ ] **Regen the runtime bundle manifest** (`make runtimebundle-generate`) so
      the embedded `bin/pair-wrap` matches the rebuilt shim; run
      `runtimebundle-drift-check`.
- [ ] **Atlas.** Update `atlas/go-migration-inventory.md` (pair-wrap → routed;
      pair-scribe → routed as `pair scribe` + shim). Update
      `atlas/architecture.md:730-732` "Adjacent: pair-scribe" to say it's now a
      `pair scribe` dispatch route with `cmd/pair-scribe` as the thin shim (the
      `~/.zshrc` wiring + `~/.local/bin/pair-scribe` install path are unchanged).
      Repoint any pointer that names `cmd/pair-scribe/main.go` /
      `cmd/pair-wrap/main.go` as the logic home (now `cmd/internal/*cmd`).

## Verification

- `make test` green (all moved pair-wrap tests + new parity/route tests +
  scribecmd's first tests).
- `runtimebundle-drift-check` clean (manifest matches tree; only pair-wrap is
  bundled, scribe is not).
- Real wrapped session end-to-end (start → turns → slug refresh → exit) via the
  rebuilt `bin/pair-wrap` shim; `pair scribe -log … -- cmd` and
  `~/.local/bin/pair-scribe` both still work.
- `git grep -n 'cmd/pair-scribe/main.go\|cmd/pair-wrap/main.go'` in atlas/README
  finds no stale "logic lives here" pointers.
