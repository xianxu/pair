# Carbonyl Browser Tab Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A right-pane `pair term` tab that runs Carbonyl (Chromium in the
terminal), labelled `name · host`, navigable from a URL field in the tab strip,
and shared with the agent through a per-tab DevTools record that `pair browser`
prints.

**Engine risk, accepted 2026-09-19.** Carbonyl is unmaintained (last upstream
commit 2023-02-26) and bundles Chromium 111. The operator accepted it because
nothing maintained renders a real browser as terminal text *and* exposes CDP for
the agent. Two consequences are design constraints, not asides: the engine stays
behind a swappable seam (a tab kind that launches a binary and speaks CDP, so a
future engine keeps the record, `pair browser`, chords, labels and lifecycle),
and a remote URL is never opened silently (`IsLocalURL` + confirm-on-remote).

**Architecture:** A new package `cmd/internal/browsertab` holds the pure core:
URL normalization, labels, default names, the version gate, launch argv, the
record codec, and the per-tab state machine (`Step`). It also holds three thin
IO seams: the profile directory, the record store, and a CDP client over
`github.com/coder/websocket`. `pair term` (`cmd/internal/termcmd`) gains a
browser tab kind whose controller goroutine runs `Step` and executes its
effects. `cmd/internal/browsertab/fakecarbonyl` is the stateful fake of the
Carbonyl binary behind the same seams (argv, pty, DevTools port file, HTTP +
websocket protocol, process group). `pair browser` (`cmd/internal/browsercmd`)
reads records and checks liveness.

**Tech Stack:** Go 1.26, `creack/pty` (via `ptychild`), `charmbracelet/ultraviolet`
input events, new dependency `github.com/coder/websocket` v1.8.15 (zero
transitive deps, ISC), Chrome DevTools Protocol (Target + Page domains) as
spoken by Carbonyl 0.0.3 (Chrome 111).

**Dependency choice — APPROVED by the operator 2026-09-19** (the first
non-terminal third-party dep): `coder/websocket` versus a small in-tree RFC
6455 implementation. Recorded for the reviewer, since the alternative below is
no longer open.

| | In-tree | `coder/websocket` |
|---|---|---|
| What we'd need | client + server (the fake) | both included |
| Size | ~350 lines of framing, masking, fragmentation and close handshake | none of ours |
| Correctness burden | own fuzzing and conformance | carried upstream |
| Transitive deps | none | none |
| Other | — | maintained; its `Dial` sends no Origin header, which Chrome 111's DevTools requires |

In-tree buys nothing but independence from one well-scoped module, and the
module is actively maintained (last commit 2026-06-15, ISC, zero transitive
deps). The operator approved it.

**Source of truth for scope:** `workshop/issues/000292-carbonyl-browser-tab.md`
(`## Spec`, `## Done when`). The spike measurements that motivate the numbers
here are in that issue's `## Log`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `NormalizeURL` / `IsLocalURL` | `cmd/internal/browsertab/url.go` | new |
| `Label` | `cmd/internal/browsertab/label.go` | new |
| `DefaultName` | `cmd/internal/browsertab/name.go` | new |
| `Version` / `ParseVersion` / `VersionNotice` | `cmd/internal/browsertab/version.go` | new |
| `Argv` / `DefaultFPS` | `cmd/internal/browsertab/launch.go` | new |
| `ParseActivePort` / `Endpoint` | `cmd/internal/browsertab/endpoint.go` | new |
| `OwnedName` / `ParseOwnedName` / `BirthHash` | `cmd/internal/browsertab/owned.go` | new |
| `Record` / `EncodeRecord` / `DecodeRecord` / `RecordFile` | `cmd/internal/browsertab/record.go` | new |
| `Resolve` (name → record, ambiguity) | `cmd/internal/browsertab/record.go` | new |
| `State` / `Event` / `Effect` / `Step` | `cmd/internal/browsertab/machine.go` | new |
| `TabChip` (label), `RenameField` → `StripField` (prefix) | `cmd/internal/termcmd/strip.go` | modified |
| `RenameEditor` (reused for URLs; paste insert) | `cmd/internal/termcmd/rename.go` | modified |
| `ChordAltB` / `ChordAltShiftB` / `ActionNewBrowserTab` / `ActionTerminalNewBrowserTab` | `cmd/internal/workbenchshortcut/shortcut.go` | modified |

- **NormalizeURL** turns operator input into a URL Chrome will navigate to.
  `localhost:1111` → `http://localhost:1111`; `example.com/x` →
  `https://example.com/x`; keeps `http(s)://`, `file://`, `about:`, `data:`;
  rejects empty input and control characters.
  - **Relationships:** used by the URL field commit (new tab and navigate). #293
    will use it for Alt+click.
  - **DRY rationale:** one normalization for every entry point (field, Alt+click,
    agent CLI if one is ever added).
- **IsLocalURL** answers "does this URL stay on this machine or a private
  network": loopback, `localhost`, `*.localhost`, `*.test`/`*.localdomain`,
  RFC 1918 and link-local addresses, plus `file:`, `about:` and `data:`.
  Everything else is remote.
  - **Why it exists:** Carbonyl bundles Chromium 111 and has had no upstream
    commit since 2023-02-26, so it renders untrusted web content with 3.5 years
    of unpatched CVEs. The operator accepted the engine on 2026-09-19 on the
    condition that a remote URL is not opened silently.
  - **Used by:** the URL field's confirm-on-remote (Task 1.8), and pair#293's
    Alt+click routing, which should prefer `open` for remote URLs.
- **Label** returns `name · host[:port]`, or just `name` when there's no host
  (about:blank, data:, file:). The result is sanitized by the strip's existing
  `rowtext.Sanitize`, not here. The strip is the single egress, as with tab
  names (see `strip.go` `chipLabel`).
  - **DRY rationale:** the strip and any future pane title derive the label from
    one function.
- **DefaultName** picks the first free name from `web`, `web-2`, `web-3`, …
  given the taken set. The taken set is this `pair term`'s tab names plus the
  names of the tag's live records (other split panes).
- **Version / ParseVersion / VersionNotice** parse `carbonyl --version` output
  (`Carbonyl 0.0.3`, `Carbonyl 0.0.3-next.ab80a27`). `VersionNotice` returns
  the notice text below `MinVersion = 0.0.3`, or `""`.
- **Argv** is the launch argv:
  `bin --remote-debugging-port=0 --user-data-dir=<profile> --fps=<fps> <url>`.
  `DefaultFPS = 15` (Log: `pair term` alone costs 21% of a core at 60 fps and 6%
  at 15 on a full-motion page; M2 re-measures the chain).
- **ParseActivePort / Endpoint** parse `<profile>/DevToolsActivePort`
  (`"<port>\n<ws path>\n"`) into `Endpoint{HTTP: "http://127.0.0.1:<port>", WS:
  "ws://127.0.0.1:<port><path>"}`. They reject non-numeric ports, ports outside
  1–65535, and paths not starting with `/devtools/browser/`.
- **OwnedName / ParseOwnedName** is the one naming scheme for every file this
  feature leaves on disk: `<ownerPID>-<birthHash12>-<tabID>`, where
  `birthHash12` is the first 12 hex digits of sha256 of the owner's
  `procutil.StrictIdentity`.
  - A profile dir is `<owned>`. A record is `<owned>.json`, and its atomic-write
    temp is `<owned>.json.tmp`.
  - Because every name carries its owner, a sweep proves the owner dead from the
    name alone. That covers a truncated record and a stranded temp, whose
    contents can't be trusted or decoded (plan-gate PQ-2).
  - **DRY rationale:** one scheme, one parser, one proof for profiles, records
    and temps.
- **Record** is the per-tab pointer an agent reads (fields in Task 3.1).
  `DecodeRecord` goes through `strictjson.Decode` (duplicate keys and trailing
  data rejected) and then `Validate`: version 1, non-empty name and tag, pids
  > 0, and a loopback `cdp_http`/`cdp_websocket`. A hand-edited record therefore
  can't send an agent to a remote host (ARCH-SECURE).
  - **Relationships:** 1:1 with a live browser tab. Written only by the tab's
    owner (`pair term`), or by the dead-owner sweep.
- **State / Event / Effect / Step** is the browser tab's lifecycle as one
  `(State, Event) → (State, []Effect)` function (ARCH-ORDER; enumeration below).
  `pair term`'s controller is the only caller, and nothing mutates `State`
  outside `Step`.
  - **Future extensions:** #293's "reuse the most recent unnamed tab" reads
    `State.Named` and last-use time; #285 (remember tabs) can persist
    `State.URL`.
- **StripField** generalizes today's `RenameField` with a `Prefix` (`rename` or
  `url`). The detached form (Tab out of range) draws the new-tab URL field as a
  trailing chip. That form already exists for a rename whose tab exited.
- **RenameEditor** is reused as the URL editor unchanged. Paste becomes
  insertable: a `uv.PasteEvent` inserts its runes with newlines and C0/C1
  controls dropped. Today rename drops paste entirely.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `CDPClient` | `cmd/internal/browsertab/cdp.go` | new | DevTools websocket (`coder/websocket`) |
| `ProfileStore` (Create / Sweep) | `cmd/internal/browsertab/profiledir.go` | new | filesystem under `os.UserCacheDir()/pair/browser` |
| `RecordStore` (Write / Remove / List / SweepDeadOwners) | `cmd/internal/browsertab/store.go` | new | `<scope>/browser-<tag>/` via artifactpath |
| `OwnerProbe` | `cmd/internal/browsertab/probe.go` | new | `storagegc.OSProcessProbe` + `procutil.StrictIdentity` |
| `fakecarbonyl` | `cmd/internal/browsertab/fakecarbonyl/fake.go` | new | stateful fake of the `carbonyl` binary |
| `Child.KillGroup` | `cmd/internal/ptychild/child.go` | new | `syscall.Kill(-pgid, SIGKILL)` |
| `browserController` | `cmd/internal/termcmd/browser.go` | new | goroutine running `Step` + effects |
| `terminalTab.kind` / `newBrowserTab` / field sessions | `cmd/internal/termcmd/presentation.go`, `run.go` | modified | pty children, strip, zellij focus |
| `Paths.BrowserDir` + family registration | `cmd/internal/artifactpath/{paths,manifest,gc}.go` | modified | pair data root |
| `browsercmd.Run` (`pair browser`) | `cmd/internal/browsercmd/browser.go` | new | stdout JSON, env scope |
| `carbonylconformance` probe | `cmd/probes/carbonylconformance/main.go` | new | the real `carbonyl` |

- **CDPClient** holds one websocket to the browser endpoint.
  - `Dial(ctx, ws)` sets a read limit of 1 MiB.
  - `DiscoverTargets(ctx)` sends `Target.setDiscoverTargets{discover:true}`.
  - `Events()` yields `TargetEvent{Kind: Created|Changed|Destroyed, Target{ID,
    Type, URL, Title}}`.
  - `Navigate(ctx, targetID, url)` sends `Target.attachToTarget{targetId,
    flatten:true}`, caches the `sessionId` per target, then sends `Page.navigate`
    with that `sessionId`.
  - `Done()` closes when the read loop ends; `Close()` shuts the connection.
  - **Injected into:** `browserController` through a `dial func(ctx, ws string)
    (devTools, error)` field, where `devTools` is the four-method interface
    `{Events; Navigate; Done; Close}`. Production passes `browsertab.Dial`.
    Tests run the same real client against `fakecarbonyl`'s real websocket, so
    the protocol itself is the seam (ARCH-MOCK: no function-call mock).
- **ProfileStore** has the root `os.UserCacheDir()/pair/browser`, deliberately
  **outside** the Pair data root: storagegc's inventory recurses into
  directories and turns every unknown file or symlink into a scope-wide blocker
  (`storagegc/inventory.go:176-256`), and a Chromium profile has thousands of
  files plus `Singleton*` symlinks.
  - `Create(owner, tabID)` makes the dir with mode 0700.
  - `Sweep(probe)` removes every dir whose parsed owner the probe proves dead.
  - **Injected into:** the controller (Create at launch, removal at close) and
    `newBrowserTab` (Sweep before Create).
- **RecordStore** is `Store{Dir: artifactpath.ResolveScoped(scope,
  tag).BrowserDir()}`. `Write` is atomic (temp + rename, 0600); `Remove`
  ignores ENOENT; `List` returns entries with per-file decode errors;
  `SweepDeadOwners(probe)` removes records whose `Owner` is proved dead.
- **OwnerProbe** is `Inspect(Identity) Liveness`, where `Liveness` is
  `storagegc.Liveness`: `ProcessAlive`, `ProcessDead` or `ProcessUnknown`. It
  reuses `storagegc.OSProcessProbe` (ARCH-DRY, already pid+birth reuse-safe)
  and adds `InspectHash(pid, birthHash12)` for profile names.
  - **Unknown never deletes.** The sweep removes only on `ProcessDead`.
- **fakecarbonyl** is the stateful fake. Its state: `{url, title, targetID,
  sessions map[sessionID]targetID, conns set}`, plus a helper child in the same
  process group. It speaks the same argv, draws the same row-0 bar and terminal
  modes, writes the same `DevToolsActivePort`, and serves the same
  `/json/version`, `/json/list` and browser websocket.
  - It's entered through `fakecarbonyl.Invoked()` / `Main(args)` from a
    package `TestMain`, so a test binary *is* the fake when launched with
    `PAIR_FAKE_CARBONYL=1`.
  - **Live conformance:** `cmd/probes/carbonylconformance` runs the same
    assertions against the real `carbonyl` (Task 2.6).
- **Child.KillGroup / `Options.KillGroup`** makes killing the whole process
  group structural (plan-gate PQ-7).
  - A child started with `Options{KillGroup: true}` has **every kill site in
    ptychild** send SIGKILL to `-pid` instead of just the leader: `Close()`, the
    pump's delivery-failure path, and its context-cancel path. So on every path
    that leads to a reap, the group is killed **before** the reap. That rules
    out a reaped leader followed by a recycled pgid, and helpers can't outlive
    it.
  - `KillGroup()` is the same kill exposed for the controller, which runs it
    early (before profile removal). It's a no-op once `c.done` is closed.
  - The ordering "controller Close before `child.Close`" in `removeTab` and
    `closeAll` is still followed, so the profile is removed after death. But
    the no-orphan guarantee no longer depends on that ordering.
  - Residual race: check → reap → pid reused → kill needs a pid wraparound
    within microseconds. It's the same window `os.Process.Signal` accepts, and
    it's documented in the method comment.
- **browserController** is one goroutine per browser tab, owning `State`. It
  reads events from a buffered channel (64) plus a close signal, calls `Step`,
  and runs effects in order. Async effects (Dial, Navigate) run in their own
  goroutines under the controller's context and report back as events. Its
  lifetime is the tab: `Close()` cancels, the loop runs `Step(Close)` effects,
  and `Close()` waits for exit, bounded at 3 s.

---

## Operating envelope and cross-cutting design

### ARCH-CONSTRAINTS

| Constraint | Budget / range | Basis | When exceeded |
|---|---|---|---|
| Alt+B → URL field visible | one strip repaint (<16 ms) | existing rename path | n/a |
| Enter → tab visible | Carbonyl paints in ~1 s | spike (first frame within 6 s window; DevToolsActivePort ~1 s) | Carbonyl's own startup; the strip shows the tab immediately |
| DevTools discovery | poll 50 ms, give up at 10 s | domain assumption; spike ~1 s | Degraded: timed notice, tab stays usable, no label/URL field |
| Idle CPU per tab | 0% (Carbonyl ≥0.0.3) | measured | <0.0.3: notice (Carbonyl spins a core) |
| Full-motion page | Carbonyl ~5% + `pair term` ~6% at 15 fps | measured | the cap bounds output linearly; M2 records zellij + couch |
| Hidden browser tab | still parsed by its endpoint (no paint) | `presentation.go:166-192` | bounded by the same cap |
| Record writes | ≤ 4/s per tab (250 ms coalescing) | page titles are page-controlled | coalesced; last write wins |
| Tab close / `pair term` exit | ≤ 2 s per tab for group death, run in parallel for all tabs | SIGKILL is immediate; poll `kill(-pgid,0)` for ESRCH | proceed; the profile sweep collects leftovers |
| Memory | ~150–300 MB RSS per tab (Chromium) | domain assumption, measured in M2 | no cap on tab count: operator-driven, one keypress each |

