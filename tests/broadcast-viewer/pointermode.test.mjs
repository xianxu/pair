// The pointer page's wiring (#412 M4 review, BR-15): caps toggles capture
// and the hint; strokes become overlapping batches; release and turning off
// end a stroke.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { pointerMode } from '../../cmd/internal/broadcast/web/viewer.js';

function fakes() {
  const listeners = {};
  const classes = new Set();
  const stage = {
    addEventListener: (n, f) => { listeners[n] = f; },
    querySelector: () => ({ getBoundingClientRect: () => ({ left: 0, top: 0, width: 800, height: 240 }) }),
    setPointerCapture() {},
    classList: { toggle: (c, on) => (on ? classes.add(c) : classes.delete(c)) },
  };
  const hint = { textContent: '', classList: { remove() {} } };
  const posts = [];
  const ticks = [];
  let stopped = 0;
  const mode = pointerMode({ cols: 80, rows: 24 }, stage, hint, {
    post: (b) => posts.push(b),
    every: (fn) => { ticks.push(fn); return ticks.length; },
    stop: () => { stopped++; },
  });
  const at = (col, row) => ({ clientX: col * 10 + 5, clientY: row * 10 + 5, pointerId: 1, preventDefault() {} });
  const fire = (name, e) => listeners[name](e);
  return { mode, classes, hint, posts, ticks, fire, at, stopped: () => stopped };
}

test('caps toggles capture and the hint', () => {
  const f = fakes();
  f.mode.setOn(true);
  assert.ok(f.classes.has('pointer'));
  assert.match(f.hint.textContent, /You can point/);
  f.mode.setOn(false);
  assert.ok(!f.classes.has('pointer'));
  assert.match(f.hint.textContent, /off/);
});

test('nothing is posted while pointing is off or outside the screen', () => {
  const f = fakes();
  f.fire('pointerdown', f.at(3, 3));
  assert.equal(f.posts.length, 0);
  f.mode.setOn(true);
  f.fire('pointerdown', { clientX: 900, clientY: 5, pointerId: 1, preventDefault() {} });
  assert.equal(f.posts.length, 0);
});

test('a stroke posts overlapping batches and ends on release', () => {
  const f = fakes();
  f.mode.setOn(true);
  f.fire('pointerdown', f.at(1, 1));
  assert.deepEqual(f.posts[0], { cols: 80, rows: 24, down: true, points: [[1, 1]] });
  f.fire('pointermove', f.at(2, 1));
  f.fire('pointermove', f.at(2, 1)); // same cell: not repeated
  f.fire('pointermove', f.at(3, 2));
  f.ticks[0]();
  assert.deepEqual(f.posts[1], { cols: 80, rows: 24, down: true, points: [[1, 1], [2, 1], [3, 2]] });
  f.fire('pointermove', f.at(4, 2));
  f.fire('pointerup', f.at(4, 2));
  assert.deepEqual(f.posts[2], { cols: 80, rows: 24, down: false, points: [[3, 2], [4, 2]] });
  assert.equal(f.stopped(), 1);
});

test('turning pointing off mid-stroke ends the stroke', () => {
  const f = fakes();
  f.mode.setOn(true);
  f.fire('pointerdown', f.at(1, 1));
  f.fire('pointermove', f.at(5, 1));
  f.mode.setOn(false);
  const last = f.posts[f.posts.length - 1];
  assert.equal(last.down, false);
  const n = f.posts.length;
  f.fire('pointermove', f.at(6, 1));
  f.fire('pointerup', f.at(6, 1));
  assert.equal(f.posts.length, n);
});
