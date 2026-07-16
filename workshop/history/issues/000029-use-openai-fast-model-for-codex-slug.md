---
status: done
actual_hours: 0.6
updated: 2026-05-31
---

# Use an OpenAI fast model for Codex slugs

## Spec

When the active pair agent is Codex, `pair-slug` should generate the summary
slug through an OpenAI fast model instead of shelling out to `claude -p`.
Claude sessions should keep the existing Claude CLI path.

## Plan

- [x] Make `pair-slug` choose model/provider defaults by `PAIR_AGENT`.
- [x] Add an OpenAI Responses API caller for Codex slug generation.
- [x] Cover provider selection and response parsing with tests.
- [x] Verify with focused Go tests and full repo tests.

## Log


- 2026-05-31: closed — env GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-slug -count=1; env GOCACHE=/private/tmp/pair-go-cache make test; bin/pair-slug rebuilt
- 2026-06-01: OpenAI model docs identify smaller variants like
  `gpt-5.4-mini` / `gpt-5.4-nano` for lower-latency, lower-cost workloads,
  and list `codex-mini-latest` as deprecated. Choose `gpt-5.4-mini` as the
  default Codex slug model.
- 2026-06-01: Implemented agent-aware model provider selection:
  `PAIR_AGENT=codex` uses the OpenAI Responses API with default
  `gpt-5.4-mini`; other agents keep `claude -p` with default
  `claude-haiku-4-5`. `PAIR_SLUG_MODEL` still overrides the model string.
- 2026-06-01: Focused test passed:
  `env GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-slug -count=1`.
- 2026-06-01: Rebuilt `bin/pair-slug` and verified full suite:
  `env GOCACHE=/private/tmp/pair-go-cache make test`.
