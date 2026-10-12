// The collaboration pieces both viewers share: the terminal broadcast's
// (cmd/internal/broadcast/web) and the web cast's (#470, served by the
// webcast package from broadcast.CollabJS). Pointer colors and the swatch
// rail (#446, #449), and the sticky-note dialog (#447). Note text is only
// ever set as textContent.

// Pointer colors (#446). Couch sends the palette and this stream's assigned
// color in `caps`; the page keeps no colors of its own and posts only an
// index. A viewer's pick is remembered in this browser.
const COLOR_KEY = 'couch.pointerColor';

export const HEX_COLOR = /^#[0-9a-fA-F]{6}$/;

// pointerColor is the color a page posts: the viewer's pick when it is in
// the palette, else the one Couch assigned.
export function pointerColor(picked, assigned, size) {
  return Number.isInteger(picked) && picked >= 0 && picked < size ? picked : assigned;
}

// colorStore remembers the pick. Storage can be missing or throw (a private
// window, blocked site data); the page then works without remembering.
export function colorStore(storage) {
  const store = safeStorage(storage);
  return {
    get() {
      const v = store.get(COLOR_KEY);
      return /^[0-9]$/.test(v || '') ? Number(v) : null;
    },
    set(i) {
      store.set(COLOR_KEY, String(i)); // if not remembered, the pick still holds for this page
    },
  };
}

// safeStorage is the pages' one way to keep anything: storage can be
// missing or throw (a private window, blocked site data), and the page then
// works without remembering.
export function safeStorage(storage) {
  return {
    get(key) {
      try {
        return storage ? storage.getItem(key) : null;
      } catch {
        return null;
      }
    },
    set(key, value) {
      try {
        storage?.setItem(key, value);
      } catch {
        // Not remembered.
      }
    },
  };
}

// poster makes a page's one request function: same origin, a relative
// route from routes, no credentials, no referrer, plus any headers (a
// remote-driving seat's key, #467). It resolves to the status and the
// response text (a refusal sentence the page shows).
export function poster(fetchFn, routes, headers = {}) {
  return (route, body) => {
    if (!routes.includes(route)) {
      return Promise.reject(new Error('unknown route'));
    }
    return fetchFn(route, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      credentials: 'omit',
      referrerPolicy: 'no-referrer',
      cache: 'no-store',
    }).then(async (r) => ({ status: r.status, text: await r.text() }));
  };
}

// RAIL_INSET is the gap between the rail and the screen's right edge.
export const RAIL_INSET = 6;

// railAt docks the rail (#449) to the drawn screen: its right edge
// RAIL_INSET inside the screen's, its middle on the screen's (the CSS
// translates it up by half its height). rect is the screen's client rect;
// viewW the window's width.
export function railAt(rect, viewW) {
  return {
    top: rect.top + rect.height / 2,
    right: Math.max(0, viewW - rect.right) + RAIL_INSET,
  };
}

// renderSwatches draws one button per palette color into el, marking the
// current one. make(tag) creates an element.
export function renderSwatches(el, palette, current, pick, make) {
  el.replaceChildren(...palette.map((hex, i) => {
    const b = make('button');
    b.type = 'button';
    b.className = i === current ? 'swatch on' : 'swatch';
    b.style.background = hex;
    b.setAttribute('aria-label', `Pointer color ${i + 1}`);
    b.title = `Pointer color ${i + 1}`;
    b.setAttribute('aria-pressed', String(i === current));
    b.addEventListener('click', () => pick(i));
    return b;
  }));
}


export const MAX_NOTE_BYTES = 1024;

// Note boxes (#488): every note is a box on the screen, the area a
// submission crops. A drag draws it; a tap gets the default box, a tenth
// of the screen each way, centred on the tap. Both casts call gestureBox in
// their own units: the terminal in cells (cells: true; a drag covers the
// cells it touched, and a box is at least the pin's two cells), the web
// cast in top-page CSS px. view is the screen shown, {x, y, w, h}; min is
// how far (either axis) a drag must go to be one (DRAG_MIN). The box is
// [x, y, w, h], inside view.
export const DRAG_MIN = {
  cells: { mouse: 1, touch: 2 },
  px: { mouse: 8, touch: 16 },
};

export function dragMin(unit, pointerType) {
  return DRAG_MIN[unit][pointerType === 'touch' ? 'touch' : 'mouse'];
}

export function gestureBox(from, to, view, { min, cells = false }) {
  const tap = Math.abs(to[0] - from[0]) < min && Math.abs(to[1] - from[1]) < min;
  const minW = cells ? 2 : 1, minH = 1;
  const axis = (a, b, origin, size, least) => {
    if (tap) {
      const span = Math.min(size, Math.max(least, cells ? Math.round(size / 10) : size / 10));
      const start = cells ? a - Math.floor(span / 2) : a - span / 2;
      return [Math.min(Math.max(start, origin), origin + size - span), span];
    }
    let lo = Math.max(Math.min(a, b), origin);
    const hi = Math.min(Math.max(a, b) + (cells ? 1 : 0), origin + size);
    let span = Math.max(hi - lo, least);
    span = Math.min(span, size);
    lo = Math.min(lo, origin + size - span);
    return [lo, span];
  };
  const [x, w] = axis(from[0], to[0], view.x, view.w, minW);
  const [y, h] = axis(from[1], to[1], view.y, view.h, minH);
  const r = cells ? (v) => v : (v) => Math.round(v * 10) / 10;
  return [r(x), r(y), r(w), r(h)];
}

