// Pointing and notes in the web cast viewer (#470). A viewer picks a color
// on the swatch rail, draws strokes everyone sees, and leaves notes pinned on
// an element (a tap) or a dragged region. Strokes and notes are recording
// records: a viewer posts them (stroke, note, note-state), Couch's board
// checks and anchors them, and they come back to every viewer in the record
// stream, so this module draws what the model holds, never its own guess.
// Its own stroke is drawn as it is made and its echo skipped.
//
// The pure helpers are exported for the node tests; mountAnnotate is the
// shell, called once by viewer.js. It owns its own elements (rail, mode
// buttons, strokes canvas, pins, dialog), so the viewer's page stays as it
// is, and note text is only ever set as textContent. collab.js is the
// terminal viewer's (served from broadcast.CollabJS; the file here is a
// symlink, for the node tests).

import { mountFeedbackPanel } from './feedback-panel.js';
import { mountFeedbackGlow } from './feedback-glow.js';
import { captureDraftController } from './capture-draft.js';
import { annotationPolicy } from './annotation-policy.js';
import { HEX_COLOR, pointerColor, colorStore, safeStorage, poster, sayer, railAt, renderSwatches, noteDialog, sessionNoteOwner, gestureBox, noteGesture, dragMin, boxCentre, validName, VIEWER_NAME_KEY, MAX_VIEWER_NAME } from './collab.js';
import { feedbackReady, feedbackScopeKey, onFeedbackPage } from './page-load.js';

export { validName };

// CHUNK_MS is how often a stroke in progress is posted; MAX_POINTS caps a
// chunk (the board's bound).
export const CHUNK_MS = 50;
export const MAX_POINTS = 64;
// FADE_MS: a stroke fades out over this long after its last point.
export const FADE_MS = 1500;

const MAX_NAME = MAX_VIEWER_NAME;
const ID_KEY = 'couch.viewerId';
const NAME_KEY = VIEWER_NAME_KEY;
const ID_RE = /^[A-Za-z0-9_-]{1,64}$/;

// randomID is 12 hex digits from the browser's CSPRNG.
export function randomID(crypto) {
  const b = new Uint8Array(6);
  crypto.getRandomValues(b);
  return [...b].map((x) => x.toString(16).padStart(2, '0')).join('');
}

// identityStore is who this viewer says they are: an id made once per
// browser and a name asked once. It is attribution, not authentication
// (atlas, Notes and pointer). Storage can be missing or throw; the page then
// keeps the identity for this visit only.
export function identityStore(storage, crypto) {
  const store = safeStorage(storage);
  let id = store.get(ID_KEY);
  if (!ID_RE.test(id || '') || id === 'operator') {
    id = 'w' + randomID(crypto);
    store.set(ID_KEY, id);
  }
  // A stored name is read by the board's rules (plain, one line, at most
  // 40 characters); anything else is asked for again.
  let name = validName(store.get(NAME_KEY) || '');
  return {
    id: () => id,
    name: () => name,
    setName(v) { name = v; store.set(NAME_KEY, v); },
    // adopt is this visit's id, not remembered: a remote-driving seat's
    // (#467), so a driver's notes are attributable to the driver.
    adopt(v) { if (ID_RE.test(v) && v !== 'operator') id = v; },
    viewer: () => ({ id, name }),
  };
}


// chunker collects one stroke's points ([x, y] in page px, at ms) and cuts
// them into posts: each at most MAX_POINTS, every point [x, y, ms after the
// chunk's first].
export function chunker() {
  let pending = [];
  return {
    add(x, y, at) {
      const last = pending.at(-1);
      if (last && last[0] === x && last[1] === y) return;
      pending.push([x, y, at]);
    },
    take() {
      const out = [];
      for (let i = 0; i < pending.length; i += MAX_POINTS) {
        const part = pending.slice(i, i + MAX_POINTS), t0 = part[0][2];
        out.push(part.map(([x, y, at]) => [round(x), round(y), Math.max(0, Math.round(at - t0))]));
      }
      pending = [];
      return out;
    },
  };
}

const round = (v) => Math.round(v * 10) / 10;