Workload class: interactive UI (keystroke → strip) plus a background stream
(Carbonyl frames). Concurrency extent: one controller goroutine per browser tab,
plus at most one in-flight Dial and one Navigate per tab. All are cancelled by
the tab's context and joined by `Close()`.

### ARCH-SECURE

- **Boundaries crossed:**
  - Carbonyl's DevTools output (page-controlled url and title) → strip and record.
  - `DevToolsActivePort` (a file in a dir we created) → `ParseActivePort`.
  - A record file (written by us, but read across a process boundary and
    hand-editable) → `pair browser` → an agent.
- **Typed at the boundary:** `ParseActivePort` checks port and path shape;
  `DecodeRecord` does strict JSON plus `Validate` (loopback addresses only,
  positive pids, version 1); CDP messages decode into typed structs, and
  unknown methods are ignored. Titles and URLs are sanitized at the strip egress
  (`rowtext.Sanitize`), and before they're written to the record
  (`rowtext.SanitizeAndFit(s, 512)`), so an agent printing the record can't be
  handed terminal escapes.
- **Truncated or older record:** `DecodeRecord` errors. `pair browser` reports
  `unreadable record <file>: <err>` on stderr, skips it, and never deletes it
  (it isn't the owner).
- **Credentials:** none. The DevTools port is full browser control for local
  processes; that's accepted because the profile is throwaway (Spec, Security).
  The profile dir is 0700, and records are 0600 in a scope dir that already
  holds the tag's other files.
- **Launch:** argv array, no shell; the URL is a single argv element. Navigate
  sends the URL as a JSON string.
- **Blast radius in tests:** every test uses `t.TempDir()` for the data root and
  sets `XDG_CACHE_HOME`/`HOME` so `os.UserCacheDir()` resolves under the temp
  dir (Task 1.4 pins this). The fake binary is the test binary, and no test
  runs the real `carbonyl`; only the opt-in probe does.

### Adversarial test strategy (plan-gate PQ-3)

Hand-written case tables are blind to the malformed input they didn't imagine.
So every function that parses bytes this feature didn't produce in-process gets
a Go native fuzz target (`testing.F`).
- **Seeds:** the table cases, plus the malformed forms in the Seeds column.
- **Properties:** asserted on every input. None of them restates the
  implementation.
- **Running:** the seed corpus runs as an ordinary test in `make test`. Each
  target also runs `go test -run=^$ -fuzz=^<Target>$ -fuzztime=15s <pkg>` at
  its milestone close (Tasks 1.9, 2.8, 3.5), and any crasher it finds lands in
  `testdata/fuzz/` as a permanent regression.

| Target | Input source | Seeds (beyond the table) | Properties |
|---|---|---|---|
| `FuzzNormalizeURL` (Task 1.1) | operator typing/paste, #293 Alt+click | `"\x1b[2J"`, `"a\u0085b"`, 64 KiB of `a`, `"http://"`, `"::"`, `"localhost:99999"`, `"127.0.0.1.evil.com:1"` | never panics; on success the output has no C0/C1/space, `url.Parse` succeeds with a non-empty scheme, and normalizing it again is a fixed point |
| `FuzzIsLocalURL` (Task 1.1) | same | `"http://127.0.0.1.evil.com/"`, `"http://localhost.evil.com/"`, `"http://[::1]/"`, `"http://0x7f.0.0.1/"`, `"http://127.1/"`, `"http://10.0.0.1/"`, `"http://user@localhost@evil.com/"` | never panics; **local is a closed set**: every string it calls local parses to a host that is loopback, RFC 1918, link-local, or a reserved local suffix — asserted by re-deriving from `net.ParseIP`/`url.Parse`, not by restating the predicate |
| `FuzzParseActivePort` (Task 1.1) | a file in a dir Chromium writes | `""`, `"\n"`, `"65536\n/devtools/browser/x"`, `"1\n/devtools/browser/\x00"`, `"1 \n/devtools/browser/x"`, CRLF | never panics; on success both addresses are exactly `http://127.0.0.1:<1-65535>` / `ws://127.0.0.1:<port>/devtools/browser/…` with no whitespace or control bytes |
| `FuzzParseOwnedName` (Task 1.1) | directory entries anyone can create | `"0-aaaaaaaaaaaa-1"`, `"01-aaaaaaaaaaaa-1"`, `"9999999999999999999-aaaaaaaaaaaa-1"`, uppercase hex, trailing `/` | never panics; on success `OwnedName` round-trips to the same string and pid > 0 |
| `FuzzDecodeCDPMessage` (Task 2.1) | websocket frames from Carbonyl (page-influenced) | truncated JSON, duplicate `id`, `id` as string, `params` as array, 1 MiB title, title with `\x1b]52;` | never panics; a decoded `TargetInfo`'s URL/Title pass through `rowtext.Sanitize` before any egress (asserted via `TargetEvent`'s constructor) |
| `FuzzDecodeRecord` (Task 3.1) | hand-editable record files | duplicate keys, trailing data, truncated, version 2, `cdp_http` of `http://127.0.0.1.evil.com:9` / `http://[::1]:9` / `http://10.0.0.1:9`, 1 MiB title, pid `-1` | never panics; on success `Validate()` holds, both addresses are loopback, and `EncodeRecord` → `DecodeRecord` round-trips |
| `FuzzStripLabel` (Task 2.4) | page-controlled url/title reaching the strip | titles/urls with ESC, OSC 52, BEL, C1, bidi overrides, wide runes | `RenderStrip(w, model)` output never contains a byte < 0x20, 0x7f, or a C1 rune, and its display width ≤ w |

### ARCH-ORDER — the browser tab state machine

States: `Launching`, `Connecting`, `Ready`, `Degraded`, `Closed`. Every state
also carries `Name`, `Named`, `Target{ID,URL,Title}`, `Pending` (the latest
navigate requested before a target is known) and `Endpoint`.

| State | Event | → State | Effects (in order) |
|---|---|---|---|
| Launching | PortFound(ep) | Connecting | Dial(ep) |
| Launching | PortTimeout | Degraded | Notice("DevTools unavailable") |
| Launching, Connecting | Navigate(u) | same | — (Pending = u, latest wins) |
| Connecting | Connected | Ready | — |
| Connecting | ConnectFailed(e) | Degraded | Notice("DevTools: " + e) |
| Ready | Target(t), no current target or t.ID == current | Ready | Relabel, Publish, then Navigate(Pending) if Pending ≠ "" (Pending = "") |
| Ready | Target(t), t.ID ≠ current target | Ready | — (Carbonyl shows one page; others ignored) |
| Ready | TargetGone(id == current) | Ready | Relabel, Publish (url/title empty) |
| Ready | Navigate(u), target known | Ready | Navigate(u) |
| Ready | Navigate(u), no target yet | Ready | — (Pending = u) |
| Ready | Disconnected | Degraded | Unpublish, Relabel, Notice("DevTools disconnected") |
| Degraded | Navigate(u) | Degraded | Notice("DevTools unavailable; use Carbonyl's own bar") |
| any but Closed | Renamed(n) | same | Relabel, Publish if Ready |
| any but Closed | Close | Closed | Unpublish, Hangup, KillGroup, RemoveProfile |
| Closed | Connected | Closed | Hangup (a late dial must not leak a socket) |
| Closed | anything else | Closed | — |

- **The event most likely to be mishandled:** a late `Connected` or `Target`
  arriving after `Close`. It must not publish a record for a dead browser. The
  `Closed` rows make that unrepresentable. `TestLateEventsAfterCloseNeverPublish`
  drives every event kind after `Close` and asserts no `Publish`, plus `Hangup`
  for `Connected`.
- **Uncertain outcomes:**
  - A `Navigate` whose response never arrives: the navigate goroutine is
    bounded by a 10 s context. Its outcome doesn't change state; the label moves
    only on `Target` events. So a lost response is harmless and nothing retries.
  - A failed `KillGroup`: EPERM can't happen for our own child; ESRCH is
    success. The controller then polls for group death, bounded at 2 s. On
    timeout it still removes the profile and logs. Pair can't prove more, and
    the sweep catches the residue.
- **Interrupting events that apply:**
  - Carbonyl exits by itself. Its tab's exit watcher sends Close.
  - Websocket loss (→ Disconnected).
  - `pair term` SIGHUP/SIGTERM. `closeAll` closes every controller in parallel.
  - `pair term` SIGKILL. There's no event: the kernel SIGHUPs the pty's group,
    and the dead-owner sweep handles the files.
  - A second actor (the agent) navigating: Target events relabel. No conflict,
    since the page is shared on purpose.
  - **Not applicable:** retries of an already-applied step (nothing retries),
    and out-of-order completions (Navigate completions carry no state).
- **Nondeterminism:** port-file timing, dial completion, websocket event order,
  and child exit timing. Tests inject order by calling `Step` directly for the
  machine, and by using fakecarbonyl env knobs for the integration tests:
  `PAIR_FAKE_CARBONYL_PORT_DELAY_MS` delays the port file, and
  `PAIR_FAKE_CARBONYL_NO_PORT=1` never writes it.
- **Structural enforcement:** `State` fields are unexported except through
  accessor methods (`Name()`, `Label()`, `Phase()`, `Target()`), so only `Step`
  can build a new `State`. The controller holds `State` in a field that only
  its loop goroutine touches.

### ARCH-FUNERAL

| Artifact | Created by | Last needed by | Removed by | Bound |
|---|---|---|---|---|
| Carbonyl process group | `newBrowserTab` (`ptychild.Start`) | its tab | `KillGroup` on Close; kernel SIGHUP if `pair term` dies | one per browser tab |
| Profile dir (~10–50 MB) | controller launch | Carbonyl | RemoveProfile on Close; else `ProfileStore.Sweep` at the next browser launch by any `pair term` of this user | one per tab, plus dead owners' dirs until the next launch |
| Record `browser-<tag>/<owned>.json` (~600 B) and its temp `<owned>.json.tmp` | controller Publish | agents via `pair browser` | Unpublish on Close removes both; else `SweepDeadOwners` (proof from the name, contents never trusted) at the next browser launch in the tag; storagegc recognizes both names and collects the dir with the tag | one record plus at most one temp per tab |
| `browser-<tag>/` directory | first Publish | — | storagegc with the tag (registered, Task 3.2) | one per tag that ever had a browser |
| Controller goroutines, websocket | tab creation | the tab | `Close()` joins them | per tab |

### ARCH-MOCK

- **Dependency surface consumed from Carbonyl:**
  - Argv flags `--remote-debugging-port=0`, `--user-data-dir`, `--fps`, and the
    positional URL.
  - `--version` output.
  - The `DevToolsActivePort` file.
  - `/json/version`.
  - The browser websocket: `Target.setDiscoverTargets`, the
    `targetCreated`/`targetInfoChanged`/`targetDestroyed` events,
    `Target.attachToTarget` (flatten) and `Page.navigate`.
  - Row-0 bar text (the probe only).
  - Terminal modes `?1049h ?1003h ?1006h`.
  - One process group; death on pty SIGHUP.
- **Fake:** `fakecarbonyl`, with the state model above. It's persisted only in
  its process, and the port file is in the profile dir.
- **Tests on the fake:** the controller integration tests (M2), the termcmd
  browser-tab tests (M1, M2), and the `pair browser` end-to-end test (M3).
- **Live conformance:** `make probe-carbonyl` runs
  `cmd/probes/carbonylconformance` against the real binary. It needs Carbonyl
  on PATH and skips with exit 0 and a message otherwise. It asserts:
  - version ≥ 0.0.3;
  - the port file appears;
  - `/json/version` contains `Carbonyl`;
  - navigate emits `targetInfoChanged` with the new url and title;
  - idle CPU stays under 5% over 5 s;
  - SIGKILL to the group leaves no processes;
  - pty-owner death kills the tree.
- **Cadence:** run it at every milestone close touching `browsertab`, and quote
  its output in the close evidence. It's a per-probe target under `cmd/probes/`
  because it imports `browsertab` (atlas/index.md rule).

### ARCH-PURE / ARCH-DRY

- **Pure, with tests that need no IO:** all of `url.go`, `label.go`,
  `name.go`, `version.go`, `launch.go`, `endpoint.go`, `owned.go`,
  `record.go` and `machine.go`.
- **IO seams:** `cdp.go`, `profiledir.go`, `store.go`, `probe.go`, the
  controller, and `browsercmd`.
- **Reused, not reimplemented:**
  - `rowtext.Sanitize` / `SanitizeAndFit` for page text;
  - `RenameEditor` and `DecodeRenameInput` for the URL field;
  - `strictjson.Decode` for records;
  - `storagegc.ProcessIdentity` and `OSProcessProbe` for liveness;
  - `artifactpath.Paths.tagged` for the record dir;
  - `workbenchshortcut`'s chord table, `GlobalBinding` and `RoleBinding` for
    the keys;
  - `layoutcmd.SwitchRightTerminalTab` for delivering Shift+Alt+B from other
    panes;
  - `dispatcher.Families()` for `pair browser`.

### Existing behavior this plan relies on (plan-gate PQ-1)

Each claim this plan makes about code or binaries that already exist is either
verified at a cited site or pinned by a named test in this plan. Nothing is
carried on trust. PQ-1 was an instance of this: the plan claimed Ctrl+U already
cleared the field, and it didn't.

