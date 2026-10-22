# pair scribe

A `script(1)`-replacement that supports pause/resume of the typescript via
signals. Built because macOS `script(1)`:

- Doesn't open the typescript with `O_APPEND`, so the file can't be
  truncated externally without corrupting the binary's view of its own
  write position (verified — produces sparse output and can abort with
  `assertion: advance > 0`).
- Has no way to pause logging while a TUI program (claude, nvim, lazygit,
  …) floods the typescript with redraw bytes that aren't useful for
  "copy last command output."

`pair scribe` is signal-controllable: `SIGUSR1` pauses on-disk capture,
`SIGUSR2` resumes. Terminal output to the user is never paused — only the
log file.

It isn't part of pair's runtime — it's user shell tooling that swaps for
`script(1)` at the top of the zsh session. The logic lives in
`cmd/internal/scribecmd`; since #104 it is reached only as the `pair scribe`
subcommand (the standalone `pair-scribe` binary was folded into the single
`pair` — there is no separate binary to maintain).

## Build

From the pair repo root:

    make install

Produces `~/.local/bin/pair` (the single binary). Static, no runtime deps.

## Use

Same shape as `script -q -F LOG CMD`:

    pair scribe -log PATH -- CMD [ARGS...]

In `~/.zshrc`, replace

    exec /usr/bin/script -q -F "$_ZSH_SCRIPT_LOG" /bin/zsh

with

    exec pair scribe -log "$_ZSH_SCRIPT_LOG" -- /bin/zsh

Then in `preexec` / `precmd`, send signals to `$_ZSH_SCRIPT_LOG_OWNER`
around commands whose output you don't want captured.

## Child output and lifetime

On Darwin, scribe keeps a parent-side slave descriptor until it finishes reading
master output. This preserves final bytes from one-shot commands that never
open `/dev/tty`; XNU can otherwise discard unread output when the child closes
its last slave descriptor. Scribe then releases its slave and waits once for its
direct child, preserving the child's exit result. Other platforms use the
upstream PTY startup behavior.

The child remains the controlling-terminal session leader. Its exit drains and
revokes that terminal, so scribe reaches EOF even if a descendant keeps a slave
open or ignores hangup. Scribe returns its direct child's status without waiting
for descendants. A live direct child that closes stdio is still waited for.

If initialization fails after launch, scribe kills its owned direct child,
closes the master and slave, and reaps it; the initialization error remains the
reported failure. Forced process termination or broken output transport can
abandon unread bytes. Logging pause/resume affects the log only; it does not
change child or terminal ownership.

This ownership correction applies to scribe. Wrap and Couch/term's `ptychild`
have different EOF transitions and are tracked separately in #769 and #770.
