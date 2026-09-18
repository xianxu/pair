# Context-Aware Alt+h Help Implementation Plan (#282)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Alt+h stays Pair's key and fires in the draft, or in the right terminal through
Pair's existing routing. The agent pane keeps receiving it. When the attached client was
launched by Couch, Pair's help pager shows Couch's keys above Pair's own. A key Couch
claims replaces Pair's row for it. In a standalone Pair session the help is unchanged,
except that the floating pane's title becomes "help".

**Architecture:** Two facts decide the page, and each is read from its own authority:

- **Who presents the attached client.** The Pair client that attaches knows. Couch
  launches every client it presents with `COUCH_THREAD_TAG` equal to the thread's tag,
  on every attach, including a warm reattach of a session Couch did not create. The
  client adds that fact to the per-attach outer-tty record it already writes.
  `pair keys` reads the record.
- **Whether Pair's own behavior is hosted.** Pair's Alt+n does not reload when either
  the session env names Couch (`pair restart` refuses, #249) or the attached client
  was launched by Couch (the client refuses the restart marker after the quit
  cleanup). The help's Alt+d/Alt+n wording therefore applies when
  `launcher.CouchHostedEnv(session env) || couchPresents`.

Couch's chords move into a pure package, `couchkeys`. couchtty's framing and routing,
`couch --help`, and Pair's page all derive from it. keyhelp gains typed chord identity on
its rows and a host-agnostic `Layer`.

**Tech Stack:** Go. Packages: `cmd/internal/{launcher, keyhelp, couchkeys (new), couchtty, couchcmd, keyscmd, workbenchshortcut, dispatcher, artifactpath}`; `nvim/init.lua` (pane title).

---

## Background: what the code does today

**Couch forwards alt+d/x/n to the displayed Pair pane (#245).** `knownSequences` frames
those chords, but `Console.dispatchInputCandidate` (`couchtty/console.go:1671`)
forwards every non-`actorReserved` hit to the displayed pane. Couch acts on them only in
the switcher, and takes only Ctrl+Space, Ctrl+Backspace and Ctrl+Return from a Pair pane.

**Hosting changes Pair's own Alt+n and Alt+d.** Since #249, `pair restart` refuses when
`COUCH_THREAD_SCOPE`/`TAG` is set. Pair's Alt+n in a Couch-launched pane therefore
confirms and then silently does nothing (`pair#284`). Pair's Alt+d detaches only its
Zellij client.

**Alt+h's current path.**
1. The draft keymap or Pair's right-terminal routing calls `PairOpenHelp`
   (`nvim/init.lua:3469`).
2. That runs `zellij run --floating --name 'pair help' -- pair-help`.
3. `pair-help` runs `pair keys --center <cols> | less`.

Verified live on 2026-09-18 in a Couch thread:
- The draft nvim and the help pane inherit the Zellij server's env: all five `COUCH_*`
  vars.
- `$PAIR_OUTER_TTY_PATH` = `$PAIR_DATA_DIR/outer-tty-$PAIR_TAG` holds `/dev/ttys006`. It
  was written at 08:52, the same moment this Couch started.
- Nothing reads that record except quit cleanup and GC.

## Decisions (operator, 2026-09-18)

- **Couch does not take Alt+h.** Couch should not take keys from panes it cannot tell
  apart, and it keeps no inner-focus state (#245). So Alt+h stays Pair's: from the draft,
  and from the right terminal through Pair's existing routing, which stays unchanged.
  Under Couch the agent pane still receives Alt+h.
- **Pair's pager displays the combined page.** There is no Pair→Couch request channel
  and no Couch panel frame.
- **Presenter comes from the attached client, not the session env.** The session env
  records who *created* the Zellij session, so it misreads a session Couch adopted
  (`pair#246`). The client's own env is set by whoever launched *this attach*.
  - `launcher.PresentedByCouch(env, tag)` is `env.CouchThreadTag != "" &&
    env.CouchThreadTag == tag`. Requiring the tag to match refuses a stray env from a
    nested launch.
  - The fact is appended to the outer-tty record, which is written on every create
    and attach. This adds no new artifact family and no new lifecycle: the record
    already has cleanup and GC.
- **Hosted wording applies when Couch launched the session *or* presents this
  client.** Pair's Alt+n is gated twice (measured by the third plan review):
  1. `pair restart`, run by nvim with the *session* env, refuses when that env
     names Couch (`runcli.go:133`).
  2. The attached *client* refuses the restart marker when its own env names
     Couch, after it has already run the full quit cleanup
     (`createflow.go:129`, `:162`).

  In a session Couch adopted, check 1 passes and check 2 fires, so Alt+n *ends
  the thread without relaunching it*. The truthful predicate is
  `CouchHostedEnv(session env) || couchPresents`, and the Alt+n wording warns
  rather than saying "refused". Check 2 had no test, so Task 3 pins it. The
  destructive adopted case is recorded in `pair#284`.
- **Pane title** `'pair help'` → `'help'`. The pane opens before the page is built, so a
  context-dependent title would force the Lua to detect Couch as well. The first section
  of the page already names the layer.
- **`pair keys` from a shell is not a supported surface** and gets no special handling.
  Its output is whatever the record and env say.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `launcher.Env.CouchHosted` / `launcher.CouchHostedEnv` | `cmd/internal/launcher/run.go` | new |
| `launcher.PresentedByCouch` / `OuterRecord` / `EncodeOuterRecord` / `DecodeOuterRecord` | `cmd/internal/launcher/outerrecord.go` | new |
| `workbenchshortcut.GlobalBinding.HostedHelp` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `keyhelp.Binding.Chord` / `ContextHost` / `ContextHostMenu` | `cmd/internal/keyhelp/keyhelp.go` | modified |
| `keyhelp.Sections` / `HostedSections` (`sections(src, hosted)`) | `cmd/internal/keyhelp/sections.go` | modified / new |
| `keyhelp.Layer` | `cmd/internal/keyhelp/layer.go` | new |
| `couchkeys.Binding` / `Bindings` / `Scope` / `Action` / `HelpSections` / `Claimed` | `cmd/internal/couchkeys/couchkeys.go` | new |
| `couchChords` / `dispatchFor` and derived `seqKind.hit` / `actorReserved` / `knownSequences` / `FeedHit` | `cmd/internal/couchtty/keys.go` | modified |
| `CouchNavigationBinding` / `CouchNavigationBindings` / `couchNavigation` | `cmd/internal/couchtty/keys.go` | deleted |

- **`OuterRecord{TTY string; Couch bool}`**: the per-attach outer-tty record.
  - Format: line 1 is the tty path, unchanged. Line 2 is `presenter=couch` or
    `presenter=terminal`.
  - `DecodeOuterRecord` reads a legacy one-line record as `Couch: false`. It rejects
    anything else malformed with an error.
  - **Relationships:** one per tag. The last attach wins.
  - **DRY rationale:** one codec shared by the writer (`RecordOuterTTY`) and the reader
    (`ReadOuterPresenter`).
- **`PresentedByCouch(env, tag)`**: Couch launched *this* client for *this* thread.
- **`CouchHosted` / `CouchHostedEnv`**: `scope != "" || tag != ""`. It replaces five
  inline copies in the launcher: `runcli.go` ×3, `createflow.go:162` and
  `compaction.go:90`. It is half of the help's hosted predicate; the other half is the presenter record.
- **`HostedHelp`**: set on Alt+d, Alt+n and Ctrl+Alt+n. The wording holds whether or not
  Couch still runs.
- **`keyhelp.Binding.Chord`**: set from `GlobalBinding.Chord` and `RoleBinding.Chord`;
  zero for draft-local keys. **`Layer(host, claimed, pair)`**: the host's sections
  first, then Pair's rows minus any claimed chord in any context. Emptied sections are
  dropped and the inputs are not mutated. keyhelp never names Couch.
- **`couchkeys.Binding{Action, Scope, Chord, Key, Help, Encodings}`**:
  - Couch's seven chords: three every-pane navigation chords, plus switcher-scope
    Alt+d, Alt+x, Alt+n and Ctrl+Alt+n.
  - `Chord` is the Pair chord a binding shares (switcher chords) or claims (a future
    every-pane override). Switcher encodings derive from
    `workbenchshortcut.ChordEncodings`.
  - `HelpSections(bindings)` gives one section per scope; the switcher title names its
    opener. `Claimed(bindings)` gives the every-pane chords that are Pair chords (none
    today).
  - **DRY rationale:** Couch's chord facts live in four places today:
    `couchNavigation`, the anonymous lifecycle list in `knownSequences`,
    `actorReserved`, and `couchcmd.usage`. After this change all four, plus Pair's page,
    read one table.
  - **Placement:** a pure package imported by couchtty and couchcmd (Couch) and by
    keyscmd (Pair), so Pair never imports Couch's console.
    `couchkeys → keyhelp` points Couch→Pair, which is allowed.
- **`dispatchFor(action) (InterceptorHit, seqKind)`**: couchtty's one mapping from a
  declared action to console dispatch. `actorReserved` reads `Scope`, so routing and the
  help's context cannot disagree.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `OSRuntime.RecordOuterTTY(tag, couch)` | `cmd/internal/launcher/osruntime.go:478` | modified | `tty` + record file |
| `launcher.ReadOuterPresenter(dataDir, tag)` | `cmd/internal/launcher/osruntime.go` | new | record file |
| `keyscmd.Deps` / `RunWith` | `cmd/internal/keyscmd/keyscmd.go` | new (replaces `RunWithSources`) | env + record + embedded sources |
| `couchcmd.usageWith` | `cmd/internal/couchcmd/run.go` | new (`usage` delegates) | `couch --help` stdout |

- The writer and the reader sit side by side and resolve the path the same way
  (`artifactpath.ResolveScoped(dataDir, tag).OuterTTY()`), under the existing
  `scoped-outer-tty` binding. `keyscmd` passes `PAIR_DATA_DIR` and `PAIR_TAG` from its
  env. The live session confirmed that those resolve to `$PAIR_OUTER_TTY_PATH`.
- **Injected:**
  - `keyscmd.Deps{Sources, Getenv, CouchPresents func() (bool, error)}`. Tests inject
    all three, so they stay hermetic when the repo is tested inside a Couch thread,
    which it routinely is.
  - The launcher's `Runtime` interface carries the new `RecordOuterTTY` signature. Its
    `fakeRuntime` records the argument.
- **Test surface:**
  - The codec is pure.
  - `ReadOuterPresenter` runs against a temp data dir on the real filesystem.
  - The launcher create and attach tests assert what the fake records.
  - The acceptance test is keyscmd's full `RunWith` from env + record to the rendered
    page.
  - Live conformance is the operator smoke on a rebuilt Couch.

### Consumer enumeration (ARCH-PURPOSE shadow-sweep)

Readers of Couch's chord table after this change, all deriving from
`couchkeys.Bindings()`:

1. couchtty framing: `FeedHit` and `knownSequences`.
2. couchtty routing: `hit` and `actorReserved`.
3. `couch --help`.
4. Pair's page, through keyscmd.

Prose and inventories to sweep in Task 8:
- `couchtty.menuControls` is a README coverage list; its Keys already cover the chords.
- README:
  - line 118: Alt+h row.
  - line 151: Alt+d row.
  - line 153: Alt+n row.
  - lines 378–381: stale "Alt+d is intercepted by Couch". Also drop the actor-detach
    focus sentence, which is unreachable from the keyboard since #245.
  - lines 496–497: "existing Pair reload in draft/right panes".
- `atlas/architecture.md:299-309`: names `couchtty.CouchNavigationBindings`.
- `atlas/couch.md`:
  - `:482`: `hotkeyByte`, which moves to couchkeys.
  - `:515`: `newestPageSequence`, which moves.
  - `:648-657`: the lifecycle chords.
  - `:733-734`: "other panes retain Pair's existing in-process reload", which is false
    under #249.
- `artifactpath/deletedvocabulary_test.go`: add `CouchNavigationBinding`.

### Operating envelope and other lenses

- **ARCH-CONSTRAINTS.** A one-shot UI response: the floating pane opens on Alt+h. The
  added work is one small file read and one env check, well under 1 ms. The budget for
  `pair keys` stays under 100 ms; `time pair keys` is recorded in the Log. Nothing
  blocks and nothing fans out.
- **ARCH-SECURE.** The record crosses a process and a version boundary: it is written by
  a launcher, possibly an older one, and read by `pair keys`.
  - `DecodeOuterRecord` is strict. A legacy one-line record means *not presented*.
    Any other shape is an error, and `pair keys` then renders Pair's page without
    Couch's section and writes the cause to stderr. Strictness buys *visible
    degradation* on a malformed or truncated record. A well-formed hand-written
    `presenter=couch` record does produce Couch's section; the file is the same
    user's, and a hand edit there is a deliberate act.
  - No secrets are involved.
  - Tests inject env and the record instead of reading the developer's.
- **ARCH-ORDER.** The record's presenter changes on attach (overwrite; the last attach
  wins) and on quit cleanup (removed). `pair keys` reads it once, as a point-in-time
  observation.
  - *Two clients attached at once* (Couch plus a direct terminal): the last writer
    wins, and the other client's Alt+h shows the other layer. Accepted and recorded.
  - *Couch crashes*: its Pair client dies with the pty, so the record says `couch`
    while no client is attached, and nobody can press Alt+h. The next `pair resume`
    overwrites it.
  - *A raw `zellij attach` that bypasses Pair* would leave a stale `couch`. That is
    outside the Pair workflow; accepted.
  - The event most likely to be mishandled is a failed tty probe at attach. Today
    `RecordOuterTTY` *removes* the record in that case, and it must keep doing so,
    because a leftover `couch` line from an earlier attach would lie. There is a test
    for it.
- **ARCH-FUNERAL.** Creates nothing new. The outer-tty record gains one line (about 16
  bytes). Its writers, cleanup (`lifecycle.go:163`) and GC (`artifactpath/gc.go:226`)
  are unchanged.
- **ARCH-MOCK.** `fakeRuntime` (launcher) is extended rather than replaced. The record
  reader is tested on the real filesystem. No external binary is involved.
- **ARCH-PURE.**
  - Pure: `couchkeys` (including `HelpSections` and `Claimed`), `Layer`, `sections`,
    the `OuterRecord` codec, `PresentedByCouch`, `CouchHosted`, `dispatchFor`.
  - IO: `RecordOuterTTY`, `ReadOuterPresenter`, `keyscmd.Run`.

---

## Chunk 1: Pair's side — hosted rule, presenter record, hosted wording, `Layer`

### Task 1: one hosted rule in the launcher

**Files:** `cmd/internal/launcher/run.go` (after `Env`); `runcli.go:117,133,146`, `createflow.go:162`, `compaction.go:90`; test `cmd/internal/launcher/couchhosted_test.go` (new).

- [ ] **Step 1: Failing test**

```go
package launcher

import "testing"

func TestCouchHostedIsEitherThreadVar(t *testing.T) {
	for _, tc := range []struct {
		scope, tag string
		want       bool
	}{{"", "", false}, {"s", "", true}, {"", "t", true}, {"s", "t", true}} {
		env := map[string]string{"COUCH_THREAD_SCOPE": tc.scope, "COUCH_THREAD_TAG": tc.tag}
		if got := CouchHostedEnv(func(k string) string { return env[k] }); got != tc.want {
			t.Errorf("CouchHostedEnv %+v = %v", tc, got)
		}
		if got := (Env{CouchThreadScope: tc.scope, CouchThreadTag: tc.tag}).CouchHosted(); got != tc.want {
			t.Errorf("Env.CouchHosted %+v = %v", tc, got)
		}
	}
}
```

- [ ] **Step 2: Run → FAIL.** `go test ./cmd/internal/launcher/ -run CouchHosted` fails with `undefined: CouchHostedEnv`.
- [ ] **Step 3: Implement** in `run.go`:

```go
// CouchHosted reports whether Couch launched this Pair process's session:
// Couch sets COUCH_THREAD_SCOPE and COUCH_THREAD_TAG on every thread it
// launches (couchcore/launch_existing.go), and either marks it hosted. The
// launcher's hosted refusals and Pair's hosted help wording share this rule
// (#282).
func (e Env) CouchHosted() bool { return e.CouchThreadScope != "" || e.CouchThreadTag != "" }

// CouchHostedEnv is CouchHosted read from a process environment.
func CouchHostedEnv(getenv func(string) string) bool {
	return Env{CouchThreadScope: getenv("COUCH_THREAD_SCOPE"), CouchThreadTag: getenv("COUCH_THREAD_TAG")}.CouchHosted()
}
```

Replace the five `X.CouchThreadScope != "" || X.CouchThreadTag != ""` guards with
`X.CouchHosted()`. Leave `createflow.go:80` (`&&`, a different predicate) and
`compaction.go:91` alone.

- [ ] **Step 4: Run.** `go test ./cmd/internal/launcher/` → PASS, including `TestCheckpointHostedRestartAndRenameRefuseBeforeMutation`.
- [ ] **Step 5: Commit.**

```bash
git add cmd/internal/launcher/run.go cmd/internal/launcher/runcli.go cmd/internal/launcher/createflow.go cmd/internal/launcher/compaction.go cmd/internal/launcher/couchhosted_test.go
git commit -m "#282: launcher: one CouchHosted rule for refusals and help"
```

### Task 2: the attached client records whether Couch presents it

**Files:**
- Create: `cmd/internal/launcher/outerrecord.go`, `outerrecord_test.go`
- Modify: `cmd/internal/launcher/runtime.go:122-124` (interface), `osruntime.go:478-493` (writer + new reader), `createflow.go:769`, `lifecycle.go:82`, `createflow_test.go:289` (fake)
- Modify: `cmd/internal/artifactpath/manifest.go` (`NonArtifactSources` += `cmd/internal/launcher/outerrecord.go`: the codec names no path)

- [ ] **Step 1: Failing tests**

`outerrecord_test.go`:

```go
package launcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestOuterRecordRoundTrips(t *testing.T) {
	for _, r := range []OuterRecord{{TTY: "/dev/ttys006", Couch: true}, {TTY: "/dev/ttys001"}} {
		got, err := DecodeOuterRecord(EncodeOuterRecord(r))
		if err != nil || got != r {
			t.Fatalf("%+v → %+v, %v", r, got, err)
		}
	}
}

// A record written by a launcher from before #282 has only the tty line. It
// was never Couch-aware, so it says nothing about Couch: not presented.
func TestLegacyOuterRecordIsNotPresented(t *testing.T) {
	got, err := DecodeOuterRecord("/dev/ttys006\n")
	if err != nil || got.Couch || got.TTY != "/dev/ttys006" {
		t.Fatalf("%+v, %v", got, err)
	}
}

// Anything else is refused, never guessed: a hand edit or truncation must not
// conjure a Couch section.
func TestMalformedOuterRecordIsAnError(t *testing.T) {
	for _, raw := range []string{"", "\n", "tty\n", "/dev/ttys006\npresenter=maybe\n", "/dev/ttys006\npresenter=couch\nextra\n", "/dev/ttys006\ncouch\n"} {
		if _, err := DecodeOuterRecord(raw); err == nil {
			t.Errorf("%q decoded without error", raw)
		}
	}
}

// Couch launched this client for this thread: its tag must match, so a stray
// COUCH_THREAD_TAG inherited by a nested launch is not mistaken for Couch.
func TestPresentedByCouchRequiresTheThreadsTag(t *testing.T) {
	if !PresentedByCouch(Env{CouchThreadScope: "s", CouchThreadTag: "couch-a"}, "couch-a") {
		t.Error("Couch's own client not recognised")
	}
	for _, env := range []Env{{}, {CouchThreadScope: "s"}, {CouchThreadTag: "couch-b"}} {
		if PresentedByCouch(env, "couch-a") {
			t.Errorf("%+v recognised as Couch for couch-a", env)
		}
	}
}

// The reader and the writer share one path resolution; a missing record is
// "not presented" (a non-tty attach removes it), not an error.
func TestReadOuterPresenter(t *testing.T) {
	dir := t.TempDir()
	if couch, err := ReadOuterPresenter(dir, "work"); err != nil || couch {
		t.Fatalf("missing record: %v %v", couch, err)
	}
	paths, err := artifactpath.ResolveScoped(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.OuterTTY(), []byte(EncodeOuterRecord(OuterRecord{TTY: "/dev/ttys006", Couch: true})), 0o600); err != nil {
		t.Fatal(err)
	}
	if couch, err := ReadOuterPresenter(dir, "work"); err != nil || !couch {
		t.Fatalf("couch record: %v %v", couch, err)
	}
	if err := os.WriteFile(paths.OuterTTY(), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadOuterPresenter(dir, "work"); err == nil {
		t.Fatal("malformed record read without error")
	}
}

// DecodeOuterRecord is the one parser of input this change did not produce:
// the record is truncatable, hand-editable and cross-version. Couch is true
// only for the exact two-line presenter=couch shape, and nothing panics.
func FuzzDecodeOuterRecord(f *testing.F) {
	for _, seed := range []string{"/dev/ttys006\n", "/dev/ttys006\npresenter=couch\n", "/dev/ttys006\npresenter=terminal\n",
		"", "\n", "tty\n", "/dev/ttys006\npresenter=maybe\n", "/dev/ttys006\npresenter=couch\nextra\n"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		got, err := DecodeOuterRecord(raw)
		lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
		exact := len(lines) == 2 && strings.HasPrefix(lines[0], "/dev/") && lines[1] == "presenter=couch"
		if err == nil && got.Couch != exact {
			t.Fatalf("%q decoded Couch=%v", raw, got.Couch)
		}
	})
}

// An attach whose stdin is not a tty removes the record rather than keeping an
// earlier attach's presenter line, which would lie about who presents now.
func TestNonTTYAttachRemovesThePresenterRecord(t *testing.T) {
	dir := t.TempDir()
	paths, err := artifactpath.ResolveScoped(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.OuterTTY(), []byte(EncodeOuterRecord(OuterRecord{TTY: "/dev/ttys006", Couch: true})), 0o600); err != nil {
		t.Fatal(err)
	}
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devnull.Close()
	saved := os.Stdin
	os.Stdin = devnull
	defer func() { os.Stdin = saved }()
	NewScopedOSRuntime(dir, dir, "/pair").RecordOuterTTY("work", true)
	if _, err := os.Stat(paths.OuterTTY()); !os.IsNotExist(err) {
		t.Fatalf("record survived a non-tty attach: %v", err)
	}
}
```

(The third plan review measured this passing in the sandbox. Check
`NewScopedOSRuntime`'s signature before relying on it.) In `createflow_test.go` (it needs
the `strconv` import) and in the attach test that exercises `lifecycle.go:82`, extend
`fakeRuntime.RecordOuterTTY` to record `tag+"|"+strconv.FormatBool(couch)`. Then assert
that a create or attach with `Env{CouchThreadScope: "s", CouchThreadTag: <tag>}`
records `…|true`, and that one with empty Couch env records `…|false`. Find the
existing assertions with `grep -n ttyRecorded cmd/internal/launcher/*_test.go`; the
review confirmed `baseOpts` supports both cases.

- [ ] **Step 2: Run → FAIL** (undefined codec and reader; signature mismatch).
- [ ] **Step 3: Implement**

`outerrecord.go`:

```go
package launcher

import (
	"fmt"
	"strings"
)

// OuterRecord is what an attaching Pair client writes about its outer terminal
// (outer-tty-<tag>): the tty, and whether Couch is the one presenting it.
// Written on every create and attach, so the last attach wins (#282).
type OuterRecord struct {
	TTY   string
	Couch bool
}

const (
	presenterCouch    = "presenter=couch"
	presenterTerminal = "presenter=terminal"
)

func EncodeOuterRecord(r OuterRecord) string {
	presenter := presenterTerminal
	if r.Couch {
		presenter = presenterCouch
	}
	return r.TTY + "\n" + presenter + "\n"
}

// DecodeOuterRecord is strict. A one-line record predates #282 and says
// nothing about Couch; every other shape is refused rather than guessed.
func DecodeOuterRecord(raw string) (OuterRecord, error) {
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	if lines[0] == "" || !strings.HasPrefix(lines[0], "/dev/") {
		return OuterRecord{}, fmt.Errorf("outer-tty record: bad tty line %q", lines[0])
	}
	switch {
	case len(lines) == 1:
		return OuterRecord{TTY: lines[0]}, nil
	case len(lines) == 2 && lines[1] == presenterCouch:
		return OuterRecord{TTY: lines[0], Couch: true}, nil
	case len(lines) == 2 && lines[1] == presenterTerminal:
		return OuterRecord{TTY: lines[0]}, nil
	}
	return OuterRecord{}, fmt.Errorf("outer-tty record: unexpected shape (%d lines)", len(lines))
}

// PresentedByCouch reports whether Couch launched THIS client for THIS thread.
// The client's env is set per attach by whoever launched it -- unlike the
// Zellij server's env, which records who created the session (#282).
func PresentedByCouch(env Env, tag string) bool {
	return env.CouchThreadTag != "" && env.CouchThreadTag == tag
}
```

`runtime.go`: the interface becomes `RecordOuterTTY(tag string, couch bool)`, with the
comment "writes the launching tty and whether Couch presents this client to
outer-tty-<tag> (or removes it when stdin isn't a tty)".

`osruntime.go`: `RecordOuterTTY(tag string, couch bool)` writes
`EncodeOuterRecord(OuterRecord{TTY: outer, Couch: couch})`. Keep the remove-when-not-a-tty
branch: a stale `couch` line from an earlier attach would lie. Add alongside:

```go
// ReadOuterPresenter reports whether the client that last attached to tag was
// presented by Couch. No record (never attached from a tty, or quit) is "no".
func ReadOuterPresenter(dataDir, tag string) (bool, error) {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return false, err
	}
	raw, err := os.ReadFile(paths.OuterTTY())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	record, err := DecodeOuterRecord(string(raw))
	return record.Couch, err
}
```

(Check `osruntime.go`'s imports for `errors` and `os`.) Call sites:
`createflow.go:769` becomes `rt.RecordOuterTTY(chosenTag, PresentedByCouch(env,
chosenTag))`, and `lifecycle.go:82` becomes `rt.RecordOuterTTY(tag,
PresentedByCouch(env, tag))`.

- [ ] **Step 4: Run.** `go test ./cmd/internal/launcher/ ./cmd/internal/artifactpath/` → PASS. If artifactpath's classifier objects to the new read in `osruntime.go`, extend that file's existing classification. It already carries the `outer-tty` family through `RecordOuterTTY`. Record what changed in the Log.
- [ ] **Step 5: Commit.** `git add cmd/internal/launcher cmd/internal/artifactpath/manifest.go && git commit -m "#282: the attaching client records whether Couch presents it"`

### Task 3: hosted wording on Pair's global bindings

**Files:** `cmd/internal/workbenchshortcut/shortcut.go:132-168`; tests `shortcut_test.go`, `cmd/internal/launcher/createflow_test.go` (`TestCouchClientRefusesRestartMarker`).

- [ ] **Step 1: Failing test**

```go
// Under Couch, Pair's Alt+n / Ctrl+Alt+n do not reload (launcher
// TestCheckpointHostedRestartAndRenameRefuseBeforeMutation,
// TestCouchClientRefusesRestartMarker) and Pair's Alt+d detaches only its
// Zellij client. Alt+Shift+C also changes when hosted (compaction.go:90 hands
// the restart to Couch), but it is not a GlobalBinding: its wording comes from
// nvim's "compact session" desc, which stays true. HostedHelp can only carry
// GlobalBinding rows (#282).
func TestHostedHelpCoversExactlyTheChordsHostingChanges(t *testing.T) {
	want := map[Chord]bool{ChordAltD: true, ChordAltN: true, ChordCtrlAltN: true}
	for _, b := range GlobalBindings() {
		if (b.HostedHelp != "") != want[b.Chord] {
			t.Errorf("%s HostedHelp=%q", ChordName(b.Chord), b.HostedHelp)
		}
	}
}
```

- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement.** Add after `Help`:

```go
	// HostedHelp replaces Help when Couch launched the session or presents
	// this client, for a chord whose Pair behavior changes there (#282); empty
	// means Help holds either way. Alt+n does not reload under Couch: the
	// session-env refusal is pinned by launcher
	// TestCheckpointHostedRestartAndRenameRefuseBeforeMutation, and the
	// client-side marker refusal by TestCouchClientRefusesRestartMarker.
	HostedHelp string
```

Values (keyed literals). None carries its own parentheses, because the `(outside the
agent pane)` suffix follows:
- `ChordAltD`: `"detach only this Zellij client; Couch's own detach is in its switcher"`
- `ChordAltN`: `"does not reload under Couch and may end the thread; relaunch from the Couch switcher"`
- `ChordCtrlAltN`: `"same as Alt+n under Couch"`

Also pin the client-side check (`createflow.go:162`), which had no test.
`createflow_test.go` drives `RunLaunch` through `fakeRuntime`. Find the existing
restart-marker test with `grep -n ReadRestartMarker cmd/internal/launcher/*_test.go`
and use it as the model:

```go
// A Couch-launched client never relaunches from a Pair restart marker: it has
// already run the full quit cleanup, then refuses (createflow.go:162). This is
// why Alt+n under Couch does not reload, and can end an adopted thread
// (pair#284) -- the fact Alt+n's HostedHelp states (#282).
func TestCouchClientRefusesRestartMarker(t *testing.T) {
	// Arrange as the existing restart-marker test does, with
	// opts.Env.CouchThreadScope/CouchThreadTag set to the launched tag and the
	// fake returning a restart marker for the session after the handoff.
	// Assert: exit code 1, stderr contains "legacy hosted restart intent
	// refused", and the fake recorded no second create/attach (no relaunch).
}
```

Fill in the body from the model test. The three assertions above are the contract.

- [ ] **Step 4: Run.** `go test ./cmd/internal/workbenchshortcut/ ./cmd/internal/launcher/` → PASS. `TestCouchClientRefusesRestartMarker` passes on first run: it pins existing behavior. Before trusting that, check it fails if the `createflow.go:162` guard is removed. The generated Lua is unaffected, because Help is not rendered into it; Task 8's full run covers the drift test.
- [ ] **Step 5: Commit.** `git add cmd/internal/workbenchshortcut cmd/internal/launcher/createflow_test.go && git commit -m "#282: GlobalBinding.HostedHelp for the chords hosting changes"`

### Task 4: keyhelp — chord identity, hosted sections, `Layer`

**Files:** `cmd/internal/keyhelp/keyhelp.go`, `sections.go`; create `layer.go`, `layer_test.go`; append `sections_test.go`; `artifactpath/manifest.go` (`NonArtifactSources` += `cmd/internal/keyhelp/layer.go`).

- [ ] **Step 1: Failing tests**

`sections_test.go`, appended:

```go
// Rows documenting a workbench chord carry it, so a host overrides them by
// identity rather than by label (#282). Draft-local keys carry none.
func TestRowsCarryTheirWorkbenchChord(t *testing.T) {
	secs, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	byChord := map[workbenchshortcut.Chord]bool{}
	for _, s := range secs {
		for _, b := range s.Bindings {
			if b.Chord != 0 {
				byChord[b.Chord] = true
			}
			if b.Key == "Alt+⏎" && b.Chord != 0 {
				t.Errorf("draft-local Alt+⏎ carries chord %v", b.Chord)
			}
		}
	}
	for _, g := range workbenchshortcut.GlobalBindings() {
		if Catalog.Includes(g.NvimKey) && !byChord[g.Chord] {
			t.Errorf("global %s row lacks its chord", g.NvimKey)
		}
	}
	for _, r := range workbenchshortcut.RoleBindings() {
		if roleChordKey(r.Chord) != "" && !byChord[r.Chord] {
			t.Errorf("role %s row lacks its chord", workbenchshortcut.ChordName(r.Chord))
		}
	}
}

// Hosted wording replaces exactly the rows that declare it; everything else is
// byte-identical to standalone.
func TestHostedSectionsUseHostedWordingOnlyWhereDeclared(t *testing.T) {
	plain, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	hosted, err := HostedSections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	hostedHelp := map[workbenchshortcut.Chord]string{}
	for _, g := range workbenchshortcut.GlobalBindings() {
		hostedHelp[g.Chord] = g.HostedHelp
	}
	for i := range plain {
		for j, p := range plain[i].Bindings {
			h := hosted[i].Bindings[j]
			if want := hostedHelp[p.Chord]; p.Chord != 0 && want != "" {
				if !strings.HasPrefix(h.Desc, want) {
					t.Errorf("%s hosted desc %q, want %q", p.Key, h.Desc, want)
				}
			} else if h.Desc != p.Desc {
				t.Errorf("%s changed when hosted: %q → %q", p.Key, p.Desc, h.Desc)
			}
		}
	}
}
```

`layer_test.go`:

```go
package keyhelp

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func sec(title string, rows ...Binding) Section { return Section{Title: title, Bindings: rows} }

func TestLayerPutsHostFirstAndDropsClaimedChordsEverywhere(t *testing.T) {
	host := []Section{sec("Host", Binding{Key: "Alt+h", Desc: "host help", Chord: workbenchshortcut.ChordAltH, Context: ContextHost})}
	pair := []Section{
		sec("Session", Binding{Key: "Alt+h", Desc: "pair help", Chord: workbenchshortcut.ChordAltH}, Binding{Key: "Alt+l", Desc: "log", Chord: workbenchshortcut.ChordAltL}),
		sec("Only claimed", Binding{Key: "Alt+h", Desc: "pair help again", Chord: workbenchshortcut.ChordAltH}),
		sec("Draft", Binding{Key: "Alt+⏎", Desc: "send"}),
	}
	got := Layer(host, []workbenchshortcut.Chord{workbenchshortcut.ChordAltH}, pair)
	if len(got) != 3 || got[0].Title != "Host" || got[1].Title != "Session" || got[2].Title != "Draft" {
		t.Fatalf("sections = %+v", got)
	}
	for _, s := range got[1:] {
		for _, b := range s.Bindings {
			if b.Chord == workbenchshortcut.ChordAltH {
				t.Errorf("claimed chord survived in %q: %+v", s.Title, b)
			}
		}
	}
	if len(got[1].Bindings) != 1 || got[1].Bindings[0].Key != "Alt+l" {
		t.Errorf("unclaimed row lost: %+v", got[1])
	}
}

// A zero chord means "no identity", so a draft-local row can never be claimed.
func TestLayerNeverDropsChordlessRows(t *testing.T) {
	got := Layer(nil, []workbenchshortcut.Chord{0}, []Section{sec("Draft", Binding{Key: "Alt+⏎", Desc: "send"})})
	if len(got) != 1 || len(got[0].Bindings) != 1 {
		t.Fatalf("chordless row dropped: %+v", got)
	}
}

func TestLayerDoesNotMutateItsInputs(t *testing.T) {
	pair := []Section{sec("Session", Binding{Key: "Alt+h", Chord: workbenchshortcut.ChordAltH}, Binding{Key: "Alt+l", Chord: workbenchshortcut.ChordAltL})}
	Layer(nil, []workbenchshortcut.Chord{workbenchshortcut.ChordAltH}, pair)
	if len(pair[0].Bindings) != 2 || pair[0].Bindings[0].Key != "Alt+h" {
		t.Fatalf("input mutated: %+v", pair)
	}
}
```

- [ ] **Step 2: Run → FAIL** (build: `Binding.Chord`, `HostedSections`, `Layer`, `ContextHost`).
- [ ] **Step 3: Implement**

`keyhelp.go`:
- `Binding` gains `Chord workbenchshortcut.Chord`, with the comment "the workbench chord
  this row documents; zero for a draft-local key. A host overrides a row by this
  identity (Layer)". Add the import.
- The `Context` const block gains `ContextHost` ("a host takes it from every pane") and
  `ContextHostMenu` ("a host's own menu"), with `String()` values `"host"` and `"host
  menu"`.
- The `Section` doc comment gains "A section with no bindings is a notice: its title is
  the whole message."

`sections.go`:

```go
// Sections is Pair's help as Pair behaves standalone. HostedSections is the
// same document with each chord's HostedHelp where Couch hosting changes
// Pair's behavior (#282).
func Sections(src SourceReader) ([]Section, error)       { return sections(src, false) }
func HostedSections(src SourceReader) ([]Section, error) { return sections(src, true) }
```

Rename the current body to `sections(src SourceReader, hosted bool)`. In the
`GlobalBindings()` loop, use `help := b.Help; if hosted && b.HostedHelp != "" { help =
b.HostedHelp }` before the `(outside the agent pane)` suffix. Build `roleChord :=
map[string]workbenchshortcut.Chord{}` next to `roleHelp`. On each row, set `Chord` from
`globalBindings[e.Key].Chord` for `SourceGlobal` and from `roleChord[e.Key]` for
`SourceRole`.

`layer.go`:

```go
package keyhelp

import "github.com/xianxu/pair/cmd/internal/workbenchshortcut"

// Layer stacks a host's help above Pair's (#282). A chord the host claims never
// reaches Pair, so every Pair row documenting it is dropped -- in every
// context, since a host claims a chord from every pane. Other rows are
// untouched; a Pair section left empty is dropped. Pure and host-agnostic:
// Pair learns which chords it gives up, never who took them. Inputs are not
// modified.
func Layer(host []Section, claimed []workbenchshortcut.Chord, pair []Section) []Section {
	taken := map[workbenchshortcut.Chord]bool{}
	for _, c := range claimed {
		if c != 0 {
			taken[c] = true
		}
	}
	out := append([]Section(nil), host...)
	for _, s := range pair {
		kept := make([]Binding, 0, len(s.Bindings))
		for _, b := range s.Bindings {
			if b.Chord == 0 || !taken[b.Chord] {
				kept = append(kept, b)
			}
		}
		if len(kept) > 0 {
			out = append(out, Section{Title: s.Title, Bindings: kept})
		}
	}
	return out
}
```

- [ ] **Step 4: Run.** `go test ./cmd/internal/keyhelp/ ./cmd/internal/artifactpath/` → PASS. The existing keyhelp tests pass untouched.
- [ ] **Step 5: Commit.** `git add cmd/internal/keyhelp cmd/internal/artifactpath/manifest.go && git commit -m "#282: keyhelp: chord identity on rows, hosted wording, Layer"`

## Chunk 2: Couch's chord table as shared data

### Task 5: `couchkeys` package

**Files:** create `cmd/internal/couchkeys/couchkeys.go`, `couchkeys_test.go`; `artifactpath/manifest.go` (`NonArtifactSources` += `cmd/internal/couchkeys/couchkeys.go`).

- [ ] **Step 1: Failing tests**

```go
package couchkeys

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/keyhelp"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestEveryBindingIsComplete(t *testing.T) {
	for _, b := range Bindings() {
		if b.Action == 0 || b.Key == "" || b.Help == "" || len(b.Encodings) == 0 || (b.Scope != ScopeEveryPane && b.Scope != ScopeSwitcher) {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

// One byte sequence, one chord: the interceptor's first match would silently
// shadow a second.
func TestEncodingsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, b := range Bindings() {
		for _, e := range b.Encodings {
			if prior, ok := seen[string(e)]; ok {
				t.Errorf("%q is both %s and %s", e, prior, b.Key)
			}
			seen[string(e)] = b.Key
		}
	}
}

// A switcher chord's bytes are Pair's outside the switcher, so they are exactly
// Pair's encodings for the declared chord -- derived, not restated.
func TestSwitcherChordsCarryPairsEncodings(t *testing.T) {
	for _, b := range Bindings() {
		if b.Scope == ScopeSwitcher && (b.Chord == 0 || !reflect.DeepEqual(b.Encodings, workbenchshortcut.ChordEncodings(b.Chord))) {
			t.Errorf("%s encodings are not Pair's %v", b.Key, b.Chord)
		}
	}
}

// An every-pane chord whose bytes are a Pair chord takes that chord from Pair
// and must declare it, or Layer cannot drop Pair's row (#282).
func TestEveryPaneChordsDeclareThePairChordTheyTake(t *testing.T) {
	for _, b := range Bindings() {
		if b.Scope != ScopeEveryPane {
			continue
		}
		for _, e := range b.Encodings {
			if chord, ok := workbenchshortcut.DecodeChord(e); ok && chord != b.Chord {
				t.Errorf("%s takes Pair's %s without declaring it", b.Key, workbenchshortcut.ChordName(chord))
			}
		}
	}
}

// Couch leaves Alt+h to Pair: it is not a Couch chord in any scope (#282).
func TestCouchDoesNotClaimAltH(t *testing.T) {
	for _, b := range Bindings() {
		if b.Chord == workbenchshortcut.ChordAltH {
			t.Fatalf("%s claims Alt+h", b.Key)
		}
		for _, e := range b.Encodings {
			if chord, ok := workbenchshortcut.DecodeChord(e); ok && chord == workbenchshortcut.ChordAltH {
				t.Fatalf("%s frames Alt+h", b.Key)
			}
		}
	}
}

// `couch --help` renders Help verbatim and keeps internal operation names off
// its public surface (couchcmd TestPublicHelpListsOnlyPublicSurface).
func TestHelpAvoidsInternalOperationNames(t *testing.T) {
	for _, b := range Bindings() {
		for _, word := range []string{"start", "park", "resume"} {
			if strings.Contains(b.Help, word) {
				t.Errorf("%s help %q uses %q", b.Key, b.Help, word)
			}
		}
	}
}

func TestBindingsReturnsCopies(t *testing.T) {
	first := Bindings()
	first[0].Help = "mutated"
	first[0].Encodings[0][0] ^= 0xff
	second := Bindings()
	if second[0].Help == "mutated" || bytes.Equal(second[0].Encodings[0], first[0].Encodings[0]) {
		t.Fatal("Bindings leaked its table")
	}
}

func TestHelpSectionsRenderEveryChordByScope(t *testing.T) {
	secs := HelpSections(Bindings())
	if len(secs) != 2 {
		t.Fatalf("sections=%d, want every-pane + switcher", len(secs))
	}
	out := keyhelp.Render(secs)
	for _, b := range Bindings() {
		if !strings.Contains(out, b.Key) || !strings.Contains(out, b.Help) {
			t.Errorf("missing %s: %s", b.Key, b.Help)
		}
	}
	for _, r := range secs[0].Bindings {
		if r.Context != keyhelp.ContextHost {
			t.Errorf("%s in every-pane section: %v", r.Key, r.Context)
		}
	}
	for _, r := range secs[1].Bindings {
		if r.Context != keyhelp.ContextHostMenu {
			t.Errorf("%s in switcher section: %v", r.Key, r.Context)
		}
	}
	if !strings.Contains(secs[1].Title, "Ctrl+Space") {
		t.Errorf("switcher title %q does not name its opener", secs[1].Title)
	}
}

// Only every-pane chords that are Pair chords are claimed; switcher chords share
// Pair's bytes but never take them from a Pair pane.
func TestClaimedIsEveryPanePairChordsOnly(t *testing.T) {
	if got := Claimed(Bindings()); len(got) != 0 {
		t.Fatalf("today's table claims %v", got)
	}
	extra := Binding{Action: ActionSwitch, Scope: ScopeEveryPane, Chord: workbenchshortcut.ChordAltL, Key: "Alt+l", Help: "probe", Encodings: workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltL)}
	if got := Claimed(append(Bindings(), extra)); len(got) != 1 || got[0] != workbenchshortcut.ChordAltL {
		t.Fatalf("claimed = %v", got)
	}
}
```

- [ ] **Step 2: Run → FAIL** (`undefined: Bindings`).
- [ ] **Step 3: Implement** `couchkeys.go`. Move the comments on `hotkeyByte`,
`previousByte` and `newestPageSequence` from `couchtty/keys.go:17-39` onto the exported
constants verbatim, renaming the identifiers in the prose.

```go
// Package couchkeys declares Couch's keyboard contract: which chords Couch
// takes, where it acts on them, and what they mean (#282). It is data.
// couchtty's Interceptor and Console routing derive from it, `couch --help`
// renders it, and Pair's Alt+h page shows it when Couch presents the thread.
// A package of its own so Pair can describe Couch's keys without importing
// Couch's console.
package couchkeys

import (
	"github.com/xianxu/pair/cmd/internal/keyhelp"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

const SwitchLegacy = 0x00               // <moved comment>
const PreviousLegacy = 0x08             // <moved comment>
const NewestPageSequence = "\x1b[13;5u" // <moved comment>

// Scope is where Couch acts on a chord.
type Scope uint8

const (
	// ScopeEveryPane: Couch takes the chord from whichever Pair pane is
	// displayed; Pair never sees it.
	ScopeEveryPane Scope = iota + 1
	// ScopeSwitcher: Couch acts only while its switcher has focus. While a Pair
	// pane is displayed the same bytes reach Pair (#245), which has its own
	// meaning for them.
	ScopeSwitcher
)

// Action is what Couch does; couchtty maps each to its console dispatch.
type Action uint8

const (
	ActionSwitch Action = iota + 1
	ActionPrevious
	ActionNewestPage
	ActionDetach
	ActionPark
	ActionRelaunch
)

// Binding is one chord Couch declares. Chord is the Pair chord it shares (a
// switcher chord) or takes (an every-pane chord whose bytes are Pair's);
// zero for a Couch-only key.
type Binding struct {
	Action    Action
	Scope     Scope
	Chord     workbenchshortcut.Chord
	Key, Help string
	Encodings [][]byte
}

// Help wording avoids "start", "park" and "resume": `couch --help` renders it
// and keeps internal operation names off its public surface.
var bindings = []Binding{
	{Action: ActionSwitch, Scope: ScopeEveryPane, Key: "Ctrl+Space", Help: "open the Couch switcher",
		Encodings: [][]byte{{SwitchLegacy}, []byte("\x1b[32;5u")}},
	{Action: ActionPrevious, Scope: ScopeEveryPane, Key: "Ctrl+Backspace", Help: "return to the previous Couch thread",
		Encodings: [][]byte{{PreviousLegacy}, []byte("\x1b[127;5u")}},
	{Action: ActionNewestPage, Scope: ScopeEveryPane, Key: "Ctrl+Return", Help: "jump to the newest notification thread",
		Encodings: [][]byte{[]byte(NewestPageSequence), []byte("\x1b[13;5:1u"), []byte("\x1b[13;5:2u")}},
	switcher(ActionDetach, workbenchshortcut.ChordAltD, "Alt+d", "detach every live thread and leave Couch; their sessions keep running"),
	switcher(ActionPark, workbenchshortcut.ChordAltX, "Alt+x", "shut down every live thread and leave Couch (asks first)"),
	switcher(ActionRelaunch, workbenchshortcut.ChordAltN, "Alt+n", "relaunch the highlighted thread on the current binary, keeping its conversation"),
	switcher(ActionRelaunch, workbenchshortcut.ChordCtrlAltN, "Ctrl+Alt+n", "same as Alt+n"),
}

// switcher declares a switcher chord. Its bytes are Pair's for the same chord --
// outside the switcher they ARE Pair's key -- so they come from Pair's table.
func switcher(action Action, chord workbenchshortcut.Chord, key, help string) Binding {
	return Binding{Action: action, Scope: ScopeSwitcher, Chord: chord, Key: key, Help: help, Encodings: workbenchshortcut.ChordEncodings(chord)}
}

// Bindings returns every declared chord in help order, deep-copied.
func Bindings() []Binding {
	out := make([]Binding, len(bindings))
	for i, b := range bindings {
		b.Encodings = make([][]byte, len(bindings[i].Encodings))
		for n, e := range bindings[i].Encodings {
			b.Encodings[n] = append([]byte(nil), e...)
		}
		out[i] = b
	}
	return out
}

const (
	helpTitleEveryPane = "Couch — from every Pair pane (Couch takes these first)"
	helpTitleSwitcher  = "Couch switcher"
)

// HelpSections is Couch's layer of the key help: one section per scope, in
// table order, worded by the table and nothing else. `couch --help` and Pair's
// Alt+h page both render it (#282).
func HelpSections(bs []Binding) []keyhelp.Section {
	var pane, menu []keyhelp.Binding
	opener := ""
	for i, b := range bs {
		row := keyhelp.Binding{Key: b.Key, Desc: b.Help, Order: i, Chord: b.Chord}
		switch b.Scope {
		case ScopeEveryPane:
			row.Context, row.Group = keyhelp.ContextHost, helpTitleEveryPane
			pane = append(pane, row)
			if b.Action == ActionSwitch && opener == "" {
				opener = b.Key
			}
		case ScopeSwitcher:
			row.Context = keyhelp.ContextHostMenu
			menu = append(menu, row)
		}
	}
	title := helpTitleSwitcher
	if opener != "" {
		title += " (open it with " + opener + ")"
	}
	var out []keyhelp.Section
	if len(pane) > 0 {
		out = append(out, keyhelp.Section{Title: helpTitleEveryPane, Bindings: pane})
	}
	if len(menu) > 0 {
		for i := range menu {
			menu[i].Group = title
		}
		out = append(out, keyhelp.Section{Title: title, Bindings: menu})
	}
	return out
}

// Claimed is every Pair chord Couch takes from every pane: the rows Pair's
// layer gives up (keyhelp.Layer).
func Claimed(bs []Binding) []workbenchshortcut.Chord {
	var out []workbenchshortcut.Chord
	for _, b := range bs {
		if b.Scope == ScopeEveryPane && b.Chord != 0 {
			out = append(out, b.Chord)
		}
	}
	return out
}
```

Before finalising the switcher Help strings, check the current semantics:
`couchtty/console.go` `onDetachHotkey` and `onParkHotkey` (panel branch: `Operation:
"leave"`, `LeaveDetach`/`LeavePark`) and `onRelaunchHotkey` (panel:
`SelectedThreadAddress`). `pair#279` makes the Alt+d row true again; the help describes
the contract.

- [ ] **Step 4: Run.** `go test ./cmd/internal/couchkeys/ ./cmd/internal/artifactpath/` → PASS.
- [ ] **Step 5: Commit.** `git add cmd/internal/couchkeys cmd/internal/artifactpath/manifest.go && git commit -m "#282: couchkeys — Couch's chord table as data, with scope"`

### Task 6: couchtty frames and routes from couchkeys; `couch --help` renders it

Task 6 changes no behavior in framing or routing. couchcmd moves in the same task, so the
tree compiles at the end of it.

**Files:**
- Modify: `cmd/internal/couchtty/keys.go`:
  - lines 17-39 (constants go to couchkeys);
  - lines 61-84 (`hit`);
  - lines 118-159 (table and `actorReserved`);
  - lines 200-243 (`knownSequences`);
  - lines 305-315 (`FeedHit` single-byte loop).

  Its imports go from `workbenchshortcut` to `couchkeys`. `workbenchshortcut`'s only
  uses in `keys.go` were the lifecycle block, which moves.
- Modify: `cmd/internal/couchtty/console.go:1404` (`couchkeys.NewestPageSequence`)
- Modify tests: `keys_test.go` (replace `TestCouchNavigationReservationContract` at `:704`; constant renames), `console_newest_page_test.go`, `console_notice_expiry_test.go`
- Modify: `cmd/internal/couchcmd/run.go:795-823` (`usage` → `usageWith`), `shortcut_help_test.go`

- [ ] **Step 1: Failing tests**

`keys_test.go` (replacing the old contract test):

```go
// Every declared chord frames as its declared action, and Couch takes it from
// a Pair pane exactly when its scope says every pane (#282).
func TestCouchChordContract(t *testing.T) {
	for _, b := range couchkeys.Bindings() {
		want, _ := dispatchFor(b.Action)
		for _, encoding := range b.Encodings {
			var it Interceptor
			_, hit, _ := it.FeedHit(encoding)
			if hit != want {
				t.Errorf("%s %q framed as %v, want %v", b.Key, encoding, hit, want)
			}
			if hit.actorReserved() != (b.Scope == couchkeys.ScopeEveryPane) {
				t.Errorf("%s: actorReserved=%v, scope=%v", b.Key, hit.actorReserved(), b.Scope)
			}
		}
	}
	for _, encoding := range []string{"\x1b[3;5~", "\x1b[57349;5u", "\r"} {
		var it Interceptor
		if _, hit, _ := it.FeedHit([]byte(encoding)); hit != HitNone {
			t.Fatalf("invented alias %q", encoding)
		}
	}
}

// Alt+h reaches Pair from every pane: Couch neither frames nor routes it.
func TestAltHPassesThroughToPair(t *testing.T) {
	for _, encoding := range workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltH) {
		var it Interceptor
		before, hit, _ := it.FeedHit(encoding)
		if hit != HitNone || !bytes.Equal(before, encoding) {
			t.Errorf("Alt+h %q: hit=%v forwarded=%q", encoding, hit, before)
		}
	}
}

// A declared action with no dispatch arm would frame as HitNone and be
// forwarded -- the silent failure AllInterceptorHits guards. Conversely every
// console hit except the mouse must come from a declared chord.
func TestEveryCouchActionDispatches(t *testing.T) {
	reached := map[InterceptorHit]bool{}
	for _, b := range couchkeys.Bindings() {
		hit, kind := dispatchFor(b.Action)
		if hit == HitNone || kind == seqNone {
			t.Errorf("%s (action %d) has no dispatch", b.Key, b.Action)
		}
		reached[hit] = true
	}
	for _, hit := range AllInterceptorHits() {
		if hit != HitMouse && !reached[hit] {
			t.Errorf("hit %v has a handler but no declared chord", hit)
		}
	}
}

// Routing asks the hit, so two chords sharing a hit must share a scope.
func TestEachHitHasOneScope(t *testing.T) {
	scope := map[InterceptorHit]couchkeys.Scope{}
	for _, b := range couchkeys.Bindings() {
		hit, _ := dispatchFor(b.Action)
		if prior, ok := scope[hit]; ok && prior != b.Scope {
			t.Errorf("hit %v declared with scopes %v and %v", hit, prior, b.Scope)
		}
		scope[hit] = b.Scope
	}
}
```

Rename the constants in the three test files (idempotent; `\b` protects `seqHotkey`):

```bash
perl -pi -e 's/\bhotkeyByte\b/couchkeys.SwitchLegacy/g; s/\bpreviousByte\b/couchkeys.PreviousLegacy/g; s/\bnewestPageSequence\b/couchkeys.NewestPageSequence/g' \
  cmd/internal/couchtty/keys_test.go cmd/internal/couchtty/console_newest_page_test.go cmd/internal/couchtty/console_notice_expiry_test.go
```

Then add the `couchkeys` import to each of those three files.

`couchcmd/shortcut_help_test.go`: point `TestShortcutHelpDerivesReservations`'s loop at
`couchkeys.Bindings()` and append:

```go
// `couch --help` renders Couch's chords through couchkeys.HelpSections -- the
// function Pair's Alt+h page uses -- and a binding added to the table reaches
// it (#282).
func TestCouchHelpRendersEveryDeclaredChord(t *testing.T) {
	extra := couchkeys.Binding{Action: couchkeys.ActionSwitch, Scope: couchkeys.ScopeSwitcher, Key: "Ctrl+Alt+z", Help: "probe row for #282", Encodings: [][]byte{[]byte("\x1bz")}}
	var with, without, def bytes.Buffer
	usageWith(&with, append(couchkeys.Bindings(), extra))
	usageWith(&without, couchkeys.Bindings())
	usage(&def)
	if !strings.Contains(with.String(), extra.Help) || strings.Contains(without.String(), extra.Help) {
		t.Fatal("couch --help does not render from the passed table")
	}
	if def.String() != without.String() {
		t.Fatal("usage does not render couchkeys.Bindings()")
	}
}
```

- [ ] **Step 2: Run → FAIL** (`undefined: dispatchFor`, `usageWith`).
- [ ] **Step 3: Implement**

`keys.go`:
1. Delete the three constants and their comments. Leave a one-line pointer: `// Couch's
   chord table and its encodings live in couchkeys (#282).`
2. Add the table and the mapping:

```go
// couchChord joins one declared chord to the console's dispatch vocabulary.
type couchChord struct {
	couchkeys.Binding
	hit  InterceptorHit
	kind seqKind
}

var couchChords = func() []couchChord {
	declared := couchkeys.Bindings()
	out := make([]couchChord, 0, len(declared))
	for _, binding := range declared {
		hit, kind := dispatchFor(binding.Action)
		out = append(out, couchChord{Binding: binding, hit: hit, kind: kind})
	}
	return out
}()

// dispatchFor is the one mapping from a declared action to the console's hit
// and sequence kind. An action with no arm returns HitNone and would be
// forwarded to the child; TestEveryCouchActionDispatches makes that a failure.
func dispatchFor(action couchkeys.Action) (InterceptorHit, seqKind) {
	switch action {
	case couchkeys.ActionSwitch:
		return HitSwitch, seqSwitch
	case couchkeys.ActionPrevious:
		return HitPrevious, seqPrevious
	case couchkeys.ActionNewestPage:
		return HitNewestPage, seqNewestPage
	case couchkeys.ActionDetach:
		return HitDetach, seqDetach
	case couchkeys.ActionPark:
		return HitPark, seqPark
	case couchkeys.ActionRelaunch:
		return HitRelaunch, seqRelaunch
	}
	return HitNone, seqNone
}
```

3. Derive the rest from `couchChords`:
   - `hit()` loops `couchChords` for `kind == k` and returns its `hit`; drop the
     hardcoded park/detach/relaunch switch. Keep the doc comment and append "It reads
     couchChords, so a declared chord is recognised with no second edit."
   - `actorReserved()` returns the first matching row's `Scope ==
     couchkeys.ScopeEveryPane`, or false.
   - `knownSequences` keeps the paste markers, then one loop over `couchChords`
     appending each encoding with `len > 1` and its `kind`. Delete the anonymous
     lifecycle block. At the loop, add "Switcher chords are framed here too; Console
     forwards them while a Pair pane has focus (couchkeys ScopeSwitcher, #245)."
   - The `FeedHit` single-byte loop ranges over `couchChords` and returns `chord.hit`.
4. Delete `CouchNavigationBinding`, `couchNavigation` and `CouchNavigationBindings`.
5. The `Interceptor` doc comment says "The chord TABLE is not shared". Half of that is
   now stale: switcher encodings derive from Pair's table through couchkeys. Reword it:
   Couch's *declarations* live in couchkeys; the switcher chords borrow Pair's
   encodings; the every-pane chords are Couch's alone.
6. Run `gofmt -l cmd/internal` before committing. The new `couchkeys` import has to
   sort correctly in `console.go` and `run.go`.

`console.go:1404` becomes `DecodePanelKeys([]byte(couchkeys.NewestPageSequence))`, and
gains the import.

`couchcmd/run.go`:

```go
func usage(w io.Writer) { usageWith(w, couchkeys.Bindings()) }

func usageWith(w io.Writer, bindings []couchkeys.Binding) {
	// … lines 796-810 unchanged …
	fmt.Fprintln(w)
	// Couch's chords, laid out by the function Pair's Alt+h page uses (#282).
	fmt.Fprint(w, keyhelp.Render(couchkeys.HelpSections(bindings)))
	fmt.Fprintln(w, "\nThe agent also reserves these terminal-tab keys:")
	// … agent-reserved loop unchanged …
	fmt.Fprintln(w, "Other workbench keys reach the focused agent. Click another pane to leave it.")
}
```

Drop "While a Pair pane is displayed:" and "Use the switcher for Couch lifecycle
operations.". Imports gain `couchkeys` and `keyhelp`; `couchtty` stays for the console.

- [ ] **Step 4: Run.** `go vet ./cmd/internal/couchtty/ ./cmd/internal/couchcmd/ && go test ./cmd/internal/couchtty/ ./cmd/internal/couchcmd/ ./cmd/internal/couchkeys/` → PASS. Every existing framing and routing test passes unchanged: `TestRelaunchChordsAreInterceptedAndAltShiftNIsNot`, `TestActorLifecycleCandidatesPassThrough`, `TestConsoleRunAltDActorInputDoesNotDispatchDetach`, `TestInterceptorCandidateByteConservation`, `TestEveryInterceptedChordHasAHandler`, the newest-page tests, and `TestPublicHelpListsOnlyPublicSurface`. If `TestNotificationPTYConformance` or `TestCouchProductionSoak` fail under the sandbox (pty/exec), rerun them unsandboxed before judging.
- [ ] **Step 5: Commit.** `git add cmd/internal/couchtty cmd/internal/couchcmd && git commit -m "#282: couchtty frames and routes from couchkeys; couch --help renders it"`

## Chunk 3: Pair's page, title, docs

### Task 7: `pair keys` composes the page; pane title "help"

**Files:** `cmd/internal/keyscmd/keyscmd.go` (`Deps`, `RunWith`; delete `RunWithSources`); `keyscmd_test.go`; `cmd/internal/dispatcher/dispatcher_test.go:356`; `nvim/init.lua:3470`; `tests/workbench-route-nvim-test.sh:167` (pins the pane name).

- [ ] **Step 1: Failing tests** (`keyscmd_test.go`; migrate every existing `Run(args, …)` to `RunWith(args, deps(nil, false), …)` and `RunWithSources(nil, failingSources{}, …)` to `RunWith(nil, Deps{Sources: failingSources{}, Getenv: env(nil), CouchPresents: presents(false, nil)}, …)`)

```go
func env(vars map[string]string) func(string) string { return func(k string) string { return vars[k] } }

func presents(couch bool, err error) func() (bool, error) {
	return func() (bool, error) { return couch, err }
}

func deps(vars map[string]string, couch bool) Deps {
	return Deps{Sources: keyhelp.DefaultSources(), Getenv: env(vars), CouchPresents: presents(couch, nil)}
}

var couchLaunched = map[string]string{"COUCH_THREAD_SCOPE": "0123456789abcdef", "COUCH_THREAD_TAG": "couch-0123456789abcdef"}

func run(t *testing.T, d Deps) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := RunWith(nil, d, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	return stdout.String()
}

// Standalone Pair's Alt+h is unchanged, byte for byte (#282 Done-when 3).
func TestStandalonePageIsPairsSections(t *testing.T) {
	secs, err := keyhelp.Sections(keyhelp.DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	if got := run(t, deps(nil, false)); got != keyhelp.Render(secs) {
		t.Fatal("standalone page differs from Render(Sections)")
	}
}

// Presented by Couch: Couch's layer leads, from the table couch --help uses,
// and Pair's rows follow.
func TestCouchPresentedPageLeadsWithCouchsKeys(t *testing.T) {
	out := run(t, deps(couchLaunched, true))
	if !strings.HasPrefix(out, "Couch") {
		t.Fatalf("page does not lead with Couch:\n%s", out)
	}
	for _, b := range couchkeys.Bindings() {
		if !strings.Contains(out, b.Help) {
			t.Errorf("missing Couch's %s", b.Key)
		}
	}
	if !strings.Contains(out, "send buffer + clear") {
		t.Error("Pair's layer is missing")
	}
}

// The two facts are independent. Hosted wording follows Pair's own refusal
// rule (env); Couch's section follows who presents the client (record).
func TestPresenterAndHostingAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		name          string
		vars          map[string]string
		couch         bool
		wantCouch     bool
		wantHostedRow bool
	}{
		// Presented by Couch means the client refuses a restart marker, so
		// Alt+n does not reload even though the session env is not Couch's.
		{"adopted: Couch presents, env not Couch's", nil, true, true, true},
		{"Couch gone, reattached from a terminal", couchLaunched, false, false, true},
		{"Couch-launched and presented", couchLaunched, true, true, true},
		{"standalone", nil, false, false, false},
	} {
		out := run(t, deps(tc.vars, tc.couch))
		if got := strings.Contains(out, "open the Couch switcher"); got != tc.wantCouch {
			t.Errorf("%s: couch section=%v", tc.name, got)
		}
		if got := strings.Contains(out, "does not reload under Couch"); got != tc.wantHostedRow {
			t.Errorf("%s: hosted wording=%v", tc.name, got)
		}
	}
}

// A record the reader cannot trust never conjures Couch's section; Pair's page
// still renders, and the cause goes to stderr.
func TestUnreadablePresenterRendersPairsPage(t *testing.T) {
	d := deps(nil, false)
	d.CouchPresents = presents(true, errors.New("outer-tty record: unexpected shape"))
	var stdout, stderr bytes.Buffer
	if code := RunWith(nil, d, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(stdout.String(), "open the Couch switcher") || !strings.Contains(stdout.String(), "send buffer + clear") {
		t.Fatalf("page:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unexpected shape") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

// Argument errors return before reading the record.
func TestBadArgumentsNeverReadThePresenter(t *testing.T) {
	d := deps(nil, false)
	d.CouchPresents = func() (bool, error) { t.Fatal("read the record on a usage error"); return false, nil }
	var stdout, stderr bytes.Buffer
	RunWith([]string{"--bogus"}, d, &stdout, &stderr)
	RunWith([]string{"--help"}, d, &stdout, &stderr)
}

// The production wiring: Run reads the record the launcher writes, from the
// session's PAIR_DATA_DIR and PAIR_TAG (#282).
func TestRunReadsTheAttachRecord(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COUCH_THREAD_SCOPE", "")
	t.Setenv("COUCH_THREAD_TAG", "")
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")
	paths, err := artifactpath.ResolveScoped(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	record := launcher.EncodeOuterRecord(launcher.OuterRecord{TTY: "/dev/ttys006", Couch: true})
	if err := os.WriteFile(paths.OuterTTY(), []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("exit %d stderr %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "open the Couch switcher") {
		t.Fatalf("Run ignored the record:\n%s", stdout.String())
	}
}

// `pair --help` still advertises `pair keys`; outside a session it prints
// Pair's page and nothing on stderr.
func TestRunOutsideASessionIsQuiet(t *testing.T) {
	for _, k := range []string{"COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "PAIR_DATA_DIR", "PAIR_TAG"} {
		t.Setenv(k, "")
	}
	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("exit %d stderr %q", code, stderr.String())
	}
	secs, err := keyhelp.Sections(keyhelp.DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != keyhelp.Render(secs) {
		t.Fatal("outside a session, pair keys is not Pair's standalone page")
	}
}
```

Test imports gain `couchkeys`, `keyhelp`, `launcher`, `artifactpath`, `errors` and `os`.
If artifactpath's classifier objects to a `_test.go` resolving the outer-tty path, move
`TestRunReadsTheAttachRecord` into launcher and keep keyscmd's wiring check to the
env-empty case.

- [ ] **Step 2: Run → FAIL** (`undefined: RunWith`, `Deps`).
- [ ] **Step 3: Implement** `keyscmd.go`:

```go
// Deps is every seam `pair keys` reads through, so tests stay hermetic even
// when the repo is tested inside a Couch thread.
type Deps struct {
	Sources keyhelp.SourceReader
	Getenv  func(string) string
	// CouchPresents reports whether the client attached to this session was
	// launched by Couch -- read from the attach record, not the session env,
	// which names whoever created the session (#282).
	CouchPresents func() (bool, error)
}

func Run(args []string, stdout, stderr io.Writer) int {
	return RunWith(args, Deps{
		Sources: keyhelp.DefaultSources(),
		Getenv:  os.Getenv,
		CouchPresents: func() (bool, error) {
			dataDir, tag := os.Getenv("PAIR_DATA_DIR"), os.Getenv("PAIR_TAG")
			if dataDir == "" || tag == "" {
				return false, nil // outside a Pair session: nothing presents it
			}
			return launcher.ReadOuterPresenter(dataDir, tag)
		},
	}, stdout, stderr)
}
```

In `RunWith`, the argument loop is unchanged, and so is the always-exit-0 comment
(moved onto `Run`). Then:

```go
	couch, perr := deps.CouchPresents()
	if perr != nil {
		// An untrusted record never conjures Couch's section.
		_, _ = fmt.Fprintf(stderr, "pair keys: %v\n", perr)
		couch = false
	}
	build := keyhelp.Sections
	if couch || launcher.CouchHostedEnv(deps.Getenv) {
		// Pair's Alt+n does not reload when either the session env names
		// Couch (pair restart refuses) or Couch launched this client (it
		// refuses the restart marker). Hosted wording is true of the key.
		build = keyhelp.HostedSections
	}
	sections, err := build(deps.Sources)
	if err != nil { /* unchanged diagnostic path */ }
	if couch {
		bs := couchkeys.Bindings()
		sections = keyhelp.Layer(couchkeys.HelpSections(bs), couchkeys.Claimed(bs), sections)
	}
```

The imports gain `os`, `launcher` and `couchkeys`. These are new package edges for
keyscmd. There is no cycle: launcher and couchkeys do not import keyscmd, and couchkeys
imports only keyhelp and workbenchshortcut. Confirm with `go build ./...`.

`dispatcher_test.go` `TestDispatchKeysReturnsKeybindings`: first lines
`t.Setenv("COUCH_THREAD_SCOPE", ""); t.Setenv("COUCH_THREAD_TAG", "");
t.Setenv("PAIR_DATA_DIR", ""); t.Setenv("PAIR_TAG", "")`, with the comment "hermetic:
this repo is tested inside Couch threads".

`nvim/init.lua:3470`: `pair_open_workbench_view('help', '70%', '15%', { 'pair-help'
})`. The pane is opened before the page is built, so the title cannot depend on the
host; the page's first section names the layer. `tests/workbench-route-nvim-test.sh:167`
pins the old name in `want_views`: change `--name pair help` to `--name help`. `make
test` runs it (`Makefile.local:255`).

- [ ] **Step 4: Run.** The `init.lua` edit makes keyhelp's `TestEmbeddedSourcesMatchTree` report a stale bundle until it is regenerated, so rebuild first:

```bash
make build > "$TMPDIR/282-build.log" 2>&1 || exit 1
go test ./cmd/internal/keyscmd/ ./cmd/internal/dispatcher/ ./cmd/internal/keyhelp/ ./cmd/internal/launcher/ || exit 1
bash tests/workbench-route-nvim-test.sh || exit 1
```

Expected: all pass. `TestPairHelpShimInvokesPairKeys` may fail under the sandbox (exec); if so, rerun it unsandboxed.
- [ ] **Step 5: Commit.** `git add cmd/internal/keyscmd cmd/internal/dispatcher/dispatcher_test.go nvim/init.lua tests/workbench-route-nvim-test.sh && git commit -m "#282: Alt+h shows Couch's keys over Pair's when Couch presents the client"`

### Task 8: docs, guards, full verification

**Files:** `README.md`; `atlas/architecture.md:299-309`; `atlas/couch.md` (`:482`, `:515`, `:648-657`, `:733-734`); `cmd/internal/artifactpath/deletedvocabulary_test.go`; test `cmd/internal/couchcmd/readme_test.go`.

- [ ] **Step 1: README guard** (`readme_test.go`)

```go
// Every chord Couch declares has an operator-facing home in README's couch
// section. README spells the navigation chords Ctrl-Space style; accept that
// spelling of a Ctrl+ label.
func TestREADMEDocumentsEveryCouchChord(t *testing.T) {
	section := couchREADMESection(t)
	for _, b := range couchkeys.Bindings() {
		if !strings.Contains(section, b.Key) && !strings.Contains(section, strings.Replace(b.Key, "Ctrl+", "Ctrl-", 1)) {
			t.Errorf("README's couch section does not document %s (%s)", b.Key, b.Help)
		}
	}
}
```

All seven keys were present when this was planned. Run the test; if it fails, add
the missing mention.

- [ ] **Step 2: Prose.** Find each site by its content. The line numbers below
  are from planning time, and the earlier edits in this step shift the later ones.
  - README Pair keys table:
    - Alt+h row (118): "…Same content as `pair keys`. When Couch presents the thread,
      Couch's keys come first."
    - Alt+d row (151): append "In a thread Couch launched, this detaches only the
      Zellij client; Couch's own detach is in the switcher."
    - Alt+n row (153): replace the trailing bold sentence with "**In a thread Couch
      launched, Pair's own Alt+n is refused; relaunch from the Couch switcher (Alt+n
      there).**"
  - README Couch section:
    - lines 378–381: replace with "While a Pair pane is displayed, Alt+d reaches Pair,
      which detaches only its Zellij client; Couch's detach is the switcher's (#245)."
      Drop "Detaching an actor moves focus to the switcher…": that actor branch is
      unreachable from the keyboard since #245.
    - lines 496–497: correct "the existing Pair reload in draft/right panes…" to say
      that Pair refuses an inner reload in a Couch-launched thread (#249) and the
      switcher's Alt+n is the route.
    - At the navigation chords (425–455), add: "Alt+h in the draft shows Couch's keys
      above Pair's; Couch does not take Alt+h, so the agent pane still receives it."
  - `atlas/architecture.md` Keybind help paragraph: replace the
    `couchtty.CouchNavigationBindings` sentence with the two-fact model:
    - Presenter: `OuterRecord` written per attach by `PresentedByCouch`, read by
      `ReadOuterPresenter`.
    - Hosting: `CouchHostedEnv` (the `pair restart` refusal) or the presenter
      record (the client's marker refusal) selects `HostedSections`.
    - `couchkeys` is the chord table read by couchtty framing and routing, `couch
      --help` and the page.
    - `keyhelp.Layer` drops claimed chords by typed `Chord`.
    - The pane is titled "help".
  - `atlas/couch.md`:
    - `:482`/`:515`: rename `hotkeyByte`/`newestPageSequence` to
      `couchkeys.SwitchLegacy`/`couchkeys.NewestPageSequence`.
    - `:648-657`: name the couchkeys table and `Scope` as what routing reads.
    - `:733-734`: replace "other panes retain Pair's existing in-process reload"
      with the #249 refusal plus the switcher route.
  - `atlas/architecture.md:1140` describes the outer-tty record's contents ("records
    the launcher controlling TTY for compatibility"). Add the presenter line: who
    writes it (`PresentedByCouch`, per attach), who reads it (`pair keys`), and that
    a non-tty attach removes the record.
  - Where the atlas describes hosted wording, state the predicate as "Couch
    launched the session (`CouchHostedEnv`) or presents this client (the record)".
    Name both Alt+n gates: `pair restart` and the client's marker refusal.
  - `deletedvocabulary_test.go`: add a `CouchNavigationBinding` entry following the
    file's pattern.

- [ ] **Step 3: Full verification.** Run the full `make test` before close, and `go test ./...` as well, because `make -k test` can skip Go tests. Redirect to files: `| head` produces SIGPIPE phantom failures.

```bash
make test > "$TMPDIR/282-make-test.log" 2>&1; echo "make test exit=$?"; tail -30 "$TMPDIR/282-make-test.log"
go test ./... > "$TMPDIR/282-go-test.log" 2>&1; echo "go test exit=$?"; grep -E '^(FAIL|---)' "$TMPDIR/282-go-test.log" | head -20
```

Expected: both exit 0. Sandbox-only failures (pty/exec: `TestNotificationPTYConformance`,
`TestCouchProductionSoak`, `TestPairHelpShimInvokesPairKeys`) are rerun unsandboxed, and
both runs are recorded.

- [ ] **Step 4: Behavior evidence** (record in the issue Log)

```bash
# Baseline: origin/main's standalone page. A fresh worktree lacks the
# gitignored runtime bundle, so generate it; `make` can't run there (the
# Makefile is a symlink into ../ariadne). Confirm generatecmd's flags with -h.
git worktree add "$TMPDIR/main282" origin/main || exit 1
(cd "$TMPDIR/main282" && go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime) || exit 1
(cd "$TMPDIR/main282" && env -u COUCH_THREAD_SCOPE -u COUCH_THREAD_TAG PAIR_DATA_DIR="$TMPDIR/none" go run ./cmd/pair-go keys) > "$TMPDIR/keys-main.txt" || exit 1
git worktree remove --force "$TMPDIR/main282"
make build > "$TMPDIR/282-build.log" 2>&1 || exit 1   # confirm the binary paths it produces
env -u COUCH_THREAD_SCOPE -u COUCH_THREAD_TAG PAIR_DATA_DIR="$TMPDIR/none" ./bin/pair keys > "$TMPDIR/keys-standalone.txt"
diff "$TMPDIR/keys-main.txt" "$TMPDIR/keys-standalone.txt" && echo STANDALONE-UNCHANGED
time ./bin/pair keys > /dev/null
./bin/couch --help
```

In this session the record predates the change: it holds one line, so it reads "not
presented". Couch's section therefore appears in a live run only after Couch relaunches
the thread on the new binary. That happens in the operator smoke.

- [ ] **Step 5: Operator smoke** (ask; don't claim without it). Rebuild, restart Couch, and relaunch or reattach a thread so its client writes the new record.
  - Alt+h in the draft: the pane is titled "help". Couch's sections come first, then
    Pair's with the hosted Alt+d/Alt+n wording.
  - Alt+h in the right terminal: the same page.
  - Alt+h in the agent pane: reaches the agent, no help.
  - Pair's own Alt+d in a Couch-displayed Pair pane. Observe what Couch shows
    afterwards, and confirm the README 378–381 rewrite and the Alt+d `HostedHelp`
    ("detach only this Zellij client…") describe it. Record the observation.
  - In a standalone `pair` session, Alt+h: Pair's page unchanged, titled "help".

- [ ] **Step 6: Commit.** `git add README.md atlas cmd/internal/couchcmd/readme_test.go cmd/internal/artifactpath/deletedvocabulary_test.go && git commit -m "#282: README + atlas: Alt+h under Couch, current Alt+d/Alt+n routing"`

## Done-when mapping (the issue's third revision)

| Done-when | Evidence |
|---|---|
| When Couch presents the thread, Alt+h (draft; right terminal via Pair's routing) shows Couch's keys above Pair's, from the table `couch --help` renders; a binding added reaches both | `TestCouchPresentedPageLeadsWithCouchsKeys`, `TestHelpSectionsRenderEveryChordByScope`, `TestCouchHelpRendersEveryDeclaredChord`, smoke |
| A chord Couch claims replaces Pair's row; no key has two meanings in one context; Pair's rows that hosting changes show hosted wording when Couch launched the session or presents the client | `TestLayerPutsHostFirstAndDropsClaimedChordsEverywhere`, `TestClaimedIsEveryPanePairChordsOnly`, `TestEveryPaneChordsDeclareThePairChordTheyTake`, `TestHostedSectionsUseHostedWordingOnlyWhereDeclared`, `TestPresenterAndHostingAreIndependent`, `TestCouchClientRefusesRestartMarker` |
| Standalone Pair's Alt+h unchanged (title aside) | `TestStandalonePageIsPairsSections` + main-vs-branch `diff`, smoke |
| Couch's keys appear iff the attached client was launched by Couch for this thread, including adopted sessions and not after Couch exits | `TestPresentedByCouchRequiresTheThreadsTag`, `TestReadOuterPresenter`, `TestLegacyOuterRecordIsNotPresented`, `TestMalformedOuterRecordIsAnError`, `TestPresenterAndHostingAreIndependent`, launcher fake-runtime assertions |
| Couch does not take Alt+h; the agent pane receives it | `TestCouchDoesNotClaimAltH`, `TestAltHPassesThroughToPair`, smoke |

## Revisions

### 2026-09-18: two operator redirects before approval

**First draft (replaced):** Pair *detected* Couch. It read *hosted* from the Zellij
server's `COUCH_*` env and *live* from the supervisor lease. The operator objected that
the env records who created the session, so a session Couch adopts later (`#246`) would
read as standalone.

**Second draft (replaced):** Couch intercepted Alt+h from every pane and showed Couch's
keys over Pair's rows in a new Couch panel frame. The operator objected that Couch must
not take keys from panes it cannot tell apart. The agent pane and the right pane keep
their keys; Alt+h triggers from the draft.

**This plan:** Alt+h stays Pair's; Pair's pager shows the combined page (operator's
choice over a Couch panel reached by a new Pair→Couch channel); Pair's routing from the
right terminal is kept (operator's choice). *Presenter* comes from the attached client's
own env, recorded per attach. *Hosting* comes from Pair's refusal rule (amended by the third review: or Couch presents). `pair keys` from
a shell is not a supported surface. The pane title becomes "help".

- Kept from earlier drafts:
  - `CouchHosted`;
  - `HostedHelp`;
  - `couchkeys` and couchtty's derivation;
  - `keyhelp.Binding.Chord`/`Layer`/`HostedSections`;
  - `couch --help` rendering;
  - the doc sweep.
- Dropped:
  - the lease probe;
  - `Presence`/`Page`;
  - Couch's Alt+h interception;
  - `MenuFrameHelp` and PgUp/PgDn.
- The first plan review's findings still apply to the kept parts and are folded in:
  - couchcmd compiling at every task;
  - the `keys.go` import swap;
  - the main-baseline bundle generation;
  - the wider doc sweep;
  - keyed literals;
  - `git add` in every commit.
- Third plan review (this design): Issues Found. The reviewer built all 8 tasks
  in a scratch worktree, and standalone output was byte-identical. Folded in:
  - The pane-name pin in `tests/workbench-route-nvim-test.sh:167`.
  - The hosted-wording predicate became `CouchHostedEnv || couchPresents`.
    Alt+n is gated twice: an adopted Couch-presented thread passes
    `pair restart`, but its client refuses the marker after the quit cleanup,
    so the thread ends without relaunching. The earlier "reload pair is correct
    for adopted sessions" claim was wrong. The destructive case is recorded in
    `pair#284`, and the client refusal gets a test.
  - `pair keys` stays quiet outside a session.
  - Tests for the non-tty removal and for `Run`'s production wiring.
  - Bundle regeneration before targeted tests.
  - The atlas outer-tty description, the stale `Interceptor` doc, and gofmt.

### 2026-09-18: plan-quality advisories folded in (change-code round 1)

- The record parser gets `FuzzDecodeOuterRecord`, not only the literal table.
- The ARCH-SECURE claim is corrected: strictness gives visible degradation on
  malformed input. A well-formed hand edit is a deliberate act.
- Task 3's comment names Alt+Shift+C's hosted change and why no row needs it.
- The doc sweep is anchored by content, not by line number.

### 2026-09-18: implemented, closed

All 8 tasks landed as `dbc489ba`..`9d0e257c`. The step checkboxes above are
the plan as written; the issue's `## Plan` carries the ticks. The close review
said SHIP, with five advisories; the dispositions are in the issue Log. One
changed the plan's wording. The Alt+d/Alt+n `HostedHelp` strings in Task 3 were
shortened ("detach only this Zellij client, not the Couch thread"; "does not
reload under Couch; relaunch from the Couch switcher"), because the 124-col
hosted row un-centred the page. `TestNoLayerWidensThePage` now pins the width
rule.
