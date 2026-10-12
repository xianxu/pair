// Pointing and notes in the web cast viewer (#470): the pure helpers.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { identityStore, chunker, strokeStore, liveNotes, noteBox, regionOf, noteDraft, MAX_POINTS, FADE_MS } from '../../cmd/internal/webcast/web/annotate.js';

const crypto = { getRandomValues: (b) => { b.fill(0xab); return b; } };
const memStorage = () => { const m = new Map(); return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, String(v)), m }; };

test('identity: made once per browser, a name asked once; storage may throw', () => {
  const s = memStorage();
  const a = identityStore(s, crypto);
  assert.match(a.id(), /^w[0-9a-f]{12}$/);
  assert.equal(a.name(), '');
  a.setName('Ana');
  const b = identityStore(s, crypto);
  assert.deepEqual(b.viewer(), { id: a.id(), name: 'Ana' });
  s.setItem('couch.viewerId', 'operator'); // reserved: never reused
  assert.notEqual(identityStore(s, crypto).id(), 'operator');
  const broken = { getItem() { throw new Error('blocked'); }, setItem() { throw new Error('blocked'); } };
  const c = identityStore(broken, crypto);
  c.setName('Bo');
  assert.equal(c.name(), 'Bo');
  assert.match(c.id(), /^w/);
});

test('chunker: points relative to the chunk, duplicates dropped, at most MAX_POINTS a post', () => {
  const c = chunker();
  c.add(10, 20, 1000); c.add(10, 20, 1005); c.add(11.26, 20, 1016);
  assert.deepEqual(c.take(), [[[10, 20, 0], [11.3, 20, 16]]]);
  assert.deepEqual(c.take(), []);
  for (let i = 0; i < MAX_POINTS + 3; i++) c.add(i, 0, 2000 + i);
  const posts = c.take();
  assert.equal(posts.length, 2);
  assert.equal(posts[0].length, MAX_POINTS);
  assert.deepEqual(posts[1][0], [MAX_POINTS, 0, 0]);
});

test('strokes: chunks join by id, fade out, catch-up and own echoes are not drawn', () => {
  const s = strokeStore();
  s.seen(5000);
  s.record({ id: 'bo.1', t: 5000, color: '#f00', pts: [[1, 1, 0], [2, 2, 10]] }, 100);
  s.record({ id: 'bo.1', t: 5050, color: '#f00', pts: [[3, 3, 0]] }, 150);
  let vis = s.visible(150);
  assert.equal(vis.length, 1);
  assert.deepEqual(vis[0].pts, [[1, 1], [2, 2], [3, 3]]);
  assert.equal(vis[0].alpha, 1);
  assert.ok(s.visible(150 + FADE_MS / 2)[0].alpha < 0.6);
  assert.equal(s.visible(150 + FADE_MS).length, 0);
  // A stroke from a reconnect's catch-up, long before the newest record.
  s.seen(60000);
  s.record({ id: 'bo.2', t: 5000, color: '#f00', pts: [[1, 1, 0]] }, 200);
  assert.equal(s.size, 0);
  // Own echo.
  s.record({ id: 'me.3', t: 60000, color: '#f00', pts: [[1, 1, 0]] }, 200, 'me.');
  assert.equal(s.size, 0);
  s.local('me.3', '#0f0', [[4, 4]], 200);
  assert.equal(s.visible(200)[0].color, '#0f0');
  s.clear();
  assert.equal(s.size, 0);
});

test('live notes: open ones since the document began', () => {
  const n = (id, seg, state = 'open') => ({ note: { id }, seg, state });
  const model = { docSeg: 2, notes: [n('old', 1), n('a', 2), n('done', 2, 'resolved'), n('b', 3)] };
  assert.deepEqual(liveNotes(model).map((x) => x.id), ['a', 'b']);
  assert.deepEqual(liveNotes({ docSeg: 0, notes: [] }), []);
  assert.deepEqual(liveNotes(null), []);
});

test('a note shows on its region, else its element as drawn, else the box it was made on', () => {
  const boxOf = (id) => (id === 7 ? [1, 2, 3, 4] : null);
  assert.deepEqual(noteBox({ region: [9, 9, 9, 9], at: { node: 7 } }, boxOf), [9, 9, 9, 9]);
  assert.deepEqual(noteBox({ at: { node: 7, box: [0, 0, 1, 1] } }, boxOf), [1, 2, 3, 4]);
  assert.deepEqual(noteBox({ at: { node: 8, box: [0, 0, 1, 1] } }, boxOf), [0, 0, 1, 1]);
  assert.equal(noteBox({ at: { node: 8 } }, boxOf), null);
});

test('a drag is a region; a small one is a tap', () => {
  assert.equal(regionOf([10, 10], [15, 13]), null);
  assert.deepEqual(regionOf([50, 40], [10, 10]), [10, 10, 40, 30]);
  assert.deepEqual(regionOf([10, 10], [60, 12]), [10, 10, 50, 2]);
  assert.deepEqual(noteDraft(3, 7, [1.234, 2, 3, 4], null), { seg: 3, node: 7, box: [1.2, 2, 3, 4] });
  assert.deepEqual(noteDraft(3, 7, [1, 2, 3, 4], [5, 6, 7, 8]), { seg: 3, node: 7, box: [1, 2, 3, 4], region: [5, 6, 7, 8] });
});
