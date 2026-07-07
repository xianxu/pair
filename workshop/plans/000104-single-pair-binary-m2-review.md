# Boundary Review — pair#104 (milestone M2)

| field | value |
|-------|-------|
| issue | 104 — Fold pair repo binaries into a single pair Go program |
| repo | pair |
| issue file | workshop/issues/000104-single-pair-binary.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c00f6a7ce9282e8e264db8670553813375c7b856..HEAD |
| command | sdlc milestone-close --issue 104 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-06T18:35:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Everything compiles and vets clean. I have a complete picture. Here's my review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 caller rewrites are correct and complete for the milestone's purpose: every in-repo consumer now derives from `pair <sub>` — launcher self-exec (`title`/`session-watch`), clipcmd self-exec (`clip <leaf>`), distiller (`scrollback render`/`changelog render`), nvim (`review open|readiness`, `scrollback open`), and zellij (`copy_command`, `Run`, `exec pair wrap`). I verified this three ways: all changed-package Go tests pass, the nvim-driven shell tests (`review-toggle`, `review-window`, `scrollback-open`, `changelog-open`) and `copy-on-select-test.sh` pass, and a freshly-built `bin/pair` dispatches every M2-caller subcommand form to the right `RunXxxCLI` with no launcher misroute. The blocking coupling the plan flagged (title-poller single-instance guard) was correctly co-updated in the same change. Nothing blocks the boundary. The one thing to reconcile before closing is a plan-vs-delivery divergence: Task 2.6 explicitly directed rewriting the `.claude/settings*.json` allowlist entries, which were consciously left as-is (documented in the atlas, but the plan still claims them).

### 1. Strengths

- **Title-poller guard co-landed correctly** (`cmd/internal/titlepoller/titlepoller.go:100`). `pollerArgvMatches` now matches `"pair title "+tag+" "`, and the trailing space still defeats the 21-vs-211 prefix collision — the exact regression the plan (Task 2.2 / Risks) warned would spawn duplicate pollers. `TestPollerArgvMatches` keeps the collision case and adds a "pre-fold pair-title binary is not the running poller" case. This is the highest-risk part of M2 and it's handled + tested.
- **ARCH-PURE held cleanly.** `sessionWatcherArgv`/`titlePollerArgv` (`osruntime.go:302,306`) take the resolved `exe` as an injected param — pure, and `osruntime_test.go` pins the argv shape without IO. clipcmd injects `SelfExe` via `opts` so `run.go` stays pure; the `os.Executable()` IO sits at the CLI boundary (`runcli.go:33`). No "pure" entity needs a mock to run.
- **clipcmd sibling resolution is the right call for the transition** (`runcli.go:26-38`). Resolving `dir(os.Executable())/pair` (not `os.Executable()`) keeps copy-on-select correct even while it still runs under its own standalone name during M2 — and `copy-on-select-test.sh` genuinely exercises this by placing a fake `pair` sibling that routes `clip copy-on-select` back to the real binary.
- **Green-at-every-commit discipline respected.** Helpers stay built, the distiller flat aliases (`scrollback-render`/`changelog`) are retained, and I confirmed both nested and flat forms still route — so no window breaks a caller.
- **Atlas updated in-range** (`atlas/go-migration-inventory.md`) with an accurate, specific M2 entry.

### 2. Critical findings

None.

### 3. Important findings

- **Plan Task 2.6 divergence — `.claude/settings*.json` not rewritten** (`.claude/settings.json:21-24,34`, `settings.local.json:12-15,25`). The plan explicitly listed rewriting `bin/pair-wrap`/`bin/pair-continuation` grants to `bin/pair wrap`/`bin/pair continuation`; the implementor left them and documented the reason in the atlas ("historical exact-match permission grants, not runtime callers"). The rationale is *sound* — these are frozen Claude-Code Bash-tool permission strings (one even carries a hardcoded old `session-id`), not runtime consumers, so leaving them breaks nothing. But the durable plan still claims the rewrite will happen. **Fix:** either rewrite them, or (better) add a plan `## Revisions` entry ratifying the "leave as historical grants" decision so plan and code agree. Non-blocking.

### 4. Minor findings

- Two `selfPairExe` functions, identical name, divergent behavior: `clipcmd/runcli.go:33` returns the **sibling** `pair`; `launcher/osruntime.go:289` returns `os.Executable()` **directly**. The divergence is justified (clipcmd may run under the standalone helper name and must find the pair sibling; the launcher is always `pair`), but the shared name invites a maintainer to assume they're interchangeable. Consider disambiguating (`siblingPairExe` vs `runningPairExe`) or cross-referencing in the comments.
- Self-identifying error prefixes still emit old names: `pair-session-watch:` (`sessionwatch/runcli.go:26`), `pair-scrollback-open:` (`opener/run.go:79`), `scrollback-render` (`opener/runtime.go:40`), `pair-scrollback-render` flagset (`scrollbackcmd.go:392`). Cosmetic, pre-existing, appropriate to fold into the M3 doc/cleanup sweep.

### 5. Test coverage notes

- Strong coverage for the Go/nvim callers (argv self-exec shape, guard collision, distiller form, real self-exec chain, nvim open→`pair review open`→zellij run).
- **Inherent gap:** the zellij two-token forms (`copy_command "pair clip copy-on-select"`, `Run "pair" "scrollback" "open"`, `Run "pair" "changelog" "open"`, layout `exec pair wrap`) and the Alt+/ / Alt+l / Alt+b rebindings have **no automated test** — they rest on the implementor's reading of zellij 0.44.3 source plus the Done-when live smoke. `copy-on-select-test.sh` exercises the copy chain via a fake `pair`, not real zellij config parsing. **The operator must complete the live-session smoke** (agent pane via `exec pair wrap`, Alt-select copy, Alt+/ scrollback, Alt+l changelog, Alt+b prev-prompt) before recording the close verdict — a unit test cannot cover this surface, and it's the one part of M2 not machine-verified here.

### 6. Architectural notes for upcoming work (M3)

- The distiller flat aliases (`scrollback-render`, `changelog`) now have **no in-repo caller** (verified) — M3 Task 3.3 removes them; `dispatcher_test.go`/`main_test.go` alias assertions will need updating.
- I hit a stale local `bin/pair` that misrouted group subcommands (`review open` → launcher). `pair-dev`'s rebuild-on-launch covers dev, but M3's deployed `pair`-on-PATH + bundle-shrink is where a stale/mismatched runtime `pair` would actually bite the brew/copied layout. Worth an explicit "runtime `pair` is current" guard/smoke at M3.
- The two `selfPairExe` are a natural unification target once `pair-scribe`/helpers fold in at M3.

### 7. Plan revision recommendations

Add to `workshop/plans/000104-single-pair-binary-plan.md` a `## Revisions` entry:

> **2026-07-06 — M2 close: `.claude` allowlist left as historical grants.** Task 2.6's directive to rewrite `.claude/settings.json`/`settings.local.json` `bin/pair-wrap`/`bin/pair-continuation` entries to `bin/pair <sub>` was intentionally **not** performed: these are exact-match Claude-Code Bash-tool permission grants (frozen, one bearing a hardcoded session-id), not runtime consumers, so they are not part of the `pair <sub>` consumer set. Recorded in `atlas/go-migration-inventory.md`. The M2 "every owned caller derives from `pair <sub>`" purpose is unaffected.
