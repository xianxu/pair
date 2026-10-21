// The recording's meaning in the viewer (#460): a port of the Go reducer
// (recording.State.Apply) and of State.Canonical(). It holds no DOM; the
// renderer draws what this model accepted. Where the two disagree the Go
// package is right: the node tests hold this port to every fixture's pinned
// canonical hash, so a drift is a red test. Sketch history and control
// transitions share their own pure core, outside legacy DOM hashes.

import { SKETCH_KINDS, createSketchState, nextSketch, sketchProjection, sketchContextEffect, nextSketchControl, restoreSketchState, snapshotControl } from './sketch-model.js';

export const VERSION = 1;

const SEG_WHY = ['live', 'resume', 'navigate', 'tab'];
const containerTypes = new Set(['el', 'doc', 'shadow', 'frag']);

function fail(msg) { throw new Error(msg); }

// checkTree is record.go's checkTree: a well-formed subtree.
function checkTree(n) {
  if (!(n.id > 0)) fail('node without id');
  switch (n.y) {
    case 'el': if (!n.n) fail(`node ${n.id}: element without n`); break;
    case 'text': case 'comment': if (typeof n.v !== 'string') fail(`node ${n.id}: ${n.y} without v`); break;
    case 'shadow': if (n.mode !== 'open' && n.mode !== 'closed') fail(`node ${n.id}: shadow root without mode`); break;
    case 'doc': case 'doctype': case 'frag': break;
    default: fail(`node ${n.id}: unknown type ${JSON.stringify(n.y)}`);
  }
  if (n.sr && n.sr.y !== 'shadow') fail(`node ${n.id}: sr is not a shadow node`);
  if (n.tc && n.tc.y !== 'frag') fail(`node ${n.id}: tc is not a frag node`);
  if (n.sr) checkTree(n.sr);
  if (n.tc) checkTree(n.tc);
  for (const k of n.c || []) checkTree(k);
}

// checkRecord is each kind's check(): required fields.
function checkRecord(r) {
  const need = (ok, f) => { if (!ok) fail(`missing ${f}`); };
  switch (r.k) {
    case 'head': need(r.format === 'couch-webcast', 'format couch-webcast'); need(r.v > 0, 'v'); need(!!r.id, 'id'); need(!!r.started, 'started'); break;
    case 'seg': need(r.seg > 0, 'seg'); need(SEG_WHY.includes(r.why), 'why'); need(r.vp && r.vp.w > 0 && r.vp.h > 0, 'vp'); break;
    case 'stop': need(r.why === 'pause' || r.why === 'end', 'why'); break;
    case 'doc': need(r.doc > 0, 'doc'); need(r.tree && r.tree.y === 'doc' && r.tree.id === r.doc, 'tree (a doc node with id doc)'); checkTree(r.tree); break;
    case 'ins': need(r.p > 0, 'p'); need(r.node && r.node.id > 0, 'node'); checkTree(r.node); break;
    case 'rm': need(r.id > 0, 'id'); break;
    case 'kids': need(r.p > 0, 'p'); for (const c of r.c || []) checkTree(c); break;
    case 'attr': need(r.id > 0, 'id'); need(!!r.name, 'name'); break;
    case 'text': case 'sheet': case 'adopt': case 'scroll': case 'click': need(r.id > 0, 'id'); break;
    case 'shadow': need(r.host > 0, 'host'); need(r.root && r.root.y === 'shadow', 'root (a shadow node)'); checkTree(r.root); break;
    case 'font': need(!!r.family, 'family'); need(!!r.src, 'src'); break;
    case 'asset': need(!!r.url, 'url'); need(/^[0-9a-f]{64}$/.test(r.hash || ''), 'hash (64 lowercase hex)'); need(!!r.mime, 'mime'); break;
    case 'input': need(r.id > 0, 'id'); need('v' in r || 'checked' in r || 'sel' in r, 'v, checked or sel'); break;
    case 'nav': need(!!r.url, 'url'); break;
    case 'vp': need(r.w > 0 && r.h > 0, 'w, h'); break;
    case 'note': need(!!r.id, 'id'); need(!!(r.author && r.author.id), 'author.id'); need((r.at && r.at.node > 0) || !!r.region, 'at.node or region'); break;
    case 'note-edit': need(!!r.id, 'id'); need(typeof r.text === 'string' && !!r.text.trim() && new TextEncoder().encode(r.text).length <= 1024 && !/[\p{Cf}\p{Cs}\u0000-\u0008\u000b-\u001f\u007f-\u009f]/u.test(r.text), 'bounded nonblank plain text'); break;
    case 'note-state': need(!!r.id, 'id'); need(r.state === 'open' || r.state === 'resolved', 'state'); break;
    case 'stroke': need(!!r.id, 'id'); need(!!(r.author && r.author.id), 'author.id'); need(Array.isArray(r.pts) && r.pts.length > 0, 'pts'); break;
    case 'annotation-clear': need(r.who?.id === 'operator', 'operator attribution'); break;
  }
}

