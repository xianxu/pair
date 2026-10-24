# #838 operator check (about five minutes)

Pending final shared-parser pin, native qualification and hub-authorized stage.
The hub supplies the exact staged revision and build provenance. Refresh the
worker's Couch skill from that candidate; existing sessions may retain old
instructions. This check exercises the operator's usual Couch/Parley workflow.
Use a disposable document, without changing fleet configuration or real chat asks.

1. Submit a document with `presentation-mode: propose` and an ordinary paragraph
   ending in `🤖[Replace “old” with “new”.]`. Expect clean changed text and a visible
   highlight. Expect one retired-key notice and no newly inserted mode settings.
2. Add `🤖[Why choose that wording?]`. Expect a linked side document answering
   the question; the main paragraph stays intact. An unclear command should also
   produce a question rather than a guessed change.
3. Add this separate section and submit it:

   ```markdown
   @@chat@@
   ## Questions
   1. Why keep the bound?
   2. Change “old label” to “new label” in the paragraph above.

   This prose is outside the list. 🤖[What does the bound protect?]
   ```

   Expect each numbered ask preserved with an answer link, the requested command
   carried out, and the prose marker handled as an ordinary question. The tag and
   heading stay byte-exact; new edits remain highlighted.
4. Open a saved historical direct/propose round supplied by the existing golden
   round fixtures. Its existing rendering remains unchanged. This step may be
   checked in Parley's round fixture viewer if the live document has no old round.

Report PASS or the first mismatch and candidate revision to the hub. Clear only
#838's stage lease after acceptance; no live document is migrated automatically.
