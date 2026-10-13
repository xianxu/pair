// What the recording can't hold, marked in the viewer (pair#513): one mark
// per uncaptured element at its drawn box, labelled, following the box as it
// changes and gone when the element is.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { label, mountPlaceholders } from '../../cmd/internal/webcast/web/placeholders.js';
import { liveSizeRule, isUncaptured } from '../../cmd/internal/webcast/web/render.js';

function fakeDoc() {
  const el = () => {
    const e = { style: {}, className: '', textContent: '', children: [], parent: null };
    e.append = (c) => { c.parent = e; e.children.push(c); };
    e.remove = () => { if (e.parent) e.parent.children.splice(e.parent.children.indexOf(e), 1); e.parent = null; };
    Object.defineProperty(e, 'firstChild', { get: () => e.children[0] });
    return e;
  };
  const box = el();
  box.getBoundingClientRect = () => ({ left: 10, top: 20 });
  return { box, doc: { createElement: el, getElementById: (id) => (id === 'box' ? box : null) } };
}

test('the label names the kind; a frame reads as a frame', () => {
  assert.equal(label('canvas'), 'canvas: not captured');
  assert.equal(label('iframe'), 'frame: not captured');
});

test('marks follow the uncaptured elements and their boxes', () => {
  const { box, doc } = fakeDoc();
  let list = [{ id: 7, kind: 'canvas' }, { id: 9, kind: 'video' }, { id: 11, kind: 'object' }];
  const boxes = new Map([[7, [0, 0, 320, 80]], [9, [0, 100, 320, 180]], [11, [0, 0, 0, 0]]]);
  const win = { requestAnimationFrame: (f) => f(), addEventListener() {} };
  const v = { uncaptured: () => list, boxOf: (id) => boxes.get(id), fromPage: (x, y) => [x + 10, y + 20], onRecord() {} };
  const p = mountPlaceholders(doc, win, v);
  const layer = box.children[0];
  assert.equal(layer.className, 'uncaptured');
  const marks = () => layer.children.map((m) => [m.firstChild.textContent, m.style.left, m.style.top, m.style.width, m.style.height]);
  // A zero box (not laid out) gets no mark; the others sit at their box,
  // in the box element's px.
  assert.deepEqual(marks(), [['canvas: not captured', '0px', '0px', '320px', '80px'], ['video: not captured', '0px', '100px', '320px', '180px']]);
  // The operator's canvas grew: its mark follows.
  boxes.set(7, [0, 0, 400, 100]);
  p.draw();
  assert.deepEqual(marks()[0], ['canvas: not captured', '0px', '0px', '400px', '100px']);
  // The video went: so does its mark.
  list = [{ id: 7, kind: 'canvas' }];
  p.draw();
  assert.deepEqual(marks().map((m) => m[0]), ['canvas: not captured']);
});

// The renderer's live-size rule (render.js liveSizeRule), pure: what each
// uncaptured element is sized to, and when it isn't sized at all.

test('liveSizeRule: the live box, border-box with its aspect ratio, never [hidden]', () => {
  const r = liveSizeRule(7, 'video', [0, 10, 346, 174], true, {});
  assert.match(r, /^:where\(\[data-couch-live-size="7"\]:not\(\[hidden\]\)\)\{/);
  for (const part of ['display:inline-block', 'box-sizing:border-box', 'width:346px', 'height:174px', 'aspect-ratio:346/174']) assert.ok(r.includes(part), part);
});

test('liveSizeRule: before any layout a canvas takes its attributes; after, no box means not rendered', () => {
  assert.ok(liveSizeRule(1, 'canvas', undefined, false, { width: '320', height: '80' }).includes('width:320px;height:80px'));
  assert.ok(liveSizeRule(1, 'canvas', undefined, false, {}).includes('width:300px;height:150px'));
  assert.equal(liveSizeRule(1, 'video', undefined, false, { width: '320', height: '80' }), '', 'a video waits for its box');
  assert.equal(liveSizeRule(1, 'canvas', undefined, true, { width: '320', height: '80' }), '', 'laid out without a box: hidden');
  assert.equal(liveSizeRule(1, 'canvas', [0, 0, 0, 0], true, {}), '', 'a zero box sizes nothing');
});

test('isUncaptured: the kinds the recording cannot hold', () => {
  for (const k of ['canvas', 'video', 'object', 'embed', 'iframe', 'frame']) assert.ok(isUncaptured(k), k);
  for (const k of ['img', 'div', 'picture', 'audio']) assert.ok(!isUncaptured(k), k);
});
