// Remote driving in the web cast viewer (#467): the pure helpers.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { batcher, gestures, keyAct, keyTracker, driveView, joinSeat, seatPoster, wireAct, splitText, FLUSH_MS, MAX_ACTS, MAX_TEXT, MAX_BODY, MAX_COORD, PACE_MS, DRAG_PX, HOVER_MS, SEAT_HEADER } from '../../cmd/internal/webcast/web/drive.js';

// A seeded random source, so a failing sequence can be replayed.
function rng(seed) {
  let s = seed >>> 0;
  return () => { s = (s * 1664525 + 1013904223) >>> 0; return s / 2 ** 32; };
}

// A fake server that answers late, in its own order, and refuses some.
function fakeServer(rand, refuseRate) {
  const sent = [];
  const waiting = [];
  let inFlight = 0, maxInFlight = 0;
  return {
    sent, waiting,
    get maxInFlight() { return maxInFlight; },
    send(batch) {
      inFlight++;
      maxInFlight = Math.max(maxInFlight, inFlight);
      sent.push(batch);
      return new Promise((resolve) => waiting.push(() => {
        inFlight--;
        resolve(rand() < refuseRate ? { status: 409, text: 'The page changed. Try again.' } : { status: 204, text: '' });
      }));
    },
  };
}

// The timers and clock a batcher uses, run by hand: fire advances time to
// the timers due and runs them.
function clock() {
  const timers = [];
  let t = 1000;
  return {
    later: (fn, ms) => { timers.push({ fn, at: t + ms }); return timers.length; },
    now: () => t,
    advance(ms) { t += ms; },
    fire() {
      if (!timers.length) return 0;
      t = Math.max(t, ...timers.map((x) => x.at));
      const ts = timers.splice(0);
      for (const x of ts) x.fn();
      return ts.length;
    },
  };
}

const tick = () => new Promise((r) => setImmediate(r));

test('batcher, property: order kept, one request in flight, moves coalesce, a refusal stops the queue', async () => {
  for (let seed = 1; seed <= 200; seed++) {
    const rand = rng(seed);
    const server = fakeServer(rand, 0.1);
    const c = clock();
    const refusals = [];
    let n = 0;
    const b = batcher({ send: (x) => server.send(x), seg: () => 7, later: c.later, now: c.now, refused: (s, t) => refusals.push([s, t]) });
    const pushed = [];
    for (let i = 0; i < 60; i++) {
      const r = rand();
      const a = r < 0.4 ? { a: 'move', x: i, y: i } : r < 0.55 ? { a: 'move', x: i, y: i, button: 'left' }
        : r < 0.7 ? { a: 'wheel', id: 3, x: 1, y: 1, dx: 0, dy: 10 } : r < 0.85 ? { a: 'click', id: 2, x: i, y: 0, n: n++ } : { a: 'text', v: `t${n++}` };
      b.push(a);
      pushed.push(a);
      if (rand() < 0.3) c.fire();
      if (rand() < 0.4 && server.waiting.length) { server.waiting.shift()(); await tick(); }
    }
    for (let guard = 0; guard < 500 && (server.waiting.length || b.pending()); guard++) {
      c.fire();
      if (server.waiting.length) server.waiting.shift()();
      await tick();
    }
    assert.ok(server.maxInFlight <= 1, `seed ${seed}: ${server.maxInFlight} in flight`);
    const got = server.sent.flatMap((x) => x.acts);
    for (const x of server.sent) {
      assert.equal(x.seg, 7);
      assert.ok(x.acts.length >= 1 && x.acts.length <= MAX_ACTS);
    }
    // Every sent non-move act (clicks, text) appears in push order, with no
    // duplicates; nothing sent after a refusal was pushed before it.
    const order = (xs) => xs.filter((a) => a.n !== undefined || a.a === 'text').map((a) => a.n ?? a.v);
    const sentOrder = order(got), pushedOrder = order(pushed);
    let j = 0;
    for (const k of sentOrder) {
      while (j < pushedOrder.length && pushedOrder[j] !== k) j++;
      assert.ok(j < pushedOrder.length, `seed ${seed}: ${k} out of order`);
      j++;
    }
    // No two consecutive moves of one button kind in one batch.
    for (const x of server.sent) {
      for (let i = 1; i < x.acts.length; i++) {
        const p = x.acts[i - 1], a = x.acts[i];
        assert.ok(!(p.a === 'move' && a.a === 'move' && (p.button || '') === (a.button || '')), `seed ${seed}: uncoalesced moves`);
      }
    }
    assert.ok(refusals.every(([s, t]) => s === 409 && t === 'The page changed. Try again.'));
  }
});

