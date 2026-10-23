# Pair Local Extensions

## Repo-specific rules

- Architecture: pair is a thin, standalone, per-folder launcher of a coding
  agent; Couch is the integrated environment that configures pair and owns
  Couch-level state (e.g. per-repo remembered agent arguments). pair reads
  nothing of Couch's. Couch passes arguments explicitly; do not add a shared
  Pair/Couch preference authority or make Pair's standalone configuration read
  Couch state. See [the ownership boundary](atlas/architecture.md#pair-and-couch-state-ownership).

- Before every `sdlc close` or narrow re-close, run `make test-affected-inventory`.
  Audit each named reader's inputs before using the printed per-file repair command;
  never refresh hashes in bulk to silence a stale inventory.

- For development from Parley, use a branch in place at `~/workspace/pair` so
  the operator can smoke-test there. Do not create a separate worktree by
  default; preserve unrelated local edits when switching branches.
- After moving an issue branch into `pair:0`, run `make build` there and verify
  its HEAD is still the selected branch HEAD. Already-running Pair sessions keep
  the old binary; relaunch or start a fresh thread to test the new build. Follow
  the shared [slot-move procedure](atlas/workflow/workspace-branching.md#move-this-branch-to-n).

<!-- Add repo-specific workflow rules, conventions, or overrides here. -->
<!-- weave composes this fragment (declared `internal prose` in
     construct/base.manifest) into pair's AGENTS.md, after ariadne's exported
     Constitution. It is pair-only — never composed into a consumer's AGENTS.md. -->
