// What the recording can't hold, marked where it is drawn (pair#513): a
// canvas's pixels, a video's frames, a plugin's output, a frame whose
// document wasn't recorded. Each gets a dashed outline at its drawn box and a
// small label ("canvas: not captured"), so a viewer, or an agent reading the
// page, knows something is there. The renderer sizes a canvas to the
// operator's live layout, so the box (and the label with it) follows the
// operator's size as it changes. Recording the content itself is #515.

// label is the sentence a placeholder shows.
export function label(kind) {
  return `${kind === 'iframe' ? 'frame' : kind}: not captured`;
}

// mountPlaceholders draws the marks over the page box. v: the viewer's
// surface (uncaptured, boxOf, fromPage, onRecord). The shell also calls
// draw on every paint, so the marks follow the drawing even when no record
// arrives (a replay applied before this mounted).
export function mountPlaceholders(doc, win, v) {
  const box = doc.getElementById('box');
  const layer = doc.createElement('div');
  layer.className = 'uncaptured';
  box.append(layer);
  const marks = new Map(); // id -> element

  function draw() {
    const r = box.getBoundingClientRect();
    const seen = new Set();
    for (const { id, kind } of v.uncaptured()) {
      const b = v.boxOf(id);
      if (!b || b[2] <= 0 || b[3] <= 0) continue;
      let m = marks.get(id);
      if (!m) {
        m = doc.createElement('div');
        m.className = 'uncaptured-mark';
        const tag = doc.createElement('span');
        m.append(tag);
        layer.append(m);
        marks.set(id, m);
      }
      m.firstChild.textContent = label(kind);
      const [x0, y0] = v.fromPage(b[0], b[1]), [x1, y1] = v.fromPage(b[0] + b[2], b[1] + b[3]);
      Object.assign(m.style, { left: `${x0 - r.left}px`, top: `${y0 - r.top}px`, width: `${x1 - x0}px`, height: `${y1 - y0}px` });
      seen.add(id);
    }
    for (const [id, m] of marks) {
      if (!seen.has(id)) { m.remove(); marks.delete(id); }
    }
  }

  // One redraw per frame at most, two frames out, so the renderer has drawn
  // the record that asked for it before its boxes are read.
  let queued = false;
  const schedule = () => {
    if (queued) return;
    queued = true;
    win.requestAnimationFrame(() => win.requestAnimationFrame(() => { queued = false; draw(); }));
  };
  v.onRecord(schedule);
  win.addEventListener('resize', schedule);
  schedule();
  return { draw, marks };
}