| Claim | Status | Evidence / pin |
|---|---|---|
| Ctrl+U (`\x15`) clears a strip field | **false today**: `\x15` is swallowed (`rename_input.go`, `< 0x20` → Consume); `\x1b[127;9u` is Cmd+Backspace | Task 1.5 adds `{"\x15", RenameDeleteToStart}` + `TestCtrlUClearsTheField` |
| A pasted string reaches the pump as `uv.PasteEvent` | unverified | Task 1.5 `TestPasteInsertsThroughThePump` feeds `\x1b[200~…\x1b[201~` through `pumpStdinContext` |
| `pair term` reports mouse to zellij while the active child tracks it | cited: `terminal.ChildRequested` (`presentation.go:69`) | Task 2.5 `TestBrowserTabTurnsOnMouseReporting` asserts the host receives `?1003h` with a fake browser tab active |
| `uv.MouseClickEvent.Y` is 0-based, and the strip is row `rows-1` | unverified | Task 2.5 `TestStripClickHitsTheActiveChip` decodes a real SGR report for the last row |
| A strip-row release with no press isn't forwarded to the child | cited: presenter drops parent-area mouse (`presenter.go` `mouseInput`) | Task 2.5 asserts the fake child's `Writes()` stay empty |
| Chromium dies on pty SIGHUP; all Carbonyl processes share one pgid | measured (issue Log) | Task 2.6 probe checks 8 and 9 re-check it on every run |
| A Go program exits on SIGHUP by default (the fake's contract) | Go `os/signal` docs | Task 1.2 `TestFakeDiesOnSIGHUP` |
| `ptychild` reaps after **any** pump exit: pty EOF, `Close()` (context cancel after `Process.Kill()` of the leader), or a failed delivery (`ingest` error → `Process.Kill()` of the leader) | cited: `child.go` `pump()` and `Close()` (plan-gate PQ-7 corrected an earlier "EOF only" claim) | Task 1.3 `TestGroupKillOptionCoversEveryReapPath` |
| zellij `bind "Alt B"` delivers `\x1b[66;4u`, which nvim reads as `<M-B>` | by analogy with `Alt T` → `\x1b[84;4u` → `<M-T>` (live today) | the generated-Lua guard + operator smoke (Task 1.9) |
| storagegc blocks a scope on any unrecognized entry under the data root | cited: `storagegc/inventory.go:176-256` | why profiles live outside the root; Task 3.2/3.3 inventory test for records and temps |
| `coder/websocket` `Dial` sends no Origin header (Chrome 111 checks Origin) | read: `dial.go` sets none | Task 2.6 probe check 4 (a real dial) |

### ARCH-PURPOSE

Every Done-when bullet maps to a task: keys → 1.6/1.7; label → 2.4; URL field
→ 1.5 + 2.5; `pair browser` + DevTools test → 3.3/3.4; lifecycle → 1.3/1.8;
version notice → 1.4; cap and chain CPU → 2.7. Alt+click moved to pair#293 by
operator decision (issue Revisions); what #293 needs from here is the `Named`
flag (1.5, carried in `State` and in the record).

---

## Chunk 1: M1 — Browser tab lifecycle in `pair term`

M1 ends with a usable browser tab and no DevTools yet. Alt+B / Shift+Alt+B open
the URL field; Enter launches Carbonyl in a new tab; Alt+w, `pair term` exit and
`pair term` SIGKILL leave no processes; profiles are swept. The controller
exists but only handles `Close`; M2 adds the DevTools phases.

### Task 1.1: Pure launch helpers (url, label, name, version, argv, endpoint, profile name)

**Files:**
- Create: `cmd/internal/browsertab/url.go`, `label.go`, `name.go`, `version.go`, `launch.go`, `endpoint.go`, `owned.go`
- Test: `cmd/internal/browsertab/pure_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package browsertab

import "testing"

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"localhost:1111":         "http://localhost:1111",
		"127.0.0.1:8080/x":       "http://127.0.0.1:8080/x",
		"example.com/docs":       "https://example.com/docs",
		"http://a.test":          "http://a.test",
		"https://a.test/p?q=1":   "https://a.test/p?q=1",
		"about:blank":            "about:blank",
		"file:///tmp/x.html":     "file:///tmp/x.html",
		"  localhost:3000  ":     "http://localhost:3000",
		"data:text/html,<p>x</p>": "data:text/html,<p>x</p>",
	}
	for in, want := range cases {
		got, err := NormalizeURL(in)
		if err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "   ", "http://a\x1b[2J", "a b c"} {
		if _, err := NormalizeURL(bad); err == nil {
			t.Errorf("NormalizeURL(%q) accepted", bad)
		}
	}
}

func TestLabel(t *testing.T) {
	cases := []struct{ name, url, want string }{
		{"web", "http://localhost:1111/docs", "web · localhost:1111"},
		{"local-test", "https://example.com/", "local-test · example.com"},
		{"web", "about:blank", "web"},
		{"web", "", "web"},
		{"web", "file:///tmp/a.html", "web"},
	}
	for _, c := range cases {
		if got := Label(c.name, c.url); got != c.want {
			t.Errorf("Label(%q,%q) = %q; want %q", c.name, c.url, got, c.want)
		}
	}
}

func TestDefaultName(t *testing.T) {
	if got := DefaultName(nil); got != "web" {
		t.Fatalf("got %q", got)
	}
	if got := DefaultName(map[string]bool{"web": true, "web-2": true}); got != "web-3" {
		t.Fatalf("got %q", got)
	}
	if got := DefaultName(map[string]bool{"web-2": true}); got != "web" {
		t.Fatalf("got %q", got)
	}
}

func TestParseVersionAndNotice(t *testing.T) {
	for in, want := range map[string]Version{
		"Carbonyl 0.0.2\n":               {0, 0, 2},
		"Carbonyl 0.0.3":                 {0, 0, 3},
		"Carbonyl 0.0.3-next.ab80a27\n": {0, 0, 3},
		"Carbonyl 1.2.10":                {1, 2, 10},
	} {
		got, ok := ParseVersion(in)
		if !ok || got != want {
			t.Errorf("ParseVersion(%q) = %v,%v", in, got, ok)
		}
	}
	if _, ok := ParseVersion("garbage"); ok {
		t.Error("parsed garbage")
	}
	if VersionNotice(Version{0, 0, 2}) == "" {
		t.Error("0.0.2 must warn")
	}
	if VersionNotice(Version{0, 0, 3}) != "" || VersionNotice(Version{0, 1, 0}) != "" {
		t.Error("0.0.3+ must not warn")
	}
}

func TestArgv(t *testing.T) {
	got := Argv("/bin/carbonyl", "/p", 15, "http://x")
	want := []string{"/bin/carbonyl", "--remote-debugging-port=0", "--user-data-dir=/p", "--fps=15", "http://x"}
	if len(got) != len(want) {
		t.Fatalf("%q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%q", got)
		}
	}
}

func TestParseActivePort(t *testing.T) {
	ep, err := ParseActivePort("49932\n/devtools/browser/e986-4f\n")
	if err != nil || ep.HTTP != "http://127.0.0.1:49932" || ep.WS != "ws://127.0.0.1:49932/devtools/browser/e986-4f" {
		t.Fatalf("%+v %v", ep, err)
	}
	for _, bad := range []string{"", "0\n/devtools/browser/x", "70000\n/devtools/browser/x", "12\n/evil", "12", "x\n/devtools/browser/y"} {
		if _, err := ParseActivePort(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestOwnedNameRoundTrip(t *testing.T) {
	name := OwnedName(4242, "darwin-birth-token", 7)
	pid, hash, tab, ok := ParseOwnedName(name)
	if !ok || pid != 4242 || tab != 7 || hash != BirthHash("darwin-birth-token") || len(hash) != 12 {
		t.Fatalf("%q -> %d %q %d %v", name, pid, hash, tab, ok)
	}
	for _, bad := range []string{"", "x-y-z", "12-abc-3", "0-aaaaaaaaaaaa-1", "12-aaaaaaaaaaaa-x"} {
		if _, _, _, ok := ParseOwnedName(bad); ok {
			t.Errorf("parsed %q", bad)
		}
	}
}
```

Also write `cmd/internal/browsertab/fuzz_test.go`, following the strategy
table:

```go
func FuzzNormalizeURL(f *testing.F) {
	for _, s := range []string{"localhost:1111", "example.com/x", "about:blank", "\x1b[2J", "a\u0085b", "http://", "::", "localhost:99999", "127.0.0.1.evil.com:1", strings.Repeat("a", 1<<16)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out, err := NormalizeURL(in)
		if err != nil {
			return
		}
		for _, r := range out {
			if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) || unicode.IsSpace(r) {
				t.Fatalf("%q -> %q carries %U", in, out, r)
			}
		}
		if u, err := url.Parse(out); err != nil || u.Scheme == "" {
			t.Fatalf("%q -> %q unparseable: %v", in, out, err)
		}
		if again, err := NormalizeURL(out); err != nil || again != out {
			t.Fatalf("not a fixed point: %q -> %q -> %q (%v)", in, out, again, err)
		}
	})
}

func FuzzParseActivePort(f *testing.F) {
	for _, s := range []string{"49932\n/devtools/browser/e9\n", "", "\n", "65536\n/devtools/browser/x", "1\n/devtools/browser/\x00", "1 \n/devtools/browser/x", "1\r\n/devtools/browser/x\r\n"} {
		f.Add(s)
	}
	loop := regexp.MustCompile(`^http://127\.0\.0\.1:([1-9][0-9]{0,4})$`)
	f.Fuzz(func(t *testing.T, in string) {
		ep, err := ParseActivePort(in)
		if err != nil {
			return
		}
		m := loop.FindStringSubmatch(ep.HTTP)
		if m == nil || !strings.HasPrefix(ep.WS, "ws://127.0.0.1:"+m[1]+"/devtools/browser/") {
			t.Fatalf("%q -> %+v", in, ep)
		}
		for _, r := range ep.WS {
			if r <= 0x20 || r == 0x7f {
				t.Fatalf("%q -> ws %q", in, ep.WS)
			}
		}
	})
}

func FuzzParseOwnedName(f *testing.F) {
	for _, s := range []string{OwnedName(42, "b", 3), "0-aaaaaaaaaaaa-1", "01-aaaaaaaaaaaa-1", "9999999999999999999-aaaaaaaaaaaa-1", "1-AAAAAAAAAAAA-1", "1-aaaaaaaaaaaa-1/"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		pid, hash, tab, ok := ParseOwnedName(in)
		if !ok {
			return
		}
		if pid <= 0 || fmt.Sprintf("%d-%s-%d", pid, hash, tab) != in {
			t.Fatalf("%q -> %d %q %d", in, pid, hash, tab)
		}
	})
}
```

(`ParseActivePort` must therefore reject a `\r` in either line. The fuzz
property enforces that, so the implementation trims only `\n` and rejects any
other whitespace or control byte.)

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/browsertab/ -run 'TestNormalizeURL|TestLabel|TestDefaultName|TestParseVersion|TestArgv|TestParseActivePort|TestOwnedName|Fuzz' -count=1`
Expected: FAIL (package does not compile: undefined functions).

- [ ] **Step 3: Implement**

`url.go`:

```go
// Package browsertab is the pure core of pair term's Carbonyl browser tab
// (#292): launch argv, URLs, labels, names, the record codec, and the tab's
// lifecycle state machine. The IO seams live beside it (cdp.go, store.go,
// profiledir.go, probe.go) so the boundary is visible from the file list.
package browsertab

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

// NormalizeURL turns operator input into something Chrome navigates to.
// A bare host:port is a local dev server, so it gets http; any other bare
// host gets https. Schemes Chrome understands pass through unchanged.
func NormalizeURL(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", errors.New("empty URL")
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", errors.New("URL contains spaces or control characters")
		}
	}
	lower := strings.ToLower(s)
	for _, p := range []string{"http://", "https://", "file://", "about:", "data:", "chrome://"} {
		if strings.HasPrefix(lower, p) {
			return s, nil
		}
	}
	host := s
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	scheme := "https://"
	if h, _, found := strings.Cut(host, ":"); found && (h == "localhost" || isIPv4(h)) || host == "localhost" || isIPv4(host) {
		scheme = "http://"
	}
	u, err := url.Parse(scheme + s)
	if err != nil || u.Host == "" {
		return "", errors.New("not a URL")
	}
	return u.String(), nil
}

func isIPv4(h string) bool {
	parts := strings.Split(h, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" || len(p) > 3 || strings.Trim(p, "0123456789") != "" {
			return false
		}
	}
	return true
}
```

`label.go`:

```go
package browsertab

import "net/url"

// Label is a browser tab's strip text: its name and the site it shows. The
// strip sanitizes it at egress (termcmd chipLabel), like every tab name.
func Label(name, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return name
	}
	return name + " · " + u.Host
}
```

`name.go`:

```go
package browsertab

import "strconv"

// DefaultName is the first free automatic name: web, web-2, web-3, ...
// taken holds every name in use across the tag, so a handle is unique.
func DefaultName(taken map[string]bool) string {
	if !taken["web"] {
		return "web"
	}
	for i := 2; ; i++ {
		if n := "web-" + strconv.Itoa(i); !taken[n] {
			return n
		}
	}
}
```

`version.go`:

```go
package browsertab

import (
	"fmt"
	"regexp"
	"strconv"
)

type Version struct{ Major, Minor, Patch int }

// MinVersion is the first Carbonyl whose render loop does not spin a core
// when idle (upstream b4ab3a87, "fix idling CPU usage (#126)"; measured in
// #292's Log: 0.0.2 ~100% idle, 0.0.3 0.0%).
var MinVersion = Version{0, 0, 3}

var versionPattern = regexp.MustCompile(`Carbonyl (\d+)\.(\d+)\.(\d+)`)

func ParseVersion(out string) (Version, bool) {
	m := versionPattern.FindStringSubmatch(out)
	if m == nil {
		return Version{}, false
	}
	var v [3]int
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return Version{}, false
		}
		v[i] = n
	}
	return Version{v[0], v[1], v[2]}, true
}

func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	return v.Patch < o.Patch
}

func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

// VersionNotice is the strip notice for an old Carbonyl, or "".
func VersionNotice(v Version) string {
	if !v.Less(MinVersion) {
		return ""
	}
	return fmt.Sprintf("carbonyl %s spins a CPU core while idle; install >= %s (github.com/fathyb/carbonyl/releases)", v, MinVersion)
}
```

`launch.go`:

```go
package browsertab

import "strconv"

// DefaultFPS caps Carbonyl's frame rate. Output scales linearly with it and
// pair term, zellij and couch each re-parse every frame; measured in #292's
// Log (pair term: 21% of a core at 60 fps, 6% at 15, full-motion page).
const DefaultFPS = 15

// Argv is the Carbonyl launch argv. --user-data-dir is mandatory: without it
// Carbonyl writes its profile into its own install directory (#292 Log).
func Argv(bin, profile string, fps int, url string) []string {
	return []string{bin, "--remote-debugging-port=0", "--user-data-dir=" + profile, "--fps=" + strconv.Itoa(fps), url}
}
```

`endpoint.go`:

```go
package browsertab

import (
	"errors"
	"strconv"
	"strings"
)

// Endpoint is a browser's DevTools address, loopback by construction.
type Endpoint struct{ HTTP, WS string }

// ActivePortFile is the file Chromium writes under --user-data-dir when
// --remote-debugging-port=0 picks a port.
const ActivePortFile = "DevToolsActivePort"

func ParseActivePort(content string) (Endpoint, error) {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) < 2 {
		return Endpoint{}, errors.New("DevToolsActivePort: want port and path")
	}
	for _, line := range lines[:2] {
		for i := 0; i < len(line); i++ {
			if line[i] <= 0x20 || line[i] == 0x7f {
				return Endpoint{}, errors.New("DevToolsActivePort: whitespace or control byte")
			}
		}
	}
	port, err := strconv.Atoi(lines[0])
	if err != nil || port < 1 || port > 65535 || lines[0][0] == '0' || lines[0][0] == '+' {
		return Endpoint{}, errors.New("DevToolsActivePort: bad port")
	}
	path := lines[1]
	if !strings.HasPrefix(path, "/devtools/browser/") || len(path) == len("/devtools/browser/") {
		return Endpoint{}, errors.New("DevToolsActivePort: bad path")
	}
	host := "127.0.0.1:" + strconv.Itoa(port)
	return Endpoint{HTTP: "http://" + host, WS: "ws://" + host + path}, nil
}
```

`owned.go`:

```go
package browsertab

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
)

// BirthHash is a filename-safe digest of a procutil.StrictIdentity token.
func BirthHash(birth string) string {
	sum := sha256.Sum256([]byte(birth))
	return hex.EncodeToString(sum[:])[:12]
}

// OwnedName is the one name scheme for everything a browser tab leaves on disk
// (profile dir <owned>, record <owned>.json, its temp <owned>.json.tmp): the
// owner is in the name, so a sweep can prove it dead without trusting contents.
func OwnedName(ownerPID int, ownerBirth string, tabID int) string {
	return fmt.Sprintf("%d-%s-%d", ownerPID, BirthHash(ownerBirth), tabID)
}

var ownedNamePattern = regexp.MustCompile(`^([1-9][0-9]*)-([0-9a-f]{12})-([0-9]+)$`)

func ParseOwnedName(name string) (pid int, birthHash string, tabID int, ok bool) {
	m := ownedNamePattern.FindStringSubmatch(name)
	if m == nil {
		return 0, "", 0, false
	}
	pid, err1 := strconv.Atoi(m[1])
	tabID, err2 := strconv.Atoi(m[3])
	if err1 != nil || err2 != nil {
		return 0, "", 0, false
	}
	return pid, m[2], tabID, true
}
```

(If the `NormalizeURL` host-detection condition reads awkwardly while
implementing, restructure it into a small `isLocalHost(host string) bool`. The
tests are the contract.)

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/internal/browsertab/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/browsertab/
git commit -m "#292 M1: browsertab pure launch helpers"
```

### Task 1.2: `fakecarbonyl` — process shape (M1 part)

The M1 fake covers the process contract. It parses argv, answers `--version`,
draws the row-0 bar with Carbonyl's terminal modes, and writes
`DevToolsActivePort`. It starts a helper child in its own process group, writes
the helper's pid to `<profile>/helper.pid`, and dies on SIGHUP/SIGTERM (Go's
default). M2 adds the HTTP and websocket server; until then the port file
names a port nothing listens on. That's fine for M1, which never dials.

**Files:**
- Create: `cmd/internal/browsertab/fakecarbonyl/fake.go`
- Create: `cmd/internal/browsertab/fakecarbonyl/fake_test.go`

- [ ] **Step 1: Write the failing test** (runs the test binary as the fake under a real pty)

```go
package fakecarbonyl_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/browsertab/fakecarbonyl"
)

