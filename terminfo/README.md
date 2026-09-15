# Pair virtual terminal profile

`pair-vt-256color.ti` is the child terminal description, not a description of
the physical parent terminal. `terminal.Profile.TerminfoSource` derives it from
the same capability table used for XTGETTCAP. The profile test compares the
committed source, compiles it with `tic -x`, reads it with `infocmp`, and checks
literal `tput` results. Packaging must compile this entry ahead of time and set
`TERMINFO` to the installed directory; terminal runtime must not invoke `tic`.

The initial contract deliberately does not inherit `xterm-256color`. That entry
advertises palette changes (`ccc`, `initc`), printer operations (`mc*`), and
initialization sequences not established by this backend. Background-color
erase (`bce`) is also withheld: the fork's full-screen erase currently clears
to default attributes, although line erase uses the pen background. Applications
must use their normal non-BCE painting path until that semantic is repaired.

## Capability audit

| Advertised group | Backend source and scope |
| --- | --- |
| UTF-8, wrapping, eight-column tabs | `third_party/vt/emulator.go`, `csi_cursor.go`, `screen.go`; fixed default geometry is overridden by actual PTY dimensions |
| Cursor motion, addressing, save/restore | `handlers.go` CSI A–H/G/d and ESC 7/8; `csi_cursor.go` |
| Normal/alternate buffers, margins | `csi_mode.go` 1049 and `handlers.go` CSI r; alternate save/restore is distinct from application cursor/keypad |
| Erase, character/line insertion/deletion, scroll | `handlers.go` CSI J/K/X/@/P/L/M/S/T and ESC M; `csi_screen.go`; no selective erase or BCE claim |
| Indexed/truecolor, SGR styles | `csi_sgr.go` delegates to `uv.ReadStyle`; `screen.go` retains styled cells; `setaf/setab` always use indexed SGR and `setrgbf/setrgbb` use direct RGB |
| DEC line drawing | `handlers.go` charset selection and `charset.go`; source uses G0 selection without relying on ambient character-set state |
| Application cursor/keypad and F1–F12 | `key.go`; notably Home/End remain CSI H/F, Backspace is DEL, arrows use SS3 after `smkx` |
| SGR mouse | `mouse.go`; `kmous` identifies SGR packets, `XM` enables normal tracking plus SGR encoding; drag/all-motion are independently negotiated modes |
| Bracketed paste | `mode.go`, `paste.go`; BE/BD and PS/PE describe the exact enable/disable and delimiters |
| Cursor style and synchronized drawing | `handlers.go` DECSCUSR; `mode.go` 2026 with endpoint publication timeout ownership |

Focus reporting, Kitty keyboard flags, hyperlinks, title/cwd notifications and
clipboard **writes** are negotiated/handled through the endpoint, not inferred
from the TERM name. The profile adds no clipboard-read or graphics capability.
`Ms` is intentionally absent because its conventional clipboard capability does
not distinguish the endpoint's write-only policy.

DA1 reports the VT100 advanced-video text baseline (`CSI ? 1 ; 2 c`). DA2 reports
that baseline and Pair profile revision 1 (`CSI > 0 ; 1 ; 0 c`), without the
backend default's VT220/132-column/selective-erase claims. Extended features use
their own negotiation. XTGETTCAP reports only the declared table (plus standard
terminal-name/color aliases); unsupported names receive a negative reply and
stop the requested list. See the [xterm control-sequence reference](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html).

This source audit and compiled-metadata check do not replace the M3/M4 native
shell, nvim and Zellij conformance checks against the packaged entry.
