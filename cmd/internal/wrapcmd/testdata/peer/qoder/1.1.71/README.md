# Qoder 1.1.71 idle

Frozen from pair:4 (tag 1-pair-6), scope 5749d0ffa92b055d, on 2026-10-14.
Source scrollback-1-pair-6-qoder.raw: startup terminal-control bytes [0, 430)
plus final repaint composer/footer [27941, 29140). The omitted middle
contains startup progress and conversation; retained bytes are unchanged.
SHA256: 959b5093d43606d10130aadf77541affb8c0ed2892d39e241b106f0f9bee3083. Replay at 94 columns, 38 rows.
Contains only terminal setup, composer/footer UI and folder path.
Native submit/reply and idle settling passed the authorized #550 pair:4 smoke.

# Pending restart notice (#569)

`pending-restart.raw` is only the known restart notice's composer and its enclosing rules from pair:4's Qoder 1.1.71 scrollback, bytes [31229919,31238201). SHA256 6271e9fb9dc3faa89ab56bdff75e51220a090b9767fe68beff684a44fe4b24ad. No conversation is retained. Its relative cursor motion is replayed from row 6, then the test places the hidden native cursor back in the prompt. At 94 columns the body wraps after 'still', matching content width 91. The source starts/ends inside a synchronized update, so test framing is explicit. The pending restart was later seen consumed in the transcript; the actor/key that released it is unknown.
