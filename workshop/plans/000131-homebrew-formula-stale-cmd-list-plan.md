# Homebrew Formula Release Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish a post-Go-migration Pair release and update `xianxu/homebrew-pair` so `brew install xianxu/pair/pair` builds the current single-binary source layout.

**Architecture:** Treat the release tag, GitHub-generated tarball checksum, and Homebrew formula as one publication transaction. The formula should derive runtime helpers from the single public `pair` binary rather than restating the old multi-command layout (`ARCH-DRY`, `ARCH-PURPOSE`). External effects are git/GitHub/Homebrew operations verified by live commands rather than fakes because this issue is specifically a release publication.

**Tech Stack:** Git tags/GitHub releases, Ruby Homebrew formula, Go build, Homebrew tap repo `../homebrew-pair`.

**Spec:** `workshop/issues/000131-homebrew-formula-stale-cmd-list.md`

---

## Non-Goals

- Do not add bottles or binary-only packaging; this release fixes source-build
  Homebrew install first.
- Do not preserve old helper binaries as public or separately built commands;
  helper behavior is reached through the single `pair` dispatcher.
- Do not solve #132 beyond replacing the false caveats line with wording that
  does not send users to `pair --help` for keybindings.
- Do not build release automation in this pass; record any automation gap as a
  future extension after the manual transaction succeeds.

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `PairFormula` | `../homebrew-pair/Formula/pair.rb` | modified |
| `ReleaseNotes` | `CHANGELOG.md` | modified |

- **`PairFormula`** - Homebrew formula metadata and install logic for Pair.
  - **Relationships:** one formula points to one Pair release tag and checksum.
  - **DRY rationale:** the install block builds only `./cmd/pair-go` and installs it as `pair`; helper names are no longer restated as separate Go build targets.
  - **Future extensions:** a later binary-bottle release can reuse the same formula metadata while replacing source-build details.
- **`ReleaseNotes`** - user-facing summary for the release being tagged.
  - **Relationships:** one changelog section corresponds to one git tag.
  - **DRY rationale:** release notes live in `CHANGELOG.md`; the GitHub release can point at or summarize that section.
  - **Future extensions:** generate release notes from issue history if this becomes frequent.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `PairReleaseTag` | git/GitHub `v1.24` | new | GitHub-generated source tarball |
| `HomebrewPublish` | `../homebrew-pair` remote | modified | `brew` and tap git remote |

- **`PairReleaseTag`** - annotated or lightweight release tag pushed to GitHub so Homebrew can fetch the current source tree.
  - **Injected into:** `PairFormula` URL/checksum.
  - **Future extensions:** add a scripted release command that tags, waits for tarball availability, computes sha, and opens the tap update.
- **`HomebrewPublish`** - live formula validation through Homebrew.
  - **Injected into:** release close evidence.
  - **Future extensions:** add a disposable-prefix smoke script for release prep.

## Test Surface

- `go test ./...` over the Pair source tree guards that the release commit still
  builds and passes the existing Go test suite; no new pure functions are added
  by this release transaction.
- `ruby -c ../homebrew-pair/Formula/pair.rb` guards formula syntax.
- `brew style ../homebrew-pair/Formula/pair.rb` guards Homebrew style.
- `brew install --build-from-source xianxu/pair/pair` from a clean prefix, then
  `pair --version` and `pair --help`, guard the risky `PairFormula#install` and
  formula `test do` surfaces against the real Homebrew/GitHub integration
  (`ARCH-MOCK`: this issue is the live conformance check for the external
  packaging path).

## Tasks

- [x] Confirm no post-Go-migration release exists by checking local tags and GitHub releases; record the answer in #131.
- [x] Add a `v1.24` section to `CHANGELOG.md` covering the major user-facing changes since `v1.23`.
- [ ] Commit and push the Pair repo release commit containing the `CHANGELOG.md`, issue, and plan updates.
- [ ] Verify Pair source builds and tests from the release commit.
- [ ] Tag and push `v1.24` at that exact release commit, then verify `git rev-parse v1.24^{commit}` equals the intended commit.
- [ ] Compute the sha256 from GitHub's generated `v1.24.tar.gz`.
- [ ] Update `../homebrew-pair/Formula/pair.rb` to point at `v1.24`, use the new checksum, build only `./cmd/pair-go` as `bin/pair`, and fix stale description/caveats/comments.
- [ ] Verify the tap with `ruby -c`, `brew style`, and a source-build install/test from a clean prefix; if Homebrew cannot create a clean prefix on this machine, stop and report the blocker rather than substituting weaker evidence silently.
- [ ] Commit and push the tap update.
- [ ] Close #131 with release, tap, and Homebrew verification evidence.

## Revisions

- 2026-08-16: plan-quality PQ-1/PQ-2/PQ-3 revision. Made the release commit and
  tag target explicit, made clean Homebrew verification required instead of
  optional, and added the test surface/non-goals the gate requested.
- 2026-08-16: execution-flow revision. The release tag will point at the exact
  SDLC branch commit that contains `CHANGELOG.md`, issue, and plan updates; that
  commit is then merged through the normal SDLC flow instead of requiring a
  direct pre-close commit on `main`.