// strokeStore keeps the strokes to draw: chunks joined by record id, each
// with the time it last grew here and its end on the recording's clock. A
// stroke behind the stream (a reconnect's catch-up replays the document's
// strokes oldest first) is never drawn: at draw time, only a stroke whose
// end is within FADE_MS of the newest record time shows. Own strokes are
// added locally (local), and their echoes are skipped.
export function strokeStore() {
  const strokes = new Map(); // id -> {color, pts: [[x, y]], last, end}
  let newest = 0;
  const extend = (id, color, pts, now, end) => {
    const s = strokes.get(id) || { color, pts: [], last: now, end };
    for (const p of pts) s.pts.push([p[0], p[1]]);
    s.last = now;
    if (end !== undefined) s.end = Math.max(s.end ?? end, end);
    strokes.set(id, s);
  };
  return {
    seen(t) { newest = Math.max(newest, t); },
    record(r, now, ownPrefix) {
      if (ownPrefix && r.id.startsWith(ownPrefix)) return;
      extend(r.id, r.color, r.pts, now, r.t + Math.max(0, ...r.pts.map((p) => p[2])));
    },
    local(id, color, pts, now) { extend(id, color, pts, now, undefined); },
    // visible is every stroke still showing at now, with its opacity; the
    // ones faded out, or behind the stream, are dropped.
    visible(now) {
      const out = [];
      for (const [id, s] of strokes) {
        const age = now - s.last;
        if (age >= FADE_MS || (s.end !== undefined && newest - s.end > FADE_MS)) { strokes.delete(id); continue; }
        out.push({ id, color: s.color, pts: s.pts, alpha: 1 - age / FADE_MS });
      }
      return out;
    },
    clear() { strokes.clear(); },
    get size() { return strokes.size; },
  };
}

// liveNotes are the open notes on the current document: since it began
// (docSeg), so a resume keeps them and a navigation clears them.
export function liveNotes(model) {
  if (!model || !model.docSeg) return [];
  return model.notes.filter((n) => (typeof model.onPageLoad==='function'?onFeedbackPage(model,n.seg):feedbackReady(model)&&n.seg>=model.docSeg) && n.state === 'open').map((n) => n.note);
}

// Current-load open notes remain reachable when their anchor is absent.
export function openNotes(model, feedbackMask) {
  return (model?.notes || []).filter(n => onFeedbackPage(model,n.seg)&&n.state === 'open' && (!feedbackMask || feedbackMask.noteVisible(n.note.id,n.revision??0))).map(n => n.note);
}

export function noteList(doc, container, open) {
  const rows = new Map();
  let current = new Map();
  return { draw(notes, placementOf) {
    current = new Map(notes.map(n => [n.id, n]));
    for (const [id, row] of rows) if (!current.has(id)) { row.remove(); rows.delete(id); }
    for (const n of notes) {
      let row = rows.get(n.id);
      if (!row) {
        row = doc.createElement('button'); row.type = 'button'; row.className = 'annotation-note-row';
        row.addEventListener('click', e => { e.stopPropagation(); const latest = current.get(n.id); if (latest) open(latest, latest.author?.name || 'A viewer'); });
        rows.set(n.id, row); container.append(row);
      }
      const absent = !placementOf(n);
      row.textContent = `${n.author?.name || 'A viewer'}: ${n.text}${absent ? ' — not on this view' : ''}`;
      row.dataset.placement = absent ? 'absent' : 'visible';
    }
    container.hidden = notes.length === 0;
  } };
}

// noteBox is where a note shows, in page px: its region (every note made
// since #488 has one); else, for an older note, its element's box now
// (boxOf), while the element is drawn; else the box it was made on.
export function noteBox(note, boxOf) {
  if (note.region) return note.region;
  const b = note.at && note.at.node ? boxOf(note.at.node) : null;
  return b || (note.at && note.at.box) || null;
}

// regionOf is a note gesture's box (#488), in page px: the dragged box, or
// for a tap the default box; view is the page shown, pointerType the
// pointer's (a finger must drag further).
export function regionOf(a, b, view, pointerType, { preview = false, dragged = false } = {}) {
  return gestureBox(a, b, view, { min: dragged ? 0 : dragMin('px', pointerType), preview });
}

// noteDraft is a new note's payload, what the dialog posts: the segment
// shown, the element it anchors on (the one at the box's centre), that
// element's box as drawn here, and the note's box, its region.
export function noteDraft(seg, node, box, region) {
  return { seg, node, box: box.map(round), region };
}

// strokesAt is what a reader at moment t of a recording shows of its
// strokes (#477's replay calls this with the model at t): the current
// document's strokes, chunks joined by id, the points drawn by t, each
// fading over FADE_MS after its last point. Pure.
export function strokesAt(model, t) {
  if (!model || !model.docSeg) return [];
  const byId = new Map();
  for (const { stroke: r, seg } of model.strokes) {
    if (seg < model.docSeg || r.t > t) continue;
    const s = byId.get(r.id) || { id: r.id, color: r.color, pts: [], end: r.t };
    for (const p of r.pts) {
      if (r.t + p[2] > t) break;
      s.pts.push([p[0], p[1]]);
      s.end = Math.max(s.end, r.t + p[2]);
    }
    byId.set(r.id, s);
  }
  const out = [];
  for (const s of byId.values()) {
    if (!s.pts.length || t - s.end >= FADE_MS) continue;
    out.push({ id: s.id, color: s.color, pts: s.pts, alpha: 1 - (t - s.end) / FADE_MS });
  }
  return out;
}

