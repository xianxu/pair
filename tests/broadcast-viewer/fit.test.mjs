// Unit tests for the broadcast viewer's font fit (#395). Run by
// cmd/internal/broadcast's TestViewerNode via `node --test`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { nextFontSize, MIN_FONT, MAX_FONT } from '../../cmd/internal/broadcast/web/viewer.js';

test('width-bound screen shrinks to the viewport width', () => {
  // 1600px wide screen at 16px into an 800px-wide, tall viewport: halve.
  assert.equal(nextFontSize(16, 1600, 400, 800, 2000), 8);
});

test('height-bound screen shrinks to the viewport height', () => {
  assert.equal(nextFontSize(16, 400, 1000, 2000, 500), 8);
});

test('small screen grows to fill the viewport', () => {
  assert.equal(nextFontSize(10, 500, 200, 1000, 1000), 20);
});

test('rounds down to half a pixel', () => {
  // scale 0.7 → 11.2 → 11
  assert.equal(nextFontSize(16, 1000, 100, 700, 1000), 11);
  // scale 0.72 → 11.52 → 11.5
  assert.equal(nextFontSize(16, 1000, 100, 720, 1000), 11.5);
});

test('clamps to the font bounds', () => {
  assert.equal(nextFontSize(16, 100000, 100, 100, 100), MIN_FONT);
  assert.equal(nextFontSize(16, 10, 10, 100000, 100000), MAX_FONT);
});

test('a fitting screen keeps its size (stable fixed point)', () => {
  // Fits with room to spare under half a pixel of font growth.
  assert.equal(nextFontSize(16, 790, 400, 800, 1000), 16);
  // Exactly fits.
  assert.equal(nextFontSize(16, 800, 400, 800, 400), 16);
});

test('extreme aspect ratios stay finite and in bounds', () => {
  for (const [w, h] of [[1, 100000], [100000, 1], [3, 3]]) {
    const size = nextFontSize(16, w, h, 1280, 720);
    assert.ok(Number.isFinite(size) && size >= MIN_FONT && size <= MAX_FONT, String(size));
  }
});

test('unmeasurable input leaves the size unchanged', () => {
  for (const args of [[16, 0, 100, 800, 600], [16, 100, 100, 0, 600], [16, NaN, 1, 1, 1]]) {
    assert.equal(nextFontSize(...args), 16);
  }
});
