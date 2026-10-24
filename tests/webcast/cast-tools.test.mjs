import test from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {mountCastTools} from '../../cmd/internal/webcast/web/cast-tools.js';

// Stateful DOM fixture: moving a node retires its old parent, listeners remain
// on the node, and insertBefore reproduces browser restoration semantics.
class Node extends Element {
  constructor(tag) { super(tag); this.hidden = false; this.disabled = false; }
  get nextSibling() { return this.parentNode?.children[this.parentNode.children.indexOf(this) + 1] || null; }
  append(...nodes) { for (const node of nodes) { node.remove(); node.parentNode = this; this.children.push(node); } }
  insertBefore(node, before) { if (!before) return this.append(node); assert.equal(before.parentNode, this); node.remove(); this.children.splice(this.children.indexOf(before), 0, node); node.parentNode = this; }
  remove() { if (this.parentNode) this.parentNode.children.splice(this.parentNode.children.indexOf(this), 1); this.parentNode = null; }
  removeEventListener(type, fn) { this.listeners[type] = (this.listeners[type] || []).filter(listener => listener !== fn); }
  fire(type) { const event = {target: this, stopPropagation() { this.stopped = true; }}; for (let node = this; node; node = node.parentNode) { for (const fn of node.listeners[type] || []) fn(event); if (event.stopped) break; } return event; }
  getBoundingClientRect() { throw new Error('toolbar must never measure rows or DOM geometry'); }
}
function fixture(layout) {
  const doc = {createElement: tag => new Node(tag)}, win = new Node('window');
  win.innerWidth = 800; win.innerHeight = 600;
  const parent = new Node('main'), old = new Node('div'); parent.append(old);
  const controls = Object.fromEntries(['live', 'pointer', 'pen', 'colors', 'note', 'undo', 'redo', 'done', 'retry', 'continueDrawing', 'swatches'].map(name => { const node = new Node(name === 'colors' ? 'select' : 'button'); node.textContent = name; old.append(node); return [name, node]; }));
  return {doc, win, parent, old, controls, mount: () => mountCastTools(doc, win, {parent, liveButton: controls.live, layout})};
}
function buttons(node) { return node.children.flatMap(child => child.tagName === 'button' || child.tagName === 'select' ? [child.textContent] : buttons(child)); }
function adopt(f, tools) { tools.annotation({pointer: f.controls.pointer, note: f.controls.note, swatches: f.controls.swatches}); tools.sketch(f.controls); }

test('one column adopts the actual controls in tool order and preserves policy state and listeners', () => {
  const f = fixture(), {undo, retry, pen} = f.controls; let clicks = 0;
  pen.addEventListener('click', () => clicks++); undo.disabled = true; retry.hidden = true;
  const tools = f.mount(); adopt(f, tools);
  assert.equal(tools.root.parentNode, f.parent);
  assert.deepEqual(buttons(tools.root), ['live', 'pointer', 'pen', 'swatches', 'colors', 'note', 'undo', 'redo', 'done', 'retry', 'continueDrawing']);
  pen.fire('click'); assert.equal(clicks, 1);
  assert.equal(undo.disabled, true); assert.equal(undo.hidden, false); assert.equal(retry.hidden, true);
  tools.sketchVisibility(false); assert.equal(tools.groups.penColors.hidden, false,'palette stays selectable before recording begins'); assert.equal(tools.groups.history.hidden, true);
  tools.sketchVisibility(true); assert.equal(tools.groups.penColors.hidden, false); assert.equal(retry.hidden, true);
  assert.equal(tools.groups.colors.parentNode, tools.root);
  assert.equal(f.old.children.length, 0);
});

test('uses layout bounds on refresh and resize without reading DOM geometry', () => {
  const f = fixture(() => ({toolbar: {left: f.win.innerWidth - 68, top: 108, width: 56, height: f.win.innerHeight - 120}}));
  const tools = f.mount(); adopt(f, tools);
  assert.equal(tools.root.style.left, '732px'); assert.equal(tools.root.style.top, '108px');
  assert.equal(tools.root.style.maxHeight, '480px');
  f.win.innerWidth = 320; f.win.innerHeight = 160; f.win.fire('resize');
  assert.equal(tools.root.style.left, '252px'); assert.equal(tools.root.style.maxHeight, '40px');
  f.win.innerWidth = 768; tools.refresh(); assert.equal(tools.root.style.left, '700px');
});

test('default layout reserves the protected rail and stops overlay gestures without canceling native controls', () => {
  const f = fixture(), tools = f.mount(); adopt(f, tools); let gestures = 0;
  f.parent.addEventListener('pointerdown', () => gestures++);
  const event = f.controls.pen.fire('pointerdown'); assert.equal(gestures, 0); assert.equal(event.prevented, undefined);
  assert.equal(tools.root.style.left, '732px'); assert.equal(tools.root.style.top, '12px');
});

test('dispose restores node order and original listeners, and retires owner listeners exactly once', () => {
  const f = fixture(), original = [...f.old.children], tools = f.mount(); adopt(f, tools);
  let clicks = 0; f.controls.pen.addEventListener('click', () => clicks++);
  tools.dispose(); tools.dispose();
  assert.deepEqual(f.old.children, original); assert.deepEqual(f.parent.children, [f.old]);
  assert.equal((f.win.listeners.resize || []).length, 0);
  assert.equal(Object.values(tools.root.listeners).flat().length, 0);
  f.controls.pen.fire('click'); assert.equal(clicks, 1);
  const left = tools.root.style.left; f.win.innerWidth = 1; tools.refresh(); tools.sketch(f.controls);
  assert.equal(tools.root.style.left, left); assert.deepEqual(f.old.children, original);
});

test('dispose leaves a control moved by its adapter alone', () => {
  const f = fixture(), tools = f.mount(); adopt(f, tools);
  const replacement = new Node('div'); f.parent.append(replacement); replacement.append(f.controls.pen);
  tools.dispose(); assert.equal(f.controls.pen.parentNode, replacement);
});
