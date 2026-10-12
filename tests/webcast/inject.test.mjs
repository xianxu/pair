// The in-page control (#454), run the way Chrome runs it: as a classic script
// in its own global, here a vm context over a minimal DOM stub.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

const source = readFileSync(new URL('../../cmd/internal/webcast/inject.js', import.meta.url), 'utf8');

class El {
  constructor(tag) {
    this.tag = tag; this.attrs = {}; this.children = []; this.listeners = {};
    this.style = {}; this.textContent = ''; this.className = ''; this.shadow = null;
  }
  setAttribute(k, v) { this.attrs[k] = v; }
  appendChild(c) { this.children.push(c); return c; }
  addEventListener(n, fn) { (this.listeners[n] ||= []).push(fn); }
  attachShadow(opts) { this.shadowMode = opts.mode; this.shadow = new El('#shadow'); return this.shadow; }
  click() { for (const fn of this.listeners.click || []) fn({ preventDefault() {}, stopPropagation() {} }); }
}

function page({ top = true, readyState = 'complete', visible = true, focused = true } = {}) {
  const sent = [];
  const docListeners = {}, winListeners = {};
  const document = {
    readyState,
    visibilityState: visible ? 'visible' : 'hidden',
    documentElement: new El('html'),
    hasFocus: () => focused,
    createElement: (t) => new El(t),
    addEventListener: (n, fn) => (docListeners[n] ||= []).push(fn),
  };
  const ctx = { document, __couchcast: (s) => sent.push(JSON.parse(s)) };
  ctx.window = ctx;
  ctx.window.top = top ? ctx : {};
  ctx.addEventListener = (n, fn) => (winListeners[n] ||= []).push(fn);
  vm.createContext(ctx);
  const run = () => vm.runInContext(source, ctx);
  const fire = (target, name) => {
    for (const fn of (target === 'doc' ? docListeners : winListeners)[name] || []) fn();
  };
  const host = () => document.documentElement.children.find((c) => c.tag === 'couchcast-control');
  const button = () => host()?.shadow.children.find((c) => c.tag === 'button');
  return { ctx, sent, run, fire, document, host, button };
}

test('nothing runs in a subframe', () => {
  const p = page({ top: false });
  p.run();
  assert.deepEqual(p.sent, []);
  assert.equal(p.host(), undefined);
});

test('reports hello and visibility on load, then on each change', () => {
  const p = page({ visible: true, focused: false });
  p.run();
  assert.deepEqual(p.sent, [{ t: 'hello' }, { t: 'vis', visible: true, focused: false }]);
  p.document.visibilityState = 'hidden';
  p.fire('doc', 'visibilitychange');
  p.fire('win', 'focus');
  assert.deepEqual(p.sent.slice(2), [
    { t: 'vis', visible: false, focused: false },
    { t: 'vis', visible: false, focused: false },
  ]);
});

test('control lives in a closed shadow root on a marked host', () => {
  const p = page();
  p.run();
  assert.equal(p.host().attrs['data-couchcast-ui'], '');
  assert.equal(p.host().shadowMode, 'closed');
  assert.equal(p.button().textContent, '● Go live');
});

test('state set from Couch changes the label; a click asks to toggle', () => {
  const p = page();
  p.run();
  p.ctx.__couchcastSet(true);
  assert.equal(p.button().textContent, '❚❚ Pause');
  assert.equal(p.button().className, 'live');
  p.button().click();
  assert.deepEqual(p.sent.at(-1), { t: 'toggle' });
  p.ctx.__couchcastSet(false);
  assert.equal(p.button().textContent, '● Go live');
});

test('injected twice into one document: one control, one hello', () => {
  const p = page();
  p.run();
  p.run();
  assert.equal(p.document.documentElement.children.length, 1);
  assert.equal(p.sent.filter((m) => m.t === 'hello').length, 1);
});

test('at document start the control mounts on DOMContentLoaded', () => {
  const p = page({ readyState: 'loading' });
  p.run();
  assert.equal(p.host(), undefined);
  p.ctx.__couchcastSet(true); // state may arrive before the button exists
  p.fire('doc', 'DOMContentLoaded');
  assert.equal(p.button().textContent, '❚❚ Pause');
});
