---
gate: plan-quality
issue: 128
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-07-30T16:52:49-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Task 4's premise is wrong — csiEnd is on the fallback path, not the query-literal path
          detail: |-
            The plan claims csiEnd/oscEnd are reached only after a terminalQueryLiterals
            prefix match, so inputs are well-formed. The literal loop RETURNS on match
            (queries.go:104, :107); csiEnd at queries.go:115 sits in the switch at :113,
            reached only when no literal matched — i.e. it frames arbitrary app output in
            the replay buffer. Adopting the strict regex semantics therefore breaks the
            queries.go:126 two-byte catch-all for ESC 7 / ESC 8 / ESC = / ESC >, which the
            regex's `\x1b[@-Z\\-_]` class excludes; SequenceLen 0 makes stripTerminalQueries
            bail at queries.go:81-85 and dump the rest of the replay unstripped, restoring
            the bug pair#127 fixed. Task 4 Step 1's per-literal test cannot catch this.
          round: 1
        - id: PQ-2
          severity: Critical
          title: csiEnd's other two call sites in rename_input.go are undeclared; the -1 to 0 sentinel swap is a zero-advance loop
          detail: |-
            Task 4 lists only queries.go, but csiEnd also feeds escapeSequenceIncomplete
            (rename_input.go:193) and malformedEscapeSize (rename_input.go:208) on the tab-rename
            input path — csiEnd's own doc comment at queries.go:132-135 says so, and the
            Done-when demands csiEnd be gone from production code. malformedEscapeSize does
            `if end := csiEnd(input); end >= 0 { return end }`; with SequenceLen returning 0
            that returns 0, and the decoder at rename_input.go:117-120 re-slices by 0 and
            loops forever. escapeSequenceIncomplete would misclassify malformed-but-complete
            CSI as incomplete and pin it in state.Pending.
          round: 1
        - id: PQ-3
          severity: Important
          title: Unstated seam decision — SequenceLen collapses three distinct return meanings into one int
          detail: |-
            The existing code keeps "unterminated tail" (csiEnd -1), "not a sequence I frame"
            (regex no-match) and "unknown two-byte escape, consume 2" (queries.go:126) apart,
            and all three consumers branch on the difference. The plan picks a single int with
            0 meaning all three without saying so. State the contract explicitly and how each
            consumer preserves its current behavior under it.
          round: 1
        - id: PQ-4
          severity: Minor
          title: oscRe at wrap.go:191 is a fifth OSC framing left unmentioned
          detail: |-
            The plan retires otherEscRe from the same var block but says nothing about oscRe,
            which frames OSC identically while also capturing. It may not be worth extracting,
            but declare it out of scope rather than leaving it silent (ARCH-DRY).
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-07-30T16:58:17-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Plan now states the fallback-arm reading correctly and keeps termcmd on Lenient, so queries.go:126 is untouched.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Both rename_input.go callers named; sentinels preserved via adapters and pinned by a never-returns-zero property test.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Status{None,Incomplete,Complete} plus a per-consumer mapping table.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: oscRe declared out of scope with the capturing-vs-framing reason.
          round: 2
      findings:
        - id: PQ-5
          severity: Important
          title: The csiEnd adapter changes SS3 framing — csiEnd is a final-byte scan, not a dispatch on buf[1]
          detail: |-
            csiEnd (queries.go:136-143) ignores buf[1] and scans from index 2, and malformedEscapeSize
            (rename_input.go:205-210) calls it with input[1] == 'O' on purpose, getting 3 for "\x1bOX".
            Frame(buf, Lenient) dispatches on buf[1]; 'O' is 0x4F, inside Task 1 Step 3's two-byte class
            0x40-0x5A, so the adapter returns 2 and the decoder inserts the X into the tab name.
            TestDecodeRenameInputConsumesUnknownEscapeTerminators (rename_input_test.go:150-169) covers
            "\x1bOX" and "\x1bO@" and will fail, contradicting Task 4 Step 3's "identical pass set by
            construction". State the resolution — an introducer-independent CSI scan in Frame, or leave
            csiEnd's scan in place and adapt only oscEnd.
          round: 2
        - id: PQ-6
          severity: Minor
          title: Wrong pin literal for plain CSI, and the verbatim test bodies should compress to strategy lines
          detail: |-
            Task 4 Step 1 expects csiEnd("\x1b[31m") == 4; the sequence is five bytes and csiEnd returns
            i+1 == 5. The enumerated case tables in Task 1 Steps 1/2a and Task 4 Step 1 will be rewritten
            as code immediately; the differential-fuzz strategy line and the never-returns-zero property
            already carry the intent.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-07-30T17:01:36-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: addressed
          note: Resolved by exporting TerminatorScan as an introducer-independent scan; verified csiEnd ignores buf[1] (queries.go:136-143) and malformedEscapeSize routes SS3 into it (rename_input.go:205-208).
          round: 3
        - id: PQ-6
          disposition: not-addressed
          note: Literal corrected to 5; the enumerated case tables in Task 1 Steps 1/2a and Task 4 Step 1 remain. Minor, non-blocking.
          round: 3
      findings:
        - id: PQ-7
          severity: Minor
          title: Stale Frame(Lenient) text survives the PQ-5 fix in the consumer table, Done-when, and commit message
          detail: Task 4 Step 2 now calls ansi.TerminatorScan / ansi.OSCEnd(Lenient), but the "Consumer mapping" table, the Done-when bullet, and Task 4 Step 5's commit message still say termcmd frames through ansi.Frame(Lenient). Frame's Lenient CSI arm then has no production consumer, and an implementer reading Done-when as the acceptance criterion could restore the buf[1] dispatch PQ-5 rejected.
          round: 3
        - id: PQ-8
          severity: Minor
          title: isTerminalFinalByte lives in rename_input.go:214, not queries.go, contradicting Task 4's "rename_input.go is not modified"
          detail: The entity table places isTerminalFinalByte in cmd/internal/termcmd/queries.go and marks it deleted in favour of ansi.IsFinalByte, but it is defined at rename_input.go:214 (queries.go:138 only calls it). Deleting it means editing a file Task 4 declares untouched; leaving it means dead code, since Go does not error on an unused package-level function.
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-07-30T17:06:31-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: not-addressed
          note: Pin literal fixed (csiEnd("\x1b[31m") now 5); the three verbatim enumerated case tables remain, and two of their literals are wrong (see new finding).
          round: 4
        - id: PQ-7
          disposition: addressed
          note: Consumer table, Done-when and Task 4 Step 5's commit message all name TerminatorScan/OSCEnd now, and the Lenient CSI arm's no-consumer status is stated; the header paragraph's "adapters over it (Lenient)" is loose but unambiguous given the entity table.
          round: 4
        - id: PQ-8
          disposition: addressed
          note: Entity table and Task 4 Files both place isTerminalFinalByte at rename_input.go:214 (verified) and scope the edit to exactly that line.
          round: 4
      findings:
        - id: PQ-9
          severity: Important
          title: Strict mode is not regex-equivalent for a failed OSC — otherEscRe matches "\x1b]" as a two-byte escape, so two stated pins are wrong and Incomplete-vs-regex conflicts
          detail: otherEscRe's fourth alternative `\x1b[@-Z\\-_]` covers 0x40-0x5A plus the range 0x5C-0x5F, and `]` is 0x5D — so when the OSC alternative fails, the regex still matches 2 bytes at offset 0 (wrap.go:189). Task 1 Step 1's "unterminated OSC" and "OSC with bare ESC" pins say 0 but are 2 today, and Step 2a's Strict expectation of None should be Complete/2. CSI is unaffected only because `[` is 0x5B, the byte the class skips. State whether Strict's OSC arm falls through to the two-byte class (regex-identical, satisfying the Done-when) or whether Incomplete-preserving semantics win as a declared behavior change, with its effect at wrap.go:1018's agentPending carry path spelled out.
          round: 4
      blocked: false
