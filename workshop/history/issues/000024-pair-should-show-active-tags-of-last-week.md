---
id: 000024
status: done
deps: []
created: 2026-05-27
updated: 2026-05-27
related: [bin/pair]
actual_hours: 0.55
---

# pair should show active tags of last week

right now, pair, upon start, only shows tag based on current working directory, and if that's taken (have an active session), append a -1, -2.

we should actually present user with tags 1/ inferred from current working directory; 2/ tags used in the last 14 days inside this current directory.

## Done when

- Launching `pair` in a cwd whose basename is `<base>` shows the existing detached-session picker, *plus* any tag of the form `<base>` or `<base>-*` that has been touched (draft / log activity) within the last 14 days but doesn't currently have a live zellij session.
- Picking a historical row resumes by name (existing `pair resume <tag>` path); the prior agent + draft + session-id come back when their sidecars survive.
- No new state file. Convention-following: the cwd-prefix rule is the existing "tag prefix matches cwd basename → tag is about that repo" convention made discoverable. Tags that don't follow the convention (e.g. an explicitly typed `blogging` in cwd `pair`) are not surfaced; that's an accepted limitation.

## Spec

**Discovery (no new persistent state).** After the existing `detached_list` is built, walk `$DATA_DIR/` looking for filenames matching either `draft-<base>(-*)?.md` or `log-<base>(-*)?.md` where `<base>` is `basename "$PWD"` (sanitised the same way `default_tag` is — `[^A-Za-z0-9_-]` → `_`). For each matched file, extract `<tag>` (the part between the prefix and the extension) and the file's mtime. Aggregate per-tag as `max(mtime over its draft/log files)`. Keep only tags whose aggregate mtime is within the last 14 days (configurable via `PAIR_HISTORY_DAYS`, default 14).

**Dedup against the live set.** Subtract:
- Tags currently in `detached_list` (already shown as live rows).
- Tags whose `pair-<tag>` zellij session is alive and attached (not in `detached_list`, but in `all_pair`).
- Tags that match the computed `free_slot_tag` (the "+ new" sentinel already covers the most-recently-free slot).

**Picker rows.** Existing rows stay byte-identical. Add historical rows formatted as `pair-<tag>  (Nd ago, no live session)` where N is the floor of `(now - mtime) / 86400` (special-cases: `0d ago` → `today`, `1d ago` → `yesterday`). Right-align the annotation if fzf width allows; otherwise plain trailing text — fzf still matches on the prefix typed by the user.

**Picker order.** Detached-live rows first (newest mtime first by `pair-wrap-pid-<tag>` if available, else lexical), then historical rows newest-first, then the `+ new <BASE_TAG> session` sentinel last.

**Pick behavior.** When a historical row is picked, treat it as if the user typed `pair resume <tag>` — set `chosen_tag` from the stripped row, set `action=create` (since the zellij session is gone), and let the existing create path pick up the saved `config-<tag>-<agent>.json` and `draft-<tag>.md` if present. No new branch needed.

**Failure modes.** Anything that goes wrong in the scan (data dir missing, weird filenames, mtime not readable) silently produces no historical rows — fall back to today's picker. The feature is purely additive; the picker still works the old way if discovery yields nothing.

## Plan

- [x] M1: implement the scan + format helper in `bin/pair`. Pure-ish function: input is `$DATA_DIR`, base, `now`, the active live-tag set, and `PAIR_HISTORY_DAYS`; output is a newline-delimited list of historical tag rows (already annotated). Add a hidden `pair --debug-historical` (or env-gated probe — `PAIR_DEBUG_HISTORY=1`) that prints the discovered set + ages so the user can sanity-check on their actual `$DATA_DIR`.
- [x] M2: wire the helper into the picker — append historical rows after detached rows, before the `+ new` sentinel; update the `picked == new_label` check to also recognise a historical row (strip the annotation, set `chosen_tag`, set `action=create`).
- [x] M3: verify on a real `$DATA_DIR` — helpers verified in isolation against a fixture data dir (three cases: in-window match, widened-window match, prefix-mismatch). End-to-end user-confirmed in `~/workspace/brain` — picker shows historical tags as expected. In `~/workspace/pair` the dedup is correct: the only matching tag (`pair`) is the currently-live session, so the historical list is empty and the create flow proceeds straight to the name prompt.
- [x] M4: atlas — extend the "## `bin/pair` — launcher" decision-tree paragraph to mention the historical-tag surface. One sentence.

## Log


- 2026-05-27: closed — operator-verified end-to-end in ~/workspace/brain (picker surfaces historical rows as designed); dedup correctness confirmed in ~/workspace/pair (only matching tag hidden as currently-live). Force used: all four milestones landed in a single commit (055fec0) — no per-milestone commit boundaries for separate Review-Verdict trailers
### 2026-05-27

- 2026-05-27: filed and scoped. Convention-only design (no new state file); discovery via mtime walk of existing `draft-<tag>.md` / `log-<tag>.md` filtered by cwd-prefix and 14-day window. Operator-side convention reminder: name tags as `<cwd-base>-<subproject>` so historical discovery surfaces them in the right context.

- 2026-05-27: M1-M2 landed in `bin/pair` — `scan_history` walks `$DATA_DIR/draft-*.md` + `log-*.md`, buckets by tag via awk, keeps max-mtime per tag, sorts newest-first; `format_age` renders `today` / `yesterday` / `Nd ago`; the build loop annotates rows as `pair-<tag>  (Nd ago, no live session)` and dedups against live tags + `free_slot_tag`. Picker now triggers when **either** detached_list **or** historical_rows is non-empty, threading historical rows between detached rows and the `+ new` sentinel. Pick-handler trims the double-space-annotated tail to recover the bare `pair-<tag>` for both row shapes, and routes the historical pick through the existing `create-by-name` path (which already absorbs any surviving `draft-<tag>.md` / `config-<tag>-<agent>.json`). `HISTORY_BASE` is anchored on `basename "$PWD"` independent of `BASE_TAG` so `pair codex` in cwd `pair` still surfaces this dir's history. Env knob: `PAIR_HISTORY_DAYS` (default 14). Debug probe: `PAIR_DEBUG_HISTORY=1 pair` prints scan inputs + matched rows and exits 0 before launching zellij.

- 2026-05-27: M3 verified in isolation. Fixture data dir with five files (today / 5d / 20d / unrelated tag) exercised three scan cases: window=14d surfaces today + 5d, skips 20d (out-of-window) and unrelated (prefix-mismatch); widening to 30d pulls in the 20d row; changing the base to `unrelated` shows only that one tag. M4 atlas note added — decision-tree paragraph now mentions the historical-tag surface, the `PAIR_HISTORY_DAYS` knob, and the `PAIR_DEBUG_HISTORY=1` probe.

- 2026-05-27: end-to-end confirmed by operator. In `~/workspace/brain`, where multiple tag sidecars exist within the 14-day window, the picker correctly surfaces the historical rows. In `~/workspace/pair` only the live `pair` session matches, dedup hides it, so the create flow runs straight through to the name prompt — also correct.

- 2026-05-27: side-finding (out of scope, separate issue worth filing): `zj --session pair-<tag> action list-clients` exits 1 with "There is no active session!" when invoked from a process tree inside that same session's panes. With `set -euo pipefail`, the failing pipeline trips errexit and `bin/pair` silently exits during `detached_list` build. Doesn't affect any production launch path (pair is normally launched from a fresh terminal), but it does block self-launches and trace-style debugging from inside an active session.

