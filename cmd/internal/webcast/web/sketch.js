import { createSketchPolicy, createSketchGesture } from './sketch-policy.js';
import { createSketchRecordDisplay, paintSketchPaths, createSketchPaintScheduler } from './sketch-render.js';

// The controller owns input and drafts. The injected scene owns frozen DOM;
// send/status are the only mutation/reconciliation boundary.
export function mountSketch(doc, win, view, options) {
  const { root, box = root, scene } = options;
  const id = options.id || (() => globalThis.crypto.randomUUID());
  const policy = createSketchPolicy({ operationId: id });
  const gesture = createSketchGesture();
  let displayModel = view.model;
  let recordDisplay = options.paths ? null : createSketchRecordDisplay(displayModel);
  const requests = new Set(),
    requestObservers = new Map(),
    placements = new Map();
  const pending = new Set(),
    statuses = new Map(),
    confirmed = new Map(),
    strokeIdentity = new Map(),
    observed = new Map(),
    needsStatus = new Set(),
    drafts = new Map(),
    optimistic = new Map(),
    recoveries = new Map(),
    ink = new Map(),
    seenInk = new Set();
  let forceInk = true,
    inkTransform = '';
  const pathKey = (p) => JSON.stringify([p.sketch, p.id]);
  let disposed = false,
    generation = 0,
    timer = null,
    contact = null,
    samples = [],
    labelTarget = null,
    color = 0;
  let sceneSketch = null,
    message = '',
    paintGeneration = 0,
    entering = false,
    deferredExit = null,
    finishDeadline = null,
    recoveryEnd = null,
    drawingMode = false;
  const listeners = [];
  const make = (tag, cls, parent = root) => {
    const e = doc.createElement(tag);
    e.className = cls;
    parent.appendChild(e);
    return e;
  };
  const toolbar = make('div', 'sketch-toolbar', options.toolbar || root);
  const button = (text, action) => {
    const b = make('button', '', toolbar);
    b.textContent = text;
    b.type = 'button';
    b.addEventListener('click', action);
    return b;
  };
  const layer = make('canvas', 'sketch-layer', box),
    observer = make('canvas', 'sketch-observer', box);
  const panel = make('div', 'sketch-label', root),
    labelInput = make('textarea', 'sketch-label-input', panel);
  labelInput.setAttribute('aria-label', 'Required sketch label');
  labelInput.maxLength = 1024;
  const save = make('button', '', panel);
  save.textContent = 'Save label';
  save.addEventListener('click', () => saveLabel());
  const back = make('button', '', panel);
  back.textContent = 'Back to drawing';
  back.addEventListener('click', escape);
  const notice = make('div', 'sketch-status', root);
  notice.setAttribute('role', 'status');
  const labels = make('div', 'sketch-pending-labels', root);
  const sketchButton = button('Sketch', () => enter());
  const colors = make('select', 'sketch-color', toolbar);
  for (const [i, c] of (options.palette || ['#ff0000']).entries()) {
    const o = make('option', '', colors);
    o.value = String(i);
    o.textContent = c;
  }
  colors.addEventListener('change', () => {
    color = Number(colors.value) || 0;
  });
  const undoButton = button('Undo', () => undo()),
    redoButton = button('Redo', () => redo());
  const noteButton = button('Note', () => leave('watch'));
  noteButton.title = 'Open text for this unfinished sketch';
  const doneButton = button('Done', () => leave('watch'));
  const retryButton = button('Check confirmation', () => dispatch({ type: 'retry' }));
  const continueButton = button('Continue drawing', () => continueDrawing());
  const state = () => policy.snapshot();
  const active = () => state().active;
  const track = (p) => {
    pending.add(p);
    p.finally(() => pending.delete(p));
    return p;
  };
  async function idle() {
    while (pending.size) await Promise.all([...pending]);
  }
  function unavailableReason(s) {
    return s.ended ? 'This cast has ended. Start a new cast to sketch.'
      : !s.live ? 'Resume the cast to sketch.'
      : !s.allowed ? options.unavailableHint?.() || 'Ask the operator to enable annotation and grant you driving control to sketch.'
      : '';
  }
  function show() {
    const s = state(),
      drawing = s.phase === 'drawing';
    layer.style.pointerEvents = drawing ? 'auto' : 'none';
    layer.style.touchAction = 'none';
    observer.style.pointerEvents = 'none';
    panel.hidden = !labelTarget;
    toolbar.hidden =
      !options.showUnavailable && !s.allowed && !['preparing', 'drawing', 'finishing', 'label-required', 'reconciling', 'paused'].includes(s.phase);
    colors.disabled = ['preparing', 'drawing', 'finishing', 'label-required', 'reconciling', 'paused'].includes(s.phase);
    retryButton.hidden = s.phase !== 'reconciling';
    continueButton.hidden = !canContinue();
    sketchButton.disabled = !s.allowed || !s.live || s.ended;
    const unavailable = unavailableReason(s);
    sketchButton.title = unavailable;
    doneButton.disabled = !entering && !s.active;
    noteButton.hidden = !s.active;
    noteButton.disabled = s.phase === 'finishing';
    const status = statuses.get(s.active),
      busy = !!contact || !!gesture.snapshot().active || s.slots.some((x) => x.sketch === s.active && (x.inflight || x.queued));
    undoButton.disabled = !drawing || busy || !status?.visible_top;
    redoButton.disabled = !drawing || busy || !status?.redo_top;
    const noticeText =
      message ||
      (s.slots.some((x) => x.finishing) || deferredExit !== null
        ? 'Saving…'
        : s.phase === 'reconciling' ? 'Confirmation pending. Check before continuing.'
        : !s.active && !toolbar.hidden ? unavailable : '');
    if (notice.textContent !== noticeText) notice.textContent = noticeText;
    if (finishDeadline && !s.slots.some((x) => x.finishing) && !entering) {
      win.clearTimeout(finishDeadline.timer);
      finishDeadline.resolve();
      finishDeadline = null;
    }
    if (labels.children.length !== s.pendingLabels.length || s.pendingLabels.some((key,i)=>labels.children[i].dataset.sketch!==key)) {
      labels.replaceChildren();
      for (const key of s.pendingLabels) {
        const b = make('button', '', labels);
        b.dataset.sketch = key;
        b.textContent = 'Note';
        b.title = 'Open text for this unfinished sketch';
        b.addEventListener('click', () => openLabel(key));
      }
    }
    scheduler.invalidate();
  }
  function dispatch(event) {
    if (disposed) return;
    const effects = policy.dispatch(event);
    for (const e of effects) effect(e);
    pruneSketchCaches();
    show();
  }
  function reply(command, result) {
    if (disposed) return;
    if (result?.outcome === 'refused' && result.status?.phase === 'drawing' && command.kind !== 'begin')
      recoveries.set(command.sketch, { ...result, epoch: command.epoch });
    const key = result?.sketch || command.sketch;
    if (result?.status) statuses.set(key, structuredClone(result.status));
    if (result?.outcome === 'accepted') confirmed.set(key, {
      sketch: key, operation: command.operation, sequence: command.sequence, epoch: command.epoch,
    });
    if (result?.outcome === 'accepted' && command.kind === 'label') message = '';
    if (result?.outcome === 'accepted' && command.kind === 'stroke-end') optimistic.delete(command.body.id);
    dispatch({
      type: 'reply',
      ...result,
      operation: command.operation,
      epoch: command.epoch,
      final: result?.outcome === 'refused',
    });
    reconcileSketches();
    if (
      result?.outcome === 'accepted' &&
      recoveryEnd &&
      ['drawing','finishing'].includes(state().phase) &&
      active() === recoveryEnd.sketch
    ) {
      const terminal = recoveryEnd;
      recoveryEnd = null;
      queue('stroke-end', { id: terminal.id, truncated: true }, terminal.sketch);
    }
  }
  function setDrawingMode(value) {
    if (drawingMode === value) return;
    drawingMode = value;
    options.onMode?.(value);
  }
  // Bound the UI observer, never the writer's ownership of a started command.
  // Late transport completion cannot mutate UI after this observer timed out.
  function observeRequest(call) {
    let timeout;
    const expired = new Promise((_, reject) => {
      timeout = win.setTimeout(() => reject(Error('Confirmation pending.')), 5000);
      requestObservers.set(timeout, reject);
    });
    return Promise.race([Promise.resolve().then(call), expired]).finally(() => {
      win.clearTimeout(timeout);
      requestObservers.delete(timeout);
    });
  }
  function effect(e) {
    if (e.type === 'prepare-scene') {
      sceneSketch = e.sketch;
      setDrawingMode(true);
    } else if (e.type === 'send' || e.type === 'status') {
      const command = e.command;
      if (requests.has(command.operation)) return;
      requests.add(command.operation);
      const body = {
        ...command.body,
        operation: command.operation,
        sequence: command.sequence,
        epoch: command.epoch,
      };
      if (command.kind !== 'begin') body.sketch = command.sketch;
      track(
        Promise.resolve()
          .then(async () => {
            let result =
              e.type === 'status'
                ? await observeRequest(() => options.status(e.lookup))
                : await observeRequest(() => options.send('sketch-' + command.kind, body));
            if (e.type === 'status' && result?.outcome === 'not-observed' && !disposed)
              result = await observeRequest(() => options.send('sketch-' + command.kind, body));
            return result;
          })
          .then(
            (r) => {
              requests.delete(command.operation);
              reply(command, r);
            },
            () => {
              requests.delete(command.operation);
              reply(command, { outcome: 'unknown' });
            },
          ),
      );
    } else if (e.type === 'begin-accepted') {
      const current = state();
      if (
        current.active !== e.sketch ||
        current.epoch !== e.reply.epoch ||
        !current.allowed ||
        !current.live ||
        current.phase === 'interrupted'
      )
        return;
      sceneSketch = e.sketch;
      if (e.reply.status?.phase === 'interrupted' || !e.reply.source) {
        message = 'Drawing ended before the source was ready. Add a label to preserve its intent.';
        setAuthority({ ...state(), allowed: false });
        return;
      }
      placements.clear();
      forceInk = true;
      const stamp = ++generation;
      sceneSketch = e.sketch;
      track(
        Promise.resolve()
          .then(() => scene.prepare(e.reply.source, { sketch: e.sketch, epoch: state().epoch }))
          .then(
            () => {
              if (disposed || stamp !== generation) return;
              setDrawingMode(true);
              dispatch({ type: 'prepared', sketch: e.sketch });
            },
            (error) => {
              if (disposed || stamp !== generation) return;
              message = 'The held scene is unavailable. Try Sketch again.';
              setAuthority({ ...state(), allowed: false });
            },
          ),
      );
    } else if (e.type === 'confirmation-pending') {
      const owner = gesture.snapshot();
      if (contact || owner.waitingAnchor) {
        const stroke = contact?.stroke || owner.active;
        recoveryEnd = owner.waitingAnchor ? null : { sketch: active(), id: stroke };
        if (owner.waitingAnchor) {
          message = "Couldn't save this stroke; accepted strokes kept. Check confirmation.";
          optimistic.delete(stroke);
          generation++;
        }
        if (timer !== null) win.clearTimeout(timer);
        timer = null;
        samples = [];
        gesture.cancel();
        const local = optimistic.get(stroke);
        if (local) local.ended = true;
        if (contact && layer.hasPointerCapture?.(contact.pointer)) layer.releasePointerCapture(contact.pointer);
        contact = null;
        if (owner.waitingAnchor)
          dispatch({ type: 'gesture-finished', sketch: active(), stroke });
      }
    } else if (e.type === 'stop-gesture') {
      if (finishDeadline && ['authority','paused','ended','clear','end'].includes(e.reason))
        message = "Couldn't save; accepted strokes kept.";
      if (e.reason === 'queue-full')
        message = 'Stroke queue full. Confirming and sealing its accepted points.';
      if (e.reason === 'refused')
        message = 'Drawing stopped. Label this sketch or use Continue drawing when it is available.';
      if (e.reason === 'leave') finish(true);
      else cancel();
    } else if (e.type === 'release-scene') {
      if (sceneSketch === e.sketch) {
        generation++;
        placements.clear();
        forceInk = true;
        scene.dispose();
        sceneSketch = null;
        setDrawingMode(false);
      }
      const slot = state().slots.find((s) => s.sketch === e.sketch);
      if (labelTarget === e.sketch && (!slot || slot.retired)) {
        labelTarget = null;
        drafts.delete(e.sketch);
      }
    } else if (e.type === 'request-label') openLabel(e.sketch);
    else if (e.type === 'select-tool') options.onTool?.(e.tool);
    else if (e.type === 'refused') message = e.reason;
  }
  function queue(kind, body, sketch = active()) {
    dispatch({ type: 'queue', kind, body, sketch });
  }
  function gestureEffects(effects) {
    for (const e of effects) {
      if (e.type === 'stroke-begin') {
        const local = optimistic.get(e.stroke);
        if (local) local.at = e.anchor;
        queue('stroke-begin', { id: e.stroke, node: e.anchor.node, box: e.anchor.box });
      } else if (e.type === 'points') queue('points', { id: e.stroke, chunk: e.chunk, pts: e.points });
      else if (e.type === 'stroke-end') {
        const local = optimistic.get(e.stroke);
        if (local) local.ended = true;
        queue('stroke-end', { id: e.stroke, ...(e.truncated ? { truncated: true } : {}) });
      } else if (e.type === 'stop-gesture') {
        message = 'Stroke buffer full. Finish this stroke before continuing.';
        finish(true);
      }
    }
  }
  function flush() {
    if (timer !== null) {
      win.clearTimeout(timer);
      timer = null;
    }
    const batch = samples;
    samples = [];
    if (batch.length) gestureEffects(gesture.samples(batch));
  }
  function cancel() {
    recoveryEnd = null;
    generation++;
    if (timer !== null) win.clearTimeout(timer);
    timer = null;
    samples = [];
    gesture.cancel();
    optimistic.clear();
    if (contact && layer.hasPointerCapture?.(contact.pointer)) layer.releasePointerCapture(contact.pointer);
    contact = null;
  }
  function finish(truncated = false, event) {
    if (!contact) return;
    if (event && contact.count < 4096 && samples.length < 1024) {
      const last = point(event, contact.pressure);
      if (last[0] !== contact.last[0] || last[1] !== contact.last[1]) {
        samples.push(last);
        optimistic.get(contact.stroke)?.pts.push(last);
        contact.count++;
      }
    }
    const finished = contact;
    contact = null;
    try {
      flush();
      gestureEffects(gesture.end({ truncated }));
      if (!gesture.snapshot().active) dispatch({ type: 'gesture-finished', sketch: active(), stroke: finished.stroke });
    } finally {
      if (layer.hasPointerCapture?.(finished.pointer)) layer.releasePointerCapture(finished.pointer);
    }
    show();
  }
  function point(e, pressure) {
    const p = scene.toPage(e.clientX, e.clientY);
    return [p[0], p[1], pressure];
  }
  function consumePointer(e) {
    e.preventDefault?.();
    e.stopPropagation?.();
  }
  function down(e) {
    consumePointer(e);
    if (state().phase !== 'drawing' || contact || e.isPrimary === false || e.button !== 0 || !e.buttons)
      return;
    // Physical release does not release the accumulator's logical ownership.
    // A refused successor never allocates a contact, capture, ID or hit-test.
    if (gesture.snapshot().active) {
      const waiting = 'Finishing the previous stroke. Wait before drawing again.';
      if (message !== waiting) {
        message = waiting;
        show();
      }
      return;
    }
    let stroke, first, hit, candidate;
    let ownsCandidate = false;
    try {
      const pressure = e.pointerType === 'pen' ? Math.max(0, Math.min(1, e.pressure)) : 1;
      first = point(e, pressure);
      if (!first.every(Number.isFinite) || first.some((n) => Math.abs(n) > 1e7))
        throw Error('Invalid contact coordinates.');
      stroke = id();
      candidate = {
        pointer: e.pointerId,
        pressure,
        count: 1,
        stroke,
        last: first.slice(0, 2),
      };
      hit = gesture.start(stroke, [first]);
      ownsCandidate = true;
      layer.setPointerCapture?.(e.pointerId);
    } catch {
      if (ownsCandidate) gesture.cancel();
      if (layer.hasPointerCapture?.(e.pointerId)) layer.releasePointerCapture(e.pointerId);
      message = 'Could not start this stroke. Try again.';
      show();
      return;
    }
    contact = candidate;
    message = '';
    const sketch = active(),
      stamp = generation;
    optimistic.set(stroke, {
      sketch: active(),
      id: stroke,
      width: 3,
      color: (options.palette || ['#ff0000'])[color],
      pts: [first],
      ended: false,
      at: null,
    });
    scheduler.invalidate();
    track(
      Promise.resolve()
        .then(() => observeRequest(() => scene.hitTest(hit.point.slice(0, 2))))
        .then((at) => {
          if (!disposed && stamp === generation && gesture.snapshot().active === stroke) {
            gestureEffects(gesture.anchor(at, stroke));
            if (!gesture.snapshot().active) {
              if (message.startsWith('Finishing the previous stroke.')) message = '';
              dispatch({ type: 'gesture-finished', sketch, stroke });
            }
          }
        })
        .catch(() => {
          if (stamp === generation && gesture.snapshot().active === stroke) {
            if (!message.startsWith("Couldn't save")) message = 'This stroke has no usable anchor; accepted strokes kept.';
            cancel();
            dispatch({ type: 'gesture-finished', sketch, stroke });
            show();
          }
        }),
    );
  }
  function move(e) {
    consumePointer(e);
    if (!contact || contact.pointer !== e.pointerId || !e.buttons || state().phase !== 'drawing') return;
    e.preventDefault();
    const coalesced = e.getCoalescedEvents?.();
    const events = coalesced?.length ? coalesced : [e];
    for (const sample of events) {
      if (contact.count >= 4096 || samples.length >= 1024) {
        finish(true);
        return;
      }
      const pressure = e.pointerType === 'pen' ? Math.max(0, Math.min(1, sample.pressure)) : 1;
      contact.pressure = pressure;
      contact.count++;
      const p = point(sample, pressure);
      contact.last = p.slice(0, 2);
      samples.push(p);
      optimistic.get(contact.stroke)?.pts.push(p);
    }
    scheduler.invalidate();
    if (timer === null) timer = win.setTimeout(flush, 50);
  }
  function on(el, type, fn, capture = false) {
    el.addEventListener(type, fn, capture);
    listeners.push(() => el.removeEventListener?.(type, fn, capture));
  }
  on(layer, 'pointerdown', down);
  on(layer, 'pointermove', move);
  on(layer, 'pointerup', (e) => {
    consumePointer(e);
    if (contact?.pointer === e.pointerId) finish(false, e);
  });
  on(layer, 'pointercancel', (e) => {
    consumePointer(e);
    if (contact?.pointer === e.pointerId) finish(true);
  });
  on(layer, 'lostpointercapture', (e) => {
    consumePointer(e);
    if (contact?.pointer === e.pointerId) finish(true);
  });
  on(
    win,
    'keydown',
    (e) => {
      if (e.key !== 'Escape' || (!labelTarget && !['drawing','preparing','finishing','reconciling'].includes(state().phase) && !entering)) return;
      e.preventDefault();
      e.stopImmediatePropagation?.();
      if (labelTarget) escape();
      else leave('watch');
    },
    true,
  );
  on(labelInput, 'keydown', (e) => {
    if (e.isComposing || e.keyCode === 229) return;
    if (e.key === 'Enter') {
      e.preventDefault();
      e.stopPropagation();
      saveLabel();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      escape();
    }
  });
  function escape() {
    dispatch({ type: 'escape' });
    if (state().phase === 'drawing') {
      message = '';
      labelTarget = null;
      show();
    }
  }
  async function enter() {
    if (disposed || entering) return;
    dispatch({ type: 'sync', ...options.authority() });
    if (!state().allowed || !state().live || state().ended) return;
    entering = true;
    const stamp = generation;
    try {
      await scene.preflight?.();
      if (disposed || generation !== stamp) return;
      const before = active(),
        key = id();
      dispatch({ type: 'enter', sketch: key });
      if (active() === before) return;
      if (!finishDeadline?.failed) message = '';
      queue('begin', { seg: view.model.seg, color }, key);
      if (deferredExit) {
        dispatch({ type: 'leave', tool: deferredExit });
        deferredExit = null;
      }
      await idle();
    } catch (error) {
      message = 'The held scene is unavailable. Try Sketch again.';
      show();
    } finally {
      entering = false;
      deferredExit = null;
      show();
    }
  }
  function finishingDeadline() {
    if (finishDeadline) return finishDeadline.promise;
    const deadline = {};
    deadline.promise = new Promise((resolve) => { deadline.resolve = resolve; });
    deadline.timer = win.setTimeout(() => {
      if (disposed || finishDeadline !== deadline) return;
      deadline.failed = true;
      message = "Couldn't save; accepted strokes kept." +
        (state().slots.some((slot) => slot.inflight) ? ' Check confirmation.' : '');
      if (gesture.snapshot().waitingAnchor) {
        const stroke = gesture.snapshot().active,
          sketch = active();
        cancel();
        dispatch({ type: 'gesture-finished', sketch, stroke });
      }
      deadline.resolve();
      show();
    }, 5000);
    finishDeadline = deadline;
    return deadline.promise;
  }
  async function leave(tool = 'watch') {
    if (entering && !active()) deferredExit ||= tool;
    const deadline = finishingDeadline();
    if (!finishDeadline?.failed) message = '';
    dispatch({
      type: 'leave',
      tool,
      stroke: gesture.snapshot().active || recoveryEnd?.id,
      waitingGesture: !!gesture.snapshot().active,
    });
    finish(true);
    await Promise.race([idle(), deadline]);
  }
  function beforeAnnotationMode(mode) {
    if (!entering && !['preparing', 'drawing', 'finishing', 'reconciling', 'label-required'].includes(state().phase))
      return true;
    leave(mode);
    return false;
  }
  function openLabel(key) {
    if (disposed || state().slots.find((s) => s.sketch === key)?.finishing) return;
    const changed = labelTarget !== key;
    if (changed) labelInput.value = drafts.get(key) || '';
    labelTarget = key;
    show();
    if (changed) labelInput.focus();
  }
  async function saveLabel() {
    if (!labelTarget) return;
    const key = labelTarget, label = labelInput.value;
    reconcileSketches();
    let slot = state().slots.find((s) => s.sketch === key);
    if (slot?.finishing) {
      show();
      return;
    }
    if (needsStatus.has(key) && !slot?.inflight && !slot?.queued) {
      const lookup = confirmed.get(key);
      if (!lookup) return;
      const revision = observed.get(key);
      try {
        const result = await observeRequest(() => options.status(lookup));
        if (disposed || result?.outcome !== 'accepted' || observed.get(key) !== revision) return;
        if (result.status) statuses.set(key, structuredClone(result.status));
        dispatch({ type: 'observed', sketch: key, next_sequence: result.next_sequence });
        needsStatus.delete(key);
        reconcileSketches();
      } catch {
        if (disposed) return;
        message = 'Confirmation pending. Check before continuing.';
        show();
        return;
      }
      slot = state().slots.find((s) => s.sketch === key);
    }
    if (disposed || labelTarget !== key || !slot || slot.retired || labelInput.value !== label) return;
    if (slot.inflight || slot.queued || statuses.get(key)?.open_stroke) {
      message = 'Wait for the stroke confirmation before saving its label.';
      show();
      return;
    }
    drafts.set(key, label);
    queue('label', { label }, key);
    await idle();
  }
  async function undo() {
    if (gesture.snapshot().active) return;
    const top = statuses.get(active())?.visible_top;
    if (top) queue('undo', { id: top });
    await idle();
  }
  async function redo() {
    if (gesture.snapshot().active) return;
    const top = statuses.get(active())?.redo_top;
    if (top) queue('redo', { id: top });
    await idle();
  }
  function canContinue() {
    const s = state(),
      slot = s.slots.find((x) => x.sketch === s.active),
      reply = recoveries.get(s.active);
    return !!(
      slot &&
      reply &&
      s.phase === 'interrupted' &&
      s.allowed &&
      s.live &&
      !s.ended &&
      scene.ready &&
      reply.epoch === s.epoch &&
      slot.epoch === s.epoch &&
      reply.next_sequence === slot.nextSequence &&
      reply.status?.phase === 'drawing'
    );
  }
  async function continueDrawing() {
    if (!canContinue()) return;
    const sketch = active(),
      reply = recoveries.get(sketch);
    recoveries.delete(sketch);
    message = '';
    labelTarget = null;
    dispatch({
      type: 'recover',
      sketch,
      epoch: reply.epoch,
      phase: reply.status.phase,
      next_sequence: reply.next_sequence,
      open_stroke: reply.status.open_stroke || '',
    });
    await idle();
  }
  function setAuthority(authority) {
    const previous = state();
    if (previous.allowed === authority.allowed && previous.live === authority.live &&
        previous.ended === (authority.ended === true) && previous.epoch === authority.epoch &&
        sketchButton.title === unavailableReason(authority)) return;
    dispatch({ type: 'sync', ...authority });
  }
  async function paint() {
    const stamp = ++paintGeneration,
      sceneGeneration = generation;
    const paths = options.paths ? options.paths() : recordDisplay.paths();
    const liveKeys = new Set();
    const targets = await Promise.all(
      paths.map((p) => {
        const key = String(scene.ready) + pathKey(p);
        liveKeys.add(key);
        if (!placements.has(key))
          placements.set(
            key,
            Promise.resolve()
              .then(() => (scene.ready ? scene.placement(p) : view.placementOf(p)))
              .catch(() => null),
          );
        return placements.get(key);
      }),
    );
    if (disposed || stamp !== paintGeneration || sceneGeneration !== generation) return;
    for (const key of placements.keys()) if (!liveKeys.has(key)) placements.delete(key);
    const rect = box.getBoundingClientRect(),
      dpr = win.devicePixelRatio || 1;
    const width = Math.max(1, Math.round(rect.width * dpr)),
      height = Math.max(1, Math.round(rect.height * dpr));
    for (const canvas of [observer, layer]) {
      if (canvas.width !== width) {
        canvas.width = width;
        forceInk = true;
      }
      if (canvas.height !== height) {
        canvas.height = height;
        forceInk = true;
      }
    }
    const g = observer.getContext('2d'),
      localInk = layer.getContext('2d');
    if (!g || !localInk) return;
    g.setTransform(dpr, 0, 0, dpr, 0, 0);
    localInk.setTransform(dpr, 0, 0, dpr, 0, 0);
    localInk.clearRect(0, 0, rect.width, rect.height);
    const transform = scene.ready ? scene : view;
    const origin = transform.fromPage(0, 0),
      unit = transform.fromPage(1, 0);
    const scale = Math.hypot(unit[0] - origin[0], unit[1] - origin[1]);
    const transformKey = JSON.stringify([
      rect.width,
      rect.height,
      dpr,
      scale,
      origin[0] - rect.left,
      origin[1] - rect.top,
    ]);
    const keys = new Set(paths.map(pathKey));
    let repaint = forceInk || inkTransform !== transformKey;
    for (const key of ink.keys()) if (!keys.has(key)) repaint = true;
    for (let i = 0; i < paths.length; i++) {
      const p = paths[i],
        key = pathKey(p),
        previous = ink.get(key),
        target = targets[i];
      if (
        (!previous && seenInk.has(key)) ||
        (previous &&
          (previous.points !== p.pts ||
            previous.length > p.pts.length ||
            previous.width !== p.width ||
            previous.color !== p.color ||
            JSON.stringify(previous.box) !== JSON.stringify(target)))
      )
        repaint = true;
    }
    if (repaint) {
      g.clearRect(0, 0, rect.width, rect.height);
      ink.clear();
    }
    const toClient = (x, y) => {
      const xy = transform.fromPage(x, y);
      return [xy[0] - rect.left, xy[1] - rect.top];
    };
    for (let i = 0; i < paths.length; i++) {
      const p = paths[i],
        key = pathKey(p),
        previous = ink.get(key),
        start = previous?.length || 0;
      if (start < p.pts.length)
        paintSketchPaths(g, [{ ...p, pts: p.pts.slice(Math.max(0, start - 1)) }], {
          project: () => targets[i],
          toClient,
          scale,
        });
      ink.set(key, { points: p.pts, length: p.pts.length, box: targets[i], width: p.width, color: p.color });
      seenInk.add(key);
    }
    forceInk = false;
    inkTransform = transformKey;
    // Reconcile by accepted point prefix. Unconfirmed samples remain visibly
    // translucent, including the bounded contact waiting for async hit-test.
    const echoedPaths = new Map(paths.map((p) => [pathKey(p), p]));
    for (const [key, local] of optimistic) {
      const echoed = echoedPaths.get(pathKey(local));
      const count = echoed?.pts.length || 0;
      if (count >= local.pts.length) {
        if (local.ended) optimistic.delete(key);
        continue;
      }
      const tail = { ...local, pts: local.pts.slice(Math.max(0, count - 1)) };
      localInk.globalAlpha = 0.45;
      paintSketchPaths(localInk, [tail], {
        toClient: (x, y) => {
          const xy = transform.fromPage(x, y);
          return [xy[0] - rect.left, xy[1] - rect.top];
        },
        scale: Math.hypot(unit[0] - origin[0], unit[1] - origin[1]),
      });
      localInk.globalAlpha = 1;
    }
  }
  const scheduler = createSketchPaintScheduler({
    schedule: (fn) => win.requestAnimationFrame(fn),
    paint: () => {
      paint().catch(() => {});
    },
  });
  function pruneSketchCaches() {
    const retained = new Set(state().slots.filter((s) => !s.retired).map((s) => s.sketch));
    for (const cache of [statuses, confirmed, observed, strokeIdentity, needsStatus, drafts, recoveries])
      for (const key of cache.keys()) if (!retained.has(key)) cache.delete(key);
  }
  // Compact journal facts and full viewer metadata share this read-only seam.
  // Neither source decides command legality; the authenticated status supplies
  // the sequence after contextual operations such as expiry.
  function reconcileSketches({clear = false, authoritative = false} = {}) {
    if (clear) dispatch({ type: 'clear' });
    for (const slot of state().slots) {
      if (slot.retired) continue;
      const old = statuses.get(slot.sketch) || {};
      if (old.open_stroke) strokeIdentity.set(slot.sketch, old.open_stroke);
      const stroke = strokeIdentity.get(slot.sketch);
      let info = options.sketchMetadata?.(slot.sketch, stroke) ||
        view.model.sketchMetadata?.(slot.sketch, stroke);
      if (!info && authoritative && observed.has(slot.sketch)) info = { phase: 'cleared' };
      if (!info || (!info.phase && !info.begin)) continue;
      const sealed = !!info.stroke?.sealed;
      const signature = JSON.stringify([info.phase, info.reason, sealed]);
      if (observed.get(slot.sketch) !== signature) {
        observed.set(slot.sketch, signature);
        if ((sealed || info.phase === 'interrupted') && info.next_sequence === undefined)
          needsStatus.add(slot.sketch);
        dispatch({ type: 'observed', sketch: slot.sketch, phase: info.phase, sealed,
          next_sequence: info.next_sequence });
      }
      if (!state().slots.some(s => s.sketch === slot.sketch && !s.retired)) continue;
      statuses.set(slot.sketch, { ...old, phase: info.phase, reason: info.reason, label: info.label,
        ...(sealed || ['interrupted', 'cleared', 'complete'].includes(info.phase) ? { open_stroke: '' } : {}) });
    }
  }
  const unsubscribe = view.onRecord?.((r) => {
    if (disposed) return;
    if ((!scene.ready && !r.k.startsWith('sketch-')) || r.k === 'sketch-snapshot') {
      placements.clear();
      forceInk = true;
    }
    if (recordDisplay && displayModel !== view.model) {
      displayModel = view.model;
      recordDisplay = createSketchRecordDisplay(displayModel);
    } else recordDisplay?.accept(r);
    reconcileSketches();
    if (view.model.ended || (r.k === 'stop' && r.why === 'end')) dispatch({ type: 'end' });
    else if (r.k === 'annotation-clear') dispatch({ type: 'clear' });
    else if (['ctl', 'seg', 'stop', 'sketch-interrupt'].includes(r.k) && active()) {
      const info = view.model.sketchMetadata?.(active()) || {};
      if (info.begin) statuses.set(active(), info);
      if (info.phase === 'interrupted') setAuthority({ ...state(), allowed: false });
      else if (r.k === 'stop') setAuthority({ ...state(), live: false });
    }
    scheduler.invalidate();
  });
  const unAuthority = options.onAuthority?.(setAuthority);
  dispatch({ type: 'sync', ...options.authority() });
  return {
    enter,
    leave,
    beforeAnnotationMode,
    saveLabel,
    undo,
    redo,
    openLabel,
    setAuthority,
    reconcileSketches,
    canContinue,
    continueDrawing,
    state,
    captureBusy: () => entering || drawingMode || !!labelTarget,
    layer,
    labelInput,
    retry: () => dispatch({ type: 'retry' }),
    refresh: ({ geometry = false } = {}) => {
      if (geometry) {
        placements.clear();
        forceInk = true;
      }
      scheduler.invalidate();
    },
    dispose() {
      disposed = true;
      for (const [timer, reject] of requestObservers) {
        win.clearTimeout(timer);
        reject(Error('Sketch context retired.'));
      }
      requestObservers.clear();
      generation++;
      if (finishDeadline) {
        win.clearTimeout(finishDeadline.timer);
        finishDeadline.resolve();
        finishDeadline = null;
      }
      paintGeneration++;
      cancel();
      scheduler.dispose();
      recordDisplay = null;
      displayModel = null;
      placements.clear();
      ink.clear();
      seenInk.clear();
      statuses.clear();
      confirmed.clear();
      observed.clear();
      strokeIdentity.clear();
      needsStatus.clear();
      drafts.clear();
      recoveries.clear();
      if (typeof unsubscribe === 'function') unsubscribe();
      unAuthority?.();
      scene.dispose();
      setDrawingMode(false);
      for (const remove of listeners) remove();
      for (const el of [toolbar, layer, observer, panel, notice, labels]) el.remove();
    },
  };
}
