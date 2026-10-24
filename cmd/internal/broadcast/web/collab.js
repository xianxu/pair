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
// route from routes, the link's session cookie (same-origin credentials,
// pair#495), no referrer, plus any headers (a remote-driving seat's key,
// #467). It resolves to the status and the
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
      credentials: 'same-origin',
      referrerPolicy: 'no-referrer',
      cache: 'no-store',
    }).then(async (r) => ({ status: r.status, text: await r.text() }));
  };
}

// sessionLost asks the link whether this page's session still holds
// (pair#495): the gate answers `session` with 204 for a live session and 401
// once the code was rotated. Anything else -- the network, the cast gone --
// is not a lost session, so the caller's own retry handles it.
export async function sessionLost(fetchFn) {
  try {
    const r = await fetchFn('session', { credentials: 'same-origin', cache: 'no-store', referrerPolicy: 'no-referrer' });
    return r.status === 401;
  } catch {
    return false;
  }
}

// A viewer's name, as every cast page keeps it (pair#495): the lock page
// asks it with the code, and the web viewer's notes carry it. One key and one
// rule, the board's (webcast/board wire.go nameText): plain, one line, at
// most MAX_VIEWER_NAME characters.
export const VIEWER_NAME_KEY = 'couch.viewerName';
export const MAX_VIEWER_NAME = 40;

// validName is a name the board takes, trimmed, or '' for none.
export function validName(v) {
  const n = String(v).trim();
  return [...n].length <= MAX_VIEWER_NAME && !/[\p{Cc}\p{Cf}]/u.test(n) ? n : '';
}

