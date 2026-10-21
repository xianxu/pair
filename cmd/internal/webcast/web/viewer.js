// The web cast's live viewer (#460): the shell around the page frame. It
// follows the cast's event stream, feeds each record to the model (the
// recording's meaning) and the renderer (the sandboxed frame), lays the
// page out once at the operator's viewport and scales it to fit (never
// relayout, Principle 2), and draws the operator's pointer. The shell is
// the only script here: input from viewers (pointing, notes, #466's remote
// driving) is taken by its overlay, never by the recorded page. Title and
// URL come from whatever app is cast, so they are only ever set as text.

import { Model } from './model.js';
import { Renderer } from './render.js';
import { mountAnnotate, identityStore } from './annotate.js';
import { mountPlaceholders } from './placeholders.js';
import { joinSeat, mountDrive } from './drive.js';
import { replayPath, mountReplay } from './replay.js';
import { sessionLost } from './collab.js';

// fit is the page box's scale and place in an area: the operator's
// viewport, scaled to fit whole, centred.
export function fit(areaW, areaH, vp) {
  if (!vp || !(vp.w > 0) || !(vp.h > 0) || !(areaW > 0) || !(areaH > 0)) return { scale: 1, w: 0, h: 0, left: 0, top: 0 };
  const scale = Math.min(areaW / vp.w, areaH / vp.h);
  const w = vp.w * scale, h = vp.h * scale;
  return { scale, w, h, left: (areaW - w) / 2, top: (areaH - h) / 2 };
}

// toPage maps a point in the shell's client px to top-page CSS px: undo
// the box's place and scale, add the page's scroll.
export function toPage(clientX, clientY, box, scale, scroll) {
  return [(clientX - box.left) / scale + scroll[0], (clientY - box.top) / scale + scroll[1]];
}

// fromPage is toPage's inverse, relative to the box: where a top-page
// point shows on the scaled page.
export function fromPage(x, y, scale, scroll) {
  return [(x - scroll[0]) * scale, (y - scroll[1]) * scale];
}

// view maps what is known of the cast to what the bar shows.
export function view(model, state, ended) {
  if (ended) return { status: 'Cast ended', live: false, title: '', url: '', page: false };
  if (model && model.seg > 0) {
    return { status: model.live ? 'Live' : 'Paused', live: model.live, title: model.title || model.url || '', url: model.url || '', page: true };
  }
  if (!state) return { status: 'Connecting…', live: false, title: '', url: '', page: false };
  if (!state.live) return { status: 'Not live', live: false, title: '', url: '', page: false };
  return { status: 'Live', live: true, title: state.title || state.url || '', url: state.url || '', page: false };
}

