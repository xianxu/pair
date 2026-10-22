// Sticky notes on the broadcast viewer (#447). Couch draws each note's box
// and 💭 into the screen itself (#488); this page only keeps the list (event:
// notes), turns a tap on a 💭 into "open this note"; the dialog that owns a draft until
// Save is collab.js's noteDialog, shared with the web cast (#470).

// guardedCell reports whether a cell is on the broadcast's controls (the
// first `guard` columns of the first declared chrome row), which no note covers.
export function guardedCell(col, row, rows, guard, chromeRows = 1) {
  return !Number.isInteger(chromeRows) || chromeRows <= 0 || chromeRows >= rows || (row === rows - chromeRows && col < guard);
}

// noteVisible mirrors Couch's NoteVisible: both of the 💭's cells on the
// grid, and off the controls.
export function noteVisible(n, cols, rows, guard, chromeRows = 1) {
  return n.col >= 0 && n.row >= 0 && n.col + 1 < cols && n.row < rows && !guardedCell(n.col, n.row, rows, guard, chromeRows);
}

// noteAt is the visible note whose 💭 covers the cell, or null.
export function noteAt(list, col, row, cols, rows) {
  return list.notes.find((n) => noteVisible(n, cols, rows, list.guard, list.chrome_rows) && n.row === row && (n.col === col || n.col + 1 === col)) || null;
}

// boxHasPin mirrors Couch's hasPin: a box [col, row, w, h] on the grid
// has somewhere inside it for the 💭, off the controls. Couch places the
// pin. Both run tests/broadcast-viewer/pinspots.json.
export function boxHasPin(box, cols, rows, guard, chromeRows = 1) {
  const [c, r, w, h] = box;
  if (!(c >= 0 && r >= 0 && c < cols && r < rows && w >= 2 && h >= 1 && w <= cols - c && h <= rows - r)) return false;
  for (let row = box[1]; row < box[1] + box[3]; row++) {
    for (let col = box[0] + box[2] - 2; col >= box[0]; col--) {
      if (noteVisible({ col, row }, cols, rows, guard, chromeRows)) return true;
    }
  }
  return false;
}

const isInt = (v) => Number.isInteger(v) && v >= 0;
const isBox = (b) => Array.isArray(b) && b.length === 4 && b.every(isInt) && b[2] >= 1 && b[3] >= 1;

// parseNotes keeps only well-formed entries of a notes event.
export function parseNotes(data) {
  const empty = { guard: 0, notes: [] };
  if (!data || typeof data !== 'object' || !isInt(data.guard) || !Array.isArray(data.notes)) {
    return empty;
  }
  const notes = data.notes.filter((n) => n && isInt(n.id) && isInt(n.col) && isInt(n.row) && isBox(n.box) && isInt(n.color) && typeof n.text === 'string');
  return { guard: data.guard, notes: notes.map(({ id, col, row, box, color, text, owner, edited }) => ({
    id, col, row, box: [...box], color, text,
    ...(typeof owner === 'string' ? { owner } : {}),
    ...(typeof edited === 'boolean' ? { edited } : {}),
  })) };
}

export { MAX_NOTE_BYTES, noteBytes, noteDialog } from './collab.js';