// storedName is the name this browser's viewer gave the lock page, as the
// board takes it, or ''. Read through safeStorage: storage may be missing.
export function storedName(storage) {
  return validName(safeStorage(storage).get(VIEWER_NAME_KEY) || '');
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
// [x, y, w, h], inside view. preview suppresses the tap box until release.
export const DRAG_MIN = {
  cells: { mouse: 1, touch: 2 },
  px: { mouse: 8, touch: 16 },
};

export function dragMin(unit, pointerType) {
  return DRAG_MIN[unit][pointerType === 'touch' ? 'touch' : 'mouse'];
}

// A note press becomes a drag once it crosses the pixel threshold; moving
// back never turns it into a click. The owner discards it on release/cancel.
export function noteGesture(from, pointerType) {
  return {
    from, to: from, pointerType, dragged: false,
    move(to) {
      this.to = to;
      const min = dragMin('px', this.pointerType);
      this.dragged ||= Math.abs(to[0] - this.from[0]) >= min || Math.abs(to[1] - this.from[1]) >= min;
    },
  };
}

export function gestureBox(from, to, view, { min, cells = false, preview = false }) {
  const tap = Math.abs(to[0] - from[0]) < min && Math.abs(to[1] - from[1]) < min;
  if (preview && tap) return null; // a click has no box until release
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


// Public ownership is a UI hint only; the server authenticates the cookie.
export async function sessionNoteOwner(fetchFn) {
  try {
    const r = await fetchFn('session', { credentials: 'same-origin', cache: 'no-store', referrerPolicy: 'no-referrer' });
    return r.status === 204 ? (r.headers?.get('X-Couch-Note-Owner') || '') : '';
  } catch { return ''; }
}

const closedNote = (epoch = 0, held = '') => ({ epoch, mode: 'closed', target: null, payload: null, text: '', held, pending: null, error: '' });
const validNote = (text) => text.trim() !== '' && noteBytes(text) <= MAX_NOTE_BYTES;
const HELD_NOTE_RECOVERY = 'Your current note is preserved. Finish or cancel it before opening another note.';

// One pure lifecycle for draft, read-only and existing-note editors. Pending
// tokens belong to an epoch; neither typing nor a hold permits a second send.
// Creation locks text until its result: it cannot turn a completed draft into
// another creation. Existing-note edits can retain newer unsent text safely.
export function noteDialogTransition(previous, event) {
  let state = previous || closedNote();
  let effect = null;
  const e = event;
  if (e.type.startsWith('open-') && state.held && ['draft', 'edit'].includes(state.mode)) {
    state = { ...state, error: HELD_NOTE_RECOVERY };
  } else if (e.type.startsWith('open-')) {
    const mode = e.type.slice(5);
    state = { ...closedNote(state.epoch + 1, state.held), mode, target: e.note || null, payload: e.payload || null, blocked: mode === 'draft' && !!e.payload?.capturePending, text: e.note?.text || '' };
  } else if (e.type === 'dismiss' || e.type === 'retire') {
    state = closedNote(state.epoch + 1, state.held);
  } else if (e.type === 'input' && ['draft', 'edit'].includes(state.mode) && state.pending?.kind !== 'save-new') {
    state = { ...state, text: e.text };
  } else if (e.type === 'capture' && state.mode === 'draft' && state.payload?.request === e.request) {
    const update = e.update;
    state = { ...state, payload: { ...state.payload, ...update.payload }, blocked: 'blocked' in update ? !!update.blocked : 'pending' in update ? !!update.pending : state.blocked, captureMessage: update.message || '', retry: !!update.retry };
  } else if (e.type === 'hold') {
    state = { ...state, held: e.reason || '' };
  } else if ((e.type === 'save' || e.type === 'resolve') && !state.held && !state.pending && !(state.mode === 'draft' && state.blocked)) {
    const kind = e.type === 'resolve'
      ? (['view', 'edit'].includes(state.mode) && state.target?.resolvable ? 'resolve' : null)
      : (['draft', 'edit'].includes(state.mode) && validNote(state.text) ? (state.mode === 'draft' ? 'save-new' : 'save-edit') : null);
    if (kind) {
      const pending = { epoch: state.epoch, kind, text: state.text };
      state = { ...state, pending, error: '' };
      effect = { kind, pending, target: state.target, payload: state.payload, text: state.text };
    }
  } else if (e.type === 'result' && state.pending && e.pending === state.pending && e.pending.epoch === state.epoch) {
    if (e.status === 204 && (state.text === state.pending.text || state.pending.kind === 'resolve')) {
      state = closedNote(state.epoch + 1, state.held);
    } else {
      state = { ...state, pending: null, error: e.status === 204 ? 'The earlier text was saved. Save again to submit your latest text.' : (e.text || '').trim() || "That didn't work. Try again." };
    }
  } else if (e.type === 'refresh' && state.mode === 'view') {
    state = { ...state, target: { ...state.target, ...e.note }, text: e.note.text ?? state.text };
  }
  return { state, effect };
}

// DOM adapter: every lifecycle change goes through the transition above.
// editRequest is optional; default terminal edits contain only id and text.
export function noteDialog(els, deps) {
  let state = closedNote();
  const resolveRequest = deps.resolveRequest || ((id) => ['resolve', { id }]);
  const editRequest = deps.editRequest || ((id, text) => ['note-edit', { id, text }]);
  const render = () => {
    const editable = ['draft', 'edit'].includes(state.mode);
    els.root.classList.toggle('hidden', state.mode === 'closed');
    els.root.classList.toggle('draft', editable);
    els.root.classList.toggle('view', state.mode === 'view');
    if (els.input.value !== state.text) els.input.value = state.text;
    els.input.readOnly = state.pending?.kind === 'save-new';
    els.text.textContent = state.mode === 'view' ? state.text + (state.target?.edited ? ' (edited)' : '') : '';
    els.count.textContent = `${state.target?.edited ? '(edited) · ' : ''}${noteBytes(state.text)} / ${MAX_NOTE_BYTES}`;
    els.msg.textContent = [state.held, state.error || state.captureMessage].filter(Boolean).join(' ');
    els.save.classList.toggle('hidden', !editable);
    els.save.disabled = !editable || !!state.held || !!state.pending || !!state.blocked || !validNote(state.text);
    els.resolve.classList.toggle('hidden', !state.target?.resolvable);
    els.resolve.disabled = !!state.held || !!state.pending || !state.target?.resolvable;
    els.cancel.textContent = editable ? 'Cancel' : 'Close';
    if (els.retry) els.retry.hidden = state.mode !== 'draft' || !state.retry;
  };
  const dispatch = (event) => {
    const old = state;
    const next = noteDialogTransition(state, event);
    state = next.state;
    render();
    if (old.mode === 'draft' && old.epoch !== state.epoch && event.type !== 'result') deps.onDiscard?.(old.payload);
    if (old.mode !== 'closed' && state.mode === 'closed') deps.onClose?.();
    return next.effect;
  };
  const answer = (effect, request) => {
    let promise;
    try { promise = effect.kind === 'save-new' && deps.saveDraft ? deps.saveDraft(request[1]) : deps.post(...request); } catch { promise = Promise.reject(); }
    Promise.resolve(promise).then(
      (r) => dispatch({ type: 'result', pending: effect.pending, status: r.status, text: r.text }),
      () => dispatch({ type: 'result', pending: effect.pending, text: "Couldn't reach the broadcast. Try again." }),
    );
  };
  els.input.addEventListener('input', () => dispatch({ type: 'input', text: els.input.value }));
  const close = () => dispatch({ type: 'dismiss' });
  els.cancel.addEventListener('click', close);
  els.retry?.addEventListener('click', () => { if (state.mode === 'draft' && state.retry && !state.pending) deps.onRetry?.(); });
  els.save.addEventListener('click', () => {
    if (els.save.disabled) return;
    const extra = state.mode === 'draft' && deps.beforeSave ? deps.beforeSave() : {};
    if (!extra) return;
    const effect = dispatch({ type: 'save' });
    if (!effect) return;
    answer(effect, effect.kind === 'save-new'
      ? ['note', { ...effect.payload, color: deps.color(), text: effect.text, ...extra }]
      : editRequest(effect.target.id, effect.text));
  });
  els.resolve.addEventListener('click', () => {
    const effect = dispatch({ type: 'resolve' });
    if (effect) answer(effect, resolveRequest(effect.target.id));
  });
  els.root.addEventListener('keydown', (e) => { if (e.key === 'Escape') close(); });
  const openExisting = (mode, note, resolvable) => {
    const epoch = state.epoch;
    dispatch({ type: `open-${mode}`, note: { ...note, resolvable } });
    const opened = state.epoch !== epoch;
    (opened && mode === 'edit' ? els.input : els.cancel).focus?.();
    return opened;
  };
  return {
    draft(payload) { dispatch({ type: 'open-draft', payload }); els.input.focus?.(); },
    projectDraft(request, update) {
      if (state.mode !== 'draft' || state.payload?.request !== request) return false;
      dispatch({ type: 'capture', request, update });
      return true;
    },
    view(note, { resolvable = false } = {}) { return openExisting('view', note, resolvable); },
    edit(note, { resolvable = true } = {}) { return openExisting('edit', note, resolvable); },
    update(list) {
      if (!['view', 'edit'].includes(state.mode)) return;
      const note = list.notes.find((n) => n.id === state.target.id);
      if (!note) dispatch({ type: 'retire' });
      else dispatch({ type: 'refresh', note });
    },
    hold(reason) { dispatch({ type: 'hold', reason }); },
    close,
    get open() { return state.mode !== 'closed'; },
    get editable() { return state.mode === 'draft' || state.mode === 'edit'; },
    get replacementBlocked() { return state.error === HELD_NOTE_RECOVERY; },
    get draftPayload() { return state.mode === 'draft' ? state.payload : null; },
    // What a submission must not take from under the operator (pair#849): a
    // new note not yet saved, or an existing note whose text was changed.
    get unsaved() {
      if (state.mode === 'draft') return { draft: true };
      if (state.mode === 'edit' && state.text !== (state.target?.text ?? '')) return { id: state.target?.id || '', text: state.target?.text || '' };
      return null;
    },
  };
}

// sayer is a page's one status toast: a sentence shown for a moment (a
// refused request no dialog is open for). Every module of the page shares
// the one element, so two never overlap.
export function sayer(doc, win, root = doc.body) {
  let toast = root.querySelector('.annotate-msg');
  if (!toast) {
    toast = doc.createElement('div');
    toast.className = 'annotate-msg';
    toast.setAttribute('role', 'status');
    toast.hidden = true;
    root.append(toast);
  }
  return (text) => {
    toast.textContent = text;
    toast.hidden = false;
    if (toast.couchTimer) win.clearTimeout(toast.couchTimer);
    toast.couchTimer = win.setTimeout(() => { toast.hidden = true; }, 3000);
  };
}
