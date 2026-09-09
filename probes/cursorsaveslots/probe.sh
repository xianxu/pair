#!/bin/sh
# Does this emulator keep DECSC (ESC 7 / ESC 8) and SCOSC (CSI s / CSI u) in
# SEPARATE cursor-save slots?
#
# pair#199's tab strip paints with DECSC/DECRC. So does zsh, to draw its
# right-hand prompt: terminfo sc/rc are ESC 7 / ESC 8 on xterm-256color. One
# shared slot means our paint clobbers the child's saved cursor and the child's
# restore lands where WE saved -- the operator's "cursor ends up in the tab bar"
# and a right-prompt drawn on the strip's row.
#
# If the slots are separate, moving our paint to CSI s / CSI u closes the hole
# outright. If they alias, only cursor tracking does.
printf '\033[2J'

printf '\033[5;1H'   ; printf '\033[s'   # SCOSC remembers row 5
printf '\033[10;1H'  ; printf '\0337'    # DECSC remembers row 10
printf '\033[20;1H'                      # go somewhere else entirely

printf '\033[u'      ; printf 'SCORC_MARK'   # restore SCOSC: row 5 if separate
printf '\033[20;1H'
printf '\0338'       ; printf 'DECRC_MARK'   # restore DECSC: row 10 if separate

sleep 30