test('batcher, property: every request is one the wire takes — body under MAX_BODY, coordinates in range, paced', async () => {
  const enc = new TextEncoder();
  for (let seed = 1; seed <= 100; seed++) {
    const rand = rng(seed);
    const c = clock();
    const sent = [];
    const b = batcher({ send: (x) => { sent.push({ x, at: c.now() }); return Promise.resolve({ status: 204 }); }, seg: () => 3, later: c.later, now: c.now, refused: () => assert.fail('refused') });
    for (let i = 0; i < 80; i++) {
      const r = rand();
      const coord = () => (rand() - 0.3) * 4000; // some off the frame's left or top
      if (r < 0.3) b.push({ a: 'text', v: (rand() < 0.5 ? 'é"\\\n' : 'あ').repeat(Math.floor(rand() * 3000)) || 'x' });
      else if (r < 0.6) b.push({ a: 'click', id: 2, x: coord(), y: coord() });
      else b.push({ a: 'move', button: 'left', x: coord(), y: coord() });
      if (rand() < 0.5) c.advance(Math.floor(rand() * 30));
      await tick();
    }
    for (let guard = 0; guard < 2000 && b.pending(); guard++) { c.fire(); await tick(); }
    assert.equal(b.pending(), 0, `seed ${seed}: stuck`);
    for (let i = 0; i < sent.length; i++) {
      const body = JSON.stringify(sent[i].x);
      assert.ok(enc.encode(body).length <= MAX_BODY, `seed ${seed}: a ${enc.encode(body).length}-byte body`);
      for (const a of sent[i].x.acts) {
        for (const k of ['x', 'y']) if (k in a) assert.ok(a[k] >= 0 && a[k] <= MAX_COORD, `seed ${seed}: ${k}=${a[k]}`);
        if (a.a === 'text') assert.ok([...a.v].length <= MAX_TEXT);
      }
      if (i > 0) assert.ok(sent[i].at - sent[i - 1].at >= PACE_MS, `seed ${seed}: requests ${sent[i].at - sent[i - 1].at} ms apart`);
    }
  }
});

test('splitText and wireAct', () => {
  assert.deepEqual(splitText('abc'), ['abc']);
  const big = 'あ'.repeat(1500);
  const parts = splitText(big);
  assert.equal(parts.join(''), big);
  assert.ok(parts.every((p) => new TextEncoder().encode(JSON.stringify(p)).length <= 2048 + 2));
  assert.ok(splitText('😀'.repeat(MAX_TEXT + 1)).every((p) => !p.includes('\uFFFD')));
  assert.deepEqual(wireAct({ a: 'move', x: -40, y: MAX_COORD * 2 }), { a: 'move', x: 0, y: MAX_COORD });
  assert.deepEqual(wireAct({ a: 'text', v: 'x' }), { a: 'text', v: 'x' });
});

test('batcher: moves wait FLUSH_MS, a click goes at once; wheels on one element add up; long text splits', async () => {
  const sent = [];
  const c = clock();
  const b = batcher({ send: (x) => { sent.push(x); return Promise.resolve({ status: 204 }); }, seg: () => 1, later: c.later, now: c.now, refused: () => {} });
  c.advance(PACE_MS);
  b.push({ a: 'move', x: 1, y: 1 });
  b.push({ a: 'move', x: 2, y: 2 });
  assert.equal(sent.length, 0);
  assert.equal(c.fire(), 1);
  assert.deepEqual(sent[0].acts, [{ a: 'move', x: 2, y: 2 }]);
  await tick();
  c.advance(PACE_MS);
  b.push({ a: 'wheel', id: 4, x: 1, y: 1, dx: 0, dy: 5 });
  b.push({ a: 'wheel', id: 4, x: 1, y: 1, dx: 1, dy: 5 });
  b.push({ a: 'click', id: 2, x: 0, y: 0 });
  assert.deepEqual(sent[1].acts, [{ a: 'wheel', id: 4, x: 1, y: 1, dx: 1, dy: 10 }, { a: 'click', id: 2, x: 0, y: 0 }]);
  await tick();
  c.advance(PACE_MS);
  b.push({ a: 'text', v: 'é'.repeat(MAX_TEXT + 3) });
  for (let i = 0; i < 5; i++) { c.fire(); await tick(); }
  const texts = sent.slice(2).flatMap((x) => x.acts).map((a) => a.v).join('');
  assert.equal(texts, 'é'.repeat(MAX_TEXT + 3));
  assert.equal(FLUSH_MS, 50);
});

