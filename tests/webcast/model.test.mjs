// The viewer's model (#460) against the Go reducer: every fixture's pinned
// canonical hash, the reducer's refusals, and the catch-up rule.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { Model, goJSON } from '../../cmd/internal/webcast/web/model.js';
import { fixtures, records, pins } from './fixtures.mjs';

const sha = (s) => createHash('sha256').update(s).digest('hex');

function reduce(recs, t = Infinity) {
  const m = new Model();
  for (const r of recs) { if (r.t > t) break; m.apply(r); }
  return m;
}

for (const name of fixtures) {
  test(`${name}: canonical equals the Go reducer at every pin`, () => {
    const recs = records(name);
    for (const p of pins(name)) {
      assert.equal(sha(goJSON(reduce(recs, p.t).canonical())), p.hash, `${name} at t=${p.t}`);
    }
  });
}

// catchUp is recording.CatchUp, for the join-mid-cast checks below.
export function catchUp(recs) {
  let last = -1, doc = -1;
  recs.forEach((r, i) => { if (r.k === 'seg') { last = i; if (r.why !== 'resume') doc = i; } });
  if (last < 0) return [...recs];
  if (doc < 0) doc = 0;
  const out = [], kept = new Set();
  recs.forEach((r, i) => {
    if (i === doc && doc < last) { out.push(r); return; }
    switch (r.k) {
      case 'head': case 'asset': if (i < last) { out.push(r); return; } break;
      case 'note': if (i >= doc) { kept.add(r.id); out.push(r); } return;
      case 'note-state': if (i >= doc && kept.has(r.id)) out.push(r); return;
      case 'stroke': if (i >= doc) out.push(r); return;
    }
    if (i >= last) out.push(r);
  });
  return out;
}

test('joining at any cut of segments rebuilds the same page', () => {
  const recs = records('segments');
  for (let k = 1; k <= recs.length; k++) {
    const whole = goJSON(reduce(recs.slice(0, k)).canonical());
    const joined = reduce([...catchUp(recs.slice(0, k))]);
    assert.equal(goJSON(joined.canonical()), whole, `cut ${k}`);
  }
});

const head = { t: 0, k: 'head', format: 'couch-webcast', v: 1, id: 'x', started: 's' };
const seg = (n, why = 'live') => ({ t: 1, k: 'seg', seg: n, why, tab: { id: 'a', url: 'http://x/' }, vp: { w: 100, h: 50, dpr: 1 } });
const doc = { t: 1, k: 'doc', doc: 1, url: 'http://x/', tree: { id: 1, y: 'doc', c: [{ id: 2, y: 'el', n: 'html', c: [{ id: 3, y: 'el', n: 'body' }] }] } };

test('refusals leave the state as it was, as the reducer does', () => {
  const m = reduce([head, seg(1), doc]);
  const before = goJSON(m.canonical());
  for (const bad of [
    { t: 2, k: 'ins', p: 99, after: 0, node: { id: 4, y: 'el', n: 'p' } }, // unknown parent
    { t: 2, k: 'ins', p: 3, after: 0, node: { id: 2, y: 'el', n: 'p' } }, // id already live
    { t: 2, k: 'attr', id: 1, name: 'x', v: 'y' }, // not an element
    { t: 2, k: 'rm', id: 1 }, // a document
    { t: 0, k: 'title', v: 'back in time' },
    { t: 2, k: 'seg', seg: 1, why: 'live', tab: {}, vp: { w: 1, h: 1, dpr: 1 } }, // segment not increasing
  ]) {
    assert.throws(() => m.apply(bad), undefined, JSON.stringify(bad));
    assert.equal(goJSON(m.canonical()), before);
  }
  assert.throws(() => new Model().apply(seg(1)), /first record must be head/);
  assert.throws(() => new Model().apply({ ...head, v: 2 }), /version 2/);
});

test('changes outside a live segment are refused; unknown kinds are ignored', () => {
  const m = reduce([head, seg(1), doc, { t: 2, k: 'stop', why: 'pause' }]);
  assert.throws(() => m.apply({ t: 3, k: 'title', v: 'x' }), /no live segment/);
  m.apply({ t: 3, k: 'ctl', who: 'v1' });
});

test('hover and pointer are tracked outside the canonical', () => {
  const m = reduce([head, seg(1), doc]);
  const before = goJSON(m.canonical());
  m.apply({ t: 2, k: 'hover', id: 3 });
  m.apply({ t: 2, k: 'ptr', x: 10, y: 20 });
  assert.equal(m.hover, 3);
  assert.deepEqual(m.ptr, { x: 10, y: 20, t: 2 });
  assert.equal(goJSON(m.canonical()), before);
  m.apply({ t: 3, k: 'hover', id: 99 }); // a node never sent: no refusal, no hover
  assert.equal(m.hover, 0);
  m.apply({ t: 3, k: 'hover', id: 3 });
  m.apply({ t: 3, k: 'rm', id: 3 });
  assert.equal(m.hover, 0, 'removing the hovered node clears hover');
});

test('goJSON escapes as Go does', () => {
  assert.equal(goJSON({ a: '<b>&\u2028' }), '{\n "a": "\\u003cb\\u003e\\u0026\\u2028"\n}');
});

test('stop end is final: a seg or ctl after it is refused, as the reducer does', () => {
  const m = reduce([head, seg(1), doc, { t: 2, k: 'stop', why: 'end' }]);
  assert.throws(() => m.apply({ ...seg(2), t: 3 }), /cast is over/);
  assert.throws(() => m.apply({ t: 3, k: 'ctl', act: 'grant' }), /cast is over/);
  const paused = reduce([head, seg(1), doc, { t: 2, k: 'stop', why: 'pause' }]);
  paused.apply({ ...seg(2, 'resume'), t: 3 });
});
