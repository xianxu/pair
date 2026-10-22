import { createPlacement } from './web/annotation-placement.js';

// Trusted adapter inside couchcast, never the page's JavaScript world.
export function operatorTransport(report, win) {
  let serial = 0;
  let annotationGeneration = null;
  const waiting = new Map();
  let unsettled = null;
  let scope, generation = 0;
  const stale = 'The recording document changed. Start a new annotation.';
  const wire = (request) => report({ t: 'operator', request });
  const ask = (op, body, id = String(++serial), origin = annotationGeneration, timeoutMs = 5000) => {
    if (waiting.has(id)) return waiting.get(id).promise;
    let finish;
    const promise = new Promise((resolve) => { finish = resolve; });
    const resolve = finish;
    if (origin === null && op !== 'status' && op !== 'cancel-draft') { resolve({request:id,outcome:'refused',error:'Annotation permission is not ready.'}); return promise; }
    if (waiting.size >= 8) { resolve({ outcome: 'refused', error: 'Annotation is busy.' }); return promise; }
    const timer = win.setTimeout(() => {
      waiting.delete(id);
      resolve({ request: id, outcome: 'unknown', error: 'No confirmation yet. Retry checks the same operation.' });
    }, Math.min(5000, timeoutMs));
    waiting.set(id, { resolve, timer, promise });
    wire({ op, request: id, body, generation: origin });
    return promise;
  };
  return {
    setAnnotationGeneration(value) {
      const valid = typeof value === 'string' && /^(0|[1-9][0-9]{0,19})$/.test(value) && BigInt(value) <= 18446744073709551615n;
      annotationGeneration = valid ? value : null;
      return valid;
    },
    setScope(doc, ended = false) {
      const next = ended ? 0 : doc;
      if (next === scope) return;
      scope = next;
      generation++;
      // The recorder discards this document's operation ledger on retirement.
      // Forget reconciliation with it, without claiming the write was accepted.
      unsettled = null;
      for (const [id, pending] of waiting) {
        win.clearTimeout(pending.timer);
        pending.resolve({ request: id, outcome: 'refused', error: stale });
      }
      waiting.clear();
    },
    reply(reply) {
      const p = waiting.get(reply.request);
      if (!p) return;
      if (reply.outcome === 'pending' && !reply.state) return;
      waiting.delete(reply.request);
      win.clearTimeout(p.timer);
      p.resolve(reply);
    },
    ask,
    annotationGeneration: () => annotationGeneration,
    async mutate(op, body) {
      if (!unsettled) unsettled = { id: String(++serial), op, body, generation, origin: annotationGeneration };
      const current = unsettled;
      const same = current.op === op && JSON.stringify(current.body) === JSON.stringify(body);
      // Even an edited draft cannot create another note while the previous
      // write might have committed. Reconcile first; never retry under a new ID.
      const sent = current.sent;
      current.sent = true;
      const result = await ask(sent ? 'status' : current.op, sent ? {} : current.body, current.id, current.origin);
      if (current.generation !== generation) return { status: 409, text: stale };
      const settled = result.outcome === 'accepted' || result.outcome === 'refused';
      if (settled && unsettled === current) unsettled = null;
      if (!same && settled) return { status: 409, text: 'The previous annotation is settled. This change was not submitted; keep its text and save again or start a new note.' };
      return result.outcome === 'accepted' ? { status: 204 } : { status: 409, text: result.error || 'Confirmation pending. Retry checks the same operation.' };
    },
  };
}

