// Remote driving in the web cast viewer (#467). A viewer takes a seat when
// the viewer opens (drive/hello; a cast without remote driving answers 404
// and nothing here mounts), asks the operator for control, and, once
// granted, uses the operator's tab through the scaled frame: pointer, touch
// and keyboard become acts aimed at elements (nodeAt) at page coordinates
// (toPage), sent in order to drive/input. The operator can take control
// back at any moment; who drives comes from the ctl records in the stream,
// never from this page's own guess. The cast tab decides what is allowed
// (drive.Confine); this page only decides what to send.
//
// The pure parts (batcher, gestures, keyAct, driveView) are exported for the
// node tests; mountDrive is the shell, called by viewer.js.

import { poster, sayer } from './collab.js';

export const SEAT_HEADER = 'X-Couch-Seat';
// FLUSH_MS: moves and wheels wait this long to coalesce; anything else is
// sent at once (behind the request in flight).
export const FLUSH_MS = 50;
// HOVER_MS: at most one hover move this often (20/s).
export const HOVER_MS = 50;
// DRAG_PX: a press that moves this far (window px) is a drag, not a click.
export const DRAG_PX = 4;
// DOUBLE_MS: a click this soon after the last, within DRAG_PX of it, is the
// second of a double click. Counted here because Chrome's pointer events
// carry detail 0 on every click (pair#505, measured: pointerup:0 for both
// presses of a double click; only mouseup and click count), so reading
// detail never sent a double click.
export const DOUBLE_MS = 500;
export const MAX_ACTS = 64;
export const MAX_TEXT = 1000;
// The wire's limits (drive.ParseBatch, the input route's budget): a body
// of at most MAX_BODY bytes, coordinates in [0, MAX_COORD], and requests
// spaced PACE_MS apart (25/s, under the seat's 30/s).
export const MAX_BODY = 8 << 10;
export const MAX_COORD = 1 << 20;
export const PACE_MS = 40;
// TEXT_BYTES bounds one text act's encoded size, so a batch always fits.
const TEXT_BYTES = 2048;
const utf8 = new TextEncoder();
const bytes = (v) => utf8.encode(JSON.stringify(v)).length;
// runeBytes is one character's size inside a JSON string.
const runeBytes = (r) => {
  const c = r.codePointAt(0);
  if (r === '"' || r === '\\') return 2;
  if (c < 0x20) return 6;
  return c < 0x80 ? 1 : c < 0x800 ? 2 : c < 0x10000 ? 3 : 4;
};

// joinSeat asks for a seat: {viewer, key, name}, or null when the cast
// can't be driven (404) or doesn't answer.
export async function joinSeat(fetchFn, name) {
  try {
    const r = await fetchFn('drive/hello', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: name || '' }),
      credentials: 'same-origin', referrerPolicy: 'no-referrer', cache: 'no-store',
    });
    if (!r.ok) return null;
    const s = await r.json();
    return s && typeof s.viewer === 'string' && typeof s.key === 'string' ? s : null;
  } catch {
    return null;
  }
}

// seatPoster posts to a drive route with the seat's key: {status, text}.
export function seatPoster(fetchFn, key, generation) {
  const post = poster(fetchFn, DRIVE_ROUTES, { [SEAT_HEADER]: key, 'X-Couch-Drive-Generation': String(generation) });
  return (route, body) => post(`drive/${route}`, body).then((r) => ({ status: r.status, text: r.text.trim() }));
}
const DRIVE_ROUTES = ['drive/control', 'drive/input'];

// wireAct is a as the wire takes it: coordinates within [0, MAX_COORD] (a
// drag captured past the frame's left or top edge goes negative).
export function wireAct(a) {
  const out = { ...a };
  for (const k of ['x', 'y']) if (k in out) out[k] = Math.min(MAX_COORD, Math.max(0, out[k]));
  return out;
}

// splitText cuts a text act's value into pieces of at most MAX_TEXT runes
// and TEXT_BYTES encoded bytes, never inside a character.
export function splitText(v) {
  const out = [];
  let cur = '', n = 0, size = 0;
  for (const r of v) {
    const c = runeBytes(r);
    if (cur && (n + 1 > MAX_TEXT || size + c > TEXT_BYTES)) { out.push(cur); cur = ''; n = 0; size = 0; }
    cur += r; n++; size += c;
  }
  if (cur) out.push(cur);
  return out;
}

