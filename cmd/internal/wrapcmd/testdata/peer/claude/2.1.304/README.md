# Claude 2.1.304 visible-pane resize control

`tab-wrap.raw` and `tab-wrap.events.json` preserve an authentic idle Claude
94x39 → 94x38 → 94x39 sequence from ops:0 on 2026-10-22. The test maps those
body heights to the real status-bar planner's one → two → one row layout.
This is positive replay coverage; it does not reproduce the intermittent live
stall. The resize-then-silence test deliberately withholds the following native
repaint to demonstrate why delivery then remains refused.

Source: local repo scope aef274ee2fbbcabd, tag1-ops-71, raw range
[56954443,56968250), between output markers13:20:24 and13:23:13 PDT.
Resize events carry byte offsets, not timestamps; those markers cannot prove
post-resize elapsed silence or keystroke timing. No agent input was recorded.

Sanitization retains native CSI controls, cursor geometry, and the final
composer, bypass-permissions footer and update banner. Transcript glyphs are
replaced with x at the same display width; zero-width combining glyphs are
removed. OSC payloads are also obfuscated with x, retaining semicolon structure.
No repaint or show-cursor controls were fabricated. Source offsets are included
in the event fixture to distinguish native source bytes from sanitized offsets.

SHA256:
- Source raw:39d5056e4eb25b565c0bc053daa1f00d0664811724ec6465964a9ebcf0a0576d
- Sanitized:8be58ead01198b66fad6218b90412dc9fc4a1c0685396931e7642e2bf1c1c747
