// Note boxes (#488), shared by the terminal viewer and the web cast: a drag
// draws a box, a tap gets the default box. Run by TestViewerNode.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { gestureBox, dragMin, boxCentre } from '../../cmd/internal/broadcast/web/collab.js';

const cells = (min = 1) => ({ min, cells: true });
const px = (min = 8) => ({ min });
const grid = { x: 0, y: 0, w: 200, h: 50 };

test('cells: a tap gets a tenth of the screen each way, centred on it', () => {
  assert.deepEqual(gestureBox([100, 25], [100, 25], grid, cells()), [90, 23, 20, 5]);
});

test('cells: the default box shifts, never shrinks, to stay on the screen', () => {
  assert.deepEqual(gestureBox([0, 0], [0, 0], grid, cells()), [0, 0, 20, 5]);
  assert.deepEqual(gestureBox([199, 49], [199, 49], grid, cells()), [180, 45, 20, 5]);
});

test('cells: a small screen still gets a box the pin fits in', () => {
  assert.deepEqual(gestureBox([3, 2], [3, 2], { x: 0, y: 0, w: 8, h: 4 }, cells()), [2, 2, 2, 1]);
  assert.deepEqual(gestureBox([0, 0], [0, 0], { x: 0, y: 0, w: 1, h: 1 }, cells()), [0, 0, 1, 1]);
});

test('cells: a drag covers every cell it touched, either direction', () => {
  assert.deepEqual(gestureBox([10, 5], [14, 8], grid, cells()), [10, 5, 5, 4]);
  assert.deepEqual(gestureBox([14, 8], [10, 5], grid, cells()), [10, 5, 5, 4]);
  assert.deepEqual(gestureBox([10, 5], [10, 7], grid, cells()), [10, 5, 2, 3]); // a one-column drag widens to the pin
});

test('cells: a drag past the edge is clamped to the screen', () => {
  assert.deepEqual(gestureBox([190, 40], [260, 90], grid, cells()), [190, 40, 10, 10]);
  assert.deepEqual(gestureBox([199, 3], [199, 9], grid, cells()), [198, 3, 2, 7]);
});

test('touch needs a longer drag than a mouse', () => {
  assert.equal(dragMin('cells', 'touch'), 2);
  assert.equal(dragMin('cells', 'mouse'), 1);
  assert.equal(dragMin('cells', 'pen'), 1);
  assert.equal(dragMin('px', 'touch'), 16);
  assert.equal(dragMin('px', 'mouse'), 8);
  // A one-cell wobble is a drag for a mouse and a tap for a finger.
  assert.deepEqual(gestureBox([100, 25], [101, 25], grid, cells(dragMin('cells', 'mouse'))), [100, 25, 2, 1]);
  assert.deepEqual(gestureBox([100, 25], [101, 25], grid, cells(dragMin('cells', 'touch'))), [90, 23, 20, 5]);
  assert.deepEqual(gestureBox([100, 100], [110, 100], { x: 0, y: 0, w: 1000, h: 800 }, px(dragMin('px', 'touch'))), [50, 60, 100, 80]);
});

test('px: the page view is scrolled; boxes are in top-page px, rounded to 0.1', () => {
  const view = { x: 0, y: 1200, w: 1280, h: 720 };
  assert.deepEqual(gestureBox([640, 1560], [643, 1562], view, px()), [576, 1524, 128, 72]);
  assert.deepEqual(gestureBox([5, 1201], [5, 1201], view, px()), [0, 1200, 128, 72]);
  assert.deepEqual(gestureBox([10.04, 1300], [60, 1350.26], view, px()), [10, 1300, 50, 50.3]);
  assert.deepEqual(gestureBox([1200, 1900], [1400, 2000], view, px()), [1200, 1900, 80, 20]); // clamped to the view
  assert.deepEqual(gestureBox([10, 1300], [60, 1300], view, px()), [10, 1300, 50, 1]);
});

test('a note anchors at its box centre', () => {
  assert.deepEqual(boxCentre([10, 20, 30, 40]), [25, 40]);
});
