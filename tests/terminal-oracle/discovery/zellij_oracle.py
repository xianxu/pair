"""Disposable native oracle; temporary artifacts belong to one run only."""
import base64
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


class CaptureTail:
    """Bound diagnostic retention independently of terminal repaint volume."""
    def __init__(self, limit=8192):
        self.limit = limit
        self.data = bytearray()

    def append(self, chunk):
        self.data.extend(chunk[-self.limit:])
        del self.data[:-self.limit]

    def text(self):
        return self.data.decode(errors='replace')[-2000:]


def run(wire, cols=8, rows=4, resize=None):
    # This scope includes setup/spawn failures, not just the running child.
    with tempfile.TemporaryDirectory(prefix='pw', dir='/tmp') as directory:
        return _run(Path(directory), wire, cols, rows, resize)


def _run(root, wire, cols, rows, resize):
    session = 'pw' + str(os.getpid()) + str(time.monotonic_ns())[-5:]
    child = root / 'child.py'
    encoded = base64.b64encode(wire.encode()).decode()
    child.write_text(
        'import os,time,base64\n'
        f'os.write(1,base64.b64decode({encoded!r}))\n'
        f'open({str(root / "ready")!r},"w").write("ready")\n'
        'time.sleep(90)\n'
    )
    (root / 'layout.kdl').write_text(
        'layout { pane borderless=true command="/usr/bin/python3" '
        '{ args "' + str(child) + '"; }; }'
    )
    (root / 'config.kdl').write_text(
        'pane_frames false\ndefault_shell "/bin/sh"\non_force_close "quit"\n'
    )
    env = os.environ.copy()
    for key in ['ZELLIJ', 'ZELLIJ_SESSION_NAME', 'ZELLIJ_PANE_ID']:
        env.pop(key, None)
    env.update(
        TERM='xterm-256color', XDG_CACHE_HOME=str(root / 'cache'),
        XDG_CONFIG_HOME=str(root / 'config'), ZELLIJ_SOCKET_DIR=str(root / 'socket'),
    )
    master, slave = pty.openpty()
    process = reader = None
    stop = threading.Event()
    captured = CaptureTail()

    def drain():
        try:
            while not stop.is_set():
                if not select.select([master], [], [], .1)[0]:
                    continue
                chunk = os.read(master, 65536)
                if not chunk:
                    break
                captured.append(chunk)
        except OSError:
            pass

    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', rows, cols, 0, 0))
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
        for _ in range(100):
            if (root / 'ready').exists() or process.poll() is not None:
                break
            time.sleep(.05)
        time.sleep(.15)
        if resize:
            fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack('HHHH', resize[1], resize[0], 0, 0))
            time.sleep(.4)
        result = subprocess.run(
            ['zellij', '--session', session, 'action', 'dump-screen', '--full'],
            env=env, capture_output=True, text=True, timeout=10,
        )
        assert result.returncode == 0, dict(
            returncode=result.returncode, stderr=result.stderr[-2000:], pty=captured.text(),
        )
        return dict(
            cols=cols, rows=rows, resize=resize, returncode=result.returncode,
            stdout=result.stdout, stderr=result.stderr[-2000:], ptyerror='',
        )
    finally:
        try:
            if process is not None:
                try:
                    subprocess.run(
                        ['zellij', 'kill-session', session], env=env,
                        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10,
                    )
                finally:
                    try:
                        process.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        process.terminate()
                        try:
                            process.wait(timeout=2)
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


if __name__ == '__main__':
    print(json.dumps(run(**json.load(sys.stdin)), ensure_ascii=False))
