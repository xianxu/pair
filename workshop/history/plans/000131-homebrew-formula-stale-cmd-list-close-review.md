# Boundary Review — pair#131 (whole-issue close)

| field | value |
|-------|-------|
| issue | 131 — homebrew formula cannot build: stale cmd list |
| repo | pair |
| issue file | workshop/issues/000131-homebrew-formula-stale-cmd-list.md |
| boundary | whole-issue close |
| window | 2dca338b7b62909038b9fed2fe6eaa65b6368d31..HEAD |
| command | `sdlc close --issue 131` |
| reviewer | codex |
| timestamp | 2026-08-16T20:42:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Verdict

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The review found the core #131 release purpose delivered: runtime-bundle
generation bootstraps from clean source, the public launcher handles
`pair --version`, release notes and atlas updates were present, and the
plan-gate ledger had no open findings.

## Findings

### Important

- `README.md` did not document the newly added public
  `pair version | --version` command, even though `cmd/internal/launcher/help.go`
  and the changelog did.

### Minor

- `VersionText()` satisfies the smoke check but does not include an actual
  tag/commit. Consider build-injected version metadata in future release
  automation.

## Resolution

- Added `pair version, --version` to the README command usage table.
- Added a release lesson requiring clean archive/package-manager smokes for
  generated ignored assets.

## Verification

- Pre-close: `go test ./...`; `go test ./cmd/internal/runtimebundlegen
  ./cmd/internal/runtimebundle -count=1`; `git diff --check`; `ruby -c
  Formula/pair.rb`; `brew style Formula/pair.rb`; `brew reinstall
  --build-from-source xianxu/pair/pair`; `brew test xianxu/pair/pair`;
  `/opt/homebrew/opt/pair/bin/pair --version`;
  `/opt/homebrew/opt/pair/bin/pair --help`; `brew info xianxu/pair/pair`.
- Post-review docs delta: `git diff --check`.
