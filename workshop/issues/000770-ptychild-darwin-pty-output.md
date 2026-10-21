---
id: 000770
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a61a72098aafdeb057d32f68b98d9793d0b804de' # card fields mirrored from issue-cards; edit via sdlc
---

# Preserve ptychild Darwin PTY final output and endpoint lifetime

## Problem

#768 audited ptychild.Start, used by Couch PtyRunner and termcmd and confirmed a generic-command PTY owner exposed to Darwin last-slave-close final-output loss. #768 fixes scribe only; this owner needs its own lifetime design.

Native planning probe from #768: openpty/fork, setsid/TIOCSCTTY, child writes final-marker and closes all slave stdio, then acknowledges out-of-band before parent first read. Baseline lost the bytes; child /dev/tty open/close and parent retained-slave variants each preserved them. Retained variant reached EOF and exited7. Additional descendant probe kept an inherited slave open in a descendant ignoring SIGHUP: Darwin still delivered exact marker and EOF on leader exit while descendant was alive. These are feasibility probes, not qualification of this owner. Sources: https://bugs.ruby-lang.org/issues/20682 and Apple XNU tty_tty.c/kern_exit.c.

Termination-semantics risk: Master EOF ends terminal input before child reap and final publication. Retaining the slave changes the observable input-ended boundary; child/descendant termination and endpoint publication must remain attributable and deterministic.

## Spec

Design output preservation for this owner while explicitly preserving its EOF, termination and publication contract. Compare child controlling-tty helper versus retained-slave lifecycle. Preserve direct argument semantics, status/signals, exact output and cleanup ownership. No deadline tuning or retry after EOF.

## Done when

- A native regression forces all child slave descriptors closed before first read and verifies exact final bytes, EOF and status.
- Live closed-stdio targets, inherited-slave descendants (including ignored SIGHUP), cancellation and abort have explicit tested termination semantics.
- Existing owner lifecycle consumers and executable startup errors remain correct; resources are reaped/released exactly once.

## Plan

Plan first; independently claim from main. Do not stack on #768.

## Log

Filed unclaimed per TL approval of #768 on 2026-10-21. Probe provenance and audited owner paths are in workshop/plans/000768-darwin-pty-final-output-plan.md, planning commit b55089f72c233e0532918172a29b60291f798cfb. Retention must not be generalized without resolving the termination-semantics risk above.