// boxCentre is a box's middle, where a note anchors.
export function boxCentre(b) {
  return [b[0] + b[2] / 2, b[1] + b[3] / 2];
}

// noteBytes counts UTF-8 bytes, as Couch's cap does.
export function noteBytes(text) {
  return new TextEncoder().encode(text).length;
}


// noteDialog drives the one sticky-note dialog, the terminal broadcast's
// (#447) and the web cast's (#470). els: root, text, input, count, msg,
// save, cancel, resolve (element-likes). deps: post(route, body) →
// Promise<{status, text}>, color() → the viewer's pointer color, and
// optionally resolveRequest(id) → [route, body] (default ['resolve', {id}])
// and beforeSave() → an object merged into the note body last, or null to
// stop the save (the web cast's name field).
//   draft(payload)             a new note; Save posts 'note' with payload,
//                              color and text (the terminal's payload is its
//                              cell and grid, the web cast's its anchor)
//   view(note, {resolvable})   an existing note, read-only
//   update(list)               a new notes list: a viewed note that is gone
//                              (resolved elsewhere, or pointing turned off)
//                              closes its dialog
//   hold(reason)               a non-empty reason disables Save and Resolve
//                              and says why (a paused web cast, pair#499);
//                              '' releases it. A draft keeps its text.
//   close()
export function noteDialog(els, deps) {
  let state = null; // {kind: 'draft', payload} | {kind: 'view', id}
  let held = ''; // why Save and Resolve are off, or 
  const resolveRequest = deps.resolveRequest || ((id) => ['resolve', { id }]);
  // Each open bumps the generation; a request's answer applies only to the
  // dialog that sent it, not to one opened since.
  let gen = 0;
  const show = (on) => els.root.classList.toggle('hidden', !on);
  const say = (text) => {
    els.msg.textContent = text || '';
  };
  const refreshCount = () => {
    const n = noteBytes(els.input.value);
    els.count.textContent = `${n} / ${MAX_NOTE_BYTES}`;
    els.save.disabled = held !== '' || n === 0 || n > MAX_NOTE_BYTES || els.input.value.trim() === '';
  };
  const close = () => {
    gen++;
    state = null;
    els.input.value = '';
    say('');
    show(false);
  };
  const answer = (req, onDone) => {
    const sent = gen;
    return req.then((r) => {
      if (sent !== gen) {
        return;
      }
      if (r.status === 204) {
        onDone();
      } else {
        say((r.text || '').trim() || "That didn't work. Try again.");
      }
    }).catch(() => sent === gen && say("Couldn't reach the broadcast. Try again."));
  };
  els.input.addEventListener('input', refreshCount);
  els.cancel.addEventListener('click', close);
  els.save.addEventListener('click', () => {
    if (state?.kind !== 'draft' || els.save.disabled) {
      return;
    }
    const extra = deps.beforeSave ? deps.beforeSave() : {};
    if (!extra) {
      return;
    }
    els.save.disabled = true;
    answer(deps.post('note', { ...state.payload, color: deps.color(), text: els.input.value, ...extra }), close)
      .finally(() => state?.kind === 'draft' && refreshCount());
  });
  els.resolve.addEventListener('click', () => {
    if (state?.kind === 'view' && held === '') {
      answer(deps.post(...resolveRequest(state.id)), close);
    }
  });
  els.root.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      close();
    }
  });
  return {
    draft(payload) {
      gen++;
      state = { kind: 'draft', payload };
      els.root.classList.add('draft');
      els.root.classList.remove('view');
      els.text.textContent = '';
      els.input.value = '';
      say(held);
      els.resolve.classList.add('hidden');
      els.save.classList.remove('hidden');
      els.cancel.textContent = 'Cancel';
      refreshCount();
      show(true);
      els.input.focus?.();
    },
    view(note, { resolvable }) {
      gen++;
      state = { kind: 'view', id: note.id };
      els.root.classList.add('view');
      els.root.classList.remove('draft');
      els.text.textContent = note.text;
      say(held);
      els.resolve.classList.toggle('hidden', !resolvable);
      els.resolve.disabled = held !== '';
      els.save.classList.add('hidden');
      els.cancel.textContent = 'Close';
      show(true);
      // Focus inside the dialog, so Escape reaches it as it does a draft.
      els.cancel.focus?.();
    },
    update(list) {
      if (state?.kind === 'view' && !list.notes.some((n) => n.id === state.id)) {
        close();
      }
    },
    hold(reason) {
      const was = held;
      held = reason || '';
      els.resolve.disabled = held !== '';
      if (state?.kind === 'draft') {
        refreshCount();
      }
      if (state && held !== was) {
        say(held);
      }
    },
    close,
    get open() {
      return state !== null;
    },
  };
}

// sayer is a page's one status toast: a sentence shown for a moment (a
// refused request no dialog is open for). Every module of the page shares
// the one element, so two never overlap.
export function sayer(doc, win) {
  let toast = doc.querySelector('.annotate-msg');
  if (!toast) {
    toast = doc.createElement('div');
    toast.className = 'annotate-msg';
    toast.setAttribute('role', 'status');
    toast.hidden = true;
    doc.body.append(toast);
  }
  return (text) => {
    toast.textContent = text;
    toast.hidden = false;
    if (toast.couchTimer) win.clearTimeout(toast.couchTimer);
    toast.couchTimer = win.setTimeout(() => { toast.hidden = true; }, 3000);
  };
}