test('batcher: a refusal drops the queue and reports the sentence; a failed fetch too; stop sends nothing more', async () => {
  const c = clock();
  const refused = [];
  let answer = { status: 401, text: 'Join the cast again.' };
  const sent = [];
  const b = batcher({ send: (x) => { sent.push(x); return answer instanceof Error ? Promise.reject(answer) : Promise.resolve(answer); }, seg: () => 1, later: c.later, now: c.now, refused: (s, t) => refused.push([s, t]) });
  b.push({ a: 'click', id: 1, x: 0, y: 0 });
  b.push({ a: 'text', v: 'queued behind' });
  await tick(); await tick();
  assert.deepEqual(refused, [[401, 'Join the cast again.']]);
  assert.equal(b.pending(), 0);
  answer = new Error('offline');
  c.advance(PACE_MS);
  b.push({ a: 'text', v: 'x' });
  await tick(); await tick();
  assert.deepEqual(refused[1], [0, "Couldn't reach the cast."]);
  b.stop();
  b.push({ a: 'click', id: 1, x: 0, y: 0 });
  await tick();
  assert.equal(sent.length, 2);
});

// A viewer frame at scale 0.5 (window px * 2 = page px), with element 9
// everywhere but a hole at the top left.
const deps = (out) => ({
  toPage: (x, y) => [x * 2, y * 2],
  nodeAt: (x, y) => (x < 10 && y < 10 ? 0 : 9),
  emit: (a) => out.push(a),
});
const ev = (o) => ({ pointerId: 1, pointerType: 'mouse', button: 0, buttons: 0, detail: 1, timeStamp: 0, clientX: 0, clientY: 0, ...o });

test('gestures: a mouse click, a double click, a drag, a press on nothing', () => {
  const out = [];
  const g = gestures(deps(out));
  g.down(ev({ clientX: 50, clientY: 60 }));
  g.move(ev({ clientX: 51, clientY: 61 })); // under DRAG_PX: still a click
  g.up(ev({ clientX: 51, clientY: 61 }));
  assert.deepEqual(out, [{ a: 'click', id: 9, button: 'left', count: 1, x: 100, y: 120 }]);
  out.length = 0;
  g.down(ev({ clientX: 50, clientY: 60, detail: 2 }));
  g.up(ev({ clientX: 50, clientY: 60, detail: 2 }));
  assert.equal(out[0].count, 2);
  out.length = 0;
  g.down(ev({ clientX: 50, clientY: 60 }));
  g.move(ev({ clientX: 50 + DRAG_PX, clientY: 60 }));
  g.move(ev({ clientX: 70, clientY: 60 }));
  g.up(ev({ clientX: 80, clientY: 60 }));
  assert.deepEqual(out.map((a) => a.a), ['down', 'move', 'move', 'up']);
  assert.deepEqual(out[0], { a: 'down', id: 9, button: 'left', x: 100, y: 120 });
  assert.deepEqual(out.at(-1), { a: 'up', button: 'left', x: 160, y: 120 });
  out.length = 0;
  g.down(ev({ clientX: 1, clientY: 1 })); // nothing there
  g.move(ev({ clientX: 30, clientY: 30, buttons: 1, timeStamp: 500 })); // held: no hover
  g.up(ev({ clientX: 1, clientY: 1 }));
  g.down(ev({ clientX: 50, clientY: 60, button: 2 })); // right button: nothing
  g.up(ev({ clientX: 50, clientY: 60, button: 2 }));
  assert.deepEqual(out, []);
});