// batcher is the one gate to the wire: it sends acts in order, one request
// in flight, only as the server takes them (wireAct's coordinates, text in
// splitText's pieces, a body under MAX_BODY, requests PACE_MS apart).
// Consecutive moves (same button) coalesce into the latest, and consecutive
// wheels on one element add up; they wait FLUSH_MS for company, anything
// else goes as soon as the pace allows. A batch carries at most MAX_ACTS
// acts and the segment seg() says when it is sent. A refusal (anything but
// 204) drops what is queued, since the acts after a refused one would land
// on a page the driver didn't see, and is reported with the server's
// sentence (the cast tab lets go of whatever the driver held). Each act
// carries the segment seg() said when it was pushed, the page the driver
// was looking at: a batch holds acts of one segment, so acts made on a
// page that has since gone are refused, never played into the new one.
//   deps: send(batch) → Promise<{status, text}>, seg(), later(fn, ms),
//   now() → ms, refused(status, text)
export function batcher(deps) {
  let queue = [];
  let inFlight = false, timer = null, stopped = false, last = -Infinity;
  const take = () => {
    const batch = { seg: queue[0].seg, acts: [] };
    let size = bytes(batch);
    while (queue.length && batch.acts.length < MAX_ACTS && queue[0].seg === batch.seg) {
      const add = bytes(queue[0].a) + (batch.acts.length ? 1 : 0); // and its comma
      if (batch.acts.length && size + add > MAX_BODY) break;
      batch.acts.push(queue.shift().a);
      size += add;
    }
    return batch;
  };
  const fire = () => { timer = null; go(); };
  const go = () => {
    if (inFlight || stopped || !queue.length) return;
    const wait = last + PACE_MS - deps.now();
    if (wait > 0) { if (timer === null) timer = deps.later(fire, wait); return; }
    last = deps.now();
    const batch = take();
    inFlight = true;
    deps.send(batch).then((r) => r, () => ({ status: 0, text: "Couldn't reach the cast." })).then((r) => {
      inFlight = false;
      if (stopped) return;
      if (r.status !== 204) {
        queue = [];
        deps.refused(r.status, r.text);
        return;
      }
      if (queue.length) go();
    });
  };
  const soon = () => { if (timer === null && !inFlight) timer = deps.later(fire, FLUSH_MS); };
  return {
    push(act) {
      if (stopped) return;
      const a = wireAct(act), seg = deps.seg();
      const last = queue.at(-1), prev = last && last.seg === seg ? last.a : null;
      if (a.a === 'move' && prev && prev.a === 'move' && (prev.button || '') === (a.button || '')) {
        last.a = a;
      } else if (a.a === 'wheel' && prev && prev.a === 'wheel' && prev.id === a.id) {
        last.a = { ...a, dx: prev.dx + a.dx, dy: prev.dy + a.dy };
      } else if (a.a === 'text') {
        for (const v of splitText(a.v)) queue.push({ a: { a: 'text', v }, seg });
      } else {
        queue.push({ a, seg });
      }
      if (a.a === 'move' || a.a === 'wheel') soon();
      else go();
    },
    // stop drops everything and sends nothing more (control has gone).
    stop() { stopped = true; queue = []; },
    pending: () => queue.length + (inFlight ? 1 : 0),
  };
}

const round = (v) => Math.round(v * 10) / 10;