content_hash: 536e7aa12090bf023212ab6dcf5727736383ed014b10177917bbca5d01a4d64b
---

# Gate ledger — pair#128 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-07-30T16:52:49-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] Task 4's premise is wrong — csiEnd is on the fallback path, not the query-literal path
  The plan claims csiEnd/oscEnd are reached only after a terminalQueryLiterals
  prefix match, so inputs are well-formed. The literal loop RETURNS on match
  (queries.go:104, :107); csiEnd at queries.go:115 sits in the switch at :113,
  reached only when no literal matched — i.e. it frames arbitrary app output in
  the replay buffer. Adopting the strict regex semantics therefore breaks the
  queries.go:126 two-byte catch-all for ESC 7 / ESC 8 / ESC = / ESC >, which the
  regex's `\x1b[@-Z\\-_]` class excludes; SequenceLen 0 makes stripTerminalQueries
  bail at queries.go:81-85 and dump the rest of the replay unstripped, restoring
  the bug pair#127 fixed. Task 4 Step 1's per-literal test cannot catch this.
- **PQ-2** [Critical] csiEnd's other two call sites in rename_input.go are undeclared; the -1 to 0 sentinel swap is a zero-advance loop
  Task 4 lists only queries.go, but csiEnd also feeds escapeSequenceIncomplete
  (rename_input.go:193) and malformedEscapeSize (rename_input.go:208) on the tab-rename
  input path — csiEnd's own doc comment at queries.go:132-135 says so, and the
  Done-when demands csiEnd be gone from production code. malformedEscapeSize does
  `if end := csiEnd(input); end >= 0 { return end }`; with SequenceLen returning 0
  that returns 0, and the decoder at rename_input.go:117-120 re-slices by 0 and
  loops forever. escapeSequenceIncomplete would misclassify malformed-but-complete
  CSI as incomplete and pin it in state.Pending.