func TestMain(m *testing.M) {
	if fakecarbonyl.Invoked() {
		os.Exit(fakecarbonyl.Main(os.Args[1:]))
	}
	os.Exit(m.Run())
}

func TestFakeDrawsBarWritesPortAndDiesWithItsGroup(t *testing.T) {
	prof := t.TempDir()
	cmd := exec.Command(os.Args[0], "--remote-debugging-port=0", "--user-data-dir="+prof, "--fps=15", "http://localhost:1111/")
	cmd.Env = append(os.Environ(), fakecarbonyl.EnvInvoke+"=1")
	f, err := pty.Start(cmd)
	if err != nil {
		t.Skipf("no pty here: %v", err) // sandboxed runs cannot allocate a pty
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	out := string(buf[:n])
	for _, want := range []string{"\x1b[?1049h", "\x1b[?1003h", "\x1b[?1006h", "[❮][❯][↻][ http://localhost:1111/"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	waitFile(t, filepath.Join(prof, "DevToolsActivePort"))
	helper := readPID(t, filepath.Join(prof, "helper.pid"))
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(helper, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("helper %d survived its group's SIGKILL", helper)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestFakeDiesOnSIGHUP(t *testing.T) {
	prof := t.TempDir()
	cmd := exec.Command(os.Args[0], "--remote-debugging-port=0", "--user-data-dir="+prof, "about:blank")
	cmd.Env = append(os.Environ(), fakecarbonyl.EnvInvoke+"=1")
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waitFile(t, filepath.Join(prof, "DevToolsActivePort"))
	_ = cmd.Process.Signal(syscall.SIGHUP)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("fake survived SIGHUP; real Carbonyl dies on it (#292 Log)")
	}
	_ = syscall.Kill(readPID(t, filepath.Join(prof, "helper.pid")), syscall.SIGKILL) // not in a pty session here
}

func TestFakeVersion(t *testing.T) {
	cmd := exec.Command(os.Args[0], "--version")
	cmd.Env = append(os.Environ(), fakecarbonyl.EnvInvoke+"=1", fakecarbonyl.EnvVersion+"=0.0.2")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "Carbonyl 0.0.2" {
		t.Fatalf("%q %v", out, err)
	}
}

func waitFile(t *testing.T, p string) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if _, err := os.Stat(p); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s never appeared", p)
}

func readPID(t *testing.T, p string) int {
	t.Helper()
	waitFile(t, p)
	b, _ := os.ReadFile(p)
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return pid
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/browsertab/fakecarbonyl/ -count=1`
Expected: FAIL (package missing).

- [ ] **Step 3: Implement `fake.go`**

```go
// Package fakecarbonyl is the stateful fake of the Carbonyl binary (#292,
// ARCH-MOCK). A test binary becomes the fake when launched with
// PAIR_FAKE_CARBONYL=1: its TestMain calls Invoked/Main before flag parsing.
// cmd/probes/carbonylconformance checks the same contract against the real
// binary, so a divergence is detected rather than trusted.
package fakecarbonyl

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	EnvInvoke    = "PAIR_FAKE_CARBONYL"
	EnvHelper    = "PAIR_FAKE_CARBONYL_HELPER"
	EnvVersion   = "PAIR_FAKE_CARBONYL_VERSION"    // default 0.0.3
	EnvNoPort    = "PAIR_FAKE_CARBONYL_NO_PORT"    // "1": never write DevToolsActivePort
	EnvPortDelay = "PAIR_FAKE_CARBONYL_PORT_DELAY_MS"
)

func Invoked() bool { return os.Getenv(EnvInvoke) == "1" || os.Getenv(EnvHelper) == "1" }

// Main runs the fake and returns its exit code.
func Main(args []string) int {
	if os.Getenv(EnvHelper) == "1" {
		// A Chromium helper stand-in: lives until its group is killed.
		select {}
	}
	fs := flag.NewFlagSet("carbonyl", flag.ContinueOnError)
	port := fs.Int("remote-debugging-port", -1, "")
	profile := fs.String("user-data-dir", "", "")
	_ = fs.Int("fps", 60, "")
	version := fs.Bool("version", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		v := os.Getenv(EnvVersion)
		if v == "" {
			v = "0.0.3"
		}
		fmt.Println("Carbonyl " + v)
		return 0
	}
	if *profile == "" || *port != 0 {
		fmt.Fprintln(os.Stderr, "fakecarbonyl: want --remote-debugging-port=0 --user-data-dir=DIR")
		return 2
	}
	url := "about:blank"
	if fs.NArg() > 0 {
		url = fs.Arg(0)
	}
	b := newBrowser(*profile, url) // M2 grows this into the DevTools server
	b.draw()
	if err := b.startHelper(); err != nil {
		fmt.Fprintln(os.Stderr, "fakecarbonyl:", err)
		return 1
	}
	if os.Getenv(EnvNoPort) != "1" {
		if ms, _ := strconv.Atoi(os.Getenv(EnvPortDelay)); ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
		if err := b.writePortFile(); err != nil {
			fmt.Fprintln(os.Stderr, "fakecarbonyl:", err)
			return 1
		}
	}
	// SIGHUP/SIGTERM keep Go's default action (exit), as Chromium dies on the
	// pty SIGHUP (#292 Log). Block until then.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	<-sig
	return 0
}

type browser struct {
	profile  string
	url      string
	title    string
	targetID string
	port     int // 0 until M2's server listens
}

func newBrowser(profile, url string) *browser {
	return &browser{profile: profile, url: url, title: "Fake " + url, targetID: "FAKE-TARGET-1"}
}

// draw emits Carbonyl's startup modes and row-0 navigation bar.
func (b *browser) draw() {
	fmt.Printf("\x1b[?1049h\x1b[?1003h\x1b[?1006h\x1b[?25l\x1b[1;1H[❮][❯][↻][ %s ]", b.url)
}

func (b *browser) startHelper() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self)
	cmd.Env = append(os.Environ(), EnvHelper+"=1")
	if err := cmd.Start(); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.profile, "helper.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o600)
}

func (b *browser) writePortFile() error {
	port := b.port
	if port == 0 {
		port = 1 // M1: nothing listens; M2 replaces with the real listener
	}
	content := fmt.Sprintf("%d\n/devtools/browser/FAKE-BROWSER\n", port)
	tmp := filepath.Join(b.profile, ".DevToolsActivePort.tmp")
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(b.profile, "DevToolsActivePort"))
}
```

- [ ] **Step 4: Run tests** (outside the sandbox: pty allocation is blocked in
  it, see memory `sandbox_blocks_pty_child_tests`)

Run: `go test ./cmd/internal/browsertab/fakecarbonyl/ -count=1 -v`
Expected: PASS for both tests (or SKIP for the pty test when sandboxed; in that
case rerun unsandboxed before committing).

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/browsertab/fakecarbonyl/
git commit -m "#292 M1: fakecarbonyl process contract (bar, modes, port file, helper group)"
```

### Task 1.3: `ptychild.Child.KillGroup`

**Files:**
- Modify: `cmd/internal/ptychild/child.go`:
  - `Options.KillGroup bool`, stored on `Child` as `killGroup`;
  - a `kill()` helper used at all three current `c.cmd.Process.Kill()` sites
    (the pump's delivery-failure and context-cancel paths, and `Close()`);
  - the exported `KillGroup()` after `Signal`.
- Modify: `cmd/internal/ptychild/fake.go` (fake branch)
- Test: `cmd/internal/ptychild/child_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestKillGroupReachesGrandchildren(t *testing.T) {
	// sh starts a background sleeper in its own group (no job control in -c),
	// then waits; both must die from one KillGroup.
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "bg.pid")
	c, err := Start(Options{Argv: []string{"/bin/sh", "-c", "sleep 60 & echo $! > " + pidFile + "; wait"}, Size: Size{Rows: 10, Cols: 40}})
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	defer c.Close()
	var bg int
	for i := 0; i < 200 && bg == 0; i++ {
		b, _ := os.ReadFile(pidFile)
		bg, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		time.Sleep(10 * time.Millisecond)
	}
	if bg == 0 {
		t.Fatal("background pid never written")
	}
	if err := c.KillGroup(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(bg, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d survived KillGroup", bg)
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case <-c.Exited():
	case <-time.After(2 * time.Second):
		t.Fatal("leader not reaped after KillGroup")
	}
}

// TestGroupKillOptionCoversEveryReapPath pins PQ-7: with Options.KillGroup,
// every path that ends in a reap kills the group first. The grandchild ignores
// SIGHUP, so the kernel's pty hangup cannot be what kills it -- only the group
// SIGKILL can.
func TestGroupKillOptionCoversEveryReapPath(t *testing.T) {
	for _, path := range []string{"close", "delivery-failure"} {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			pidFile := filepath.Join(dir, "bg.pid")
			var failed atomic.Bool
			opts := Options{
				Argv:      []string{"/bin/sh", "-c", `(trap "" HUP; exec sleep 60) & echo $! > ` + pidFile + `; echo ready; wait`},
				Size:      Size{Rows: 10, Cols: 40},
				KillGroup: true,
			}
			if path == "delivery-failure" {
				opts.Sink = func(ctx context.Context, b OutputBatch) error {
					if bytes.Contains(b.Raw, []byte("ready")) {
						failed.Store(true)
						return errors.New("sink refuses")
					}
					return nil
				}
			}
			c, err := Start(opts)
			if err != nil {
				t.Skipf("no pty: %v", err)
			}
			bg := waitPIDFile(t, pidFile)
			if path == "close" {
				_ = c.Close()
			} else {
				<-c.Exited()
				if !failed.Load() {
					t.Fatal("precondition: the sink never failed")
				}
				defer c.Close()
			}
			assertProcessGone(t, bg, 2*time.Second)
		})
	}
}

func TestWithoutTheOptionCloseKillsOnlyTheLeader(t *testing.T) {
	// Documents why the option exists: the SIGHUP-immune grandchild survives a
	// plain Close. The test kills it itself afterwards.
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "bg.pid")
	c, err := Start(Options{Argv: []string{"/bin/sh", "-c", `(trap "" HUP; exec sleep 60) & echo $! > ` + pidFile + `; wait`}, Size: Size{Rows: 10, Cols: 40}})
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	bg := waitPIDFile(t, pidFile)
	_ = c.Close()
	defer syscall.Kill(bg, syscall.SIGKILL)
	time.Sleep(200 * time.Millisecond)
	if syscall.Kill(bg, 0) != nil {
		t.Skip("platform killed it anyway; the option is still the guarantee")
	}
}

func TestKillGroupAfterReapIsANoop(t *testing.T) {
	c, err := Start(Options{Argv: []string{"/bin/sh", "-c", "exit 0"}, Size: Size{Rows: 10, Cols: 40}})
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	defer c.Close()
	<-c.Exited()
	if err := c.KillGroup(); err != nil {
		t.Fatalf("KillGroup after reap must be a no-op, got %v", err)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/ptychild/ -run TestKillGroup -count=1`
Expected: FAIL (`KillGroup` undefined).

- [ ] **Step 3: Implement**

Replace **every** leader-only kill before a reap with `c.kill()`: the two in
`pump()` (delivery failure, context cancel), the one in `Close()`, **and** the
one in `Start()`'s `initTerminal`-failure path (`cmd.Process.Kill()` then
`cmd.Wait()`). The fourth came from the round-3 plan-gate note. A `grep -n
'Process.Kill' cmd/internal/ptychild/child.go` after the change should show
only the body of `kill()` itself:

```go
// kill ends the child for every ptychild path that is about to reap it
// (pump delivery failure, pump context cancel, Close). With Options.KillGroup
// it SIGKILLs the whole group BEFORE the reap, so no path reaps a leader whose
// helpers are still alive (#292 PQ-7).
func (c *Child) kill() {
	if c.killGroup {
		_ = c.KillGroup()
		return
	}
	_ = c.cmd.Process.Kill()
}
```

```go
// KillGroup SIGKILLs the child's whole process group. pty.Start makes the
// child a session and group leader, so pgid == pid, and a program that spawns
// helpers (Chromium: GPU, renderer, network) keeps them in that group (#292
// Log: all six Carbonyl processes shared one pgid).
//
// It signals only while the leader is unreaped (c.done open), because the
// pgid is the leader's pid and a reaped pid can be reused. ptychild reaps on
// three paths -- pty EOF, Close, and a failed delivery -- and with
// Options.KillGroup the latter two call this before reaping; on EOF every slave
// holder is already gone. After the reap it is a no-op. The remaining window
// (check, reap, pid reuse, kill) needs a pid wraparound in microseconds, the
// same window os.Process.Signal accepts.
func (c *Child) KillGroup() error {
	if c.fake != nil {
		return c.fakeSignal(syscall.SIGKILL)
	}
	select {
	case <-c.done:
		return nil
	default:
	}
	pid := c.PID()
	if pid <= 0 {
		return fmt.Errorf("ptychild: no process")
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
```

(`fakeSignal` on an ended fake returns an error today. KillGroup must be a
no-op after the end, so the fake branch reads `if c.Done() { return nil };
return c.fakeSignal(syscall.SIGKILL)`. `Child.Done() bool` is at
child.go:225 and `Exited() <-chan struct{}` at :238.)

- [ ] **Step 4: Run tests unsandboxed**

Run: `go test ./cmd/internal/ptychild/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/ptychild/
git commit -m "#292 M1: ptychild KillGroup (SIGKILL the pty child's process group while unreaped)"
```

### Task 1.4: Profile store, owner probe, and the launcher config

**Files:**
- Create: `cmd/internal/browsertab/probe.go`, `profiledir.go`, `launcher.go`
- Test: `cmd/internal/browsertab/profiledir_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package browsertab

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type fakeProbe map[int]storagegc.Liveness

func (f fakeProbe) InspectHash(pid int, _ string) storagegc.Liveness { return f[pid] }
func (f fakeProbe) Inspect(id Identity) storagegc.Liveness      { return f[id.PID] }

func TestSweepRemovesOnlyProvablyDeadOwners(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"100-aaaaaaaaaaaa-1", "200-bbbbbbbbbbbb-1", "300-cccccccccccc-2", "not-ours"} {
		if err := os.MkdirAll(filepath.Join(root, n, "Default"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	store := ProfileStore{Root: root}
	removed, err := store.Sweep(fakeProbe{100: storagegc.ProcessDead, 200: storagegc.ProcessAlive, 300: storagegc.ProcessUnknown})
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || filepath.Base(removed[0]) != "100-aaaaaaaaaaaa-1" {
		t.Fatalf("removed %v", removed)
	}
	for _, keep := range []string{"200-bbbbbbbbbbbb-1", "300-cccccccccccc-2", "not-ours"} {
		if _, err := os.Stat(filepath.Join(root, keep)); err != nil {
			t.Errorf("%s removed: %v", keep, err)
		}
	}
}

func TestCreateProfileIsPrivate(t *testing.T) {
	store := ProfileStore{Root: filepath.Join(t.TempDir(), "browser")}
	dir, err := store.Create(Identity{PID: 42, Birth: "b"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{store.Root, dir} {
		st, err := os.Stat(p)
		if err != nil || st.Mode().Perm() != 0o700 {
			t.Fatalf("%s mode %v %v", p, st.Mode(), err)
		}
	}
	if filepath.Base(dir) != OwnedName(42, "b", 3) {
		t.Fatalf("dir %s", dir)
	}
}

func TestDefaultProfileRootHonorsCacheEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp) // linux
	t.Setenv("HOME", tmp)           // darwin: ~/Library/Caches
	root, err := DefaultProfileRoot()
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(tmp, root); err != nil || rel == ".." || filepath.IsAbs(rel) || len(rel) >= 2 && rel[:2] == ".." {
		t.Fatalf("root %s escaped the test home %s", root, tmp)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/browsertab/ -run 'TestSweep|TestCreateProfile|TestDefaultProfileRoot' -count=1`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement**

`probe.go`:

```go
package browsertab

import (
	"strconv"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// Identity is a process as a record or profile name remembers it.
type Identity struct {
	PID   int    `json:"pid"`
	Birth string `json:"birth"`
}

// CurrentIdentity is this process's identity, or an error when the platform
// cannot prove birth (then nothing pair term creates can later be swept).
func CurrentIdentity(pid int) (Identity, error) {
	id, err := storagegc.CurrentProcessIdentity(pid)
	if err != nil {
		return Identity{}, err
	}
	return Identity{PID: id.PID, Birth: id.Birth}, nil
}

// OwnerProbe answers "is this process still the one that was recorded".
// Unknown never licenses a deletion.
type OwnerProbe interface {
	Inspect(Identity) storagegc.Liveness
	InspectHash(pid int, birthHash string) storagegc.Liveness
}

type OSOwnerProbe struct{}

func (OSOwnerProbe) Inspect(id Identity) storagegc.Liveness {
	return storagegc.OSProcessProbe{}.Inspect(storagegc.ProcessIdentity{PID: id.PID, Birth: id.Birth})
}

func (OSOwnerProbe) InspectHash(pid int, birthHash string) storagegc.Liveness {
	birth := procutil.StrictIdentity(strconv.Itoa(pid))
	if birth == "" {
		// Dead, or identity unavailable: disambiguate through Inspect's
		// kill(0) path with a birth that cannot match.
		if storagegc.OSProcessProbe{}.Inspect(storagegc.ProcessIdentity{PID: pid, Birth: "\x00"}) == storagegc.ProcessDead {
			return storagegc.ProcessDead
		}
		return storagegc.ProcessUnknown
	}
	if BirthHash(birth) != birthHash {
		return storagegc.ProcessDead
	}
	return storagegc.ProcessAlive
}
```

(`storagegc.Liveness` is a string type with `ProcessAlive`, `ProcessDead`
and `ProcessUnknown`, at storagegc/process.go:18-23.)

`profiledir.go`:

```go
package browsertab

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// DefaultProfileRoot is where browser profiles live: the user cache dir, NOT
// the Pair data root. storagegc walks the data root and blocks a whole scope
// on any file it does not recognise; a Chromium profile is thousands of them.
func DefaultProfileRoot() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pair", "browser"), nil
}

type ProfileStore struct{ Root string }

func (s ProfileStore) Create(owner Identity, tabID int) (string, error) {
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(s.Root, 0o700); err != nil {
		return "", err
	}
	dir := filepath.Join(s.Root, OwnedName(owner.PID, owner.Birth, tabID))
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// Remove deletes one profile dir; a missing dir is success.
func (s ProfileStore) Remove(dir string) error {
	if filepath.Dir(dir) != s.Root {
		return errors.New("browsertab: refusing to remove a dir outside the profile root")
	}
	return os.RemoveAll(dir)
}

// Sweep removes the profiles whose owner is provably dead. A name it cannot
// parse is not ours to judge and stays.
func (s ProfileStore) Sweep(probe OwnerProbe) ([]string, error) {
	entries, err := os.ReadDir(s.Root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var removed []string
	var errs []error
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, hash, _, ok := ParseOwnedName(e.Name())
		if !ok || probe.InspectHash(pid, hash) != storagegc.ProcessDead {
			continue
		}
		dir := filepath.Join(s.Root, e.Name())
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, err)
			continue
		}
		removed = append(removed, dir)
	}
	return removed, errors.Join(errs...)
}
```

`launcher.go` is the config `pair term` needs to launch, resolved once:

```go
package browsertab

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// EnvBinary overrides the carbonyl binary (tests point it at a fakecarbonyl
// test binary; operators can point it at a release zip).
const EnvBinary = "PAIR_CARBONYL"

// ResolveBinary finds carbonyl: $PAIR_CARBONYL, else PATH.
func ResolveBinary() (string, error) {
	if b := os.Getenv(EnvBinary); b != "" {
		return b, nil
	}
	return exec.LookPath("carbonyl")
}

// ProbeVersion runs `bin --version` bounded at 2 s. Env carries test knobs.
func ProbeVersion(ctx context.Context, bin string, env []string) (Version, bool) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--version")
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		return Version{}, false
	}
	return ParseVersion(string(out))
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/internal/browsertab/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/browsertab/
git commit -m "#292 M1: browser profile store, owner probe, binary resolution"
```

### Task 1.5: Strip field generalization, paste, labels, named flag

**Files:**
- Modify: `cmd/internal/termcmd/strip.go` (`TabChip`, `RenameField` → `StripField{Tab, Prefix, Text}`, `renameChip` → `fieldChip(prefix, text)`)
- Modify: `cmd/internal/termcmd/presentation.go` (`terminalTab` fields `kind`, `named`, `label`; `stripModelLocked` uses `label` when non-empty; `activeRename` → `activeField{tabID, prefix, editor}`)
- Modify: `cmd/internal/termcmd/run.go` pump: the rename branch becomes a field branch; `uv.PasteEvent` inserts
- Test: `cmd/internal/termcmd/strip_test.go`, `rename_input_test.go` or a new `field_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestStripDrawsURLFieldDetachedForANewTab(t *testing.T) {
	got := RenderStrip(60, StripModel{
		Tabs:   []TabChip{{Name: "terminal 1"}},
		Active: 0,
		Field:  &StripField{Tab: -1, Prefix: "url", Text: "localhost:11│"},
	})
	if got.Body != "[terminal 1] [url: localhost:11│]" {
		t.Fatalf("%q", got.Body)
	}
}

func TestStripDrawsBrowserLabelSanitized(t *testing.T) {
	got := RenderStrip(60, StripModel{Tabs: []TabChip{{Name: "web · localhost:1111\x1b[2J"}}, Active: 0})
	if strings.Contains(got.Body, "\x1b") || !strings.HasPrefix(got.Body, "[web · localhost:1111") {
		t.Fatalf("%q", got.Body)
	}
}

func TestCtrlUClearsTheField(t *testing.T) {
	_, events, _ := DecodeRenameInput(RenameDecoderState{}, []byte("\x15"), false, false)
	if len(events) != 1 || events[0].Kind != RenameDeleteToStart {
		t.Fatalf("%+v", events)
	}
	e := NewRenameEditor("http://localhost:1111/")
	e, _ = e.Apply(events[0])
	if e.Text() != "" {
		t.Fatalf("%q", e.Text())
	}
}

func TestPasteInsertsIntoAnOpenField(t *testing.T) {
	e := NewRenameEditor("")
	e = InsertPaste(e, "http://x.test/\n\x1b[31mred")
	if e.Text() != "http://x.test/[31mred" {
		t.Fatalf("%q", e.Text())
	}
}
```

Also add `TestPasteInsertsThroughThePump`, in `run_test.go` style with
`fakeMux`: open a field with Alt+r, feed `"\x1b[200~http://p.test/\x1b[201~"`
through `pumpStdinContext`, then `\r`. Assert `finishField` received the
commit `http://p.test/`. This pins that paste reaches the pump as
`uv.PasteEvent` rather than as keys.

Update every existing test that references `RenameField`/`Rename:` to
`StripField`/`Field:` with `Prefix: "rename"`. The existing rename tests must
keep passing unchanged in meaning, and the rendered text stays `[rename: …]`.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/termcmd/ -run 'TestStrip|TestPaste' -count=1`
Expected: FAIL to compile (`StripField`, `InsertPaste` undefined).

- [ ] **Step 3: Implement**
  - `strip.go`: rename the type to `StripField` with a `Prefix string` field and
    the model field `Rename *RenameField` → `Field *StripField`. Replace
    `renameChip(text)` with `fieldChip(prefix, text string) string { return "["
    + prefix + ": " + rowtext.Sanitize(text) + "]" }`, and use
    `m.Field.Prefix` at both call sites. Update the doc comments that say
    "rename" to "field (rename or url)".
  - `rename_input.go`: add `{"\x15", RenameDeleteToStart}` handling. `\x15`
    is a single byte, so it goes in the `< utf8.RuneSelf` branch before the
    `>= 0x20` test: `if input[0] == 0x15 { events = append(events,
    RenameEvent{Kind: RenameDeleteToStart}) }`. It isn't a
    `renameControlSequences` row, because those are ESC-led. Rename gains
    readline's Ctrl+U too, deliberately.
  - `rename.go`: add

    ```go
    // InsertPaste inserts pasted text at the caret. A field is one line, so
    // newlines and every control character are dropped rather than committing.
    func InsertPaste(e RenameEditor, text string) RenameEditor {
    	for _, r := range text {
    		if r < 0x20 || (r >= 0x7f && r < 0xa0) {
    			continue
    		}
    		e, _ = e.Apply(RenameEvent{Kind: RenameInsert, Rune: r})
    	}
    	return e
    }
    ```
  - `presentation.go`: rename `activeRename` → `activeField{tabID int; prefix
    string; editor RenameEditor}` and `m.rename` → `m.field`.
    - `beginRename` becomes `beginField(tabID int, prefix, initial string)`,
      where tabID -1 means a detached new-tab field. Alt+r calls it with the
      active tab's id, `"rename"` and `t.name`.
    - `refreshRename` becomes `refreshField`.
    - `finishRename` becomes `finishField(id int, prefix string, outcome
      RenameOutcome) error`, which dispatches on prefix:
      - `rename` on a **shell** tab: today's behavior (`t.name =
        outcome.Name`).
      - `rename` on a **browser** tab: the mux does **not** write `t.name` or
        `t.named`. It sends `EvRenamed{Name}` to the controller, whose
        `FxRelabel` writes `t.name`, `t.named` and `t.label` from `State`.
        `State` is the single authority for a browser tab's name and named
        flag. The strip, the record, and #293's reuse rule all read what the
        controller wrote, so they can't disagree (plan-gate minor). The
        controller exists from Task 1.8; until then, this branch is unreachable
        because no browser tab can be created.
      - `url`: Task 1.8 fills this in, including the remote-URL confirmation.
    - `terminalTab` gains:

      ```go
      kind    tabKind // tabShell or tabBrowser
      named   bool    // browser: operator renamed it (#293 reuse rule reads it)
      label   string  // browser: strip text from the controller; "" = name
      browser *browserController
      ```

      Add `type tabKind int; const (tabShell tabKind = iota; tabBrowser)`.
    - `stripModelLocked`: `TabChip{Name: t.label}` when `t.label != ""`, else
      `t.name`.
  - `run.go`: rename `renameSession` → `fieldSession{tabID int; prefix string;
    editor RenameEditor; decoder RenameDecoderState}`.
    - In the field branch, before the "not a key → continue" check, add: `if
      paste, ok := event.Event.(uv.PasteEvent); ok { field.editor =
      InsertPaste(field.editor, paste.Content); mux.refreshField(field.tabID,
      field.editor); continue }`.
    - Update the `ptyWriter` interface methods to the new names and update
      `fakeMux` in `run_test.go` to match.

- [ ] **Step 4: Run the whole package**

Run: `go test ./cmd/internal/termcmd/ -count=1`
Expected: PASS (the pty tests unsandboxed).

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/termcmd/
git commit -m "#292 M1: strip field generalizes rename (url prefix, detached), paste inserts, tab labels"
```

### Task 1.6: Chords, bindings and help for Alt+B / Shift+Alt+B

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
  - chords `ChordAltB`, `ChordAltShiftB` before `chordMax`;
  - actions `ActionNewBrowserTab`, `ActionTerminalNewBrowserTab`;
  - `globalBindings` row;
  - `roleBindings` row;
  - `Decide` cases;
  - `chordSequences` rows;
  - `ChordName` cases;
  - `TabChordFor` case.
- Modify: `zellij/config.kdl` (`bind "Alt B" { WriteChars "\u{1b}[66;4u"; }`, next to `Alt T` at line 97)
- Modify: `cmd/internal/keyhelp/catalog.go`
  - include rows (role `Alt+b (terminal)`, global `<M-B>`);
  - `roleChordKey` case.
- Modify: `cmd/internal/layoutcmd/layoutcmd.go` `RunSwitchTerminalTab`: accept `new-browser`
- Modify: `nvim/init.lua` (`_G.PairTermNewBrowserTab = function() pair_switch_terminal_tab('new-browser') end` beside `PairTermNewTab`, ~line 3437)
- Regenerate: `nvim/workbench_actions.lua` via `go run ./cmd/internal/workbenchshortcut/generatecmd`
- Modify: `cmd/internal/termcmd/run.go` `namedChord` (`alt+b`, `alt+shift+b`), `runDecision` (treat `ActionNewBrowserTab` like `ActionNewTab`, and `ActionTerminalNewBrowserTab` like `ActionTerminalNewTab`, through `TabChordFor`)
- Tests: existing guards (listed in Step 2) plus new ones below.

- [ ] **Step 1: Write the failing tests** (in `workbenchshortcut/shortcut_test.go`)

