import { projectPoint, pressureWidth } from './annotation-anchor.js';

// Display data contains accepted facts, never receipts or Undo/Redo rules.
const need = (ok, why) => {
  if (!ok) throw Error(`sketch display: ${why}`);
};
const safe = (n) => Number.isSafeInteger(n) && n >= 0;
const finite = (n) => Number.isFinite(n) && Math.abs(n) <= 1e7;
const id = (s) =>
  typeof s === 'string' &&
  s.length > 0 &&
  new TextEncoder().encode(s).length <= 128 &&
  !/[\p{Cc}\p{Cf}\p{Cs}]/u.test(s);
const point = (p) => Array.isArray(p) && p.length === 3 && p.every(finite) && p[2] >= 0 && p[2] <= 1;
const copyPoints = (pts) => pts.map((p) => p.map((n) => (n === 0 ? 0 : n)));
function pick(value, keys) {
  const out = {};
  for (const key of keys) if (value[key] !== undefined) out[key] = structuredClone(value[key]);
  return out;
}
function sketchBegin(begin) {
  need(begin && id(begin.id) && begin.author && id(begin.author.id), 'invalid sketch header');
  return {
    ...pick(begin, ['id', 'doc', 'tab', 'vp', 't']),
    author: pick(begin.author, ['id', 'name', 'color']),
  };
}
function strokeBegin(begin) {
  need(
    begin &&
      id(begin.id) &&
      finite(begin.width) &&
      begin.width > 0 &&
      begin.at &&
      safe(begin.at.node) &&
      begin.at.node > 0 &&
      Array.isArray(begin.at.box) &&
      begin.at.box.length === 4 &&
      begin.at.box.every(finite) &&
      begin.at.box[2] > 0 &&
      begin.at.box[3] > 0,
    'invalid stroke header',
  );
  return {
    id: begin.id,
    width: begin.width,
    at: pick(begin.at, ['node', 'tag', 'kind', 'box', 'classes', 'path', 'sel', 'stable', 'route', 'text']),
  };
}
function metadata(value, current = { phase: 'drawing', reason: '', label: '' }) {
  const out = { ...current };
  for (const k of ['phase', 'reason', 'label'])
    if (k in value) {
      need(typeof value[k] === 'string', 'invalid sketch metadata');
      out[k] = value[k];
    }
  if ('next_sequence' in value || 'digest' in value) {
    const sequence=typeof value.next_sequence==='number'?String(value.next_sequence):value.next_sequence;
    need(typeof sequence==='string'&&/^[1-9][0-9]*$/.test(sequence)&&sequence.length<=16&&Number.isSafeInteger(Number(sequence))&&typeof value.digest==='string'&&/^[0-9a-f]{64}$/.test(value.digest),'invalid display version');
    out.next_sequence=sequence;out.digest=value.digest;
  }
  return out;
}
function flags(value, current = { visible: false, sealed: false, truncated: false }) {
  const out = { ...current };
  for (const k of ['visible', 'sealed', 'truncated'])
    if (k in value) {
      need(typeof value[k] === 'boolean', 'invalid stroke flag');
      out[k] = value[k];
    }
  return out;
}

export function displayFromSketchProjection(projection) {
  return {
    sketches: projection.sketches
      .filter((s) => s.phase !== 'cleared')
      .map((s) => ({
        begin: sketchBegin(s.begin),
        phase: s.phase,
        reason: s.reason,
        label: s.label,
        ...(s.next_sequence && s.receipts?.length ? {next_sequence:String(s.next_sequence),digest:s.receipts.at(-1).digest}:{}),
        strokes: s.strokes.map((p) => ({
          begin: strokeBegin(p.begin),
          pts: copyPoints(p.pts),
          visible: s.visible.includes(p.begin.id) || (!p.sealed && p.pts.length > 0 && s.phase === 'drawing'),
          sealed: p.sealed,
          truncated: p.truncated,
        })),
      })),
  };
}

