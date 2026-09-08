#!/bin/sh
# Does SGR 2 (faint) survive zellij, and is it distinguishable from normal?
#
# pair#217 wants the tab strip dimmed when the pane loses focus. SGR 2 is the
# obvious attribute, but a terminal that ignores it renders NOTHING different --
# a silent no-op, which is the failure mode this repo keeps paying for. This
# asks whether zellij passes it through as a distinct rendition.
printf '\033[2J\033[H'
printf '\033[5;1H';  printf '\033[0mNORMAL_SAMPLE'
printf '\033[7;1H';  printf '\033[2mFAINT_SAMPLE\033[0m'
printf '\033[9;1H';  printf '\033[38;5;244mCOLOUR_SAMPLE\033[0m'
sleep 25
