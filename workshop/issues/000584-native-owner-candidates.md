---
id: 000584
status: open
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: '17eea3b7ad2b5aac277b975b2142a44f7a95ad0a' # card fields mirrored from issue-cards; edit via sdlc
---

# Native owner scan blocks fleet resume on unrelated processes

## Problem

#556 native-owner admission blocks Resume from disk fleet-wide on macOS. It requires command evidence for every process before filtering Pair wrappers. Real live root services return EINVAL for kern.procargs2 and are omitted from the command snapshot; zombies and process-table churn also leave metadata without command evidence. Two read-only probes reproduced the exact refusal with 205–207 missing entries. ariadne:5 is parked with a proven native binding but cannot resume.

## Spec

Filter native-owner candidates from kernel metadata before requiring argv; unrelated processes and kernel-proven zombies do not block. Missing metadata or unreadable argv for a possible Pair owner remains uncertainty, never proof of absence. Fence candidate generation/parent changes across observations, with one bounded re-observation and refusal if still unstable. Preserve orphan/native-ID collision checks and exact-ID admission.

## Done when

- An unreadable unrelated root service and kernel-proven zombies cannot prevent proven Resume from disk.
- A possible Pair owner's missing argv or identity still refuses with actionable process evidence.
- Candidate process churn gets one bounded retry; stable re-observation succeeds and persistent churn refuses.
- Fake process-table regressions prove these behaviors and existing native-owner collision protections; required recorded suite and build pass, close review is SHIP.

## Plan

- Add failing process-table regressions at the ownership observation boundary.
- Filter candidates before argv certainty and retry unstable candidate observations once.
- Run targeted tests, host read-only observation, recorded suite and build; close and land on SHIP.

## Log

- 2026-10-15: TL authorized this fleet-blocking regression repair from main. Existing source at e1311121 and 6d65c4c4 is identical on the failing path. PID 32793 endpointsecurityd is live (exact identity 1792103969.915933), native argv EINVAL with successful identity; five other macOS services match. Eighteen kernel-confirmed zombies also exist. No production effects during diagnosis. ARCH-PURPOSE: certainty belongs to possible native owners, not every OS process; ARCH-ORDER: unknown owner evidence is not absence.
