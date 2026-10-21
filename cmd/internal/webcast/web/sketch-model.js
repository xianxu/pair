// Pure sketch history contract. DOM matching and transport admission are callers.
export const SKETCH_KINDS = [
  'sketch-begin',
  'sketch-stroke-begin',
  'sketch-points',
  'sketch-stroke-end',
  'sketch-undo',
  'sketch-redo',
  'sketch-label',
  'sketch-interrupt',
];
const enc = new TextEncoder();
const unicodeString = (value) => typeof value === 'string' && !/\p{Cs}/u.test(value);
const hex64 = (x) => typeof x === 'string' && /^[0-9a-f]{64}$/.test(x);
const ident = (x) =>
  typeof x === 'string' && x.length > 0 && enc.encode(x).length <= 128 && !/[\p{Cc}\p{Cf}\p{Cs}]/u.test(x);
const safe = (x) => Number.isSafeInteger(x) && x >= 0;
const number = (x) => Number.isFinite(x) && Math.abs(x) <= 1e7;
const box = (x) => Array.isArray(x) && x.length === 4 && x.every(number) && x[2] > 0 && x[3] > 0;
const need = (ok, message) => {
  if (!ok) throw new Error(`sketch: ${message}`);
};
export function checkSketchRecord(r) {
  need(SKETCH_KINDS.includes(r.k), 'unknown kind');
  need(safe(r.t), 'safe sketch time');
  need(ident(r.operation) && safe(r.sequence) && hex64(r.digest), 'invalid operation metadata');
  if (r.k === 'sketch-begin') {
    need(
      ident(r.id) && ident(r.author?.id) && /^#[0-9a-fA-F]{6}$/.test(r.author?.color || ''),
      'invalid identity or author',
    );
    need(
      !('name' in r.author) ||
        (typeof r.author.name === 'string' &&
          enc.encode(r.author.name).length <= 128 &&
          !/[\p{Cf}\p{Cs}\u0000-\u0008\u000b-\u001f\u007f-\u009f]/u.test(r.author.name)),
      'invalid author name',
    );
    need(r.owner === 'operator' || hex64(r.owner), 'invalid owner');
    need(Number.isSafeInteger(r.doc) && r.doc > 0 && ident(r.tab), 'invalid source');
    need(
      r.vp &&
        Number.isSafeInteger(r.vp.w) &&
        r.vp.w > 0 &&
        Number.isSafeInteger(r.vp.h) &&
        r.vp.h > 0 &&
        number(r.vp.dpr) &&
        r.vp.dpr > 0,
      'invalid viewport',
    );
    return;
  }
  need(ident(r.sketch), 'invalid sketch identity');
  if (
    ['sketch-stroke-begin', 'sketch-points', 'sketch-stroke-end', 'sketch-undo', 'sketch-redo'].includes(r.k)
  )
    need(ident(r.id), 'invalid stroke identity');
  switch (r.k) {
    case 'sketch-stroke-begin':
      for (const key of ['tag', 'route', 'kind', 'sel', 'text'])
        if (r.at && key in r.at) need(unicodeString(r.at[key]), 'invalid anchor string');
      for (const key of ['classes', 'path'])
        if (r.at && key in r.at)
          need(Array.isArray(r.at[key]) && r.at[key].every(unicodeString), 'invalid anchor array');
      if (r.at && 'stable' in r.at) {
        need(
          r.at.stable && typeof r.at.stable === 'object' && !Array.isArray(r.at.stable),
          'invalid stable evidence',
        );
        for (const key of ['id', 'data-testid', 'data-test', 'data-cy'])
          if (key in r.at.stable) need(typeof r.at.stable[key] === 'string', 'invalid stable identity');
        for (const [key, value] of Object.entries(r.at.stable)) {
          if (typeof value === 'string')
            need(unicodeString(key) && unicodeString(value), 'invalid stable Unicode');
        }
      }
      need(
        Number.isSafeInteger(r.at?.node) &&
          r.at.node > 0 &&
          box(r.at.box) &&
          (!r.at.kind || r.at.kind === 'element') &&
          Number.isFinite(r.width) &&
          r.width > 0 &&
          r.width <= 1e7,
        'invalid anchor or width',
      );
      break;
    case 'sketch-points':
      need(
        safe(r.chunk) &&
          Array.isArray(r.pts) &&
          r.pts.length > 0 &&
          r.pts.length <= 64 &&
          r.pts.every((p) => Array.isArray(p) && p.length === 3 && p.every(number) && p[2] >= 0 && p[2] <= 1),
        'invalid points or chunk',
      );
      break;
    case 'sketch-label':
      need(
        typeof r.label === 'string' &&
          r.label.trim().length > 0 &&
          enc.encode(r.label).length <= 1024 &&
          !/[\p{Cf}\p{Cs}\u0000-\u0008\u000b-\u001f\u007f-\u009f]/u.test(r.label),
        'invalid label',
      );
      break;
    case 'sketch-interrupt':
      need(['expiry', 'navigation', 'handoff', 'reset', 'end'].includes(r.reason), 'invalid reason');
      break;
  }
}
const fields = {
  'sketch-begin': ['id', 'author', 'owner', 'doc', 'tab', 'vp'],
  'sketch-stroke-begin': ['sketch', 'id', 'at', 'width'],
  'sketch-points': ['sketch', 'id', 'chunk', 'pts'],
  'sketch-stroke-end': ['sketch', 'id'],
  'sketch-undo': ['sketch', 'id'],
  'sketch-redo': ['sketch', 'id'],
  'sketch-label': ['sketch', 'label'],
  'sketch-interrupt': ['sketch', 'reason'],
};
function known(o, keys, optional = []) {
  const out = {};
  for (const k of keys)
    if (
      o[k] !== undefined &&
      !(
        optional.includes(k) &&
        (!o[k] ||
          (Array.isArray(o[k]) && !o[k].length) ||
          (typeof o[k] === 'object' && !Array.isArray(o[k]) && !Object.keys(o[k]).length))
      )
    )
      out[k] = o[k];
  return out;
}
function byteOrder(a, b) {
  const x = enc.encode(a),
    y = enc.encode(b);
  for (let i = 0; i < Math.min(x.length, y.length); i++) if (x[i] !== y[i]) return x[i] - y[i];
  return x.length - y.length;
}
function canonicalJSON(o) {
  if (Array.isArray(o)) return '[' + o.map(canonicalJSON).join(',') + ']';
  if (o && typeof o === 'object')
    return (
      '{' +
      Object.keys(o)
        .sort(byteOrder)
        .map((k) => JSON.stringify(k) + ':' + canonicalJSON(o[k]))
        .join(',') +
      '}'
    );
  return JSON.stringify(Object.is(o, -0) ? 0 : o);
}
export function sketchCanonical(r) {
  const out = known(r, ['operation', 'sequence', ...(fields[r.k] || [])]);
  if (out.author) out.author = known(out.author, ['id', 'name', 'color'], ['name', 'color']);
  if (out.vp) out.vp = known(out.vp, ['w', 'h', 'dpr']);
  if (out.at)
    out.at = known(
      out.at,
      ['node', 'tag', 'classes', 'stable', 'route', 'kind', 'path', 'sel', 'text', 'box'],
      ['tag', 'classes', 'stable', 'route', 'kind', 'path', 'sel', 'text', 'box'],
    );
  if (out.at?.stable) {
    out.at.stable = Object.fromEntries(
      Object.entries(out.at.stable).filter(([, v]) => typeof v === 'string'),
    );
    if (!Object.keys(out.at.stable).length) delete out.at.stable;
  }
  return canonicalJSON(out).replace(
    /[<>&\u2028\u2029]/g,
    (c) => '\\u' + c.charCodeAt(0).toString(16).padStart(4, '0'),
  );
}
export async function sketchDigest(r) {
  const bytes = await globalThis.crypto.subtle.digest('SHA-256', enc.encode(sketchCanonical(r)));
  return Array.from(new Uint8Array(bytes), (b) => b.toString(16).padStart(2, '0')).join('');
}

