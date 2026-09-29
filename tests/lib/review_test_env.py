"""Isolate process-level review fixtures from the invoking Pair/editor session."""
import os
import pathlib
import tempfile

# Both explicitly consumed paths and defaults used by draft/review helpers.
ARTIFACT_KEYS = tuple('PAIR_' + name for name in (
    'ADAPT_LOG_PATH', 'AGENT_CONFIG_PATH', 'AGENT_OUTPUT_PATH', 'AGENT_PATH',
    'AGENT_PICKS_PATH', 'AGENT_PID_PATH', 'CHANGELOG_READY_PATH', 'DRAFT_PANE_PATH',
    'DRAFT_PATH', 'IMAGE_CAPTURE_DONE_PATH', 'IMAGE_CAPTURE_PATH', 'LAST_LEFT_PANE_PATH',
    'LAYOUT_MODE_PATH', 'LOG_PATH', 'NVIM_PID_FILE', 'PAIR_WRAP_PID_PATH', 'QUOTE_PATH',
    'REVIEW_CONTEXT_PATH', 'REVIEW_DEFINITION_REQUEST_PATH', 'REVIEW_DEFINITION_RESULT_PATH',
    'REVIEW_HANDOFF_PATH', 'REVIEW_LANDED_PATH', 'REVIEW_MODE_PATH', 'REVIEW_OPEN_PATH',
    'REVIEW_TARGET_PATH', 'SCROLLBACK_ANSI_PATH', 'SCROLLBACK_EVENTS_PATH',
    'SCROLLBACK_PENDING_PATH', 'SCROLLBACK_RAW_PATH', 'SCROLLBACK_VIEWPORT_PATH',
    'SLUG_PATH', 'SLUG_PROPOSED_PATH', 'ZELLIJ_ACTIONS_PATH',
))


class ReviewTestEnvironment:
    """Poison inherited paths outside the fixture and check actual child effects.

    The caller directory is deliberately separate from each test's fixture tree.
    Every integration run therefore verifies that its real subprocesses leave
    inherited writable artifacts alone, even when launched outside a Pair shell.
    """
    def __enter__(self):
        self._caller = tempfile.TemporaryDirectory(prefix='pair-test-caller-')
        self.caller = pathlib.Path(self._caller.name).resolve()
        self.inherited = dict(os.environ)
        self.sentinels = {}
        for key in (*ARTIFACT_KEYS, 'NVIM_LOG_FILE'):
            path = self.caller / key
            data = ('caller-owned sentinel: ' + key + '\n').encode()
            path.write_bytes(data)
            self.sentinels[path] = data
            self.inherited[key] = str(path)
        self.inherited.update(PAIR_SESSION_ID='caller-conversation', ZELLIJ_SESSION_NAME='caller-session',
                              NVIM='caller-editor-socket', PAIR_DATA_DIR=str(self.caller))
        self.before = set(self.caller.iterdir())
        return self

    def environment(self, fixture, product_root):
        fixture = pathlib.Path(fixture).resolve()
        prefixes = ('PAIR_', 'ZELLIJ', 'NVIM', 'CODEX_', 'CLAUDE_', 'GIT_')
        session_keys = {'VIMINIT', 'EXINIT', 'MYVIMRC', 'TMUX', 'TMUX_PANE', 'STY', 'WINDOW',
                        'ITERM_SESSION_ID', 'KITTY_LISTEN_ON', 'WEZTERM_UNIX_SOCKET'}
        env = {key: value for key, value in self.inherited.items()
               if not key.startswith(prefixes) and key not in session_keys}
        artifacts = fixture / 'isolated-artifacts'
        artifacts.mkdir(parents=True, exist_ok=True)
        for key in ARTIFACT_KEYS:
            env[key] = str(artifacts / key.lower())
        env.update(PAIR_HOME=str(product_root), PAIR_DATA_DIR=str(artifacts),
                   PAIR_SESSION_ID='fixture-session', NVIM_LOG_FILE=str(artifacts / 'nvim.log'),
                   GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1')
        for key, name in [('PAIR_QUEUE_DIR', 'queue'), ('XDG_CONFIG_HOME', 'config'),
                          ('XDG_STATE_HOME', 'state'), ('XDG_DATA_HOME', 'data'),
                          ('XDG_CACHE_HOME', 'cache'), ('XDG_RUNTIME_DIR', 'runtime')]:
            directory = artifacts / name
            directory.mkdir(mode=0o700, exist_ok=True)
            env[key] = str(directory)
        return env

    def assert_untouched(self):
        changed = [str(path) for path, data in self.sentinels.items()
                   if not path.is_file() or path.read_bytes() != data]
        created = set(self.caller.iterdir()) - self.before
        assert not changed and not created, 'fixture touched caller artifacts: ' + repr(changed + [str(p) for p in created])

    def __exit__(self, exc_type, exc, traceback):
        try:
            self.assert_untouched()
        finally:
            self._caller.cleanup()


if __name__ == '__main__':
    import subprocess
    import sys
    # Shell fixtures use the same sanitizer before any Git/editor subprocess.
    with ReviewTestEnvironment() as isolated, tempfile.TemporaryDirectory(prefix='pair-review-env-') as storage:
        result = subprocess.run(['bash', sys.argv[2], '--isolated-child', *sys.argv[3:]],
                                env=isolated.environment(storage, pathlib.Path(sys.argv[1])))
    raise SystemExit(result.returncode)
