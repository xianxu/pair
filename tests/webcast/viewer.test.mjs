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

function harness() {
  const h = { shown: [], sources: [], drawn: [], retries: [], frames: 0, flushes: 0 };
  h.v = connect({
    open: () => { const es = new FakeEventSource(); h.sources.push(es); return es; },
    renderer: (model) => ({ model, apply: (r) => h.drawn.push(r.k), flush: () => h.flushes++ }),
    show: (v) => h.shown.push(v),
    frame: () => h.frames++,
    schedule: (fn) => fn(),
    retry: (fn) => h.retries.push(fn),
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
  for (const r of [head, seg(1), doc(1, 1), { t: 2, k: 'ctl', who: 'v1' }]) h.rec(r);
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
