# #728 M3 operator check — about five minutes

The hub schedules this after #753 clears its stage and #819 finishes its look.
The worker stages/builds the selected head through `sdlc stage --build` when
authorized, reports the verified binary revisions, and the hub arranges a fresh
Couch runtime using that build. Existing running processes keep their old code.
Start the timer once that setup is ready.

Approved targets: **echo:1**, Codex, idle on `main-slot1`; optionally **echo:2**,
Claude, with echo#5 durable and nothing in flight. The hub rechecks their idle
state at check time. echo:0 and echo:3 are outside this check.

1. **Baseline (30 seconds).** The operator runs `couch --show echo:1` and
   `couch --peek echo:1 --json --transcripts`. Retain the thread address,
   recorded helper PID and `native_session` value, plus the visible conversation.

2. **Form and Codex seeding (one minute).** On echo:1, open **Switch coding
   agent**, select Codex and edit the parameters to
   `--sandbox danger-full-access`. Confirm **Restart** in **Restart with new
   parameters**. Expect the same conversation/history with the new permission
   setting, a new helper PID, and no fresh-conversation notice. Reopen the form:
   the saved parameters must remain exactly those accepted. Cancel the form.
   This operator acceptance seeds Couch's Echo repository configuration.

3. **CLI reuse (one minute).** From the hub, the operator runs:

   ```sh
   couch --switch-agent echo:1 --agent codex --confirm --json
   ```

   Inspect the returned operation ID with `couch --operation-status ID --json`,
   replacing `ID` with that exact value. Expect succeeded, the same Codex native
   conversation and saved `--sandbox danger-full-access` parameters. Recheck
   `couch --peek echo:1 --json --transcripts` and the form prefill. The CLI
   receives no explicit parameters here: it exercises the configuration just
   saved through the form.

4. **Force (one minute).** With echo:1 idle, open its same-agent form and choose
   **Force switch**. Cancel once: the current child and conversation stay put.
   Open it again and confirm the single unsaved-turn-progress warning once.
   Expect the old owned child to end, one replacement to become ready, the same
   native conversation and accepted parameters, and the reported on-disk edits
   to remain intact. Leave the agent idle.

5. **Optional Claude seeding (one minute).** If echo:2 is still idle, its
   operator opens the same-agent form, sets
   `--dangerously-skip-permissions` and confirms **Restart**. Expect the same
   Claude conversation/history and the accepted permission setting. Reopen the
   form to confirm the exact saved value, then cancel. This seeds Claude's
   separate Echo configuration without replacing Codex's value.

At five minutes, stop the check and report any pending/unknown operation by its
exact receipt ID; do not submit the same effect again. The operator reports
form / CLI / force outcomes, native IDs before/after, whether the accepted flags
persisted, and any warning. The worker records that result before whole-issue
close and landing; the hub coordinates clearing the stage afterward.

The missing-conversation fallback already passed the disposable native test for
both agents, including the exact notice and preserved old transcript bytes.
That test uses fixture agents; this check supplies the real-agent permission
flags and operator interaction evidence.
