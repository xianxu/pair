// The headless replay harness (#460): drives the viewer's model and
// renderer over a fixture through the Replayer the fidelity check uses
// (#478), unscaled at the operator's viewport, for replay_test.go to read
// canonical() and capture the frame. Assets resolve to the fixture
// server's assets/<hash>, as the proxy (#459) serves them.
import { Replayer, inBrowser } from '/web/replayer.js';
import { HOLD_MS } from '/web/render.js';

const frame = document.getElementById('page');
const deps = inBrowser(frame);
let replayer;

async function fetchRecords(url) {
  const text = await (await fetch(url)).text();
  return text.split('\n').filter(Boolean).map((l) => JSON.parse(l));
}

const open = (name, recs) => { replayer = new Replayer(recs, (hash) => `/fx/${name}/assets/${hash}`, deps); };

// load starts a fixture's replay from its first record.
window.load = async (name) => {
  const recs = await fetchRecords(`/fx/${name}/records`);
  open(name, recs);
  return recs.length;
};

// replayTo applies every record up to t and returns the drawn canonical.
window.replayTo = (t) => replayer.replayTo(t);

// join replays a fresh viewer from the catch-up at t, then the records
// after it up to until: a viewer joining mid-cast.
window.join = async (name, t, until) => {
  const after = (await fetchRecords(`/fx/${name}/records`)).filter((r) => r.t > t && r.t <= until);
  open(name, [...await fetchRecords(`/fx/${name}/catchup?t=${t}`), ...after]);
  return replayer.replayTo(until);
};

// replayAt draws the page as the real tab looked at t (Replayer.replayAt).
window.replayAt = async (t, windowMs) => (await replayer.replayAt(t, windowMs)).canonical;

// releaseAt is when a viewer that drew a fixture up to t shows that
// moment's document (#475): t itself, or when the paint held over t is
// released, the stylesheets bound or HOLD_MS after its segment began.
window.releaseAt = async (name, t) => {
  await window.load(name);
  const r = replayer;
  let began = 0;
  const step = async () => {
    const rec = r.recs[r.next++];
    if (rec.k === 'seg' || rec.k === 'head') began = rec.t;
    r.feed(rec);
    r.renderer.flush();
    if (r.renderer.holding) await r.renderer.bound();
    return rec.t;
  };
  while (r.next < r.recs.length && r.recs[r.next].t <= t) await step();
  let at = t;
  while (r.renderer.holding && r.next < r.recs.length && r.recs[r.next].t < began + HOLD_MS) at = await step();
  return r.renderer.holding ? Math.max(at, began + HOLD_MS) : at;
};

window.boxOf = (id) => replayer.renderer.boxOf(id);
window.nodeAt = (x, y) => replayer.renderer.nodeAt(x, y);
window.harnessReady = true;
