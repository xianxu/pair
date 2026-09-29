# Boundary Review — pair#346 (milestone M1)

| field | value |
|-------|-------|
| issue | 346 — Stale temporary store blocks Couch startup |
| repo | pair |
| issue file | workshop/issues/000346-stale-temporary-store.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b622052f6f85db7f1585de823a8fa591b6d7e2d3..0387a3af11abe360f0d6261ca91c646c3ebcc2eb |
| command | sdlc milestone-close --issue 346 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-29T10:36:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1's core change is correct and well tested. Structural validation is now separate from availability checks: `validateStructure` versus `validate` in `cmd/internal/storagegc/stores.go:54-85`. `RegisterStore` checks only the registry structure and the store it is registering. `ReadRegistry`, `CompleteMigration` and the collector still require every registered store to be readable, so a missing store never quietly turns into permission to collect. `ForgetMissingStore` is exact and cautious. It refuses paths that still exist, symlinked paths, symlinked parent directories, paths that are not registered, unclean paths and permission failures. It also resets migration acknowledgment and removes only the named store. I re-ran the focused `storagegc` and `couchcmd` tests and they pass. Three problems remain, and each is cheap to fix. The regression meant to prove the operator's registry is untouched checks a file nothing could ever write, so it cannot fail. README does not document the new `pair gc` flag. And `pair gc` does not tell the operator how to recover when a registered store is missing, although the Spec requires it to ("communicates recovery requirements").

**1. Strengths**
- The availability split is correct (`stores.go:54-85`, `:116`). `CompleteMigration` now calls `validate()` explicitly (`:156`), so acknowledging the store list still requires every store to be available.
- `requireMissingStore` / `requireDirectoryOrMissing` (`stores.go:238-273`) check every component of the path with `Lstat`. A dangling alias or an `EACCES` error therefore cannot count as proof that a store is gone. `TestUnavailablePermissionsAreNotMissingStoreEvidence` covers this.
- `TestUnavailableStorePreventsCollectionAfterRegistration` builds a real eligible payload and confirms it survives both preview and apply while a store is unavailable. That tests the retention safeguard directly.
- `TestListWithMissingAuxiliaryStoreUsesIsolatedRoots` runs production `couchcmd.Run --list` with the actual incident shape (an extra store registered, then deleted), and confirms the missing registration is kept, not silently dropped.
- The mixed-flag rejection in `gccmd` is checked with "no files written" assertions (`run_test.go:92-106`).

**2. Critical:** none.

**3. Important**
- **The isolation check can never fail** (`cmd/internal/couchcmd/stale_store_test.go:16-24,58-61`).
  - The sentinel is written to `operatorRoot/stores.json`, but the registry lives at `<root>/.retention/stores.json`.
  - `PAIR_DATA_DIR`/`COUCH_STORE_DIR` point at `operatorRoot` only until the loop on the next lines blanks every `PAIR_*`/`COUCH_*` variable, and they are then reset to fresh roots.
  - So no code path can touch the sentinel. The byte-equality assertion proves nothing, and a real leak would go to the actual `~/.local/share/pair` without being noticed.
  - Plan item "Isolate smoke HOME/XDG/Pair/Couch roots" is ticked. But `tests/couch-recovery-smoke.sh` is unchanged, and the plan's test strategy ("sentinel operator-shaped root stays byte-identical through isolated subprocess") was not built.
  - **Fix:** put a byte-exact sentinel `.retention/stores.json` under a fake "operator" HOME/XDG root that stays the ambient default. Run the production path, ideally as a subprocess that inherits a polluted environment. Assert the sentinel is unchanged and that the isolated root received the registration. Then either audit and fix the smoke script's isolation, or untick the plan item and add a Revisions entry.
- **README has no entry for `--forget-missing-store`** (README.md:812-816 documents the `pair gc` store flags). Add a short paragraph: what the flag does, that it resets migration acknowledgment, that the operator must re-acknowledge the list with `--complete-migration`, and to remount instead if the store is only temporarily unavailable.
- **`pair gc` gives no recovery pointer when a store is missing** (`gccmd/run.go:143-155`, `collector.go:123`).
  - When `ReadRegistry` fails, the "Registered Couch stores" list is silently left out, so the operator cannot see which path to pass.
  - The block reason is just `registered store X unavailable: lstat …`, with no mention of remounting or `--forget-missing-store`.
  - The only pointer is in the atlas, so the Done-when item "communicates recovery requirements" is only met on paper.
  - **Fix:** when a store is unavailable, list the registered stores from `readRegistryFile`, marking which are unavailable, and print both next actions (remount/restore, or `pair gc --forget-missing-store <exact path>`).