export function createSketchDisplay() {
  let sketches = new Map(),
    anchors = new Map(), currentAnchors = new Map();
  const work = { points: 0, patches: 0, bootstraps: 0, anchorVisits: 0 };
  function anchor(index, currentIndex, document, node, delta) {
    if (!safe(document) || document === 0) return;
    let nodes = index.get(document);
    if (!nodes) {
      nodes = new Map();
      index.set(document, nodes);
    }
    const count = (nodes.get(node) || 0) + delta;
    if (count > 0) nodes.set(node, count);
    else nodes.delete(node);
    if (!nodes.size) index.delete(document);
    const currentCount = (currentIndex.get(node) || 0) + delta;
    if (currentCount > 0) currentIndex.set(node, currentCount);
    else currentIndex.delete(node);
  }
  const snapshot = () => ({
    sketches: Array.from(sketches.values(), (s) => ({
      ...structuredClone(s.header),
      strokes: Array.from(s.strokes.values(), (p) => structuredClone(p)),
    })),
  });
  function install(view) {
    need(view && Array.isArray(view.sketches) && view.sketches.length <= 256, 'invalid bootstrap');
    const next = new Map(),
      nextAnchors = new Map(), nextCurrentAnchors = new Map();
    let anchorVisits = 0;
    for (const s of view.sketches) {
      const begin = sketchBegin(s.begin);
      need(
        !next.has(begin.id) && Array.isArray(s.strokes) && s.strokes.length <= 64,
        'duplicate or excessive sketches/strokes',
      );
      const strokes = new Map(),
        anchorNodes = new Set();
      let count = 0;
      for (const p of s.strokes) {
        const b = strokeBegin(p.begin);
        need(
          !strokes.has(b.id) && Array.isArray(p.pts) && p.pts.length <= 4096 && p.pts.every(point),
          'invalid bootstrap path',
        );
        count += p.pts.length;
        anchorNodes.add(b.at.node);
        anchorVisits++;
        strokes.set(b.id, { begin: b, pts: copyPoints(p.pts), ...flags(p) });
      }
      need(count <= 16384, 'sketch point bound');
      next.set(begin.id, { header: { begin, ...metadata(s) }, strokes, points: count, anchorNodes });
      if (s.phase !== 'cleared') for (const node of anchorNodes) anchor(nextAnchors, nextCurrentAnchors, begin.doc, node, 1);
    }
    sketches = next;
    anchors = nextAnchors;
    currentAnchors = nextCurrentAnchors;
    work.anchorVisits += anchorVisits;
    work.bootstraps++;
  }
  function applyBatch(patches) {
    need(Array.isArray(patches) && patches.length <= 32, 'invalid batch');
    let clear = false;
    const changes = new Map();
    let appended = 0;
    function sketch(key, header) {
      if (changes.has(key)) return changes.get(key);
      const original = clear ? null : sketches.get(key);
      need(original || header?.begin, 'unknown sketch');
      const out = {
        original,
        header: original ? { ...original.header } : { begin: sketchBegin(header.begin), ...metadata({}) },
        strokes: new Map(),
      };
      changes.set(key, out);
      return out;
    }
    function stroke(s, key, begin) {
      if (s.strokes.has(key)) return s.strokes.get(key);
      const original = s.original?.strokes.get(key);
      need(original || begin, 'unknown stroke');
      const out = {
        original,
        begin: original?.begin || strokeBegin(begin),
        ...flags(original || {}),
        tail: [],
        length: original?.pts.length || 0,
      };
      s.strokes.set(key, out);
      return out;
    }
    for (const patch of patches) {
      need(patch && typeof patch === 'object', 'invalid patch');
      if (patch.clear) {
        need(
          patch.clear === true && !patch.sketch && !patch.header && !patch.stroke && !patch.points,
          'invalid clear',
        );
        changes.clear();
        clear = true;
        appended = 0;
        continue;
      }
      need(id(patch.sketch), 'invalid sketch identity');
      const s = sketch(patch.sketch, patch.header);
      if (patch.header) {
        if (patch.header.begin) {
          const begin = sketchBegin(patch.header.begin);
          need(
            begin.id === patch.sketch &&
              (!s.original || JSON.stringify(begin) === JSON.stringify(s.header.begin)),
            'conflicting sketch header',
          );
          s.header.begin = begin;
        }
        s.header = { ...s.header, ...metadata(patch.header, s.header) };
      }
      if (patch.stroke) {
        const key = patch.stroke.begin?.id || patch.stroke.id;
        need(id(key) && (!patch.stroke.id || patch.stroke.id === key), 'invalid stroke identity');
        const p = stroke(s, key, patch.stroke.begin);
        if (patch.stroke.begin)
          need(
            JSON.stringify(strokeBegin(patch.stroke.begin)) === JSON.stringify(p.begin),
            'conflicting stroke header',
          );
        Object.assign(p, flags(patch.stroke, p));
      }
      if (patch.points) {
        const input = patch.points;
        need(
          id(input.id) &&
            safe(input.offset) &&
            Array.isArray(input.pts) &&
            input.pts.length > 0 &&
            input.pts.length <= 64 &&
            input.pts.every(point),
          'invalid point patch',
        );
        const p = stroke(s, input.id);
        need(input.offset <= p.length, 'point gap');
        if (input.offset < p.length) {
          need(input.offset + input.pts.length <= p.length, 'overlapping points');
          const oldLength = p.original?.pts.length || 0;
          need(
            input.pts.every((pt, i) => {
              const index = input.offset + i,
                old = index < oldLength ? p.original.pts[index] : p.tail[index - oldLength];
              return pt.every((v, j) => v === old[j]);
            }),
            'conflicting points',
          );
        } else {
          need(input.pts.length <= 4096 - p.length, 'stroke point bound');
          const pts = copyPoints(input.pts);
          p.tail.push(...pts);
          p.length += pts.length;
          appended += pts.length;
        }
      }
    }
    // Validate counts using indexed lengths, never visiting old path points.
    for (const s of changes.values()) {
      let total = 0,
        count = s.original?.strokes.size || 0;
      for (const [key, p] of s.strokes) {
        if (!s.original?.strokes.has(key)) count++;
        total += p.length - (p.original?.pts.length || 0);
      }
      need(count <= 64, 'stroke count bound');
      const oldCount = s.original?.points || 0;
      need(oldCount + total <= 16384, 'sketch point bound');
      s.points = oldCount + total;
    }
    need(
      (clear ? 0 : sketches.size) + Array.from(changes.values()).filter((s) => !s.original).length <= 256,
      'sketch count bound',
    );
    if (clear) {
      sketches.clear();
      anchors.clear();
      currentAnchors.clear();
    }
    for (const [key, s] of changes) {
      const target = s.original || { header: s.header, strokes: new Map(), anchorNodes: new Set() };
      const wasActive = !!s.original && target.header.phase !== 'cleared';
      const active = s.header.phase !== 'cleared';
      if (wasActive && !active)
        for (const node of target.anchorNodes) {
          anchor(anchors, currentAnchors, target.header.begin.doc, node, -1);
          work.anchorVisits++;
        }
      target.header = s.header;
      target.points = s.points;
      for (const [sid, p] of s.strokes) {
        const st = p.original || { begin: p.begin, pts: [] };
        if (!p.original) {
          work.anchorVisits++;
          const node = p.begin.at.node;
          if (!target.anchorNodes.has(node)) {
            target.anchorNodes.add(node);
            if (wasActive && active) anchor(anchors, currentAnchors, target.header.begin.doc, node, 1);
          }
        }
        st.pts.push(...p.tail);
        for (const k of ['visible', 'sealed', 'truncated']) st[k] = p[k];
        target.strokes.set(sid, st);
      }
      if (!wasActive && active)
        for (const node of target.anchorNodes) anchor(anchors, currentAnchors, target.header.begin.doc, node, 1);
      sketches.set(key, target);
    }
    work.points += appended;
    work.patches += patches.length;
  }
  return {
    metadata(sketch, stroke) {
      const s = sketches.get(sketch);
      if (!s) return null;
      const p = s.strokes.get(stroke);
      return { phase: s.header.phase, reason: s.header.reason, label: s.header.label,
        ...(s.header.next_sequence!==undefined?{next_sequence:Number(s.header.next_sequence),digest:s.header.digest}:{}),
        ...(p ? { stroke: { sealed: p.sealed, truncated: p.truncated } } : {}) };
    },
    hasAnchor: (document, node) =>
      safe(document) && document > 0 && safe(node) && node > 0 && anchors.get(document)?.has(node) === true,
    hasCurrentAnchor: (node) => safe(node) && node > 0 && currentAnchors.has(node),
    install,
    apply: (patch) => applyBatch([patch]),
    applyBatch,
    snapshot,
    stats: () => ({ ...work }),
    paths({ borrow = false } = {}) {
      const out = [];
      for (const s of sketches.values()) {
        if (s.header.phase === 'cleared') continue;
        for (const p of s.strokes.values())
          if (p.pts.length && (p.visible || !p.sealed))
            out.push({
              sketch: s.header.begin.id,
              id: p.begin.id,
              at: borrow ? p.begin.at : structuredClone(p.begin.at),
              width: p.begin.width,
              color: s.header.begin.author.color,
              pts: borrow ? p.pts : copyPoints(p.pts),
              sealed: p.sealed,
              truncated: p.truncated,
              source: pick(s.header.begin, ['doc', 'tab', 'vp', 't']),
            });
      }
      return out;
    },
  };
}