export function mountOperator({ host, root, report }, { annotate, palette, css, sketch }, doc = document, win = window) {
  const style = doc.createElement('style');
  style.textContent = css + '\n.operator-box{position:fixed;inset:0;pointer-events:none}.note-dialog{pointer-events:auto}.annotate.operator-driving{visibility:hidden}.annotation-hint{max-width:120px;color:#222;background:#fff;border-radius:5px;font:12px/1.3 system-ui;text-align:right}.annotation-hint:empty{display:none}';
  root.appendChild(style);
  const box = doc.createElement('div'); box.className = 'operator-box'; root.appendChild(box);
  const transport = operatorTransport(report, win);
  const callbacks = new Set();
  const callback = (...args) => { for (const fn of callbacks) fn(...args); };
  let revision = -1, strokeSeq = 0;
  // Only trusted CDP object references establish original identity. Paths and
  // matching text remain replacement evidence, never an original-node map.
  // Recorder IDs increase across documents; backend IDs are never used here.
  const originals = new Map();
  let originalDocument = 0, visibleNodeIDs = new Set(), sketchMount = null;
  const wantsOriginal = (id) => visibleNodeIDs.has(id) || sketchMount?.receiver?.hasAnchor(originalDocument,id) === true;
  const view = {
    model: { seg: 0, docSeg: 0, notes: [] },
    onRecord(fn) { callbacks.add(fn); return () => callbacks.delete(fn); },
    toPage: (x, y) => [x + win.scrollX, y + win.scrollY],
    fromPage: (x, y) => [x - win.scrollX, y - win.scrollY],
    boxOf: () => null,
    placementOf: createPlacement({ document: () => doc, route: () => win.location?.href || '', original: id => originals.get(id)?.deref() || null, exclude: n => n === host, onChange: () => callback({k:'layout',t:0},view.model) }),
  };
  let picked = 0;
  const post = (route, body) => {
    if (route === 'note-edit') return transport.mutate('edit-note', { seg: body.seg, id: body.id, text: body.text });
    if (route === 'note') return transport.mutate('save-note', { draft: body.draft, color: body.color, text: body.text });
    if (route === 'note-state') return transport.mutate('resolve', { seg: view.model.seg, id: body.id, state: body.state });
    return transport.ask('stroke', { seg: body.seg, id: body.id, color: body.color, pts: body.pts }).then((r) => ({ status: r.outcome === 'accepted' ? 204 : 409, text: r.error || 'Stroke was not confirmed.' }));
  };
  const prepareDraft = (region) => {
    // Temporarily remove our entire host from hit-testing. The path helper is
    // the recorder observer's own grammar, including open shadow boundaries.
    const display = host.style.display;
    host.style.display = 'none';
    let node;
    try {
      const x = region[0] + region[2] / 2 - win.scrollX, y = region[1] + region[3] / 2 - win.scrollY;
      node = doc.elementFromPoint(x, y);
      while (node?.shadowRoot) {
        const inner = node.shadowRoot.elementFromPoint(x, y);
        if (!inner || inner === node) break;
        node = inner;
      }
    } finally { host.style.display = display; }
    const path = node && win.__couchcastPath?.(node);
    if (!path) throw Error('This element is not recorded yet.');
    const rect = node.getBoundingClientRect();
    return { seg: view.model.seg, path, box: [rect.x + win.scrollX, rect.y + win.scrollY, rect.width, rect.height], region };
  };
  const captureCall = async (op, body, { timeoutMs, context }) => {
    const { request, viewer, ...payload } = body;
    if (op === 'draft-status' || op === 'cancel-draft') payload.draft = body.draft || request;
    // Status and cancellation have distinct envelope IDs; their body identifies
    // the prepared capture. Save/prepare retain their idempotency request IDs.
    const reply = await transport.ask(op, payload, op === 'draft-status' || op === 'cancel-draft' ? undefined : request, context, timeoutMs);
    if (reply.outcome === 'refused' || reply.outcome === 'unknown') throw Error(reply.error || 'Capture was not confirmed. Retry capture.');
    return reply;
  };
  captureCall.context = () => transport.annotationGeneration();
  const ui = annotate.mountAnnotation(doc, win, view, {
    root, box, stage: box, palette, operator: true, annotationAllowed: false, post, prepareDraft, captureCall,
    who: { id: () => 'operator', name: () => 'Operator', viewer: () => ({ id: 'operator', name: 'Operator' }) },
    colors: { get: () => picked, set: (n) => { picked = n; } },
  });
  if (sketch?.mount) {
    const mounted = sketchMount = sketch.mount({host,root,box,report,view,annotation:ui,palette,css:sketch.css,shellURL:sketch.shellURL},doc,win);
    ui.sketch = mounted.tool;
	win.__couchcastOperatorSketch = mounted.tool;
  }
  win.__couchcastCaptureBusy = () => ui.dialog.open || !!sketchMount?.captureBusy();
  // #523 supplies this capability input through CDP in the isolated world.
  // Keep it independent of view snapshots; no switch or authority is added here.
  win.__couchcastAnnotationAllowed = (allowed, generation) => {
    const known = transport.setAnnotationGeneration(generation);
    // Delivery revisions suppress duplicate calls. A new generation can arrive
    // as true after a coalesced Off/On; no old local stroke may return with it.
    ui.strokes.clear();
    ui.setAnnotationAllowed(allowed === true && known, true);
    callback({ k: 'view', t: 0 }, view.model); // repaint the cleared store
  };
  win.__couchcastOperatorReply = (reply) => transport.reply(reply);
  win.__couchcastOperatorMode = (mode) => { ui.setMode(mode); return ui.mode === mode; };
  win.__couchcastOperatorOriginal = (document, id, node) => {
    if (document <= 0 || document !== originalDocument || !wantsOriginal(id) || node?.nodeType !== 1 || node.ownerDocument !== doc || !node.isConnected || node === host) return false;
    const previous = originals.get(id);
    // A dead weak reference still reserves its identity until the note retires.
    if (previous) return previous.deref() === node;
    originals.set(id, new WeakRef(node));
    callback({ k: 'layout', t: 0 }, view.model);
    return true;
  };
  win.__couchcastOperatorView = (snapshot) => {
    if (snapshot.revision < revision) return;
    revision = snapshot.revision;
    if (snapshot.doc !== originalDocument) { originals.clear(); originalDocument = snapshot.doc; }
    visibleNodeIDs = new Set(snapshot.notes.map(note => note.at?.node));
    for (const id of originals.keys()) if (!wantsOriginal(id)) originals.delete(id);
    transport.setScope(snapshot.doc, snapshot.ended);
    view.model = { seg: snapshot.seg, docSeg: snapshot.doc, paused: snapshot.paused, live: !snapshot.paused && !snapshot.ended, ended: snapshot.ended, driver: snapshot.driver, notes: snapshot.notes.map((note) => ({ note, seg: snapshot.doc, state: 'open' })) };
    callback({ k: 'view', t: snapshot.t }, view.model);
    for (const stroke of snapshot.strokes) {
      if (stroke.seq <= strokeSeq) continue;
      strokeSeq = stroke.seq;
      callback({ ...stroke, k: 'stroke' }, view.model);
    }
  };
  win.addEventListener('scroll', () => callback({ k: 'scroll', t: 0 }, view.model), { passive: true });
  return ui;
}
