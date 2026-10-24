/**
 * Ephemeral display rows over accepted annotations and sketch-note entries.
 * Sources remain authoritative; annotation ownership is resolved by openNote.
 * Consumes each input once and preserves first occurrence of each typed key.
 */
export function feedbackEntries(notes = [], sketches = []) {
  const rows = [];
  const seen = new Set();
  for (const note of notes) {
    const id = note?.id;
    if (typeof id !== 'string' || id.length === 0) continue;
    const key = 'note:' + id;
    if (seen.has(key)) continue;
    seen.add(key);
    const text = note.text;
    rows.push({key, kind: 'note', id, text: typeof text === 'string' ? text : '', editable: false, source: note});
  }
  for (const sketch of sketches) {
    const id = sketch?.sketch;
    if (typeof id !== 'string' || id.length === 0) continue;
    const key = 'sketch:' + id;
    if (seen.has(key)) continue;
    seen.add(key);
    rows.push({key, kind: 'sketch', id, text: String(sketch.label ?? ''), editable: sketch.editable === true, source: sketch});
  }
  return rows;
}
