// The fidelity check's page (#478): webcast.CheckFidelity opens it at
// r/<id>/fidelity.html in a headless Chrome and, for each of the
// recording's frames in time order, calls fidelity.replayAt(t, window) and
// screenshots the viewport. The page frame sits at (0,0) at the operator's
// viewport, unscaled; there is no shell to leave out of the capture.
import { Replayer, inBrowser } from './replayer.js';

async function start() {
  const res = await fetch('records', { cache: 'no-store' });
  if (!res.ok) throw new Error(`records: ${res.status}`);
  const recs = (await res.text()).split('\n').filter(Boolean).map((l) => JSON.parse(l));
  const replayer = new Replayer(recs, (hash) => `assets/${hash}`, inBrowser(document.getElementById('page')));
  return {
    // replayAt draws t and says whether anything live was drawn, at which
    // viewport (CSS px).
    replayAt: async (t, windowMs) => {
      const { live, vp } = await replayer.replayAt(t, windowMs);
      return { live, w: vp.w, h: vp.h };
    },
  };
}

window.fidelity = { ready: start().then((api) => Object.assign(window.fidelity, api)) };
