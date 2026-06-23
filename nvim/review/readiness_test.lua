-- nvim/review/readiness_test.lua — run via `nvim -l nvim/review/readiness_test.lua`
-- (or `make test-lua`). Pure; no IO. Covers all five review-start cases.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'readiness.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

eq(M.classify({ is_git = false }), 'stop', 'not a git repo')
eq(M.classify({ is_git = true, is_tracked = false }), 'track', 'git-managed but untracked')
eq(M.classify({ is_git = true, is_tracked = true, on_review_branch = true, file_matches = true }),
  'resume', 'review branch, our file')
eq(M.classify({ is_git = true, is_tracked = true, on_review_branch = true, file_matches = false }),
  'interact', 'review branch, another file')
eq(M.classify({ is_git = true, is_tracked = true, on_review_branch = false, is_clean = true }),
  'new', 'clean, not on a review branch')
eq(M.classify({ is_git = true, is_tracked = true, on_review_branch = false, is_clean = false }),
  'interact', 'dirty, not on a review branch')

if fails > 0 then os.exit(1) end
print('readiness_test ok')