// gestures turns pointer events into acts. A press that doesn't move
// DRAG_PX is a click on release (the second of a double click when it comes
// within DOUBLE_MS and DRAG_PX of the last click); one that does is a drag:
// down where it began, moves, up. Touch is the same
// with one finger; two fingers scroll (wheel acts by the move of their
// middle). A mouse moving with no button down hovers (moves, at most every
// HOVER_MS). Events are plain objects with pointerId, pointerType, clientX,
// clientY, button, buttons, timeStamp. Coordinates may leave the
// frame (the pointer is captured); the batcher clamps them for the wire.
//   deps: toPage(cx, cy) → [x, y], nodeAt(x, y) → id, emit(act)
export function gestures(deps) {
  const touches = new Map(); // pointerId -> {x, y} (client)
  let press = null; // {id, start: [cx, cy], page: [x, y], dragging}
  let pinch = null; // {mid: [cx, cy]}
  let lastHover = -Infinity;
  let lastClick = null; // {t, x, y}: a single click that may become a double
  const at = (e) => deps.toPage(e.clientX, e.clientY);
  const pt = (p) => ({ x: round(p[0]), y: round(p[1]) });
  const mid = () => {
    const ps = [...touches.values()];
    return [ps.reduce((s, p) => s + p.x, 0) / ps.length, ps.reduce((s, p) => s + p.y, 0) / ps.length];
  };
  const endPress = () => {
    if (press && press.dragging) deps.emit({ a: 'up', button: 'left', ...pt(press.last) });
    press = null;
  };
  return {
    down(e) {
      if (e.pointerType === 'touch') touches.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (touches.size >= 2) {
        endPress(); // a second finger: whatever the first did ends, and they scroll
        pinch = { mid: mid() };
        return;
      }
      if (e.button !== 0) return;
      const p = at(e);
      const id = deps.nodeAt(p[0], p[1]);
      press = id ? { id, start: [e.clientX, e.clientY], page: p, last: p, dragging: false } : null;
    },
    move(e) {
      if (e.pointerType === 'touch' && touches.has(e.pointerId)) touches.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pinch) {
        if (touches.size < 2) return;
        const m = mid(), a = deps.toPage(pinch.mid[0], pinch.mid[1]), b = deps.toPage(m[0], m[1]);
        pinch.mid = m;
        const dx = a[0] - b[0], dy = a[1] - b[1]; // fingers up, page down
        if (!dx && !dy) return;
        const id = deps.nodeAt(b[0], b[1]);
        if (id) deps.emit({ a: 'wheel', id, ...pt(b), dx: round(dx), dy: round(dy) });
        return;
      }
      const p = at(e);
      if (press) {
        if (!press.dragging && Math.hypot(e.clientX - press.start[0], e.clientY - press.start[1]) < DRAG_PX) return;
        if (!press.dragging) {
          press.dragging = true;
          lastClick = null;
          deps.emit({ a: 'down', id: press.id, button: 'left', ...pt(press.page) });
        }
        press.last = p;
        deps.emit({ a: 'move', button: 'left', ...pt(p) });
        return;
      }
      if (e.pointerType === 'mouse' && !e.buttons && e.timeStamp - lastHover >= HOVER_MS) {
        lastHover = e.timeStamp;
        deps.emit({ a: 'move', ...pt(p) });
      }
    },
    up(e) {
      touches.delete(e.pointerId);
      if (pinch) {
        if (touches.size === 0) pinch = null;
        return;
      }
      if (!press) return;
      if (press.dragging) {
        press.last = at(e);
        endPress();
        return;
      }
      const second = lastClick !== null && e.timeStamp - lastClick.t <= DOUBLE_MS &&
        Math.hypot(e.clientX - lastClick.x, e.clientY - lastClick.y) < DRAG_PX;
      deps.emit({ a: 'click', id: press.id, button: 'left', count: second ? 2 : 1, ...pt(press.page) });
      lastClick = second ? null : { t: e.timeStamp, x: e.clientX, y: e.clientY };
      press = null;
    },
    cancel(e) {
      touches.delete(e.pointerId);
      if (touches.size === 0) pinch = null;
      endPress();
    },
    // reset forgets every press, finger and the last click: control ended,
    // and the cast tab lets go of what it held (a click after it never
    // pairs with one before into a double click).
    reset() { touches.clear(); press = null; pinch = null; lastClick = null; },
    // wheel: a mouse wheel or trackpad over the page, in window px.
    wheel(e) {
      const p = at(e), q = deps.toPage(e.clientX + e.deltaX, e.clientY + e.deltaY);
      const id = deps.nodeAt(p[0], p[1]);
      if (id) deps.emit({ a: 'wheel', id, ...pt(p), dx: round(q[0] - p[0]), dy: round(q[1] - p[1]) });
    },
  };
}

