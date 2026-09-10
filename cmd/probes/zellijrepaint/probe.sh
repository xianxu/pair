#!/bin/sh
# Print a marker ONCE, then go quiet. Anything that puts the marker back on the
# host pty afterwards did so from zellij's own buffer, not from the child.
printf 'ZELLIJ_REPAINT_MARKER\n'
sleep 30
