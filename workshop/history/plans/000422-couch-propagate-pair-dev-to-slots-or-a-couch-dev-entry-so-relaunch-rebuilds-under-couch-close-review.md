# Boundary Review — pair#422 (whole-issue close)

| field | value |
|-------|-------|
| issue | 422 — couch: propagate PAIR_DEV to slots (or a couch dev entry) so relaunch rebuilds under couch |
| repo | pair |
| issue file | workshop/issues/000422-couch-propagate-pair-dev-to-slots-or-a-couch-dev-entry-so-relaunch-rebuilds-under-couch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 53ec05c9bd8c99d1dfeacb682e3a24606a3b02fc..9e1c0d0040b505f285a320dd6d156af464622aee |
| command | sdlc close --issue 422 |
| reviewer | claude |
| timestamp | 2026-10-09T23:35:11-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change is ready to ship. `couch-dev` copies the design of `pair-dev`: it sets dev mode (`PAIR_DEV=1`), rebuilds once through the shared `dev_rebuild` hook, and then runs the sibling `couch`. Slots pick up dev mode because they inherit Couch's environment, so Couch itself needs no new code. Two new Go tests pin that inheritance: `couchcore.buildExecCommand`/`mergeChildEnvironment` and `couchcmd.configuredRunner.childEnv` both keep the inherited `PAIR_DEV`.

I checked the claims in the docs and comments against the code:
- `childEnv` only clears inherited `PAIR_*_PATH` variables, then appends the supplied and runtime environment (`cmd/internal/couchcmd/singleton.go:287-299`).
- `mergeChildEnvironment` keeps every inherited key that isn't overridden (`cmd/internal/couchcore/runner.go:129`).

I ran these tests:
- `bash tests/dev-rebuild-test.sh`: all 6 cases pass, including the three new couch-dev cases.
- The two new Go tests: both pass.
- The `artifactpath` package: it fails, but only on `nvim/review/restore_*.lua` inventory entries that predate this change (the issue Log says the same). Nothing in its output involves `bin/` or `couch-dev`.

Wiring is complete: the `.gitignore` exception, `SHELL_BINS`, the `NonArtifactSources` entry, the install-layout symlink check, the README dev note and the atlas dev-Couch bullet.

1. **Strengths**
   - Building no propagation code and pinning the inheritance that already exists is the minimal correct fix. The test comments explain why the pin is needed (`cmd/internal/couchcore/runner_test.go:177`).
   - `bin/couch-dev:26-28` reuses `dev_rebuild` rather than writing a second build path (ARCH-DRY on the build hook).
   - The shell test goes through a symlink, which exercises the real `~/.local/bin` install shape. It also uses a fake sibling `couch` to check arguments and `PAIR_DEV` at the exec point, and covers the case where the build fails (`tests/dev-rebuild-test.sh:66-100`).
   - The README and atlas are both updated for the new entry point.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **ARCH-DRY, duplicated script code.** The symlink-resolving loop in `bin/couch-dev:17-23` is a verbatim copy of `bin/pair-dev:23-29`. These two are the only instances in this window. It could become a `resolve_here` helper in `bin/lib/`. It's six lines and acceptable.
   - **Stale comments and messages in the shared hook.** `bin/lib/dev-rebuild.sh` still says only `pair-dev` → `bin/pair` call it, and it prints messages prefixed `pair-dev:` even when `couch-dev` calls it. Its "fix, then Alt+n" hint also doesn't fit a Couch launch. The instances are the header comment and the three `echo` lines.

5. **Test coverage**
   - The two states in Done-when are covered. Dev mode is checked by the new couch-dev cases plus the Go pins. Deployed mode doing nothing is covered by the existing `dev_rebuild` case 1, and deployed `couch` has no code change.
   - The Go tests pass on the base too. That's expected, since they pin behavior that already exists rather than fixing a regression.

6. **Architecture**
   - ARCH-DRY: flagged, Minor only (the duplicated loop above). The rebuild hook itself is shared correctly.
   - ARCH-PURE: pass. The new code is launcher glue, and the env-merge logic it relies on is already a pure function with unit tests.
   - ARCH-PURPOSE: pass. It covers slot create, Alt+n and `couch --relaunch` (the relaunch path is confirmed in the Log). The only follow-up is the outside request to pair:1 for #421, which is separate work.

