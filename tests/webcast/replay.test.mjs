// The replay (#471): the seek rule against the format's own catch-up
// invariant, the timeline the bar draws, the bar's time placement, #t=
// links, and the player reaching every fixture pin in any order.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { Model, goJSON } from '../../cmd/internal/webcast/web/model.js';
import { catchUp, timeline, placer, parseMoment, momentHash, Player, GAP_SHARE } from '../../cmd/internal/webcast/web/replay.js';
import { fixtures, records, pins } from './fixtures.mjs';

const sha = (s) => createHash('sha256').update(s).digest('hex');

function reduce(recs) {
  const m = new Model();
  for (const r of recs) m.apply(r);
  return m;
}

// What the catch-up promises (webcast-format.md, Joining mid-cast): the
// same DOM, assets, DocSeg, current-document notes as the whole prefix.
function promised(m) {
  return {
    canonical: goJSON(m.canonical()),
    docSeg: m.docSeg,
    notes: m.notes.filter((n) => n.seg >= m.docSeg).map((n) => [n.note.id, n.state]),
    assets: [...m.assets].map(([u, a]) => [u, a.hash]).sort(),
  };
}

// A renderer that draws nothing: the player's state is the model's.
const fakeRenderer = (h) => (model) => { h.renderers++; return { model, apply() {}, flush() {}, revokeAll() { h.revoked++; } }; };

// cuts are the prefixes worth checking: right before and after every
// record that the rule treats specially, and an even sample between.
function cuts(recs) {
  const at = new Set([0, recs.length]);
  recs.forEach((r, i) => {
    if (['seg', 'stop', 'ctl', 'note', 'note-state', 'stroke', 'asset', 'doc'].includes(r.k)) { at.add(i); at.add(i + 1); }
  });
  const step = Math.max(1, Math.floor(recs.length / 60));
  for (let i = 0; i < recs.length; i += step) at.add(i);
  return [...at].filter((i) => i >= 1 && i <= recs.length).sort((a, b) => a - b);
}

for (const name of fixtures) {
  test(`${name}: the catch-up of any prefix reduces to what the prefix does`, () => {
    const recs = records(name);
    const whole = new Model();
    let applied = 0;
    for (const i of cuts(recs)) {
      for (; applied < i; applied++) whole.apply(recs[applied]);
      assert.deepEqual(promised(reduce(catchUp(recs.slice(0, i)))), promised(whole), `${name}, cut ${i} (t=${recs[i - 1].t})`);
    }
  });
}

// synthetic covers what no fixture has: assets before a later segment, a
// note (and its state) on an earlier document, control before the latest
// segment, and a resume keeping the document's notes.
const H = 'a'.repeat(64), H2 = 'b'.repeat(64);
const vp = { w: 800, h: 600, dpr: 1 };
const page = (t, id, url) => ({ t, k: 'doc', doc: id, url, tree: { id, y: 'doc', c: [{ id: id + 1, y: 'el', n: 'html', c: [{ id: id + 2, y: 'el', n: 'body', c: [{ id: id + 3, y: 'el', n: 'img', a: { src: 'pic.png' } }] }] }] } });
const who = { id: 'v1', name: 'Ana' };
const synthetic = [
  { t: 0, k: 'head', format: 'couch-webcast', v: 1, id: 'syn', started: 's' },
  { t: 0, k: 'seg', seg: 1, why: 'live', tab: { id: 't', url: 'http://x/' }, vp },
  page(0, 10, 'http://x/'),
  { t: 5, k: 'asset', url: 'http://x/pic.png', hash: H, mime: 'image/png', size: 1 },
  { t: 6, k: 'note', id: 'n1', author: who, text: 'old', at: { node: 13 } },
  { t: 7, k: 'note-state', id: 'n1', state: 'resolved' },
  { t: 8, k: 'ctl', op: 'request', who, pending: [who] },
  { t: 9, k: 'ctl', op: 'grant', who, driver: who },
  { t: 10, k: 'stop', why: 'pause' },
  { t: 20, k: 'seg', seg: 2, why: 'navigate', tab: { id: 't', url: 'http://x/2' }, vp },
  page(20, 20, 'http://x/2'),
  { t: 21, k: 'asset', url: 'http://x/pic.png', hash: H2, mime: 'image/png', size: 1 },
  { t: 22, k: 'note', id: 'n2', author: who, text: 'new', at: { node: 23 } },
  { t: 23, k: 'stroke', id: 's1', author: who, color: '#f00', pts: [[1, 1, 0], [2, 2, 40]] },
  { t: 24, k: 'stop', why: 'pause' },
  { t: 30, k: 'seg', seg: 3, why: 'resume', tab: { id: 't', url: 'http://x/2' }, vp },
  page(30, 20, 'http://x/2'),
  { t: 31, k: 'note-state', id: 'n2', state: 'resolved' },
  { t: 32, k: 'stop', why: 'end' },
];