```go
func TestBrowserChordsDecode(t *testing.T) {
	for seq, want := range map[string]Chord{
		"\x1bb": ChordAltB, "\x1b[98;3u": ChordAltB,
		"\x1b[66;4u": ChordAltShiftB,
	} {
		if got, ok := DecodeChord([]byte(seq)); !ok || got != want {
			t.Errorf("%q -> %v %v, want %v", seq, got, ok, want)
		}
	}
	// No legacy \x1bB: a global fires inside the escape deadline (#243).
	if got, ok := DecodeChord([]byte("\x1bB")); ok && got == ChordAltShiftB {
		t.Error("legacy \\x1bB must not be Shift+Alt+B")
	}
}

func TestShiftAltBIsAnAgentReservedInPaneGlobalDeliveredToTheRightPane(t *testing.T) {
	b, ok := globalDraftAction(ChordAltShiftB)
	if !ok || !b.HandledInPane || !b.AgentReserved || b.NvimKey != "<M-B>" {
		t.Fatalf("%+v %v", b, ok)
	}
	chord, ok := TabChordFor(ActionTerminalNewBrowserTab)
	if !ok || chord != ChordAltShiftB || !IsGlobalChord(chord) {
		t.Fatalf("%v %v", chord, ok)
	}
}

func TestAltBIsRightPaneOnly(t *testing.T) {
	if d := Decide(ShortcutInput{Role: PaneRoleRightTerminal, Chord: ChordAltB}); d.Action != ActionNewBrowserTab {
		t.Fatalf("%+v", d)
	}
	for _, role := range []PaneRole{PaneRoleLeftAgent, PaneRoleLeftDraft} {
		if d := Decide(ShortcutInput{Role: role, Chord: ChordAltB}); d.Disposition != DispositionPass {
			t.Fatalf("role %v: %+v", role, d)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure, then implement until these pass, together with every existing guard**
  - `workbenchshortcut`:
    - `TestRoleBindingsCoverTerminalSwitch`
    - `TestRenderedLuaGlobalMapsMatchCommittedFile` (regenerate the Lua)
    - `TestAgentReservationMetadataIsExact`
    - `TestAgentReservationPolicyMatrix`
    - `TestTabChordForDeliversGlobalChords`
  - `keyhelp`:
    - `TestEveryGlobalChordIsClassified`
    - `TestZellijShortcutConfigHasNoImplicitConsumers`
    - `TestEveryNvimKeymapIsClassified`
  - `termcmd`: `TestEveryHandledTerminalChordIsDocumented`
  - `couchcmd`: the `couch --help` reserved-chord list (`run.go`,
    `shortcut_help_test.go`), and the live conformance table
    (`shortcut_conformance_live_test.go`), which is updated but runs only live
  - shell: `tests/term-pane-shortcuts-test.sh`

  Treat this list as a starting point, not the contract. The contract is that
  every existing guard passes; a guard not listed here that fails is part of
  this task.

Run: `go test ./cmd/internal/workbenchshortcut/ ./cmd/internal/keyhelp/ ./cmd/internal/termcmd/ ./cmd/internal/couchcmd/ ./cmd/internal/layoutcmd/ -count=1`
Expected: first FAIL (new tests), then PASS after implementation.

Implementation specifics:
- **`globalBindings`:**

  ```go
  {Chord: ChordAltShiftB, Action: ActionTerminalNewBrowserTab, LuaFunction: "PairTermNewBrowserTab", NvimKey: "<M-B>", FocusDraft: false, HandledInPane: true, AgentReserved: true,
  	Help: "new browser tab in the right pane (type a URL, Enter), from any pane"},
  ```
- **`roleBindings`:** `{Chord: ChordAltB, Role: PaneRoleRightTerminal, Help: "new browser tab: type a URL, Enter opens it; click the active browser tab to change its URL"}`.
- **`Decide`:**
  - `PaneRoleRightTerminal` gets `case ChordAltB: return handle(ActionNewBrowserTab)`.
  - The left roles need no case (default Pass).
- **`chordSequences`:** `{"\x1bb", ChordAltB}, {"\x1b[98;3u", ChordAltB}` and `{"\x1b[66;4u", ChordAltShiftB}`.
- **`ChordName`:** `"Alt+b"` / `"Alt+Shift+B"`.
- **`TabChordFor`:** `case ActionTerminalNewBrowserTab: return ChordAltShiftB, true`.
- **keyhelp catalog:**
  - `{Key: "Alt+b (terminal)", Display: "Alt+b", Group: groupTerminal, Order: 11, Context: ContextTerminal, Source: SourceRole}`
  - `{Key: "<M-B>", Display: "Shift+Alt+b", Group: groupTerminal, Order: 12, Context: ContextGlobal, Source: SourceGlobal}`
  - `roleChordKey`: `case workbenchshortcut.ChordAltB: return "Alt+b (terminal)"`.
- **`RunSwitchTerminalTab`:** `case "new-browser": action = workbenchshortcut.ActionTerminalNewBrowserTab`, and the usage string gains `|new-browser`.
- **The scrollback viewer's buffer-local `<M-B>`** (`nvim/scrollback.lua:487`)
  stays. If `TestEveryNvimKeymapIsClassified` or a collision guard flags it,
  record it as a deliberate per-buffer override in that guard's allowance list
  with a comment citing #292 Spec ("the viewer's buffer-local Shift+Alt+B
  outranks the global there").

- [ ] **Step 3: Commit**

```bash
git add cmd/internal/workbenchshortcut/ cmd/internal/keyhelp/ cmd/internal/layoutcmd/ cmd/internal/termcmd/run.go cmd/internal/couchcmd/ zellij/config.kdl nvim/init.lua nvim/workbench_actions.lua tests/term-pane-shortcuts-test.sh
git commit -m "#292 M1: Alt+b (right pane) and Shift+Alt+b (global) browser-tab chords, Alt+h rows"
```

### Task 1.7: Browser tabs own the tab keys (no #227 passthrough)

**Files:**
- Modify: `cmd/internal/termcmd/presentation.go` `activeChildOwnsScreen` (line ~320)
- Test: `cmd/internal/termcmd/passthrough_test.go`

- [ ] **Step 1: Failing test.** In the passthrough test style, admit a fake
  child tab with `kind: tabBrowser` whose endpoint is on the alt screen (feed
  `"\x1b[?1049h"`), then assert `activeChildOwnsScreen()` is false. A shell
  tab fed the same bytes must stay true (regression guard for #227).

```go
func TestBrowserTabOnAltScreenDoesNotTakeRoleChords(t *testing.T) {
	m := newPresentationFixture(t) // existing helper; use whatever builds a mux over ttyio.NewFake()
	child := addPresentationTab(t, m, 1, "\x1b[?1049h")
	if !m.activeChildOwnsScreen() {
		t.Fatal("precondition: a shell tab on the alt screen owns the screen (#227)")
	}
	m.mu.Lock()
	m.tabs[0].kind = tabBrowser
	m.mu.Unlock()
	if m.activeChildOwnsScreen() {
		t.Fatal("a browser tab must not receive Alt+t/w/r/b/←/→ (#292)")
	}
	_ = child
}
```

- [ ] **Step 2: Implement.** Make `activeChildOwnsScreen` return `t != nil &&
  t.child != nil && t.kind == tabShell && t.child.Endpoint().Modes().AltScreen`.
  Extend its interface comment: "A browser tab is pair-owned chrome around
  Chromium, which has no use for the tab keys (#292)."

- [ ] **Step 3: Run** `go test ./cmd/internal/termcmd/ -run 'Passthrough|BrowserTab' -count=1`. Expected: PASS.

- [ ] **Step 4: Commit** `#292 M1: browser tabs keep pair term's tab keys`.

### Task 1.8: `newBrowserTab`, the controller shell (Close only), and lifecycle tests

**Files:**
- Create: `cmd/internal/termcmd/browser.go` (controller + `newBrowserTab` + deps)
- Create: `cmd/internal/browsertab/machine.go`. The M1 subset:
  - `Launching` and `Closed`;
  - `Close` → `[Unpublish, Hangup, KillGroup, RemoveProfile]`;
  - `Renamed` → `[Relabel]`, so renaming a browser tab works in M1 through
    the single authority.

  M2 fills in the rest of the table.
- Modify: `cmd/internal/termcmd/presentation.go`
  - `removeTab`: close the controller before `child.Close()`, outside the
    lock;
  - `closeAll`: close controllers in parallel before children;
  - `finishField` `url` branch;
  - transient notice: `flash string`, `flashUntil time.Time`, shown when no
    permanent `notice`, cleared by an `time.AfterFunc` repaint.
- Modify: `cmd/internal/termcmd/run.go`
  - the pump special-cases `ChordAltB`/`ChordAltShiftB` like `ChordAltR`,
    starting a `fieldSession{tabID: -1, prefix: "url"}`;
  - Shift+Alt+B also runs `rt.RunZellijAction("focus-pane-id",
    rt.CurrentPaneID())` when the id is non-empty;
  - `runShellOnHost` sets `mux.browser = osBrowserDeps()`.
- Create: `cmd/internal/termcmd/browser_test.go`
- Modify: `cmd/internal/termcmd/*_test.go` TestMain (add one if absent) with the fakecarbonyl hook.

`browserDeps` (injected; production values in `osBrowserDeps`):

```go
type browserDeps struct {
	binary   func() (string, error)  // browsertab.ResolveBinary
	env      []string                // extra child env (tests: PAIR_FAKE_CARBONYL=1)
	fps      int                     // browsertab.DefaultFPS
	profiles browsertab.ProfileStore // Root from DefaultProfileRoot
	probe    browsertab.OwnerProbe   // OSOwnerProbe{}
	owner    browsertab.Identity     // CurrentIdentity(os.Getpid())
	version  func(bin string) (browsertab.Version, bool)
}
```

`newBrowserTab(url string) error`, in order:
1. Resolve the binary. On error, flash `carbonyl not found (install it, or set PAIR_CARBONYL)` and return.
2. On the first launch per mux, check the version in a goroutine:
   `VersionNotice(v)` non-empty → flash it for 10 s.
3. `profiles.Sweep(probe)`: errors go to the diagnostic log only.
4. `profiles.Create(owner, id)`.
5. Pick the name with `DefaultName(taken)` over this mux's tab names (M3 adds
   the tag's record names).
6. `ptychild.Start(Options{Argv: browsertab.Argv(bin, profile, fps, url), Size: childSizeLocked(), Env: shellEnv + deps.env, KillGroup: true, Sink: …})`,
   with the same ready-gate as `newTab`. `KillGroup: true` is what makes
   every ptychild reap path kill Chromium's helpers first (PQ-7).
7. Build `terminalTab{kind: tabBrowser, name, label: browsertab.Label(name, url), browser: ctrl}`,
   where `ctrl = newBrowserController(…)` holds `profile`, `child`, and
   `State{Launching, Name: name}`.
8. `admitTab`, then start the exit watcher (`<-child.Exited(); m.removeTab(id)`,
   as `newTab` does).

On any failure after step 4, remove the profile dir, so a failed launch leaves
nothing behind.

Controller (M1):

```go
type browserController struct {
	mailbox browserMailbox
	cancel  context.CancelFunc
	done    chan struct{}
	state   browsertab.State // loop goroutine only
	fx      browserEffects   // child, profiles, profile dir, mux callbacks
}

// browserMailbox never drops an event and never blocks the sender (the mux
// sends while holding its lock). EvTarget and EvTargetGone for the same target
// COALESCE: each is a full snapshot, so a newer one replaces the queued one in
// place. Every other kind (Renamed, Navigate, PortFound, Connected, ...) is
// appended. The queue's size is bounded by distinct targets plus operator and
// lifecycle actions, and none of those is page-driven, so a page can't grow it.
type browserMailbox struct {
	mu     sync.Mutex
	queue  []browsertab.Event
	notify chan struct{} // cap 1
}

func (b *browserMailbox) put(ev browsertab.Event) {
	b.mu.Lock()
	if ev.Kind == browsertab.EvTarget || ev.Kind == browsertab.EvTargetGone {
		for i := range b.queue {
			q := b.queue[i]
			if (q.Kind == browsertab.EvTarget || q.Kind == browsertab.EvTargetGone) && q.Target.ID == ev.Target.ID {
				b.queue[i] = ev
				b.mu.Unlock()
				return
			}
		}
	}
	b.queue = append(b.queue, ev)
	b.mu.Unlock()
	select {
	case b.notify <- struct{}{}:
	default:
	}
}

func (b *browserMailbox) take() []browsertab.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	q := b.queue
	b.queue = nil
	return q
}

func (c *browserController) Send(ev browsertab.Event) { c.mailbox.put(ev) }

// Close runs the Close transition's effects and joins the loop, bounded.
func (c *browserController) Close() {
	c.cancel()
	select {
	case <-c.done:
	case <-time.After(3 * time.Second):
	}
}
```

The loop:

```go
for {
	select {
	case <-ctx.Done():
		// Queued events are discarded: Close supersedes them, and the Closed
		// rows of the table ignore everything that could still arrive.
		s, fx := browsertab.Step(c.state, browsertab.Event{Kind: browsertab.EvClose})
		c.state = s
		c.run(fx)
		close(c.done)
		return
	case <-c.mailbox.notify:
		for _, ev := range c.mailbox.take() {
			s, fx := browsertab.Step(c.state, ev)
			c.state = s
			c.run(fx)
		}
	}
}
```

Add `TestMailboxCoalescesTargetsAndKeepsEverythingElse` (pure, in
`browser_test.go`). It puts Renamed(a), Target(t1,u1), Navigate(x),
Target(t1,u2), Renamed(b) and checks that `take()` returns exactly
Renamed(a), Target(t1,u2), Navigate(x), Renamed(b), in that order, with nothing
dropped.

Effects in M1:
- `Relabel` → `mux.setBrowserLabel(tabID, state)` takes the mux lock and
  writes `t.name = state.Name()`, `t.named = state.Named()` and `t.label =
  state.Label()`, then `paintStripLocked` + `renamePane`. It's the only writer
  of those three fields on a browser tab.
- `KillGroup` → `child.KillGroup()`, then poll `syscall.Kill(-pid, 0)` until
  ESRCH, bounded at 2 s.
- `RemoveProfile` → `profiles.Remove(dir)`.
- `Unpublish` and `Hangup` are no-ops until M2/M3.

- [ ] **Step 1: Write the failing tests** (`browser_test.go`; the pty tests skip when sandboxed)

```go
func TestMain(m *testing.M) {
	if fakecarbonyl.Invoked() {
		os.Exit(fakecarbonyl.Main(os.Args[1:]))
	}
	os.Exit(m.Run())
}

// testBrowserDeps launches this test binary as fakecarbonyl, with profiles
// under a temp root.
func testBrowserDeps(t *testing.T, extraEnv ...string) browserDeps {
	root := filepath.Join(t.TempDir(), "browser")
	owner, err := browsertab.CurrentIdentity(os.Getpid())
	if err != nil {
		t.Skipf("no strict identity on this platform: %v", err)
	}
	return browserDeps{
		binary:   func() (string, error) { return os.Args[0], nil },
		env:      append([]string{fakecarbonyl.EnvInvoke + "=1"}, extraEnv...),
		fps:      browsertab.DefaultFPS,
		profiles: browsertab.ProfileStore{Root: root},
		probe:    browsertab.OSOwnerProbe{},
		owner:    owner,
		version:  func(string) (browsertab.Version, bool) { return browsertab.Version{0, 0, 3}, true },
	}
}

func TestAltWOnABrowserTabKillsItsGroupAndRemovesTheProfile(t *testing.T) {
	m, deps := newBrowserMux(t) // mux over hostty.FakeHost with one shell tab (/bin/sh) + deps
	if err := m.newBrowserTab("http://localhost:1111/"); err != nil {
		t.Fatal(err)
	}
	tab := activeTab(m)
	profile := tab.browser.profileDir()
	helper := waitHelperPID(t, profile)
	leader := tab.child.PID()
	m.closeActive()
	assertGone(t, leader, helper)
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("profile %s survived close: %v", profile, err)
	}
	_ = deps
}

func TestPairTermExitClosesEveryBrowser(t *testing.T) {
	m, _ := newBrowserMux(t)
	var pids []int
	var profiles []string
	for i := 0; i < 2; i++ {
		if err := m.newBrowserTab("http://localhost:1111/"); err != nil {
			t.Fatal(err)
		}
		tab := activeTab(m)
		pids = append(pids, tab.child.PID(), waitHelperPID(t, tab.browser.profileDir()))
		profiles = append(profiles, tab.browser.profileDir())
	}
	m.closeAll()
	assertGone(t, pids...)
	for _, p := range profiles {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("profile %s survived closeAll", p)
		}
	}
}

func TestCarbonylExitingOnItsOwnRemovesItsTab(t *testing.T) {
	m, _ := newBrowserMux(t)
	if err := m.newBrowserTab("http://localhost:1111/"); err != nil {
		t.Fatal(err)
	}
	tab := activeTab(m)
	_ = syscall.Kill(tab.child.PID(), syscall.SIGTERM)
	waitFor(t, func() bool { return tabCount(m) == 1 })
}

func TestFailedLaunchLeavesNoProfile(t *testing.T) {
	m, deps := newBrowserMux(t)
	m.browser.binary = func() (string, error) { return "/nonexistent/carbonyl", nil }
	if err := m.newBrowserTab("http://x.test/"); err == nil {
		t.Fatal("launch of a missing binary succeeded")
	}
	entries, _ := os.ReadDir(deps.profiles.Root)
	if len(entries) != 0 {
		t.Fatalf("profile residue: %v", entries)
	}
}
```

Plus two pump-level tests with `fakeMux`, in the existing `run_test.go` style:
- Feeding `"\x1bb"` then `"localhost:1111\r"` calls `beginField(-1, "url",
  "")` and then `finishField(-1, "url", Commit "localhost:1111")`.
- Feeding `"\x1b[66;4u"` also records a `focus-pane-id <CurrentPaneID>` zellij
  action on the fake runtime.

The SIGKILL crash test lives in its own process, because `pair term` itself
must die:

```go
// TestPairTermSIGKILLTakesCarbonylAlong runs this test binary as a real
// `pair term` (runShellOnHost over its own pty) with PAIR_TERMINAL_BROWSER_CHILD=1,
// opens a browser tab by typing Alt+b, a URL, Enter, reads the fake's profile
// dir from the child's stderr line "browser-profile <dir>", SIGKILLs the
// pair term, and asserts every Carbonyl pid is gone within 2 s (the kernel's
// pty SIGHUP, #292 Log). The profile dir SURVIVES (no cleanup ran); a second
// newBrowserTab in this process, whose sweep proves the dead owner, removes it.
func TestPairTermSIGKILLTakesCarbonylAlong(t *testing.T) { /* per the comment */ }
```

For this test the child mode needs one test-only hook: `browserDeps.onLaunch
func(profile string)`, which the child sets to print `browser-profile <dir>` on
stderr. The soak child (`soak_test.go:25`) is the template for re-execing the
test binary under a pty.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/termcmd/ -run 'Browser|PairTermSIGKILL|CarbonylExiting|FailedLaunch' -count=1`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement** `machine.go` (M1 subset) with its own unit test:

```go
func TestCloseFromLaunchingKillsAndCleansInOrder(t *testing.T) {
	s := NewState("web", "http://x/")
	s2, fx := Step(s, Event{Kind: EvClose})
	if s2.Phase() != PhaseClosed {
		t.Fatal(s2.Phase())
	}
	want := []EffectKind{FxUnpublish, FxHangup, FxKillGroup, FxRemoveProfile}
	if len(fx) != len(want) {
		t.Fatalf("%v", fx)
	}
	for i := range want {
		if fx[i].Kind != want[i] {
			t.Fatalf("%v", fx)
		}
	}
	if _, fx := Step(s2, Event{Kind: EvClose}); len(fx) != 0 {
		t.Fatal("Close is idempotent")
	}
}
```

Then implement `browser.go` and the presentation/run changes described above.
In `removeTab`, capture `removed.browser` under the lock and call
`removed.browser.Close()` after `m.mu.Unlock()` and **before**
`removed.child.Close()`. In `closeAll`, after `presenter.Release`, close all
controllers concurrently (a `sync.WaitGroup`), then close all children.

- [ ] **Step 4: Run the package unsandboxed; then the full suite**

Run: `go test ./cmd/internal/termcmd/ ./cmd/internal/browsertab/... ./cmd/internal/ptychild/ -count=1`
Expected: PASS.
Then: `TMPDIR=<scratchpad> make test > $TMPDIR/make-test.log 2>&1; tail -30 $TMPDIR/make-test.log`
Expected: all green (memory `full_make_test_before_sdlc_close`).

- [ ] **Step 5: Commit** `#292 M1: browser tab launch, lifecycle controller, group kill and profile sweep`.

