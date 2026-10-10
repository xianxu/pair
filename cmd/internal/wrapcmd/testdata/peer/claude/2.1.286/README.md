# Claude Code 2.1.286 peer composer captures

Captured 2026-09-30 by `TestPeerLiveConformance` in disposable PTYs/repositories,
using safe mode and existing native login credentials. No existing Pair/Couch
conversation was contacted. Commands used `-peer-live-harness=claude
-peer-live-use-local-auth -peer-live-capture-dir=/tmp/pair-353-claude` with the
scenario flags below.

- `startup-fix-lint.raw`, `startup-create-util.raw`, `startup-write-test.raw`:
  authenticated empty composers with three rotating native hints. The invariant
  is the known ruled composer, cursor at column two on the prompt row, a single
  `Try "…"` suggestion with no embedded quotes/newlines or extra suffix, and all
  nonblank suggestion glyphs faint. Wording is not invariant. Tests also reject
  ordinarily styled identical text, unrelated faint text, multiline text and a
  displaced cursor. Wrapper human-input ownership remains an independent gate.
- `paste-short.raw`: `-peer-live-scenario=paste-short` rendered the exact 100-byte
  peer envelope. No submission.
- `submit.raw`: `-peer-live-submit` sent that harmless no-tools envelope exactly
  once through the production dispatcher and observed the composer clear after
  submission. This establishes transport submission, not task completion.
- `paste-multiline.raw`: `-peer-live-scenario=paste-multiline` rendered a collapsed
  paste marker, `[Pasted text #1 +5 lines]`. Until #418 such messages expired
  unsubmitted, because the marker does not prove the complete body. Since #418,
  by operator decision, the strict form submits: the marker is the composer's
  whole content, the cursor sits right after it, and `+5` equals the envelope's
  newline count.
- `paste-wrapped.raw`: `-peer-live-scenario=paste-wrapped` shows native word
  wrapping at the captured width minus four columns. The matching projection
  permits only single ordinary inter-word spaces; leading/trailing, repeated,
  tab or other ambiguous whitespace is never normalized through this route.

Small messages, qualified word wrapping and the strict collapsed marker are
supported. Claude wraps at spaces only, never after a hyphen (#418, captured
from ariadne:2 on 2.1.295 and pinned in `TestPeerComposerClaudeCapturedHyphenWrap`).
No generic faint-text acceptance or loose marker fallback was introduced.
