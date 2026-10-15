# pair#556 — private Codex config RPC measurement

Exploratory measurement of installed Codex 0.162.1. No production implementation or support claim. All configuration values were synthetic; this evidence omits configuration and authentication contents.

## Isolation and process scope

Scratch root: `/private/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-codex-config-probe.VbWjof`

Each case used its own `<case>/home`, HOME-derived CODEX_HOME `<case>/home/.codex`, `<case>/cwd`, XDG directories, and TMPDIR. The subprocess environment was an explicit whitelist; real provider/authentication environment variables were not inherited. Credential storage was set to private file storage. Proxies pointed to a closed loopback port. No network trace was collected, so this is not a claim that every provider startup code path is network-free.

The exact native executable was:
`/opt/homebrew/lib/node_modules/@openai/codex/node_modules/@openai/codex-darwin-arm64/vendor/aarch64-apple-darwin/bin/codex`

Only the scratch app-server child was owned. It ran in a new process session, stdin was closed after the probe, and the script waited/reaped it. Every successful RPC case exited 0 naturally; failed profile cases exited 1. No real Codex state/auth or running agent was used. No thread/create, thread/resume, turn, or model requests were sent.

## Commands and RPC

Exact measurement entry commands:

```sh
python3 /tmp/pair-codex-config-probe.py
python3 /tmp/pair-codex-config-profile-probe.py
python3 /tmp/pair-codex-config-defaults-probe.py
python3 /tmp/pair-codex-config-relative-probe.py
```

Subprocess argv (configuration flag values redacted):

```text
codex app-server --stdio
codex -c model_provider=<synthetic-cli-provider> -c sqlite_home=<private-case-state-directory> app-server --stdio
codex -p probe app-server --stdio
codex -c profile=<synthetic-profile-name> app-server --stdio
codex -c sqlite_home=<synthetic-relative-directory> app-server --stdio
```

The actual executable is the absolute path above, not a PATH lookup. The cwd was always the relevant private `<case>/cwd`. Scripts remain under `/tmp` for exact synthetic reproduction; they are measurement artifacts, not repository source.

Exact protocol shapes (each object sent as one JSON line):

```json
{"id":1,"method":"initialize","params":{"clientInfo":{"name":"pair-private-config-probe","version":"0.0.0"},"capabilities":{"experimentalApi":true}}}
{"method":"initialized"}
{"id":2,"method":"config/read","params":{"includeLayers":true,"cwd":"<absolute-private-case-cwd>"}}
```

`initialize` latency includes process creation through response. `config/read` latency measures send through response. Timings are observations, not a benchmark distribution.

## Results

| Case | Initialize ms | Config read ms | Exit | Observation |
|---|---:|---:|---:|---|
| base | 1748.8 | 8.3 | 0 | Explicit synthetic provider/sqlite home returned correctly |
| override | 1782.1 | 7.6 | 0 | Both supplied CLI -c overrides returned correctly |
| profile-root | 40.1 | not sent | 1 | --profile is rejected for app-server |
| profile-config | 39.4 | not sent | 1 | Legacy profile config key rejected |
| profile-root-recheck | 51.9 | not sent | 1 | Same app-server profile restriction |
| profile-config-recheck | 48.7 | not sent | 1 | Same legacy profile restriction |
| defaults | 107.0 | 3.0 | 0 | model_provider and sqlite_home returned null; actual DBs under CODEX_HOME |
| relative | 72.6 | 3.0 | 0 | Config-file sqlite_home resolved relative to config directory |
| env-relative | 55.8 | 3.1 | 0 | CODEX_SQLITE_HOME resolved relative to cwd for actual DBs, but config/read sqlite_home remained null |
| cli-relative | 78.1 | 3.4 | 0 | CLI -c sqlite_home resolved relative to cwd in response and actual DB placement |

No case indicated authentication required. Response bodies were not saved or printed; only key presence and equality against synthetic expected values were emitted.

`config/read` is merged configuration, not the complete resolved runtime Config: omitted provider/home defaults are not materialized, and the SQLite environment override is not reflected. includeLayers+cwd alone is therefore insufficient as a universal runtime configuration oracle.

## Created scratch paths

Successful starts created these relative namespaces (no contents inspected):

- `home/.codex/installation_id`
- `home/.codex/.tmp/plugins.sync.lock`
- `home/.codex/.tmp/git-<generated>/objects` and, when observed, `refs`
- `home/.codex/tmp/arg0`
- `home/.codex/skills/.system/.codex-system-skills.marker`
- `home/.codex/skills/.system/{imagegen,openai-docs,review-agent,skill-creator,skill-installer}/...` (shipped skills/assets/scripts)

At each selected SQLite home, startup created `state_5.sqlite`, `logs_2.sqlite`, `goals_1.sqlite`, `memories_1.sqlite`, and `queue_1.sqlite`, with their `-wal`/`-shm` files observed in the initial cases. These are startup writes/migrations despite sending no conversation RPC.

Observed selected SQLite directories, relative to each case:

| Case | Relative selected SQLite directory |
|---|---|
| base | `state-base/` |
| override | `state-cli/` |
| defaults | `home/.codex/` |
| relative | `home/.codex/relative-store/` |
| env-relative | `cwd/relative-store/` |
| cli-relative | `cwd/relative-store/` |

Failed profile cases created only `home/.codex/tmp` and `home/.codex/tmp/arg0` in the measured after-exit listing.

## API/source references and limitation

Matching tag `rust-v0.162.1`:

- ConfigReadParams has only include_layers and optional cwd; no request-level profile selector: https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/app-server-protocol/src/protocol/v2/config.rs#L389
- App-server profile restriction: https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/cli/src/main.rs#L1910
- Native history provider resolution: https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/app_server_session/provider_selection.rs#L40
- SQLite configuration/default/environment precedence: https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/core/src/config/mod.rs#L4112

This candidate mechanism adds bounded metadata-process startup writes on an explicit launch. It is not a read-only inventory/startup-preview mechanism, and requires a plan delta approval before production implementation. Profile parity is unresolved; merely forwarding -p or translating it to legacy -c profile is invalid on the measured version.

## Existing Pair inventory scope

`cmd/internal/sessioninventory/runtime_os.go:62` constructs Codex roots as `filepath.Join(homeDir, ".codex", "sessions")`. `DefaultOSRuntime` uses os.UserHomeDir. Neither method reads CODEX_HOME. Existing launcher callers at `osruntime.go:705` and `recovery_authority_os.go:167,176` pass HOME, not the resolved Codex home.

Therefore custom CODEX_HOME is another provider-scope limitation: even a correctly resolved state database cannot authorize rollout paths under a custom Codex home through the current default codex-sessions root. No root logic was modified in this measurement.
