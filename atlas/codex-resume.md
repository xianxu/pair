# Codex read-only resume selection

Pair maps the local configuration and history behavior of Codex0.162.1
(`rust-v0.162.1`). It reads files; it does not start Codex or app-server to discover
configuration. The launcher offers ordinary latest selectors to this adapter
before deciding whether they can use native writer admission.

## Guarded support and native compatibility

A complete local pre-cloud bootstrap with `forced_login_method = "api"` rules out
backend-authenticated cloud policy without reading credentials. Only complete
observations and a unique native result can become a guarded explicit UUID.
Unknown cloud eligibility retains the original invocation and the wrapper prints
one line saying concurrent native resume is **not guarded**. An absent cloud cache,
an API-key environment variable, or an explicit provider does not prove cloud
policy absent. Auth/credential files and keychains are never read.

| Invocation/evidence | Result |
|---|---|
| Local API-only bootstrap, supported config and unique latest | Resolve once, pin UUID, reserve and verify scope before execution |
| Cloud-unknown or unsupported managed/executor configuration | Native pass-through, one not-guarded note |
| Picker, name/prefix, remote or worktree creation | Existing native compatibility where allowed by the selector policy |
| Missing/incomplete DB evidence, tied activity or read limit | Native pass-through; Pair does not repair or guess |
| Exact UUID | Existing guarded admission; custom CODEX_HOME supplies its transcript root |
| Strict recovery/ResumeRequired with unresolved evidence | Existing refusal; no silent downgrade |

The current machine's ordinary configuration was measured as cloud-unknown;
read-only DB comparisons do not claim guarded support for that configuration.
Supported positive tests use real private API-only configuration through the same
reader, collector and reservation path.

## Configuration ownership

`sessioninventory/codex_config*.go` owns typed projection and the thin reader.
The reader receives invocation inputs from the existing native selector parser;
provider, profile and prompt syntax are not reparsed by a second argv grammar.

- `CODEX_HOME` selects the physical native home, defaulting to `HOME/.codex`.
  A relative explicit home resolves from process startup cwd, before `-C`.
- Base `config.toml` and selected `NAME.config.toml` use profile-v2 semantics.
  Legacy profile selectors and unresolved inputs retain native behavior.
- System/user/profile/trusted-project/session layers retain source precedence.
  Project provider keys are ignored; trusted project SQLite and worktree settings
  may apply. Unsupported managed/MDM requirements remain uncertainty.
- File paths retain their origin directory; CLI values and `CODEX_SQLITE_HOME`
  use effective cwd. Configured `sqlite_home` precedes the environment value;
  the default is Codex home. Transcript and state homes remain separate.
- Worktrees defaults on. Validated current/primary/linked checkout metadata
  supplies corresponding existing directories. Nested logical cwd aliases whose
  native interpretation is unsupported use compatibility.
- Named `environments.toml`, executor-server scope and unsupported relocation
  also use compatibility. No broad Codex-home scan discovers policy.

On macOS, the small `darwin && cgo` adapter asks CoreFoundation whether the native
managed preference keys are forced. Darwin builds without CGO cannot prove those
inputs absent and use compatibility. Other unimplemented platform-policy scopes
also remain unknown. TOML parsing uses the pinned Go TOML library, not a line or
regular-expression approximation.

## Distinct latest algorithms

`codex_latest.go` keeps two source-derived collectors:

- TUI filters active, visible interactive history by history provider and the
  effective worktree/all cwd set. A malformed native top result never permits
  selecting a different lower result as though native did that.
- Exec lists all native source kinds by provider, then compares the latest
  `turn_context.cwd` found by reverse rollout reads. Only complete absence permits
  DB-cwd fallback; `--all` removes the cwd predicate. Timestamp-cursor pagination
  preserves the observation when rows arrive or disappear between pages.

UpdatedAt has no specified UUID tiebreaker. The adapter retains compatibility
when timestamp ambiguity prevents proving the native result. Every inspected
transcript snapshot stays fenced through final inventory validation. DB paths,
filename IDs and native metadata must agree inside the authorized transcript root.

## Handoff and budgets

The existing launcher resolver pins one UUID before reservation. A serialized,
bounded namespace binding on the existing admission request covers physical
homes, provider, mode and cwd scope; the scoped digest also binds final argv.
Before Claim, the wrapper reprojects its actual child environment/cwd and checks
the binding. Mismatch refuses with retry advice without consuming the reservation.
No second latest lookup occurs. Removing or forging scope cannot downgrade the
stored request into a legacy claim. Existing grant validation and CAS still own
writer admission and teardown.

Configuration reads cap files at1MiB and total observations at4MiB, with64 named
reads and64KiB Git metadata. Selector work uses the existing30s/256MiB/10000-entry
bounds,1MiB SQLite page output and32MiB per reverse transcript scan. Exhaustion is
unknown, not absence. Scope JSON is bounded at8KiB and grant encoding at64KiB;
the existing1MiB admission store cap and lifecycle remain.

Custom native-home construction also feeds ledger identity and recovery
revalidation, so a selected custom-home UUID does not later fall back to the
default transcript root. Invalid home evidence refuses affected Codex proof;
other-agent-only ledger paths remain unaffected.

## Verification

Run focused normal tests, then the same selection under `-race`:

```sh
go test ./cmd/internal/sessioninventory ./cmd/internal/launcher ./cmd/internal/wrapcmd \
  -run 'Test.*(CodexConfig|CodexLatest|NativeSelectorCodex|NativeResume|NativeAdmission|NativeCreate|NativeInventory)' -count=1
```

Private real TOML, SQLite and JSONL fixtures cover provenance and candidate
mutation; the stateful admission fixtures control handoff ordering. Native
configuration files are never modified by these tests. Any live operator action
is scheduled through the hub. Normal release verification also runs the complete
recorded suite and source/artifact ownership guards.