// A signature is what an id may never change: its type and tag.
const signature = (n) => `${n.y}:${n.ns || ''}:${n.n || ''}`;

// Model is the state of one recording at its latest record. Apply folds a
// record in atomically: it checks every reference first and throws on a
// refusal, leaving the state as it was.
export class Model {
  #sketches;
  #control;
  #ctlSeen;

  // Control and sketch projections are owned read views. Only apply() can
  // change the accepted history, receipts and driving generation.
  get control() { return structuredClone(this.#control); }

  // paused: a page was cast and the cast is paused on it (not ended). The
  // one viewer-side rule for "viewers do nothing" (pair#499), as Go's
  // recording.State.Paused.
  get paused() { return this.seg > 0 && !this.live && !this.ended; }

  constructor({allowSketchSnapshot = false} = {}) {
    this.allowSketchSnapshot = allowSketchSnapshot;
    this.#sketches = createSketchState(); this.#control = {driver:null,pending:[]}; this.#ctlSeen = false;
    this.head = null;
    this.t = 0;
    this.seg = 0; this.docSeg = 0; this.why = ''; this.live = false;
    this.ended = false; // a stop end came: the cast is over
    this.tab = { id: '', url: '' }; this.vp = { w: 0, h: 0, dpr: 0 };
    this.url = ''; this.title = '';
    this.assets = new Map(); // absolute URL -> asset record
    this.notes = []; this.strokes = [];
    this.types = new Map(); // id -> signature, for the whole recording
    this.resetDOM();
  }

  resetDOM() {
    this.top = null; this.selection = null; this.focus = 0; this.lastClick = null; this.fonts = [];
    this.hover = 0; this.ptr = null;
    this.nodes = new Map();
    this.sheets = new Map(); // id -> {owner, text}
    this.adopted = new Map(); // doc or shadow id -> [sheet ids]
    this.inputs = new Map(); this.scrolls = new Map();
    this.boxes = new Map(); this.boxesT = 0;
    this.awaitTop = false;
  }

  node(id) { return this.nodes.get(id); }

  sketchProjection() { return sketchProjection(this.#sketches); }

  apply(r) {
    if (!this.head) {
      if (r.k !== 'head') fail('the first record must be head');
      if (r.v !== VERSION) fail(`format version ${r.v}; this reader knows ${VERSION}`);
    }
    if (r.k === 'sketch-snapshot') {
      if (!this.allowSketchSnapshot) fail('sketch snapshot is catch-up only');
      if (!Number.isSafeInteger(r.t) || r.t < this.t) fail('invalid snapshot time');
      const control = snapshotControl(r.control), sketches = restoreSketchState(r.sketches);
      if (this.ended && sketches.active) fail('ended snapshot still drawing');
      if (sketches.active) {
        const begin = sketches.sketches.get(sketches.active).begin;
        if (begin.doc !== this.docSeg || begin.tab !== this.tab.id) fail('snapshot active source differs');
        if (begin.author.id !== (control.driver?.id || 'operator')) fail('snapshot active author is not driving');
      }
      this.#control = control; this.#sketches = sketches; this.#ctlSeen = true; this.t = r.t;
      return;
    }
    if (SKETCH_KINDS.includes(r.k)) {
      const delta = nextSketch(this.#sketches, r, {
        live: this.live, ended: this.ended, doc: this.docSeg,
        tab: this.tab.id, driver: this.#control.driver?.id || '',
      });
      if (delta.duplicate) return;
      if (r.t < this.t) fail(`t=${r.t} is before the previous record's t=${this.t}`);
      delta.commit(); this.t = r.t; return;
    }
    if (r.t < this.t) fail(`t=${r.t} is before the previous record's t=${this.t}`);
    checkRecord(r);
    this.applyKind(r);
    this.t = r.t;
  }

  inSegment() { if (!this.live || !this.top) fail('no live segment with a top document (a change outside seg … stop)'); }

  applyKind(r) {
    if ((r.k === 'seg' || r.k === 'ctl' || r.k === 'annotation-clear') && this.ended) fail(`a ${r.k} after stop end: the cast is over`);
    switch (r.k) {
      case 'head':
        if (this.head) fail('a second head');
        this.head = r; return;
      case 'seg':
        if (r.seg <= this.seg) fail(`segment ${r.seg} after segment ${this.seg}`);
        if (r.why !== 'resume' || (r.tab?.id || '') !== this.tab.id) sketchContextEffect(this.#sketches,'interrupt','navigation');
        this.resetDOM();
        this.seg = r.seg; this.why = r.why; this.live = true; this.tab = r.tab || { id: '', url: '' }; this.vp = vpOf(r.vp);
        if (r.why !== 'resume' || this.docSeg === 0) this.docSeg = r.seg;
        this.url = this.tab.url || ''; this.title = this.tab.title || ''; this.awaitTop = true;
        return;
      case 'stop':
        if (!this.live) fail('stop without a live segment');
        sketchContextEffect(this.#sketches,r.why==='pause'?'pause':'interrupt','end');
        this.live = false; this.awaitTop = false;
        if (r.why === 'end') { this.ended = true; this.#control={driver:null,pending:[]}; }
        return;
      case 'ctl': {
        const next = nextSketchControl(this.#control, r, this.#ctlSeen);
        if (r.op === 'reset') sketchContextEffect(this.#sketches, 'interrupt', 'reset');
        else if ((this.#control.driver?.id || '') !== (next.driver?.id || '')) {
          sketchContextEffect(this.#sketches, 'interrupt', 'handoff');
        }
        this.#control = next; this.#ctlSeen = true; return;
      }
      case 'doc': return this.applyDoc(r);
      case 'asset': this.assets.set(r.url, r); return;
      case 'note': this.notes.push({ note: r, state: 'open', seg: this.seg }); return;
      case 'note-edit': {
        if (!this.live || this.ended) fail('note edit requires a live segment');
        const n = this.notes.find(x => x.note.id === r.id);
        if (!n) fail(`note ${JSON.stringify(r.id)} is unknown`);
        if (n.state !== 'open') fail('note edit requires an open note');
        n.note = { ...n.note, text: r.text, edited: true }; n.revision = (n.revision || 0) + 1; return;
      }
      case 'note-state': {
        const n = this.notes.find((x) => x.note.id === r.id);
        if (!n) fail(`note ${JSON.stringify(r.id)} is unknown`);
        n.state = r.state; n.revision = (n.revision || 0) + 1;
        n.resolvedBy = r.state === 'resolved' && r.who ? { ...r.who } : null; return;
      }
      case 'stroke': this.strokes.push({ stroke: r, seg: this.seg }); return;
      case 'annotation-clear': sketchContextEffect(this.#sketches,'clear'); this.notes = []; this.strokes = []; return;
      case 'frame': return;
      case 'ins': case 'rm': case 'kids': case 'attr': case 'text': case 'shadow': case 'sheet': case 'adopt':
      case 'font': case 'input': case 'scroll': case 'selection': case 'nav': case 'title': case 'vp':
      case 'click': case 'focus': case 'layout': case 'hover': case 'ptr':
        this.inSegment();
        return this.applyInSegment(r);
      default:
        return; // a kind this version doesn't know is ignored
    }
  }

  applyDoc(r) {
    if (!r.host) {
      if (!this.awaitTop) fail('a top doc comes only right after seg');
      this.checkNew(r.tree);
      this.top = this.build(r.tree, null);
      this.top.url = r.url; this.url = r.url; this.awaitTop = false;
      return;
    }
    this.inSegment();
    const host = this.element(r.host);
    if (host.n !== 'iframe' && host.n !== 'frame') fail(`host ${r.host} is a ${host.n}, not an iframe`);
    this.checkNew(r.tree, host.frame ? [host.frame] : []);
    if (host.frame) this.forget(host.frame);
    host.frame = this.build(r.tree, null);
    host.frame.host = host; host.frame.url = r.url;
  }

  applyInSegment(r) {
    switch (r.k) {
      case 'ins': {
        const p = this.container(r.p);
        let at = 0;
        if (r.after) {
          const i = p.kids.findIndex((k) => k.id === r.after);
          if (i < 0) fail(`after ${r.after} is not a child of ${r.p}`);
          at = i + 1;
        }
        this.checkNew(r.node);
        p.kids.splice(at, 0, this.build(r.node, p));
        return;
      }
      case 'rm': {
        const n = this.nodes.get(r.id);
        if (!n) fail(`node ${r.id} is unknown`);
        if (n.y === 'doc') fail(`node ${r.id} is a document; documents go with seg or a new doc for their frame`);
        this.detach(n); this.forget(n);
        return;
      }
      case 'kids': {
        const p = this.container(r.p);
        const c = r.c || [];
        for (const k of c) this.checkNew(k, p.kids);
        dupIDs(c, r.p);
        for (const k of p.kids) this.forget(k);
        p.kids = c.map((k) => this.build(k, p));
        return;
      }
      case 'attr': {
        const n = this.element(r.id);
        if (r.v === null || r.v === undefined) { if (n.a) delete n.a[r.name]; } else { (n.a ||= {})[r.name] = r.v; }
        return;
      }
      case 'text': {
        const n = this.nodes.get(r.id);
        if (!n || (n.y !== 'text' && n.y !== 'comment')) fail(`node ${r.id} is not a text or comment node`);
        n.v = r.v ?? '';
        return;
      }
      case 'shadow': {
        const host = this.element(r.host);
        if (host.shadow) fail(`element ${r.host} already has a shadow root`);
        this.checkNew(r.root);
        host.shadow = this.build(r.root, null); host.shadow.host = host;
        return;
      }
      case 'sheet': {
        if (r.owner) {
          const o = this.element(r.owner);
          if (o.n !== 'style' && o.n !== 'link') fail(`sheet owner ${r.owner} is a ${o.n}, not a style or link`);
        }
        if (this.nodes.has(r.id)) fail(`id ${r.id} names a live node, not a sheet`);
        this.claim(r.id, 'sheet');
        this.types.set(r.id, 'sheet');
        this.sheets.set(r.id, { owner: r.owner || 0, text: r.text ?? '' });
        return;
      }
      case 'adopt': {
        const n = this.nodes.get(r.id);
        if (!n || (n.y !== 'doc' && n.y !== 'shadow')) fail(`node ${r.id} is not a document or shadow root`);
        for (const id of r.sheets || []) if (!this.sheets.has(id)) fail(`sheet ${id} is unknown`);
        this.adopted.set(r.id, [...(r.sheets || [])]);
        return;
      }
      case 'font': this.fonts.push(r); return;
      case 'input': this.element(r.id); this.inputs.set(r.id, r); return;
      case 'scroll': {
        const n = this.nodes.get(r.id);
        if (!n || (n.y !== 'el' && n.y !== 'doc')) fail(`node ${r.id} is not an element or document`);
        this.scrolls.set(r.id, [r.x || 0, r.y || 0]);
        return;
      }
      case 'selection':
        for (const p of [r.anchor, r.focus]) if (p && !this.nodes.has(p[0])) fail(`selection node ${p[0]} is unknown`);
        this.selection = (r.anchor || r.focus) ? r : null;
        return;
      case 'nav': this.url = r.url; return;
      case 'title': this.title = r.v ?? ''; return;
      case 'vp': this.vp = vpOf(r); return;
      case 'click':
        if (!this.nodes.has(r.id)) fail(`node ${r.id} is unknown`);
        this.lastClick = r; return;
      case 'focus':
        if (r.id && !this.nodes.has(r.id)) fail(`node ${r.id} is unknown`);
        this.focus = r.id || 0; return;
      case 'hover':
        // Drawn only, outside the canonical state: a hover on a node this
        // viewer doesn't have (one the recorder never sent) clears the hover
        // rather than forcing a resync.
        this.hover = this.nodes.has(r.id) ? r.id : 0; return;
      case 'ptr': this.ptr = { x: r.x, y: r.y, t: r.t }; return;
      case 'layout': {
        const boxes = r.boxes || {};
        for (const id of Object.keys(boxes)) if (!this.nodes.has(Number(id))) fail(`layout of unknown node ${id}`);
        this.boxes = new Map(Object.entries(boxes).map(([id, b]) => [Number(id), b]));
        this.boxesT = r.t;
        return;
      }
    }
  }

  element(id) {
    const n = this.nodes.get(id);
    if (!n || n.y !== 'el') fail(`node ${id} is not a live element`);
    return n;
  }

  container(id) {
    const n = this.nodes.get(id);
    if (!n) fail(`node ${id} is unknown`);
    if (!containerTypes.has(n.y)) fail(`node ${id} is a ${n.y} and has no children`);
    return n;
  }

  claim(id, sig) {
    const was = this.types.get(id);
    if (was !== undefined && was !== sig) fail(`id ${id} was ${was} and is now ${sig}; an id names one thing`);
  }

  // checkNew: every id of a subtree about to be added is new to the live
  // DOM (or inside one of the subtrees it replaces), unique within it, and
  // keeps its recording-wide signature.
  checkNew(n, replacing = []) {
    const seen = new Set();
    const walk = (n) => {
      if (seen.has(n.id)) fail(`id ${n.id} appears twice in one subtree`);
      seen.add(n.id);
      const live = this.nodes.get(n.id);
      if (live && !replacing.some((r) => under(live, r))) fail(`id ${n.id} is already live`);
      this.claim(n.id, signature(n));
      if (n.sr) walk(n.sr);
      if (n.tc) walk(n.tc);
      for (const k of n.c || []) walk(k);
    };
    walk(n);
  }

  build(n, parent) {
    const d = { id: n.id, y: n.y, n: n.n || '', ns: n.ns || '', mode: n.mode || '', url: '', v: n.v ?? '',
      a: n.a && Object.keys(n.a).length ? { ...n.a } : null,
      parent, kids: [], host: null, shadow: null, content: null, frame: null };
    this.nodes.set(n.id, d);
    this.types.set(n.id, signature(n));
    if (n.sr) { d.shadow = this.build(n.sr, null); d.shadow.host = d; }
    if (n.tc) { d.content = this.build(n.tc, null); d.content.host = d; }
    for (const k of n.c || []) d.kids.push(this.build(k, d));
    return d;
  }

  detach(n) {
    if (n.parent) n.parent.kids = n.parent.kids.filter((k) => k !== n);
    else if (n.host && n.host.shadow === n) n.host.shadow = null;
    else if (n.host && n.host.content === n) n.host.content = null;
  }

  forget(n) {
    if (this.nodes.get(n.id) === n) this.nodes.delete(n.id);
    this.inputs.delete(n.id); this.scrolls.delete(n.id); this.boxes.delete(n.id); this.adopted.delete(n.id);
    if (n.n === 'style' || n.n === 'link') {
      for (const [id, sh] of this.sheets) if (sh.owner === n.id) this.sheets.delete(id);
    }
    if (this.focus === n.id) this.focus = 0;
    if (this.hover === n.id) this.hover = 0;
    for (const sub of [n.shadow, n.content, n.frame]) if (sub) this.forget(sub);
    for (const k of n.kids) this.forget(k);
  }

  // ownedSheet is the newest sheet an owner element has, or undefined.
  ownedSheet(id) {
    let newest = 0, text;
    for (const [sid, sh] of this.sheets) if (sh.owner === id && sid > newest) { newest = sid; text = sh.text; }
    return text;
  }

  adoptedTexts(id) {
    const out = [];
    for (const sid of this.adopted.get(id) || []) { const sh = this.sheets.get(sid); if (sh) out.push(sh.text); }
    return out;
  }

  // canonicalNode is one node in State.Canonical()'s shape. kids lets the
  // renderer walk its own DOM while values come from here.
  canonicalNode(n, kids = (n) => n.kids, sub = (n, which) => n[which]) {
    const c = { id: n.id, y: n.y };
    if (n.n) c.n = n.n;
    if (n.ns) c.ns = n.ns;
    if (n.url) c.url = n.url;
    if (n.a && Object.keys(n.a).length) c.a = sortKeys(n.a);
    if (n.y === 'text' || n.y === 'comment') c.v = n.v;
    if (n.mode) c.mode = n.mode;
    const input = this.inputs.get(n.id);
    if (input) {
      const ci = {};
      if ('v' in input && input.v !== null) ci.v = input.v;
      if ('checked' in input && input.checked !== null) ci.checked = input.checked;
      if (input.sel && input.sel.length) ci.sel = input.sel;
      c.$input = ci;
    }
    if (this.scrolls.has(n.id)) c.$scroll = this.scrolls.get(n.id);
    const sheet = this.ownedSheet(n.id);
    if (sheet !== undefined) c.$sheet = sheet;
    const adopt = this.adoptedTexts(n.id);
    if (adopt.length) c.$adopt = adopt;
    for (const [key, which] of [['sr', 'shadow'], ['tc', 'content'], ['frame', 'frame']]) {
      const s = sub(n, which);
      if (s) c[key] = this.canonicalNode(s, kids, sub);
    }
    const ks = kids(n);
    if (ks.length) c.c = ks.map((k) => this.canonicalNode(k, kids, sub));
    return c;
  }

  canonicalState(doc) {
    const c = { seg: this.seg, live: this.live, url: this.url, title: this.title, vp: { w: this.vp.w, h: this.vp.h, dpr: this.vp.dpr } };
    if (this.focus) c.focus = this.focus;
    if (this.selection) c.selection = [this.selection.anchor || null, this.selection.focus || null];
    if (this.fonts.length) c.fonts = this.fonts.map((f) => [f.family, f.src]);
    c.doc = doc;
    return c;
  }

  // canonical is State.Canonical() as an object.
  canonical() { return this.canonicalState(this.top ? this.canonicalNode(this.top) : null); }
}

function vpOf(v) { return { w: v.w || 0, h: v.h || 0, dpr: v.dpr || 0 }; }

function under(n, root) {
  for (; n; n = n.parent || n.host) if (n === root) return true;
  return false;
}

function dupIDs(c, parent) {
  const seen = new Set([parent]);
  const walk = (n) => {
    if (seen.has(n.id)) fail(`id ${n.id} appears twice`);
    seen.add(n.id);
    if (n.sr) walk(n.sr);
    if (n.tc) walk(n.tc);
    for (const k of n.c || []) walk(k);
  };
  c.forEach(walk);
}

function sortKeys(o) {
  const out = {};
  for (const k of Object.keys(o).sort(byBytes)) out[k] = o[k];
  return out;
}

// byBytes orders strings as Go sorts map keys: by UTF-8 bytes.
function byBytes(a, b) {
  const ea = new TextEncoder().encode(a), eb = new TextEncoder().encode(b);
  for (let i = 0; i < ea.length && i < eb.length; i++) if (ea[i] !== eb[i]) return ea[i] - eb[i];
  return ea.length - eb.length;
}

// goJSON is Go's json.MarshalIndent(v, "", " "): one-space indent, and
// HTML-safe escapes, so its bytes (and hash) equal the Go reducer's.
export function goJSON(v) {
  return JSON.stringify(v, null, 1).replace(/[<>&\u2028\u2029]/g,
    (ch) => '\\u' + ch.charCodeAt(0).toString(16).padStart(4, '0'));
}
