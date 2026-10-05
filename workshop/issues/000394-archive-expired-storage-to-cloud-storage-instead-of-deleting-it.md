---
id: 000394
status: open
deps: []
github_issue:
created: 2026-10-05
updated: 2026-10-05
estimate_hours:
card_mirror: 'e7494709e12ea43725e21702f0a3f0c82893aff2' # card fields mirrored from issue-cards; edit via sdlc
---

# Archive expired storage to cloud storage instead of deleting it

## Problem

Pair's retention collector (#239, `cmd/internal/storagegc`) deletes data once its
retention period has passed. #393 raised every period to a year so that nothing is
lost in practice, but that only postpones deletion and lets local disk grow. The
operator wants expired data kept in cloud storage instead of deleted.

## Spec

(Requirements; the design is open.)

- When the collector decides an item is `Eligible`, it uploads the item to cloud
  storage and removes it locally only after a verified upload. A failed or uncertain
  upload leaves the local copy (it's in the quarantine transaction already,
  `.retention/quarantine/<id>`, so that is the natural place to hook in).
- The liveness and ownership proofs stay exactly as they are; archiving changes what
  happens after the decision, not the decision.
- Restoring is a documented path: list what was archived and pull an item back to
  its original artifact path.
- Agent-agnostic and with no hardcoded paths: the destination is configuration, and
  with no configuration the behaviour is what it is today.

Open questions for the operator:
- Which provider? The operator's frame (2026-10-05): storage is cheap, e.g. Google
  AI Pro includes 10 TB of Google Drive for about $20/month, so cost isn't the
  constraint; keeping everything is the goal. That points at Google Drive first,
  most simply through a locally synced Drive folder (Drive for desktop), so pair
  only writes to a directory and needs no API credentials. A pluggable "archive
  sink" seam (local directory first; rclone/API later) keeps it provider-neutral.
- Encryption at rest and in transit. Captures and drafts can contain secrets, so
  client-side encryption (e.g. age/GPG, like the brain's recipient list) is likely
  required.
- Should the retention periods go back down once archiving is trusted (e.g. session
  60d, captures and logs 7d), so local disk stays bounded?
- Do the diagnostic logs (`diagnosticlog`) archive too, or only storagegc families?

## Done when

- (Settled after the questions above.) The collector archives instead of
  deleting through a sink seam, with a stateful fake sink in tests, covering
  upload-then-remove, failed upload keeps local, and restore round-trip.

## Plan

- [ ] Settle the open questions with the operator (provider, encryption, whether to lower the periods again)

## Log

### 2026-10-05

- Filed from the operator's request after #393 ("instead of deleting, archive it
  in cloud storage").
- Operator note: storage is cheap (Google AI Pro: 10 TB for about $20/month), so
  archiving everything instead of deleting anything is the intent.
