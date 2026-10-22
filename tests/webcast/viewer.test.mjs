// The live viewer's shell (#460), against a fake EventSource and renderer:
// the stream's records reach the model and the renderer, a reconnect's
// catch-up starts over, a refused record resyncs, and the bar follows.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { view, connect, fit, toPage, fromPage } from '../../cmd/internal/webcast/web/viewer.js';

class FakeEventSource {
  constructor() { this.listeners = {}; this.readyState = 0; this.closed = false; }
  addEventListener(n, fn) { this.listeners[n] = fn; }
  close() { this.closed = true; this.readyState = 2; }
  emit(n, data) { this.readyState = 1; this.listeners[n]({ data: JSON.stringify(data) }); }
}

function harness(extra = {}) {
  const h = { shown: [], sources: [], drawn: [], retries: [], frames: 0, flushes: 0, revoked: 0 };
  h.v = connect({
    open: () => { const es = new FakeEventSource(); h.sources.push(es); return es; },
    renderer: (model) => ({ model, apply: (r) => h.drawn.push(r.k), flush: () => h.flushes++, revokeAll: () => h.revoked++ }),
    show: (v) => h.shown.push(v),
    frame: () => h.frames++,
    schedule: (fn) => fn(),
    retry: (fn) => h.retries.push(fn),
    ...extra,
  });
  h.es = () => h.sources.at(-1);
  h.last = () => h.shown.at(-1);
  h.rec = (r) => h.es().emit('rec', r);
  return h;
}

const head = { t: 0, k: 'head', format: 'couch-webcast', v: 1, id: 'r', started: 's' };
const seg = (n, why = 'live') => ({ t: n, k: 'seg', seg: n, why, tab: { id: 'a', url: 'http://app/', title: 'App' }, vp: { w: 1280, h: 800, dpr: 1 } });
const doc = (t, id) => ({ t, k: 'doc', doc: id, url: 'http://app/', tree: { id, y: 'doc', c: [{ id: id + 1, y: 'el', n: 'html' }] } });

test('before any record the bar shows the browser state', () => {
  const h = harness();
  h.es().emit('state', { live: false });
  assert.deepEqual(h.last(), { status: 'Not live', live: false, title: '', url: '', page: false });
  h.es().emit('state', { live: true, url: 'http://app/', title: 'App' });
  assert.equal(h.last().status, 'Live');
  assert.equal(h.last().page, false, 'live without records: no page yet');
});

test('records reach the model and the renderer; the bar follows the segment', () => {
  const h = harness();
  for (const r of [head, seg(1), doc(1, 1)]) h.rec(r);
  assert.deepEqual(h.drawn, ['head', 'seg', 'doc']);
  assert.equal(h.v.model.top.id, 1);
  assert.deepEqual(h.last(), { status: 'Live', live: true, title: 'App', url: 'http://app/', page: true });
  assert.ok(h.flushes > 0 && h.frames > 0);
  h.rec({ t: 2, k: 'nav', url: 'http://app/b' });
  assert.equal(h.last().url, 'http://app/b');
  h.rec({ t: 3, k: 'stop', why: 'pause' });
  assert.equal(h.last().status, 'Paused');
  h.rec(seg(4, 'resume'));
  h.rec(doc(4, 1));
  assert.equal(h.last().status, 'Live');
  h.rec({ t: 5, k: 'stop', why: 'end' });
  assert.equal(h.last().status, 'Cast ended');
});

test("a reconnect's catch-up starts from a fresh model", () => {
  const h = harness();
  for (const r of [head, seg(1), doc(1, 1)]) h.rec(r);
  const first = h.v.model;
  for (const r of [head, seg(1), doc(1, 1)]) h.rec(r); // EventSource reconnected: catch-up again
  assert.notEqual(h.v.model, first);
  assert.equal(h.v.model.seg, 1);
  assert.equal(h.revoked, 1, "the old drawing's blob: URLs are released");
});

test('a refused record resyncs: the stream closes and reopens after a back-off', () => {
  const h = harness();
  for (const r of [head, seg(1), doc(1, 1)]) h.rec(r);
  const es = h.es();
  h.rec({ t: 2, k: 'rm', id: 99 }); // a node this viewer never saw
  assert.ok(es.closed);
  assert.equal(h.last().status, 'Reconnecting…');
  assert.equal(h.v.model.head, null, 'the page is dropped');
  assert.equal(h.retries.length, 1);
  h.retries[0]();
  assert.notEqual(h.es(), es);
  for (const r of [head, seg(1), doc(1, 1)]) h.rec(r);
  assert.equal(h.last().status, 'Live');
});

