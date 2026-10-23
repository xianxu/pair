#!/bin/bash
# Runs inside the bootstrap's private guest environment before sealing an image.
set -euo pipefail
case "$(/usr/sbin/sysctl -n hw.model)" in VirtualMac*) ;; *) exit 125;; esac
[ "$HOME" != /Users/xianxu ]
[ -z "${COUCH_TREE-}${COUCH_SLOT-}${OPENAI_API_KEY-}${ANTHROPIC_API_KEY-}${SSH_AUTH_SOCK-}" ]
[ -d "$COUCH_ISOLATED_ROOT" ] && [ -d "$ZELLIJ_SOCKET_DIR" ]
[ -d "$PAIR_DATA_DIR" ] && [ -d "$WF_SDLC_QUEUE_ROOT" ]
# Only the new read-only input share is admitted. Attempt a fresh write to our
# snapshot, never to the source checkout or any host session directory.
share='/Volumes/My Shared Files'
[ "$(/bin/ls -A "$share")" = input ]
if /usr/bin/touch "$share/input/.pair-image-write-probe" 2>/dev/null; then
    echo 'input share unexpectedly writable' >&2
    exit 125
fi
[ -d .git ] && [ ! -L .git ]
/usr/bin/git rev-parse --verify HEAD
/usr/bin/xcrun --find clang
/usr/bin/xcrun --find make
python3 - <<'PY'
import os,pty,select
master,slave=pty.openpty()
try:
    os.write(slave,b'pair-native-pty\n')
    assert select.select([master],[],[],2)[0], 'PTY unreadable'
    assert b'pair-native-pty' in os.read(master,256)
finally:
    os.close(master);os.close(slave)
PY
nvim --headless -u NONE '+lua assert(vim.fn.has("nvim") == 1)' +qall
[ "$(zellij --version)" = 'zellij 0.45.1' ]
echo 'image qualification: private environment, read-only input, independent Git, CLT, PTY, nvim, official Zellij verified'
