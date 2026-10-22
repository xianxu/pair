// The in-page control (#454), run the way Chrome runs it: as a classic script
// in its own global, here a vm context over a minimal DOM stub.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { suppressionPolicy } from '../../cmd/internal/webcast/operator-policy.js';

const source = readFileSync(new URL('../../cmd/internal/webcast/inject.js', import.meta.url), 'utf8');

class El {
  constructor(tag) {
    this.tag = tag; this.attrs = {}; this.children = []; this.listeners = {};
    this.style = {}; this.textContent = ''; this.className = ''; this.shadow = null;
  }
  setAttribute(k, v) { this.attrs[k] = v; }
  getAttribute(k) { return this.attrs[k] ?? null; }
  appendChild(c) { this.children.push(c); return c; }
  addEventListener(n, fn) { (this.listeners[n] ||= []).push(fn); }
  attachShadow(opts) { this.shadowMode = opts.mode; this.shadow = new El('#shadow'); return this.shadow; }
  click() { for (const fn of this.listeners.click || []) fn({ preventDefault() {}, stopPropagation() {} }); }
}

function page({ top = true, readyState = 'complete', visible = true, focused = true } = {}) {
  const sent = [];
  const docListeners = {}, winListeners = {};
  const document = {
    title: '',
    readyState,
    visibilityState: visible ? 'visible' : 'hidden',
    documentElement: new El('html'),
    hasFocus: () => focused,
    createElement: (t) => new El(t),
    addEventListener: (n, fn) => (docListeners[n] ||= []).push(fn),
  };
  const ctx = { document, __couchcastSuppressionPolicy: suppressionPolicy, __couchcast: (s) => sent.push(JSON.parse(s)) };
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

test('background capture leaves an open operator UI visible and identical control refresh does no DOM writes',()=>{
 const p=page();p.run();p.ctx.__couchcastSet(true,'');
 let busy=true;p.ctx.__couchcastCaptureBusy=()=>busy;
 assert.equal(p.ctx.__couchcastUI('acquire-background','shot',0),false);
 assert.equal(p.host().hidden,false);
 let writes=0;
 const host=p.host(),button=p.button();let hidden=host.hidden,text=button.textContent;
 Object.defineProperty(host,'hidden',{get:()=>hidden,set:v=>{writes++;hidden=v;}});
 Object.defineProperty(button,'textContent',{get:()=>text,set:v=>{writes++;text=v;}});
 host.style=new Proxy(host.style,{set:(o,k,v)=>{writes++;o[k]=v;return true;}});
 const setAttribute=button.setAttribute.bind(button);button.setAttribute=(...args)=>{writes++;setAttribute(...args);};
 for(let i=0;i<10;i++)p.ctx.__couchcastSet(true,'');
 assert.equal(writes,0);
 busy=false;assert.equal(p.ctx.__couchcastUI('acquire-background','shot',0),true);
 assert.equal(host.hidden,true);p.ctx.__couchcastUI('release','shot',0);assert.equal(host.hidden,false);
});

test('nothing runs in a subframe', () => {
  const p = page({ top: false });
  p.run();
  assert.deepEqual(p.sent, []);
  assert.equal(p.host(), undefined);
});

test('reports hello, visibility and title on load, then on each change', () => {
  const p = page({ visible: true, focused: false });
  p.document.title = 'Shop';
  p.run();
  assert.deepEqual(p.sent, [{ t: 'title', title: 'Shop' }, { t: 'hello' }, { t: 'vis', visible: true, focused: false }]);
  p.document.visibilityState = 'hidden';
  p.fire('doc', 'visibilitychange');
  p.fire('win', 'focus');
  assert.deepEqual(p.sent.slice(3), [
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
  assert.equal(p.button().className, 'cast-button live');
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

test('title is reported when it changes, once per value', () => {
  const p = page({ readyState: 'loading' });
  p.run();
  assert.deepEqual(p.sent.filter((m) => m.t === 'title'), [{ t: 'title', title: '' }]);
  p.document.title = 'Cart';
  p.fire('doc', 'DOMContentLoaded');
  p.fire('win', 'load');
  assert.deepEqual(p.sent.filter((m) => m.t === 'title').map((m) => m.title), ['', 'Cart']);
});

test('the control names who drives (#467), as text, and clears when nobody does', () => {
  const p = page();
  p.run();
  const banner = () => p.host().shadow.children.find((c) => c.tag === 'div');
  p.ctx.__couchcastSet(true, 'Ana <b>');
  assert.equal(banner().textContent, 'Ana <b> is driving');
  assert.equal(banner().hidden, false);
  assert.equal(p.button().textContent, '❚❚ Pause');
  p.ctx.__couchcastSet(true, '');
  assert.equal(banner().hidden, true);
  p.ctx.__couchcastSet(false); // an old caller: nobody drives
  assert.equal(banner().hidden, true);
});

test('all Couch UI stays hidden until guard state is known, and capture tokens compose', () => {
  const p = page(); p.run();
  assert.equal(p.host().hidden, true);
  p.ctx.__couchcastSet(true, '', false);
  assert.equal(p.host().hidden, false);
  assert.equal(p.ctx.__couchcastUI('acquire', 'a', 0), true);
  assert.equal(p.host().hidden, true);
  p.ctx.__couchcastUI('acquire', 'b', 0);
  p.ctx.__couchcastUI('release', 'a', 0);
  assert.equal(p.host().hidden, true);
  p.ctx.__couchcastUI('release', 'b', 0);
  assert.equal(p.host().hidden, false);
  p.ctx.__couchcastSet(true, '', true);
  assert.equal(p.host().hidden, true);
});

test('a late control update cannot reveal UI behind a newer target guard', () => {
  const p = page(); p.run();
  p.ctx.__couchcastSet(true, '', true, 8);
  p.ctx.__couchcastSet(true, '', false, 7);
  assert.equal(p.host().hidden, true);
  p.ctx.__couchcastSet(true, '', false, 9);
  assert.equal(p.host().hidden, false);
});

test('operator installation reports readiness after an early hello', () => {
 const p = page({readyState:'loading'});
 let mounted = false;
 p.ctx.__couchcastMountOperator = () => { mounted = true; };
 p.run();
 assert.equal(mounted,false);
 assert.equal(p.sent.filter((m)=>m.t==='hello').length,1);
 p.fire('doc','DOMContentLoaded');
 assert.equal(mounted,true);
 assert.equal(p.sent.filter((m)=>m.t==='hello').length,2);
});
