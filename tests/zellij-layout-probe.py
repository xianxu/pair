"""pair#324 experiment: does native zellij apply a --new-session-with-layout layout?"""
import os, pty, select, subprocess, sys, tempfile, time

def run(name, extra, use_config=True, env_extra=None):
    d = tempfile.mkdtemp(prefix="zp-", dir="/tmp")
    marker = os.path.join(d, "marker")
    layout = os.path.join(d, "layout.kdl")
    open(layout, "w").write('layout { pane borderless=true command="/bin/sh" { args "-c" "echo ok > %s; sleep 30"; }; }\n' % marker)
    config = os.path.join(d, "config.kdl")
    open(config, "w").write('pane_frames false\nshow_startup_tips false\nshow_release_notes false\non_force_close "quit"\n')
    session = os.path.basename(d)
    argv = ["zellij", "--session", session]
    if use_config:
        argv += ["--config", config]
    argv += extra(layout, d)
    env = dict(os.environ, ZELLIJ="", ZELLIJ_SESSION_NAME="", ZELLIJ_SOCKET_DIR=os.path.join(d, "s"),
               XDG_CONFIG_HOME=os.path.join(d, "c"), XDG_CACHE_HOME=os.path.join(d, "k"), TERM="xterm-256color")
    env.update(env_extra or {})
    pid, fd = pty.fork()
    if pid == 0:
        os.execvpe(argv[0], argv, env)
    out = b""
    deadline = time.time() + 8
    while time.time() < deadline and not os.path.exists(marker):
        r, _, _ = select.select([fd], [], [], 0.2)
        if r:
            try:
                chunk = os.read(fd, 65536)
            except OSError:
                break
            out += chunk
            # answer CPR / DA so the client never blocks on a query
            if b"\x1b[6n" in chunk:
                os.write(fd, b"\x1b[1;1R")
            if b"\x1b[c" in chunk:
                os.write(fd, b"\x1b[?62;c")
    ok = os.path.exists(marker)
    print(f"{name}: layout_applied={ok} argv={argv[3:]} bytes={len(out)} tabbar={b'Tab #1' in out}", flush=True)
    subprocess.run(["zellij", "kill-session", session], env=env, capture_output=True, timeout=10)
    try:
        os.kill(pid, 9)
    except OSError:
        pass
    return ok

def snapshot(tag):
    home = os.path.expanduser("~")
    roots = [os.path.join(home, "Library/Caches/org.Zellij-Contributors.Zellij"),
             os.path.join(home, "Library/Application Support/org.Zellij-Contributors.Zellij"),
             os.path.join(home, ".config/zellij"), os.path.join(home, ".cache/zellij")]
    files = []
    for r in roots:
        for dp, _, fns in os.walk(r):
            for f in fns:
                p = os.path.join(dp, f)
                files.append(f"{p.replace(home, '~')}({os.path.getsize(p)})")
    print(f"  [{tag}] zellij state files: {files[:40]}", flush=True)

A = lambda l, d: ["--new-session-with-layout", l, "--data-dir", os.path.join(d, "data")]
snapshot("start")
for i, (name, kw) in enumerate([("E-zsh", {"env_extra": {"SHELL": "/bin/zsh"}}), ("A", {}), ("D-noconfig", {"use_config": False}), ("A", {}), ("A", {}), ("A-zsh", {"env_extra": {"SHELL": "/bin/zsh"}}), ("A", {})]):
    run(f"{i}-{name}", A, **kw)
    snapshot(f"after {i}")
