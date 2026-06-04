-- nvim/doctor.lua — pure payload builder for the `:PairDoctor` command
-- (issue #000048). No vim API here, so it runs under `nvim -l` for tests
-- (`make test-lua`, in Makefile.local). init.lua dofile's it and wraps the IO
-- (read $PAIR_HOME, hand the instruction to the agent pane via send_to_agent).
--
-- Why a command instead of a Claude skill: pair is agent-agnostic and runs in
-- arbitrary project dirs. A `.claude/skills/` entry only works under claude, and
-- a relative `doctor/doctor.sh` only resolves when the agent's cwd is the pair
-- checkout. nvim is the one substrate every agent shares, and it knows
-- $PAIR_HOME — so it can hand ANY agent a $PAIR_HOME-absolute instruction. The
-- procedure itself stays single-sourced in doctor/SKILL.md; this is the pointer.
local M = {}

-- payload(pair_home) → the instruction string with $PAIR_HOME-absolute paths
-- substituted in (NOT a literal `$PAIR_HOME` — the agent must not depend on its
-- shell to expand it), or nil when pair_home is missing/empty (the caller
-- notifies instead of sending a broken path).
function M.payload(pair_home)
  if not pair_home or pair_home == '' then return nil end
  local h = pair_home:gsub('/+$', '') -- trim trailing slash(es)
  return 'Run `bash ' .. h .. '/doctor/doctor.sh` (it reads this pair session\'s '
    .. 'adaptation flight recorder), then follow `' .. h .. '/doctor/SKILL.md`: '
    .. 'check the emitter-health line first, interpret any drift findings against `'
    .. h .. '/atlas/how-to-bring-up-a-new-harness-cli.md` §3, and propose concrete '
    .. 'matcher fixes for me to approve — don\'t edit anything silently.'
end

return M