// connect follows the cast's event stream. deps:
//   open()            an EventSource on the events route
//   renderer(model)   the renderer drawing that model ({apply, flush})
//   show(view)        renders the bar
//   frame()           called after records are drawn (once per frame)
//   schedule(fn)      runs fn before the next paint
//   retry(fn)         runs fn after a back-off
// It returns the viewer's surface: the current model and renderer, and
// onRecord(fn) for a later component to watch records (#467's control
// state) without touching the apply path.
export function connect(deps) {
  const hooks = [], capabilityHooks = [];
  const closed = (previous = {}) => Object.fromEntries(['annotation', 'drive'].map(k => [k, {...previous[k], allowed:false, state:'off'}]));
  const v = { model: null, renderer: null, capabilities: closed(), onRecord: (fn) => hooks.push(fn),
    onCapabilities(fn) { capabilityHooks.push(fn); fn(v.capabilities); } };
  const publishCapabilities = (value) => { v.capabilities = value; for (const fn of capabilityHooks) fn(value); };
  const failClosed = () => publishCapabilities(closed(v.capabilities));
  const receiveCapabilities = (value) => {
    const next = {};
    for (const k of ['annotation', 'drive']) {
      const c = value?.[k], old = v.capabilities[k];
      if (!c || !Number.isSafeInteger(c.generation) || c.generation < 0 || !Number.isSafeInteger(c.attempt) || c.attempt < 0) {
        next[k] = {...old, allowed:false, state:'off'}; continue;
      }
      if (c.generation < (old.generation || 0) || (c.generation === old.generation && c.attempt < old.attempt)) { next[k] = old; continue; }
      next[k] = {...c, allowed:c.supported === true && c.permitted === true && c.allowed === true && c.state === 'on'};
    }
    publishCapabilities(next);
  };
  let es, state = null, ended = false, queued = false;
  const fresh = () => {
    if (v.renderer && v.renderer.revokeAll) v.renderer.revokeAll(); // the old drawing's blob: stylesheets
    v.model = new Model({allowSketchSnapshot:true});
    v.renderer = deps.renderer(v.model);
    v.renderer.onDirty = paint;
  };
  const show = () => deps.show(view(v.model, state, ended));
  const paint = () => {
    if (queued) return;
    queued = true;
    deps.schedule(() => { queued = false; v.renderer.flush(); deps.frame(); });
  };
  // resync drops the page and reconnects for a fresh catch-up: a record
  // the model refuses means this viewer missed something.
  const resync = (why) => {
    console.warn('couch viewer: resyncing:', why);
    es.close();
    es = null;
    failClosed();
    fresh();
    deps.show({ ...view(null, state, false), status: 'Reconnecting…' });
    deps.retry(open);
  };
  function open() {
    const source = es = deps.open();
    const current = () => es === source;
    source.addEventListener('capabilities', (e) => {
      if (!current() || ended) return;
      try { receiveCapabilities(JSON.parse(e.data)); } catch { failClosed(); }
    });
    source.addEventListener('state', (e) => { if (!current()) return; state = JSON.parse(e.data); show(); });
    source.addEventListener('rec', (e) => {
      if (!current()) return;
      const r = JSON.parse(e.data);
      if (r.k === 'head' && v.model.head) fresh(); // a reconnect's catch-up starts over
      try { v.model.apply(r); } catch (err) { resync(err); return; }
      v.renderer.apply(r);
      if (r.k === 'stop' && r.why === 'end') { ended = true; failClosed(); }
      if (r.k === 'seg') ended = false;
      for (const fn of hooks) fn(r, v.model);
      show();
      paint();
    });
    source.addEventListener('end', () => { if (!current()) return; source.close(); es = null; ended = true; failClosed(); show(); });
    source.onerror = () => {
      if (!current()) return;
      failClosed();
      if (source.readyState !== 2) { show(); return; } // EventSource reconnects by itself
      if (ended) return;
      const disconnected = () => deps.show({ ...view(null, null, true), status: 'Disconnected' });
      // A stream refused because the link's code was rotated (pair#495)
      // returns to the lock page.
      if (!deps.sessionLost) { disconnected(); return; }
      deps.sessionLost().then((lost) => { if (current() && !ended) lost ? deps.relock() : disconnected(); });
    };
  }
  fresh();
  open();
  return v;
}