export function projectSketchPoint(point, from, to) {
  return projectPoint(point, from, to);
}

// Variable-width ribbons and endpoint discs retain each sample's pressure.
// The viewer scale applies to widths, independently of anchor aspect ratio.
export function paintSketchPaths(g, paths, { toClient = (x, y) => [x, y], scale = 1, project } = {}) {
  if (!Number.isFinite(scale) || scale <= 0) return null;
  let bounds = null;
  g.save();
  for (const path of paths) {
    if (!Number.isFinite(path.width) || path.width <= 0) continue;
    const target = project ? project(path) : undefined;
    if (project && !target) continue;
    const pts = [];
    for (const raw of path.pts) {
      const p = project ? projectPoint(raw, path.at?.box, target) : raw;
      if (!p || !point(p)) {
        pts.length = 0;
        break;
      }
      const xy = toClient(p[0], p[1]);
      if (!xy?.every(Number.isFinite)) {
        pts.length = 0;
        break;
      }
      pts.push([xy[0], xy[1], (pressureWidth(path.width, p[2]) * scale) / 2]);
    }
    g.fillStyle = /^#[0-9a-fA-F]{6}$/.test(path.color || '') ? path.color : '#ff0000';
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1],
        b = pts[i],
        dx = b[0] - a[0],
        dy = b[1] - a[1],
        length = Math.hypot(dx, dy);
      if (!length) continue;
      const nx = -dy / length,
        ny = dx / length;
      g.beginPath();
      g.moveTo(a[0] + nx * a[2], a[1] + ny * a[2]);
      g.lineTo(b[0] + nx * b[2], b[1] + ny * b[2]);
      g.lineTo(b[0] - nx * b[2], b[1] - ny * b[2]);
      g.lineTo(a[0] - nx * a[2], a[1] - ny * a[2]);
      g.closePath();
      g.fill();
    }
    for (const [x, y, r] of pts) {
      if (!bounds) bounds = {left:x-r,top:y-r,right:x+r,bottom:y+r};
      else {
        bounds.left=Math.min(bounds.left,x-r);bounds.top=Math.min(bounds.top,y-r);
        bounds.right=Math.max(bounds.right,x+r);bounds.bottom=Math.max(bounds.bottom,y+r);
      }
      g.beginPath();
      g.arc(x, y, r, 0, Math.PI * 2);
      g.fill();
    }
  }
  g.restore();
  return bounds;
}