**4. Minor**
- `UnregisterStore` (`stores.go:187`) still uses the full `ReadRegistry`, so removing an intact, provably empty store is blocked by an unrelated missing store. That is inconsistent with the availability split. It is safe, but worth a comment or a change to `loadRegistry` plus availability checks only on the target store.
- `ForgetMissingStore` cannot tell an unmounted volume from a deleted directory (both return ENOENT). The atlas says to remount instead. Consider naming this in the CLI's success output too.
- `stores_test.go:108` leaves a stray blank line where the old assertion was removed.

**5. Test coverage notes**
- Covered well: the registry logic and the collector safeguards.
- Not covered: the isolation claim (the first Important finding).
- There is no table or property test for `validateStructure` over malformed registries (unclean paths, unsorted entries, null stores), which the plan's strategy table lists. Existing tests cover some of these cases only indirectly.

**6. Architectural notes**
- **ARCH-DRY: pass.** The two ancestor helpers overlap slightly but are acceptable.
- **ARCH-PURE: pass.** `validateStructure` is pure. The filesystem checks sit in small helpers that do only I/O.
- **ARCH-PURPOSE: flag.** The prevention part of the Spec (isolation proven by a regression) is only met nominally; see the first Important finding.
- **ARCH-MOCK: pass.** Tests use a real temporary filesystem as a stateful fake.
- **ARCH-CONSTRAINTS: pass.** Only metadata checks, bounded by the number of registered stores.
- **ARCH-SECURE: flag.** Same issue as the first Important finding: ambient operator state is not demonstrably protected. Handling of registry data read from disk is sound (structure check plus a canonical check at collection time).
- **ARCH-ORDER: pass.** All changes to `migration_complete` and the store list happen under the coordinator lock. Forgetting a store puts the registry back into the not-acknowledged state explicitly.
- **ARCH-FUNERAL: flag, carried into M2/M3.** Temporary stores still have no automatic end. Removal is manual only, and depends on isolation keeping them out of the registry. Once isolation is proven that is acceptable, but record it as the chosen lifecycle.

**7. Plan revision recommendations**
- Either implement the smoke/subprocess isolation sentinel, or append a `## Revisions` entry saying the smoke-script isolation item was satisfied by existing fixtures (naming them) and untick or reword it.
- The fifth M1 checkbox (tests/atlas/close) is still unticked. Tick it once `sdlc milestone-close` records the verdict.

```findings
findings:
  - id: new
    severity: Important
    family: test-oracle-cannot-fail
    title: |
      Operator-registry sentinel in couchcmd stale_store_test is unreachable, so isolation is unproven
    detail: |
      The sentinel is written at operatorRoot/stores.json, not .retention/stores.json, and the env loop immediately blanks PAIR_DATA_DIR, so no code path can modify it. Build an operator-shaped default root with a real .retention/stores.json sentinel and run the production path (ideally a subprocess with a polluted inherited environment). The ticked smoke-isolation plan item has no matching change to tests/couch-recovery-smoke.sh.
  - id: new
    severity: Important
    family: readme-surface-gap
    title: |
      README does not document pair gc --forget-missing-store
    detail: |
      README.md around lines 812-816 lists the pair gc store flags. Add the new abandonment flag, its migration reset, the --complete-migration re-acknowledgment, and remounting as the alternative for a temporarily unavailable store.
  - id: new
    severity: Important
    family: outage-diagnostic-actionability
    title: |
      pair gc gives no recovery guidance or store list when a registered store is unavailable
    detail: |
      gccmd/run.go:143 hides the registered-store list when ReadRegistry fails, and the collector block reason is a raw lstat error. Print the structurally valid inventory with unavailable stores marked, and name both next actions (restore/remount, or --forget-missing-store with the exact path) to meet the Spec's "communicates recovery requirements".
  - id: new
    severity: Minor
    family: availability-split-consistency
    title: |
      UnregisterStore still requires every unrelated store to be available
    detail: |
      stores.go:187 uses the full ReadRegistry, so removing an intact store with an empty-store proof is blocked by an unrelated outage. This is safe but inconsistent with the structure/availability split.
```
