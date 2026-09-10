#!/bin/bash
# TOP-edge variant: reserve row 1 (region 2..N) and ask the same questions,
# plus the one the bottom edge never had — does homing the cursor land on the
# strip, and does origin mode (DECOM) redirect it in zellij?
out="$1"; mode="$2"
rows=$(tput lines); cols=$(tput cols)
stty -echo -icanon
q() { printf '\033[6n'; IFS='[;' read -r -s -d R _ r c; echo "$1 row=$r col=$c" >> "$out"; }
echo "pane rows=$rows cols=$cols region=2..$rows mode=$mode" >> "$out"
printf '\033[2;%dr' "$rows"
if [ "$mode" = decom ]; then printf '\033[?6h'; fi
# Paint the strip on row 1. Under DECOM, CUP is region-relative, so origin mode
# comes off for the paint and DECRC restores it (DECSC saves DECOM) — the exact
# bytes a top-edge Reservation would have to emit.
printf '\0337\033[?6l\033[1;1H\033[2KTOP_STRIP\0338'
# decom2: re-assert origin mode AFTER the paint rather than trusting DECRC to
# restore it, so the reading is about DECOM itself.
if [ "$mode" = decom2 ] || [ "$mode" = childregion ] || [ "$mode" = nvim ]; then printf '\033[?6h'; fi
case "$mode" in
childregion)
	# What a full-screen child does: set ITS OWN scroll region in ITS OWN
	# coordinates (it believes rows 1..N-1), home, and scroll inside it. DECSTBM
	# parameters are absolute even under DECOM, so rows 1..10 here are pane rows
	# 1..10 — the strip included — not the 2..11 the child means.
	printf '\033[1;10r\033[H'
	i=1; while [ $i -le 15 ]; do echo "child $i"; i=$((i+1)); done
	q "after-child-scroll"
	;;
nvim)
	f=$(mktemp); seq 1 400 > "$f"
	stty rows "$((rows - 1))"      # the child believes N-1 rows, as under pair term
	echo DONE >> "$out"
	exec nvim -u NONE -n "$f"
	;;
scroll)
	printf '\033[%d;1H' "$rows"
	i=1; while [ $i -le 10 ]; do echo "short $i"; i=$((i+1)); done
	q "after-short-lines"
	;;
wrap)
	printf '\033[%d;1H' "$rows"
	i=1; while [ $i -le 5 ]; do echo "fill $i"; i=$((i+1)); done
	q "after-short-lines"
	printf '%*s' "$((cols + 10))" '' | tr ' ' W
	q "after-wrapping-line"
	printf '\nAFTER_1\nAFTER_2\n'
	q "after-more-lines"
	;;
plain|decom|decom2)
	printf '\033[H'          # what `clear` and every full-screen app does first
	printf 'HOME_MARKER'
	q "after-home"
	;;
esac
echo DONE >> "$out"
sleep 30