export function createSketchState() {
  return { active: '', history_bytes: 0, sketches: new Map(), work: { points: 0, stacks: 0 } };
}
function seal(s, h, truncated) {
  if (!h.open) return;
  const stroke = h.strokes.get(h.open);
  stroke.sealed = true;
  stroke.truncated = truncated;
  if (stroke.pts.length) h.visible.push(h.open);
  h.open = '';
}
export function sketchContextEffect(s, action, reason = '') {
  if (action === 'clear') {
    for (const h of s.sketches.values()) {
      seal(s, h, true);
      h.phase = 'cleared';
    }
    s.active = '';
    return;
  }
  if (!s.active) return;
  const h = s.sketches.get(s.active);
  seal(s, h, true);
  if (action === 'interrupt') {
    h.phase = 'interrupted';
    h.reason = reason;
    s.active = '';
  }
}
export function sketchProjection(s) {
  return {
    active: s.active,
    history_bytes: s.history_bytes,
    sketches: Array.from(s.sketches.values(), (h) => ({
      begin: structuredClone(h.begin),
      phase: h.phase,
      reason: h.reason,
      label: h.label,
      strokes: Array.from(h.strokes.values(), (x) => structuredClone(x)),
      visible: [...h.visible],
      redo: [...h.redo],
      next_sequence: h.next,
      receipts: Array.from(h.receipts.values(), (x) => structuredClone(x)),
    })),
  };
}
function ownedRecord(r) {
  return { ...JSON.parse(sketchCanonical(r)), t: r.t, k: r.k, digest: r.digest };
}
function ownedPoints(points) {
  return points.map((point) => point.map((value) => (value === 0 ? 0 : value)));
}
function wireSize(r) {
  const o = JSON.parse(sketchCanonical(r));
  o.t = r.t;
  o.k = r.k;
  o.digest = r.digest;
  return (
    enc.encode(
      canonicalJSON(o).replace(
        /[<>&\u2028\u2029]/g,
        (c) => '\\u' + c.charCodeAt(0).toString(16).padStart(4, '0'),
      ),
    ).length + 1
  );
}
export function nextSketch(s, r, c) {
  checkSketchRecord(r);
  const id = r.k === 'sketch-begin' ? r.id : r.sketch,
    h = s.sketches.get(id),
    semantic = r.k + ':' + sketchCanonical(r);
  const old = h?.receipts.get(r.operation);
  if (old) {
    need(
      old.sequence === r.sequence && old.digest === r.digest && old.semantic === semantic,
      'operation identity conflict',
    );
    return { duplicate: true, commit() {} };
  }
  if (h) need(r.sequence === h.next, 'sequence gap');
  const size = wireSize(r);
  need(size <= 32 * 1024 * 1024 - s.history_bytes, 'history byte limit');
  let target = h,
    effect;
  if (r.k === 'sketch-begin') {
    need(!h, 'identity already used');
    need(r.sequence === 0, 'begins at sequence zero');
    need(!s.active, 'another sketch is drawing');
    need(s.sketches.size < 256, 'sketch count limit');
    need(c.live && !c.ended && r.doc === c.doc && r.tab === c.tab, 'source is not current and live');
    need(r.author.id === (c.driver || 'operator'), 'author is not driving');
    target = {
      begin: ownedRecord(r),
      phase: 'drawing',
      reason: '',
      label: '',
      strokes: new Map(),
      visible: [],
      redo: [],
      open: '',
      next: 0,
      receipts: new Map(),
      points: 0,
    };
    effect = () => {
      s.sketches.set(id, target);
      s.active = id;
    };
  } else {
    need(h, 'unknown sketch');
    if (r.k === 'sketch-label') {
      need(!c.ended && !h.label && h.phase !== 'cleared' && h.phase !== 'complete', 'label is final');
      need(!h.open, 'end stroke before label');
      effect = () => {
        h.label = r.label;
        if (h.phase === 'drawing') h.phase = 'complete';
        if (s.active === id) s.active = '';
      };
    } else if (r.k === 'sketch-stroke-end') {
      const stroke = h.strokes.get(r.id);
      need(stroke, 'unknown stroke');
      need(!['cleared', 'complete'].includes(h.phase), 'history is closed');
      if (stroke.sealed) effect = () => {};
      else {
        need(h.open === r.id, 'stroke is not open');
        effect = () => seal(s, h, false);
      }
    } else {
      need(h.phase === 'drawing' && s.active === id, 'history is closed');
      if (r.k !== 'sketch-interrupt')
        need(
          c.live && !c.ended && h.begin.doc === c.doc && h.begin.tab === c.tab,
          'drawing is paused or stale',
        );
      switch (r.k) {
        case 'sketch-stroke-begin': {
          need(!h.open, 'another stroke is open');
          need(!h.strokes.has(r.id), 'stroke identity already used');
          need(h.strokes.size < 64, 'stroke limit');
          const stroke = { begin: ownedRecord(r), pts: [], next_chunk: 0, sealed: false, truncated: false };
          effect = () => {
            h.strokes.set(r.id, stroke);
            h.open = r.id;
            h.redo = [];
          };
          break;
        }
        case 'sketch-points': {
          const stroke = h.strokes.get(r.id);
          need(stroke && h.open === r.id && !stroke.sealed, 'stroke is not open');
          need(r.chunk === stroke.next_chunk, 'chunk out of sequence');
          need(r.pts.length <= 4096 - stroke.pts.length && r.pts.length <= 16384 - h.points, 'point limit');
          const pts = ownedPoints(r.pts);
          effect = () => {
            stroke.pts.push(...pts);
            stroke.next_chunk++;
            h.points += pts.length;
            s.work.points += pts.length;
          };
          break;
        }
        case 'sketch-undo':
          need(!h.open && h.visible.length && h.visible.at(-1) === r.id, 'undo requires visible stack top');
          effect = () => {
            h.visible.pop();
            h.redo.push(r.id);
            s.work.stacks++;
          };
          break;
        case 'sketch-redo':
          need(!h.open && h.redo.length && h.redo.at(-1) === r.id, 'redo requires redo stack top');
          effect = () => {
            h.redo.pop();
            h.visible.push(r.id);
            s.work.stacks++;
          };
          break;
        case 'sketch-interrupt':
          effect = () => sketchContextEffect(s, 'interrupt', r.reason);
          break;
      }
    }
  }
  const receipt = { operation: r.operation, sequence: r.sequence, digest: r.digest, semantic };
  return {
    duplicate: false,
    commit() {
      effect();
      target.receipts.set(r.operation, receipt);
      target.next++;
      s.history_bytes += size;
    },
  };
}

