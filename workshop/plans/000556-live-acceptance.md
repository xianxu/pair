# #556 M1 acceptance: prepared disk recovery

Run headless checks in the issue checkout. The live portion is for the owning hub
and operator after their scheduled restart. No stage, restart, or working-slot
mutation is part of this script. M2's concurrent-writer barrier is separate.

## Headless evidence

These stateful fixtures allocate private temporary stores and fake process/agent
boundaries; they never change a real slot, native transcript, or Zellij session.

```sh
go test ./cmd/internal/couchcore -run 'TestProvenRecovery|TestPrepared|TestResumeCandidate|TestRecoveryFresh|TestRecoveryStartFence|TestRecoveryLaunch|TestRecoveryFinal|TestRecoveryConfirmation|TestPeekTagged|TestTaggedRecovery' -count=1 -timeout=120s
go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/couchmessage -run 'TestRecovery|TestResumeCandidates|TestMenuRecovery|TestHubMutation|TestSlotOperation' -count=1 -timeout=90s
go test ./cmd/internal/launcher -run 'TestRecovery|TestConfirmedRecovery' -count=1 -timeout=90s
```

The fixtures cover proven/no-screen native resume (empty Prompt), an explicitly
selected ranked key including alternatives, missing/incomplete screen evidence,
Fresh orientation, exact-stop confirmation/cancellation/replay/drift, foreign
survivors, expired claim fencing, late registration and unconfirmed launch. They
also check no signal or spawn before preservation/confirmation, unchanged native
files/defaults, and shared menu/CLI admission. Tagged prior-screen readers reject
wrong roots, token substitutions and changed physical files.

## Short operator/hub check

1. Restart Couch onto the accepted build using the operator's normal procedure.
   Existing live rows remain Continue, auto-attach and message. Do not use any
   working slot for the remaining steps. In the normal new-slot form allocate a
   **new disposable slot**, with the operator's normal saved agent configuration.
   Substitute that exact address for `pair:N` below. Record `--show` first and
   verify it is the disposable slot.

```sh
couch --show pair:N
couch --resume-candidates pair:N --json
```

2. In the scratch agent finish one short turn containing a distinctive marker,
   for example `#556 scratch acceptance`. Use the slot menu's ordinary Park action
   to stop it cleanly. Inspect the candidate JSON and its proven native ID, then
   request resume. Inspect the operation receipt ID printed by each action:

```sh
couch --resume-candidates pair:N --json
couch --resume pair:N --json
couch --operation-status RECEIPT --json
couch --peek pair:N --lines 30
```

   Expected: the same conversation/marker; no fresh orientation prompt. Candidate
   inference never runs automatically. If alternatives exist, their printed opaque
   keys are selected explicitly with the following command; no numeric guess or
   hand-edited record is needed. Rare ambiguous/no-screen cases are exercised by
   the private headless fixtures above rather than corrupting this live slot.

```sh
couch --resume pair:N --candidate EXACT_PRINTED_KEY --json
```

3. On this scratch slot only, request Fresh. If running owners need to stop, the
   result names their exact PID, start identity and role and supplies a token.
   Read that stop list. Cancel once by taking no action: it must keep running.
   Repeat the Fresh request to get a current offer; the owning hub can confirm it
   directly without navigating the menu:

```sh
couch --reboot pair:N --confirm --json
couch --recovery-confirmation EXACT_PRINTED_TOKEN --json
couch --operation-status RECEIPT --json
couch --peek pair:N --previous --all
```

   Expected: one Fresh agent on a new tag; the old transcript remains on disk.
   Its first message points at the previous-screen command, branch/issue and
   retained capture coordinates, rather than pasting screen bytes. Previous peek
   names/reads the original screen. With absent screen evidence, the agent gets
   branch/issue plus an honest unavailable notice. Native resume uses no Prompt.

4. An already consumed token cannot stop/start again; a same-request CLI receipt
   replay returns its existing receipt. An unused token expires after two minutes
   or a Couch restart. Do not restart Couch just to test expiration. The headless
   fake-clock tests cover it deterministically. Leave the scratch slot to the
   operator's usual cleanup; do not kill a process discovered by name alone.

Record the build hash, scratch address, receipt IDs, selected native ID and whether
previous peek and first-message orientation matched. Report any failed/unconfirmed
launch with its exact attempt and diagnostics; do not automatically reboot again.
