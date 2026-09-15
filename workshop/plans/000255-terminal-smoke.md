# #255 operator smoke candidate

M4 prepares an isolated candidate. Operator acceptance, issue close and merge remain pending. The original display-corruption and missing live-selection symptoms are not declared resolved by automated checks alone.

## Launch

Open a **new outer terminal window/tab outside an existing Zellij session**, then run:

```sh
python3 /tmp/pair255-smoke-gfr6g7pd/launch.py
```

The launcher uses candidate Couch, Pair and its sibling launch helper plus copied runtime assets. It creates fresh threads in a scratch Git repository with agent, draft and right terminal panes. Couch registry, Pair artifacts, cache and Zellij socket live under `/tmp/pair255-smoke-gfr6g7pd/`; installed binaries and current sessions are unchanged. HOME and agent authentication/configuration remain available; this is session isolation, not an agent sandbox. Agent transcripts may still use their ordinary user storage.

A real fresh outer terminal matters: clearing Zellij environment variables inside an existing pane does not remove actual process ancestry. The launcher clears inherited Pair/Couch/Zellij/CMUX identity and supplies candidate paths. Shell startup may override PATH; in the right pane, `command -v pair`, `command -v couch` and `$PAIR_HOME` should refer to this candidate root.

## Check

1. Create visible output first: ask the fresh agent to print thirty numbered lines including `café 界 👩‍💻`, without editing files. In the right pane, print a few wrapped lines or open nvim. While agent output is active, drag-select text in the agent pane and hold the mouse button. Highlight should follow the drag before release. Release and verify copied text. Repeat in the right terminal and nvim, including wrapped Unicode text.
2. Open/close the Couch switcher, switch threads and right terminal tabs repeatedly, resize the window, and repeat selection. Check pane borders, cursor placement, text and colors for flashing, replacement glyphs or stale fragments.
3. Exercise actual Codex and Claude interaction: ordinary typing, the configured Return/Alt+Return behavior, Codex Option+Up questions, paste and copy. In an Other/free-text choice, type an answer before submitting.
4. Detach and reattach a smoke thread, verify its existing history/conversation remains, and repeat held selection. Exercise normal and alternate screen programs.
5. From the candidate right pane, run `(sleep 5; pair notify "smoke attention") &`, then switch to another Couch thread. Verify attention appears for the originating thread without disturbing its screen; switch back and continue typing/selection.
6. Keep the candidate active through a representative longer session. Report terminal app/version, duration, operations and any artifact/highlight loss; a brief clean run alone is not sustained acceptance.

Typing meets the provisional latency target in isolated measurements. Saturated-history switching narrowly misses100ms: pooled p95101ms at80×24 and115ms at240×80, versus baseline187ms/215ms. Please also assess switching responsiveness. These timings include independent interpreter IPC/parsing and are not native terminal measurements.

## Stop or revert

Close only the smoke threads through the candidate Couch. The same launcher can list its private state:

```sh
python3 /tmp/pair255-smoke-gfr6g7pd/launch.py couch --list
python3 /tmp/pair255-smoke-gfr6g7pd/launch.py zellij list-sessions
```

If a smoke session remains, identify its exact name from this isolated listing and terminate only that session with the same launcher and `zellij kill-session <exact-name>`. Close the new terminal. Your usual installed Couch/Pair remains available throughout. Keep the candidate directory and logs until results are reviewed; it can be removed after its sessions have ended. No global session cleanup is necessary.
