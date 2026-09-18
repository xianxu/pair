"""Does Zellij honour synchronized output (DECSET 2026) from a pane process? (#262)

`pair term` brackets every frame it presents into its Zellij pane. The bracket
only prevents a half-drawn pane from reaching the screen if Zellij withholds the
pane's update until the bracket closes. The native oracle (zellij_oracle.py)
cannot answer that: `dump-screen` reads the pane grid, which updates either way.
This measures what Zellij sends to its CLIENT instead.

The pane writes a marker inside an open bracket, holds HOLD seconds, then closes
it. A control run writes the marker with no bracket. Honoured means the control's
marker reaches the client at once and the bracketed marker only after the
close. Exit 0 = honoured, 1 = not honoured, 2 = inconclusive (the control never
reached the client, so nothing was measured).

Measured 2026-09-17, Zellij 0.45.1: control 0.012s; bracketed 1.016s after the
write, with the close at 1.004s. Honoured.
"""
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import struct
import subprocess
import sys
import tempfile
import termios
import threading
import time

HOLD = 1.0
SETTLE = 1.5  # zellij startup before the pane writes


def run(bracket):
    with tempfile.TemporaryDirectory(prefix='pw', dir='/tmp') as directory:
        return _run(Path(directory), bracket)


def _run(root, bracket):
    session = 'pw' + str(os.getpid()) + str(time.monotonic_ns())[-5:]
    begin, end = (r'\x1b[?2026h', r'\x1b[?2026l') if bracket else ('', '')
    child = root / 'child.py'
    child.write_text(
        'import os,time\n'
        f'time.sleep({SETTLE})\n'
        f'open({str(root / "t_write")!r},"w").write(repr(time.time()))\n'
        f'os.write(1,b"{begin}MARKER262")\n'
        f'time.sleep({HOLD})\n'
        f'open({str(root / "t_end")!r},"w").write(repr(time.time()))\n'
        f'os.write(1,b"{end}")\n'
        'time.sleep(30)\n'
    )
    (root / 'layout.kdl').write_text(
        'layout { pane borderless=true command="/usr/bin/python3" '
        '{ args "' + str(child) + '"; }; }'
    )
    # Tips and release notes open over the pane and hide its output.
    (root / 'config.kdl').write_text(
        'pane_frames false\ndefault_shell "/bin/sh"\non_force_close "quit"\n'
        'show_startup_tips false\nshow_release_notes false\n'
    )
    env = os.environ.copy()
    for key in ['ZELLIJ', 'ZELLIJ_SESSION_NAME', 'ZELLIJ_PANE_ID']:
        env.pop(key, None)
    env.update(
        TERM='xterm-256color', XDG_CACHE_HOME=str(root / 'cache'),
        XDG_CONFIG_HOME=str(root / 'config'), ZELLIJ_SOCKET_DIR=str(root / 'socket'),
    )
    master, slave = pty.openpty()
    stop = threading.Event()
    seen = {}
    received = bytearray()

    def drain():
        try:
            while not stop.is_set():
                if not select.select([master], [], [], .01)[0]:
                    continue
                chunk = os.read(master, 65536)
                if not chunk:
                    break
                received.extend(chunk)
                del received[:-4096]
                if 'marker' not in seen and b'MARKER262' in received:
                    seen['marker'] = time.time()
        except OSError:
            pass

    process = reader = None
    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 10, 40, 0, 0))
        process = subprocess.Popen(
            ['zellij', '--session', session, '--config', str(root / 'config.kdl'),
             '--new-session-with-layout', str(root / 'layout.kdl'),
             '--data-dir', str(root / 'data')],
            stdin=slave, stdout=slave, stderr=slave, env=env, start_new_session=True,
        )
        os.close(slave)
        slave = None
        reader = threading.Thread(target=drain, daemon=True)
        reader.start()
        time.sleep(SETTLE + HOLD + 1.0)
        wrote = float((root / 't_write').read_text())
        closed = float((root / 't_end').read_text())
        marker = seen.get('marker')
        return dict(bracket=bracket, closed_after=round(closed - wrote, 3),
                    marker_after=None if marker is None else round(marker - wrote, 3))
    finally:
        try:
            if process is not None:
                subprocess.run(['zellij', 'kill-session', session], env=env,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=2)
        finally:
            stop.set()
            if reader is not None:
                reader.join(timeout=1)
            if slave is not None:
                os.close(slave)
            os.close(master)


def verdict(control, held):
    if control['marker_after'] is None or control['marker_after'] > HOLD / 2:
        return 2
    if held['marker_after'] is not None and held['marker_after'] >= held['closed_after']:
        return 0
    return 1


if __name__ == '__main__':
    control, held = run(False), run(True)
    code = verdict(control, held)
    print(json.dumps(dict(control=control, bracketed=held,
                          result=['honoured', 'NOT honoured', 'inconclusive'][code])))
    sys.exit(code)
