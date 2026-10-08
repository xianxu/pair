// The viewer's connection state machine (#395 BR-6), against a fake
// EventSource. A viewer must never show a frozen screen as if it were live.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { connect, RETRY_DELAYS } from '../../cmd/internal/broadcast/web/viewer.js';

const CONNECTING = 0, OPEN = 1, CLOSED = 2;

class FakeEventSource {
  constructor() {
    this.readyState = CONNECTING;
    this.listeners = {};
    this.onerror = null;
    this.closed = false;
  }
  addEventListener(name, fn) { this.listeners[name] = fn; }
  close() { this.closed = true; this.readyState = CLOSED; }
  emit(name, data) { this.readyState = OPEN; this.listeners[name]({ data: JSON.stringify(data) }); }
  fail(state) { this.readyState = state; this.onerror?.(); }
}

function harness() {
  const h = { sources: [], rendered: [], resets: 0, status: 'Connecting…', stale: false, timers: [] };
  h.deps = {
    open: () => { const es = new FakeEventSource(); h.sources.push(es); return es; },
    render: (m) => h.rendered.push(m),
    theme: (t) => { h.theme = t; },
    reset: () => { h.resets++; },
    show: (text) => { h.status = text; },
    setStale: (on) => { h.stale = on; },
    later: (fn, ms) => h.timers.push({ fn, ms }),
  };
  h.es = () => h.sources[h.sources.length - 1];
  h.fire = () => { const t = h.timers.shift(); t.fn(); return t.ms; };
  connect(h.deps);
  return h;
}

const frame = { cols: 10, rows: 2, b: '' };

test('a frame renders and clears any status', () => {
  const h = harness();
  h.es().emit('frame', frame);
  assert.deepEqual(h.rendered, [frame]);
  assert.equal(h.status, '');
  assert.equal(h.stale, false);
});

test('a transient error while the browser reconnects dims and says so', () => {
  const h = harness();
  h.es().emit('frame', frame);
  h.es().fail(CONNECTING);
  assert.equal(h.stale, true);
  assert.equal(h.status, 'Reconnecting…');
  assert.equal(h.timers.length, 0, 'the browser retries by itself; no timer of ours');
  h.es().emit('frame', frame);
  assert.equal(h.stale, false);
  assert.equal(h.status, '');
});

test('a connection closed for good dims, says so, and retries with backoff', () => {
  const h = harness();
  h.es().emit('frame', frame);
  h.es().fail(CLOSED);
  assert.equal(h.stale, true);
  assert.match(h.status, /^Disconnected/);
  const delays = [];
  for (let i = 0; i < RETRY_DELAYS.length; i++) {
    delays.push(h.fire());
    h.es().fail(CLOSED);
  }
  assert.deepEqual(delays, RETRY_DELAYS);
  assert.equal(h.status, 'Disconnected');
  assert.equal(h.timers.length, 0, 'gives up after the last delay');
  assert.equal(h.stale, true);
});

test('a viewer turned away at the start does not sit on "Connecting…"', () => {
  const h = harness();
  h.es().fail(CLOSED); // e.g. 503 at the viewer cap
  assert.match(h.status, /^Disconnected/);
  assert.equal(h.timers.length, 1);
});

test('a successful reconnect resets the backoff', () => {
  const h = harness();
  h.es().fail(CLOSED);
  assert.equal(h.fire(), RETRY_DELAYS[0]);
  h.es().emit('frame', frame);
  h.es().fail(CLOSED);
  assert.equal(h.fire(), RETRY_DELAYS[0]);
});

test('end resets the screen, shows the reason and never retries', () => {
  const h = harness();
  h.es().emit('frame', frame);
  h.es().emit('end', { reason: 'the operator stopped broadcasting' });
  assert.equal(h.resets, 1);
  assert.equal(h.es().closed, true);
  assert.equal(h.status, 'Broadcast ended: the operator stopped broadcasting');
  h.es().fail(CLOSED);
  assert.equal(h.timers.length, 0);
  assert.equal(h.status, 'Broadcast ended: the operator stopped broadcasting');
  assert.equal(h.stale, false);
});

test('a theme event reaches the renderer', () => {
  const h = harness();
  h.es().emit('theme', { foreground: '#eeeeee' });
  assert.deepEqual(h.theme, { foreground: '#eeeeee' });
});
