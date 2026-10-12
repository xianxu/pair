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
  const hooks = [];
  const v = { model: null, renderer: null, onRecord: (fn) => hooks.push(fn) };
  let es, state = null, ended = false, queued = false;
  const fresh = () => { v.model = new Model(); v.renderer = deps.renderer(v.model); v.renderer.onDirty = paint; };
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
    fresh();
    deps.show({ ...view(null, state, false), status: 'Reconnecting…' });
    deps.retry(open);
  };
  function open() {
    es = deps.open();
    es.addEventListener('state', (e) => { state = JSON.parse(e.data); show(); });
    es.addEventListener('rec', (e) => {
      const r = JSON.parse(e.data);
      if (r.k === 'head' && v.model.head) fresh(); // a reconnect's catch-up starts over
      try { v.model.apply(r); } catch (err) { resync(err); return; }
      v.renderer.apply(r);
      if (r.k === 'stop' && r.why === 'end') ended = true;
      if (r.k === 'seg') ended = false;
      for (const fn of hooks) fn(r, v.model);
      show();
      paint();
    });
    es.addEventListener('end', () => { es.close(); ended = true; show(); });
    es.onerror = () => {
      if (es.readyState === 2) { if (!ended) deps.show({ ...view(null, null, true), status: 'Disconnected' }); }
      else show(); // EventSource reconnects by itself
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
  const frame = $('page'), box = $('box'), stage = $('stage'), pointer = $('pointer');
  let placed = { scale: 1, left: 0, top: 0 };
  const scroll = () => { const w = frame.contentWindow; return w ? [w.scrollX, w.scrollY] : [0, 0]; };

  const layout = () => {
    const m = v.model, vp = m && m.seg > 0 ? m.vp : null;
    box.hidden = !vp;
    if (!vp) return;
    placed = fit(stage.clientWidth, stage.clientHeight, vp);
    frame.style.width = `${vp.w}px`;
    frame.style.height = `${vp.h}px`;
    frame.style.transform = `scale(${placed.scale})`;
    Object.assign(box.style, { width: `${placed.w}px`, height: `${placed.h}px`, left: `${placed.left}px`, top: `${placed.top}px` });
  };
  const drawPointer = () => {
    const p = v.model && v.model.live && v.model.ptr;
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
  const viewer = new URLSearchParams(win.location.search).get('viewer');
  const v = connect({
    open: () => new win.EventSource(viewer ? `events?viewer=${encodeURIComponent(viewer)}` : 'events'),
    renderer: (model) => new Renderer(frame, model, (abs) => {
      const a = model.assets.get(abs);
      return a ? new URL(`assets/${a.hash}`, win.location.href).href : '';
    }),
    show: render,
    frame: () => { layout(); drawPointer(); },
    schedule: (fn) => win.requestAnimationFrame(fn),
    retry: (fn) => win.setTimeout(fn, 1000),
  });
  win.addEventListener('resize', () => { layout(); drawPointer(); });

  // The viewer contract (atlas/webcast-format.md): the conformance check,
  // wave 2's notes and #466's remote driving use it.
  win.couchViewer = {
    resolveAsset: (abs) => { const a = v.model.assets.get(abs); return a ? new URL(`assets/${a.hash}`, win.location.href).href : undefined; },
    canonical: () => v.renderer.canonical(),
    ready: () => v.renderer.ready(),
    boxOf: (id) => v.renderer.boxOf(id),
    nodeAt: (x, y) => v.renderer.nodeAt(x, y),
    toPage: (clientX, clientY) => {
      const r = box.getBoundingClientRect();
      return toPage(clientX, clientY, r, placed.scale, scroll());
    },
    onRecord: v.onRecord,
  };
  return v;
}

if (typeof document !== 'undefined' && document.getElementById('page')) mount(document, window);