- **PQ-3** [Important] Unstated seam decision — SequenceLen collapses three distinct return meanings into one int
  The existing code keeps "unterminated tail" (csiEnd -1), "not a sequence I frame"
  (regex no-match) and "unknown two-byte escape, consume 2" (queries.go:126) apart,
  and all three consumers branch on the difference. The plan picks a single int with
  0 meaning all three without saying so. State the contract explicitly and how each
  consumer preserves its current behavior under it.
- **PQ-4** [Minor] oscRe at wrap.go:191 is a fifth OSC framing left unmentioned
  The plan retires otherEscRe from the same var block but says nothing about oscRe,
  which frames OSC identically while also capturing. It may not be worth extracting,
  but declare it out of scope rather than leaving it silent (ARCH-DRY).

## Round 2 — 2026-07-30T16:58:17-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Plan now states the fallback-arm reading correctly and keeps termcmd on Lenient, so queries.go:126 is untouched.
- PQ-2 — addressed — Both rename_input.go callers named; sentinels preserved via adapters and pinned by a never-returns-zero property test.
- PQ-3 — addressed — Status{None,Incomplete,Complete} plus a per-consumer mapping table.
- PQ-4 — addressed — oscRe declared out of scope with the capturing-vs-framing reason.

### Raised

- **PQ-5** [Important] The csiEnd adapter changes SS3 framing — csiEnd is a final-byte scan, not a dispatch on buf[1]
  csiEnd (queries.go:136-143) ignores buf[1] and scans from index 2, and malformedEscapeSize
  (rename_input.go:205-210) calls it with input[1] == 'O' on purpose, getting 3 for "\x1bOX".
  Frame(buf, Lenient) dispatches on buf[1]; 'O' is 0x4F, inside Task 1 Step 3's two-byte class
  0x40-0x5A, so the adapter returns 2 and the decoder inserts the X into the tab name.
  TestDecodeRenameInputConsumesUnknownEscapeTerminators (rename_input_test.go:150-169) covers
  "\x1bOX" and "\x1bO@" and will fail, contradicting Task 4 Step 3's "identical pass set by
  construction". State the resolution — an introducer-independent CSI scan in Frame, or leave
  csiEnd's scan in place and adapt only oscEnd.
