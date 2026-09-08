#!/bin/sh
rows=$(tput lines); cols=$(tput cols)
printf 'PROBE rows=%s cols=%s\n' "$rows" "$cols"
# DECSTBM: scroll region = rows 1..N-1, so the bottom row is excluded.
printf '\033[1;%dr' "$((rows - 1))"
# Paint the bottom row, bracketed by save/restore exactly as couchtty.PaintRow does.
printf '\033[s\033[%d;1H\033[2KRESERVED_ROW_MARKER\033[u' "$rows"
# Force many scrolls inside the region.
i=0; while [ $i -lt 200 ]; do echo "scroll line $i"; i=$((i+1)); done
sleep 30
