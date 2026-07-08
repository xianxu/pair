---
status: active
type: pensive
created: 2026-07-07
---

# pair frame title shows no token count — ctxmeter resolved empty (SCROLL is a red herring)

**Symptom.** A `claude` agent pane's zellij frame reads `claude [~/workspace/brain]` — the base title
with **no `(count)`** — while the right of the frame shows `SCROLL: 0/2005`.

**The SCROLL indicator is unrelated.** `SCROLL: N/M` is zellij's own scroll-mode frame indicator (the
pane is scrolled back), not a pair string — grep for `SCROLL` in pair finds only `PAIR_SCROLLBACK_*`.
Red herring; ignore it.

**The real cause: the count resolved empty.** `titlepoller.frameTitle(agent, count, cwd)` renders
`"<agent> (<count>) [<cwd>]"` only when `count != ""`, else falls back to `"<agent> [<cwd>]"`
(`cmd/internal/titlepoller/titlepoller.go:72`). The observed title is exactly the **no-count branch**,
so `count == ""`.

**The count path (all traced this session):**
`updateFrameTitles` (`titlepoller/run.go:189`) → `rt.ContextCount(tag, agent)` →
`OSRuntime.ContextCount` (`titlepoller/runtime.go:94`) invokes the exact `pair context` code path
in-process: `contextcmd.Run([tag, agent], EnvFromOS(), &buf)`; empty buffer ⇒ empty count →
`contextcmd.TranscriptPath` resolves the claude jsonl → `ctxmeter.ContextTokens` (`ctxmeter.go:17`)
reads the **last** record that is `type:"assistant"`, **not** `isSidechain`, **not** model
`"<synthetic>"`, and sums `input_tokens + cache_creation_input_tokens + cache_read_input_tokens`;
returns `found=false` (→ empty) when nothing qualifies.

**So `count==""` ⇒ one of:**
1. **`TranscriptPath` didn't resolve/locate this session's transcript** — the (tag, agent) → claude
   `~/.claude/projects/<munged-cwd>/<session-id>.jsonl` mapping missed (wrong munge, missing
   `config-<tag>-<agent>.json`, session-id mismatch). **Most likely.**
2. **ctxmeter found no qualifying record** — the transcript's last `assistant` record was filtered
   (`<synthetic>` model, or `isSidechain:true`). Less likely for a busy session, but possible if the
   tail records are synthetic/sidechain.
3. **Frame meter activity-gated off** (`age < 2*opts.PollInterval`, `run.go:138`) — but that only
   *stops refreshing*; it can't blank an already-set title. Only relevant if the count was **never**
   set since session start (→ collapses to cause 1/2).

Not model-specific: the claude parser matches any non-`<synthetic>` model, so `claude-opus-4-8[1m]`
(the 1M-context variant) is fine on its own — but *confirm* the transcript path munging isn't thrown
by the model/session variant.

**NEXT STEP (concrete).** Get the pair tag for this session, then run **`pair context <tag> claude`**
directly. If it prints empty, the bug is in `contextcmd` — read
`cmd/internal/contextcmd/contextcmd.go` (`TranscriptPath` + `Run`): verify it locates the right claude
jsonl for this session and that ctxmeter isn't filtering the tail. If `pair context` prints a number
but the frame doesn't show it, the bug is downstream (activity gate / rename-pane / the frameCache).

Not yet checked: the pair **tag** (needed to run `pair context`); the `contextcmd` resolver itself
(the read that would confirm cause 1). No code changed — read-only investigation. No issue filed yet.
