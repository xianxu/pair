---
id: '000047'
status: done
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1.5
actual_hours: 2
---

# pair-doctor stale-emitter-binary probe

## Problem

pair-doctor reads `adapt-<tag>.jsonl` and reports what's in it — but it's **blind
to why the log is thin**. The motivating incident (the thread that produced #46):
the flight recorder went silent for every Go-emitted aspect (1/2/4/5) while only
nvim's Lua emitter (aspect 7) logged. Root cause was not drift but a **stale
`pair-wrap`/`pair-slug` binary** — the installed copy predated #000045, so it had
no adapt-logging code at all. The doctor faithfully showed "only prompt-search
fired" and could not say *the emitters themselves are stale*. A human burned real
time concluding the telemetry was broken.

This is ironic: the doctor's whole job is making silent drift observable, yet it's
blind to the silent staleness of its own emitters — the exact failure class #46
(`pair-dev`) addresses at launch time. The doctor should be able to *diagnose* the
condition `pair-dev` *prevents*.
