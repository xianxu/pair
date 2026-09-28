---
id: '000140'
status: wontfix
created: 2026-08-16
updated: 2026-08-17
---

# Muse Return rewrite only in composer

## Problem

Muse currently participates in the pair Return remap convention, while
selection menus and permission prompts depend on known visible markers for
overlay bypass. Codex now uses a safer rule: rewrite plain Return only when
Pair positively identifies the live composer/input box. Muse should follow the
same contract so AskUserQuestion/request_user_input menus and future UI
variants can use plain Return to select without Pair having to enumerate every
menu footer.
