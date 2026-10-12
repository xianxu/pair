// The replay the fidelity check (#478) and the conformance harness draw
// with: records before the window at once, the window in real time, and
// only forward. A fake renderer and clock stand in for the page.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Replayer } from '../../cmd/internal/webcast/web/replayer.js';
import { Model, goJSON } from '../../cmd/internal/webcast/web/model.js';
import { fixtures, records } from './fixtures.mjs';

// fake is a page: a clock that sleeps instantly, a renderer that notes when
// (on the fake clock) each record was drawn, and animations that note
// finish and pause.
function fake() {
  const log = { applied: [], layouts: [], anims: [] };
  let now = 0;
  const anim = { finish: () => log.anims.push(['finish', now]), pause: () => log.anims.push(['pause', now]) };
  const deps = {
    renderer: (model) => ({
      apply: (r) => log.applied.push([r.t, now]),
      flush: () => {},
      ready: async () => {},
      canonical: () => model.canonical(),
    }),
    now: () => now,
    sleep: async (ms) => { now += Math.max(0, ms); },
    paint: async () => {},
    animations: () => [anim],
    layout: (vp) => log.layouts.push([vp.w, vp.h]),
  };
  return { deps, log, clock: () => now };
}

const name = fixtures.find((f) => f === 'blog') || fixtures[0];
const recs = records(name);

test('replayAt draws the records before the window at once and the window in real time', async () => {
  const { deps, log } = fake();
  const r = new Replayer(recs, (h) => `assets/${h}`, deps);
  const t = recs[recs.length - 1].t;
  const out = await r.replayAt(t, 1000);
  for (const [rt, drawn] of log.applied) {
    if (rt <= t - 1000) assert.equal(drawn, 0, `record at ${rt} is before the window`);
    else assert.equal(drawn, rt - (t - 1000), `record at ${rt} drawn at its time in the window`);
  }
  assert.equal(log.applied.length, recs.filter((x) => x.t <= t).length);
  assert.deepEqual(log.anims.map((a) => a[0]), ['finish', 'pause']);
  assert.equal(log.anims[1][1], 1000, 'animations pause at t');
  // What was drawn is the reducer's state at t.
  const m = new Model();
  for (const x of recs) if (x.t <= t) m.apply(x);
  assert.equal(out.canonical, goJSON(m.canonical()));
  assert.deepEqual(out.vp, { w: m.vp.w, h: m.vp.h });
  assert.equal(out.live, m.live);
  assert.ok(log.layouts.length > 0 && log.layouts.every(([w, h]) => w > 0 && h > 0), 'the frame follows the viewport');
});

test('replayAt only moves forward, and never waits for what is drawn already', async () => {
  const { deps, log, clock } = fake();
  const r = new Replayer(recs, (h) => `assets/${h}`, deps);
  const end = recs[recs.length - 1].t;
  const a = Math.floor(end / 2);
  await r.replayAt(a, 1000);
  const drawnBefore = log.applied.length;
  const before = clock();
  await r.replayAt(a + 300, 1000);
  // The second window starts at a, not at a + 300 - 1000.
  assert.equal(clock() - before, 300);
  const fresh = log.applied.slice(drawnBefore).map(([rt]) => rt);
  assert.ok(fresh.every((rt) => rt > a && rt <= a + 300), 'each record is drawn once');
  r.reset();
  assert.equal(r.next, 0);
});

test('an asset resolves through the given URL', async () => {
  const { deps } = fake();
  let resolve;
  deps.renderer = (model, res) => { resolve = res; return { apply() {}, flush() {}, ready: async () => {}, canonical: () => model.canonical() }; };
  const r = new Replayer(recs, (h) => `assets/${h}`, deps);
  await r.replayTo(recs[recs.length - 1].t);
  const [url, asset] = [...r.model.assets][0] || [];
  assert.ok(url, `${name} has assets`);
  assert.equal(resolve(url), `assets/${asset.hash}`);
  assert.equal(resolve('http://nowhere.invalid/x.png'), '');
});