// Control remains strict after a catch-up bootstrap; only its initial baseline
// may omit the preceding chatter, as in the pre-sketch reducer contract.
export function nextSketchControl(current, r, seen) {
  const driver = r.driver || null,
    pending = r.pending || [],
    who = r.who?.id;
  need(
    who && ['reset', 'request', 'grant', 'deny', 'revoke', 'release', 'leave'].includes(r.op),
    'invalid control record',
  );
  if (r.op === 'reset')
    need(who === 'operator' && !driver && !pending.length, 'reset must leave no driver or requests');
  if (driver) need(typeof driver.id === 'string' && driver.id.length, 'driver without identity');
  const ids = new Set(driver ? [driver.id] : []);
  for (const p of pending) {
    need(p.id && !ids.has(p.id), 'duplicate or absent control identity');
    ids.add(p.id);
  }
  if (r.op !== 'reset')
    need(
      r.op === 'request'
        ? pending.some((p) => p.id === who)
        : r.op === 'grant'
          ? driver?.id === who
          : driver?.id !== who && !pending.some((p) => p.id === who),
      'inconsistent control record',
    );
  if (seen) {
    let d = current.driver || null,
      ps = [...(current.pending || [])];
    const at = ps.findIndex((p) => p.id === who),
      driving = d?.id === who;
    switch (r.op) {
      case 'reset':
        d = null;
        ps = [];
        break;
      case 'request':
        need(!driving && at < 0, 'already drives or asked');
        ps.push(r.who);
        break;
      case 'grant':
        need(at >= 0, 'viewer has not asked');
        ps.splice(at, 1);
        d = r.who;
        break;
      case 'deny':
        need(at >= 0, 'viewer has not asked');
        ps.splice(at, 1);
        break;
      case 'revoke':
        need(driving, 'viewer is not driving');
        d = null;
        break;
      case 'release':
      case 'leave':
        if (driving) d = null;
        else {
          need(at >= 0, 'viewer neither drives nor asked');
          ps.splice(at, 1);
        }
        break;
    }
    need(
      (d?.id || '') === (driver?.id || '') &&
        ps.length === pending.length &&
        ps.every((p, i) => p.id === pending[i].id),
      'control stated result differs',
    );
  }
  return { driver: structuredClone(driver), pending: structuredClone(pending) };
}

