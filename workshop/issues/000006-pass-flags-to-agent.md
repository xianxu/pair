---
id: 000006
status: working
deps: [000001]
created: 2026-05-02
updated: 2026-05-02
---

# pass flags to agent

`bin/pair` currently exposes only `<agent>` and `<variant>` as positional args. The layout invokes the agent as `exec ${PAIR_AGENT:-claude}` with no way to pass extra flags through. So `pair claude --resume`, `pair claude -- --model haiku-4-5`, `pair codex -- -p "say hi"` — none of those work.

## Spec

**Argv shape (Unix `--` separator):**

```
pair                                  # claude, no flags
pair claude                           # claude, no flags
pair claude work                      # variant=work, no flags
pair claude -- --resume               # claude --resume
pair claude work -- --model haiku-4-5 # variant + flags
pair codex -- -p "say hi"             # extra args to codex
```

Everything before `--` is positional (agent, variant) as today. Everything after `--` is appended to the agent's command line.

**Implementation sketch:**
1. `bin/pair`: walk `"$@"`, split on `--`. Stash positionals into AGENT/VARIANT as today; join post-`--` args into `$PAIR_AGENT_ARGS` and export.
2. `zellij/layouts/main.kdl`: change agent pane command to `exec ${PAIR_AGENT:-claude} ${PAIR_AGENT_ARGS:-}`. Shell expands and word-splits.
3. `bin/pair --help`: add the `--` syntax to the USAGE block.
4. `README.md`: document under Usage.

**Semantic:** flags apply only on **create** (when the agent process is spawned). Re-invoking `pair claude -- --resume` against an existing detached session attaches without applying the flags — the agent is already running with whatever flags were used at create time. Document this.

**Caveat to call out:** word-splitting `$PAIR_AGENT_ARGS` doesn't preserve args containing spaces. `pair claude -- --system-prompt "hello world"` would become `--system-prompt hello world`, four args, not what the user wants. Most CLI flags don't have spaces, but worth a note. A future v2 could template the layout file at launcher time to pass args as a proper KDL list.

## Plan

- [ ] Parse argv in `bin/pair`: split on `--`, populate `$PAIR_AGENT_ARGS`, export.
- [ ] Update `zellij/layouts/main.kdl` agent pane command to use `$PAIR_AGENT_ARGS`.
- [ ] Update `bin/pair --help` USAGE section with the `--` syntax and an example.
- [ ] Update `README.md` Usage section.
- [ ] Update `atlas/architecture.md` (the `bin/pair` section).
- [ ] Note the create-only / attach-no-effect semantic in help and README.
- [ ] Note the args-with-spaces caveat in README.
- [ ] `bash -n`, `zellij setup --check`, `zellij setup --dump-layout` clean.
- [ ] Manual smoke test: `pair claude -- --version` (or whatever flag claude recognizes) shows the flag took effect.

## Log

### 2026-05-02

Filed during conversation; user has follow-up questions before implementation. Will pick up later.
