import ctypes
import http.server
import json
import os
from pathlib import Path
import signal
import socket
import socketserver
import subprocess
import threading
import time

ROOT = Path(__file__).parent.resolve()
CLI = '/Applications/Tailscale.app/Contents/MacOS/Tailscale'
ENV = {'HOME': os.path.expanduser('~'), 'PATH': '/usr/bin:/bin:/usr/sbin:/sbin',
       'TMPDIR': str(ROOT), 'TAILSCALE_BE_CLI': '1'}
os.chmod(ROOT, 0o700)
SOCK = ROOT / 's'
assert len(str(SOCK).encode()) <= 100
events = []
mu = threading.Lock()
received = threading.Event()
ready = threading.Event()
overflow = threading.Event()
output = bytearray()
libc = ctypes.CDLL(None, use_errno=True)

def record(kind, **fields):
    row = {'at': time.time(), 'kind': kind, **fields}
    with mu:
        if len(events) >= 128:
            overflow.set()
            return
        events.append(row)
        (ROOT / 'events.json').write_text(json.dumps(events, indent=2))
    print(json.dumps(row), flush=True)

def query(args):
    p = subprocess.run([CLI, *args], env=ENV, capture_output=True, timeout=8)
    if len(p.stdout) > 1 << 20 or len(p.stderr) > 65536:
        raise RuntimeError('metadata output exceeded bound')
    if p.returncode:
        raise RuntimeError('metadata failed: ' + p.stderr.decode(errors='replace')[:512])
    return json.loads(p.stdout)

class Handler(http.server.BaseHTTPRequestHandler):
    def setup(self):
        self.request.settimeout(2)
        super().setup()

    def log_message(self, *_):
        pass

    def do_GET(self):
        pid = self.connection.getsockopt(0, 2)  # Darwin SOL_LOCAL / LOCAL_PEERPID
        uid, gid = ctypes.c_uint(), ctypes.c_uint()
        rc = libc.getpeereid(self.connection.fileno(), ctypes.byref(uid), ctypes.byref(gid))
        peer = {'pid': pid, 'uid': uid.value, 'gid': gid.value, 'getpeereid_rc': rc}
        before = subprocess.run(['/bin/ps', '-p', str(pid), '-o', 'pid=,ppid=,uid=,lstart=,comm='],
                                env=ENV, capture_output=True, timeout=2)
        peer['process_before'] = before.stdout.decode(errors='replace')[:1024]
        headers = {key: [v[:512] for v in self.headers.get_all(key, [])]
                   for key in ['Tailscale-User-Login', 'X-Forwarded-For',
                               'X-Forwarded-Host', 'X-Forwarded-Proto',
                               'Tailscale-Funnel-Request', 'Host']}
        after = subprocess.run(['/bin/ps', '-p', str(pid), '-o', 'pid=,ppid=,uid=,lstart=,comm='],
                               env=ENV, capture_output=True, timeout=2)
        peer['process_after'] = after.stdout.decode(errors='replace')[:1024]
        record('request', peer=peer, headers=headers, direct_control=pid == os.getpid())
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.send_header('Content-Length', '32')
        self.end_headers()
        self.wfile.write(b'Pair private probe received.\nOK\n\n')
        if pid != os.getpid():
            received.set()

class Server(socketserver.UnixStreamServer):
    allow_reuse_address = False

def drain(pipe):
    while True:
        block = pipe.read1(4096)
        if not block:
            return
        if len(output) + len(block) > 65536:
            overflow.set()
            return
        output.extend(block)
        (ROOT / 'serve-output.txt').write_bytes(output)
        if b'Available within your tailnet' in output or b'Press Ctrl+C' in output:
            ready.set()

server = None
worker = None
child = None
reader = None
result = 'not-run'
try:
    state = query(['status', '--json', '--peers=false'])
    if state.get('BackendState') != 'Running':
        raise RuntimeError('Tailscale is not Running')
    host = state['Self']['DNSName'].rstrip('.')
    if host not in state.get('CertDomains', []):
        raise RuntimeError('HTTPS certificate domain is not enabled')
    baseline = query(['serve', 'status', '--json'])
    record('baseline', serve=baseline, version=state.get('Version'), hostname=host)
    if baseline != {}:
        raise RuntimeError('Serve baseline is occupied; refusing all changes')
    server = Server(str(SOCK), Handler)
    os.chmod(SOCK, 0o600)
    worker = threading.Thread(target=server.serve_forever)
    worker.start()
    direct = socket.socket(socket.AF_UNIX)
    direct.settimeout(5)
    direct.connect(str(SOCK))
    direct.sendall(b'GET /control HTTP/1.0\r\nHost: localhost\r\nTailscale-User-Login: forged.invalid\r\nX-Forwarded-For: 100.1.1.1\r\n\r\n')
    while direct.recv(4096):
        pass
    direct.close()
    child = subprocess.Popen([CLI, 'serve', '--https=443', 'unix:' + str(SOCK)],
                             env=ENV, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                             stderr=subprocess.STDOUT, start_new_session=True)
    record('foreground-start', pid=child.pid, argv=[CLI, 'serve', '--https=443', 'unix:' + str(SOCK)])
    reader = threading.Thread(target=drain, args=(child.stdout,))
    reader.start()
    until = time.monotonic() + 30
    while not ready.wait(0.1):
        if child.poll() is not None or overflow.is_set() or time.monotonic() > until:
            raise RuntimeError('foreground Serve did not become ready within 30s')
    active = query(['serve', 'status', '--json'])
    if str(SOCK) not in json.dumps(active) or 'true' in json.dumps(active.get('AllowFunnel', {})):
        raise RuntimeError('exact private mapping not confirmed or Funnel present')
    record('ready', url='https://' + host + '/probe', serve=active, socket_mode=oct(SOCK.stat().st_mode & 0o777))
    # The operator visits from iPhone. A Mac self-request is an optional separate
    # control, not a replacement for tailnet identity evidence from the phone.
    until = time.monotonic() + 180
    while not received.wait(0.2):
        if child.poll() is not None or overflow.is_set() or time.monotonic() > until:
            raise RuntimeError('no proxied request within bounded 180s window')
    time.sleep(1)
    result = 'proxied-request-captured; identity still requires evidence review'
except Exception as exc:
    result = 'FAILED: ' + str(exc)
finally:
    if child is not None:
        if child.poll() is None:
            os.killpg(child.pid, signal.SIGINT)
        try:
            child.wait(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(child.pid, signal.SIGKILL)
            child.wait(timeout=3)
        if reader is not None:
            reader.join(timeout=3)
        record('foreground-joined', pid=child.pid, exit=child.returncode,
               reader_joined=reader is None or not reader.is_alive())
    if server is not None:
        server.shutdown()
        server.server_close()
        worker.join(timeout=3)
    if SOCK.exists():
        SOCK.unlink()
    try:
        final = query(['serve', 'status', '--json'])
        record('cleanup', serve=final, socket_absent=not SOCK.exists(), baseline_restored=final == {})
        if final != {}:
            result += '; CLEANUP UNCONFIRMED: mapping remains; no reset attempted'
    except Exception as exc:
        result += '; CLEANUP UNCONFIRMED: ' + str(exc)
    record('result', detail=result, inherited_identity_variables=[])
