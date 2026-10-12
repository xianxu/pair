---
id: 000466
status: open
deps: [000457]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '4e8d7a6863519a3a28fed2b90f85eabf360c485f' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: remote-driving contract — viewer input, driver control, attribution in the format

## Problem

Remote driving is in couch-broadcast-web's MVP for internal viewers (the operator, 10-12): a product manager uses an app hosted by another operator, for product design and live debugging. Outside users (research participants, customers) stay later, with the privacy work. pair#457's format and seams are one-way (operator's Chrome → viewers): no reverse channel, no input injection, no attribution of who acted. This issue extends the contract before the components that need it are built (ops `teams/devinfra/projects/couch-broadcast-web/project.md`, Later requirements: remote driving, now MVP).

## Spec

Design and contract only, as an additive v1 revision of `atlas/webcast-format.md` and #457's Go seams (no component implementation):
- **Viewer → Couch input messages:** element-addressed (node id) plus page coordinates for pointer actions (click, double click, pointer down/up/move while dragging, wheel/scroll), and keyboard input (text, keys, composition) to the focused element; scoped to the page: never browser UI (address bar, downloads, file pickers, devtools, new windows, permission prompts).
- **Driver control:** one driver at a time; a viewer requests control, the operator grants (and can revoke at any time); the driver gives it back. Who may request: anyone with the viewer link can watch; driving needs the operator's grant per viewer.
- **Attribution in the recording:** context records (`click`, `input`, `focus`, `hover`, …) gain a source (operator or viewer id), and control changes are records, so an agent can tell a participant's actions from the operator's.
- **Seams:** an input-injection interface on the browser-control side (`webcast`, CDP `Input.dispatch*` into the cast tab) with a fake; the reverse channel on the live transport (viewer shell → Couch; the recorded page stays script-free).
- Security notes: injected input is confined to the cast tab's page; the operator sees who is driving in Couch and in the page.

## Done when

- The format page and seams carry the additions, with validator/reducer support and fakes; fixtures show a driven segment.
- pair#458/#460 and the remote-driving build issue can implement against it without asking.
- The TL reviewed it (the operator if a decision is theirs).

## Plan

- [ ]

## Log

### 2026-10-12