// mount wires the viewer into the page: the bar, the scaled frame, the
// pointer, and the viewer contract on window.couchViewer.
export function mount(doc, win) {
  const $ = (id) => doc.getElementById(id);
  const box = $('box'), stage = $('stage'), pointer = $('pointer');
  // #page is the frame with the current document; the renderer swaps it
  // with a sibling holding the previous paint (#475), so look it up.
  const frame = () => $('page');
  let placed = { scale: 1, left: 0, top: 0 };
  const scroll = () => { const w = frame().contentWindow; return w ? [w.scrollX, w.scrollY] : [0, 0]; };

  const layout = () => {
    const m = v && v.model, vp = m && m.seg > 0 ? m.vp : null;
    box.hidden = !vp;
    if (!vp) return;
    placed = fit(stage.clientWidth, stage.clientHeight, vp);
    const f = frame();
    f.style.width = `${vp.w}px`;
    f.style.height = `${vp.h}px`;
    f.style.transform = `scale(${placed.scale})`;
    Object.assign(box.style, { width: `${placed.w}px`, height: `${placed.h}px`, left: `${placed.left}px`, top: `${placed.top}px` });
  };
  const drawPointer = () => {
    const p = v && v.model && v.model.live && v.model.ptr;
    pointer.hidden = !p;
    if (!p) return;
    const [x, y] = fromPage(p.x, p.y, placed.scale, scroll());
    pointer.style.transform = `translate(${x}px, ${y}px)`;
  };
  const render = (s) => {
    const status = $('status');
    status.textContent = s.status;
    status.className = s.live ? 'status live' : 'status';
    $('title').textContent = s.title;
    $('url').textContent = s.url;
    $('message').hidden = s.page;
    doc.body.classList.toggle('paused', s.page && !s.live);
    doc.title = s.title ? `${s.title} · Couch web cast` : 'Couch web cast';
  };
  // resolveAsset: the asset proxy's URL for an absolute URL, or ''.
  const resolveAsset = (abs) => {
    const a = v.model.assets.get(abs);
    return a ? new URL(`assets/${a.hash}`, win.location.href).href : '';
  };
  const shell = {
    renderer: (model) => new Renderer(frame(), model, resolveAsset),
    show: render,
    frame: () => { layout(); drawPointer(); win.couchViewer?.placeholders?.draw(); },
    schedule: (fn) => win.requestAnimationFrame(fn),
  };
  // A recording (#471) plays from its records; a cast follows its stream,
  // as the seat this viewer takes for remote driving (#467) when the cast
  // offers one: the stream holds the seat. A ?viewer= names the stream
  // without a seat (tests).
  let v = null, seat = null;
  let started;
  if (replayPath(win.location.pathname)) {
    v = mountReplay(doc, win, shell, (x, y) => fromPage(x, y, placed.scale, scroll()));
    started = Promise.resolve();
  } else {
    let storage = null;
    try { storage = win.localStorage; } catch { /* none */ }
    started = joinSeat((url, init) => win.fetch(url, init), identityStore(storage, win.crypto).name()).then((s) => {
      seat = s;
      const viewer = seat ? seat.viewer : new URLSearchParams(win.location.search).get('viewer');
      v = connect({
        ...shell,
        open: () => new win.EventSource(viewer ? `events?viewer=${encodeURIComponent(viewer)}` : 'events'),
        sessionLost: () => sessionLost((url, init) => win.fetch(url, init)),
        relock: () => win.location.reload(),
        retry: (fn) => win.setTimeout(fn, 1000),
      });
    });
  }
  win.addEventListener('resize', () => { layout(); drawPointer(); });

  // The viewer contract (atlas/webcast-format.md): the conformance check,
  // wave 2's notes and #466's remote driving use it.
  win.couchViewer = {
    resolveAsset: (abs) => resolveAsset(abs) || undefined,
    canonical: () => v.renderer.canonical(),
    ready: async () => { await started; await v.loaded; return v.renderer.ready(); }, // a replay first loads its records
    boxOf: (id) => v.renderer.boxOf(id),
    placementOf: (note) => v.renderer.placementOf(note),
    onPlacementChange: (fn) => { started.then(() => { v.renderer.placementChanged = fn; }); },
    // uncaptured: the drawn elements whose content the recording can't
    // hold (pair#513), as [{id, kind}].
    uncaptured: () => v.renderer.uncaptured(),
    nodeAt: (x, y) => v.renderer.nodeAt(x, y),
    toPage: (clientX, clientY) => {
      const r = box.getBoundingClientRect();
      return toPage(clientX, clientY, r, placed.scale, scroll());
    },
    // fromPage is toPage's inverse: where a top-page point shows in the
    // window (#470's pins and strokes).
    fromPage: (x, y) => {
      const r = box.getBoundingClientRect(), [px, py] = fromPage(x, y, placed.scale, scroll());
      return [r.left + px, r.top + py];
    },
    // onRecord(fn): fn(r, model) for every record applied, from the stream
    // once it opens (registered before its first record arrives).
    onRecord: (fn) => { started.then(() => v.onRecord(fn)); },
    onCapabilities: (fn) => { started.then(() => v.onCapabilities?.(fn)); },
  };
  const surface = { ...win.couchViewer, get model() { return v.model; }, get seat() { return seat; }, get annotate() { return win.couchViewer.annotate; } };
  started.then(() => {
    win.couchViewer.placeholders = mountPlaceholders(doc, win, win.couchViewer);
    // Pointing and notes (#470), when the cast takes them; remote driving
    // (#467), when it offers a seat.
    mountAnnotate(doc, win, surface).then((a) => { win.couchViewer.annotate = a; });
    win.couchViewer.drive = mountDrive(doc, win, surface, seat);
  });
  return started.then(() => v);
}

if (typeof document !== 'undefined' && document.getElementById('page')) mount(document, window);
