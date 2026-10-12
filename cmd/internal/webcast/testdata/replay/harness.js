// The headless replay harness (#460): drives the viewer's model and
// renderer over a fixture, unscaled at the operator's viewport, for
// replay_test.go to read canonical() and capture the frame. Assets resolve
// to the fixture server's assets/<hash>, as the proxy (#459) serves them.
import { Model, goJSON } from '/web/model.js';
import { Renderer, HOLD_MS } from '/web/render.js';

// #page is the frame with the current document (#475: its sibling holds
// the previous paint while a new document's stylesheets bind).
const page = () => document.getElementById('page');
let fx, recs, next, model, renderer;

const fresh = () => {
  model = new Model();
  renderer = new Renderer(page(), model, (abs) => {
    const a = model.assets.get(abs);
    return a ? `/fx/${fx}/assets/${a.hash}` : '';
  });
  // As the shell does: a redraw outside a record (a held paint released, a
  // stylesheet or image loaded) flushes before the next paint.
  renderer.onDirty = () => requestAnimationFrame(() => renderer.flush());
};

async function fetchRecords(url) {
  const text = await (await fetch(url)).text();
  return text.split('\n').filter(Boolean).map((l) => JSON.parse(l));
}

// feed applies a record and, as viewer.js does, sizes the page to the
// tab's viewport, which a tab switch can change (#472's rec-segments).
function feed(r) {
  model.apply(r);
  renderer.apply(r);
  if (model.seg > 0) {
    page().style.width = `${model.vp.w}px`;
    page().style.height = `${model.vp.h}px`;
  }
}

async function settle() {
  renderer.flush();
  await renderer.ready();
  renderer.flush();
  await new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok)));
  return goJSON(renderer.canonical());
}

// load starts a fixture's replay from its first record.
window.load = async (name) => {
  fx = name;
  for (const f of document.querySelectorAll('body > iframe')) f.contentDocument.replaceChildren(); // a fresh viewer holds no paint
  recs = await fetchRecords(`/fx/${name}/records`);
  next = 0;
  fresh();
  return recs.length;
};

// replayTo applies every record up to t and returns the drawn canonical.
window.replayTo = async (t) => {
  while (next < recs.length && recs[next].t <= t) feed(recs[next++]);
  return settle();
};

// join replays a fresh viewer from the catch-up at t, then the records
// after it up to until: a viewer joining mid-cast.
window.join = async (name, t, until) => {
  fx = name;
  fresh();
  for (const r of await fetchRecords(`/fx/${name}/catchup?t=${t}`)) feed(r);
  for (const r of await fetchRecords(`/fx/${name}/records`)) if (r.t > t && r.t <= until) feed(r);
  return settle();
};

const frameDocs = () => [page().contentDocument, ...[...page().contentDocument.querySelectorAll('iframe')].map((f) => f.contentDocument).filter(Boolean)];

// replayAt draws the page as the real tab looked at t: records up to
// t - window at once (their transitions finished), then the last window in
// real time, so a transition or animation running at t is at the phase it
// had in the tab; animations are paused at t for the capture.
window.replayAt = async (t, windowMs) => {
  const start = t - windowMs;
  while (next < recs.length && recs[next].t <= start) feed(recs[next++]);
  await settle();
  for (const d of frameDocs()) for (const a of d.getAnimations()) { try { a.finish(); } catch { /* an infinite animation has no end */ } }
  const t0 = performance.now();
  const until = (rt) => new Promise((ok) => setTimeout(ok, Math.max(0, rt - start - (performance.now() - t0))));
  while (next < recs.length && recs[next].t <= t) {
    const rt = recs[next].t;
    await until(rt);
    while (next < recs.length && recs[next].t === rt) feed(recs[next++]);
    renderer.flush();
  }
  await until(t);
  if (renderer.holding) { await renderer.bound(); renderer.flush(); } // the viewer swaps in the page as soon as its stylesheets apply, and redraws (onDirty)
  for (const d of frameDocs()) for (const a of d.getAnimations()) a.pause();
  return goJSON(renderer.canonical());
};

// releaseAt is when a viewer that drew a fixture up to t shows that
// moment's document (#475): t itself, or when the paint held over t is
// released, the stylesheets bound or HOLD_MS after its segment began.
window.releaseAt = async (name, t) => {
  await window.load(name);
  let began = 0;
  const step = async () => {
    const r = recs[next++];
    if (r.k === 'seg' || r.k === 'head') began = r.t;
    feed(r);
    renderer.flush();
    if (renderer.holding) await renderer.bound();
    return r.t;
  };
  while (next < recs.length && recs[next].t <= t) await step();
  let at = t;
  while (renderer.holding && next < recs.length && recs[next].t < began + HOLD_MS) at = await step();
  return renderer.holding ? Math.max(at, began + HOLD_MS) : at;
};

window.boxOf = (id) => renderer.boxOf(id);
window.nodeAt = (x, y) => renderer.nodeAt(x, y);
window.harnessReady = true;