test('onRecord sees each accepted record with the model', () => {
  const h = harness();
  const seen = [];
  h.v.onRecord((r, m) => seen.push(`${r.k}@${m.seg}`));
  for (const r of [head, seg(1), doc(1, 1), { t: 2, k: 'ctl', op: 'request', who: {id:'v1'}, pending:[{id:'v1'}] }]) h.rec(r);
  assert.deepEqual(seen, ['head@0', 'seg@1', 'doc@1', 'ctl@1']);
});

test('the end event closes the stream and says so', () => {
  const h = harness();
  h.es().emit('end', { reason: 'ended' });
  assert.ok(h.es().closed);
  assert.equal(h.last().status, 'Cast ended');
});

test('fit scales the operator viewport to the area, whole and centred, never relaid out', () => {
  assert.deepEqual(fit(640, 800, { w: 1280, h: 800 }), { scale: 0.5, w: 640, h: 400, left: 0, top: 200 });
  assert.deepEqual(fit(2000, 800, { w: 1280, h: 800 }), { scale: 1, w: 1280, h: 800, left: 360, top: 0 });
  const ipad = fit(1024, 700, { w: 1280, h: 800 });
  assert.ok(Math.abs(ipad.scale - 0.8) < 1e-9 && ipad.w === 1024);
  assert.equal(fit(0, 0, { w: 1, h: 1 }).w, 0);
});

test('toPage and fromPage map between the scaled box and the page', () => {
  const box = { left: 100, top: 50 };
  assert.deepEqual(toPage(150, 100, box, 0.5, [0, 300]), [100, 400]);
  const [x, y] = fromPage(100, 400, 0.5, [0, 300]);
  assert.deepEqual([x + box.left, y + box.top], [150, 100]);
});

test('view: ended wins; a page wins over the browser state', () => {
  assert.equal(view({ seg: 1, live: true, url: 'u', title: '' }, { live: false }, false).title, 'u');
  assert.equal(view({ seg: 1, live: true }, null, true).status, 'Cast ended');
  assert.equal(view(null, null, false).status, 'Connecting…');
});

test('a hover on a node this viewer lacks never forces a resync', () => {
  const h = harness();
  for (const r of [head, seg(1), doc(1, 1)]) h.rec(r);
  const es = h.es();
  h.rec({ t: 2, k: 'hover', id: 999 });
  assert.equal(es.closed, false);
  assert.equal(h.retries.length, 0);
  assert.equal(h.v.model.hover, 0);
});

// pair#495: a stream the server closed for good (a rotated code ends the
// session) returns the page to the lock; a closed stream whose session holds
// is shown as disconnected, and a cast that ended asks nothing.
test('a closed stream whose session is gone reloads into the lock page', async () => {
  let relocked = 0;
  const h = harness({ sessionLost: async () => true, relock: () => { relocked++; } });
  h.es().emit('state', { live: true });
  h.es().close();
  h.es().onerror();
  await new Promise((r) => setImmediate(r));
  assert.equal(relocked, 1);
});

test('a closed stream whose session holds says disconnected', async () => {
  let relocked = 0;
  const h = harness({ sessionLost: async () => false, relock: () => { relocked++; } });
  h.es().emit('state', { live: true });
  h.es().close();
  h.es().onerror();
  await new Promise((r) => setImmediate(r));
  assert.equal(relocked, 0);
  assert.equal(h.last().status, 'Disconnected');
});

test('an ended cast never probes its session', async () => {
  let probed = 0;
  const h = harness({ sessionLost: async () => { probed++; return true; }, relock: () => {} });
  h.es().emit('end', {});
  h.es().onerror();
  await new Promise((r) => setImmediate(r));
  assert.equal(probed, 0);
});

test('a stream the browser is reconnecting is not probed', async () => {
  let probed = 0;
  const h = harness({ sessionLost: async () => { probed++; return true; }, relock: () => {} });
  h.es().readyState = 0;
  h.es().onerror();
  await new Promise((r) => setImmediate(r));
  assert.equal(probed, 0);
});

