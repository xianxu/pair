---
gate: plan-quality
issue: 447
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-11T13:21:58-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Console ignores the session's empty OnNotes after its own pointer-off flip, so stale notes reappear when pointing is turned back on
          detail: The console flips its pointer phase off before calling s.DisablePointer (console_pointer.go:76). The resulting OnNotes(empty) arrives while the phase is off, and the plan says to ignore it, so pointerNotes keeps the old list. Only resetPointer clears it, at broadcast end. Clear pointerNotes on the console's pointer-off transition (click and watch), and test that re-enabling shows no notes and the count is 0.
          family: pointer-off-clears-notes-ordering
          round: 1
        - id: PQ-2
          severity: Important
          title: acceptNote checks pointing-on under PointerState.mu, then adds under NoteState.mu, so a concurrent DisablePointer clear can run between them and the note survives
          detail: 'Interleaving: check on → DisablePointer flips and clears an empty board → add lands. The note then exists while pointing is off, goes out to every stream, and survives re-enable. Make the add conditional on the pointer generation it observed (the offIfGeneration pattern in pointer.go), or clear on EnablePointer too. Add an injected-ordering test.'
          family: pointer-off-clears-notes-ordering
          round: 1
        - id: PQ-3
          severity: Minor
          title: readPointerBody's guard order is stated twice, once without the rate check
          detail: The second statement matches server.go:135-160; drop the first.
          family: plan-self-consistency
          round: 1
        - id: PQ-4
          severity: Minor
          title: Test steps enumerate cases in prose instead of one adversarial strategy line per risky function
          detail: For ParseNote, fuzz seeded with the refusal corpus and assert no echo. For DrawNotes, a property check (order independence, no orphan wide-char half) over random wide rows and note sets.
          family: test-strategy-not-enumeration
          round: 1
        - id: PQ-5
          severity: Minor
          title: Note text check refuses control characters but allows format characters (bidi overrides, zero-width), so the dialog can show spoofed text
          family: untrusted-text-charset
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T13:23:45-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Console resyncs snapshot+version from s.Notes() on every pointer-phase transition; re-enable test added.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Board moved under p.mu so add and clear share one critical section; the fix's lock-order defect is raised separately.
          round: 2
        - id: PQ-3
          disposition: addressed
          round: 2
        - id: PQ-4
          disposition: addressed
          note: FuzzParseNote seeded with the refusal corpus, and a DrawNotes property test, now in the plan.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Rune scan refuses unicode.Cf.
          round: 2
      findings:
        - id: PQ-6
          severity: Critical
          title: addNote runs lands under p.mu, but Hub.Current is a hub-loop round-trip and the hub loop takes p.mu in OnPointerHidden, so the two deadlock
          detail: 'The plan claims lands "only reads the hub''s current frame (its own leaf lock)". That is wrong: Hub.Current goes through h.do and waits on the hub loop (hub.go:231-241, 461-462). The loop calls OnPointerHidden (hub.go:277-283), which calls s.pointer.generation() and takes p.mu (session.go:97-99, pointer.go:104-107). An add racing the pointer grace timer therefore freezes the hub loop. Separately, landsNow is described as "the checks acceptPoint makes", which include s.pointer.On() (session.go:386); passed as lands, it relocks p.mu and deadlocks on every add. Rule: nothing called while p.mu is held may take p.mu or wait on another goroutine. Fix: read gen, then run the frame checks with no lock held, then call addNote(n, gen), which refuses unless p.on && p.gen == gen (the offIfGeneration pattern). Keep the On check out of landsNow. Add a test that fires OnPointerHidden on a real hub during an add.'
          family: lock-held-across-foreign-call
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T13:24:35-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: addressed
          note: gen read first, frame check unlocked, addNote(n, gen) refuses on off/flip in-lock; landsNow drops On(); lock rule in pointer.go; hub-loop deadlock test in Task 5.
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-10-11T13:25:33-07:00"
      agent: claude
      findings:
        - id: PQ-7
          severity: Minor
          title: Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
          detail: '2nd finding in this family. Rule: a task step names what to build and points to its Core concepts entry for how, without restating the mechanism. Follow Core concepts (set content " " and width 1, keep the style) and drop the uv.EmptyCell wording in Task 3.'
          family: plan-self-consistency
          round: 4
      blocked: false