// paintStrokes draws strokes ({color, pts, alpha}) on a canvas covering
// rect (client px), mapping page px with toClient. Live and replay share it.
export function paintStrokes(canvas, list, toClient, rect, dpr) {
  const w = Math.round(rect.width * dpr), h = Math.round(rect.height * dpr);
  if (canvas.width !== w || canvas.height !== h) { canvas.width = w; canvas.height = h; }
  const g = canvas.getContext('2d');
  g.setTransform(dpr, 0, 0, dpr, 0, 0);
  g.clearRect(0, 0, rect.width, rect.height);
  g.lineCap = 'round';
  g.lineJoin = 'round';
  g.lineWidth = 4;
  for (const s of list) {
    g.globalAlpha = s.alpha;
    g.strokeStyle = s.color;
    g.beginPath();
    s.pts.forEach(([x, y], i) => {
      const [cx, cy] = toClient(x, y);
      if (i === 0) g.moveTo(cx - rect.left, cy - rect.top); else g.lineTo(cx - rect.left, cy - rect.top);
    });
    if (s.pts.length === 1) { const [cx, cy] = toClient(s.pts[0][0], s.pts[0][1]); g.lineTo(cx - rect.left + 0.1, cy - rect.top); }
    g.stroke();
  }
  g.globalAlpha = 1;
}

// CHIP_ROOM is the size, in client px, a box needs to hold its 💭 chip
// (26 px) inside its corner.
export const CHIP_ROOM = 34;

// pinLayer keeps one pin per note in container: draw(notes, boxOf, place)
// adds and removes pins only when the set changes and otherwise moves them,
// so a chip being tapped is never replaced under the finger. open(note,
// author) is a tap on a pin's chip. Live and replay share it.
export function pinLayer(doc, container, open) {
  const els = new Map(); // note id -> element
  const make = (n) => {
    const pin = doc.createElement('div');
    pin.className = 'pin';
    pin.dataset.note = n.id;
    pin.style.setProperty('--pin', n.author && HEX_COLOR.test(n.author.color || '') ? n.author.color : '#888888');
    const chip = doc.createElement('button');
    chip.type = 'button';
    chip.className = 'chip';
    chip.textContent = '💭';
    const entry = { pin, chip, note: n };
    chip.addEventListener('click', (e) => { e.stopPropagation(); open(entry.note, entry.note.author?.name || 'A viewer'); });
    pin.append(chip);
    container.append(pin);
    return entry;
  };
  return {
    draw(notes, boxOf, place, { placementOf } = {}) {
      const ids = new Set(notes.map((n) => n.id));
      for (const [id, entry] of els) if (!ids.has(id)) { entry.pin.remove(); els.delete(id); }
      for (const n of notes) {
        const entry = els.get(n.id) || make(n);
        entry.note = n;
        els.set(n.id, entry);
        const { pin, chip } = entry;
        const by = n.author?.name || 'A viewer';
        chip.title = `${by}: ${n.text}${n.edited ? ' (edited)' : ''}`;
        chip.setAttribute('aria-label', `Note from ${by}${n.edited ? ' (edited)' : ''}`);
        const b = placementOf ? placementOf(n) : noteBox(n, boxOf);
        pin.hidden = !b;
        if (b) {
          place(pin, b);
          // A box too thin for the chip carries it outside its corner.
          pin.classList.toggle('thin', parseFloat(pin.style.width) < CHIP_ROOM || parseFloat(pin.style.height) < CHIP_ROOM);
        }
      }
    },
  };
}

// NOTE_ROUTES is every request this module makes besides palette.json
// (collab.js's poster sends them).
const NOTE_ROUTES = ['stroke', 'note', 'note-edit', 'note-state'];

export const MODE_TIPS = {
  point: 'Point: draw on the page for everyone. Click again to stop.',
  note: 'Note: drag a box, or tap for a small one, to leave a note. Click again to stop.',
};

// readPalette reads palette.json (pair#506): the swatches' colors and
// whether this viewer's link may annotate. Any other shape, including the
// bare array an older server sent, is null: the cast takes no notes.
export function readPalette(x) {
  if (!x || typeof x !== 'object' || Array.isArray(x) || typeof x.annotate !== 'boolean') return null;
  const p = x.palette;
  if (!Array.isArray(p) || !p.length || !p.every((c) => typeof c === 'string' && HEX_COLOR.test(c))) return null;
  return { palette: p, annotate: x.annotate };
}

