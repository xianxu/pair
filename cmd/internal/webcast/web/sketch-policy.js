// UI recovery and command scheduling only. The recording reducer decides
// history legality; transports execute these immutable effects and report facts.
const need = (ok, why) => {
  if (!ok) throw Error(`sketch policy: ${why}`);
};
const own = (value) => structuredClone(value);
const safe = (n) => Number.isSafeInteger(n) && n >= 0;
const kinds = new Set(['begin', 'stroke-begin', 'points', 'stroke-end', 'undo', 'redo', 'label']);

export function createSketchPolicy({ operationId, maxQueuedChunks = 16 } = {}) {
  need(
    Number.isSafeInteger(maxQueuedChunks) && maxQueuedChunks > 0 && maxQueuedChunks <= 16,
    'invalid queue bound',
  );
  let serial = 0,
    active = null,
    allowed = false,
    live = false,
    ended = false,
    epoch = null;
  const slots = new Map();
  const operations = new Map();
  const identifier = operationId || (() => `sketch-operation-${++serial}`);
  const current = () => slots.get(active);
  function phase(slot) {
    if (!slot) return 'off';
    if (slot.uncertain) return 'reconciling';
    return slot.phase;
  }
  function pump(slot, effects) {
    if (slot.inflight || slot.uncertain || slot.retired || !slot.queue.length) return;
    const next = slot.queue[0];
    if (next.kind !== 'label' && (!allowed || !live || slot.epoch !== epoch || slot.id !== active)) return;
    const command = {
      kind: next.kind,
      body: next.body,
      operation: String(identifier()),
      sequence: slot.sequence,
      epoch: slot.epoch,
      sketch: slot.id,
    };
    need(
      command.operation.length > 0 &&
        new TextEncoder().encode(command.operation).length <= 128 &&
        !operations.has(command.operation),
      'invalid operation identity',
    );
    slot.queue.shift();
    slot.inflight = command;
    operations.set(command.operation, slot);
    effects.push({ type: 'send', command: own(command) });
  }
  return {
    snapshot() {
      return {
        phase: phase(current()),
        active,
        allowed,
        live,
        ended,
        epoch,
        slots: Array.from(slots.values(), (s) => ({
          sketch: s.id,
          epoch: s.epoch,
          phase: phase(s),
          queued: s.queue.length,
          inflight: s.inflight ? own(s.inflight) : null,
          nextSequence: s.sequence,
          retired: s.retired,
        })),
        pendingLabels: Array.from(slots.values())
          .filter((s) => !s.retired && !s.labelled && ['interrupted', 'label-required'].includes(s.phase))
          .map((s) => s.id),
      };
    },
    dispatch(event) {
      const effects = [];
      let slot = event.sketch ? slots.get(event.sketch) : current();
      switch (event.type) {
        case 'sync': {
          need(
            typeof event.allowed === 'boolean' && typeof event.live === 'boolean' && safe(event.epoch),
            'invalid authority',
          );
          const changed = epoch !== null && event.epoch !== epoch;
          allowed = event.allowed;
          live = event.live;
          ended = event.ended === true;
          epoch = event.epoch;
          if (slot && !slot.retired) {
            if (changed || !allowed || ended) {
              slot.queue = slot.queue.filter((c) => c.kind === 'label' && !ended);
              slot.phase = 'interrupted';
              slot.sourceReady = false;
              effects.push(
                { type: 'stop-gesture', reason: ended ? 'ended' : 'authority' },
                { type: 'release-scene', sketch: slot.id },
              );
              if (ended) slot.retired = true;
              if (slot.inflight) slot.uncertain = true;
            } else if (!live) {
              slot.queue = slot.queue.filter((c) => c.kind === 'label');
              slot.phase = 'paused';
              effects.push({ type: 'stop-gesture', reason: 'paused' });
              if (slot.inflight) slot.uncertain = true;
            } else if (slot.phase === 'paused') slot.phase = slot.sourceReady ? 'drawing' : 'preparing';
            pump(slot, effects);
          }
          break;
        }
        case 'enter':
          slot = current();
          if (
            !allowed ||
            !live ||
            ended ||
            (slot &&
              !slot.retired &&
              slot.epoch === epoch &&
              ['preparing', 'drawing', 'label-required'].includes(slot.phase))
          ) {
            effects.push({ type: 'refused', reason: 'Drawing is unavailable.' });
            break;
          }
          need(
            typeof event.sketch === 'string' && event.sketch.length > 0 && !slots.has(event.sketch),
            'invalid new sketch',
          );
          if (slots.size >= 256) {
            effects.push({
              type: 'refused',
              reason: 'Finish an outstanding sketch before starting another.',
            });
            break;
          }
          slot = {
            id: event.sketch,
            epoch,
            phase: 'preparing',
            sourceReady: false,
            queue: [],
            inflight: null,
            uncertain: false,
            retired: false,
            labelled: false,
            sequence: 0,
            open: false,
            tool: 'watch',
          };
          slots.set(slot.id, slot);
          active = slot.id;
          effects.push({ type: 'prepare-scene', sketch: slot.id, epoch });
          break;
        case 'prepared':
          if (!slot || slot.retired || slot.epoch !== epoch || !allowed || !live) break;
          if (event.nextSequence !== undefined) {
            need(safe(event.nextSequence), 'invalid sequence');
            slot.sequence = event.nextSequence;
          }
          slot.sourceReady = true;
          slot.phase = 'drawing';
          break;
        case 'leave':
          if (slot && !slot.retired && slot.phase === 'drawing') {
            slot.phase = 'label-required';
            slot.tool = event.tool || 'watch';
            effects.push(
              { type: 'stop-gesture', reason: 'leave' },
              { type: 'request-label', sketch: slot.id },
            );
          }
          break;
        case 'escape':
          if (slot?.phase === 'label-required' && allowed && live && slot.epoch === epoch)
            slot.phase = 'drawing';
          break;
        case 'queue': {
          if (!slot || slot.retired || ended || !kinds.has(event.kind)) {
            effects.push({ type: 'refused', reason: 'Sketch is unavailable.' });
            break;
          }
          if (
            event.kind !== 'label' &&
            (!allowed ||
              !live ||
              slot.epoch !== epoch ||
              slot.id !== active ||
              slot.uncertain ||
              (event.kind === 'begin' ? slot.phase !== 'preparing' : slot.phase !== 'drawing'))
          ) {
            effects.push({ type: 'refused', reason: 'Drawing is unavailable.' });
            break;
          }
          if (event.kind === 'label') {
            const label = event.body?.label;
            if (typeof label !== 'string' || !label.trim() || new TextEncoder().encode(label).length > 1024) {
              effects.push({ type: 'refused', reason: 'Add a nonblank label of at most 1024 bytes.' });
              break;
            }
          }
          if (['undo', 'redo'].includes(event.kind) && (slot.open || slot.inflight || slot.queue.length)) {
            effects.push({ type: 'refused', reason: 'Wait for the stroke confirmation.' });
            break;
          }
          if (
            event.kind === 'points' &&
            slot.queue.filter((c) => c.kind === 'points').length >= maxQueuedChunks
          ) {
            effects.push({ type: 'stop-gesture', reason: 'queue-full' });
            break;
          }
          slot.queue.push({ kind: event.kind, body: own(event.body) });
          pump(slot, effects);
          break;
        }
        case 'reply': {
          slot = operations.get(event.operation);
          if (!slot || (event.epoch !== undefined && event.epoch !== slot.inflight.epoch)) break;
          if (event.outcome !== 'accepted' && !(event.outcome === 'refused' && event.final === true)) {
            slot.uncertain = true;
            effects.push({ type: 'confirmation-pending', sketch: slot.id });
            break;
          }
          const command = slot.inflight;
          if (event.outcome === 'accepted' && event.next_sequence !== undefined) {
            need(event.next_sequence === slot.sequence + 1, 'reply sequence differs');
          }
          if (event.outcome === 'accepted' && command.kind === 'begin' && event.sketch !== undefined) {
            need(
              typeof event.sketch === 'string' &&
                event.sketch.length > 0 &&
                (event.sketch === slot.id || !slots.has(event.sketch)),
              'conflicting accepted sketch',
            );
            const old = slot.id;
            slots.delete(old);
            slot.id = event.sketch;
            slots.set(slot.id, slot);
            if (active === old) active = slot.id;
          }
          slot.inflight = null;
          operations.delete(event.operation);
          slot.uncertain = false;
          if (event.outcome === 'accepted') {
            slot.sequence++;
            if (command.kind === 'begin')
              effects.push({ type: 'begin-accepted', sketch: slot.id, reply: own(event) });
            if (command.kind === 'stroke-begin') slot.open = true;
            if (command.kind === 'stroke-end') slot.open = false;
            if (command.kind === 'label') {
              slot.labelled = true;
              slot.retired = true;
              slot.phase = 'off';
              slot.queue = [];
              effects.push({ type: 'release-scene', sketch: slot.id });
              if (slot.id === active) {
                effects.push({ type: 'select-tool', tool: slot.tool });
                active = null;
              }
              slots.delete(slot.id);
            }
          } else {
            slot.queue = [];
            slot.phase = 'interrupted';
            effects.push({ type: 'stop-gesture', reason: 'refused', error: event.error || '' });
            if (command.kind === 'begin') {
              slot.retired = true;
              effects.push({ type: 'release-scene', sketch: slot.id });
              slots.delete(slot.id);
              if (active === slot.id) active = null;
            }
          }
          pump(slot, effects);
          break;
        }
        case 'retry':
          if (slot?.inflight) {
            slot.uncertain = true;
            const command = slot.inflight;
            const lookup = { operation: command.operation, sequence: command.sequence, epoch: command.epoch };
            if (command.kind !== 'begin') lookup.sketch = command.sketch;
            effects.push({ type: 'status', command: own(command), lookup });
          }
          break;
        case 'clear':
        case 'end':
          if (event.type === 'end') ended = true;
          for (const s of slots.values()) {
            s.retired = true;
            s.queue = [];
            s.phase = 'off';
            effects.push({ type: 'release-scene', sketch: s.id });
          }
          effects.push({ type: 'stop-gesture', reason: event.type });
          break;
      }
      return effects;
    },
  };
}

