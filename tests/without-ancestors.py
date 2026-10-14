#!/usr/bin/env python3
"""Run a test command reparented to init, with bounded failure cleanup.

Successful commands may intentionally leave sidecars running. On failure or
supervisor cancellation, terminate the invocation's private process group.
"""
import os
import select
import signal
import subprocess
import sys
import time

args = sys.argv[1:]
timeout = 30.0
if args[:1] == ["--timeout"]:
    timeout = float(args[1])
    args = args[2:]

cancelled = 0


def cancel(signum, _frame):
    global cancelled
    cancelled = signum


# Record cancellation even during startup, before the group exists. The worker
# cannot launch a command until the supervisor acknowledges its ready message.
signal.signal(signal.SIGTERM, cancel)
signal.signal(signal.SIGINT, cancel)
read_fd, write_fd = os.pipe()
start_read, start_write = os.pipe()
intermediate = os.fork()
if intermediate == 0:
    os.close(read_fd)
    os.close(start_write)
    os.setsid()  # The group ID is known to the supervisor before the second fork.
    worker = os.fork()
    if worker:
        os._exit(0)
    signal.signal(signal.SIGTERM, signal.SIG_DFL)
    signal.signal(signal.SIGINT, signal.SIG_DFL)
    code = 1
    try:
        deadline = time.monotonic() + 5
        while os.getppid() != 1:
            if time.monotonic() >= deadline:
                raise RuntimeError("test worker was not reparented to init")
            time.sleep(0.01)
        os.write(write_fd, b"ready\n")
        if os.read(start_read, 1) != b"x":
            os._exit(1)
        code = subprocess.run(args, timeout=timeout).returncode
    except Exception as error:
        print(error, file=sys.stderr, flush=True)
    finally:
        try:
            os.write(write_fd, f"{code}\n".encode())
        except BrokenPipeError:
            pass
        os.close(write_fd)
        # Remain alive until the supervisor accepts the result or kills the
        # group, so failure cleanup always addresses an owned live group.
        os.read(start_read, 1)
        os.close(start_read)
        os._exit(0)

os.close(write_fd)
os.close(start_read)
code = 1
buffer = b""
deadline = time.monotonic() + timeout + 10
try:
    while not cancelled and time.monotonic() < deadline:
        readable, _, _ = select.select([read_fd], [], [], 0.05)
        if not readable:
            continue
        chunk = os.read(read_fd, 1024)
        if not chunk:
            break
        buffer += chunk
        while b"\n" in buffer:
            line, buffer = buffer.split(b"\n", 1)
            if line == b"ready":
                if not cancelled:
                    os.write(start_write, b"x")
            else:
                code = int(line)
                break
        else:
            continue
        break
finally:
    # Closing the handshake also prevents a late-starting worker from launching
    # after cancellation raced with setsid (when killpg could find no group).
    os.close(read_fd)
    if cancelled:
        code = 128 + cancelled
    if code != 0:
        try:
            os.killpg(intermediate, signal.SIGKILL)
        except ProcessLookupError:
            pass
    os.close(start_write)
    os.waitpid(intermediate, 0)
sys.exit(code if code >= 0 else 128 - code)