// mountAnnotate adds pointing and notes to the viewer. v is the viewer's
// surface (couchViewer: boxOf, nodeAt, toPage, fromPage, onRecord, model).
// It mounts only when the cast takes notes (palette.json answers). Under a
// view-only link (annotate false) it draws others' strokes and notes
// read-only: the rail never shows, so no mode can be picked, and a note
// opens without Resolve; the server refuses the posts too.
export async function mountAnnotate(doc, win, v) {
  let answer;
  try {
    const r = await win.fetch('palette.json', { credentials: 'same-origin', cache: 'no-store' });
    answer = r.ok ? readPalette(await r.json()) : null;
  } catch { answer = null; }
  if (!answer) return null;
  const { palette, annotate } = answer;
  const owner = await sessionNoteOwner((...args) => win.fetch(...args));

  let storage = null;
  try { storage = win.localStorage; } catch { /* no storage: nothing remembered */ }
  const who = identityStore(storage, win.crypto);
  if (v.seat) who.adopt(v.seat.viewer);
  const colors = colorStore(storage);
  let generation;
  const post = poster((url, init) => win.fetch(url, {...init, headers: {...init.headers, 'X-Couch-Annotation-Generation': String(generation)}}), NOTE_ROUTES);
  const captureCall = async (op, body, { timeoutMs, context }) => {
    const abort = new AbortController();
    const timer = win.setTimeout(() => abort.abort(), timeoutMs);
    try {
      const reply = await win.fetch(op, {method:'POST',credentials:'same-origin',cache:'no-store',referrerPolicy:'no-referrer',signal:abort.signal,headers:{'Content-Type':'application/json','X-Couch-Annotation-Generation':String(context ?? generation)},body:JSON.stringify(body)});
      if (!reply.ok) throw Error((await reply.text()).trim() || 'Capture request failed.');
      return await reply.json();
    } finally { win.clearTimeout(timer); }
  };
  captureCall.context = () => generation;
  const ui = mountAnnotation(doc, win, v, { captureCall, root: doc.body, box: doc.getElementById('box'), stage: doc.getElementById('stage'), palette, who, colors, post, annotate, owner, annotationAllowed: false });
  v.onCapabilities?.((caps) => {
    const c = caps?.annotation;
    ui.setAnnotationAllowed(c?.allowed === true, generation !== c?.generation, c?.state);
    generation = c?.generation;
  });
  return ui;
}

