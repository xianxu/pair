-- nvim/review/readiness.lua — pure git-readiness classifier for the review-start
-- flow (issue #66 M4a'). Given git facts about the target file + repo, decide the
-- review-start action; the AGENT acts on it. PURE (no IO) — the thin git-fact
-- gathering lives in cmd/internal/reviewcmd (bin/pair-review-readiness; #93 M3),
-- which invokes this classifier via `nvim --headless` so it stays the single
-- source of the 4-case decision. The 4 cases are
-- workshop/targets/review-protocol.md's "Readiness probe".
local M = {}

-- facts: { is_git, is_tracked, on_review_branch, file_matches, is_clean }
-- returns one of:
--   'stop'     not a git repo → ask the operator to create one (don't auto-init)
--   'track'    git-managed but untracked → start tracking (then a new review)
--   'resume'   on a review/<slug> branch whose scoped file IS the target → resume
--   'new'      not on a review branch + clean → new review/<slug>
--   'interact' on another file's review branch, OR not on a review branch + dirty
function M.classify(f)
  if not f.is_git then return 'stop' end
  if not f.is_tracked then return 'track' end
  if f.on_review_branch then
    return f.file_matches and 'resume' or 'interact'
  end
  return f.is_clean and 'new' or 'interact'
end

return M