7. **Plan revisions:** none needed.

```findings
findings:
  - id: new
    severity: Minor
    family: launcher-script-shared-helpers
    title: |
      bin/couch-dev duplicates pair-dev's symlink-resolving loop verbatim (ARCH-DRY)
    detail: |
      bin/couch-dev:17-23 copies bin/pair-dev:23-29. These two are the only instances in the window. A resolve_here helper in bin/lib/ would make it one copy.
  - id: new
    severity: Minor
    family: shared-hook-doc-names-single-caller
    title: |
      bin/lib/dev-rebuild.sh header and messages still name only pair-dev as the caller
    detail: |
      Instances: the header comment (pair-dev then bin/pair is described as the caller), and the three echo lines prefixed "pair-dev:" with a "fix, then Alt+n" hint. couch-dev now calls the hook too, so the prefix and hint are misleading on a Couch launch.
```

---

## Re-review — 2026-10-09T23:41:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 422 — couch: propagate PAIR_DEV to slots (or a couch dev entry) so relaunch rebuilds under couch |
| repo | pair |
| issue file | workshop/issues/000422-couch-propagate-pair-dev-to-slots-or-a-couch-dev-entry-so-relaunch-rebuilds-under-couch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 53ec05c9bd8c99d1dfeacb682e3a24606a3b02fc..c2f5ce78b305ee9d7c676b8208ab54e3152959f4 |
| command | sdlc close --issue 422 |
| reviewer | claude |
| timestamp | 2026-10-09T23:41:20-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change is ready to ship. `bin/couch-dev` follows `pair-dev`'s design: it sets `PAIR_DEV=1`, rebuilds once through the shared `dev_rebuild` hook, and then runs the sibling `couch`. Slots pick up dev mode by inheriting Couch's environment. Two Go tests pin that inheritance (`couchcore` `buildExecCommand` and `couchcmd` `childEnv`), and the shell test covers the launch through a symlink, passing every argument, and a failed build. I re-ran `tests/dev-rebuild-test.sh` and all 6 cases pass.

**Prior findings:**
- **BR-2 is fixed.** Commits 5924e039 and 4d898f76 change the hook's header to name both callers. The three `echo` lines now start with `dev-rebuild:`, and the failure hint says "relaunch" instead of "Alt+n". I grepped the tree and nothing parses the old `pair-dev:` prefix, so changing it breaks nothing. This was a wording fix, so no regression test is needed.
- **BR-1 is still open.** `bin/couch-dev:17-23` is still a word-for-word copy of the symlink-resolving loop in `bin/pair-dev:23-29`. It is Minor and doesn't block shipping.

1. **Strengths**
   - Couch got no new propagation code. Instead, tests pin the environment inheritance that already exists (`cmd/internal/couchcore/runner_test.go:177`, `cmd/internal/couchcmd/singleton_runtime_test.go:249`).
   - Both scripts reuse the one `dev_rebuild` hook rather than adding a second build path.
   - The new launcher is fully wired: the `.gitignore` exception, `SHELL_BINS`, `NonArtifactSources`, the install-layout symlink check, the README and the atlas.
   - The hook's messages are now neutral about which script called them. That fixes the whole BR-2 family (header plus all three `echo` lines), not just one line.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** BR-1 is still open (see the findings block below).
5. **Test coverage:** Done-when has two states, dev and deployed, and both are covered. Dev mode is covered by the new couch-dev shell cases and the two Go pins. Deployed mode doing nothing is covered by the existing `dev_rebuild` case 1, and plain `couch` has no code change. The build-failure path is tested for couch-dev too.
6. **Architecture:**
   - **ARCH-DRY: flagged, Minor.** This is BR-1, the duplicated loop.
   - **ARCH-PURE: pass.** The new code is only launcher glue, and the environment-merge logic it relies on already has unit tests.
   - **ARCH-PURPOSE: pass.** Slot create, Alt+n and `couch --relaunch` all get dev mode through inheritance.
7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      bin/couch-dev:17-23 still copies bin/pair-dev:23-29 verbatim; no bin/lib resolve helper landed. Minor, non-blocking.
  - id: BR-2
    disposition: addressed
    note: |
      5924e039/4d898f76: header names both callers; all three echo lines now "dev-rebuild:" with a "relaunch" hint; grep shows no consumer of the old prefix.
```