### Task 1.9: M1 verification and close

- [ ] Run `go vet ./...` and `TMPDIR=<scratchpad> make test` (log to a file, don't pipe to `head`).
- [ ] Fuzz 15 s each: `go test -run=^$ -fuzz=^FuzzNormalizeURL$ -fuzztime=15s ./cmd/internal/browsertab/`, and the same for `FuzzParseActivePort` and `FuzzParseOwnedName`.
- [ ] Build `bin/pair` (`make build`). Ask the operator to smoke-test in a live
  session:
  - Alt+b in the right pane, type `localhost:<some dev server>`, Enter → a
    Carbonyl tab opens.
  - Alt+w closes it, and `pgrep -fl carbonyl` is empty.
  - Shift+Alt+b from the draft focuses the right pane with the URL field open.
  - Alt+h lists both rows.

  Record their answer in `## Log`.
- [ ] Update `atlas/architecture.md` in the `pair term` section (~l.594):
  "`termcmd` keeps numbered tabs …" gains browser tabs, their owner-bound
  lifecycle, and the profile root outside the data root.
- [ ] `sdlc milestone-close --issue 292 --milestone M1` and handle its review.

---

## Chunk 2: M2 — DevTools: label, navigation, conformance, and the cap

### Task 2.1: Add `coder/websocket`; the CDP client

**Files:**
- Modify: `go.mod`, `go.sum` (`go get github.com/coder/websocket@v1.8.15`; needs `proxy.golang.org` + `sum.golang.org` reachable)
- Create: `cmd/internal/browsertab/cdp.go`
- Test: `cmd/internal/browsertab/cdp_test.go` (against an in-process `httptest.Server` running the fakecarbonyl DevTools handler, Task 2.2)

CDP message shapes (Chrome 111):

```go
type cdpRequest struct {
	ID        int64  `json:"id"`
	Method    string `json:"method"`
	Params    any    `json:"params,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}
type cdpMessage struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *struct{ Code int; Message string } `json:"error,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}
type TargetInfo struct {
	ID    string `json:"targetId"`
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
}
type TargetEventKind int
const (TargetCreated TargetEventKind = iota; TargetChanged; TargetDestroyed)
type TargetEvent struct { Kind TargetEventKind; Target TargetInfo }
```

The client API is given under Integration points. Its read loop dispatches:
- responses by `id` to a `map[int64]chan cdpMessage`;
- `Target.targetCreated` / `targetInfoChanged` (`params.targetInfo`) and
  `Target.targetDestroyed` (`params.targetId`) to the `events` channel
  (buffered 64, oldest dropped when full). `Changed` is a full snapshot, so
  losing an intermediate one is harmless.

Other methods are ignored. Decoding is a pure function the read loop calls,
`decodeCDPMessage(b []byte) (cdpMessage, error)` plus `targetEventOf(msg)
(TargetEvent, bool)`, so `FuzzDecodeCDPMessage` exercises exactly what the
loop runs. `targetEventOf` sanitizes URL and Title with
`rowtext.SanitizeAndFit(…, 2048)` at construction, so no unsanitized page text
exists past the decode boundary. `call(ctx, method, params, session)` writes under a
mutex and waits for its id or `ctx.Done()`. On read-loop exit every pending
call fails with `ErrClosed` and `Done()` closes.

- [ ] **Step 1:** Write `cdp_test.go`:
  - `TestDiscoverYieldsTheExistingPage`: dial the fake → `DiscoverTargets` → the
    first event is `Created` with `type=="page"` and the launch URL.
  - `TestNavigateChangesTargetInfo`: `Navigate(id, "http://b.test/")` → a
    `Changed` event with URL `http://b.test/` and title `Fake http://b.test/`.
  - `TestCloseFailsPendingCalls`: server closes mid-call → the call returns an
    error and `Done()` closes.
  - `TestCallRespectsContext`: the server never answers → the call returns
    `context.DeadlineExceeded` within its deadline.
  - `FuzzDecodeCDPMessage`: seeds and properties from the strategy table.
- [ ] **Step 2:** Run it, confirm it fails, implement, then run and confirm it passes.
- [ ] **Step 3:** Commit `#292 M2: CDP client over coder/websocket`.

### Task 2.2: fakecarbonyl DevTools server

**Files:** Modify `cmd/internal/browsertab/fakecarbonyl/fake.go`; add `server.go`.

- `NewHandler(b *Browser) http.Handler`, exported so `cdp_test.go` can mount it
  in-process:
  - `GET /json/version` → `{"Browser":"Fake/111 (Carbonyl)","webSocketDebuggerUrl":"ws://<host>/devtools/browser/FAKE-BROWSER"}`
  - `GET /json/list` → `[{"id":targetID,"type":"page","url":url,"title":title,"webSocketDebuggerUrl":…}]`
  - `GET /devtools/browser/FAKE-BROWSER` → `websocket.Accept`. Then per message:
    - `Target.setDiscoverTargets` → reply `{}`, then send `Target.targetCreated{targetInfo}`.
    - `Target.attachToTarget{targetId, flatten:true}` → reply `{"sessionId":"S-<n>"}`.
    - `Page.navigate{url}` with a known `sessionId` → set url and title, redraw
      row 0 (when running as a process), reply `{"frameId":"F"}`, broadcast
      `Target.targetInfoChanged` to every connection that enabled discovery.
    - An unknown method → an `{"error":{"code":-32601}}` reply.
- **Process mode:** `Main` listens on `127.0.0.1:0` before writing the port
  file, so the port in the file is real. The M1 `port == 0 → 1` placeholder is
  deleted.
- **State is mutex-guarded:** `Browser{mu; url, title, targetID; sessions; conns}`.
- The Task 1.2 process test is extended with a `curl`-free check:
  `http.Get(ep.HTTP + "/json/version")` contains `Carbonyl`.
- Commit `#292 M2: fakecarbonyl DevTools server (discover, attach, navigate)`.

### Task 2.3: The full state machine

**Files:** `cmd/internal/browsertab/machine.go`, `machine_test.go`.

Implement the whole ARCH-ORDER table above. `State` keeps unexported fields,
with accessors `Phase()`, `Name()`, `Named()`, `Target()`, `Label()` (=
`Label(name, target.URL)`) and `Endpoint()`. `NewState(name, launchURL string)`
seeds `Target.URL = launchURL`, so the label shows the site before DevTools
connects. `Event` fields: `Kind`, `Endpoint`, `Target TargetInfo`, `URL`,
`Name`, `Err`. `Effect`: `Kind`, `URL`, `TargetID`, `Endpoint`, `Text`.

Tests (pure, table-driven):
- **`TestTransitionTable`:** one case per row of the ARCH-ORDER table,
  asserting the next phase and the exact effect kinds in order.
- **`TestLateEventsAfterCloseNeverPublish`:** for every `EventKind`, `Step(closed, ev)` never yields `FxPublish` or `FxRelabel`; `EvConnected` yields exactly `[FxHangup]`.
- **`TestPendingNavigateIsSentOnceWhenTheTargetArrives`:** Navigate(a),
  Navigate(b) while Launching → … → Target(page) gives one `FxNavigate("b")`,
  and a second Target event gives no Navigate.
- **`TestOnlyTheFirstPageTargetIsFollowed`:** a second page's Target events
  don't relabel.
- **`TestRenamedMarksNamed`:** Renamed("local-test") → `Named()` true and the
  label uses the new name.
- **`TestEventSequenceInvariants`:** a seeded random walk of 2,000 steps over
  all event kinds (`math/rand` with a fixed seed, logged on failure) asserts
  three things. First, `FxPublish` only ever appears while Phase == Ready and
  the target is known. Second, after the first `EvClose` no effect other than
  `FxHangup` appears. Third, `FxKillGroup` appears exactly once, at the first
  Close. These are independently stated invariants, not a restatement of the
  table.

Commit `#292 M2: browser tab state machine`.

### Task 2.4: Controller DevTools phases and the label

**Files:** `cmd/internal/termcmd/browser.go`, `browser_test.go`.

- **Port discovery:** on start, the controller spawns the port watcher, which
  polls `<profile>/DevToolsActivePort` every 50 ms. On success it sends
  `EvPortFound{ParseActivePort(content)}` (a parse error is retried until 10 s).
  At 10 s it sends `EvPortTimeout`. The watcher is cancelled by the controller
  context.
- **`FxDial`:** a goroutine runs `deps.dial(ctx, ep.WS)` →
  `DiscoverTargets` → sends `EvConnected`, then forwards every client event as
  `EvTarget` / `EvTargetGone`. When `client.Done()` closes it sends
  `EvDisconnected`. On a dial error it sends `EvConnectFailed{err}`.
- **`FxHangup`:** `client.Close()`.
- **`FxRelabel`:** unchanged from M1 (`setBrowserLabel` writes name, named and
  label from `State`). M2 only adds the phases that emit it.
- **`FxNotice`:** `mux.flash(text, 8*time.Second)`.
- **`FxPublish` / `FxUnpublish`:** still no-ops (M3).

Integration tests on the real fake (unsandboxed):
- **`TestBrowserTabLabelFollowsNavigation`:** open a tab on
  `http://localhost:1111/` → the strip shows `web · localhost:1111`. Then
  navigate *from a second CDP client* (the agent's position): dial
  `ParseActivePort` of the tab's profile, `Navigate(…, "http://other.test:2/")`.
  The strip becomes `web · other.test:2` within 2 s.
- **`TestNoPortFileDegradesWithANotice`:** with `PAIR_FAKE_CARBONYL_NO_PORT=1`,
  a timeout the test injects as 200 ms through `browserDeps.portTimeout` flashes
  `DevTools unavailable`. The tab stays open, and Alt+w still cleans up.
- **`TestCloseDuringSlowPortLeavesNoRecordOrSocket`:** with
  `PAIR_FAKE_CARBONYL_PORT_DELAY_MS=500`, close the tab at 100 ms and wait
  1 s. Assert the fake's connection count stays 0; the fake exposes it at
  `/test/conns`, a test-only route. Also assert the processes are gone.

Also `FuzzStripLabel` in `cmd/internal/termcmd/strip_test.go`, from the
strategy table: `RenderStrip` over a model whose chip is `browsertab.Label(name,
url)` with fuzzed `url`.

Commit `#292 M2: controller dials DevTools; strip label follows navigation`.

### Task 2.5: Click the active browser tab → URL field → navigate

**Files:** `cmd/internal/termcmd/presentation.go`, `run.go`, tests.

- **`mux.stripClick(ev uv.MouseClickEvent) (tabID int, initial string, ok bool)`:**
  under the lock, it requires all of:
  - `ev.Button == uv.MouseLeft`;
  - `m.rows >= 2 && ev.Y == int(m.rows)-1` (the strip row; `Y` is 0-based);
  - the active tab is `tabBrowser`;
  - `ev.X` falls inside the active tab's `TabSpan` from `RenderStrip(int(m.cols), m.stripModelLocked())`.

  It returns the active tab id and `t.browser` state's current URL. Carbonyl
  turns on `?1003h`, so zellij forwards mouse while a browser tab is active.
  With a shell tab active there's no mouse reporting, and nothing here runs.
- **In the pump's mouse branch,** before `mux.writeEvents`: `if click, ok :=
  event.Event.(uv.MouseClickEvent); ok { if id, initial, ok :=
  mux.stripClick(click); ok { field = beginField(id, "url", initial); continue }
  }`. Swallow the matching `MouseReleaseEvent` on the strip row too. Otherwise
  Carbonyl gets a release with no press, which `presenter` drops anyway
  (`ParentPressMouse`); assert that in the test rather than assume it.
- **`finishField` `url` branch:**
  - `NormalizeURL(text)`; on error, flash `not a URL: <err>`.
  - **Remote URLs are confirmed once.** When `!browsertab.IsLocalURL(url)` and
    this field session hasn't confirmed yet, don't act: flash `Chromium 111,
    unpatched since 2023 — Enter again to open <host>, or Esc` for 8 s, mark
    the session confirmed, and re-open the field with the same text. A second
    Enter proceeds. Esc cancels. Local URLs never see this.
    - It's one keystroke, it names the risk where the decision is made, and it
      keeps remote pages possible.
    - The confirmed flag lives on the pump's `fieldSession`, so it's per
      session: a later edit of the same tab confirms again.
  - With `tabID == -1` → `newBrowserTab(url)`.
  - Otherwise → `t.browser.Send(Event{Kind: EvNavigate, URL: url})`.
  - Tests: `TestLocalURLOpensWithoutConfirmation`,
    `TestRemoteURLNeedsASecondEnter` (first Enter creates no tab and flashes;
    second opens), `TestEscAfterTheWarningCreatesNothing`.
- **Tests:**
  - A pump test with a fake mux: a strip-row click on the active browser chip
    begins a url field prefilled with the current URL; Ctrl+U (`\x15`, mapped
    in Task 1.5) clears it; typing and `\r` commits `EvNavigate`.
  - `TestBrowserTabTurnsOnMouseReporting`, `TestStripClickHitsTheActiveChip`
    (a real SGR report `\x1b[<0;<col>;<rows>M` decoded by the pump), and the
    release-not-forwarded assertion. These are the PQ-1 inventory pins.
  - A click one column past the chip's span is forwarded to the child
    unchanged.
  - An integration test on the fake: click, then `other.test:2`, then Enter →
    the label becomes `web · other.test:2`.

Commit `#292 M2: clicking the active browser tab opens the URL field; Enter navigates`.

### Task 2.6: Live conformance probe

**Files:** Create `cmd/probes/carbonylconformance/main.go`; Modify `Makefile`
(`probe-carbonyl:` target, `go run ./cmd/probes/carbonylconformance`), and
`atlas/index.md` (the `cmd/probes/` note lists it).

Behavior:
- **Discovery:** resolve with `browsertab.ResolveBinary()`. If it's missing,
  print `SKIP: carbonyl not installed` and exit 0.
- **Launch:** `browsertab.Argv` on `about:blank` into a temp profile, under a
  real pty (`ptychild.Start`, size 30×100), serving a two-page site from an
  in-process `httptest.Server`.
- **Asserts** (each prints `ok <name>` or `FAIL <name>: <detail>`; exit 1 on
  any FAIL):
  1. `ProbeVersion` ≥ `MinVersion`.
  2. `DevToolsActivePort` appears within 10 s, and `ParseActivePort` accepts it.
  3. `/json/version` `Browser` contains `Carbonyl`.
  4. `Dial` + `DiscoverTargets` yields one `page` target.
  5. `Navigate(page2)` yields `targetInfoChanged` with page2's URL and title
     within 5 s.
  6. Idle CPU: the group's summed `ps -o time=` rises by less than 0.25 s over
     5 s.
  7. Row 0 of the child's screen (`Endpoint` text) starts with `[❮][❯][↻][`.
  8. `KillGroup` leaves no member of the group within 2 s.
  9. A second launch, whose pty-owning helper process (the probe re-execs
     itself as the owner) is SIGKILLed, leaves no Carbonyl processes within 2 s.

  Tests 8 and 9 are the real-binary counterparts of the fake-backed lifecycle
  tests.
- **Where it can run:** a `cmd/probes` probe can import `cmd/internal/…`
  (atlas/index.md). It runs unsandboxed only (pty).

Run it against Carbonyl 0.0.3 (the scratch release zip, `PAIR_CARBONYL=<path>`)
and against the operator's 0.0.2 install. Expected: all ok on 0.0.3; on 0.0.2,
FAIL on the version and idle-CPU checks, which confirms that the probe detects
the bug. Paste both outputs into the Log.

Commit `#292 M2: carbonyl live conformance probe`.

### Task 2.7: Measure the chain and settle the cap

- [ ] With the operator: in a live couch thread, open a browser tab on the
  scratch animation page (`anim.html` from the spike, served on localhost),
  then on an idle page.
  - Sample 10 s of CPU time for the Carbonyl group, `pair term` (the tab's
    parent), the zellij server for that session, and couch.
  - Use the spike's `termcost.py` approach: `ps -o time=` deltas.
    `doctor/perf.sh` also works if the operator prefers it.
  - Repeat at `PAIR_BROWSER_FPS`-equivalent caps 60/30/15/10, launched by hand
    with the same argv from a shell tab: the fps is a constant in code, so
    manual launches vary it without a code change.
- [ ] Record the table in `## Log`. Keep `DefaultFPS = 15` unless the chain at
  15 fps costs more than ~25% of a core on the full-motion page; if it does,
  lower it and record why. If the operator finds 15 fps too choppy while
  scrolling, raise it, again with the numbers.
- [ ] Update the `DefaultFPS` comment to cite the recorded measurement.
- [ ] Commit `#292 M2: frame cap settled from the chain measurement`.

### Task 2.8: M2 close

- [ ] `go vet ./...`, `TMPDIR=<scratchpad> make test`, `make probe-carbonyl`
  (with `PAIR_CARBONYL` pointing at 0.0.3).
- [ ] Fuzz 15 s each: `FuzzDecodeCDPMessage` (`./cmd/internal/browsertab/`),
  `FuzzStripLabel` (`./cmd/internal/termcmd/`).
- [ ] Operator smoke test: the label follows navigation; clicking the tab →
  field → Enter navigates.
- [ ] Atlas: `atlas/architecture.md` `pair term` section, covering the
  DevTools controller, the state machine and the conformance probe.
- [ ] `sdlc milestone-close --issue 292 --milestone M2`.

---

## Chunk 3: M3 — Record and `pair browser`

### Task 3.1: Record codec and resolver (pure)

**Files:** `cmd/internal/browsertab/record.go`, `record_test.go`.

```go
type Record struct {
	Version      int      `json:"version"`
	Tag          string   `json:"tag"`
	Name         string   `json:"name"`
	Named        bool     `json:"named"`
	URL          string   `json:"url"`
	Title        string   `json:"title"`
	CDPHTTP      string   `json:"cdp_http"`
	CDPWebSocket string   `json:"cdp_websocket"`
	Browser      Identity `json:"browser"`
	Owner        Identity `json:"owner"`
	TabID        int      `json:"tab_id"`
	ProfileDir   string   `json:"profile_dir"`
}

const RecordVersion = 1

func RecordFile(owner Identity, tabID int) string     // OwnedName(owner.PID, owner.Birth, tabID) + ".json"
func RecordTempFile(owner Identity, tabID int) string // RecordFile(owner, tabID) + ".tmp"
func EncodeRecord(r Record) ([]byte, error)      // json.MarshalIndent + "\n"
func DecodeRecord(b []byte) (Record, error)      // strictjson.Decode + r.Validate()
func (r Record) Validate() error                 // version, name, tag, pids>0, loopback cdp_http (http://127.0.0.1:<port>) and cdp_websocket (ws://127.0.0.1:<port>/devtools/browser/…)
func Resolve(live []Record, name string) (Record, error) // exactly one Name match; else ErrNotFound / ErrAmbiguous (listing owners)
```

The state machine's `FxPublish` carries `State.Record(tag, owner, browser
Identity, tabID, profile)`. That method sanitizes `URL` and `Title` with
`rowtext.SanitizeAndFit(…, 512)` and fills the endpoint from
`State.Endpoint()`.

Tests:
- `FuzzDecodeRecord`: seeds and properties from the strategy table.
- A round trip.
- Rejection of each class: a duplicate key, trailing data, version 2, an empty
  name, pid 0, and each of `cdp_http = "http://10.0.0.1:9"`,
  `"http://127.0.0.1.evil.com:9"` and `"https://127.0.0.1:9"`.
- `Resolve` for 0, 1 and 2 matches.
- `State.Record` strips `\x1b` from a title.

Commit `#292 M3: browser record codec and resolver`.

### Task 3.2: `Paths.BrowserDir` and family registration

**Files:** `cmd/internal/artifactpath/paths.go`, `manifest.go`, `gc.go` and their tests.

- `func (p Paths) BrowserDir() string { return p.tagged("browser-", "") }`
  beside `QueueDir` (paths.go:484).
- Register it. The existing guards fail until each is done (digest of
  `artifactpath`, 2026-09-19):
  - `Families` token `{Name: "browser", Token: "browser-"}` (manifest.go ~175),
    and add it to paths.go's `SourceClassification`.
  - `ResolvedBindings`: `{Name: "scoped-browser", Family: "browser", Resolver:
    "ResolveScoped", Member: "BrowserDir"}`.
  - List every consumer file (`cmd/internal/termcmd/browser.go`,
    `cmd/internal/browsercmd/browser.go`, `cmd/internal/browsertab/store.go`)
    as a `ResolvedConsumer`.
  - `GCClassifications`: collectable with the owner tag, directory family.
  - `ownerCandidates` in gc.go ~210: `ArtifactMember{Path: p.BrowserDir(),
    Family: "browser", Directory: true}`, plus a member pattern
    `^[1-9][0-9]*-[0-9a-f]{12}-[0-9]+\.json(\.tmp)?$` (`browserMemberPattern`).
    It covers the record **and** its temp, so a temp stranded by a SIGKILL mid-publish is
    a recognized, collectable member, not a scope-wide storagegc blocker
    (`storagegc/inventory.go:236-256`). Add the prefix in
    `NewMatchIndex` (~386).
  - `RenameArtifacts`: **not** registered. Records are live-session pointers
    that die with the session, and `pair rename` is offline.
    `RenameArtifacts`'s guard (if any) records that with a comment citing #292.
- Tests that must pass:
  - `TestProductionArtifactReferencesAreExactlyClassified`
  - `TestNonArtifactSourceCannotImportArtifactpath`
  - `TestGCManifestExhaustive`
  - `TestGCCollectableFamiliesHaveExactCandidates`
  - Add `TestBrowserDirIsTagScoped` (two tags → two dirs; a tag containing `-`
    doesn't collide with a longer tag's dir under `MatchIndex`).

Run: `go test ./cmd/internal/artifactpath/ ./cmd/internal/storagegc/ -count=1`
Expected: PASS.

Commit `#292 M3: artifactpath browser-<tag>/ family, registered for GC`.

### Task 3.3: Record store, publishing, and the dead-owner sweep

**Files:** `cmd/internal/browsertab/store.go`, `store_test.go`;
`cmd/internal/termcmd/browser.go` (`FxPublish` / `FxUnpublish`, name uniqueness
across the tag, sweep at launch).

- **`Store{Dir}`:**
  - `Write(owner Identity, tabID int, r Record) error`:
    1. MkdirAll 0700.
    2. Write `RecordTempFile(owner, tabID)`: O_CREATE|O_TRUNC, 0600, write,
       fsync.
    3. Rename it onto `RecordFile(owner, tabID)`.

    The temp's name is deterministic, so a tab has at most one. It carries the
    owner, so it's sweepable. It matches `browserMemberPattern`, so storagegc
    recognizes it (PQ-2).
  - `Remove(owner, tabID) error`: removes the record and its temp; ENOENT is
    nil.
  - `List() ([]Entry, error)`: `Entry{File string; Record Record; Err error}`
    for `*.json` only. Temps are never read.
  - `SweepDeadOwners(probe OwnerProbe) ([]string, error)`: for every entry
    whose name parses with `ParseOwnedName` (after trimming `.json` or
    `.json.tmp`), remove it when `probe.InspectHash(pid, hash) ==
    ProcessDead`. It never decodes contents, so truncated records and stranded
    temps are covered by the same proof. A name that doesn't parse isn't ours
    and stays.
- **Where pair term gets scope and tag:** `workbenchshortcut.DataDirFromEnv()`
  and `os.Getenv("PAIR_TAG")`, the same pair `OSRuntime` already reads for its
  pane sidecars (`termcmd/run.go`, `LastLeftPaneStore`). They're resolved once
  into `browserDeps.records *browsertab.Store`.
- **No record when there's no valid tag.** If `PAIR_TAG` is empty (someone ran
  `pair term` by hand outside a session), or `artifactpath.ResolveScoped`
  rejects it, `records` is nil. `FxPublish`/`FxUnpublish` become no-ops, and the
  first launch flashes `no pair tag: this browser is not shared with the agent`.
  The tab itself works. Standalone pair always has a tag (it's couch that's
  optional, not the tag), so the no-tag case is only a hand-run `pair term`.
- **Controller:**
  - `FxPublish` is coalesced: at most one write per 250 ms, and the last state
    wins (a `time.AfterFunc` that re-reads the latest snapshot inside the
    loop).
  - `FxUnpublish` cancels a pending coalesced write and then removes the file.
- **`newBrowserTab`:** before naming, `SweepDeadOwners(probe)`, then add the
  names of every *live* record (`probe.Inspect(r.Browser) == ProcessAlive`) to
  the taken set.
- **Tests:**
  - `store_test.go`:
    - `List` never reads `.tmp`: plant a temp holding garbage and assert it's
      absent from the entries.
    - `SweepDeadOwners` with a fake probe:
      - a dead owner's record, **its truncated record**, and **its stranded
        temp** are all removed;
      - alive and unknown owners' files stay;
      - a file whose name doesn't parse stays.
    - A deterministic-name test: two `Write`s for one tab leave exactly one
      `.json` and no `.tmp`.
  - An artifactpath/storagegc test (Task 3.2): a scope holding a live tag's
    `browser-<tag>/<owned>.json.tmp` is inventoried without a blocker.
  - A termcmd integration test on the fake: open a tab → the record appears with
    `name: web`, a loopback `cdp_http`, and the Carbonyl pid; rename it (Alt+r
    `local-test`) → the record's `name` is `local-test` and `named` is true;
    navigate → the record's `url` follows; Alt+w → the record is gone.
  - Two muxes on the same tag each open a tab → the names are `web` and
    `web-2`.

Commit `#292 M3: browser records published by the owner; dead-owner sweep`.

### Task 3.4: `pair browser`

**Files:** Create `cmd/internal/browsercmd/browser.go`, `browser_test.go`;
Modify `cmd/internal/dispatcher/dispatcher.go` (`Families()`: `{Name:
"browser", Summary: "print a right-pane browser tab's DevTools address (JSON)",
Status: "implemented"}` and its buffered `Dispatch` case);
`cmd/internal/launcher/help.go` `UsageText` (one line).

```
usage: pair browser [<name> | <tag>:<name>]
  no argument   list this session's browser tabs (JSON array)
  <name>        one tab of this session's tag
  <tag>:<name>  one tab of <tag> in this repo's scope
Output fields: tag, name, named, url, title, cdp_http, cdp_websocket, pid, profile_dir.
Connect with Playwright: chromium.connectOverCDP(cdp_http). Carbonyl shows one
page: drive browser.contexts()[0].pages()[0], don't open new pages.
```

- **Scope:** `PAIR_DATA_DIR` when set. Otherwise
  `launcher.ScopedLaunchDataDir(adapt.DataDir(), cwd)`, using the global data
  dir resolver the launcher uses. The tag is `PAIR_TAG`, or the argument's
  prefix. No tag and no argument → exit 2 with usage.
- **Reading:** `Store.List()`, then keep entries whose `Browser` identity is
  `ProcessAlive`. Undecodable or non-live entries go to stderr (`skipped
  <file>: <reason>`) and are never deleted: the reader isn't the owner.
- **Exit codes:** 0 with JSON; 1 for not found or ambiguous (message lists
  candidates); 2 for usage.
- **Tests,** with an injected `Env` / `Probe` / `Getwd`:
  - no args lists two live records and skips a dead one;
  - `web` resolves;
  - `tag:web` resolves across tags;
  - an ambiguous name → 1;
  - a hand-edited non-loopback record is skipped with a stderr line;
  - dispatch routing: `TestEveryImplementedFamilyIsRoutable` and
    `TestLuaBuiltPairSubcommandsAreDeclaredAndRoutable` stay green.
- **The Done-when end-to-end test** (termcmd or browsercmd package,
  unsandboxed):
  1. A mux opens a browser tab on the fake with `PAIR_DATA_DIR`/`PAIR_TAG` set.
  2. Run `browsercmd.Run([]string{tag + ":web"}, …)` and parse its JSON.
  3. `browsertab.Dial(cdp_websocket)` → `DiscoverTargets` →
     `Navigate(page, "http://agent.test:9/")`.
  4. Assert the mux's strip label becomes `web · agent.test:9` and the record's
     `url` follows.

Commit `#292 M3: pair browser prints the tab's DevTools address`.

### Task 3.5: M3 close

- [ ] `go vet ./...`, `TMPDIR=<scratchpad> make test`, `make probe-carbonyl`.
- [ ] Fuzz 15 s: `FuzzDecodeRecord` (`./cmd/internal/browsertab/`).
- [ ] Operator smoke test in couch:
  - open a tab, rename it `local-test`;
  - in the agent pane run `pair browser local-test`, then a short Playwright
    (or `curl <cdp_http>/json/list`) check that sees the operator's page;
  - Alt+w → `pair browser local-test` exits 1;
  - `pgrep -fl carbonyl` is empty;
  - `ls ~/Library/Caches/pair/browser` is empty.
- [ ] Atlas:
  - `atlas/architecture.md`: the record, `pair browser`, and the lifecycle
    table;
  - `atlas/storage-retention.md`: the `browser-<tag>/` family, and why profiles
    live outside the data root;
  - `atlas/index.md`: nothing new unless a new atlas file is added.
- [ ] `workshop/lessons.md` if the boundary reviews found anything.
- [ ] `sdlc milestone-close --issue 292 --milestone M3`, then `sdlc close
  --issue 292 --verified '<evidence>'`, then `sdlc pr` → `sdlc merge`.

---

## Revisions

### 2026-09-19 — plan-gate round 1

Each finding was answered at the class level, not just at the site it named.
- **PQ-1 (the plan assumed existing behavior that isn't there):** Ctrl+U was
  claimed but isn't mapped.
  - Task 1.5 adds `\x15 → RenameDeleteToStart`.
  - The new "Existing behavior this plan relies on" table lists every such
    claim with its citation or its pinning test.
- **PQ-2 (files with no removal path):** a crash could strand a record temp.
  - One owner-bearing name scheme (`OwnedName`) now covers profiles, records and
    temps.
  - The sweep proves the owner dead from the name, never from contents.
  - The GC member pattern covers `.json(.tmp)?`.
  - Tests cover a truncated record and a stranded temp.
- **PQ-3 (no adversarial tests):** an adversarial test strategy table adds
  six fuzz targets, one for each parser of untrusted input, with properties.
  Each milestone close runs the relevant ones.
- **Minor findings:**
  - `State` is the single authority for a browser tab's name and named flag
    (the mux no longer writes them).
  - The mailbox coalesces snapshots per target and never drops a control event.
  - Scope and tag come from `DataDirFromEnv`/`PAIR_TAG`, and no tag means no
    record.
  - The dependency choice is weighed in a table.

### 2026-09-19 — plan-gate round 2 (advisory PQ-7)

- **The claim:** the plan said ptychild reaps only after pty EOF. It also reaps
  after `Close()` and after a failed delivery, and both of those killed only
  the leader.
- **The class fix:** `Options.KillGroup` routes every kill site in ptychild
  through a group kill before the reap. The no-orphan guarantee is now
  structural instead of depending on the caller's close order.
- **Pinned by** `TestGroupKillOptionCoversEveryReapPath`, using a grandchild
  that ignores SIGHUP, so only the group kill can end it.
- **Corrected** the relied-on-behavior table row.
