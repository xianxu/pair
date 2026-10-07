---
id: 000406
status: open
deps: [pair#395]
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: '8a593f46a9fc7bdcc552b6fcfe694ad628c66c4c' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch broadcast: optional end-to-end encryption with a key in the link fragment

## Problem

#395 streams the composed Couch screen to browser viewers over a
`cloudflared` quick tunnel. Cloudflare's edge terminates TLS, so it sees every
frame in plaintext. Deferred from #395 on 2026-10-07 so that the plaintext-over-TLS
broadcast ships first.

## Spec

- The broadcaster encrypts each SSE frame payload with a per-session key (for
  example AES-GCM, with a fresh nonce for each frame).
- The key travels in the link's `#fragment`. Browsers never send the fragment to
  the server, so the tunnel and the local server relay only ciphertext.
- The browser viewer decrypts with WebCrypto before writing to xterm.js.
- Opt-in per broadcast; the plaintext link keeps working without it.

## Done when

- With encryption on, a capture of the SSE stream at the local server contains
  no plaintext frame bytes, and the viewer renders the session normally.
- A viewer with a wrong or missing fragment key shows an error, not garbage.

## Plan

- [ ]

## Log

### 2026-10-07

- Filed from #395's design discussion.
