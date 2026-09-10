#!/bin/bash
# Where does zellij put the cursor when a line WRAPS at the bottom margin of a
# DECSTBM region? The pane process asks zellij itself (DSR, ESC[6n), so the
# answer is zellij's own cursor, not an inference from how it rendered.
out="$1"
rows=$(tput lines); cols=$(tput cols)
stty -echo -icanon
q() {
	printf '\033[6n'
	IFS='[;' read -r -s -d R _ r c
	echo "$1 row=$r col=$c" >> "$out"
}
echo "pane rows=$rows cols=$cols region=1..$((rows - 1))" >> "$out"
printf '\033[1;%dr' "$((rows - 1))"    # reserve the bottom row, as pair term does
printf '\033[%d;1H' "$((rows - 1))"    # sit on the region's bottom margin
q "at-bottom-margin"
printf 'a short line\n'
q "after-short-line"                   # xterm: still on the bottom margin
printf '%*s' "$((cols + 10))" '' | tr ' ' W
q "after-wrapping-line"                # xterm: bottom margin, col 11
printf '\n'
q "after-its-newline"                  # xterm: bottom margin, col 1
echo DONE >> "$out"
sleep 30
