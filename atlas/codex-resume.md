# Codex resume

Ordinary indirect Codex resumes keep native CLI behavior. Latest, pickers,
names and prefixes pass through with their original arguments before any
home, configuration, transcript, Git or database probing. Pair does not predict
the selected conversation or reserve a guessed identity. The wrapper prints
one line explaining that concurrent resume is not guarded.

Full canonical UUIDs use existing exact-ID validation. Standalone launches
retain Reserve → Claim → Publish admission; Couch-hosted recovery keeps main's
conversation-registry checks and exact pinning. This home route adds no
ownership policy. Strict required/recovery launches still require proven
identity; indirect compatibility is not proof.

## Private app server

The launcher probes `codex --version` once (read-only, 3s bound) and sets
`PAIR_AGENT_VERSION` on every launch — the parsed version, or empty when the
probe is unavailable; the empty write also clears any value inherited from an
outer pane, so the wrapper copies this launch's version (and nothing stale)
into the ready record's `agent_version`. At 0.164.0 and above every Codex
launch also gets a leading `--no-daemon`, so the session runs without the
machine's shared background server and cannot be stopped by its
incompatible-feature-settings prompt (pair#891). Below the gate, or when the
probe is unparseable, the argv is unchanged, so an unknown CLI never gains a
flag it may not support. Pair never restarts or contacts the shared server.
The flag is a Codex global option in `codexBoolGlobalOption`, so the wrapper's
fresh-start strip still walks past it to drop a stale `resume <id>`.

The version is a launch-time value: an in-pane fresh start reuses the launch's
env and argv, so a mid-session Codex upgrade keeps the old record (and the old
policy) until the next launch.

## Transcript home

`sessioninventory.ResolveCodexHome` owns the small home projection:

- Nonempty CODEX_HOME must name an existing directory and is canonicalized.
  It does not require HOME. Relative values use the proven startup directory
  before native `-C`; missing provenance cannot be guessed from a selected cwd.
- Empty or unset CODEX_HOME uses HOME/.codex, which need not exist yet.

`sessioninventory.NewNativeTranscriptRuntime` applies that home consistently to
exact-ID validation, ledger evidence, recovery discovery/validation and legacy
held-file ownership checks. Historical recovery has no proven startup directory;
a relative custom home therefore cannot authorize a candidate. An unavailable
Codex home blocks Codex entries only, preserving other agents in a shared ledger. This changes transcript lookup only;
the shared admission protocol remains unchanged. Session inventory rejects
Codex indirect selector requests before IO, including legacy requests carrying
provider or database metadata. Other agents retain their selector behavior.

## Boundary and verification

There is no supported native read-only resolve-resume-target API. Reading
Codex's state database is not that contract: native selection policy lives in
its clients and changes independently. Pair therefore carries no Codex latest
selector, configuration/provider policy, Git scope emulation, SQLite snapshot
or TOML dependency. A future adapter requires a supported native API that
accepts the actual invocation and returns its selected identity and namespace
before opening or writing conversation state.

Focused tests defend zero-probe indirect launches, exact-ID custom-home and
startup-directory routing, strict recovery, and unchanged admission. Inventory
tests count IO at the indirect boundary. Home tests cover canonical aliases,
relative paths, invalid directories and absent HOME. The normal full suite
also runs the source-classification guards.
