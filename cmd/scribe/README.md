# scribe

A `script(1)`-replacement that supports pause/resume of the typescript via
signals. Built because macOS `script(1)`:

- Doesn't open the typescript with `O_APPEND`, so the file can't be
  truncated externally without corrupting the binary's view of its own
  write position (verified — produces sparse output and can abort with
  `assertion: advance > 0`).
- Has no way to pause logging while a TUI program (claude, nvim, lazygit,
  …) floods the typescript with redraw bytes that aren't useful for
  "copy last command output."

`scribe` is signal-controllable: `SIGUSR1` pauses on-disk capture,
`SIGUSR2` resumes. Terminal output to the user is never paused — only the
log file.

## Build

From the pair repo root:

    make scribe-install

Produces `~/bin/scribe`. Static binary, no runtime deps.

## Use

Same shape as `script -q -F LOG CMD`:

    scribe -log PATH -- CMD [ARGS...]

In `~/.zshrc`, replace

    exec /usr/bin/script -q -F "$_ZSH_SCRIPT_LOG" /bin/zsh

with

    exec ~/bin/scribe -log "$_ZSH_SCRIPT_LOG" -- /bin/zsh

Then in `preexec` / `precmd`, send signals to `$_ZSH_SCRIPT_LOG_OWNER`
around commands whose output you don't want captured.
