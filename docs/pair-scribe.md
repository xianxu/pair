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