// A hit-test may cross the private CDP scene boundary. Keep its first gesture
// bounded and ordered until the anchor arrives; no DOM or promise lives here.
export function createSketchGesture({ maxPoints = 1024 } = {}) {
  need(Number.isSafeInteger(maxPoints) && maxPoints > 0 && maxPoints <= 1024, 'invalid gesture bound');
  let gesture = null;
  let visits = 0;
  const valid = (p) =>
    Array.isArray(p) &&
    p.length === 3 &&
    p.every(Number.isFinite) &&
    Math.abs(p[0]) <= 1e7 &&
    Math.abs(p[1]) <= 1e7 &&
    p[2] >= 0 &&
    p[2] <= 1;
  function points(input) {
    need(Array.isArray(input) && input.every(valid), 'invalid samples');
    return input.map((p) => p.map((n) => (n === 0 ? 0 : n)));
  }
  function drain() {
    const effects = [];
    while (gesture.buffer.length) {
      const chunk = gesture.buffer.splice(0, 64);
      effects.push({ type: 'points', stroke: gesture.id, chunk: gesture.chunk++, points: chunk });
    }
    if (gesture.ended) {
      effects.push({ type: 'stroke-end', stroke: gesture.id, truncated: gesture.truncated });
      gesture = null;
    }
    return effects;
  }
  return {
    start(stroke, input) {
      need(!gesture && typeof stroke === 'string' && stroke.length > 0, 'gesture already open');
      const pts = points(input);
      need(pts.length > 0 && pts.length <= maxPoints, 'gesture bound');
      visits += pts.length;
      gesture = { id: stroke, buffer: pts, anchor: null, chunk: 0, ended: false, truncated: false };
      return { type: 'hit-test', stroke, point: [...pts[0]] };
    },
    samples(input) {
      if (!gesture || gesture.ended) return [];
      const pts = points(input);
      if (pts.length > maxPoints - gesture.buffer.length) {
        gesture.ended = true;
        gesture.truncated = true;
        return [{ type: 'stop-gesture', reason: 'buffer-full' }];
      }
      visits += pts.length;
      gesture.buffer.push(...pts);
      return gesture.anchor ? drain() : [];
    },
    anchor(at, stroke = gesture?.id) {
      if (!gesture || stroke !== gesture.id || gesture.anchor) return [];
      need(
        at &&
          safe(at.node) &&
          at.node > 0 &&
          Array.isArray(at.box) &&
          at.box.length === 4 &&
          at.box.every(Number.isFinite) &&
          at.box[2] > 0 &&
          at.box[3] > 0,
        'invalid anchor',
      );
      gesture.anchor = own(at);
      return [{ type: 'stroke-begin', stroke: gesture.id, anchor: own(at) }, ...drain()];
    },
    end({ truncated = false } = {}) {
      if (!gesture) return [];
      gesture.ended = true;
      gesture.truncated ||= truncated;
      return gesture.anchor ? drain() : [];
    },
    cancel() {
      gesture = null;
    },
    snapshot() {
      return {
        active: gesture?.id || null,
        buffered: gesture?.buffer.length || 0,
        waitingAnchor: !!gesture && !gesture.anchor,
        visits,
      };
    },
  };
}
