---
id: 000290
status: open
deps: [ariadne#238]
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
---

# atlas: migrate to purpose split (map / journeys / workflow)

## Problem

ariadne#238 splits atlas by purpose:
- **the map** (`atlas/` root and feature folders): short pointers, terminology,
  design reasons
- **user journeys** (`atlas/journeys/`): steps in the user's words plus an
  interruption table of current behavior
- **workflow** (`atlas/workflow/`)

It also sets a sorting rule for existing content: user-visible behavior goes to
`journeys/`, an invariant goes to `workshop/targets/`, a pointer or design reason
stays in the map (short), and prose that restates the code is deleted.

This repo's atlas predates that split.

**Survey (2026-09-18).**
- **Size:** 10 own pages, about 5,100 lines. `atlas/workflow/` is ariadne's
  symlink.
- **Mostly internals:** `couch.md` (1,882 lines), `architecture.md` (1,242
  lines, mostly per-feature internals), `session-identity.md`, `terminal.md`,
  `storage-retention-io.md`, `review-workbench.md`.
- **No page is written from the user's side.** User-facing fragments: the four
  exit chords in `architecture.md` ("Quit / restart semantics"), and
  `review-workbench.md` "The loop", where half the steps are the agent's.
- **Vocabulary:** mostly internal terms ("incarnation", "ThreadStore CAS",
  "nonce-bound park transaction"). couch.md's Terminology section defines
  internal nouns, not what the user deals with. "operator" appears 70 times,
  "user" 63.
- **Unclear current state:** couch.md's "Planned, not built" (couch-lite rescope)
  leaves the current behavior unclear.
- **Settled rules, scattered:** "Alt+x is the only full-quit path"; Y/N
  confirmation before teardown; "leave is unconditional"; "the user is never
  stranded". These go into the always-on product section (ariadne#236).
- **Closest thing to user docs:** `README.md` (862 lines), plus in-app help
  (`bin/pair-help`, `cmd/internal/keyhelp/`, `couchkeys`).

**Central journeys:** first launch (`pair <agent>` → picker or name → workbench
→ draft → send); the daily loop (draft, send, history and queue, scrollback,
change log); leaving and coming back (detach, quit, resume, continue, restart);
couch (switcher, start, switch, park, detach, relaunch, leave, notifications);
review from the user's side. Interruptions that need rows: agent crash mid-turn,
zellij server death, closing the terminal without Alt+x, two sessions on one
tag, a second couch, pair and couch both claiming a tag, reboot, GC removing
data, a send truncated mid-way (#211).

**Contradiction:** `atlas/architecture.md` says the resume picker "offers up to
three options"; `README.md` "Resume a session by tag" lists four.

## Spec

Apply ariadne#238's convention (AGENTS.md §8; `datatype show journey`):

1. **`atlas/index.md` sections by purpose:** Map, Journeys, Workflow. Every file
   stays linked.
2. **`atlas/journeys/`:** one `journey` page per central journey, listed above,
   in user vocabulary. For each interruption row, say what the user sees today.
   Mark a row *undecided* when no source (code, README, help text, tutorials)
   settles it, and list the undecided rows in the Log for the operator. A repo
   with no user surface can say so in `index.md` and skip this step.
3. **Sort each existing page, section by section**, using the rule above. Move
   invariants into `workshop/targets/`, cut map pages to pointers, and delete
   prose that restates the code.
4. **Fix the contradictions** listed above, and any others found while sorting,
   against the code, which is the source of truth.

## Done when

- `atlas/index.md` is organized by purpose and links every file.
- `atlas/journeys/` covers the central journeys (or `index.md` says why there are
  none). Undecided rows are listed in the Log.
- No map page restates code at length; each moved section is noted in the Log
  with its destination.
- The listed contradictions are resolved.

## Plan

- [ ] Journeys: draft, mark undecided rows, get the operator's review.
- [ ] Sort existing pages and fix contradictions.
- [ ] Rebuild `index.md` by purpose.

## Log

### 2026-09-18

Filed from a brain advisor session as one of the per-repo migrations under
ariadne#238. Order: ariadne#238, then these migrations, then `prd` (ariadne#237).
