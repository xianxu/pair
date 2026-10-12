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

export const SEAT_HEADER = 'X-Couch-Seat';
// FLUSH_MS: moves and wheels wait this long to coalesce; anything else is
// sent at once (behind the request in flight).
export const FLUSH_MS = 50;
// HOVER_MS: at most one hover move this often (20/s).
export const HOVER_MS = 50;
// DRAG_PX: a press that moves this far (window px) is a drag, not a click.
export const DRAG_PX = 4;
export const MAX_ACTS = 64;
export const MAX_TEXT = 1000;

// joinSeat asks for a seat: {viewer, key, name}, or null when the cast
// can't be driven (404) or doesn't answer.
export async function joinSeat(fetchFn, name) {
  try {
    const r = await fetchFn('drive/hello', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: name || '' }),
      credentials: 'omit', referrerPolicy: 'no-referrer', cache: 'no-store',
    });
    if (!r.ok) return null;
    const s = await r.json();
    return s && typeof s.viewer === 'string' && typeof s.key === 'string' ? s : null;
  } catch {
    return null;
  }
}

// seatPoster posts to a drive route with the seat's key: {status, text}.
export function seatPoster(fetchFn, key) {
  return (route, body) => fetchFn(`drive/${route}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', [SEAT_HEADER]: key },
    body: JSON.stringify(body),
    credentials: 'omit', referrerPolicy: 'no-referrer', cache: 'no-store',
  }).then(async (r) => ({ status: r.status, text: (await r.text()).trim() }));
}

// batcher sends acts in order, one request in flight. Consecutive moves
// (same button) coalesce into the latest, and consecutive wheels on one
// element add up; they wait FLUSH_MS for company, anything else goes at
// once. A batch carries at most MAX_ACTS acts and the segment seg() says
// at the moment it is sent. A refusal (anything but 204) drops what is
// queued, since the acts after a refused one would land on a page the
// driver didn't see, and is reported with the server's sentence.
//   deps: send(batch) → Promise<{status, text}>, seg(), later(fn, ms),
//   refused(status, text)
export function batcher(deps) {
  let queue = [];
  let inFlight = false, timer = null, stopped = false;
  const go = () => {
    timer = null;
    if (inFlight || stopped || !queue.length) return;
    const acts = queue.splice(0, MAX_ACTS);
    inFlight = true;
    deps.send({ seg: deps.seg(), acts }).then((r) => r, () => ({ status: 0, text: "Couldn't reach the cast." })).then((r) => {
      inFlight = false;
      if (r.status !== 204) {
        queue = [];
        deps.refused(r.status, r.text);
        return;
      }
      if (queue.length) go();
    });
  };
  const soon = () => { if (timer === null && !inFlight) timer = deps.later(go, FLUSH_MS); };
  return {
    push(a) {
      if (stopped) return;
      const last = queue.at(-1);
      if (a.a === 'move' && last && last.a === 'move' && (last.button || '') === (a.button || '')) {
        queue[queue.length - 1] = a;
      } else if (a.a === 'wheel' && last && last.a === 'wheel' && last.id === a.id) {
        queue[queue.length - 1] = { ...a, dx: last.dx + a.dx, dy: last.dy + a.dy };
      } else if (a.a === 'text' && [...a.v].length > MAX_TEXT) {
        const runes = [...a.v];
        for (let i = 0; i < runes.length; i += MAX_TEXT) queue.push({ a: 'text', v: runes.slice(i, i + MAX_TEXT).join('') });
      } else {
        queue.push(a);
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
// DRAG_PX is a click on release (a double click when the event says so); one
// that does is a drag: down where it began, moves, up. Touch is the same
// with one finger; two fingers scroll (wheel acts by the move of their
// middle). A mouse moving with no button down hovers (moves, at most every
// HOVER_MS). Events are plain objects with pointerId, pointerType, clientX,
// clientY, button, detail, timeStamp.
//   deps: toPage(cx, cy) → [x, y], nodeAt(x, y) → id, emit(act)
export function gestures(deps) {
  const touches = new Map(); // pointerId -> {x, y} (client)
  let press = null; // {id, start: [cx, cy], page: [x, y], dragging}
  let pinch = null; // {mid: [cx, cy]}
  let lastHover = -Infinity;
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
          deps.emit({ a: 'down', id: press.id, button: 'left', ...pt(press.page) });
        }
        press.last = p;
        deps.emit({ a: 'move', button: 'left', ...pt(p) });
        return;
      }
      if (e.pointerType === 'mouse' && e.timeStamp - lastHover >= HOVER_MS) {
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
      deps.emit({ a: 'click', id: press.id, button: 'left', count: e.detail >= 2 ? 2 : 1, ...pt(press.page) });
      press = null;
    },
    cancel(e) {
      touches.delete(e.pointerId);
      if (touches.size === 0) pinch = null;
      endPress();
    },
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

// driveView is this viewer's place in control, from the latest ctl record
// (null: none yet) and its own viewer id.
//   state: 'none' (may ask), 'asking', 'driving', or 'other' (someone else
//   drives; may ask)
export function driveView(ctl, me) {
  const driver = ctl && ctl.driver ? ctl.driver : null;
  const pending = (ctl && ctl.pending) || [];
  if (driver && driver.id === me) return { state: 'driving', button: 'Give back control', note: "You're driving." };
  if (pending.some((p) => p.id === me)) return { state: 'asking', button: 'Cancel request', note: 'Waiting for the operator…' };
  if (driver) return { state: 'other', button: 'Ask to drive', note: `${driver.name} is driving.` };
  return { state: 'none', button: 'Ask to drive', note: '' };
}

// mountDrive adds remote driving to the viewer. v is the viewer's surface
// (couchViewer: nodeAt, toPage, onRecord, model, annotate), seat the seat
// joinSeat gave (null: nothing mounts).
export function mountDrive(doc, win, v, seat) {
  if (!seat) return null;
  const post = seatPoster((url, init) => win.fetch(url, init), seat.key);
  const el = (tag, cls, parent) => { const e = doc.createElement(tag); if (cls) e.className = cls; parent?.append(e); return e; };
  const bar = doc.querySelector('.bar');
  const button = el('button', 'drive-button', bar);
  button.type = 'button';
  const note = el('span', 'drive-note', bar);
  note.setAttribute('role', 'status');
  const toast = el('div', 'annotate-msg', doc.body);
  toast.setAttribute('role', 'status');
  toast.hidden = true;
  let toastTimer = null;
  const say = (text) => {
    toast.textContent = text;
    toast.hidden = false;
    if (toastTimer !== null) win.clearTimeout(toastTimer);
    toastTimer = win.setTimeout(() => { toast.hidden = true; }, 3000);
  };
  const box = doc.getElementById('box');
  const layer = el('div', 'drive', box);
  // The keyboard's target: focused while driving, so keys, typing, IME and
  // an iPad's on-screen keyboard all come here and never reach the shell.
  const keys = el('textarea', 'drive-keys', doc.body);
  keys.setAttribute('aria-label', 'Typing into the cast page');
  keys.autocapitalize = 'off';
  keys.setAttribute('autocomplete', 'off');
  keys.spellcheck = false;

  let ctl = null, view = driveView(null, seat.viewer), batch = null, composing = false;
  const send = (a) => { if (batch) batch.push(a); };
  const g = gestures({ toPage: (x, y) => v.toPage(x, y), nodeAt: (x, y) => v.nodeAt(x, y), emit: send });

  const refused = (status, text) => {
    say(text || "That didn't work.");
    if (status === 401) offerRejoin();
  };
  const offerRejoin = () => {
    button.textContent = 'Rejoin';
    button.onclick = () => win.location.reload();
  };
  const render = () => {
    view = driveView(ctl, seat.viewer);
    const driving = view.state === 'driving' && !!(v.model && v.model.live);
    button.textContent = view.button;
    button.onclick = () => control(view.state === 'driving' || view.state === 'asking' ? 'release' : 'request');
    button.classList.toggle('on', view.state === 'driving');
    note.textContent = view.state === 'driving' && !driving ? 'The cast is paused.' : view.note;
    layer.classList.toggle('on', driving);
    doc.body.classList.toggle('driving', view.state === 'driving');
    if (view.state === 'driving' && !batch) {
      batch = batcher({ send: (b) => post('input', b), seg: () => v.model.seg, later: (fn, ms) => win.setTimeout(fn, ms), refused });
      v.annotate?.setMode('watch'); // pointing and notes wait while driving
    } else if (view.state !== 'driving' && batch) {
      batch.stop();
      batch = null;
      keys.blur();
    }
  };
  const control = (op) => post('control', { op }).then((r) => { if (r.status !== 204) refused(r.status, r.text); }, () => say("Couldn't reach the cast."));

  v.onRecord((r) => {
    if (r.k === 'ctl') ctl = r;
    if (r.k === 'ctl' || r.k === 'seg' || r.k === 'stop') render();
  });

  const listen = (type, fn) => layer.addEventListener(type, (e) => {
    if (!layer.classList.contains('on')) return;
    e.preventDefault();
    fn(e);
  }, { passive: false });
  listen('pointerdown', (e) => {
    layer.setPointerCapture?.(e.pointerId);
    keys.focus({ preventScroll: true }); // in the gesture: an iPad raises its keyboard
    g.down(e);
  });
  listen('pointermove', (e) => g.move(e));
  listen('pointerup', (e) => g.up(e));
  listen('pointercancel', (e) => g.cancel(e));
  listen('wheel', (e) => g.wheel(e));
  listen('contextmenu', () => {});

  keys.addEventListener('keydown', (e) => {
    if (!batch) return;
    const a = keyAct(e, 'down');
    if (!a) return;
    e.preventDefault(); // sent by name: the textarea mustn't type it too
    send(a);
  });
  keys.addEventListener('keyup', (e) => {
    const a = batch && keyAct(e, 'up');
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
  return { render, control, view: () => view, seat };
}