content_hash: 8fd700d06749f01a5f825c172ca5b44925811d1c453b691f070df62d6067d717
---

# Gate ledger — pair#447 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T13:21:58-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `pointer-off-clears-notes-ordering` Console ignores the session's empty OnNotes after its own pointer-off flip, so stale notes reappear when pointing is turned back on
  The console flips its pointer phase off before calling s.DisablePointer (console_pointer.go:76). The resulting OnNotes(empty) arrives while the phase is off, and the plan says to ignore it, so pointerNotes keeps the old list. Only resetPointer clears it, at broadcast end. Clear pointerNotes on the console's pointer-off transition (click and watch), and test that re-enabling shows no notes and the count is 0.
- **PQ-2** [Important] `pointer-off-clears-notes-ordering` acceptNote checks pointing-on under PointerState.mu, then adds under NoteState.mu, so a concurrent DisablePointer clear can run between them and the note survives
  Interleaving: check on → DisablePointer flips and clears an empty board → add lands. The note then exists while pointing is off, goes out to every stream, and survives re-enable. Make the add conditional on the pointer generation it observed (the offIfGeneration pattern in pointer.go), or clear on EnablePointer too. Add an injected-ordering test.
- **PQ-3** [Minor] `plan-self-consistency` readPointerBody's guard order is stated twice, once without the rate check
  The second statement matches server.go:135-160; drop the first.
- **PQ-4** [Minor] `test-strategy-not-enumeration` Test steps enumerate cases in prose instead of one adversarial strategy line per risky function
  For ParseNote, fuzz seeded with the refusal corpus and assert no echo. For DrawNotes, a property check (order independence, no orphan wide-char half) over random wide rows and note sets.
- **PQ-5** [Minor] `untrusted-text-charset` Note text check refuses control characters but allows format characters (bidi overrides, zero-width), so the dialog can show spoofed text

## Round 2 — 2026-10-11T13:23:45-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Console resyncs snapshot+version from s.Notes() on every pointer-phase transition; re-enable test added.
- PQ-2 — addressed — Board moved under p.mu so add and clear share one critical section; the fix's lock-order defect is raised separately.
- PQ-3 — addressed
- PQ-4 — addressed — FuzzParseNote seeded with the refusal corpus, and a DrawNotes property test, now in the plan.
- PQ-5 — addressed — Rune scan refuses unicode.Cf.

### Raised

- **PQ-6** [Critical] `lock-held-across-foreign-call` addNote runs lands under p.mu, but Hub.Current is a hub-loop round-trip and the hub loop takes p.mu in OnPointerHidden, so the two deadlock
  The plan claims lands "only reads the hub's current frame (its own leaf lock)". That is wrong: Hub.Current goes through h.do and waits on the hub loop (hub.go:231-241, 461-462). The loop calls OnPointerHidden (hub.go:277-283), which calls s.pointer.generation() and takes p.mu (session.go:97-99, pointer.go:104-107). An add racing the pointer grace timer therefore freezes the hub loop. Separately, landsNow is described as "the checks acceptPoint makes", which include s.pointer.On() (session.go:386); passed as lands, it relocks p.mu and deadlocks on every add. Rule: nothing called while p.mu is held may take p.mu or wait on another goroutine. Fix: read gen, then run the frame checks with no lock held, then call addNote(n, gen), which refuses unless p.on && p.gen == gen (the offIfGeneration pattern). Keep the On check out of landsNow. Add a test that fires OnPointerHidden on a real hub during an add.

## Round 3 — 2026-10-11T13:24:35-07:00 (claude) — passed

### Disposed

- PQ-6 — addressed — gen read first, frame check unlocked, addNote(n, gen) refuses on off/flip in-lock; landsNow drops On(); lock rule in pointer.go; hub-loop deadlock test in Task 5.

## Round 4 — 2026-10-11T13:25:33-07:00 (claude) — passed

### Raised

- **PQ-7** [Minor] `plan-self-consistency` Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
  2nd finding in this family. Rule: a task step names what to build and points to its Core concepts entry for how, without restating the mechanism. Follow Core concepts (set content " " and width 1, keep the style) and drop the uv.EmptyCell wording in Task 3.

## Open findings

- **PQ-7** [Minor] `plan-self-consistency` Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
