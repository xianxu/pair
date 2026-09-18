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
SETTLE = 1.5    # zellij startup before the pane writes
GRACE = 0.5     # watch this long past the close before reading the verdict
DEADLINE = 20.0  # a pane that never closes its bracket is an instrument failure


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
        # Watch relative to the CLOSE, not to launch: a slow zellij start must
        # shorten nothing we read a verdict from.
        deadline = time.time() + DEADLINE
        while not (root / 't_end').exists():
            if time.time() > deadline:
                raise TimeoutError('the pane never closed its bracket')
            time.sleep(.02)
        closed = float((root / 't_end').read_text())
        time.sleep(max(0.0, closed + GRACE - time.time()))
        watched = time.time() - closed
        wrote = float((root / 't_write').read_text())
        marker = seen.get('marker')
        return dict(bracket=bracket, closed_after=round(closed - wrote, 3),
                    watched_after_close=round(watched, 3),
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
    # 0 and 1 are verdicts about zellij, so each needs a window that could have
    # seen the other outcome; anything the window cannot account for is 2.
    if control['marker_after'] is None or control['marker_after'] > HOLD / 2:
        return 2
    if held['marker_after'] is None:
        return 1 if held['watched_after_close'] >= GRACE else 2
    return 0 if held['marker_after'] >= held['closed_after'] else 1


def main():
    # A failure to MEASURE (no writable /tmp, no pty, zellij too slow to start
    # the pane) says nothing about zellij, so it must never read as exit 1.
    try:
        control, held = run(False), run(True)
    except Exception as error:  # noqa: BLE001 -- every instrument failure is inconclusive
        print(json.dumps(dict(result='inconclusive', error=repr(error))))
        return 2
    code = verdict(control, held)
    print(json.dumps(dict(control=control, bracketed=held,
                          result=['honoured', 'NOT honoured', 'inconclusive'][code])))
    return code


if __name__ == '__main__':
    sys.exit(main())