- **PQ-6** [Minor] Wrong pin literal for plain CSI, and the verbatim test bodies should compress to strategy lines
  Task 4 Step 1 expects csiEnd("\x1b[31m") == 4; the sequence is five bytes and csiEnd returns
  i+1 == 5. The enumerated case tables in Task 1 Steps 1/2a and Task 4 Step 1 will be rewritten
  as code immediately; the differential-fuzz strategy line and the never-returns-zero property
  already carry the intent.

## Round 3 — 2026-07-30T17:01:36-07:00 (claude) — passed

### Disposed

- PQ-5 — addressed — Resolved by exporting TerminatorScan as an introducer-independent scan; verified csiEnd ignores buf[1] (queries.go:136-143) and malformedEscapeSize routes SS3 into it (rename_input.go:205-208).
- PQ-6 — not-addressed — Literal corrected to 5; the enumerated case tables in Task 1 Steps 1/2a and Task 4 Step 1 remain. Minor, non-blocking.

### Raised

- **PQ-7** [Minor] Stale Frame(Lenient) text survives the PQ-5 fix in the consumer table, Done-when, and commit message
  Task 4 Step 2 now calls ansi.TerminatorScan / ansi.OSCEnd(Lenient), but the "Consumer mapping" table, the Done-when bullet, and Task 4 Step 5's commit message still say termcmd frames through ansi.Frame(Lenient). Frame's Lenient CSI arm then has no production consumer, and an implementer reading Done-when as the acceptance criterion could restore the buf[1] dispatch PQ-5 rejected.
- **PQ-8** [Minor] isTerminalFinalByte lives in rename_input.go:214, not queries.go, contradicting Task 4's "rename_input.go is not modified"
  The entity table places isTerminalFinalByte in cmd/internal/termcmd/queries.go and marks it deleted in favour of ansi.IsFinalByte, but it is defined at rename_input.go:214 (queries.go:138 only calls it). Deleting it means editing a file Task 4 declares untouched; leaving it means dead code, since Go does not error on an unused package-level function.

## Round 4 — 2026-07-30T17:06:31-07:00 (claude) — passed

### Disposed

- PQ-6 — not-addressed — Pin literal fixed (csiEnd("\x1b[31m") now 5); the three verbatim enumerated case tables remain, and two of their literals are wrong (see new finding).
- PQ-7 — addressed — Consumer table, Done-when and Task 4 Step 5's commit message all name TerminatorScan/OSCEnd now, and the Lenient CSI arm's no-consumer status is stated; the header paragraph's "adapters over it (Lenient)" is loose but unambiguous given the entity table.
- PQ-8 — addressed — Entity table and Task 4 Files both place isTerminalFinalByte at rename_input.go:214 (verified) and scope the edit to exactly that line.

### Raised

- **PQ-9** [Important] Strict mode is not regex-equivalent for a failed OSC — otherEscRe matches "\x1b]" as a two-byte escape, so two stated pins are wrong and Incomplete-vs-regex conflicts
  otherEscRe's fourth alternative `\x1b[@-Z\\-_]` covers 0x40-0x5A plus the range 0x5C-0x5F, and `]` is 0x5D — so when the OSC alternative fails, the regex still matches 2 bytes at offset 0 (wrap.go:189). Task 1 Step 1's "unterminated OSC" and "OSC with bare ESC" pins say 0 but are 2 today, and Step 2a's Strict expectation of None should be Complete/2. CSI is unaffected only because `[` is 0x5B, the byte the class skips. State whether Strict's OSC arm falls through to the two-byte class (regex-identical, satisfying the Done-when) or whether Incomplete-preserving semantics win as a declared behavior change, with its effect at wrap.go:1018's agentPending carry path spelled out.

## Open findings

- **PQ-6** [Minor] Wrong pin literal for plain CSI, and the verbatim test bodies should compress to strategy lines
- **PQ-9** [Important] Strict mode is not regex-equivalent for a failed OSC — otherEscRe matches "\x1b]" as a two-byte escape, so two stated pins are wrong and Incomplete-vs-regex conflicts
