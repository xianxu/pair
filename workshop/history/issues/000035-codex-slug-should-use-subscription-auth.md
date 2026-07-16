---
id: 000035
status: done
deps: []
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 1.0
actual_hours: 0.4
---

# Codex slug should use subscription auth

## Done when

- Codex pair-slug generation works when the user is authenticated to Codex CLI through subscription tokens but has no `OPENAI_API_KEY`.
- Existing direct OpenAI API slug tests still pass.
- The atlas documents the Codex credential path accurately.

## Spec

The live Codex session is authenticated through `~/.codex/auth.json`, but
`pair-slug` currently calls the OpenAI Responses API directly for Codex and
requires `OPENAI_API_KEY`. On this machine `OPENAI_API_KEY` is unset, so the
turn-end slug proposal silently fails and `slug-proposed-pair` stays stale.

Codex slugging should use the same authenticated CLI path as the running Codex
session when no API key is present.

## Plan

- [x] Add a Codex CLI model runner for slug generation.
- [x] Route Codex slugging through direct OpenAI API when `OPENAI_API_KEY` is present, otherwise through `codex exec`.
- [x] Add a process-level fake Codex CLI regression test for the no-API-key path.
- [x] Update atlas and run verification.

## Log

### 2026-06-01

- Live check: `PAIR_SLUG_LOG=/private/tmp/pair-slug-live.log pair-slug` logged
  `model "gpt-5.4-mini" failed: OPENAI_API_KEY is not set`.
- `~/.codex/auth.json` has Codex subscription tokens and no API key, so the
  direct Responses API path cannot use this session's auth.
- Implemented Codex CLI fallback through `codex exec --sandbox read-only
  --ephemeral --output-last-message` when `OPENAI_API_KEY` is absent.
- Verification: `go test ./cmd/pair-slug`, `make test`, and `make pair-slug`.
  The build exited 0 and refreshed `bin/pair-slug`; Go also printed a sandboxed
  module-cache stat warning for `/Users/xianxu/go/pkg/mod/cache`.
- Closed: `go test ./cmd/pair-slug` and `make test` pass; `make pair-slug`
  refreshed `bin/pair-slug`, with only the sandboxed Go module-cache stat
  warning noted above.
- Live verification: inside Codex's command sandbox, `codex exec` reported
  `failed to initialize in-process app-server client: Operation not permitted`.
  Rerunning the same `bin/pair-slug` command unsandboxed succeeded through Codex
  CLI subscription auth and wrote `slug-proposed-pair`.
