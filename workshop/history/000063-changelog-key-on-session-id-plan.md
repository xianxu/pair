# Key the changelog on session_id Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Key the change-log file set on the agent's persisted `session_id` so a fresh session (Alt+Shift+N) opens an empty log and a resume (Alt+n) reopens the same growing one.

**Architecture:** The change-log base path (built today as `changelog-<tag>-<agent>` in two independent places — `bin/pair-changelog-open` and the draft-nvim `.ready` watcher) gains a `-<session_id>` suffix. The session id is resolved from one canonical source — the per-tag config JSON the launcher/​watcher already writes (`config-<tag>-<agent>.json`, key `session_id`) — with an exported `PAIR_SESSION_ID` env var as a launch-time fast path. No new "epoch" concept, no pointer/archive file: a different session id **is** a different file, which **is** the reset.

**Tech Stack:** POSIX sh (`bin/pair`, `bin/pair-changelog-open`), Lua (`nvim/init.lua`), Go is untouched (`cmd/pair-changelog` derives every path from the `--log` flag it's handed).

---

## Why this is more subtle than "add a suffix" (read before coding)

The spec frames `session_id` as "minted on a fresh start, reused on resume." That is exactly true **only for claude and for any resume**. The session id is known at `bin/pair` launch time in three cases:

- **claude fresh session** — `bin/pair` mints `$new_sid` via `uuidgen` and pre-injects `--session-id` (`bin/pair:2061-2069`).
- **explicit resume (any agent)** — `$explicit_resume` is parsed from argv (`bin/pair:1993/1997`); the Alt+n restart re-execs with `--resume <id>` so it lands here too.

But for a **codex/agy fresh session there is no `--session-id` flag** — the id is discovered *asynchronously* by `bin/pair-session-watch.sh` (`extract_id`, ~60s deadline) and written to the config *after* zellij (and the draft nvim) have already started. So:

- At launch we can only export `PAIR_SESSION_ID` for the known-at-launch cases; for codex/agy fresh it is empty at launch.
- Therefore **the config file is the canonical source** and both consumers must fall back to reading it. nvim must re-resolve on each watcher tick (the id may land mid-session). `pair-changelog-open` runs on demand (Alt+l), by which point the watcher has long since written the config.

Design consequences, baked into the tasks below:

1. **Single source of truth = the config** (ARCH-DRY). The env var is a launch-time cache of it, never an independent fact.
2. **Resolution order in every consumer:** `PAIR_SESSION_ID` (if non-empty) → `config.session_id` → none.
3. **Empty/no id ⇒ the old base** `changelog-<tag>-<agent>` with **no suffix** — so every pre-existing test and on-disk log stays valid (backward compatible; the `[ -z ]`/`nil` branch is the legacy path).
4. **Full uuid in the filename, no truncation.** All three agents' ids are lowercase `8-4-4-4-12` hex uuids (`pair-session-watch.sh:157,167`; claude via `uuidgen|tr A-Z a-z`) — path-safe and ~36 chars (total path well under any limit). Truncating would introduce a (tiny) collision risk that does not exist today, for a cosmetic gain on a non-user-facing data-dir filename. The spec's "can be truncated" is explicitly optional; we decline it (Root-Cause / Simplicity-First).

---

## Core concepts

This work adds no new data shape; it threads one existing fact (`session_id`) into one existing path-construction (the change-log base) at the two seams that build it. The "entities" here are integration seams, not pure nouns — the only pure-ish nugget is the base-path string-build, which we keep identical in shape across the shell and Lua consumers.

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| change-log base path (`changelog-<tag>-<agent>[-<sid>]`) | `bin/pair-changelog-open`, `nvim/init.lua` | modified |

- **change-log base path** — the `$base` (shell) / `marker` prefix (Lua) that every change-log file hangs off (`.md/.anchor/.cleaned/.status/.ready/.openlock/.distill.lock`). Today `changelog-<tag>-<agent>`; gains an optional `-<sid>` suffix.
  - **Relationships:** 1:1 with a `(tag, agent, session)` triple. The Go distiller derives `.ready` and the lock/status siblings purely from the `--log`/`--anchor` paths it is handed, so changing the base in `pair-changelog-open` propagates to all siblings for free — **only the two base-builders change.**
  - **DRY rationale:** The base is built in exactly two places that must agree (the opener that the distiller writes through, and the nvim watcher that polls `.ready`). They are not extractable into one shared module (sh vs Lua), so the contract is "identical resolution order + identical suffix shape," pinned by tests on both sides. No third builder exists.
  - **Future extensions:** reaping old `changelog-…-<sid>.*` files (noted out-of-scope follow-up); a `/clear`-aware live id (atlas:479 gap — out of scope).

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `PAIR_SESSION_ID` export | `bin/pair` | new | launch-time session id |
| session-id resolution (env → config) | `bin/pair-changelog-open` | modified | config JSON read |
| session-id resolution (env → config) | `nvim/init.lua` `pair_start_changelog_ready_watch` | modified | config JSON read |

- **`PAIR_SESSION_ID` export** — one `export` just before the zellij launch, valued from the ids already in scope.
  - **Injected into:** inherited by every zellij pane → read by `pair-changelog-open` and the draft nvim.
  - **Future extensions:** if a `/clear` live-id is ever surfaced, this is the single point to refresh it.
- **session-id resolution** (both consumers) — `PAIR_SESSION_ID` if non-empty, else `session_id` from `config-<tag>-<agent>.json`, else none.
  - **Injected into:** the base-path build above.
  - **Future extensions:** a shared resolver if a third consumer ever appears (none today).

**Test surface.** Two shell e2e/keying tests assert the base-path contract from the `pair-changelog-open` side; the existing headless nvim flash test (`changelog-notify-test.sh`) is extended to assert the watcher resolves `PAIR_SESSION_ID` into the keyed `.ready` path. No Go changes ⇒ no Go test changes (the `cmd/pair-changelog` suite already covers `.ready` derivation from `--log`).

---

## Tasks

> TDD note: the keying lives in shell + Lua glue (no pure Go function), so the
> "test" steps are shell/headless-nvim assertions. Per the lessons file, when we
> change file-name output, the **shell** tests are the ones that gate (they don't
> show up in `go test`) — so we lead with them and always finish on `make test`.

### Task 1: `pair-changelog-open` resolves the session id and keys the base

**Files:**
- Modify: `bin/pair-changelog-open:28` (the `base=` line + a resolution block above it)
- Test: `tests/changelog-session-key-test.sh` (new), wired into `test-changelog` in `Makefile.local`

- [ ] **Step 1: Write the failing keying test** — `tests/changelog-session-key-test.sh`

  A fast, focused test (no model/distiller): set `$RAW` empty so the `[ -s "$RAW" ]`
  guard skips the distiller; `pair-changelog-open` then just resolves the base,
  touches `<base>.md`, and opens the fake nvim on it. Assert which file nvim opened.
  Cover four cases against a fake `nvim` (records its file arg) on PATH:

  ```sh
  #!/bin/sh
  # Focused keying test for bin/pair-changelog-open (#63): the change-log base is
  # keyed on the resolved session id (PAIR_SESSION_ID → config → none). No model
  # or distiller runs — $RAW is empty, so the opener only resolves the base and
  # opens the (fake) viewer on it; we assert the path nvim was handed.
  set -eu
  PAIR_HOME=$(cd "$(dirname "$0")/.." && pwd); export PAIR_HOME
  tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-changelog-key.XXXXXX"); trap 'rm -rf "$tmp"' EXIT
  export PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t PAIR_AGENT=claude
  mkdir -p "$PAIR_DATA_DIR"
  fakebin="$tmp/bin"; mkdir -p "$fakebin"
  cat > "$fakebin/nvim" <<EOF
  #!/bin/sh
  # last arg is the log path
  for a in "\$@"; do :; done; printf '%s\n' "\$a" > "$tmp/nvim-arg"
  EOF
  chmod +x "$fakebin/nvim"; export PATH="$fakebin:$PATH"
  fail=0
  opened() { tail -n1 "$tmp/nvim-arg"; }
  run() { rm -f "$tmp/nvim-arg"; "$PAIR_HOME/bin/pair-changelog-open"; }

  # (a) PAIR_SESSION_ID set → keyed base
  PAIR_SESSION_ID=aaaa1111-2222-3333-4444-555566667777 run
  case "$(opened)" in *"changelog-t-claude-aaaa1111-2222-3333-4444-555566667777.md") ;;
    *) echo "FAIL (a) env-keyed base: $(opened)"; fail=1 ;; esac

  # (b) fresh session = different id → different (empty) file
  printf 'old log\n' > "$PAIR_DATA_DIR/changelog-t-claude-aaaa1111-2222-3333-4444-555566667777.md"
  PAIR_SESSION_ID=bbbb1111-2222-3333-4444-555566667777 run
  case "$(opened)" in *"changelog-t-claude-bbbb1111-2222-3333-4444-555566667777.md") ;;
    *) echo "FAIL (b) fresh id base: $(opened)"; fail=1 ;; esac
  [ -s "$PAIR_DATA_DIR/changelog-t-claude-bbbb1111-2222-3333-4444-555566667777.md" ] \
    && { echo "FAIL (b) fresh log not empty"; fail=1; }

  # (c) resume = same id → same file, prior content intact
  PAIR_SESSION_ID=aaaa1111-2222-3333-4444-555566667777 run
  grep -q 'old log' "$PAIR_DATA_DIR/changelog-t-claude-aaaa1111-2222-3333-4444-555566667777.md" \
    || { echo "FAIL (c) resume lost prior content"; fail=1; }

  # (d) env unset → fall back to config.session_id
  unset PAIR_SESSION_ID
  printf '{"agent":"claude","args":[],"session_id":"cccc1111-2222-3333-4444-555566667777"}' \
    > "$PAIR_DATA_DIR/config-t-claude.json"
  run
  case "$(opened)" in *"changelog-t-claude-cccc1111-2222-3333-4444-555566667777.md") ;;
    *) echo "FAIL (d) config-fallback base: $(opened)"; fail=1 ;; esac

  # (e) no env, no config session_id → legacy unsuffixed base (backward compat)
  rm -f "$PAIR_DATA_DIR/config-t-claude.json"
  run
  case "$(opened)" in *"changelog-t-claude.md") ;;
    *) echo "FAIL (e) legacy base: $(opened)"; fail=1 ;; esac

  [ "$fail" = 0 ] && echo "PASS changelog-session-key-test" || exit 1
  ```

- [ ] **Step 2: Run it; verify it FAILS** — case (a) opens `changelog-t-claude.md` (no suffix yet).

  Run: `sh tests/changelog-session-key-test.sh`
  Expected: `FAIL (a) env-keyed base: …/changelog-t-claude.md`

- [ ] **Step 3: Implement the resolution + keyed base** in `bin/pair-changelog-open`

  Replace the `base=` line (`bin/pair-changelog-open:28`) with a resolution block. `jq` is already a hard dependency elsewhere on this path; guard its presence so the fallback degrades to the legacy base rather than erroring:

  ```sh
  # Per-session keying (#63): a fresh agent session gets its own change log; a
  # resume reuses it. PAIR_SESSION_ID is exported by bin/pair when the id is known
  # at launch (claude fresh / any resume); otherwise (codex/agy fresh, where the
  # id is discovered async by pair-session-watch.sh) fall back to the per-tag
  # config that the watcher writes. No id at all → the legacy unsuffixed base.
  sid="${PAIR_SESSION_ID:-}"
  if [ -z "$sid" ] && command -v jq >/dev/null 2>&1; then
      cfg="$PAIR_DATA_DIR/config-$PAIR_TAG-$PAIR_AGENT.json"
      [ -f "$cfg" ] && sid=$(jq -r '.session_id // empty' "$cfg" 2>/dev/null || true)
  fi
  base="$PAIR_DATA_DIR/changelog-$PAIR_TAG-$PAIR_AGENT${sid:+-$sid}"
  ```

- [ ] **Step 4: Run the keying test; verify PASS**

  Run: `sh tests/changelog-session-key-test.sh`
  Expected: `PASS changelog-session-key-test`

- [ ] **Step 5: Wire the test into the Makefile** — append to the `test-changelog` recipe in `Makefile.local`:

  ```make
  test-changelog: $(BIN_DIR)/pair-changelog $(BIN_DIR)/pair-scrollback-render
  	sh tests/changelog-open-test.sh
  	sh tests/changelog-session-key-test.sh
  ```

- [ ] **Step 6: Confirm the pre-existing e2e still passes (legacy/no-sid path)**

  Run: `sh tests/changelog-open-test.sh`
  Expected: `PASS changelog-open-test` (it sets no `PAIR_SESSION_ID` and writes no
  config → resolves to the unsuffixed base — case (e) — so its `changelog-t-claude.*`
  assertions are unchanged).

- [ ] **Step 7: Commit**

  ```bash
  git add bin/pair-changelog-open tests/changelog-session-key-test.sh Makefile.local
  git commit -m "#63: key the change-log base on session_id (opener + fallback)"
  ```

### Task 2: `bin/pair` exports `PAIR_SESSION_ID` at the launch chokepoint

**Files:**
- Modify: `bin/pair` — add one export just before the zellij launch (after the codex `--no-alt-screen` block at `bin/pair:2128`, before the dev-rebuild at `:2166`).

- [ ] **Step 1: Add the export.** Both source ids are plain (non-`local`) script-body vars in scope at this point: `$explicit_resume` (set at `:1993/1997`, also on the Alt+n restart re-exec) and `$new_sid` (set in the claude block at `:2065`). codex/agy fresh leaves both empty → empty export → consumers fall back to the config.

  ```sh
  # Export the resolved session id (when known at launch) so the change log keys
  # its file set per-session (#63): a fresh session opens an empty log, a resume
  # reopens the same growing one. claude-fresh mints it ($new_sid) and a resume
  # pins it ($explicit_resume); codex/agy fresh sessions discover the id async via
  # pair-session-watch.sh, so it is empty here and pair-changelog-open + the draft
  # nvim watcher fall back to reading .session_id from the per-tag config.
  export PAIR_SESSION_ID="${explicit_resume:-${new_sid:-}}"
  ```

- [ ] **Step 2: Shell-lint the change** (the script is shellcheck-clean; keep it so).

  Run: `command -v shellcheck >/dev/null && shellcheck -x bin/pair || echo "shellcheck not installed — skip"`
  Expected: no new warnings attributable to the added line (the `:-` defaults guard unset vars under `set -u`).

- [ ] **Step 3: Sanity-check scope + ordering** — confirm the export sits AFTER the two id-producing blocks and BEFORE the `zellij --new-session-with-layout` at `bin/pair:2195`, so the value is inherited by the panes.

  Run: `grep -n 'PAIR_SESSION_ID\|^zellij\|new_sid=\|explicit_resume=' bin/pair`
  Expected: the `export PAIR_SESSION_ID=` line number is greater than the `new_sid=`/`explicit_resume=` assignments and less than the `zellij` launch.

- [ ] **Step 4: Commit**

  ```bash
  git add bin/pair
  git commit -m "#63: export PAIR_SESSION_ID at launch for change-log keying"
  ```

### Task 3: draft-nvim `.ready` watcher resolves the same keyed path

**Files:**
- Modify: `nvim/init.lua` — `pair_start_changelog_ready_watch` (`:2694-2705`); add a small session-id resolver beside it.
- Test: extend `tests/changelog-notify-test.sh`.

- [ ] **Step 1: Extend the headless flash test to the keyed path.** In `tests/changelog-notify-test.sh`, set `PAIR_SESSION_ID` on the nvim env (`:69`) and change the dropped marker (`:55`) to the keyed name, so the test proves the watcher reads `PAIR_SESSION_ID` and polls the suffixed `.ready`:

  - Line ~55: `local marker = dd .. '/changelog-test-claude-deadbeef-0000-1111-2222-333344445555.ready'`
  - Line ~69: `env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_SESSION_ID=deadbeef-0000-1111-2222-333344445555 \`

- [ ] **Step 2: Run it; verify it FAILS** — the watcher still polls the unsuffixed `changelog-test-claude.ready`, so the keyed marker never fires.

  Run: `bash tests/changelog-notify-test.sh`
  Expected: `FAIL dropped marker fires the flash` (and the consume assert).

- [ ] **Step 3: Implement the resolver + keyed marker** in `nvim/init.lua`. Add a small helper directly above `pair_start_changelog_ready_watch`, and resolve **inside the timer callback** so a late-discovered (codex/agy) id is picked up without an nvim restart:

  ```lua
  -- Resolve the change-log session id (#63): the env var bin/pair exports when the
  -- id is known at launch (claude-fresh / any resume), else the per-tag config the
  -- session watcher writes (codex/agy discover it async). Mirrors the env→config
  -- order in bin/pair-changelog-open so the polled .ready path matches the base the
  -- opener builds. (A focused reader, not pair_read_saved_config(): that one is
  -- defined later in the file and also reads the agent-<tag> file — overkill here.)
  local function pair_changelog_session_id(data_dir, tag, agent)
    local sid = vim.env.PAIR_SESSION_ID
    if sid and sid ~= '' then return sid end
    local cf = io.open(data_dir .. '/config-' .. tag .. '-' .. agent .. '.json', 'r')
    if not cf then return nil end
    local body = cf:read('*a'); cf:close()
    local ok, parsed = pcall(vim.json.decode, body)
    if ok and type(parsed) == 'table' and parsed.session_id and parsed.session_id ~= '' then
      return parsed.session_id
    end
    return nil
  end

  local function pair_start_changelog_ready_watch()
    local data_dir = vim.env.PAIR_DATA_DIR
      or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
    local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
    local agent = vim.env.PAIR_AGENT or 'claude'
    vim.fn.timer_start(2000, function()
      local sid = pair_changelog_session_id(data_dir, tag, agent)
      local base = data_dir .. '/changelog-' .. tag .. '-' .. agent
      if sid then base = base .. '-' .. sid end
      local marker = base .. '.ready'
      if not vim.loop.fs_stat(marker) then return end
      os.remove(marker) -- one-shot: consume the marker so the flash fires once
      pair_flash_notify('✓ change log ready · Alt+l')
    end, { ['repeat'] = -1 })
  end
  pair_start_changelog_ready_watch()
  ```

- [ ] **Step 4: Run the flash test; verify PASS**

  Run: `bash tests/changelog-notify-test.sh`
  Expected: `changelog-notify-test: all passed` (keyed marker fires + consumed).

- [ ] **Step 5: Commit**

  ```bash
  git add nvim/init.lua tests/changelog-notify-test.sh
  git commit -m "#63: draft-nvim .ready watcher resolves the per-session keyed path"
  ```

### Task 4: Atlas + full-suite verification

**Files:**
- Modify: `atlas/architecture.md` — the Change-log **State** bullet (`:442-448`).

- [ ] **Step 1: Update the atlas State bullet** to note the per-session keying and the resolution order. Edit the bullet to read (key edit: the base is `changelog-<tag>-<agent>-<session_id>`, resolved from `PAIR_SESSION_ID` → config, legacy unsuffixed when no id):

  > **State** (`$PAIR_DATA_DIR`, per `(tag, agent, session)` — the base is
  > `changelog-<tag>-<agent>-<session_id>`, keyed so a fresh session starts an empty
  > log and a resume reopens the same one, #63; `session_id` resolved from the
  > exported `PAIR_SESSION_ID` → the per-tag `config` JSON, falling back to the
  > legacy unsuffixed base when no id is known): `…<base>.md` (the log…), `.anchor`,
  > `.cleaned`, `.status`, `.ready`, `.openlock`, `.distill.lock`.

  Also add `tests/changelog-session-key-test.sh` to the Tests sentence (`:450-454`).

- [ ] **Step 2: Confirm `atlas/index.md` already links `architecture.md`** (no new file added).

  Run: `grep -q architecture.md atlas/index.md && echo OK`
  Expected: `OK`

- [ ] **Step 3: Run the full suite** (the gate is `make test`, not a subset — per lessons #57).

  Run: `make -f Makefile.local test`
  Expected: all sub-targets pass, including `test-changelog` (now two scripts) and
  `test-statusline` (the extended notify test).

- [ ] **Step 4: Commit the atlas update**

  ```bash
  git add atlas/architecture.md
  git commit -m "#63: atlas — change-log state is keyed per session_id"
  ```

---

## Done when (mirrors the issue)

- A fresh session (Alt+Shift+N) opens an **empty** change log; a resume (Alt+n)
  reopens the **same growing** one — because a different `session_id` is a
  different file and the same `session_id` is the same file.
- `pair-changelog-open` and the draft-nvim `.ready` watcher resolve the id the
  same way (`PAIR_SESSION_ID` → config) and build the same base.
- `make test` green; `tests/changelog-session-key-test.sh` covers fresh-vs-resume
  + config-fallback + legacy; `changelog-notify-test.sh` covers the keyed watcher.

## Out of scope (noted, not built)

- Reaping old `changelog-…-<sid>.*` files (one log per conversation now accrues;
  strictly better than one unbounded file).
- `/clear` rotating claude's live id mid-session (pair keeps the launch id —
  atlas:479 gap); a `/clear`-fresh change log is a separate enhancement.

## Accepted caveat — one-time migration discontinuity

The first resume *after this ships* re-keys to the suffixed name, so any pre-#63
`changelog-<tag>-<agent>.md` is orphaned (no longer read). This is harmless —
arguably desirable: that old per-tag file is exactly the accreting pile #63
targets. It is adjacent to the deferred "reaping" item above; no migration step.

## Revisions

### 2026-06-17 — plan-quality judge notes folded in (verdict INFO, pre-implementation)
- **Test the nvim config-fallback branch (judge note 1).** Task 3's headless test
  originally only set `PAIR_SESSION_ID` on the nvim env → it exercised only the
  env branch, leaving the architecturally-motivated config-fallback (the codex/agy
  async path) unrun in Lua. Reworked Task 3 Step 1 to drive **all three** watcher
  branches in one boot by mutating `vim.env.PAIR_SESSION_ID` from the driver and
  asserting **keyed-marker consumption** (the unambiguous signal the watcher
  resolved that exact path): (i) legacy/no-id → unsuffixed marker; (ii) env id →
  suffixed marker; (iii) env cleared + config written → config-resolved suffixed
  marker. Consumption (not flash text) is asserted across phases to avoid the
  ~2s flash-revert timing overlap.
- **Documented the migration discontinuity (judge note 2)** as the "Accepted
  caveat" above; also note it in the issue `## Log` at close.
