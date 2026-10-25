# Adapter: editing the file

You answer a round (`../round.md`) by changing the document on disk. Which
path you take depends on whether Couch tracks the document. Never edit a
tracked document directly: the operator's lock and the compare-and-swap below
are what keep your write from overwriting their unsaved work. Every `couch`
call is the whole shell command (AGENTS §15).

**Temporary files.** Each temporary file below states when you delete it. All
of them live under `$TMPDIR`, so one you could not delete (a refused tool
call, a lost conversation) is left in scratch space, not in the repository.

## Is the document tracked?

Decide in this order, and edit directly only on a confirmed **untracked**:

1. Frontmatter `type: session-sync`: **tracked**, whatever the registry says
   and wherever you are (Couch routes these through its own storage without a
   registration). If Couch can't be reached (outside a Couch slot, or a
   failing call), write nothing and tell the operator; never fall back to a
   direct edit.
2. Outside a Couch slot (`COUCH_TREE` unset): **untracked**.
3. `couch --session-sync lookup FILE --json` exits 0 with Code `ok`:
   a non-null Registration is **tracked**; an explicit `null` Registration is
   **untracked**.
4. Anything else (an error, a malformed answer, a usage message from an older
   Couch) is **unknown**: write nothing, and tell the operator the lookup
   failed. Unknown ownership is never permission to edit directly.

## Routing decisions

New documents are always highlight (`../collaboration-modes.md` §1). The
propose rows below and the propose passages in this file describe Couch until
pair#838: it still converts edits to proposals in an older document whose
legacy `presentation-mode: propose` it honors. Never write that key, and follow
these rows only when Couch reports such a document.

Apply this table to **every existing document changed by the round**, including
the submitted file and any parent. The action is the body-write path; a tracked
frontmatter-only exception remains in the Tracked section below.

| Case | Action | Reply evidence |
|---|---|---|
| tracked direct/highlight (submitted or parent) | apply | round spans |
| tracked propose (submitted or parent) | apply | round spans |
| untracked direct/highlight (submitted or parent) | file tools | observed diff |
| untracked propose (submitted or parent) | refuse | track or Pair workbench |
| unknown owner (submitted or parent) | refuse | lookup error |

## Tracked

One round reviews one snapshot: the text whose hash you write against.

1. **Owed events:** `couch --session-sync status --json FILE`. Keep the IDs it
   lists as owed, and the contents of those deliveries (the operator's
   submits): they are the asks this round answers. A `delivery-gap` or
   `not-recording` refusal means owed events are unknown, not zero: follow the
   refusal's advice before writing.