const capabilities = (a, d, generation = 1) => Object.fromEntries([['annotation', a], ['drive', d]].map(([k, on]) => [k, {supported:true, permitted:true, allowed:on, state:on?'on':'off', generation, attempt:0}]));
test('capability availability starts closed, replays latest and intersects the role', () => {
 const h=harness(), seen=[];
 h.v.onCapabilities(c=>seen.push(c));
 assert.equal(seen.at(-1).annotation.allowed,false);
 for(const a of [false,true]) for(const d of [false,true]) {
  h.es().emit('capabilities',capabilities(a,d));
  assert.equal(seen.at(-1).annotation.allowed,a);assert.equal(seen.at(-1).drive.allowed,d);
 }
 let late;h.v.onCapabilities(c=>late=c);assert.equal(late.drive.allowed,true);
 const denied=capabilities(true,true,2);denied.annotation.permitted=false;
 h.es().emit('capabilities',denied);assert.equal(seen.at(-1).annotation.allowed,false);
 h.es().emit('capabilities',capabilities(true,true,1));assert.equal(seen.at(-1).annotation.allowed,false);
 h.es().emit('capabilities',{});assert.equal(seen.at(-1).drive.allowed,false);
});
test('disconnect fails closed and retired event sources cannot mutate replacements', async () => {
 let settle; const h=harness({sessionLost:()=>new Promise(r=>settle=r),relock:()=>{throw Error('stale relock');}});
 h.es().emit('capabilities',capabilities(true,true));
 const old=h.es();old.readyState=2;old.onerror();
 assert.equal(h.v.capabilities.drive.allowed,false);
 for(const r of [head,seg(1),doc(1,1)])h.rec(r);
 h.rec({t:2,k:'rm',id:99});h.retries[0]();
 const replacement=h.es();replacement.emit('capabilities',capabilities(false,false,2));
 old.emit('capabilities',capabilities(true,true,3));old.emit('end',{});old.emit('rec',head);
 settle(true);await Promise.resolve();
 assert.equal(h.v.capabilities.drive.allowed,false);assert.equal(replacement.closed,false);assert.equal(h.v.model.head,null);
});

test('live and reconnect viewer install the reducer sketch bootstrap before live points', async () => {
  const {Model}=await import('../../cmd/internal/webcast/web/model.js');
  const {catchUp}=await import('../../cmd/internal/webcast/web/replay.js');
  const records=[head,seg(1),doc(2,1),{t:3,k:'sketch-begin',id:'sk',author:{id:'operator',color:'#ff0000'},owner:'operator',doc:1,tab:'a',vp:{w:100,h:100,dpr:1},operation:'op0',sequence:0,digest:'a'.repeat(64)}];
  const source=new Model();for(const r of records)source.apply(r);
  const h=harness();for(let reconnect=0;reconnect<2;reconnect++)for(const r of catchUp(records))h.rec(r);
  assert.equal(h.retries.length,0);assert.deepEqual(h.v.model.sketchProjection(),source.sketchProjection());
  const stroke={t:4,k:'sketch-stroke-begin',sketch:'sk',id:'p',at:{node:9,box:[0,0,10,10]},width:4,operation:'op1',sequence:1,digest:'a'.repeat(64)};
  source.apply(stroke);h.rec(stroke);assert.deepEqual(h.v.model.sketchProjection(),source.sketchProjection());
});
test('drawing capabilities preserve safe epoch/holder and fail closed on disconnect', () => {
  const h = harness();
  h.es().emit('capabilities', {
    ...capabilities(true, true),
    drawing: { epoch: 7, holder: 'seat', ready: true },
  });
  assert.deepEqual(h.v.capabilities.drawing, { epoch: 7, holder: 'seat', ready: true });
  h.es().emit('capabilities', {
    ...capabilities(true, true),
    drawing: { epoch: 6, holder: 'other', ready: true },
  });
  assert.equal(h.v.capabilities.drawing.holder, 'seat');
  h.es().onerror();
  assert.equal(h.v.capabilities.drawing.ready, false);
  h.es().emit('capabilities', {
    ...capabilities(true, true),
    drawing: { epoch: 8, holder: 'seat', ready: true },
  });
  assert.equal(h.v.capabilities.drawing.ready, true);
});
