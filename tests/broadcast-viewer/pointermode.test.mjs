// The pointer page's wiring (#412 M4 review, BR-15): caps toggles capture
// (#447: no hint bar; the mode button carries a tooltip); strokes become overlapping batches; release and turning off
// end a stroke.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { pointerMode, pointerColor, colorStore, renderSwatches } from '../../cmd/internal/broadcast/web/viewer.js';

function fakes() {
  const listeners = {};
  const classes = new Set();
  const stage = {
    addEventListener: (n, f) => { listeners[n] = f; },
    querySelector: () => ({ getBoundingClientRect: () => ({ left: 0, top: 0, width: 800, height: 240 }) }),
    setPointerCapture() {},
    classList: { toggle: (c, on) => (on ? classes.add(c) : classes.delete(c)) },
  };
  const posts = [];
  const ticks = [];
  let stopped = 0;
  const mode = pointerMode({ cols: 80, rows: 24 }, stage, {
    post: (b) => posts.push(b),
    every: (fn) => { ticks.push(fn); return ticks.length; },
    stop: () => { stopped++; },
  });
  const at = (col, row) => ({ clientX: col * 10 + 5, clientY: row * 10 + 5, pointerId: 1, preventDefault() {} });
  const fire = (name, e) => listeners[name](e);
  return { mode, classes, posts, ticks, fire, at, stopped: () => stopped };
}

test('caps toggles capture', () => {
  const f = fakes();
  f.mode.setOn(true);
  assert.ok(f.classes.has('pointer'));
  f.mode.setOn(false);
  assert.ok(!f.classes.has('pointer'));
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
  assert.deepEqual(f.posts[0], { cols: 80, rows: 24, down: true, color: 0, points: [[1, 1]] });
  f.fire('pointermove', f.at(2, 1));
  f.fire('pointermove', f.at(2, 1)); // same cell: not repeated
  f.fire('pointermove', f.at(3, 2));
  f.ticks[0]();
  assert.deepEqual(f.posts[1], { cols: 80, rows: 24, down: true, color: 0, points: [[1, 1], [2, 1], [3, 2]] });
  f.fire('pointermove', f.at(4, 2));
  f.fire('pointerup', f.at(4, 2));
  assert.deepEqual(f.posts[2], { cols: 80, rows: 24, down: false, color: 0, points: [[3, 2], [4, 2]] });
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

// Pointer colors (#446).
const PALETTE = ['#ffaf00', '#56b4e9', '#009e73', '#f0e442', '#d55e00', '#cc79a7'];

function colorFakes(storage) {
  const f = {};
  const listeners = {};
  const stage = {
    addEventListener: (n, fn) => { listeners[n] = fn; },
    querySelector: () => ({ getBoundingClientRect: () => ({ left: 0, top: 0, width: 800, height: 240 }) }),
    setPointerCapture() {},
    classList: { toggle() {} },
  };
  f.posts = [];
  f.shown = null;
  f.mode = pointerMode({ cols: 80, rows: 24 }, stage, {
    post: (b) => f.posts.push(b),
    every: () => 1,
    stop() {},
    store: colorStore(storage),
    swatches: {
      show: (palette, current, pick) => { f.shown = { palette, current, pick }; },
      hide: () => { f.shown = null; },
    },
  });
  f.tap = () => {
    listeners.pointerdown({ clientX: 15, clientY: 15, pointerId: 1, preventDefault() {} });
    listeners.pointerup({});
  };
  return f;
}

function memoryStorage() {
  const m = new Map();
  return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, String(v)) };
}

test('an unpicked viewer posts the color Couch assigned', () => {
  const f = colorFakes(memoryStorage());
  f.mode.setColors(PALETTE, 2);
  f.mode.setOn(true);
  assert.deepEqual(f.shown.palette, PALETTE);
  assert.equal(f.shown.current, 2);
  f.tap();
  assert.equal(f.posts[0].color, 2);
});

test('a pick is posted, survives a reload, and outranks the assignment', () => {
  const storage = memoryStorage();
  const f = colorFakes(storage);
  f.mode.setColors(PALETTE, 0);
  f.mode.setOn(true);
  f.shown.pick(4);
  assert.equal(f.shown.current, 4);
  f.tap();
  assert.equal(f.posts[0].color, 4);
  const reloaded = colorFakes(storage);
  reloaded.mode.setColors(PALETTE, 1);
  reloaded.mode.setOn(true);
  assert.equal(reloaded.shown.current, 4);
  reloaded.tap();
  assert.equal(reloaded.posts[0].color, 4);
});

test('blocked storage still lets the viewer pick', () => {
  const throwing = { getItem() { throw new Error('denied'); }, setItem() { throw new Error('denied'); } };
  for (const storage of [throwing, null]) {
    const f = colorFakes(storage);
    f.mode.setColors(PALETTE, 1);
    f.mode.setOn(true);
    f.shown.pick(5);
    f.tap();
    assert.equal(f.posts[0].color, 5);
  }
});

test('swatches show only while pointing is on, and never for a bad palette', () => {
  const f = colorFakes(memoryStorage());
  f.mode.setColors(PALETTE, 0);
  assert.equal(f.shown, null);
  f.mode.setOn(true);
  assert.ok(f.shown);
  f.mode.setOn(false);
  assert.equal(f.shown, null);
  f.mode.setColors(['#ffaf00', 'url(x)'], 0);
  f.mode.setOn(true);
  assert.equal(f.shown, null);
  f.tap();
  assert.equal(f.posts[0].color, 0);
});

