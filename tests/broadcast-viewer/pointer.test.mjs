// The pointer page's pure pieces (#412): pixel → cell mapping and stroke
// batching. Run by cmd/internal/broadcast's TestViewerNode via `node --test`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { cellAt, chunkStroke, connect, MAX_POINTS } from '../../cmd/internal/broadcast/web/viewer.js';

const rect = { left: 100, top: 50, width: 800, height: 240 }; // 80x24 cells of 10x10px

test('maps pixels to cells', () => {
  assert.deepEqual(cellAt(100, 50, rect, 80, 24), [0, 0]);
  assert.deepEqual(cellAt(109.9, 59.9, rect, 80, 24), [0, 0]);
  assert.deepEqual(cellAt(110, 60, rect, 80, 24), [1, 1]);
  assert.deepEqual(cellAt(899.9, 289.9, rect, 80, 24), [79, 23]);
});

test('points outside the screen map to nothing', () => {
  for (const [x, y] of [[99, 60], [900, 60], [150, 49], [150, 290], [NaN, 60]]) {
    assert.equal(cellAt(x, y, rect, 80, 24), null, `${x},${y}`);
  }
  assert.equal(cellAt(150, 60, { ...rect, width: 0 }, 80, 24), null);
});

test('a stroke is cut into requests that each start where the last ended', () => {
  const pts = Array.from({ length: 150 }, (_, i) => [i % 80, Math.floor(i / 80)]);
  const chunks = chunkStroke(null, pts);
  for (const c of chunks) {
    assert.ok(c.length >= 1 && c.length <= MAX_POINTS, `chunk of ${c.length}`);
  }
  for (let i = 1; i < chunks.length; i++) {
    assert.deepEqual(chunks[i][0], chunks[i - 1][chunks[i - 1].length - 1], 'chunks overlap by one point');
  }
  const flat = chunks.flatMap((c, i) => (i === 0 ? c : c.slice(1)));
  assert.deepEqual(flat, pts);
});

test('a continuing stroke starts from the previous batch\'s last point', () => {
  assert.deepEqual(chunkStroke([5, 5], [[6, 5], [7, 5]]), [[[5, 5], [6, 5], [7, 5]]]);
  assert.deepEqual(chunkStroke([5, 5], []), []);
  assert.deepEqual(chunkStroke(null, [[1, 1]]), [[[1, 1]]]);
});

test('caps events reach the page', () => {
  let caps = null;
  const listeners = {};
  connect({
    open: () => ({ addEventListener: (n, f) => { listeners[n] = f; }, close() {}, readyState: 1 }),
    render() {}, theme() {}, reset() {}, show() {}, setStale() {}, later() {},
    caps: (c) => { caps = c; },
  });
  listeners.caps({ data: '{"pointer":true}' });
  assert.deepEqual(caps, { pointer: true });
});
