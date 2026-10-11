// Sticky notes on the broadcast viewer (#447). Couch draws each note's 💭
// into the screen itself; this page only keeps the list (event: notes), turns
// a tap on a 💭 into "open this note", and owns a draft until Save. A draft
// never leaves the page before Save, and note text is only ever set as
// textContent.

export const MAX_NOTE_BYTES = 1024;

// noteBytes counts UTF-8 bytes, as Couch's cap does.
export function noteBytes(text) {
  return new TextEncoder().encode(text).length;
}

// guardedCell reports whether a cell is on the broadcast's controls (the
// first `guard` columns of the last row), which no note covers.
export function guardedCell(col, row, rows, guard) {
  return row === rows - 1 && col < guard;
}

// noteVisible mirrors Couch's NoteVisible: both of the 💭's cells on the
// grid, and off the controls.
export function noteVisible(n, cols, rows, guard) {
  return n.col >= 0 && n.row >= 0 && n.col + 1 < cols && n.row < rows && !guardedCell(n.col, n.row, rows, guard);
}

// noteAt is the visible note whose 💭 covers the cell, or null.
export function noteAt(list, col, row, cols, rows) {
  return list.notes.find((n) => noteVisible(n, cols, rows, list.guard) && n.row === row && (n.col === col || n.col + 1 === col)) || null;
}

// noteCell is where a tap places a note: the 💭 is two cells wide, so a tap
// on the last column places it one to the left.
export function noteCell(cell, cols) {
  return [Math.min(cell[0], cols - 2), cell[1]];
}

const isInt = (v) => Number.isInteger(v) && v >= 0;

// parseNotes keeps only well-formed entries of a notes event.
export function parseNotes(data) {
  const empty = { guard: 0, notes: [] };
  if (!data || typeof data !== 'object' || !isInt(data.guard) || !Array.isArray(data.notes)) {
    return empty;
  }
  const notes = data.notes.filter((n) => n && isInt(n.id) && isInt(n.col) && isInt(n.row) && isInt(n.color) && typeof n.text === 'string');
  return { guard: data.guard, notes: notes.map(({ id, col, row, color, text }) => ({ id, col, row, color, text })) };
}

// noteDialog drives the one sticky-note dialog. els: root, text, input,
// count, msg, save, cancel, resolve (element-likes). deps: post(route, body)
// → Promise<{status, text}>, color() → the viewer's pointer color.
//   draft(cell, grid)          a new note at cell on a {cols, rows} grid
//   view(note, {resolvable})   an existing note, read-only
//   update(list)               a new notes list: a viewed note that is gone
//                              (resolved elsewhere, or pointing turned off)
//                              closes its dialog
//   close()
export function noteDialog(els, deps) {
  let state = null; // {kind: 'draft', cell, grid} | {kind: 'view', id}
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
    const { cell, grid } = state;
    els.save.disabled = true;
    answer(deps.post('note', { cols: grid.cols, rows: grid.rows, col: cell[0], row: cell[1], color: deps.color(), text: els.input.value }), close)
      .finally(() => state?.kind === 'draft' && refreshCount());
  });
  els.resolve.addEventListener('click', () => {
    if (state?.kind === 'view') {
      answer(deps.post('resolve', { id: state.id }), close);
    }
  });
  els.root.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      close();
    }
  });
  return {
    draft(cell, grid) {
      gen++;
      state = { kind: 'draft', cell, grid };
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