test('pointerColor and colorStore refuse out-of-range values', () => {
  assert.equal(pointerColor(6, 1, 6), 1);
  assert.equal(pointerColor(-1, 1, 6), 1);
  assert.equal(pointerColor(null, 3, 6), 3);
  const storage = memoryStorage();
  storage.setItem('couch.pointerColor', 'red');
  assert.equal(colorStore(storage).get(), null);
});

test('renderSwatches draws one pressed-state button per color', () => {
  const made = [];
  const el = { replaceChildren: (...kids) => { el.kids = kids; } };
  const make = () => {
    const attrs = {};
    const b = { style: {}, attrs, setAttribute: (k, v) => { attrs[k] = v; }, addEventListener: (n, fn) => { b.click = fn; } };
    made.push(b);
    return b;
  };
  let picked = null;
  renderSwatches(el, PALETTE, 3, (i) => { picked = i; }, make);
  assert.equal(el.kids.length, 6);
  assert.equal(el.kids[3].className, 'swatch on');
  assert.equal(el.kids[3].attrs['aria-pressed'], 'true');
  assert.equal(el.kids[0].style.background, '#ffaf00');
  assert.equal(el.kids[2].title, 'Pointer color 3', 'each swatch explains itself on hover (#447 M3)');
  el.kids[1].click();
  assert.equal(picked, 1);
});

// Tap routing (#447): a 💭 opens its note on any page; comment mode drafts
// on an empty cell and never posts; turning pointing off resets the mode.
function noteFakes() {
  const listeners = {};
  const classes = new Set();
  const stage = {
    addEventListener: (n, f) => { listeners[n] = f; },
    querySelector: () => ({ getBoundingClientRect: () => ({ left: 0, top: 0, width: 800, height: 240 }) }),
    setPointerCapture() {},
    classList: { toggle: (c, on) => (on ? classes.add(c) : classes.delete(c)) },
  };
  const posts = [];
  const calls = [];
  let shownMode = null;
  let shownTip = null;
  let toggle = null;
  const list = { guard: 12, notes: [{ id: 7, col: 10, row: 5, color: 0, text: 'hi' }] };
  const mode = pointerMode({ cols: 80, rows: 24 }, stage, {
    post: (b) => posts.push(b),
    every: () => 1,
    stop() {},
    notes: () => list,
    dialog: {
      draft: (payload) => calls.push(['draft', payload]),
      view: (note, opts) => calls.push(['view', note.id, opts]),
      close: () => calls.push(['close']),
    },
    modeButton: { show: (m, t, tip) => { shownMode = m; toggle = t; shownTip = tip; }, hide: () => { shownMode = null; } },
  });
  const at = (col, row) => ({ clientX: col * 10 + 5, clientY: row * 10 + 5, pointerId: 1, preventDefault() {} });
  const fire = (name, e) => listeners[name](e);
  return { mode, classes, posts, calls, fire, at, shownMode: () => shownMode, shownTip: () => shownTip, toggle: () => toggle() };
}

test('a view page opens a 💭 read-only and posts nothing', () => {
  const f = noteFakes();
  f.fire('pointerdown', f.at(11, 5));
  assert.deepEqual(f.calls, [['view', 7, { resolvable: false }]]);
  assert.equal(f.posts.length, 0);
  f.fire('pointerdown', f.at(3, 3));
  assert.equal(f.calls.length, 1, 'a tap elsewhere does nothing on a view page');
  assert.equal(f.shownMode(), null, 'no mode button without pointing');
});

test('pointer mode: a 💭 opens resolvable and is not pointed at', () => {
  const f = noteFakes();
  f.mode.setOn(true);
  assert.equal(f.shownMode(), 'point');
  f.fire('pointerdown', f.at(10, 5));
  assert.deepEqual(f.calls.at(-1), ['view', 7, { resolvable: true }]);
  assert.equal(f.posts.length, 0);
  f.fire('pointerdown', f.at(3, 3));
  assert.equal(f.posts.length, 1, 'elsewhere it still points');
});

test('comment mode drafts on an empty cell without posting', () => {
  const f = noteFakes();
  f.mode.setOn(true);
  f.toggle();
  assert.equal(f.shownMode(), 'comment');
  assert.ok(f.classes.has('comment'));
  assert.match(f.shownTip(), /leave a note/, 'the mode button explains itself in its tooltip');
  f.fire('pointerdown', f.at(79, 3));
  assert.deepEqual(f.calls.at(-1), ['draft', { cols: 80, rows: 24, col: 78, row: 3 }]);
  f.fire('pointerdown', f.at(10, 5));
  assert.deepEqual(f.calls.at(-1), ['view', 7, { resolvable: true }], 'a 💭 opens rather than drafts');
  const n = f.calls.length;
  f.fire('pointerdown', f.at(3, 23));
  assert.equal(f.calls.length, n, 'no draft on the guarded controls');
  assert.equal(f.posts.length, 0, 'comment mode never posts a point or a note');
});

test('turning pointing off resets to pointer mode and closes the dialog', () => {
  const f = noteFakes();
  f.mode.setOn(true);
  f.toggle();
  f.mode.setOn(false);
  assert.deepEqual(f.calls.at(-1), ['close']);
  assert.equal(f.shownMode(), null);
  assert.ok(!f.classes.has('comment'));
  f.mode.setOn(true);
  assert.equal(f.shownMode(), 'point');
});
