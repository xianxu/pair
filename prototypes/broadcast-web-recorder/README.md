# broadcast-web recorder prototypes (pair#453)

Throwaway prototypes behind pair#453's comparison; the results and recommendation are in that issue's Log. This
branch is not merged and nothing here is product code.

- **A:** rrweb 2.1.6 injected over CDP into every frame (`record.mjs`, `A_INJECT`), replayed by rrweb's `Replayer`
  (`replay/a.html`).
- **B:** own recorder on CDP's DOM/CSS/Network domains plus an isolated-world observer (`record.mjs`, `B_*`),
  replayed by `replay/bplay.js` in a sandboxed, script-free frame.

## Run

Needs Google Chrome at the default macOS path and Node. Chrome must run outside the agent sandbox.

```sh
npm install && npm run setup
mkdir -p runs
node record.mjs runs/stress-AB http://127.0.0.1:8801/stress.html idle AB 20   # both recorders, one session
node fidelity.mjs runs/stress-AB                                              # replay both, pixel-diff vs the tab
node stats.cjs runs/stress-AB                                                 # volume, first-snapshot latency
node record.mjs runs/ovh-B-1 http://127.0.0.1:8801/stress.html idle B 30      # overhead: modes none | A | B
```

Scenarios (`scenarios.mjs`): `idle`, `app` (for `pages/app.html`), `browse` (scrolling, for public sites).
`node sbs.cjs runs/<run> <i>` writes the real tab, A and B side by side for screenshot `i`. Chrome runs headless
with its own throwaway profile.
