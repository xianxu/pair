---
id: 000346
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '0618ac2ff5a4412c294aa5455db6e1b126b28c03' # card fields mirrored from issue-cards; edit via sdlc
---

# Stale temporary store blocks Couch startup

## Problem

After a macOS upgrade/reboot, both interactive `couch` and `couch --list` fail with `registered store ".../scratchpad/store.tYjN" unavailable: lstat /private/tmp/claude-501: no such file or directory`.

The persistent Pair retention registry contained the normal `~/.local/share/pair/couch` store plus a temporary Claude scratchpad store from a Pair slot1 development session. The temporary directory no longer exists; the normal store remains present. The path suggests a development/smoke-test store leaked into the production inventory; the exact originating invocation is not yet verified. `/tmp` resolves to `/private/tmp` on this machine, and registration canonicalizes symlinks.

`storagegc.StoreRegistry.validate` checks every registered path. `RegisterStore` validates the existing inventory before registering even the normal store; `NewCoordinatedThreadStore` invokes registration during construction. One vanished auxiliary store therefore prevents startup and read-only listing. `UnregisterStore` also reads the fully validated registry and requires an existing canonical directory, leaving no normal recovery route for a missing store.

## Spec

Prevent development/test stores from contaminating the operator's durable retention inventory. Trace the source of the scratchpad registration and ensure smoke tests isolate both Couch namespaces and Pair retention roots.

Define supported recovery for vanished registered stores and allow unaffected stores to remain usable where safe. Preserve retention safety: an unavailable store is not proof that it had no references; do not silently enable collection or discard real thread data. Distinguish missing temporary stores from unreadable or temporarily unmounted durable stores and make the recovery action explicit.

Relevant code: `cmd/internal/storagegc/stores.go`, `cmd/internal/couchcore/retention.go`, and `cmd/internal/couchcmd/run.go`. ARCH-SECURE: tests must not write operator state. ARCH-FUNERAL: registration must define the lifecycle and removal of ephemeral stores.

## Done when

- A regression reproduces registration of an auxiliary temporary store, deletion of its directory, and subsequent launch/list behavior for the intact normal store.
- A supported, actionable recovery path handles stale missing registrations without fabricating empty-store proof or weakening garbage-collection safeguards.
- Development/smoke-test invocations isolate all persistent roots; regression coverage proves the operator registry remains untouched.
- Missing, unreadable, and temporarily unavailable durable stores have tested behavior that preserves references and communicates recovery requirements.

## Plan

## Log

### 2026-09-29

Reproduced locally: `couch --list` exited 1 with the same missing-store error as the screenshot. With the retention coordinator's exclusive flock held, backed up `stores.json`, removed only the exact missing scratchpad registration, atomically replaced the registry, and retained the original migration flag and normal store. Backup: `~/.local/share/pair/.retention/stores.json.backup-20260929-072547`. No thread payloads were removed. Afterwards `couch --list` exited 0 and enumerated existing threads. Many report lost session bindings following the reboot; those are separate from the registry startup blocker. This ticket tracks the permanent fix; no implementation changes made.
