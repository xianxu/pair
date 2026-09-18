# Native history discovery prototypes — #255 M3 preparation

These are reproducible **discovery scripts**, not production implementation or
complete qualification fixtures. They preserve the experiments that informed the
M3 native-history refinement. Each terminal is compared to its own direct-stream
baseline; their Unicode and textual-dump behavior differ.

Run from the repository root, with Python 3, Node and Zellij available:

```sh
npm ci --prefix tests/terminal-oracle
python3 tests/terminal-oracle/discovery/typed_history.py
python3 tests/terminal-oracle/discovery/dirty_rebuild.py
python3 tests/terminal-oracle/discovery/viewport_wrap.py
python3 tests/terminal-oracle/discovery/width_reflow.py
python3 tests/terminal-oracle/discovery/one_row.py
python3 tests/terminal-oracle/discovery/wide_cells.py
python3 tests/terminal-oracle/discovery/sync_hold.py
python3 -m unittest discover -s tests/terminal-oracle/discovery -p 'test_*.py' -v
```

The pinned headless dependency is `@xterm/headless@5.5.0`, from the parent directory's
lockfile. Native discovery used **Zellij 0.45.1**. The native driver starts a uniquely
named session on a disposable PTY with temporary config, data and socket paths. It
removes inherited session identity and kills only the session it created, including
on failure. It does not attach to or modify an operator session. A `TemporaryDirectory` owns the entire setup/run/teardown lifetime and removes
all generated child scripts, configuration, socket/data/cache files on success or
failure, including spawn failure. No retained-artifact mode exists. PTY diagnostics
retain only their last 8 KiB (at most 2,000 decoded characters are reported on failure);
JSON stdout contains the fixture dump for direct-baseline comparison. The caller may
save that stdout using its existing evidence-retention policy. Do not run the native scripts against a substituted shared-session driver.

| Script | What it establishes |
|---|---|
| `typed_history.py` | Clean typed-cell staging equals direct-stream xterm cells/wraps and native Zellij text at widths 4 and 6, including early-wide gaps and actual trailing spaces. Assertions fail on mismatch. |
| `dirty_rebuild.py` | Same comparison after dirty wrapped content and old history; CSI 3J plus full-region IL discards the old viewport without admitting it, then clean typed staging rebuilds history. Assertions fail on mismatch. |
| `viewport_wrap.py` | Autowrap sets soft flags; DL+IL creates hard rows. Diagnostic child-only scrolling reveals history-to-viewport linkage and excludes chrome from admitted history. Inspect xterm flags and native `HHHHAAAABBBB\nCCCCDDDD\nCHRO` prefix. |
| `width_reflow.py` | Direct output resized 4→6 versus regenerated typed logical lines at width 6; compares xterm physical cells and native dumped text. |
| `one_row.py` | One-row LF/autowrap export, with no extra two-row bottom LF; asserts `ABCDEFGH` is one logical history line. |
| `wide_cells.py` | Exploratory early-wide, explicit-space and ECH behavior. Cases intentionally differ in final viewport/history boundary; inspect their outputs rather than comparing all cases to one another. |
| `sync_hold.py` | (#262) Whether Zellij honours DECSET 2026 from a pane: times when a marker written inside an open bracket reaches Zellij's CLIENT, against an unbracketed control. `dump-screen` cannot answer this, because the pane grid updates either way. Exit 0 honoured, 1 not, 2 inconclusive. Zellij 0.45.1: honoured (control 0.012s; bracketed arrives just after the close). |
| `xterm_oracle.cjs`, `zellij_oracle.py` | JSON wire/geometry drivers shared by the prototypes. |
| `blank_provenance.go.txt` | UV storage experiment: empty Content with Width 1 survives cell operations, but UV String/Render collapses its position and partial-wide cleanup introduces printed spaces. |

To run the UV experiment without making it a production Go package:

```sh
cp tests/terminal-oracle/discovery/blank_provenance.go.txt /tmp/pair255-blank-provenance.go
go run /tmp/pair255-blank-provenance.go
```

## Findings and limits

CUP-only repaint preserves native wrap linkage; ED2 destroys that linkage in
xterm. LF alone clears xterm's destination wrap flag but does **not** clear an
existing Zellij noncanonical row. Replacing a target row with DL+IL clears it in
both. Avoid target row 1 for DL when history must remain unchanged: Zellij can
admit the deleted top row into history.

ECH preserves wrap flags, but Zellij materializes erased padding as spaces. If
that padding lies in an early-wide soft-wrap gap, later logical-history copy gains
an unwanted space. Clean scratch rows, cleared before acquiring their soft link,
avoid the discrepancy. EL2 removes that Zellij padding but clears xterm's incoming
wrap flag. An incremental first-row export cannot blindly combine those operations:
when a dirty wrapped top-row gap cannot be preserved, use a bounded typed rebuild.
The prototypes prove that rebuild works; they do not measure its frequency or
latency in the production presenter.

Native `dump-screen --full` serializes history and viewport separately. It also
trims spaces per physical viewport row, whereas merged history retains internal
spaces. Compare equivalent boundaries, or diagnostically move the child rows into
history before asserting the logical connection. Dump text does not establish
interactive selection/copy behavior by itself. Zellij drops some combining marks;
the pinned headless default width provider also differs from modern Unicode widths.
These existing differences are visible in the direct baselines and must not be
silently treated as serializer regressions or as successful Unicode conformance.

The typed prototypes use small predefined cell lists, not the future backend
metadata or production renderer. They do not prove arbitrary cursor-edit histories,
all style/link behavior, alternate-screen transitions, cancellation, synchronized
publication, clipboard selection, or resource/latency bounds. M3 must promote these
examples into fixtures for the actual publication/serializer and extend coverage.
