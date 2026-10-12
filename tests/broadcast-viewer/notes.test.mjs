// Sticky notes on the viewer page (#447): hit-testing, placement, the byte
// cap, list parsing, and the dialog. Run by TestViewerNode via `node --test`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { noteAt, noteCell, noteBytes, noteVisible, parseNotes, noteDialog, MAX_NOTE_BYTES } from '../../cmd/internal/broadcast/web/notes.js';

const list = { guard: 12, notes: [{ id: 1, col: 10, row: 5, color: 0, text: 'a' }, { id: 2, col: 78, row: 2, color: 1, text: 'b' }] };

test('a tap on either cell of a 💭 hits it', () => {
  assert.equal(noteAt(list, 10, 5, 80, 24).id, 1);
  assert.equal(noteAt(list, 11, 5, 80, 24).id, 1);
  assert.equal(noteAt(list, 12, 5, 80, 24), null);
  assert.equal(noteAt(list, 9, 5, 80, 24), null);
  assert.equal(noteAt(list, 10, 6, 80, 24), null);
});

test('a hidden note is not hit', () => {
  // The grid shrank: note 2's second cell is off the grid.
  assert.equal(noteAt(list, 78, 2, 79, 24), null);
  // Row 5 became the last row and col 10 is inside the guard.
  assert.equal(noteAt(list, 10, 5, 80, 6), null);
  assert.ok(noteVisible({ col: 12, row: 5 }, 80, 6, 12));
});

test('a tap on the last column places the 💭 one to the left', () => {
  assert.deepEqual(noteCell([79, 3], 80), [78, 3]);
  assert.deepEqual(noteCell([5, 3], 80), [5, 3]);
});

test('the byte cap counts UTF-8 like Couch', () => {
  assert.equal(noteBytes('é'), 2);
  assert.equal(noteBytes('é'.repeat(512)), MAX_NOTE_BYTES);
  assert.ok(noteBytes('é'.repeat(513)) > MAX_NOTE_BYTES);
});

test('parseNotes keeps only well-formed entries', () => {
  const got = parseNotes({ guard: 12, notes: [{ id: 1, col: 1, row: 1, color: 0, text: 'ok', extra: 'x' }, { id: '2', col: 1, row: 1, color: 0, text: 'x' },
    { id: 3, col: 1.5, row: 1, color: 0, text: 'x' }, { id: 4, col: 1, row: 1, color: 0, text: 5 }, null] });
  assert.deepEqual(got, { guard: 12, notes: [{ id: 1, col: 1, row: 1, color: 0, text: 'ok' }] });
  for (const junk of [null, 'x', {}, { guard: -1, notes: [] }, { guard: 1, notes: 'x' }]) {
    assert.deepEqual(parseNotes(junk), { guard: 0, notes: [] });
  }
});

// A fake element: enough of the DOM for the dialog, and a guard that fails
// if anything assigns innerHTML.
function el() {
  const classes = new Set();
  const listeners = {};
  const e = {
    textContent: '',
    value: '',
    disabled: false,
    classList: {
      add: (c) => classes.add(c),
      remove: (c) => classes.delete(c),
      toggle: (c, on) => (on ? classes.add(c) : classes.delete(c)),
      has: (c) => classes.has(c),
    },
    addEventListener: (n, f) => { listeners[n] = f; },
    fire: (n, ev = {}) => listeners[n](ev),
  };
  Object.defineProperty(e, 'innerHTML', { set() { throw new Error('innerHTML assigned'); } });
  return e;
}

function dialog(respond = () => ({ status: 204, text: '' })) {
  const els = { root: el(), text: el(), input: el(), count: el(), msg: el(), save: el(), cancel: el(), resolve: el() };
  els.root.classList.add('hidden');
  const posts = [];
  const d = noteDialog(els, {
    post: (route, body) => { posts.push({ route, body }); return Promise.resolve(respond(route, body)); },
    color: () => 3,
  });
  const type = (text) => { els.input.value = text; els.input.fire('input'); };
  const settle = () => new Promise((r) => setTimeout(r, 0));
  return { d, els, posts, type, settle };
}

test('a draft is saved with its cell, grid, color and text, then closes', async () => {
  const f = dialog();
  f.d.draft([4, 2], { cols: 80, rows: 24 });
  assert.ok(!f.els.root.classList.has('hidden'));
  assert.ok(f.els.save.disabled, 'empty draft cannot be saved');
  f.type('look <b>here</b>');
  assert.equal(f.posts.length, 0, 'typing posts nothing');
  assert.equal(f.els.count.textContent, `16 / ${MAX_NOTE_BYTES}`);
  f.els.save.fire('click');
  await f.settle();
  assert.deepEqual(f.posts, [{ route: 'note', body: { cols: 80, rows: 24, col: 4, row: 2, color: 3, text: 'look <b>here</b>' } }]);
  assert.ok(f.els.root.classList.has('hidden'));
  assert.ok(!f.d.open);
});