// Coalesce a burst of accepted patches into one projection walk and paint.
// The shell injects requestAnimationFrame; retirement suppresses late callbacks.
export function createSketchPaintScheduler({ schedule, paint }) {
  let pending = false,
    disposed = false;
  const work = { scheduled: 0, painted: 0 };
  return {
    invalidate() {
      if (pending || disposed) return;
      pending = true;
      work.scheduled++;
      schedule(() => {
        pending = false;
        if (disposed) return;
        work.painted++;
        paint();
      });
    },
    dispose() {
      disposed = true;
    },
    stats() {
      return { ...work };
    },
  };
}

// Feed only records already accepted by Model.apply. Legality, stacks and
// contextual seals come from the reducer's constant-size owned metadata.
export function createSketchRecordDisplay(model) {
  const display = createSketchDisplay();
  let active = '',
    open = '';
  function bootstrap() {
    const projection = model.sketchProjection();
    display.install(displayFromSketchProjection(projection));
    active = projection.active;
    open = model.sketchMetadata(active).open_stroke || '';
  }
  bootstrap();
  return {
    accept(r) {
      if (r.k === 'sketch-snapshot') {
        bootstrap();
        return;
      }
      if (r.k === 'annotation-clear') {
        display.apply({ clear: true });
        active = '';
        open = '';
        return;
      }
      const contextual = ['seg', 'stop', 'ctl'].includes(r.k);
      if (!r.k.startsWith('sketch-') && !contextual) return;
      const key = r.k === 'sketch-begin' ? r.id : r.sketch || active;
      const stroke = r.id && r.k !== 'sketch-begin' ? r.id : open;
      const info = model.sketchMetadata(key, stroke);
      if (!info.begin || info.phase === 'cleared') return;
      const patch = {
        sketch: key,
        header: { begin: info.begin, phase: info.phase, reason: info.reason, label: info.label, next_sequence:info.next_sequence,digest:info.digest },
      };
      if (info.stroke) {
        const p = info.stroke;
        patch.stroke = { begin: p.begin, visible: p.visible, sealed: p.sealed, truncated: p.truncated };
        if (r.k === 'sketch-points' && p.next_chunk === r.chunk + 1)
          patch.points = { id: r.id, offset: p.length - r.pts.length, pts: r.pts };
      }
      display.apply(patch);
      active = info.active;
      open = active ? model.sketchMetadata(active).open_stroke || '' : '';
    },
    // Borrowed display paths are read synchronously by the painter. This is
    // display-only data, not the authoritative Model or a receipt projection.
    paths: () => display.paths({ borrow: true }),
    metadata: (sketch,stroke) => display.metadata(sketch,stroke),
    snapshot: () => display.snapshot(),
    stats: () => display.stats(),
  };
}