export function snapshotControl(c) {
  need(c && typeof c === 'object', 'invalid snapshot control');
  const seen = new Set();
  if (c.driver) {
    need(typeof c.driver.id === 'string' && c.driver.id, 'snapshot driver without identity');
    seen.add(c.driver.id);
  }
  need(!('pending' in c) || Array.isArray(c.pending), 'invalid pending control');
  for (const p of c.pending || []) {
    need(typeof p.id === 'string' && p.id && !seen.has(p.id), 'invalid snapshot control identities');
    seen.add(p.id);
  }
  return { driver: structuredClone(c.driver || null), pending: structuredClone(c.pending || []) };
}
export function restoreSketchState(p) {
  need(
    p &&
      safe(p.history_bytes) &&
      p.history_bytes <= 32 * 1024 * 1024 &&
      Array.isArray(p.sketches) &&
      p.sketches.length <= 256,
    'snapshot history bound',
  );
  const out = createSketchState();
  out.active = p.active;
  out.history_bytes = p.history_bytes;
  let semanticBytes = 0,
    activeCount = 0;
  for (const v of p.sketches) {
    checkSketchRecord(v.begin);
    need(v.begin.k === 'sketch-begin', 'invalid snapshot begin');
    const id = v.begin.id;
    need(
      !out.sketches.has(id) &&
        safe(v.next_sequence) &&
        v.next_sequence > 0 &&
        Array.isArray(v.receipts) &&
        v.receipts.length === v.next_sequence &&
        Array.isArray(v.strokes) &&
        v.strokes.length <= 64,
      'invalid snapshot history identities',
    );
    need(['drawing', 'complete', 'interrupted', 'cleared'].includes(v.phase), 'invalid snapshot phase');
    need(
      v.reason === '' || ['expiry', 'navigation', 'handoff', 'reset', 'end'].includes(v.reason),
      'invalid snapshot interruption',
    );
    if (v.phase === 'drawing') need(!v.label && !v.reason, 'invalid drawing metadata');
    if (v.phase === 'complete') need(v.label && !v.reason, 'invalid completion metadata');
    if (v.phase === 'interrupted') need(v.reason, 'missing interruption evidence');
    if (v.label) checkSketchRecord({ ...v.begin, k: 'sketch-label', sketch: id, label: v.label });
    const h = {
      begin: ownedRecord(v.begin),
      phase: v.phase,
      reason: v.reason,
      label: v.label,
      strokes: new Map(),
      visible: structuredClone(v.visible),
      redo: structuredClone(v.redo),
      open: '',
      next: v.next_sequence,
      receipts: new Map(),
      points: 0,
    };
    const strokes = new Map(),
      strokeDigests = new Map();
    const pointWitness = new Map(),
      chunkWitness = new Map();
    let labelWitness = '';
    for (const [i, receipt] of v.receipts.entries()) {
      need(
        receipt.sequence === i &&
          ident(receipt.operation) &&
          hex64(receipt.digest) &&
          !h.receipts.has(receipt.operation) &&
          typeof receipt.semantic === 'string',
        'invalid snapshot receipt sequence',
      );
      need(
        enc.encode(receipt.semantic).length <= p.history_bytes - semanticBytes,
        'snapshot receipt byte bound',
      );
      const split = receipt.semantic.indexOf(':');
      need(split > 0, 'invalid receipt semantic');
      const r = {
        ...JSON.parse(receipt.semantic.slice(split + 1)),
        k: receipt.semantic.slice(0, split),
        t: 0,
        digest: receipt.digest,
      };
      checkSketchRecord(r);
      need(
        r.operation === receipt.operation &&
          r.sequence === receipt.sequence &&
          r.k + ':' + sketchCanonical(r) === receipt.semantic,
        'invalid snapshot receipt identity',
      );
      need((r.k === 'sketch-begin' ? r.id : r.sketch) === id, 'snapshot receipt belongs to another sketch');
      if (i === 0)
        need(
          r.k === 'sketch-begin' &&
            v.begin.k + ':' + sketchCanonical(v.begin) === receipt.semantic &&
            v.begin.digest === receipt.digest,
          'snapshot begin differs from receipt',
        );
      else need(r.k !== 'sketch-begin', 'snapshot second begin');
      if (r.k === 'sketch-stroke-begin') {
        need(!strokes.has(r.id), 'duplicate snapshot stroke begin');
        strokes.set(r.id, receipt.semantic);
        strokeDigests.set(r.id, receipt.digest);
      }
      if (r.k === 'sketch-points') {
        need(
          strokes.has(r.id) && r.chunk === (chunkWitness.get(r.id) || 0),
          'snapshot point receipt out of order',
        );
        const points = pointWitness.get(r.id) || [];
        points.push(...r.pts);
        need(points.length <= 4096, 'snapshot point receipt bound');
        pointWitness.set(r.id, points);
        chunkWitness.set(r.id, (chunkWitness.get(r.id) || 0) + 1);
      } else if (r.k === 'sketch-label') {
        need(!labelWitness, 'snapshot repeated label receipt');
        labelWitness = r.label;
      }
      h.receipts.set(receipt.operation, structuredClone(receipt));
      semanticBytes += enc.encode(receipt.semantic).length;
      need(semanticBytes <= p.history_bytes, 'snapshot receipt byte bound');
    }
    need(labelWitness === v.label, 'snapshot label differs from receipt');
    for (const view of v.strokes) {
      checkSketchRecord(view.begin);
      need(
        view.begin.k === 'sketch-stroke-begin' &&
          view.begin.sketch === id &&
          !h.strokes.has(view.begin.id) &&
          Array.isArray(view.pts) &&
          view.pts.length <= 4096 &&
          safe(view.next_chunk),
        'invalid snapshot stroke',
      );
      const points = pointWitness.get(view.begin.id) || [];
      need(
        view.next_chunk === (chunkWitness.get(view.begin.id) || 0) &&
          view.pts.length === points.length &&
          view.pts.every(
            (p, i) => Array.isArray(p) && p.length === 3 && p.every((v, j) => v === points[i][j]),
          ),
        'snapshot points differ from receipts',
      );
      need(
        strokes.get(view.begin.id) === view.begin.k + ':' + sketchCanonical(view.begin) &&
          strokeDigests.get(view.begin.id) === view.begin.digest,
        'snapshot stroke differs from receipt',
      );
      need(
        view.next_chunk <= view.pts.length &&
          view.pts.length <= view.next_chunk * 64 &&
          (!view.truncated || view.sealed),
        'invalid snapshot chunks',
      );
      need(
        view.pts.every(
          (p) => Array.isArray(p) && p.length === 3 && p.every(number) && p[2] >= 0 && p[2] <= 1,
        ),
        'invalid snapshot point',
      );
      need(typeof view.sealed === 'boolean' && typeof view.truncated === 'boolean', 'invalid snapshot flags');
      if (!view.sealed) {
        need(!h.open && v.phase === 'drawing', 'invalid snapshot open stroke');
        h.open = view.begin.id;
      }
      h.strokes.set(view.begin.id, {
        begin: ownedRecord(view.begin),
        pts: ownedPoints(view.pts),
        next_chunk: view.next_chunk,
        sealed: view.sealed,
        truncated: view.truncated,
      });
      h.points += view.pts.length;
    }
    need(
      strokes.size === h.strokes.size &&
        h.points <= 16384 &&
        Array.isArray(h.visible) &&
        Array.isArray(h.redo),
      'snapshot stroke/point bound',
    );
    const seen = new Set();
    for (const sid of [...h.visible, ...h.redo]) {
      const st = h.strokes.get(sid);
      need(st && st.sealed && st.pts.length && !seen.has(sid), 'invalid snapshot stack');
      seen.add(sid);
    }
    if (v.phase === 'drawing') {
      activeCount++;
      need(p.active === id, 'snapshot active mismatch');
    }
    validateSketchViewHistory(v);
    out.sketches.set(id, h);
  }
  need(
    typeof p.active === 'string' && (p.active ? activeCount === 1 : activeCount === 0),
    'snapshot active identity missing',
  );
  return out;
}

