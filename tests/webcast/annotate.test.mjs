// Pointing and notes in the web cast viewer (#470): the pure helpers.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { identityStore, validName, strokesAt, chunker, strokeStore, liveNotes, noteBox, regionOf, noteDraft, readPalette, MAX_POINTS, FADE_MS } from '../../cmd/internal/webcast/web/annotate.js';

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
  // A reconnect's catch-up, in stream order: the document's strokes oldest
  // first, then the records after them. Neither old stroke is drawn.
  s.seen(1000); s.record({ id: 'bo.2', t: 1000, color: '#f00', pts: [[1, 1, 0]] }, 200);
  s.seen(31000); s.record({ id: 'bo.3', t: 31000, color: '#f00', pts: [[1, 1, 0]] }, 200);
  s.seen(61000);
  assert.equal(s.visible(200).length, 0);
  // A live stroke at the newest time is.
  s.seen(61010); s.record({ id: 'bo.4', t: 61010, color: '#f00', pts: [[1, 1, 0]] }, 300);
  assert.deepEqual(s.visible(300).map((x) => x.id), ['bo.4']);
  s.clear();
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

test('every note gesture is a box: a drag, or the default box for a tap (#488)', () => {
  const view = { x: 0, y: 400, w: 1000, h: 600 };
  assert.deepEqual(regionOf([500, 700], [505, 703], view, 'mouse'), [450, 670, 100, 60]); // a tap
  assert.deepEqual(regionOf([50, 440], [10, 410], view, 'mouse'), [10, 410, 40, 30]);
  assert.deepEqual(regionOf([10, 410], [60, 412], view, 'mouse'), [10, 410, 50, 2]);
  assert.deepEqual(regionOf([500, 700], [512, 700], view, 'mouse'), [500, 700, 12, 1]);
  assert.deepEqual(regionOf([500, 700], [512, 700], view, 'touch'), [450, 670, 100, 60]); // a finger's wobble is a tap
  assert.deepEqual(noteDraft(3, 7, [1.234, 2, 3, 4], [5, 6, 7, 8]), { seg: 3, node: 7, box: [1.2, 2, 3, 4], region: [5, 6, 7, 8] });
});

test('a stored name is read by the board\'s rules, else asked again', () => {
  assert.equal(validName('  Ana  '), 'Ana');
  assert.equal(validName('x'.repeat(40)), 'x'.repeat(40));
  assert.equal(validName('x'.repeat(41)), '');
  assert.equal(validName('A\nB'), '');
  assert.equal(validName('A\u202eB'), '');
  for (const c of ['\u2067', '\u061c', '\u180e', '\ufff9', '\u{e0041}', '\u0085']) assert.equal(validName(`A${c}B`), '', `U+${c.codePointAt(0).toString(16)}`);
  const s = memStorage();
  s.setItem('couch.viewerName', 'n'.repeat(80));
  assert.equal(identityStore(s, crypto).name(), '');
});

test('strokesAt: what a reader at t shows, for replay', () => {
  const st = (id, t, seg, pts) => ({ stroke: { id, t, color: '#f00', pts }, seg });
  const model = { docSeg: 2, strokes: [
    st('old', 100, 1, [[0, 0, 0]]),             // an earlier document
    st('a', 1000, 2, [[1, 1, 0], [2, 2, 100]]),
    st('a', 1150, 2, [[3, 3, 0]]),              // a's next chunk
    st('b', 5000, 2, [[9, 9, 0]]),              // later
  ] };
  assert.deepEqual(strokesAt(model, 1050).map((s) => [s.id, s.pts.length]), [['a', 1]]); // only the points drawn by t
  const at = strokesAt(model, 1200);
  assert.deepEqual(at.map((s) => [s.id, s.pts.length, s.alpha.toFixed(2)]), [['a', 3, '0.97']]);
  assert.equal(strokesAt(model, 1150 + FADE_MS).length, 0);
  assert.deepEqual(strokesAt(model, 5000).map((s) => s.id), ['b']);
  assert.deepEqual(strokesAt({ docSeg: 0, strokes: [] }, 1), []);
});

// A pin is its note's box; a box too thin for the 💭 chip carries it
// outside its corner (#488).
test('pins: the box, and the chip outside a thin one', async () => {
  const { pinLayer, CHIP_ROOM } = await import('../../cmd/internal/webcast/web/annotate.js');
  const el = () => {
    const cls = new Set();
    return {
      classList: { toggle: (c, on) => (on ? cls.add(c) : cls.delete(c)), contains: (c) => cls.has(c) },
      set className(v) { cls.clear(); v.split(' ').filter(Boolean).forEach((c) => cls.add(c)); },
      style: { setProperty() {} }, dataset: {}, append() {}, addEventListener() {}, setAttribute() {}, remove() {},
    };
  };
  const doc = { createElement: el };
  const pins = [];
  const layer = pinLayer(doc, { append: (p) => pins.push(p) }, () => {});
  const place = (p, b) => { p.style.width = `${b[2]}px`; p.style.height = `${b[3]}px`; };
  const note = (id, region) => ({ id, region, text: 'x', author: { name: 'A', color: '#112233' } });
  layer.draw([note('wide', [0, 0, 200, 80]), note('thin', [0, 0, 200, CHIP_ROOM - 1])], () => null, place);
  assert.equal(pins.length, 2);
  assert.ok(!pins[0].classList.contains('thin'));
  assert.ok(pins[1].classList.contains('thin'));
});

// palette.json is input from outside the module (pair#506): the swatches'
// colors and whether this link may annotate. Anything else, including the
// bare array an older server sent, reads as "this cast takes no notes".
test('readPalette: palette and annotate, or null for any other shape', () => {
  assert.deepEqual(readPalette({ palette: ['#112233', '#aabbcc'], annotate: true }), { palette: ['#112233', '#aabbcc'], annotate: true });
  assert.deepEqual(readPalette({ palette: ['#112233'], annotate: false }), { palette: ['#112233'], annotate: false });
  for (const bad of [null, 'x', ['#112233'], {}, { palette: [], annotate: true }, { palette: ['red'], annotate: true },
    { palette: ['#112233'] }, { palette: ['#112233'], annotate: 'yes' }, { palette: '#112233', annotate: true }]) {
    assert.equal(readPalette(bad), null, JSON.stringify(bad));
  }
});