// The keys sent by name (the rest of the keyboard arrives as text).
const NAMED = new Set(['Enter', ' ', 'Tab', 'Escape', 'Backspace', 'Delete', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End', 'PageUp', 'PageDown']);

// keyAct is what a key event sends: a key act for a named key, or for a
// printable key held with Ctrl or Meta (select all, undo); null for typing,
// which arrives as text, and during an IME composition. type is 'down' or
// 'up'. What is allowed is the cast's to say.
export function keyAct(e, type) {
  if (e.isComposing || e.key === 'Process' || e.key === 'Unidentified' || e.key === 'Dead') return null;
  const chord = e.ctrlKey || e.metaKey || e.altKey;
  const printable = [...e.key].length === 1;
  if (!NAMED.has(e.key) && !(printable && chord)) return null;
  const mods = [];
  if (e.ctrlKey) mods.push('ctrl');
  if (e.metaKey) mods.push('meta');
  if (e.shiftKey) mods.push('shift');
  if (e.altKey) mods.push('alt');
  const a = { a: 'key', type, key: e.key };
  if (e.code) a.code = e.code;
  if (mods.length) a.mods = mods;
  return a;
}

// keyTracker pairs key acts: keyAct's down, and an up for every key sent
// down, with the modifiers held at release (Meta may already be up: Meta+A's
// A then goes up plain). A key never sent down sends no up.
export function keyTracker() {
  const downs = new Set();
  const id = (e) => e.code || e.key;
  return {
    down(e) {
      const a = keyAct(e, 'down');
      if (a) downs.add(id(e));
      return a;
    },
    up(e) {
      if (!downs.delete(id(e))) return null;
      return keyAct(e, 'up') || { a: 'key', type: 'up', key: e.key, ...(e.code ? { code: e.code } : {}) };
    },
    clear: () => downs.clear(),
  };
}

// driveView is everything the shell shows of control, from the latest ctl
// record (null: none yet), this viewer's id, whether the cast is live, and
// whether the seat is lost (a 401). The shell only paints it.
//   state: 'none' (may ask), 'asking', 'driving' (live: input captured),
//   'paused' (driving, the cast paused), 'other' (someone else drives; may
//   ask), 'lost' (rejoin)
//   action: what the button does: 'request', 'release' or 'rejoin'
export function driveView(ctl, me, { live = true, lost = false, allowed = true, availability = 'off' } = {}) {
  const driver = ctl && ctl.driver ? ctl.driver : null;
  const pending = (ctl && ctl.pending) || [];
  if (!allowed) return { state: 'off', button: 'Driving off', action: '', note: availability === 'broken' ? 'Cast broken. Wait for the operator to end the cast.' : availability === 'retry' ? 'Waiting for the operator to retry.' : 'Driving is off.', capture: false };
  if (lost) return { state: 'lost', button: 'Rejoin', action: 'rejoin', note: '', capture: false };
  if (driver && driver.id === me) {
    return live
      ? { state: 'driving', button: 'Give back control', action: 'release', note: "You're driving.", capture: true }
      : { state: 'paused', button: 'Give back control', action: 'release', note: 'The cast is paused.', capture: false };
  }
  if (pending.some((p) => p.id === me)) return { state: 'asking', button: 'Cancel request', action: 'release', note: 'Waiting for the operator…', capture: false };
  if (driver) return { state: 'other', button: 'Ask to drive', action: 'request', note: `${driver.name} is driving.`, capture: false };
  return { state: 'none', button: 'Ask to drive', action: 'request', note: '', capture: false };
}

// mountDrive adds remote driving to the viewer. v is the viewer's surface
// (couchViewer: nodeAt, toPage, onRecord, model, annotate), seat the seat
// joinSeat gave (null: nothing mounts).
export function mountDrive(doc, win, v, seat) {
  if (!seat) return null;
  const postAt = (g) => seatPoster((url, init) => win.fetch(url, init), seat.key, g);
  const el = (tag, cls, parent) => { const e = doc.createElement(tag); if (cls) e.className = cls; parent?.append(e); return e; };
  const bar = doc.querySelector('.bar');
  const button = el('button', 'drive-button', bar);
  button.type = 'button';
  const note = el('span', 'drive-note', bar);
  note.setAttribute('role', 'status');
  const say = sayer(doc, win);
  const box = doc.getElementById('box');
  const layer = el('div', 'drive', box);
  // The keyboard's target: focused while driving, so keys, typing, IME and
  // an iPad's on-screen keyboard all come here and never reach the shell.
  const keys = el('textarea', 'drive-keys', doc.body);
  keys.setAttribute('aria-label', 'Typing into the cast page');
  keys.autocapitalize = 'off';
  keys.setAttribute('autocomplete', 'off');
  keys.spellcheck = false;

  let allowed = false, availability = 'off', generation, consent = false, epoch = 0;
  let sketching = false;
  let ctl = null, view = driveView(null, seat.viewer), batch = null, composing = false, lost = false;
  const driving = (s) => s === 'driving' || s === 'paused';
  const captured = new Set();
  const releasePointers = () => { for (const id of captured) { if (layer.hasPointerCapture?.(id)) layer.releasePointerCapture(id); } captured.clear(); };
  const tracked = keyTracker();
  const send = (a) => { if (batch && !sketching) batch.push(a); };
  const g = gestures({ toPage: (x, y) => v.toPage(x, y), nodeAt: (x, y) => v.nodeAt(x, y), emit: send });

  const refused = (status, text) => {
    say(text || "That didn't work.");
    // The queue is gone and the cast tab let go: start the next gesture
    // fresh.
    g.reset();
    tracked.clear();
    if (status === 401) offerRejoin();
  };
  // A lost seat (401) can't be repaired in place: the stream holds the
  // seat, so rejoining reopens the page.
  const offerRejoin = () => { lost = true; render(); };
  const render = () => {
    view = driveView(!consent && ctl?.driver?.id === seat.viewer ? {...ctl, driver:null} : ctl, seat.viewer, { live: !!(v.model && v.model.live), lost, allowed, availability });
    button.textContent = view.button;
    button.disabled = !view.action;
    button.onclick = () => (view.action === 'rejoin' ? win.location.reload() : control(view.action));
    button.classList.toggle('on', driving(view.state));
    note.textContent = view.note;
    layer.classList.toggle('on', view.capture && !sketching);
    doc.body.classList.toggle('driving', driving(view.state));
    if (driving(view.state) && !sketching && !batch) {
      const post = postAt(generation);
      batch = batcher({ send: (b) => post('input', b), seg: () => v.model.seg, later: (fn, ms) => win.setTimeout(fn, ms), now: () => win.performance.now(), refused });
      v.annotate?.setMode('watch'); // pointing and notes wait while driving
    } else if ((!driving(view.state) || sketching) && batch) {
      // Control ended: forget every press and key (the cast tab lets go).
      batch.stop();
      batch = null;
      g.reset();
      releasePointers();
      tracked.clear();
      keys.blur();
      keys.value = '';
      composing = false;
    }
  };
  const control = (op) => {
    if (!allowed || !['request', 'release'].includes(op)) return Promise.resolve();
    const started = epoch;
    if (op === 'request') consent = true;
    return postAt(generation)('control', { op }).then((r) => {
      if (started !== epoch) return;
      if (r.status !== 204) { if (op === 'request') consent = false; refused(r.status, r.text); }
    }, () => { if (started === epoch) say("Couldn't reach the cast."); });
  };
  v.onCapabilities?.((caps) => {
    const c = caps?.drive;
    const changed = generation !== c?.generation;
    if (changed || c?.allowed !== true) {
      epoch++;
      if (changed && generation !== undefined) ctl = null;
      if (changed) consent = generation === undefined && c?.allowed === true;
      if (batch) { batch.stop(); batch = null; }
      g.reset(); releasePointers(); tracked.clear(); keys.blur(); keys.value = ''; composing = false;
    }
    generation = c?.generation;
    allowed = c?.allowed === true;
    availability = c?.state || 'off';
    render();
  });

  v.onRecord((r) => {
    if (r.k === 'ctl') ctl = r;
    if (r.k === 'sketch-snapshot') ctl = r.control;
    if (r.k === 'ctl' || r.k === 'sketch-snapshot' || r.k === 'seg' || r.k === 'stop') render();
  });

  const listen = (type, fn) => layer.addEventListener(type, (e) => {
    if (!layer.classList.contains('on')) return;
    e.preventDefault();
    fn(e);
  }, { passive: false });
  listen('pointerdown', (e) => {
    layer.setPointerCapture?.(e.pointerId);
    captured.add(e.pointerId);
    keys.focus({ preventScroll: true }); // in the gesture: an iPad raises its keyboard
    g.down(e);
  });
  listen('pointermove', (e) => g.move(e));
  listen('pointerup', (e) => { g.up(e); captured.delete(e.pointerId); });
  listen('pointercancel', (e) => { g.cancel(e); captured.delete(e.pointerId); });
  listen('wheel', (e) => g.wheel(e));
  listen('contextmenu', () => {});

  keys.addEventListener('keydown', (e) => {
    const a = batch && tracked.down(e);
    if (!a) return;
    e.preventDefault(); // sent by name: the textarea mustn't type it too
    send(a);
  });
  keys.addEventListener('keyup', (e) => {
    const a = batch && tracked.up(e);
    if (a) send(a);
  });
  keys.addEventListener('compositionstart', () => { composing = true; });
  keys.addEventListener('compositionupdate', (e) => { if (e.data) send({ a: 'ime', v: e.data }); });
  keys.addEventListener('compositionend', (e) => {
    composing = false;
    if (e.data) send({ a: 'text', v: e.data });
    keys.value = '';
  });
  keys.addEventListener('input', () => {
    if (composing) return;
    if (keys.value) send({ a: 'text', v: keys.value });
    keys.value = '';
  });

  render();
  return { render, control, view: () => view, seat, setSketching(value) { sketching = value === true; render(); } };
}