// Reuse the transition owner for receipt ordering and stack authority.
// Context records are absent after compaction; their retained truncation and
// interruption facts provide the seals the original reducer already accepted.
function validateSketchViewHistory(view) {
  const check = createSketchState();
  const strokes = new Map(view.strokes.map((stroke) => [stroke.begin.id, stroke]));
  const context = {
    live: true,
    doc: view.begin.doc,
    tab: view.begin.tab,
    driver: view.begin.author.id === 'operator' ? '' : view.begin.author.id,
  };
  for (const receipt of view.receipts) {
    const split = receipt.semantic.indexOf(':');
    const event = {
      ...JSON.parse(receipt.semantic.slice(split + 1)),
      k: receipt.semantic.slice(0, split),
      t: 0,
      digest: receipt.digest,
    };
    const history = check.sketches.get(view.begin.id);
    if (history) {
      const needsClosed =
        ['sketch-stroke-begin', 'sketch-undo', 'sketch-redo', 'sketch-label'].includes(event.k) ||
        (event.k === 'sketch-stroke-end' && event.id === history.open);
      if (needsClosed && history.open && strokes.get(history.open)?.truncated) {
        sketchContextEffect(check, 'pause');
      }
      if (event.k === 'sketch-label' && view.reason && history.phase === 'drawing') {
        sketchContextEffect(check, 'interrupt', view.reason);
      }
    }
    nextSketch(check, event, context).commit();
  }
  const history = check.sketches.get(view.begin.id);
  need(history, 'snapshot has no receipt history');
  const open = strokes.get(history.open);
  if (history.open && open?.sealed && open.truncated) sketchContextEffect(check, 'pause');
  if (history.phase === 'drawing' && view.reason) sketchContextEffect(check, 'interrupt', view.reason);
  if (view.phase === 'cleared') sketchContextEffect(check, 'clear');
  const equal = (a, b) => a.length === b.length && a.every((v, i) => v === b[i]);
  need(
    history.phase === view.phase &&
      history.reason === view.reason &&
      history.label === view.label &&
      equal(history.visible, view.visible) &&
      equal(history.redo, view.redo),
    'snapshot projection differs from receipt history',
  );
  for (const [id, stroke] of history.strokes) {
    const projected = strokes.get(id);
    need(
      projected?.sealed === stroke.sealed && projected.truncated === stroke.truncated,
      'snapshot seal differs from receipt history',
    );
  }
}