test('synthetic: the catch-up of every prefix reduces to what the prefix does', () => {
  for (let i = 1; i <= synthetic.length; i++) {
    assert.deepEqual(promised(reduce(catchUp(synthetic.slice(0, i)))), promised(reduce(synthetic.slice(0, i))), `cut ${i}`);
  }
});

test('a catch-up that leaves out earlier assets breaks the invariant (the check bites)', () => {
  const upto = synthetic.slice(0, 11); // through the navigate's doc
  const broken = catchUp(upto).filter((r) => r.k !== 'asset');
  assert.notDeepEqual(promised(reduce(broken)), promised(reduce(upto)));
});

test('the player over synthetic: a seek back across a navigation drops the new note', () => {
  const p = new Player(synthetic, { renderer: fakeRenderer({ renderers: 0, revoked: 0 }) });
  p.seek(31);
  assert.deepEqual(promised(p.model), promised(reduce(synthetic.slice(0, 18))));
  p.seek(6);
  assert.deepEqual(promised(p.model).notes, [['n1', 'open']]);
});

test('timeline: segments, a pause as a gap, a tab switch with none, notes', () => {
  const tl = timeline(records('segments'));
  assert.equal(tl.end, 9000);
  assert.deepEqual(tl.segs, [
    { seg: 1, why: 'live', t0: 0, t1: 2000 },
    { seg: 2, why: 'resume', t0: 5000, t1: 7000 },
    { seg: 3, why: 'tab', t0: 7000, t1: 9000 },
  ]);
  assert.deepEqual(tl.gaps, [{ t0: 2000, t1: 5000 }]);
  assert.deepEqual(tl.notes.map((n) => [n.t, n.id]), [[1500, 'seg-n1'], [5500, 'seg-n2']]);
});

test('timeline: who drove when, and the acts they caused', () => {
  const tl = timeline(records('driven'));
  assert.deepEqual(tl.drives.map((d) => [d.t0, d.t1, d.driver.name]), [[1500, 6500, 'Ana'], [7000, 7500, 'Bo'], [8500, 9000, 'Cy']]);
  assert.deepEqual(tl.acts.map((a) => [a.t, a.by]), [[2000, 'v1'], [2000, 'v1'], [2400, 'v1'], [5500, 'v1'], [6000, 'v1'], [7200, 'v2']]);
  assert.deepEqual(tl.gaps, [{ t0: 3500, t1: 5000 }]);
  assert.equal(tl.notes.length, 1);
});

test('placer: monotonic, inverse on segments, a long gap squeezed, a gap seeks to its end', () => {
  const tl = { end: 3_600_000 + 2000, gaps: [{ t0: 1000, t1: 3_601_000 }] };
  const p = placer(tl, 1000);
  const gapW = p.xAt(3_601_000) - p.xAt(1000);
  assert.ok(gapW <= GAP_SHARE * 1000 + 1e-9, `gap is ${gapW}px`);
  assert.ok(p.xAt(1000) > 400, 'the segments keep the width');
  let prev = -1;
  for (let t = 0; t <= tl.end; t += 50_000) { const x = p.xAt(t); assert.ok(x >= prev); prev = x; }
  for (const t of [0, 250, 999, 3_601_000, 3_601_500, tl.end]) assert.ok(Math.abs(p.timeAt(p.xAt(t)) - t) <= 2, `t=${t}`);
  assert.equal(p.timeAt((p.xAt(1000) + p.xAt(3_601_000)) / 2), 3_601_000);
  assert.equal(p.timeAt(-5), 0);
  assert.equal(p.timeAt(5000), tl.end);
  // A short gap keeps its real share.
  const q = placer({ end: 1000, gaps: [{ t0: 500, t1: 510 }] }, 1000);
  assert.ok(Math.abs(q.xAt(510) - q.xAt(500) - 10) < 1e-9);
  // Nothing recorded: everything is at 0.
  assert.equal(placer({ end: 0, gaps: [] }, 100).timeAt(50), 0);
});