// Both the viewer and the isolated operator world mount the same UI.
export function mountAnnotation(doc, win, v, { root, box, stage, palette, who, colors, post, operator = false, prepareDraft, captureCall, annotate = true, annotationAllowed = true, owner = '', notesLayout }) {
  let availabilityState = 'off';
  let picked = colors.get();
  const color = () => pointerColor(picked, 0, palette.length);
  const el = (tag, cls, parent) => { const e = doc.createElement(tag); if (cls) e.className = cls; parent?.append(e); return e; };

  const layer = el('div', 'annotate', box);
  const canvas = el('canvas', 'strokes', layer);
  const pins = el('div', 'pins', layer);
  const notesPanel = mountFeedbackPanel(doc, win, {parent: root, layout: notesLayout});
  const glow = mountFeedbackGlow(doc, win, {parent: box, box});
  const noteRows = notesPanel.root;
  noteRows.setAttribute('aria-label', 'Open notes');
  noteRows.setAttribute('role', 'group');
  const rail = el('div', 'annotate-rail', root);
  const swatches = el('div', 'swatches', rail);
  // A refusal of a request no dialog is open for (a stroke) shows here for
  // a moment: every refused post is seen.
  const say = message => notesPanel.setStatus('editor', message);
  const answered = (req) => req.then((r) => { if (r.status !== 204) say((r.text || '').trim() || "That didn't work."); })
    .catch(() => say("Couldn't reach the cast."));
  swatches.setAttribute('role', 'group');
  swatches.setAttribute('aria-label', 'Pointer color');
  const modeButtons = {};
  for (const [mode, glyph] of [['point', '✏️'], ['note', '💭']]) {
    const b = el('button', 'mode', rail);
    b.type = 'button';
    b.textContent = v.tools ? (mode === 'point' ? '↖ Pointer' : '▤ Note') : glyph;
    b.title = MODE_TIPS[mode];
    b.setAttribute('aria-label', MODE_TIPS[mode]);
    b.addEventListener('click', () => setMode(mode === state.mode ? 'watch' : mode));
    modeButtons[mode] = b;
  }
  v.tools?.annotation({pointer: modeButtons.point, note: modeButtons.note, swatches});

  const hint = el('span', 'annotation-hint', rail);
  hint.setAttribute('role','status');

  // The dialog: collab.js's, with a name field the first time.
  const dlg = el('div', 'note-dialog hidden', root);
  dlg.setAttribute('role', 'dialog');
  dlg.setAttribute('aria-modal', 'true');
  dlg.setAttribute('aria-label', 'Note');
  notesPanel.adoptEditor(dlg);
  const author = el('div', 'note-author', dlg);
  const text = el('div', 'note-text', dlg);
  const name = el('input', 'note-name', dlg);
  name.placeholder = 'Your name';
  name.maxLength = 40;
  name.setAttribute('aria-label', 'Your name');
  const input = el('textarea', 'note-input', dlg);
  input.rows = 5;
  input.setAttribute('aria-label', 'Note text');
  const meta = el('div', 'note-meta', dlg);
  const msg = el('span', 'note-msg', meta);
  msg.setAttribute('role', 'status');
  const count = el('span', 'note-count', meta);
  const buttons = el('div', 'note-buttons', dlg);
  const mk = (label) => { const b = el('button', '', buttons); b.type = 'button'; b.textContent = label; return b; };
  const resolve = mk('Resolve'), retry = mk('Retry capture'), cancel = mk('Cancel'), save = mk('Save');
  retry.hidden = true;
  let captures;
  const dialog = noteDialog({ root: dlg, text, input, count, msg, save, cancel, resolve, retry }, {
    post,
    saveDraft: captureCall ? body => captures.save(body) : undefined,
    onDiscard: () => captures?.cancel(),
    onRetry: () => captures?.retry(),
    color,
    onClose: () => {v.feedbackVisibility?.invalidate();drawDrag();},
    editRequest: (id, text) => ['note-edit', { seg: v.model.seg, id, text }],
    resolveRequest: (id) => ['note-state', { viewer: who.viewer(), id, state: 'resolved' }],
    beforeSave: () => {
      const n = name.value.trim();
      if (!who.name() && !n) { msg.textContent = 'Add your name first.'; name.focus?.(); return null; }
      if (n && validName(n) !== n) { msg.textContent = 'Use a plain name of at most 40 characters.'; name.focus?.(); return null; }
      if (n && n !== who.name()) who.setName(n);
      return { viewer: who.viewer() };
    },
  });

  if (captureCall) captures = captureDraftController({
    call: captureCall, project: (request, update) => dialog.projectDraft(request, update),
    id: () => 'capture-' + randomID(win.crypto), now: () => win.performance.now(),
    setTimeout: (fn, ms) => win.setTimeout(fn, ms), clearTimeout: timer => win.clearTimeout(timer),
  });

  const state = { mode: 'watch', drag: null, stroke: null, timer: null, pointer: null };
  const strokes = strokeStore();
  // The document now: the module mounts after the catch-up may have been
  // applied, so it starts from the model's, not from nothing.
  let docSeg = v.model ? v.model.docSeg : 0;
  let loadScope = feedbackScopeKey(v.model);

  // While the cast is paused viewers do nothing (pair#499): the rail's
  // controls are disabled and say why, and the note dialog is held; the
  // server refuses the posts too.
  let policy;
  let gestureVersion = 0;
  const unavailable = () => policy?.phase !== 'ready';
  function transition(event) {
    const next = annotationPolicy(policy, event);
    policy = next.state;
    if (next.effects.includes('cancel')) { gestureVersion++; endGesture(true); }
    if (next.effects.includes('discard')) dialog.close();
    state.mode = policy.mode;
    layer.classList.toggle('point', state.mode === 'point');
    layer.classList.toggle('note', state.mode === 'note');
    // Driving leaves the operator's page untouched. Otherwise only the
    // chips can take input in watch mode, including read-only OFF notes.
    layer.classList.toggle('operator-off', operator && !!policy.driver);
    noteRows.classList.toggle('operator-off', operator && !!policy.driver);
    notesPanel.setEnabled(!!v.model?.seg && !v.model.ended && !(operator && policy.driver));
    if (next.effects.includes('cancel') || next.effects.includes('discard')) notesPanel.clearHighlight();
    layer.classList.toggle('operator-driving', operator && !!policy.driver);
    dlg.hidden = operator && (!!policy.driver || (policy.phase === 'disabled' && dialog.editable && !dialog.replacementBlocked));
    if (dlg.hidden) input.blur?.();
    renderRail();
    placeRail();
  }
  function sync() {
    const m = v.model || {};
    transition({ type: 'sync', doc: m.docSeg, ready: m.seg > 0&&feedbackReady(m), allowed: annotate && annotationAllowed, paused: m.paused, ended: m.ended, driver: operator ? m.driver : '' });
  }
  // The rail's controls are rendered only when their state changes (a pick,
  // a mode, a pause or a resume); each frame only places the rail, so
  // nothing that takes input is replaced under a click.
  let renderedColor = -1;
  const renderRail = () => {
    // Projection updates may arrive between pointer-down and pointer-up.
    // Keep the clicked element attached until a color pick actually changes it.
    if (renderedColor !== color()) {
      renderedColor = color();
      renderSwatches(swatches, palette, renderedColor, (i) => { picked = i; colors.set(i); renderRail(); }, (tag) => doc.createElement(tag));
    }
    for (const [m, b] of Object.entries(modeButtons)) {
      b.classList.toggle('on', state.mode === m);
      b.setAttribute('aria-pressed', String(state.mode === m));
    }
    const off = unavailable();
    const reason = !feedbackReady(v.model) ? 'Page feedback is getting ready.' : policy?.phase === 'disabled' ? (availabilityState === 'broken' ? 'Cast broken. Wait for the operator to end the cast.' : availabilityState === 'retry' ? 'Annotation is off. Waiting for the operator to retry.' : 'Annotation is off.') : policy?.driver ? `${policy.driver} is driving.` : 'The cast is paused.';
    rail.classList.toggle('paused', off);
    for (const b of [...Object.values(modeButtons), ...swatches.querySelectorAll('button')]) b.disabled = off;
    if (v.tools) v.tools.groups.pointerColors.hidden = state.mode === 'watch' || off;
    for (const [m, b] of Object.entries(modeButtons)) b.title = off ? reason : MODE_TIPS[m];
    hint.textContent = '';
    notesPanel.setStatus('control', off ? reason : '');
    dialog.hold(off ? reason : ''); // an open note, or a pin opened while paused
  };
  const placeRail = () => {
    const r = box.getBoundingClientRect();
    const at = railAt(r.width ? r : stage.getBoundingClientRect(), win.innerWidth);
    rail.style.top = `${at.top}px`;
    rail.style.right = `${at.right}px`;
    rail.hidden = !!v.tools || !annotate || !(v.model && v.model.seg > 0 && !v.model.ended);
    for (const button of Object.values(modeButtons)) button.hidden = !annotate || !(v.model && v.model.seg > 0 && !v.model.ended);
  };
  function setMode(m) { if (v.beforeAnnotationMode?.(m) === false) return; transition({ type: 'mode', mode: m }); }

  // Own strokes: drawn at once, posted every CHUNK_MS.
  const flush = () => {
    const s = state.stroke;
    if (!s) return;
    for (const pts of s.chunks.take()) {
      answered(post('stroke', { viewer: who.viewer(), seg: s.seg, id: s.id, color: s.color, pts }));
    }
  };
  function endStroke(discard = false) {
    if (state.timer !== null) win.clearInterval(state.timer);
    state.timer = null;
    if (!discard) flush();
    state.stroke = null;
  }
  // endGesture ends whatever the pointer is doing: a stroke or a region
  // drag. Every boundary calls it: a mode change, a new document, the cast's
  // end, a cancelled pointer.
  function endGesture(discard = false) {
    endStroke(discard);
    if (state.pointer !== null && layer.hasPointerCapture?.(state.pointer)) layer.releasePointerCapture(state.pointer);
    state.pointer = null;
    if (state.drag) { state.drag = null; drawDrag(); }
  }
  const pagePoint = (e) => v.toPage(e.clientX, e.clientY);

  layer.addEventListener('pointerdown', (e) => {
    if (e.target.closest && e.target.closest('.chip')) return; // a pin's chip opens its note, in any mode
    if (state.mode === 'watch' || !(v.model && v.model.seg > 0) || unavailable()) return;
    if (dialog.open) {
      if (!captures || state.mode !== 'note') return;
      dialog.close(); // a new box retires the old capture before drawing begins
    }
    e.preventDefault();
    layer.setPointerCapture?.(e.pointerId);
    state.pointer = e.pointerId;
    const p = pagePoint(e);
    if (state.mode === 'point') {
      endStroke();
      const id = randomID(win.crypto);
      state.stroke = { id, full: `${who.id()}.${id}`, color: color(), seg: v.model.seg, chunks: chunker() };
      state.stroke.chunks.add(p[0], p[1], win.performance.now());
      strokes.local(state.stroke.full, palette[color()], [p], win.performance.now());
      state.timer = win.setInterval(flush, CHUNK_MS);
      schedule();
    } else {
      state.drag = noteGesture(p, e.pointerType);
      drawDrag();
    }
  });
  layer.addEventListener('pointermove', (e) => {
    const p = pagePoint(e);
    if (state.stroke) {
      state.stroke.chunks.add(p[0], p[1], win.performance.now());
      strokes.local(state.stroke.full, palette[state.stroke.color], [p], win.performance.now());
      schedule();
    } else if (state.drag) {
      state.drag.move(p);
      drawDrag();
    }
  });
  const up = async () => {
    if (state.stroke) endStroke();
    if (!state.drag) return;
    const { from, to, pointerType, dragged } = state.drag;
    state.drag = null;
    drawDrag();
    const region = regionOf(from, to, pageView(), pointerType, { dragged });
    if (captures) {
      const request = 'capture-' + randomID(win.crypto);
      openDraft({ request, region, capturePending: true });
      try {
        let payload;
        if (prepareDraft) payload = prepareDraft(region);
        else {
          const at = boxCentre(region), node = v.nodeAt(at[0], at[1]), anchorBox = node && v.boxOf(node);
          if (!node || !anchorBox) throw Error('This element is not recorded yet. Draw a new box.');
          payload = noteDraft(v.model.seg, node, anchorBox, region);
        }
        captures.start(payload, request);
      } catch (err) { dialog.projectDraft(request, {pending:false,blocked:true,message:err.message || 'Could not anchor this note. Draw a new box.'}); }
      return;
    }
    if (prepareDraft) {
      const document = docSeg, version = gestureVersion;
      try {
        const payload = await prepareDraft(region);
        if (payload && version === gestureVersion && document === docSeg && !unavailable() && state.mode === 'note') openDraft({ ...payload, region });
      } catch (err) { say(err.message || 'Could not anchor this note.'); }
      return;
    }
    const at = boxCentre(region);
    const node = v.nodeAt(at[0], at[1]);
    const b = node && v.boxOf(node);
    if (!node || !b) return;
    openDraft(noteDraft(v.model.seg, node, b, region));
  };
  layer.addEventListener('pointerup', up);
  layer.addEventListener('pointercancel', () => endGesture());

  function openDraft(payload) {
    name.value = who.name();
    name.hidden = !!who.name();
    author.textContent = '';
    notesPanel.showEditor(dlg);
    dialog.draft(payload);
    drawDrag();
  }

  // pageView is the part of the page shown, in page px.
  const pageView = () => {
    const r = box.getBoundingClientRect();
    const [x0, y0] = v.toPage(r.left, r.top), [x1, y1] = v.toPage(r.right, r.bottom);
    return { x: x0, y: y0, w: x1 - x0, h: y1 - y0 };
  };
  // Preview only an established drag. A click gets its default box on release.
  const dragBox = el('div', 'drag-region', layer);
  dragBox.hidden = true;
  function drawDrag() {
    const d = state.drag
      ? regionOf(state.drag.from, state.drag.to, pageView(), state.drag.pointerType, { preview: true, dragged: state.drag.dragged })
      : dialog.draftPayload?.region;
    dragBox.hidden = !d;
    if (d) place(dragBox, d);
  }

  // place puts an element over a page-px box, in the box element's client
  // px (fromPage gives the window's).
  const clientBox = (b, r) => {
    const [x0, y0] = v.fromPage(b[0], b[1]), [x1, y1] = v.fromPage(b[0] + b[2], b[1] + b[3]);
    return [x0 - r.left, y0 - r.top, x1 - x0, y1 - y0];
  };
  const place = (e, b) => {
    const [x, y, width, height] = clientBox(b, box.getBoundingClientRect());
    Object.assign(e.style, { left: `${x}px`, top: `${y}px`, width: `${width}px`, height: `${height}px` });
  };

  const openNote = (n, by) => {
    v.feedbackVisibility?.invalidate();
    notesPanel.clearHighlight();
    if (operator && !!policy.driver) return;
    notesPanel.showEditor(dlg);
    dlg.hidden = false;
    const opened = annotate && !unavailable() && n.owner && n.owner === (operator ? 'operator' : owner)
      ? dialog.edit(n) : dialog.view(n, { resolvable: annotate });
    if (opened) {
      author.textContent = by + (n.edited ? ' (edited)' : '');
      name.hidden = true;
    }
  };
  const pinned = pinLayer(doc, pins, openNote);
  const highlightNote = note => {
    if (!note) { glow.clear(); return; }
    const placed = v.placementOf?.(note, {originalOnly:true, withClip:true});
    if (!placed?.box) { glow.clear(); return; }
    glow.draw((g, rect) => {
      g.beginPath();g.rect(...clientBox(placed.clip,rect));g.clip();
      g.strokeStyle='#eab34b';g.shadowColor='#eab34b';g.shadowBlur=14;g.lineWidth=3;
      g.strokeRect(...clientBox(placed.box,rect));
    });
  };
  const revealNote = note => {
    const shown = v.reveal?.(note) === true;
    notesPanel.setStatus('editor', shown ? '' : 'Cannot show on this page: anchor unavailable.');
    if (shown) highlightNote(note);
  };
  function drawPins() {
    const visibilityToken=v.feedbackVisibility?.token();
    const notes = openNotes(v.model,operator?undefined:v.feedbackMask);
    const witnesses = new Map(notes.map(n => [n.id, v.placementOf?.(n,v.feedbackVisibility?{withWitness:true}:undefined) || null]));
    const placements = new Map(notes.map(n=>{const placed=witnesses.get(n.id);return [n.id,placed?.box||placed];}));
    const placementOf = n => placements.get(n.id);
    pinned.draw(notes, (id) => v.boxOf(id), place, { placementOf });
    notesPanel.setNotes(notes, {open: note => openNote(note, note.author?.name || ''), hover: highlightNote, reveal: revealNote, available: note => !!placements.get(note.id)});
    dialog.update({ notes });
    v.feedbackVisibility?.notesPaint(visibilityToken,(operator&&policy?.driver?[]:notes).map(n=>({id:n.id,revision:v.model.note_revisions?.[n.id],box:placements.get(n.id)})));
  }

  function drawStrokes(now) {
    paintStrokes(canvas, strokes.visible(now), (x, y) => v.fromPage(x, y), box.getBoundingClientRect(), win.devicePixelRatio || 1);
  }

  // One redraw per frame at most; two frames out, so the viewer has drawn
  // the record that asked for it before boxOf reads the page. Strokes keep
  // it going until they have faded.
  let queued = false;
  function schedule() {
    if (queued) return;
    queued = true;
    win.requestAnimationFrame(() => win.requestAnimationFrame(() => {
      queued = false;
      const now = win.performance.now();
      placeRail();
      glow.resize();
      drawPins();
      drawDrag();
      drawStrokes(now);
      if (strokes.size) schedule();
    }));
  }

  v.onRecord((r, model) => {
    const nextScope = feedbackScopeKey(model);
    if (nextScope !== loadScope) {
      loadScope = nextScope;
      gestureVersion++;endGesture(true);strokes.clear();dialog.close();
      notesPanel.clearHighlight();notesPanel.setNotes([]);glow.clear();
      pinned.draw([],()=>null,place,{placementOf:()=>null});
      drawStrokes(win.performance.now());
    }
    strokes.seen(r.t);
    if (model.docSeg && model.docSeg !== docSeg) {
      // A new document: a stroke doesn't span documents. (A resync's fresh
      // model passes through 0, which is no document; landing back on the
      // same one changes nothing.)
      const was = docSeg;
      docSeg = model.docSeg;
      if (was) { strokes.clear(); endGesture(true); }
    }
    if (r.k === 'annotation-clear') { strokes.clear(); gestureVersion++; endGesture(true); setMode('watch'); }
    if (r.k === 'stroke'&&feedbackReady(model)) strokes.record(r, win.performance.now(), `${who.id()}.`);
    // A pause or the end puts the pen down (setMode ends the gesture and
    // renders the rail, disabled while paused); a resume enables it again.
    if (r.k === 'stop') { setMode('watch'); if (r.why === 'end') dialog.close(); }
    sync();
    schedule();
  });
  v.onFeedback?.(schedule);
  v.onVisibility?.(schedule);
  win.addEventListener('resize', schedule);
  v.onPlacementChange?.(schedule);
  // Menu mode changes deliberately leave focus on the page. Its keyboard
  // events do not traverse our shadow root, so operator Escape is captured
  // at the window boundary. Off/unavailable annotation leaves page keys alone.
  (operator ? win : root).addEventListener('keydown', (e) => {
    if (e.key !== 'Escape' || (operator && (unavailable() || (state.mode === 'watch' && !dialog.open)))) return;
    transition({ type: 'escape' });
    e.stopPropagation();
    e.preventDefault?.();
  }, operator);
  sync();
  schedule();
  const dispose = () => { glow.dispose(); notesPanel.dispose(); };
  win.addEventListener('pagehide', dispose, {once:true});
  return { notes: notesPanel, controls: {pointer:modeButtons.point, note:modeButtons.note, swatches}, dispose, setMode, setAnnotationAllowed(allowed, reset = false, availability = 'off') { availabilityState = availability; if (reset) { gestureVersion++; endGesture(true); setMode('watch'); } annotationAllowed = allowed === true; sync(); }, captureBusy:()=>dialog.open||!!state.drag||!!state.stroke,dialog, strokes, who, annotate, get mode() { return state.mode; } };
}
