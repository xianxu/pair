// The replay the fidelity check (#478) and the conformance harness draw
// with: records before the window at once, the window in real time, and
// only forward. A fake renderer and clock stand in for the page.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Replayer } from '../../cmd/internal/webcast/web/replayer.js';
import { Model, goJSON } from '../../cmd/internal/webcast/web/model.js';
import { catchUp } from '../../cmd/internal/webcast/web/replay.js';
import { fixtures, records } from './fixtures.mjs';

// fake is a page: a clock that sleeps instantly, a renderer that notes when
// (on the fake clock) each record was drawn, and animations that note
// finish and pause.
function fake() {
  const log = { applied: [], layouts: [], anims: [] };
  let now = 0;
  const anim = {
    playState: 'running',
    finish() { log.anims.push(['finish', now]); },
    pause() { this.playState = 'paused'; log.anims.push(['pause', now]); },
    play() { this.playState = 'running'; log.anims.push(['play', now]); },
  };
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

const name = 'rec-blog'; // assets, scrolling and a fade-in
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

test('a segment that is not a navigation finishes the animations of the document it snapshots', async () => {
  for (const [why, finishedAtSeg] of [['live', true], ['resume', true], ['tab', true], ['navigate', false]]) {
    const { deps, log } = fake();
    const seg = { t: 500, k: 'seg', seg: 1, why, tab: { id: 'a', url: 'http://x/' }, vp: { w: 800, h: 600, dpr: 1 } };
    const doc = { t: 500, k: 'doc', doc: 1, url: 'http://x/', tree: { id: 1, y: 'doc', c: [{ id: 2, y: 'el', n: 'html', c: [] }] } };
    const r = new Replayer([{ t: 0, k: 'head', format: 'couch-webcast', v: 1, id: 'x', started: '2026-10-12T00:00:00Z' }, seg, doc], (h) => h, deps);
    await r.replayAt(800, 1000);
    // finish before the window, and at the segment (t=500, 700 ms into the
    // window from -200) unless it navigated.
    const finishes = log.anims.filter(([k]) => k === 'finish').map(([, at]) => at);
    assert.deepEqual(finishes, finishedAtSeg ? [0, 700] : [0], why);
  }
});

test('a moment shortly after the last continues it: paused animations resume at their phase', async () => {
  const { deps, log, clock } = fake();
  const r = new Replayer(recs, (h) => `assets/${h}`, deps);
  const a = Math.floor(recs[recs.length - 1].t / 2);
  await r.replayAt(a, 1000);
  log.anims.length = 0;
  const before = clock();
  await r.replayAt(a + 150, 1000);
  assert.deepEqual(log.anims.map(([k]) => k), ['play', 'pause'], 'resumed, never finished');
  assert.equal(clock() - before, 150, 'only the 150 ms since the last moment run');
});

test('a moment before the last seeks through the catch-up, one segment at most', async () => {
  let shorter = false;
  for (const name of fixtures) {
    const all = records(name);
    const { deps, log } = fake();
    const r = new Replayer(all, (h) => `assets/${h}`, deps);
    const end = all[all.length - 1].t;
    await r.replayAt(end, 1000);
    log.applied.length = 0;
    const t = Math.floor(end * 0.6);
    const out = await r.replayAt(t, 1000);
    const m = new Model();
    for (const x of all) if (x.t <= t) m.apply(x);
    assert.equal(out.canonical, goJSON(m.canonical()), `${name}: the seek draws the state at ${t}`);
    // The seek feeds the catch-up of the records up to the window's start,
    // then the window: not the whole prefix again.
    const before = all.filter((x) => x.t <= t - 1000), inWindow = all.filter((x) => x.t > t - 1000 && x.t <= t);
    assert.equal(log.applied.length, catchUp(before).length + inWindow.length, name);
    shorter ||= catchUp(before).length < before.length;
  }
  assert.ok(shorter, 'some fixture has a segment the seek skips');
});
