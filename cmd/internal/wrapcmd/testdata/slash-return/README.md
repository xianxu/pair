# Native slash Return regression captures

These are literal native terminal paint prefixes for pair#162, separate from
`testdata/tty` and its version qualification catalog. Replay at 120 columns by
38 rows. No native launch, login or network access is needed for the test.

| File | Native agent | Composer | Source capture | Retained bytes |
| --- | --- | --- | --- | --- |
| `claude.raw` | Claude Code 2.1.309 | `/status` | `pair162-claude-status.9fSyZW/native.raw` | 2,585 of 6,256 |
| `codex.raw` | Codex CLI 0.163.0 | `/model` | `pair162-codex-trace.9b1sR6/native.raw` | 3,599 of 6,334 |

Both source runs were captured on 2026-10-22 under
`/Volumes/TestScratch/tmp/pair-slot2/`. Claude's source was the successful native
status round trip; Codex's source was the successful production drive trace.
The source files are local evidence, not runtime test dependencies.

Extraction fed each original capture into the production terminal model one
byte at a time and stopped at the first snapshot where `Complete` was true and
`qualifiedComposerText` returned the exact command above. No bytes after that
boundary are retained. In particular, Claude's account/status panel is excluded.
The temporary extraction test was removed after writing the prefixes.

Path redaction replaced only ASCII letters and digits within captured absolute
or home-relative paths with ASCII `x`. Separators, punctuation, byte counts,
escape sequences, styles, cursor motion and cell widths are unchanged. No
synthetic paint or command echo was added. The zero-based raw prefix ranges are:

- Claude: offset 414, length 36.
- Codex: offsets 690, 1,579, 1,948, 2,434 and 3,479; length 54 each.

No email or operator-name match was present in the retained prefixes. The test
also changes the final snapshot's command characters to equal-width prose in
memory; that negative case does not modify the fixtures.

SHA-256 hashes:

| Agent | Full original source | Original prefix | Redacted fixture |
| --- | --- | --- | --- |
| Claude | `a22000c324e0805e0ebd80344ba5642109f938b52f5a391228f36ccf090e57cf` | `009d41a39691c02446d75987eb1a77aa00e9c0c88b04f284ac619c131829331b` | `f4567bef18053c34265fcde6242138eedde65b90cf7cd53ef3312cc59d85bc46` |
| Codex | `0797a33dda0ca9400f82a291ab06c1b6db1cb5e07175e9c2e196b2c3d419aeed` | `91113b6d9ceab40d3798ecd8edfed81e410795081c86a176f95d081dd7c291a3` | `56642dc678c8123b9cd33a5fd33a7eebdfb91daa43794ffe9d2d1b4658aa5320` |

`TestSlashReturnNativeCapture` verifies the extraction boundary, exact composer
text, native submit bytes, and absence of slash authority on incomplete,
unknown or prose evidence throughout bytewise replay. Recognized active
non-command composers and the final prose/incomplete mutations must retain the
newline remap. Inactive startup/menu frames retain their separate Return policy;
this regression does not claim they are non-submitting.