test('gestures: hover moves only with no button, at most every HOVER_MS', () => {
  const out = [];
  const g = gestures(deps(out));
  for (let t = 0; t < 200; t += 10) g.move(ev({ clientX: 30 + t, clientY: 30, timeStamp: t }));
  assert.equal(out.length, Math.ceil(200 / HOVER_MS));
  assert.ok(out.every((a) => a.a === 'move' && !a.button));
  out.length = 0;
  g.move(ev({ pointerType: 'touch', clientX: 30, clientY: 30, timeStamp: 1000 }));
  assert.deepEqual(out, [], 'a finger that is not down does not hover');
});

test('gestures, touch traces: a tap, a one-finger drag, two fingers scroll, a cancel and a late second finger', () => {
  const out = [];
  const g = gestures(deps(out));
  const t = (id, x, y) => ev({ pointerType: 'touch', pointerId: id, clientX: x, clientY: y });
  g.down(t(1, 50, 60)); g.up(t(1, 50, 60));
  assert.deepEqual(out, [{ a: 'click', id: 9, button: 'left', count: 1, x: 100, y: 120 }]);
  out.length = 0;
  g.down(t(1, 50, 60)); g.move(t(1, 50, 80)); g.up(t(1, 50, 90));
  assert.deepEqual(out.map((a) => a.a), ['down', 'move', 'up']);
  out.length = 0;
  // Two fingers move up 10 px: the page scrolls down 20 px (scale 0.5).
  g.down(t(1, 50, 60)); g.down(t(2, 70, 60));
  g.move(t(1, 50, 50)); g.move(t(2, 70, 50));
  g.up(t(1, 50, 50)); g.move(t(2, 70, 40)); g.up(t(2, 70, 40));
  assert.deepEqual(out.map((a) => a.a), ['wheel', 'wheel']);
  assert.equal(out.reduce((s, a) => s + a.dy, 0), 20);
  assert.ok(out.every((a) => a.id === 9 && a.dx === 0));
  out.length = 0;
  // A drag the second finger interrupts ends with an up, then scrolls.
  g.down(t(1, 50, 60)); g.move(t(1, 50, 80));
  g.down(t(2, 70, 80));
  assert.deepEqual(out.map((a) => a.a), ['down', 'move', 'up']);
  g.cancel(t(1, 50, 80)); g.cancel(t(2, 70, 80));
  out.length = 0;
  // Cancelled mid-drag: an up, nothing after.
  g.down(t(1, 50, 60)); g.move(t(1, 50, 90)); g.cancel(t(1, 50, 90)); g.up(t(1, 50, 90));
  assert.deepEqual(out.map((a) => a.a), ['down', 'move', 'up']);
  out.length = 0;
  // Malformed: an up and a move for a finger never down.
  g.up(t(5, 1, 1)); g.move(t(6, 50, 50));
  assert.deepEqual(out, []);
});

test('gestures: a drag captured past the frame\'s edge ends with an up the wire takes', async () => {
  const out = [];
  const g = gestures(deps(out));
  g.down(ev({ clientX: 50, clientY: 50 }));
  g.move(ev({ clientX: -50, clientY: 50, buttons: 1 }));
  g.up(ev({ clientX: -60, clientY: 50 }));
  assert.deepEqual(out.map((a) => a.a), ['down', 'move', 'up']);
  const sent = [];
  const c = clock();
  const b = batcher({ send: (x) => { sent.push(x); return Promise.resolve({ status: 204 }); }, seg: () => 1, later: c.later, now: c.now, refused: () => assert.fail() });
  for (const a of out) b.push(a);
  for (let i = 0; i < 10; i++) { c.fire(); await tick(); }
  assert.deepEqual(sent.flatMap((x) => x.acts).filter((a) => a.a === 'up'), [{ a: 'up', button: 'left', x: 0, y: 100 }]);
});

test('keyTracker: every key sent down is sent up, also after its modifier is released', () => {
  const k = (o) => ({ key: 'a', code: 'KeyA', ctrlKey: false, metaKey: false, shiftKey: false, altKey: false, isComposing: false, ...o });
  const tr = keyTracker();
  assert.deepEqual(tr.down(k({ metaKey: true })), { a: 'key', type: 'down', key: 'a', code: 'KeyA', mods: ['meta'] });
  assert.deepEqual(tr.up(k({})), { a: 'key', type: 'up', key: 'a', code: 'KeyA' }, 'Meta released first');
  assert.equal(tr.up(k({})), null, 'one up per down');
  assert.equal(tr.down(k({})), null, 'typing');
  assert.equal(tr.up(k({})), null);
  tr.down(k({ key: 'Enter', code: 'Enter' }));
  tr.clear();
  assert.equal(tr.up(k({ key: 'Enter', code: 'Enter' })), null);
});