2. **Changes since your last read:** `couch --session-sync diff FILE`. Couch
   moves its baseline on your write, so the diff never holds your own edits
   (`couch --skill`: "anything outside what you last wrote or read with
   `--all`"). It must come before step 3, which marks everything seen and
   empties it.
3. **The snapshot:** `couch --session-sync read --all FILE` prints the text and
   its hash H. This text is the round's input: its `@@chat@@` tags and any legacy
   mode keys come from it (`../collaboration-modes.md` §3, §7), and the markers from its body. The diff only points at what
   moved; an edit made between steps 2 and 3 is in the snapshot even though
   the diff missed it, so read the markers from the snapshot, not the diff.
4. **Compute** the clean result (`../round.md`) on the snapshot. For every
   change to the body, write `{old, occurrence, new, explain}` records to an
   EDITS.json file made with `mktemp "${TMPDIR:-/tmp}/review-edits.XXXXXX"`.
   `old` is nonempty; `occurrence` counts plain, non-overlapping matches from
   1 (omit it for the first); to insert, repeat nearby text in `new`. Write
   each `explain` for the operator: say why that edit helps, not what the
   record syntax does. Resolve every record against this one snapshot.
5. **Apply:** `couch --session-sync apply EDITS.json FILE --if-hash H --fold ID[,ID] --json`.
   Fold only IDs from step 1 whose asks this edit answers. A kept delivery you
   didn't answer, and one arriving after step 1, stay owed. With no IDs,
   leave out `--fold`. Delete EDITS.json after success or refusal. Couch
   applies the entire batch or none of it, converts proposals when the
   edit's section resolves to `presentation-mode: propose`, and derives the round. Never type
   proposal-marker syntax into `new`. A record replacing the operator's own
   ready request marker with a side-doc link, or answering a `🤖[fold in]` at
   that link, applies directly even under `propose`; the engine checks the
   marker and link in the base text. Ordinary prose edits still become proposals,
   even when they answer a ready marker. When the link opens a side doc, include
   its matching `discussions:` frontmatter entry in the same batch as the link.
   Check that both paths match; Pair lands the batch atomically. Do not split
   them into two writes. Pair does not verify that path pairing for you.
6. **Report the landed edits:** `couch --session-sync round FILE --json` gives
   the round's spans and their `explain` reasons; list those edits in the
   reply. A no-op answer can use `write --all` with the unchanged snapshot
   solely to fold its owed IDs; it makes no body edit. Generated content and
   frontmatter-only changes use their `write` verbs, whose changes also get
   a derived round. For a standalone frontmatter-only change under
   `presentation-mode: propose`, use `write --frontmatter`: Pair refuses markers
   inside frontmatter. The atomic link/index `apply` batch above remains required
   when frontmatter accompanies a side-doc link; keep its metadata literal, not
   wrapped in proposal markers.

Refusals:

- **Exit 3, the operator holds the lock:** they are still editing. Don't wait
  in a polling loop or force it. Say plainly that you are waiting for the
  operator's next submit. Keep the pending ask, and answer it after that submit,
  starting again from step 1. The ask stays yours to answer even if
  that round's diff is empty. If `status` shows the lock held for more than 30
  minutes, ping the operator in one line in your pane (for the TL's session
  file, a Needs you item instead).
- **Exit 6, stale:** the document changed since your read. Go back to step 1,
  refresh every input, and rebuild every record on the new snapshot. Do not
  replay records based on the old bytes.
- **Exit 1, `edits-invalid`:** `Invalid` lists the bad record indices and
  reasons (`empty old`, `not found`, `overlap`); malformed JSON has an error
  instead. Nothing was written. Fix the listed records and retry the full
  batch, never a partial subset.
- **Exit 1, `propose-too-large` or `propose-unencodable`:** Pair could not
  express this rewrite as short markers. Write the intended new text in a
  side doc proposal (`../collaboration-modes.md` §4), then use a fresh
  snapshot and `apply` to link it in place as `→ [proposal](…)`. The operator
  can later ask `🤖[fold in]` beside that link. Without a ready operator marker,
  Pair writes the link as a pending proposal (`🤖{ → \\[proposal\\](…)}`), while
  the matching `discussions:` entry applies directly in that same batch. Treat
  the pending link as linked: do not create another side doc or answer. If the
  operator rejects it, remove its index entry in the next round. Do not
  hand-write markers.
- **Exit 1, `operator-owned`:** the batch tried to change an operator-owned
  mode key (`collaboration-mode`, `presentation-mode`, `processing-mode`, or a readable legacy alias)
  or a section's mode-comment line. These choices belong to the operator. Leave the
  document unchanged and ask the operator to set the key; do not retry that
  edit through another verb.

If an apply response is lost, read the file and its round before retrying;
an exit code you did not receive is not proof that nothing changed. If the
round read itself fails after a successful apply, name that uncertainty in
the reply rather than inventing a list of edits.

## Side docs and a parent

A round that answers a question or a chat item may create side docs, and a
fold-in changes a parent document (`../collaboration-modes.md` §4 orders the writes). A new side
doc keeps the existing create/write path: no one holds a lock on a file that
doesn't exist yet. Every existing document the round changes, the submitted
one and a parent alike, takes its own path above or below by its own tracked
or untracked lookup. A tracked parent gets its own snapshot (`read --all`, its
hash) and its own `apply --if-hash`, with no `--fold`: its owed events belong
to its own rounds. On exit 6 (stale), read the parent again and rebuild the
fold records.
On exit 3 (the operator holds the parent's lock), end the round before the
submitted document is written, so its ask stays for the next submit.

## Untracked

Read the document's frontmatter and heading comments before any write. If the edit's section resolves to
`presentation-mode: propose`, refuse a direct edit: Couch cannot enforce proposals
on an untracked file. Tell the operator to track it as `type: session-sync` or
register it with Couch, or use a hosted Pair review workbench. Do not change
its presentation on the operator's behalf. For `direct` or `highlight`, edit
the file in place with your harness's file tools. There is no lock, so read
the file again right before writing. Save the pre-write text, then read back
the result and report the observed before/after diff; there is no Couch round
to cite for this path.

**Changes since your last read** exclude your own edits. After each write,
re-read the file and save that text, the file as you left it, to a copy made
with `mktemp "${TMPDIR:-/tmp}/review-base.XXXXXX"`; remember the copy's path,
and delete the copy it replaces. Delete the last copy when you are done with
the document: the operator says so, or you hand the slot to other work.
Next round, the changes are `diff -u COPY FILE`: the operator's edits since
you left it, never your own, and never measured against the operator's
previous send. Without a saved copy (a first read, or a new conversation),
read the whole file instead.
