---
id: 000034
status: done
estimate_hours: 0.5
deps: []
created: 2026-05-31
updated: 2026-06-01
actual_hours: 0.4
---

# codex quota model picker return remap

## Done when

- Codex quota/model-fallback choice prompts accept normal Return to confirm the highlighted option.
- Normal Codex textarea Return still inserts a newline.
- The detector uses a stable prompt signal or prompt-shape marker rather than option text when possible.

## Spec

After the ask-user-question fix, another Codex choice prompt still bypassed the
overlay-aware Return suspension. The observed prompt appeared when Codex ran out
of quota and asked whether to use a mini model. Pressing normal Return moved the
selection to the next item instead of confirming the highlighted option; the user
needed Alt+Return to select/confirm.

This appears to be a distinct Codex picker from the plan-mode
`request_user_input` prompt. The current OSC-based detector watches
`OSC 9;Plan mode prompt:...`, so quota/model-fallback prompts may emit a
different OSC body or only visible picker text. Raw scrollback showed no useful
prompt-specific OSC 9 body near this picker; the stable visible footer was
`Press enter to confirm or esc to go back`.

## Plan

- [x] Capture raw scrollback and OSC frames around the quota/model-fallback prompt.
- [x] Add the most stable detected signal to Codex overlay detection.
- [x] Add a regression test proving the quota/model picker sets `pickerActive`.
- [x] Verify normal Codex textarea Return remains remapped to newline.

## Log


- 2026-06-01: closed — env GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-wrap -count=1; env GOCACHE=/private/tmp/pair-go-cache make test; make pair-wrap; strings bin/pair-wrap contains Press enter to confirm or esc to go back
### 2026-05-31

- Filed from live observation: quota-exhaustion model-choice prompt required
  Alt+Return to confirm, while normal Return moved selection to the next item.

### 2026-06-01

- Inspected `/Users/xianxu/.local/share/pair/scrollback-pair-codex.raw` around
  the quota picker. Nearby OSC frames were only title/status notifications; the
  actionable picker footer was `Press enter to confirm or esc to go back`.
- Added that footer to the Codex visible-text fallback detector with regression
  coverage.
- Verified with focused `pair-wrap` tests and the full repo test target.
