// Unit tests for the pointer rail's dock (#449): a column on the screen's
// right edge, centered on its height, outside the stage so swatch taps never
// start a stroke. Run by cmd/internal/broadcast's TestViewerNode.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { railAt, RAIL_INSET } from '../../cmd/internal/broadcast/web/viewer.js';

const web = new URL('../../cmd/internal/broadcast/web/', import.meta.url);

test('docks to the right edge and middle of a letterboxed screen', () => {
  // A 600×300 screen centered in a 1000px-wide window, 100px from the top.
  const rect = { top: 100, height: 300, right: 800 };
  assert.deepEqual(railAt(rect, 1000), { top: 250, right: 200 + RAIL_INSET });
});

test('a screen as wide as the window keeps the inset', () => {
  assert.deepEqual(railAt({ top: 0, height: 400, right: 390 }, 390), { top: 200, right: RAIL_INSET });
});

test('a screen overflowing the window never pushes the rail off it', () => {
  assert.equal(railAt({ top: 0, height: 10, right: 500 }, 390).right, RAIL_INSET);
});

test('the rail is outside the stage and only its buttons take pointer events', () => {
  const html = readFileSync(new URL('index.html', web), 'utf8');
  const stage = html.match(/<main id="stage">.*?<\/main>/s)[0];
  assert.ok(!stage.includes('id="rail"'), 'rail must not be inside #stage');
  assert.match(html, /<div id="rail"><div id="swatches"/);
  const css = readFileSync(new URL('viewer.css', web), 'utf8');
  const block = (sel) => css.match(new RegExp(`(^|\\n)${sel} \\{([^}]*)\\}`))[2];
  assert.match(block('#rail'), /pointer-events: none/);
  assert.match(block('\\.swatch'), /pointer-events: auto/);
});

test('a stale screen dims the rail with it', () => {
  const css = readFileSync(new URL('viewer.css', web), 'utf8');
  assert.match(css, /#stage\.stale,\s*#rail\.stale \{[^}]*opacity/);
});
