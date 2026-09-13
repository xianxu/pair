---
id: 000246
status: open
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
---

# Adopt a stopped standalone Pair session into Couch

## Problem

An operator may start Pair standalone, later quit it, and want to continue that
same session as a Couch-managed thread. Creating a new Couch thread at the same
path does not express adoption of the existing session and its history.

## Spec

Feature proposal: provide an explicit Adopt Pair action in Couch. The operator
selects an exact stopped standalone Pair session; Couch registers and resumes
that session as a managed thread, preserving its Pair address, native agent
binding, draft, and available history. A path may narrow candidate discovery but
must not silently choose a session when several share the path.

Initial scope is cold adoption after standalone Pair has quit, not takeover of
a live attached or detached process. Validate the session is stopped and has
an established resumable native binding and reconstructible launch profile.
Uncertain ownership or missing evidence must produce an actionable refusal,
not a fresh conversation masquerading as a resume.

Pair continues to own its address claims and session artifacts; Couch owns the
new thread registration. Preserve the existing address rather than copying or
renaming artifacts to a newly allocated couch tag, subject to design validation
of existing namespace assumptions. Repeated adoption resolves to the same
managed thread. Concurrent adoption/resume and interrupted registration must
not create duplicate owners or overwrite source history.

Current Couch cold resume requires VerifiedPark from a Couch-owned lifecycle
transaction. A standalone quit is not that evidence. Design an explicit
adoption transition with validated Pair evidence; do not fabricate a historical
Couch park or bypass normal resume guards. Reuse existing resume/attachment
machinery after establishing valid adoption authority (ARCH-DRY, ARCH-ORDER,
ARCH-SECURE). Enumerate state and failure recovery before implementation.

UI placement and the exact candidate inventory remain design questions. The
desired experience is similar to creating a thread, but selecting an existing
session instead of starting a new conversation.

## Done when

- A stopped standalone session can be selected and resumed inside Couch with
  the same native conversation, Pair artifacts, and working path.
- The resulting thread supports normal Couch lifecycle operations.
- Duplicate attempts reuse the existing registration; ambiguous, running,
  malformed, or unresumable candidates are refused without modifying history.
- Tests cover discovery through actual attachment, concurrent ownership races,
  interruption/retry, and preservation of source artifacts using portable state.
- Operator docs describe adoption and distinguish it from new-thread creation.

## Plan

- [ ] Validate standalone quit evidence, launch-profile recovery, and address compatibility.
- [ ] Settle the adoption UI and durable transaction design with the operator.
- [ ] Author the durable implementation plan and implement through SDLC gates.

## Log

### 2026-09-13

Recorded from operator proposal during Couch switching discussion. Inspected
atlas/session-identity.md, atlas/couch.md, couchcore/threadtag.go and
couchcore/resume.go: standalone Pair does not write ThreadStore; Couch allocates
new reserved addresses and cold resume requires VerifiedPark. No existing
adoption issue found among active issues. No implementation started.

Related #245 proposes passing all Pair shortcuts through the focused agent,
including Alt+X. Adoption depends on a clean stopped session, not that specific
key: its documentation must describe whichever quit entry point remains valid.
