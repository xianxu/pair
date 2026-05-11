---
id: 000019
status: open
deps: []
created: 2026-05-10
updated: 2026-05-10
related: [homebrew-pair/Formula/pair.rb]
---

# Drop Python from pair's runtime path

## Problem

In commit `14dc879`, pair-wrap and pair-scrollback-render were ported
to Go (`cmd/pair-wrap`, `cmd/scrollback-render`), but the Python
originals were kept as fallbacks:

- `bin/pair-wrap.py` — renamed from `bin/pair-wrap`; the Go binary now
  occupies that path. Kept so a broken Go build doesn't ship a wedge.
- `bin/pair-scrollback-render` — the pyte-based renderer, also kept
  as a fallback. `bin/pair-scrollback-open` prefers
  `$PAIR_HOME/bin/scrollback-render` (Go) when present and falls back
  to `python3 bin/pair-scrollback-render` otherwise.

Carrying both has costs: it leaves `python3 + pyte` in the dependency
graph (the brew formula vendors pyte into a private venv, see
`pair-bootstrap` and homebrew-pair Formula), and "two implementations"
is a maintenance trap — a future bug fix lands in one and not the
other.

Once the Go binaries have soaked through a few days of real use and
no Alt+/ / Alt+i / agent-output-span regressions have surfaced, drop
the Python.

## Spec

Three drops, ideally in one commit per repo:

**In this repo (`pair`):**

1. Delete `bin/pair-wrap.py`.
2. Delete `bin/pair-scrollback-render` (the Python pyte renderer).
3. Simplify `bin/pair-scrollback-open` to invoke the Go binary
   directly — remove the python3 + pyte preflight + fallback branch.
   Hard-fail with a clear "build the Go renderer: make
   scrollback-render-install" message if `bin/scrollback-render` is
   missing.
4. Drop the `pair-bootstrap` target's pyte-install step in
   `Makefile.local` (and the comment paragraph explaining why pyte
   was needed). The target either becomes a no-op or gets removed —
   double-check no other runtime Python deps need it before
   deleting outright.
5. Update `cmd/scribe/README.md` and `cmd/scrollback-render/README.md`
   (if it gets one) — drop any "Python fallback" language.
6. Update `atlas/architecture.md` — the scrollback section currently
   mentions "Downstream pair-scrollback-render reads both files and
   replays through pyte" / similar phrasing. Re-anchor on the Go
   binary.

**In `homebrew-pair`** (separate repo):

The v1.16 formula update (the one that introduced `depends_on "go" =>
:build` and the Go build step) deliberately kept the Python pieces
intact behind the Go binaries during the soak — same fallback shape
as in the pair repo. When this issue closes, the Formula also loses
its remaining Python surface:

7. Remove `include Language::Python::Virtualenv` at the top of the
   Formula. With pyte gone there's no virtualenv to create.
8. Remove the two `resource` blocks (`pyte`, `wcwidth`) entirely.
9. Remove `depends_on "python@3"`. python@3 was only there to host
   pair-wrap.py and the pyte venv; both are gone in pair's commit
   for drops 1-2. Once the dep is removed, brew won't pull
   python@3 as a transitive dep of pair anymore.
10. In the `install` block, delete the two-line venv setup:

        venv = virtualenv_create(libexec/"venv", "python3")
        venv.pip_install resources

    And the surrounding comment paragraph about the venv being a
    fallback.
11. Bump version (this is release-worthy on its own — drops two
    runtime dep trees: python@3 plus its closure, and the pyte/wcwidth
    bundled resources). Suggested commit subject: `pair X.Y: drop
    python@3 + pyte venv (now pure-Go runtime)`.

End state of the formula after these drops: the only runtime deps are
fzf, jq, neovim, par, zellij. Go remains as `depends_on "go" =>
:build`. Resources block is gone. The `install` block builds the two
Go binaries, installs the bin/nvim/zellij trees, and symlinks
bin/pair onto PATH — nothing else.

After publishing, verify on a clean machine (or `tart`-VM target):

    brew uninstall pair
    brew cleanup -s        # purge pyte/wcwidth cached resources
    brew install xianxu/pair/pair
    # ensure python@3 was NOT pulled in as a dep:
    brew deps pair | grep -i python && echo "REGRESSION" || echo "clean"

**In `ariadne`** (separate repo, separate session):

12. `scripts/close-issue.py` is the only Python left across the
    ariadne-styled repos that pair inherits. Port to bash — close-issue
    isn't perf-sensitive, but consistency wins. (Lower priority than
    the pair drops above; can ship anytime.)

## Plan

- [ ] Soak Go binaries in real sessions for ~3-5 days; track any
      regressions in this issue's Log section.
- [ ] Drops 1-6 (pair repo) in one commit. Subject suggestion:
      `pair: drop python from runtime path`.
- [ ] Tag pair release matching the drop (e.g. v1.17).
- [ ] Drops 7-11 (homebrew-pair). Update url + sha256 to the new pair
      tag. Verify `brew deps pair` no longer lists python@3 on a
      clean machine.
- [ ] Drop 12 (ariadne) — separate, lower priority.

## Log

- 2026-05-10: filed. Go ports landed in commit `14dc879`; Python
  fallbacks intentionally retained for the soak window.
- 2026-05-10: expanded homebrew-pair section to cover dropping
  `include Language::Python::Virtualenv`, pyte/wcwidth resources,
  `depends_on "python@3"`, and the venv install step — the v1.16
  formula bump only added the Go build path alongside; the python
  surface stays until this issue closes.
