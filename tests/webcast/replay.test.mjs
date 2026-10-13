// The replay (#471): the seek rule against the format's own catch-up
// invariant, the timeline the bar draws, the bar's time placement, #t=
// links, and the player reaching every fixture pin in any order.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { Model, goJSON } from '../../cmd/internal/webcast/web/model.js';
import { catchUp, timeline, placer, parseMoment, momentHash, Player, GAP_SHARE, replaySource, replayPath, clock } from '../../cmd/internal/webcast/web/replay.js';
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
    strokes: m.strokes.filter((s) => s.seg >= m.docSeg).map((s) => s.stroke),
    assets: [...m.assets].map(([u, a]) => [u, a.hash]).sort(),
    hover: m.hover,
    ptr: m.ptr && [m.ptr.x, m.ptr.y],
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

// The catch-up corpus covers what no fixture has: assets before a later
// segment, a note (and its state) on an earlier document, control before
// the latest segment, a resume keeping the document's notes, and hover and
// ptr (only the latest of each is kept). It carries Go's answers at every
// cut (TestCatchUpCorpus keeps them true to recording.CatchUp).
const corpus = JSON.parse(readFileSync(new URL('./catchup_corpus.json', import.meta.url), 'utf8'));
const synthetic = corpus.records;

test('catchUp keeps exactly what recording.CatchUp keeps, at every cut of the corpus', () => {
  assert.equal(corpus.kept.length, synthetic.length, 'Go\'s answers cover every cut');
  for (let i = 1; i <= synthetic.length; i++) {
    const upto = synthetic.slice(0, i);
    assert.deepEqual(catchUp(upto).map((r) => upto.indexOf(r)), corpus.kept[i - 1], `cut ${i}`);
  }
});

test('synthetic: the catch-up of every prefix reduces to what the prefix does', () => {
  for (let i = 1; i <= synthetic.length; i++) {
    assert.deepEqual(promised(reduce(catchUp(synthetic.slice(0, i)))), promised(reduce(synthetic.slice(0, i))), `cut ${i}`);
  }
});

const capabilityRecords = readFileSync(new URL('../../cmd/internal/webcast/recording/testdata/capability-events.jsonl', import.meta.url), 'utf8')
  .trim().split('\n').map(JSON.parse);

test('capability clears survive catch-up at every prefix, including a later resume', () => {
  for (let i = 1; i <= capabilityRecords.length; i++) {
    const prefix = capabilityRecords.slice(0, i);
    assert.deepEqual(promised(reduce(catchUp(prefix))), promised(reduce(prefix)), `cut ${i}`);
  }
});