test('gestures: a wheel over the frame is in page px', () => {
  const out = [];
  gestures(deps(out)).wheel({ clientX: 50, clientY: 60, deltaX: 0, deltaY: 30 });
  assert.deepEqual(out, [{ a: 'wheel', id: 9, x: 100, y: 120, dx: 0, dy: 60 }]);
});

test('keyAct: named keys and chords are sent; typing and compositions are not', () => {
  const k = (o) => ({ key: 'a', code: 'KeyA', ctrlKey: false, metaKey: false, shiftKey: false, altKey: false, isComposing: false, ...o });
  assert.deepEqual(keyAct(k({ key: 'Enter', code: 'Enter' }), 'down'), { a: 'key', type: 'down', key: 'Enter', code: 'Enter' });
  assert.deepEqual(keyAct(k({ key: 'ArrowLeft', code: 'ArrowLeft', shiftKey: true }), 'up'), { a: 'key', type: 'up', key: 'ArrowLeft', code: 'ArrowLeft', mods: ['shift'] });
  assert.deepEqual(keyAct(k({ key: 'a', metaKey: true }), 'down'), { a: 'key', type: 'down', key: 'a', code: 'KeyA', mods: ['meta'] });
  assert.deepEqual(keyAct(k({ key: 'v', code: 'KeyV', ctrlKey: true }), 'down').mods, ['ctrl'], 'paste is sent; the cast refuses it');
  assert.equal(keyAct(k({ key: ' ', code: 'Space' }), 'down').key, ' ');
  assert.equal(keyAct(k({}), 'down'), null, 'a letter is typed as text');
  assert.equal(keyAct(k({ key: 'A', shiftKey: true }), 'down'), null);
  assert.equal(keyAct(k({ key: 'Enter', isComposing: true }), 'down'), null);
  assert.equal(keyAct(k({ key: 'Process' }), 'down'), null);
  assert.equal(keyAct(k({ key: 'Shift' }), 'down'), null);
});

test('driveView: from the latest ctl', () => {
  const ana = { id: 'v1', name: 'Ana' }, bo = { id: 'v2', name: 'Bo' };
  assert.equal(driveView(null, 'v1').state, 'none');
  assert.equal(driveView({ k: 'ctl', pending: [ana] }, 'v1').state, 'asking');
  assert.deepEqual(driveView({ k: 'ctl', driver: ana }, 'v1'), { state: 'driving', button: 'Give back control', note: "You're driving." });
  assert.deepEqual(driveView({ k: 'ctl', driver: bo, pending: [ana] }, 'v1').state, 'asking');
  assert.deepEqual(driveView({ k: 'ctl', driver: bo }, 'v1'), { state: 'other', button: 'Ask to drive', note: 'Bo is driving.' });
  assert.equal(driveView({ k: 'ctl', op: 'revoke' }, 'v1').state, 'none');
});

test('joinSeat and seatPoster: the seat, or none; the key in its header', async () => {
  const seat = { viewer: 'v3', key: 'k'.repeat(64), name: 'Ana' };
  let req;
  assert.deepEqual(await joinSeat(async (url, init) => { req = [url, init]; return { ok: true, json: async () => seat }; }, 'Ana'), seat);
  assert.equal(req[0], 'drive/hello');
  assert.deepEqual(JSON.parse(req[1].body), { name: 'Ana' });
  assert.equal(await joinSeat(async () => ({ ok: false, status: 404 }), 'Ana'), null);
  assert.equal(await joinSeat(async () => { throw new Error('offline'); }, ''), null);
  assert.equal(await joinSeat(async () => ({ ok: true, json: async () => ({ nope: 1 }) }), ''), null);
  const post = seatPoster(async (url, init) => { req = [url, init]; return { status: 403, text: async () => 'Ask the operator for control first.\n' }; }, seat.key);
  assert.deepEqual(await post('input', { seg: 1, acts: [] }), { status: 403, text: 'Ask the operator for control first.' });
  assert.equal(req[0], 'drive/input');
  assert.equal(req[1].headers[SEAT_HEADER], seat.key);
  assert.equal(req[1].credentials, 'omit');
});
