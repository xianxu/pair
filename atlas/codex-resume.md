# Codex resume

Ordinary indirect Codex resumes keep native CLI behavior. Latest, pickers,
names and prefixes pass through with their original arguments before any
home, configuration, transcript, Git or database probing. Pair does not predict
the selected conversation or reserve a guessed identity. The wrapper prints
one line explaining that concurrent resume is not guarded.

Full canonical UUIDs use the existing exact-ID validation and
Reserve → Claim → Publish admission flow. Strict required/recovery launches
still require proven identity; indirect compatibility is not proof.

## Transcript home

`sessioninventory.ResolveCodexHome` owns the small home projection:

- Nonempty CODEX_HOME must name an existing directory and is canonicalized.
  It does not require HOME. Relative values use the proven startup directory
  before native `-C`; missing provenance cannot be guessed from a selected cwd.
- Empty or unset CODEX_HOME uses HOME/.codex, which need not exist yet.

`launcher/native_inventory_os.go` applies that home consistently to exact-ID
validation, ledger evidence and recovery. This changes transcript lookup only;
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
