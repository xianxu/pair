---
id: '000139'
status: done
started: 2026-08-17T10:59:29-07:00
created: 2026-08-16
updated: 2026-08-19
estimate_hours: 5.83
actual_hours: 23.40
---

# Agy Return rewrite only in composer

## Problem

Agy currently participates in the pair Return remap convention, while overlay
handling depends on known visible prompt markers. Codex now uses a safer rule:
rewrite plain Return only when Pair positively identifies the live
composer/input box. Agy should follow the same contract so permission pickers
and future UI variants keep plain Return as confirm instead of receiving an
accidental newline.
