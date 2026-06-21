-- nvim/review/seam.lua — the cross-process review-seam paths (issue #66). ONE
-- place for the `review-<tag>.open` contract (and `.mode`, M4), so the three
-- actors — the draft reader (nvim/init.lua), the pane writer (nvim/review.lua),
-- and bin/pair-review-open — can't diverge on the empty-tag fallback
-- (milestone-review I3, ARCH-DRY: review.lua used `PAIR_TAG or 'default'`, which
-- leaves an empty string as `''` since `''` is truthy in Lua, while init.lua fell
-- back to `default` — they'd look for different files if the tag were ever empty).
-- PURE (no vim API) so it loads under `nvim -l` too.
local M = {}

-- The canonical tag with the shared fallback (matches `${PAIR_TAG:-default}`).
function M.tag(env_tag)
  return (env_tag and env_tag ~= '') and env_tag or 'default'
end

-- The open-state file path, or nil when no data dir. Single source of the formula.
function M.open_state(data_dir, env_tag)
  if not data_dir or data_dir == '' then return nil end
  return data_dir .. '/review-' .. M.tag(env_tag) .. '.open'
end

-- The review-target path (seam #6, M4a'): `{file, status: proposed|ready}` — what
-- to review, set by :PairReview (proposes) + the agent (marks ready), read by Alt+r.
function M.target_path(data_dir, env_tag)
  if not data_dir or data_dir == '' then return nil end
  return data_dir .. '/review-target-' .. M.tag(env_tag) .. '.json'
end

return M
