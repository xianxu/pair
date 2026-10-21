---
id: 000770
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# Preserve ptychild Darwin PTY final output and endpoint lifetime

## Problem

#768 audited ptychild.Start, used by Couch PtyRunner and termcmd and confirmed a generic-command PTY owner exposed to Darwin last-slave-close final-output loss. #768 fixes scribe only; this owner needs its own lifetime design.

Native planning probe from #768: openpty/fork, setsid/TIOCSCTTY, child writes final-marker and closes all slave stdio, then acknowledges out-of-band before parent first read. Baseline lost the bytes; child /dev/tty open/close and parent retained-slave variants each preserved them. Retained variant reached EOF and exited7. Additional descendant probe kept an inherited slave open in a descendant ignoring SIGHUP: Darwin still delivered exact marker and EOF on leader exit while descendant was alive. These are feasibility probes, not qualification of this owner. Sources: https://bugs.ruby-lang.org/issues/20682 and Apple XNU tty_tty.c/kern_exit.c.

Termination-semantics risk: Master EOF ends terminal input before child reap and final publication. Retaining the slave changes the observable input-ended boundary; child/descendant termination and endpoint publication must remain attributable and deterministic.
