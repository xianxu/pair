---
id: 000270
status: open
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# Draft hit-enter prompt has no trigger, no recovery

## Problem

Operator report with screenshot (2026-09-16): the draft nvim pane is sitting at

    Press ENTER or type command to continue

with the draft text intact above it and the custom statusline replaced by the
prompt. Every keystroke queues behind that prompt, so `Alt+Return` — and Alt+q,
Alt+h, and ordinary typing — all appear dead. **The draft cannot submit.**

It is intermittent: "this doesn't always happen, but not sure how I get into
this bad state." The session that reported it was *already* in the state when
noticed, and the only known way out is restarting the session (park/resume, or
Alt+n) — which is also the thing that destroys the evidence.

**#190 owns the prevention half and already diagnosed the mechanism**:
`cmdheight = 0` (`nvim/init.lua:247`) leaves no command line for a message to
land in, so anything that does not fit forces nvim's hit-enter prompt; its fix
is one non-blocking `pair_notify` seam replacing the `vim.notify` sites. What
#190 could *not* do is name the trigger — "which one actually fired cannot be
recovered — nvim's message history died with the process" — and it says nothing
about recovery or detection. This issue is those three: **which path fires, how
the operator gets out without restarting, and how pair notices at all.**

Two things have changed since #190 was written (2026-09-05):

- The `vim.notify` count in `nvim/init.lua` is now **31**, up from the 22 that
  issue counted. The class is growing, not shrinking.
- #190 assumes the trigger is a `vim.notify`. **It may not be.** An nvim *error*
  — a failing autocmd or callback, `E5108` and friends — also forces the prompt
  under `cmdheight=0`, and a notify seam would not touch that path. If that is
  what is happening here, #190 can land in full and this symptom survives it.

Constraints found while filing, which bound the possible designs:

- **The draft nvim is not remotely addressable.** Nothing passes `--listen` or
  calls `serverstart`, so there is no socket to run `:messages` through or to
  send a dismissing `CR` to. Keys can reach the pane only the way the draft send
  path already reaches it — `zellij action write-chars` / `send-keys` at the pane
  id — which is therefore the only transport a recovery gesture can use.
- **A recovery chord cannot be handled by the stuck nvim**, since that is what is
  blocked. It has to be handled by something above it (a zellij/couch-level
  binding, or a pair-side command), the way couch already intercepts its six
  chords ahead of the pane.
- `vim.opt.more` is not set anywhere in `nvim/init.lua`.

**Ordering note worth acting on:** the trigger can only be identified from a
*live* stuck pane. If #190 lands first and the trigger was a notify site, the
state stops happening and the question becomes unanswerable — which is fine for
the operator and bad for knowing whether the error path (above) was ever
involved. Capture the evidence before #190 ships, not after.

## Spec

- **Name the trigger from evidence, not inference.** A stuck pane still holds
  its message history: dismissing the prompt with one Enter and running
  `:messages` (plus `:echo v:errmsg`) names what fired. That is a one-minute
  operator step that nothing can reconstruct afterwards. Anything less and this
  issue re-derives #190's hypothesis and stops.
- **Make the next occurrence self-documenting.** The draft should record what it
  was about to say, and from where, to a durable place (the pair log is already
  wired and read by `:PairDoctor`), so identifying a future instance does not
  depend on someone noticing in time. This is the part that makes the class
  tractable rather than anecdotal.
- **Detection before recovery.** Settle whether nvim exposes the state
  in-process (`mode()` / `state()`, a UI message event, `vim.ui_attach`) from
  its documented behavior rather than assumption. If no reliable in-process
  signal exists, the detector lives outside the pane, where the prompt text is
  visible on the pane's screen.
- **Recovery must not be able to fire blind.** The dismissing keystroke is a
  bare `CR`, and a bare `CR` sent to a *healthy* draft inserts a newline or
  submits the draft. So the gesture is gated on the detected state and refuses
  otherwise — an unconditional "send Enter to the draft" button would trade this
  bug for a worse one.
- **Recovery is not a restart.** Park/resume works today and costs the operator
  their session; the point of this issue is that it should not be the answer.
- **Prevention stays with #190 unless the evidence moves it.** If the trigger is
  a notify site, #190's seam is the fix and this issue keeps trigger, detection
  and recovery. If an error path can reach the prompt, prevention for *that*
  needs an owner — `vim.o.more` / `shortmess`, or an error-capturing wrapper —
  and #190's spec does not cover it.

## Done when

- The trigger for at least one real occurrence is named in the Log, with its
  message text, recovered from a stuck pane rather than inferred.
- Whether an nvim *error* (not just a notification) can produce the prompt under
  `cmdheight=0` is settled by a headless test, and if it can, prevention for it
  has a named owner.
- A stuck draft is recoverable without restarting the session, by a gesture that
  is proved both ways: it clears the prompt when the pane is stuck, and it
  refuses (or no-ops) when the pane is healthy — no draft text mutated, nothing
  submitted.
- pair records a blocking message when it happens, with enough state to identify
  the path, so a later occurrence does not need a live pane.
- A test drives the real blocking condition and asserts the draft still accepts
  input afterwards. `tests/run-headless.sh` already bounds a hung nvim (the #60
  timeout harness), which is the shape this needs.

## Estimate

```estimate
# pending: derived after the plan clears change-code's plan-quality gate
```

## Plan

- [ ] **Operator step, before the next restart of a stuck draft:** press Enter in
      the draft pane, run `:messages` and `:echo v:errmsg`, and paste both into
      the Log. Unreconstructable later; everything else here can wait.
- [ ] Determine whether an nvim error reaches the hit-enter prompt under
      `cmdheight=0`, headless, either way — it decides whether #190's notify seam
      is sufficient prevention.
- [ ] Settle the detection signal (in-process nvim state vs. reading the pane's
      screen from outside), citing nvim's documented behavior, not a guess.
- [ ] Record the blocking message + originating path durably before it blocks.
- [ ] Recovery gesture, handled outside the stuck nvim, gated on the detected
      state; regression-test that it clears a stuck pane and refuses a healthy
      one.
- [ ] Reconcile with #190: confirm the seam covers the observed trigger, or name
      what else must change.

## Log

### 2026-09-16

- Filed from an operator report + screenshot: the draft pane at `Press ENTER or
  type command to continue`, draft text intact, `Alt+Return` unable to submit.
  The reporting session was itself in the state; the operator's workaround is
  restarting it. **Task only — not started, at their request.**
- Checked while filing, so the next session does not re-derive it:
  `nvim/init.lua:247` sets `cmdheight = 0`; `vim.notify` appears **31** times in
  `nvim/init.lua` (22 when #190 was written); `vim.opt.more` is unset; no
  `--listen`/`serverstart` anywhere for the draft nvim, so there is no socket for
  remote diagnosis or recovery, and keys can only reach the pane through the
  `zellij action write-chars`/`send-keys` transport the draft send already uses.
- Relationship to #190 recorded in Problem rather than as a `deps:` entry:
  neither blocks the other, but they must not be worked blind to each other —
  #190 prevents the notify class, this one identifies the actual trigger, adds a
  non-restart recovery, and covers the error path #190's seam would miss.