test('#t= links', () => {
  assert.equal(parseMoment('#t=1530'), 1530);
  assert.equal(parseMoment('#t=0'), 0);
  for (const bad of ['', '#t=', '#t=-1', '#t=1.5', '#x=1', '#t=12abc', '#t=1e3', null]) assert.equal(parseMoment(bad), null, String(bad));
  assert.equal(momentHash(1530.4), '#t=1530');
  assert.equal(parseMoment(momentHash(42)), 42);
});


function seeded(n) { let s = n; return () => { s = (s * 1103515245 + 12345) & 0x7fffffff; return s / 0x7fffffff; }; }

for (const name of fixtures) {
  test(`${name}: seeking reaches every pin, forward, backward and shuffled`, () => {
    const recs = records(name), ps = pins(name);
    const h = { renderers: 0, revoked: 0 };
    const p = new Player(recs, { renderer: fakeRenderer(h) });
    const check = (pin) => { p.seek(pin.t); assert.equal(p.error, null); assert.equal(sha(goJSON(p.model.canonical())), pin.hash, `${name} at t=${pin.t}`); };
    ps.forEach(check);
    [...ps].reverse().forEach(check);
    const rnd = seeded(471), shuffled = [...ps].sort(() => rnd() - 0.5);
    shuffled.forEach(check);
  });
}

test('a forward seek within a segment applies on; across a seg or backward it rebuilds', () => {
  const h = { renderers: 0, revoked: 0 };
  const p = new Player(records('segments'), { renderer: fakeRenderer(h) });
  p.seek(500);
  const r0 = p.rebuilds;
  p.seek(1500);
  assert.equal(p.rebuilds, r0, 'forward in segment 1');
  p.seek(5500);
  assert.equal(p.rebuilds, r0 + 1, 'across the resume');
  p.seek(1000);
  assert.equal(p.rebuilds, r0 + 2, 'backward');
  assert.equal(h.revoked, h.renderers - 1, 'each rebuild frees the old drawing');
});

test('playing advances by speed, skips the pause, and stops at the end', () => {
  const p = new Player(records('segments'), { renderer: fakeRenderer({ renderers: 0, revoked: 0 }) });
  p.seek(0);
  p.play();
  p.tick(1000);
  assert.equal(p.t, 1000);
  p.setSpeed(2);
  p.setSpeed(3); // not offered: ignored
  assert.equal(p.speed, 2);
  p.tick(600); // 1000 + 1200 = 2200: inside the pause, so on to the resume
  assert.equal(p.t, 5000);
  assert.equal(p.model.live, true);
  p.tick(10_000);
  assert.equal(p.t, 9000);
  assert.equal(p.playing, false);
  p.play(); // at the end: from the start
  assert.equal(p.t, 0);
  assert.equal(p.playing, true);
  p.pause();
  p.tick(1000);
  assert.equal(p.t, 0);
});

test('a record the model refuses stops the player in error, keeping what it drew', () => {
  const recs = records('segments').slice();
  const bad = recs.findIndex((r) => r.t > 1000);
  recs.splice(bad, 0, { t: recs[bad].t, k: 'attr', id: 999999, name: 'x', v: 'y' });
  const p = new Player(recs, { renderer: fakeRenderer({ renderers: 0, revoked: 0 }) });
  p.seek(1000);
  const before = goJSON(p.model.canonical());
  p.play();
  p.tick(500); // forward in the segment, onto the bad record
  assert.ok(p.error, 'in error');
  assert.equal(p.playing, false);
  assert.equal(goJSON(p.model.canonical()), before);
  p.seek(0);
  assert.equal(goJSON(p.model.canonical()), before, 'stays in error');
});
