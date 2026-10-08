# UTF-8-safe streaming parser

This is the streaming parser from `github.com/charmbracelet/x/ansi` v0.11.7,
with its MIT license and streaming parser tests retained. Public handler/value
types and the transition table remain upstream types. Pair's vt emulator,
terminal-control observer and output-boundary observer all use this package.

Local correction (#379): OSC/DCS/SOS/PM/APC payload bytes stay in string state.
A bounded UTF-8 prefix tracker prevents continuation bytes from acting as C1
controls, including `0x9c` in Claude's `✻` notification marker. Standalone C1 ST,
ESC-ST, OSC BEL, cancellation and reset retain their control semantics. Payload
storage still uses the upstream data cap, independently of prefix tracking.
Non-control high bytes remain opaque data; standalone C1 bytes retain their
existing transitions. Malformed
prefixes never swallow a following ASCII control.

Upstream tracking: https://github.com/charmbracelet/x/issues/848 and unmerged
PRs https://github.com/charmbracelet/x/pull/946 and
https://github.com/charmbracelet/x/pull/976 (checked 2026-10-07). The closed
https://github.com/charmbracelet/x/pull/886 instead removed bare C1 termination.
Our correction also retains SOS/PM/APC string state and validates the first
continuation range. The upstream OSC `string_terminator` test previously expected
truncation of UTF-8 `末`; its expectation is corrected here.

When upstream supplies equivalent coverage and behavior, replace this package
at all three consumers and run the compatibility, bounded-storage, observer and
captured-incident regressions before removing it.