test('seeking across a capability clear preserves history without resurrecting cleared marks', () => {
  const p = new Player(capabilityRecords, { renderer: fakeRenderer({ renderers: 0, revoked: 0 }) });
  for (const t of [8, 1, 7, 6, 4, 9, 0, 8]) {
    p.seek(t);
    assert.equal(p.error, null);
    assert.deepEqual(promised(p.model), promised(reduce(capabilityRecords.filter((r) => r.t <= t))), `seek ${t}`);
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
  assert.deepEqual(promised(p.model), promised(reduce(synthetic.filter((r) => r.t <= 31))));
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

test('timeline: who drove when, and the acts they caused (from the recorded fixture\'s own ctl records)', () => {
  const recs = records('rec-driven');
  const tl = timeline(recs);
  // The spans the ctl records state: each from the record that made a
  // viewer the driver to the one that ended it.
  const want = [];
  let cur = null;
  for (const r of recs) {
    if (r.k !== 'ctl') continue;
    const d = r.driver || null;
    if (cur && (!d || d.id !== cur.id)) { want.push([cur.t0, r.t, cur.name]); cur = null; }
    if (d && !cur) cur = { id: d.id, name: d.name, t0: r.t };
  }
  assert.ok(want.length >= 2, 'the fixture drives twice');
  assert.deepEqual(tl.drives.map((d) => [d.t0, d.t1, d.driver.name]), want);
  // Every act carries a driver's by, inside one of the driver's spans.
  const acts = recs.filter((r) => r.by);
  assert.deepEqual(tl.acts.map((a) => [a.t, a.by]), acts.map((r) => [r.t, r.by]));
  assert.ok(acts.some((r) => r.k === 'click') && acts.some((r) => r.k === 'input'));
  for (const a of acts) assert.ok(want.some(([t0, t1]) => a.t >= t0 && a.t <= t1), `act at ${a.t} outside a drive`);
  const stops = recs.filter((r) => r.k === 'stop' && r.why === 'pause');
  assert.equal(tl.gaps.length, stops.length);
  assert.equal(tl.notes.length, recs.filter((r) => r.k === 'note').length);
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

// The source behind the shell, with fakes for the page: frames run when
// the test says, the clock is the test's.
function source(recs, hash = '') {
  const h = { shown: [], hashes: [], frames: 0, queue: [], clock: 0, hash, on: null, updates: 0 };
  h.v = replaySource({
    fetchRecords: async () => recs,
    renderer: fakeRenderer({ renderers: 0, revoked: 0 }),
    show: (v) => h.shown.push(v),
    frame: () => h.frames++,
    schedule: (fn) => h.queue.push(fn),
    now: () => h.clock,
    hash: () => h.hash,
    setHash: (x) => { h.hash = x; h.hashes.push(x); },
    ui: (player, on) => { h.on = on; return { update: () => h.updates++ }; },
  });
  h.paint = () => { const q = h.queue.splice(0); for (const fn of q) fn(); };
  return h;
}

test('replay opens at #t=, paused, and draws one frame', async () => {
  const h = source(records('segments'), '#t=5500');
  const p = await h.v.loaded;
  assert.equal(p.t, 5500);
  assert.equal(p.playing, false);
  h.paint();
  assert.equal(h.frames, 1);
  assert.equal(h.updates, 1);
  assert.deepEqual(h.shown.at(-1), { title: 'http://app.local/', url: 'http://app.local/', page: true, status: 'Replay', live: true });
  assert.equal(h.v.model, p.model);
});

test('without #t= replay opens at the start; a bad #t= too', async () => {
  for (const hash of ['', '#t=x']) assert.equal((await source(records('segments'), hash).v.loaded).t, 0);
});

test('a seek moves the playhead and the link; a pause in the recording shows as paused', async () => {
  const h = source(records('segments'));
  await h.v.loaded;
  h.on.seek(3000);
  assert.equal(h.hash, '#t=3000');
  h.paint();
  assert.equal(h.shown.at(-1).status, 'Replay · paused');
  assert.equal(h.shown.at(-1).live, false);
  h.hash = '#t=7500';
  h.v.onHash();
  assert.equal(h.v.player.t, 7500);
});

test('play runs on frames by the clock; pausing writes the link', async () => {
  const h = source(records('segments'));
  await h.v.loaded;
  h.on.toggle();
  h.paint(); // first frame: no time has passed
  assert.equal(h.v.player.t, 0);
  h.clock = 1000;
  h.paint();
  assert.equal(h.v.player.t, 1000);
  h.on.speed(4);
  h.clock = 1250;
  h.paint();
  assert.equal(h.v.player.t, 2000);
  h.on.toggle();
  assert.equal(h.hash, '#t=2000');
  h.paint();
  assert.equal(h.queue.length, 0, 'no frames while paused');
});

test('playing to the end stops and writes the link', async () => {
  const h = source(records('segments'), '#t=8900');
  await h.v.loaded;
  h.on.toggle();
  h.paint();
  h.clock = 1000;
  h.paint();
  assert.equal(h.v.player.playing, false);
  assert.equal(h.hash, '#t=9000');
});

test('an unreadable recording says so', async () => {
  const shown = [];
  const v = replaySource({ fetchRecords: async () => { throw new Error('404'); }, show: (x) => shown.push(x), renderer: () => ({}), schedule() {}, frame() {}, now: () => 0, hash: () => '', setHash() {}, ui() {} });
  await assert.rejects(v.loaded);
  assert.equal(shown.at(-1).status, 'Recording unavailable');
});

test('replayPath and clock', () => {
  assert.equal(replayPath('/tok/r/20261011T231955-7f3a/'), true);
  assert.equal(replayPath('/tok/'), false);
  assert.equal(replayPath('/tok/r/a/b/'), false);
  assert.equal(clock(61500), '1:01.5');
  assert.equal(clock(0), '0:00.0');
});
