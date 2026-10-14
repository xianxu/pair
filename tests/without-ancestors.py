#!/usr/bin/env python3
"""Run a test command reparented to init, with a bounded lifetime and exit status.

Used when testing native process ancestry from inside a developer's Zellij.
No production environment override or synthetic process-table executable is needed.
"""
import os
import signal
import subprocess
import sys
import time

args = sys.argv[1:]
timeout = 30.0
if args[:1] == ["--timeout"]:
    timeout = float(args[1])
    args = args[2:]

read_fd, write_fd = os.pipe()
intermediate = os.fork()
if intermediate == 0:
    os.close(read_fd)
    worker = os.fork()
    if worker:
        os._exit(0)
    os.setsid()
    # Tell the supervising test the private process group for timeout cleanup.
    os.write(write_fd, f"{os.getpid()}\n".encode())
    code = 1
    try:
        deadline = time.monotonic() + 5
        while os.getppid() != 1:
            if time.monotonic() >= deadline:
                raise RuntimeError("test worker was not reparented to init")
            time.sleep(0.01)
        code = subprocess.run(args, timeout=timeout).returncode
    except Exception as error:
        print(error, file=sys.stderr, flush=True)
    finally:
        os.write(write_fd, f"{code}\n".encode())
        os.close(write_fd)
        os._exit(0)

os.close(write_fd)
os.waitpid(intermediate, 0)
with os.fdopen(read_fd) as result:
    worker = int(result.readline())

    def expired(_signal, _frame):
        # Only this invocation's private worker group is addressed.
        try:
            os.killpg(worker, signal.SIGKILL)
        except ProcessLookupError:
            pass
        raise TimeoutError("reparented test command exceeded its lifetime")

    signal.signal(signal.SIGALRM, expired)
    signal.alarm(int(timeout) + 10)
    try:
        line = result.readline()
        code = int(line) if line else 1
    finally:
        signal.alarm(0)
sys.exit(code if code >= 0 else 128 - code)