test('Save is disabled for blank text and over the byte cap', () => {
  const f = dialog();
  f.d.draft([0, 0], { cols: 80, rows: 24 });
  f.type('   ');
  assert.ok(f.els.save.disabled);
  f.type('é'.repeat(513));
  assert.ok(f.els.save.disabled);
  f.type('é'.repeat(512));
  assert.ok(!f.els.save.disabled);
});

test('a refusal shows Couch\'s sentence as text and keeps the draft', async () => {
  const f = dialog(() => ({ status: 409, text: 'This color already has 8 notes. Resolve one first.\n' }));
  f.d.draft([0, 0], { cols: 80, rows: 24 });
  f.type('hi');
  f.els.save.fire('click');
  await f.settle();
  assert.equal(f.els.msg.textContent, 'This color already has 8 notes. Resolve one first.');
  assert.equal(f.els.input.value, 'hi');
  assert.ok(f.d.open);
});

test('Cancel discards a draft without a request', () => {
  const f = dialog();
  f.d.draft([0, 0], { cols: 80, rows: 24 });
  f.type('never sent');
  f.els.cancel.fire('click');
  assert.equal(f.posts.length, 0);
  assert.equal(f.els.input.value, '');
  assert.ok(f.els.root.classList.has('hidden'));
});

test('viewing renders text literally; Resolve only when allowed', async () => {
  const f = dialog();
  f.d.view({ id: 7, text: '<b>x</b>' }, { resolvable: false });
  assert.equal(f.els.text.textContent, '<b>x</b>');
  assert.ok(f.els.resolve.classList.has('hidden'));
  assert.ok(f.els.save.classList.has('hidden'));
  f.d.view({ id: 7, text: 'x' }, { resolvable: true });
  assert.ok(!f.els.resolve.classList.has('hidden'));
  f.els.resolve.fire('click');
  await f.settle();
  assert.deepEqual(f.posts, [{ route: 'resolve', body: { id: 7 } }]);
  assert.ok(!f.d.open);
});

test('a viewed note resolved elsewhere closes its dialog', () => {
  const f = dialog();
  f.d.view({ id: 7, text: 'x' }, { resolvable: true });
  f.d.update({ guard: 12, notes: [{ id: 7, col: 0, row: 0, color: 0, text: 'x' }] });
  assert.ok(f.d.open);
  f.d.update({ guard: 12, notes: [] });
  assert.ok(!f.d.open);
});

test('Escape closes the dialog', () => {
  const f = dialog();
  f.d.draft([0, 0], { cols: 80, rows: 24 });
  f.els.root.fire('keydown', { key: 'Escape' });
  assert.ok(!f.d.open);
});

test('an answer applies only to the dialog that sent it (#447 M2 review)', async () => {
  let release;
  const els = { root: el(), text: el(), input: el(), count: el(), msg: el(), save: el(), cancel: el(), resolve: el() };
  const d = noteDialog(els, { post: () => new Promise((r) => { release = r; }), color: () => 0 });
  d.draft([0, 0], { cols: 80, rows: 24 });
  els.input.value = 'first';
  els.input.fire('input');
  els.save.fire('click');
  els.cancel.fire('click');
  d.draft([5, 5], { cols: 80, rows: 24 });
  els.input.value = 'second draft';
  els.input.fire('input');
  release({ status: 204, text: '' });
  await new Promise((r) => setTimeout(r, 0));
  assert.ok(d.open, 'the old 204 must not close the new draft');
  assert.equal(els.input.value, 'second draft');
  assert.equal(els.msg.textContent, '');
});

test('viewing a note moves focus into the dialog so Escape reaches it', () => {
  const els = { root: el(), text: el(), input: el(), count: el(), msg: el(), save: el(), cancel: el(), resolve: el() };
  let focused = null;
  els.cancel.focus = () => { focused = 'cancel'; };
  const d = noteDialog(els, { post: () => Promise.resolve({ status: 204 }), color: () => 0 });
  d.view({ id: 1, text: 'x' }, { resolvable: false });
  assert.equal(focused, 'cancel');
  els.root.fire('keydown', { key: 'Escape' });
  assert.ok(!d.open);
});
