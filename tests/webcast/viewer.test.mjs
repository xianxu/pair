// The web cast's stub viewer (#454), against a fake EventSource and DOM.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { view, connect, render } from '../../cmd/internal/webcast/web/viewer.js';

class FakeEventSource {
  constructor() { this.listeners = {}; this.readyState = 0; this.closed = false; }
  addEventListener(n, fn) { this.listeners[n] = fn; }
  close() { this.closed = true; this.readyState = 2; }
  emit(n, data) { this.readyState = 1; this.listeners[n]({ data: JSON.stringify(data) }); }
}

function harness() {
  const h = { shown: [] };
  h.es = connect({ open: () => new FakeEventSource(), show: (v) => h.shown.push(v) });
  h.last = () => h.shown.at(-1);
  return h;
}

test('not live shows nothing of the page', () => {
  const h = harness();
  h.es.emit('state', { live: false });
  assert.deepEqual(h.last(), { status: 'Not live', live: false, title: '', url: '' });
});

test('live shows title and URL, and follows changes', () => {
  const h = harness();
  h.es.emit('state', { live: true, url: 'http://localhost:3000/', title: 'Shop' });
  assert.deepEqual(h.last(), { status: 'Live', live: true, title: 'Shop', url: 'http://localhost:3000/' });
  h.es.emit('state', { live: true, url: 'http://localhost:3000/cart' });
  assert.equal(h.last().title, 'http://localhost:3000/cart');
  h.es.emit('state', { live: false });
  assert.equal(h.last().url, '');
});

test('end closes the stream and says so', () => {
  const h = harness();
  h.es.emit('state', { live: true, url: 'u', title: 't' });
  h.es.emit('end', { reason: 'ended' });
  assert.ok(h.es.closed);
  assert.deepEqual(h.last(), view(null, true));
});

test('a dropped connection is not shown as live', () => {
  const h = harness();
  h.es.emit('state', { live: true, url: 'u', title: 't' });
  h.es.readyState = 0;
  h.es.onerror();
  assert.equal(h.last().live, false);
});

test('render sets text only', () => {
  const els = {};
  const doc = { title: '', getElementById: (id) => (els[id] ||= { textContent: '', className: '' }) };
  const evil = '<img src=x onerror=alert(1)>';
  render(doc, { status: 'Live', live: true, title: evil, url: evil });
  assert.equal(els.title.textContent, evil);
  assert.equal(els.url.textContent, evil);
  assert.ok(!('innerHTML' in els.title));
  assert.equal(els.status.className, 'status live');
});
