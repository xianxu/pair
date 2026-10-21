// The viewer's model (#460) against the Go reducer: every fixture's pinned
// canonical hash, the reducer's refusals, and the catch-up rule.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { Model, goJSON } from '../../cmd/internal/webcast/web/model.js';
import { fixtures, records, pins } from './fixtures.mjs';

const sha = (s) => createHash('sha256').update(s).digest('hex');

test('shared capability recording removes old annotations and admits fresh ones', () => {
  const recs = readFileSync(new URL('../../cmd/internal/webcast/recording/testdata/capability-events.jsonl', import.meta.url), 'utf8').trim().split('\n').map(JSON.parse);
  const old = reduce(recs, 4);
  assert.deepEqual(old.notes.map(x => x.note.id), ['old']);
  assert.equal(old.strokes.length, 1);
  const cleared = reduce(recs, 6);
  assert.deepEqual(cleared.notes, []); assert.deepEqual(cleared.strokes, []);
  const end = reduce(recs);
  assert.deepEqual(end.notes.map(x => x.note.id), ['new']);
  assert.deepEqual(end.strokes, []);
});

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

// Joining mid-cast is checked against Go's own CatchUp in headless Chrome
// (replay_test.go, TestLiveReplayJoinMidCast), so the rule has one source.

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
  m.apply({ t: 3, k: 'future-kind', payload: 'v1' });
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

test('annotation clear preserves replay history but removes marks through pause/resume', () => {
  const note = {t:2,k:'note',id:'n',author:{id:'v1'},at:{node:3},text:'hello'};
  const stroke = {t:2,k:'stroke',id:'s',author:{id:'v1'},pts:[[1,2,0]]};
  const recs = [head,seg(1),doc,note,stroke,{t:3,k:'stop',why:'pause'},
    {t:4,k:'annotation-clear',who:{id:'operator'}}];
  assert.equal(reduce(recs,2).notes.length,1);
  const m=reduce(recs);
  assert.equal(m.notes.length,0); assert.equal(m.strokes.length,0);
  m.apply({t:5,k:'annotation-clear',who:{id:'operator'}});
  m.apply({...seg(2,'resume'),t:6});
  assert.equal(m.notes.length,0); assert.equal(m.strokes.length,0);
  for(const who of [null,{}, {id:'v1'}]) {
    assert.throws(()=>m.apply({t:7,k:'annotation-clear',who}));
  }
  m.apply({t:8,k:'stop',why:'end'});
  assert.throws(()=>m.apply({t:9,k:'annotation-clear',who:{id:'operator'}}),/cast is over/);
});

test('note edits copy creation records, retain view identity, and attribute resolution', () => {
 const original = {t:2,k:'note',id:'n',author:{id:'a'},owner:'opaque',text:'original',at:{node:3}};
 const m=reduce([head,seg(1),doc,original]); const v=m.notes[0];
 m.apply({t:3,k:'note-edit',id:'n',text:'updated'});
 assert.equal(v.note.text,'updated'); assert.equal(v.note.edited,true); assert.equal(v.note.t,2);
 assert.equal(original.text,'original'); assert.equal(m.notes[0],v); assert.equal(v.revision,1);
 m.apply({t:4,k:'note-state',id:'n',state:'resolved',who:{id:'resolver',name:'Ada'}});
 assert.deepEqual(v.resolvedBy,{id:'resolver',name:'Ada'});
 assert.throws(()=>m.apply({t:5,k:'note-edit',id:'n',text:'no'}));
 m.apply({t:5,k:'note-state',id:'n',state:'open'}); assert.equal(v.resolvedBy,null);
 for(const text of ['', ' \n', 'x'.repeat(1025), '\u202e', '\ud800']) {
  assert.throws(()=>m.apply({t:6,k:'note-edit',id:'n',text})); assert.equal(m.t,5);
 }
});

test('historical open note edits preserve its original capture, anchor and document identity',()=>{
 const original={t:2,k:'note',id:'old',author:{id:'a'},owner:'owner',text:'draw-time text',capture:'original-capture',region:[1,2,3,4],at:{node:3,route:'https://old/'}};
 const m=reduce([head,seg(1),doc,original]);const view=m.notes[0];
 m.apply({...seg(2,'navigate'),t:3});m.apply({...doc,t:4});
 m.apply({t:5,k:'note-edit',id:'old',text:'edited from off-view list'});
 assert.equal(view.note.text,'edited from off-view list');assert.equal(view.revision,1);assert.equal(view.seg,1);
 assert.equal(view.note.capture,original.capture);assert.deepEqual(view.note.at,original.at);assert.deepEqual(view.note.region,original.region);assert.equal(original.text,'draw-time text');
 m.apply({t:6,k:'stop',why:'pause'});assert.throws(()=>m.apply({t:7,k:'note-edit',id:'old',text:'paused'}),/live/);
});
