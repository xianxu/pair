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
  return {
    get() {
      try {
        const v = storage ? storage.getItem(COLOR_KEY) : null;
        return /^[0-9]$/.test(v || '') ? Number(v) : null;
      } catch {
        return null;
      }
    },
    set(i) {
      try {
        storage?.setItem(COLOR_KEY, String(i));
      } catch {
        // Not remembered; the pick still holds for this page.
      }
    },
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
//   close()
export function noteDialog(els, deps) {
  let state = null; // {kind: 'draft', payload} | {kind: 'view', id}
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
    els.save.disabled = n === 0 || n > MAX_NOTE_BYTES || els.input.value.trim() === '';
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
    if (state?.kind === 'view') {
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
      say('');
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
      say('');
      els.resolve.classList.toggle('hidden', !resolvable);
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
    close,
    get open() {
      return state !== null;
    },
  };
}
