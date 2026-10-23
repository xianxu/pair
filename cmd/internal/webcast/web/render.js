// The viewer's renderer (#460): draws what the model accepted into the
// page frame, an iframe sandboxed without allow-scripts (Principle 3), so no
// script or handler of the recorded page ever runs here. On top of the
// sandbox it neutralizes: handlers are never set, nested frames get no src,
// and every URL goes through resolveAsset (Principle 4: a URL with no asset
// loads nothing). The model holds the recorded values; the DOM holds what is
// drawn, which deliberately differs (rewritten URLs, the hover class), so
// canonical() takes structure and ids from the DOM and values from the model.

import { createPlacement } from './annotation-placement.js';

import { rewriteCSS, resolveURL, rewriteSrcset } from './css.js';

const NS = {
  svg: 'http://www.w3.org/2000/svg',
  math: 'http://www.w3.org/1998/Math/MathML',
  xlink: 'http://www.w3.org/1999/xlink',
  xml: 'http://www.w3.org/XML/1998/namespace',
  xmlns: 'http://www.w3.org/2000/xmlns/',
};

// HOVER is the class standing in for :hover: the page's :hover rules are
// rewritten to it, and it is set on the operator's hovered element and its
// ancestors.
export const HOVER = '__couch-hover';

// The viewer's own rules in every document it draws: with scripts off a
// browser shows <noscript>, which the operator's tab did not.
const VIEWER_CSS = 'noscript{display:none!important}';

// UNCAPTURED: the elements whose content the recording can't hold (pair#513):
// a canvas's pixels, a video's frames, a plugin's output, a frame whose
// document wasn't recorded. The shell labels them where they are drawn.
const UNCAPTURED = new Set(['canvas', 'video', 'object', 'embed', 'iframe', 'frame']);
// LIVE_SIZE_ATTR marks a drawn UNCAPTURED element for its size rule. Their
// size often comes from what isn't recorded: with scripts off a browser
// draws a canvas as its (empty) fallback, and a video sized by its loaded
// metadata has none in the viewer (Safari draws no box at all; pair#513,
// the operator's iPad video). So the viewer gives each the operator's live
// layout box, at zero specificity, as the browser's own sizing would be
// (author CSS still wins).
const LIVE_SIZE_ATTR = 'data-couch-live-size';

// liveSizeRule is the zero-specificity rule that sizes one drawn UNCAPTURED
// element (pair#513), or '' for none. box: its live layout box, if the
// operator's layout has one. Once a layout has arrived, an element without
// a box wasn't rendered (hidden, display:none), so it gets nothing; before
// one, a canvas takes its attributes (default 300x150) and anything else
// waits. The rule never applies to [hidden] (it would beat the UA's
// display:none) and is border-box with the box's aspect ratio, so author CSS
// that sets only the width keeps the operator's height.
export function liveSizeRule(id, kind, box, layoutKnown, attrs) {
  let w, h;
  if (box) [, , w, h] = box;
  else if (!layoutKnown && kind === 'canvas') { w = Number(attrs.width) || 300; h = Number(attrs.height) || 150; }
  if (!(w > 0 && h > 0)) return '';
  return `:where([${LIVE_SIZE_ATTR}="${id}"]:not([hidden])){display:inline-block;box-sizing:border-box;width:${w}px;height:${h}px;aspect-ratio:${w}/${h}}`;
}

// isUncaptured: an element kind whose content the recording can't hold.
export const isUncaptured = (localName) => UNCAPTURED.has(localName);

// Sketch paths carry their immutable recording source rather than note identity.
// Original IDs are useful only within that same document generation and tab.
export function hasSketchSource(annotation, model) {
  const source = annotation.source;
  return !!source && model.docSeg > 0 && source.doc === model.docSeg &&
    source.tab === model.tab?.id && Number.isSafeInteger(source.t) && source.t >= 0 && source.t <= model.t;
}

// URL-bearing attributes, by element (null: any element).
const URL_ATTRS = {
  src: ['img', 'source', 'video', 'audio', 'input', 'embed', 'track', 'image'],
  poster: ['video'],
  background: ['body', 'table', 'td', 'th'],
  href: ['image', 'use', 'feimage'],
};
// Attributes dropped from the drawing: fetch, navigate or submit, or carry
// integrity a rewritten body would fail.
const DROP = new Set(['srcdoc', 'integrity', 'crossorigin', 'ping', 'action', 'formaction', 'http-equiv', 'autofocus', 'autoplay', 'manifest', 'data']);

export class Renderer {
  // frame is the page iframe; resolveAsset(absURL) returns a URL to load
  // the asset from, or nothing.
  // timers ({set, clear}) is the clock a paint hold's deadline runs on.
  constructor(frame, model, resolveAsset, timers = { set: (fn, ms) => setTimeout(fn, ms), clear: (id) => clearTimeout(id) }) {
    this.frame = frame;
    this.model = model;
    this.resolveAsset = resolveAsset;
    this.timers = timers;
    this.reset();
  }

  reset() {
    this.revokeAll();
    this.dom = new Map(); // id -> DOM node
    this.uncapturable = new Set(); // ids of drawn UNCAPTURED elements
    this.ids = new WeakMap(); // DOM node -> id
    this.shadows = new Map(); // host id -> shadow root (closed ones too)
    this.pending = new Map(); // absolute URL -> Map(key -> redraw) waiting for its asset
    this.blobs = new Map(); // link id -> {url, token}
    this.constructed = new Map(); // sheet id -> Map(document -> CSSStyleSheet)
    this.fontFaces = [];
    this.loads = new Set(); // in-flight stylesheet fetches
    this.hovered = [];
    this.dirty = false;
  }

  revokeAll() {
    for (const b of (this.blobs || new Map()).values()) if (b.url) URL.revokeObjectURL(b.url);
  }

  get topDoc() { return this.frame.contentDocument; }

  // apply draws one record the model has just accepted.
  apply(r) {
    const m = this.model;
    switch (r.k) {
      case 'head': case 'seg': this.reset(); this.holdPaint(); this.openDoc(this.topDoc); return;
      case 'doc': return this.drawDoc(r);
      case 'ins': {
        const p = this.dom.get(r.p);
        if (!p) return;
        const node = this.build(r.node, p.ownerDocument || p, this.parentTag(p));
        const after = r.after ? this.dom.get(r.after) : null;
        try { p.insertBefore(node, after ? after.nextSibling : p.firstChild); } catch (e) { this.warn(e); }
        this.afterInsert(r.p);
        return;
      }
      case 'rm': {
        const n = this.dom.get(r.id);
        if (!n) return;
        const parentId = this.ids.get(n.parentNode);
        this.forget(n);
        if (n.nodeType === 11) { n.replaceChildren(); this.dropShadowOf(n); } else n.remove();
        if (parentId) this.afterInsert(parentId);
        return;
      }
      case 'kids': {
        const p = this.dom.get(r.p);
        if (!p) return;
        for (const k of [...p.childNodes]) this.forget(k);
        const doc = p.ownerDocument || p;
        const tag = this.parentTag(p);
        const kids = (r.c || []).map((c) => this.build(c, doc, tag));
        // Emptied first: a document's replaceChildren checks the new
        // doctype against the root element it is about to drop.
        while (p.firstChild) p.firstChild.remove();
        for (const k of kids) { try { p.appendChild(k); } catch (e) { this.warn(e); } }
        this.afterInsert(r.p);
        return;
      }
      case 'attr': {
        const el = this.dom.get(r.id);
        if (!el) return;
        this.drawAttr(el, m.node(r.id), r.name);
        if (el.localName === 'link' || (el.localName === 'style' && r.name === 'media')) this.drawSheetOwner(r.id);
        this.dirty = true;
        return;
      }
      case 'text': {
        const n = this.dom.get(r.id);
        if (!n) return;
        n.data = this.textFor(m.node(r.id), this.parentTag(n.parentNode), n.ownerDocument);
        const owner = this.ids.get(n.parentNode);
        if (owner && m.ownedSheet(owner) !== undefined) this.drawSheetOwner(owner);
        return;
      }
      case 'shadow': {
        const host = this.dom.get(r.host);
        if (host) this.attachShadow(host, r.root, host.ownerDocument);
        return;
      }
      case 'sheet':
        if (r.owner) this.drawSheetOwner(r.owner);
        else for (const [doc, sheet] of this.constructed.get(r.id) || new Map()) this.fillSheet(sheet, r.id, doc);
        return;
      case 'adopt': return this.drawAdopt(r.id);
      case 'font': return this.drawFont(r);
      case 'asset': return this.assetArrived(r.url);
      case 'input': return this.drawInput(r.id);
      case 'scroll': case 'selection': case 'hover': case 'nav': this.dirty = true; return;
    }
  }

  // flush applies what follows the layout: scroll offsets (content
  // inserted after a scroll record can clamp it), the selection and hover.
  flush() {
    this.dirty = false;
    this.checkHold();
    for (const [id, [x, y]] of this.model.scrolls) {
      const n = this.dom.get(id);
      if (!n) continue;
      // Instant: a page's scroll-behavior: smooth would animate the replay
      // behind the operator.
      const to = { left: x, top: y, behavior: 'instant' };
      if (n.nodeType === 9) n.defaultView?.scrollTo(to);
      else if (n.scrollTo) n.scrollTo(to);
    }
    this.drawSelection();
    this.drawHover();
    this.drawLiveSizes();
  }

  // drawLiveSizes sizes each drawn UNCAPTURED element as the operator's
  // was (liveSizeRule), in the live-size sheet of the tree it is in: its
  // document or its shadow root (document rules don't pierce shadow trees).
  drawLiveSizes() {
    const rules = new Map(); // root -> rule text
    const layoutKnown = this.model.boxesT > 0;
    for (const id of this.uncapturable) {
      const n = this.dom.get(id);
      if (!n || !n.isConnected) { this.uncapturable.delete(id); continue; }
      const rule = liveSizeRule(id, n.localName, this.model.boxes.get(id), layoutKnown,
        { width: n.getAttribute('width'), height: n.getAttribute('height') });
      if (!rule) continue;
      const root = n.getRootNode();
      rules.set(root, (rules.get(root) || '') + rule);
    }
    for (const root of new Set([...rules.keys(), ...this.sized || []])) {
      const s = this.liveSheet(root);
      const text = rules.get(root) || '';
      if (s.__text !== text) { s.replaceSync(text); s.__text = text; }
      if (!root.adoptedStyleSheets.includes(s)) this.adoptLive(root);
    }
    this.sized = new Set(rules.keys());
  }

  // liveSheet is a document's or shadow root's live-size sheet.
  liveSheet(root) {
    if (!root.__couchLive) {
      const win = (root.nodeType === 9 ? root : root.ownerDocument).defaultView;
      Object.defineProperty(root, '__couchLive', { value: new win.CSSStyleSheet() });
    }
    return root.__couchLive;
  }

  // adoptLive puts a root's live-size sheet before the page's own sheets
  // (after the viewer's), so on a tie the page's CSS wins.
  adoptLive(root) {
    const live = this.liveSheet(root);
    const rest = root.adoptedStyleSheets.filter((x) => x !== live && x !== root.__couchSheet);
    const viewer = root.nodeType === 9 ? [this.viewerSheet(root)] : [];
    try { root.adoptedStyleSheets = [...viewer, live, ...rest]; } catch (e) { this.warn(e); }
  }

  // uncaptured lists the drawn elements whose content the recording can't
  // hold, by id and kind; a frame counts only when its document wasn't
  // recorded (nothing drawn in it).
  uncaptured() {
    const out = [];
    for (const id of this.uncapturable) {
      const n = this.dom.get(id);
      if (!n || !n.isConnected) continue;
      const kind = n.localName;
      if ((kind === 'iframe' || kind === 'frame') && n.contentDocument && this.ids.has(n.contentDocument)) continue; // a recorded document is drawn in it
      out.push({ id, kind });
    }
    return out;
  }

  warn(e) { if (typeof console !== 'undefined') console.warn('couch viewer:', e); }

  // ---- documents ----

  // openDoc empties a document and leaves it in standards mode, with the
  // viewer's own sheet adopted.
  openDoc(doc) {
    if (!doc) return;
    doc.open();
    doc.write('<!DOCTYPE html>');
    doc.close();
    doc.replaceChildren();
    doc.defaultView?.scrollTo({ left: 0, top: 0, behavior: 'instant' }); // a window keeps its scroll across documents; a segment starts at the top
    try { doc.fonts.clear(); } catch {}
    doc.adoptedStyleSheets = [this.viewerSheet(doc), this.liveSheet(doc)];
    // An image, stylesheet or font arriving changes the layout, and a
    // scroll applied to the shorter page was clamped: draw again, through
    // whichever renderer draws this frame now. doc.open() erased the
    // document's listeners (#475); doc.fonts keeps its own.
    const changed = () => { const r = doc.__couchRenderer; r.dirty = true; r.onDirty(); };
    doc.addEventListener('load', changed, true);
    if (!doc.__couchRenderer) doc.fonts.addEventListener('loadingdone', changed);
    doc.__couchRenderer = this;
  }

  // onDirty is called when the drawing needs a flush outside a record (an
  // image, stylesheet or font loaded; a held paint released). It flushes
  // before the next paint; the live shell replaces it with its own paint.
  onDirty() {
    const win = this.frame.ownerDocument && this.frame.ownerDocument.defaultView;
    if (win) win.requestAnimationFrame(() => this.flush());
  }

  viewerSheet(doc) {
    if (!doc.__couchSheet) {
      const s = new doc.defaultView.CSSStyleSheet();
      s.replaceSync(VIEWER_CSS);
      Object.defineProperty(doc, '__couchSheet', { value: s });
    }
    return doc.__couchSheet;
  }

  drawDoc(r) {
    let doc;
    if (!r.host) { doc = this.topDoc; if (this.hold) this.hold.drawn = true; }
    else {
      const host = this.dom.get(r.host);
      doc = host && host.contentDocument;
      if (!doc) return; // a frame not in a document has no content to draw into
      const old = this.ids.get(doc);
      if (old) this.forget(doc);
      this.openDoc(doc);
    }
    this.map(doc, r.doc);
    for (const k of r.tree.c || []) {
      try { doc.appendChild(this.build(k, doc, '')); } catch (e) { this.warn(e); }
    }
    this.afterInsert(r.doc);
    this.checkHold();
  }

  // docURL is the URL a document's relative URLs resolve against.
  docURL(doc) {
    const id = this.ids.get(doc);
    const n = id && this.model.node(id);
    if (n === this.model.top) return this.model.url;
    return (n && n.url) || this.model.url;
  }

  // ---- building ----

  map(node, id) { this.dom.set(id, node); this.ids.set(node, id); }

  parentTag(p) { return p && p.nodeType === 1 ? p.localName : ''; }

  build(n, doc, parentTag) {
    let node;
    switch (n.y) {
      case 'doctype': node = doc.implementation.createDocumentType(n.n || 'html', '', ''); break;
      case 'text': node = doc.createTextNode(this.textFor(this.model.node(n.id) || n, parentTag, doc)); break;
      case 'comment': node = doc.createComment(n.v || ''); break;
      case 'el': node = this.element(n, doc); break;
      default: node = doc.createDocumentFragment(); break; // unreachable for well-formed trees
    }
    this.map(node, n.id);
    if (n.y !== 'el') return node;
    if (n.tc && node.content) {
      this.map(node.content, n.tc.id);
      for (const k of n.tc.c || []) node.content.appendChild(this.build(k, doc, ''));
    }
    if (n.sr) this.attachShadow(node, n.sr, doc);
    for (const k of n.c || []) node.appendChild(this.build(k, doc, node.localName));
    if (node.localName === 'style' || node.localName === 'link') this.drawSheetOwner(n.id, node);
    if (this.model.inputs.has(n.id)) this.drawInput(n.id, node);
    return node;
  }

  element(n, doc) {
    let el;
    try {
      el = n.ns && NS[n.ns] ? doc.createElementNS(NS[n.ns], n.n) : doc.createElement(n.n);
    } catch {
      el = doc.createElement('span'); // a tag the DOM refuses; its values stay in the model
    }
    if (el.localName === 'iframe' || el.localName === 'frame') el.setAttribute('sandbox', 'allow-same-origin');
    if (isUncaptured(el.localName)) { this.uncapturable.add(n.id); el.setAttribute(LIVE_SIZE_ATTR, String(n.id)); }
    const model = this.model.node(n.id) || n;
    for (const name of Object.keys(model.a || {})) this.drawAttr(el, model, name);
    return el;
  }

  attachShadow(host, sr, doc) {
    let root;
    try { root = host.attachShadow({ mode: sr.mode === 'closed' ? 'closed' : 'open' }); } catch { root = doc.createDocumentFragment(); }
    this.shadows.set(this.ids.get(host), root);
    this.map(root, sr.id);
    for (const k of sr.c || []) root.appendChild(this.build(k, doc, ''));
  }

  dropShadowOf(root) {
    for (const [host, r] of this.shadows) if (r === root) this.shadows.delete(host);
  }

  // afterInsert redraws what depends on a container's children: a style
  // element's owned text, and new frames' adoptions.
  afterInsert(id) {
    const n = this.dom.get(id);
    if (n && n.localName === 'style') this.drawSheetOwner(id);
    this.dirty = true;
  }

  textFor(n, parentTag, doc) {
    const v = n.v || '';
    if (parentTag !== 'style') return v;
    return this.css(v, this.docURL(doc), `text:${n.id}`, () => this.redrawText(n.id));
  }

  redrawText(id) {
    const node = this.dom.get(id), n = this.model.node(id);
    if (node && n) node.data = this.textFor(n, this.parentTag(node.parentNode), node.ownerDocument);
  }

  // css rewrites a stylesheet's text for the frame; URLs whose asset hasn't
  // arrived are remembered under key, and redraw runs when it does.
  // own: the text came from the proxy, whose URLs (already pointing at
  // the viewer's origin) are kept.
  css(text, base, key, redraw, own = false) {
    const origin = typeof location !== 'undefined' ? location.origin + '/' : null;
    return rewriteCSS(text, base, (abs) => (own && origin && abs.startsWith(origin) ? abs : this.asset(abs, key, redraw)), HOVER);
  }

  // asset resolves an absolute URL, remembering a miss.
  asset(abs, key, redraw) {
    const u = this.resolveAsset(abs);
    if (u) return u;
    if (key) {
      if (!this.pending.has(abs)) this.pending.set(abs, new Map());
      this.pending.get(abs).set(key, redraw);
    }
    return '';
  }

  assetArrived(url) {
    const waiting = this.pending.get(url);
    if (!waiting) return;
    this.pending.delete(url);
    for (const redraw of waiting.values()) redraw();
  }

  // ---- attributes ----

  drawAttr(el, n, name) {
    const v = n && n.a ? n.a[name] : undefined;
    const lower = name.toLowerCase();
    const tag = el.localName;
    if (lower.startsWith('on') || DROP.has(lower)) return;
    if ((tag === 'iframe' || tag === 'frame') && (lower === 'src' || lower === 'sandbox')) return;
    if (tag === 'script' && lower === 'src') return;
    if (tag === 'base' && lower === 'href') return;
    if (tag === 'link' && lower === 'href') return; // drawn by drawSheetOwner
    const redraw = () => { const m = this.model.node(n.id); if (m && this.dom.get(n.id) === el) this.drawAttr(el, m, name); };
    const key = `attr:${n.id}:${name}`;
    let out = v;
    if (v !== undefined) {
      const base = this.docURL(el.ownerDocument);
      if (lower === 'style') out = this.css(v, base, key, redraw);
      else if (lower === 'srcset') out = rewriteSrcset(v, base, (abs) => this.asset(abs, key, redraw)) || undefined; // no candidate yet: none, as with src
      else if (isURLAttr(tag, lower)) {
        if (/^\s*#/.test(v)) out = v; // a fragment in this document (SVG <use href="#x">)
        else {
          const abs = resolveURL(v, base);
          out = abs && /^data:/i.test(abs) ? abs : (abs && this.asset(abs, key, redraw));
        }
        if (!out) out = undefined;
      }
    }
    if (lower === 'class' && this.hovered.includes(el)) out = `${v || ''} ${HOVER}`.trim();
    // An image with nothing to load is drawn as one the tab hasn't loaded
    // yet (a lazy image below the fold: an empty box), not as a broken one
    // showing its alt text; the alt returns with the asset.
    if (tag === 'img' && lower === 'alt' && !el.hasAttribute('src') && !el.hasAttribute('srcset')) out = undefined;
    setAttr(el, name, out);
    if (tag === 'img' && (lower === 'src' || lower === 'srcset') && n.a && 'alt' in n.a) this.drawAttr(el, n, 'alt');
  }

  // ---- styles ----

  // drawSheetOwner draws a <style>'s or <link>'s stylesheet: an owned
  // sheet's text when the model has one, else the element's own.
  drawSheetOwner(id, el = this.dom.get(id)) {
    const n = this.model.node(id);
    if (!el || !n) return;
    const owned = this.model.ownedSheet(id);
    const key = `sheet:${id}`, redraw = () => this.drawSheetOwner(id);
    if (el.localName === 'style') {
      if (owned === undefined) return;
      const text = this.css(owned, this.docURL(el.ownerDocument), key, redraw);
      const texts = [...el.childNodes].filter((c) => c.nodeType === 3);
      if (texts.length === 0) el.appendChild(el.ownerDocument.createTextNode(text)); // untracked: no id
      else texts.forEach((t, i) => { t.data = i === 0 ? text : ''; });
      return;
    }
    if (el.localName !== 'link') return;
    const rel = (n.a && n.a.rel || '').toLowerCase().split(/\s+/);
    if (!rel.includes('stylesheet')) return;
    if (owned !== undefined) return this.setLinkCSS(id, el, this.css(owned, this.docURL(el.ownerDocument), key, redraw));
    const href = n.a && n.a.href;
    const abs = href && resolveURL(href, this.docURL(el.ownerDocument));
    const proxied = abs && this.asset(abs, key, redraw);
    if (!proxied) return this.setLinkCSS(id, el, null);
    const token = {};
    this.blobs.set(id, { ...(this.blobs.get(id) || {}), token });
    const load = fetch(proxied).then((res) => (res.ok ? res.text() : '')).then((text) => {
      const b = this.blobs.get(id);
      if (!b || b.token !== token || this.dom.get(id) !== el) return;
      const base = new URL(proxied, location.href).href;
      this.setLinkCSS(id, el, this.css(text, base, key, redraw, true));
    }).catch((e) => this.warn(e)).finally(() => this.loads.delete(load));
    this.loads.add(load);
  }

  setLinkCSS(id, el, text) {
    const old = this.blobs.get(id);
    if (old && old.url) URL.revokeObjectURL(old.url);
    if (text === null) { this.blobs.delete(id); el.removeAttribute('href'); return; }
    const url = URL.createObjectURL(new Blob([text], { type: 'text/css' }));
    this.blobs.set(id, { url, token: old && old.token });
    sheetFailed.delete(el);
    el.addEventListener('load', () => this.checkHold(), { once: true });
    el.addEventListener('error', () => { sheetFailed.add(el); this.checkHold(); }, { once: true });
    el.setAttribute('href', url);
  }

  // fillSheet sets a constructed sheet's text, for the document it is
  // adopted in. (Its url()s are absolute by the format; the base is for a
  // page that broke that.)
  fillSheet(sheet, id, doc) {
    const sh = this.model.sheets.get(id);
    if (!sh) return;
    try { sheet.replaceSync(this.css(sh.text, this.docURL(doc), `csheet:${id}`, () => this.fillSheet(sheet, id, doc))); } catch (e) { this.warn(e); }
  }

  constructedSheet(id, doc) {
    if (!this.constructed.has(id)) this.constructed.set(id, new Map());
    const per = this.constructed.get(id);
    if (!per.has(doc)) {
      const s = new doc.defaultView.CSSStyleSheet();
      per.set(doc, s);
      this.fillSheet(s, id, doc);
    }
    return per.get(doc);
  }

  drawAdopt(id) {
    const target = this.dom.get(id);
    if (!target) return;
    const doc = target.nodeType === 9 ? target : target.ownerDocument;
    const sheets = (this.model.adopted.get(id) || []).map((sid) => this.constructedSheet(sid, doc));
    try { target.adoptedStyleSheets = target.nodeType === 9 ? [this.viewerSheet(doc), this.liveSheet(target), ...sheets] : [this.liveSheet(target), ...sheets]; } catch (e) { this.warn(e); }
  }

  drawFont(r) {
    const doc = this.topDoc;
    const abs = resolveURL(r.src, this.model.url);
    const u = abs && this.asset(abs, `font:${this.fontFaces.length}:${abs}`, () => this.drawFont(r));
    if (!u) return;
    try {
      const desc = {};
      if (r.style) desc.style = r.style;
      if (r.weight) desc.weight = r.weight;
      if (r.stretch) desc.stretch = r.stretch;
      if (r.range) desc.unicodeRange = r.range;
      const face = new doc.defaultView.FontFace(r.family, `url("${new URL(u, location.href).href}")`, desc);
      doc.fonts.add(face);
      this.fontFaces.push(face);
      face.load().catch(() => {});
    } catch (e) { this.warn(e); }
  }

  // ---- paint holding (#475) ----
  //
  // A browser keeps showing the old page until a new document's
  // render-blocking stylesheets have loaded, and the recording says when
  // their bytes arrived, which can be after the document did. So the
  // renderer draws a new top document into a second frame placed before
  // the first in document order, so that the first, keeping the previous
  // paint, covers it (and the shell's overlay, later still, covers both).
  // The first is hidden once the document's stylesheets are bound, or
  // HOLD_MS after the hold began. The new frame is never hidden itself:
  // Chrome throttles a hidden frame's rendering, which starts its
  // animations late. #page is always the frame with the current document
  // (what canonical(), boxOf() and the shell's layout read).
  //
  // The hold is one state on the frame pair: pair.hold is null (the
  // current frame shows its document) or {owner, drawn, cancel}: owner is
  // the renderer drawing pair.current, drawn whether its top document has
  // been drawn, cancel stops its deadline. A renderer that starts a hold
  // while one is in flight (a new segment, a resync, a seek) takes it over:
  // the previous paint stays, and the old deadline is cancelled.

  // holdPaint begins a hold for the top document about to be drawn.
  holdPaint() {
    const pair = framePair(this.frame);
    if (pair.hold) pair.hold.cancel();
    else {
      const held = pair.current, next = pair.frames.find((f) => f !== held);
      held.before(next); // under the held paint; a moved frame starts with an empty document
      next.style.visibility = '';
      next.id = held.id;
      held.removeAttribute('id');
      pair.current = next;
    }
    this.frame = pair.current;
    const timer = this.timers.set(() => this.release(), HOLD_MS);
    pair.hold = { owner: this, drawn: false, cancel: () => this.timers.clear(timer) };
  }

  // hold is this renderer's hold, or null.
  get hold() {
    const h = this.frame.__couchPair && this.frame.__couchPair.hold;
    return h && h.owner === this ? h : null;
  }

  get holding() { return !!this.hold; }

  // checkHold releases the hold once the document is drawn and every
  // stylesheet that blocks its rendering is bound (or failed, as a
  // browser stops waiting for one).
  checkHold() {
    const h = this.hold;
    if (h && h.drawn && blockingSheets(this.topDoc).every(sheetSettled)) this.release();
  }

  // bound resolves once the stylesheets whose bytes have arrived are bound
  // (the proxy fetch and the sheet's load), or after HOLD_MS, so a capture
  // shows what a viewer shows once the stylesheet it already has applies.
  async bound() {
    const settled = (async () => {
      while (this.loads.size) await Promise.allSettled([...this.loads]);
      await Promise.allSettled(blockingSheets(this.topDoc).filter((l) => !sheetSettled(l) && l.getAttribute('href'))
        .map((l) => new Promise((ok) => { l.addEventListener('load', ok, { once: true }); l.addEventListener('error', ok, { once: true }); })));
    })();
    let timer;
    await Promise.race([settled, new Promise((ok) => { timer = this.timers.set(ok, HOLD_MS); })]);
    this.timers.clear(timer);
    this.checkHold();
  }

  // release ends this renderer's hold: the held paint is hidden and
  // emptied, and the drawing redrawn.
  release() {
    const h = this.hold;
    if (!h) return;
    h.cancel();
    const pair = this.frame.__couchPair;
    pair.hold = null;
    hideHeld(pair);
    this.dirty = true;
    this.onDirty();
  }

  // ---- observer state ----

  drawInput(id, el = this.dom.get(id)) {
    const s = this.model.inputs.get(id);
    if (!el || !s) return;
    try {
      if (typeof s.v === 'string' && el.type !== 'file') el.value = s.v;
      if (typeof s.checked === 'boolean') el.checked = s.checked;
      if (Array.isArray(s.sel) && el.options) [...el.options].forEach((o, i) => { o.selected = s.sel.includes(i); });
    } catch (e) { this.warn(e); }
  }

  drawSelection() {
    const sel = this.model.selection, doc = this.topDoc;
    const s = doc && doc.getSelection && doc.getSelection();
    if (!s) return;
    const a = sel && sel.anchor && this.dom.get(sel.anchor[0]), f = sel && sel.focus && this.dom.get(sel.focus[0]);
    try {
      if (a && f && a.ownerDocument === doc && f.ownerDocument === doc) s.setBaseAndExtent(a, sel.anchor[1], f, sel.focus[1]);
      else s.removeAllRanges();
    } catch { s.removeAllRanges(); }
  }

  // drawHover puts the hover class on the operator's hovered element and
  // its ancestors (across shadow roots), as :hover would match.
  drawHover() {
    const want = [];
    for (let n = this.dom.get(this.model.hover); n; ) {
      if (n.nodeType === 1) want.push(n);
      n = n.parentNode && n.parentNode.nodeType === 11 && n.parentNode.host ? n.parentNode.host : n.parentNode;
    }
    for (const el of this.hovered) if (!want.includes(el)) el.classList.remove(HOVER);
    for (const el of want) el.classList.add(HOVER);
    this.hovered = want;
  }

  // ---- forgetting ----

  // forget drops a DOM subtree's ids (children, shadow roots, template
  // content and frame documents) and its blob URLs.
  forget(node) {
    const id = this.ids.get(node);
    if (id !== undefined) {
      this.ids.delete(node);
      if (this.dom.get(id) === node) this.dom.delete(id);
      const b = this.blobs.get(id);
      if (b) { if (b.url) URL.revokeObjectURL(b.url); this.blobs.delete(id); }
      const root = this.shadows.get(id);
      if (root) { this.shadows.delete(id); this.forget(root); }
    }
    if (node.content && node.localName === 'template') this.forget(node.content);
    if (node.localName === 'iframe' || node.localName === 'frame') {
      try { if (node.contentDocument) this.forget(node.contentDocument); } catch {}
    }
    for (const k of node.childNodes) this.forget(k);
  }

  // ---- reading back ----

  // canonical is State.Canonical() of what was drawn: the tree's shape and
  // ids from the DOM, recorded values from the model.
  canonical() {
    const m = this.model;
    const doc = this.topDoc;
    const id = doc && this.ids.get(doc);
    if (!m.top || !id) return m.canonicalState(null);
    const asModel = (node) => {
      const i = this.ids.get(node);
      return m.node(i) || { id: i, y: 'unknown', kids: [] };
    };
    const kids = (n) => {
      const node = this.dom.get(n.id);
      if (!node) return [];
      const holder = node.localName === 'template' && n.y === 'frag' ? node.content : node;
      return [...holder.childNodes].filter((c) => this.ids.has(c)).map(asModel);
    };
    const sub = (n, which) => {
      const node = this.dom.get(n.id);
      if (!node || n.y !== 'el') return null;
      let s = null;
      if (which === 'shadow') s = this.shadows.get(n.id);
      if (which === 'content') s = node.localName === 'template' ? node.content : null;
      if (which === 'frame') { try { s = node.contentDocument; } catch {} }
      return s && this.ids.has(s) ? asModel(s) : null;
    };
    return m.canonicalState(m.canonicalNode(asModel(doc), kids, sub));
  }

  // ready resolves when stylesheets, fonts and images have loaded.
  async ready() {
    while (this.loads.size) await Promise.allSettled([...this.loads]);
    const docs = [this.topDoc, ...[...this.dom.values()].filter((n) => n.nodeType === 9 && n !== this.topDoc)];
    const waits = [];
    for (const d of docs) {
      if (!d) continue;
      waits.push(d.fonts.ready);
      for (const img of d.images) if (!img.complete) waits.push(new Promise((ok) => { img.addEventListener('load', ok, { once: true }); img.addEventListener('error', ok, { once: true }); }));
      for (const l of d.querySelectorAll('link[rel~=stylesheet][href]')) if (!l.sheet) waits.push(new Promise((ok) => { l.addEventListener('load', ok, { once: true }); l.addEventListener('error', ok, { once: true }); }));
    }
    await Promise.race([Promise.allSettled(waits), new Promise((ok) => setTimeout(ok, 5000))]);
  }

  // Annotation boxes use live evidence; replay's boxOf remains historical.
  placementOf(note, options) {
    if (!this.placeAnnotation) this.placeAnnotation = createPlacement({
      document: () => this.topDoc, route: () => this.model.url,
      original: (id, note) => hasSketchSource(note, this.model) ||
        this.model.docSeg > 0 && this.model.notes?.some(entry => entry.note.id === note.id && entry.seg >= this.model.docSeg)
        ? this.dom.get(id) : null, shadow: n => this.shadows.get(this.ids.get(n)) || n.shadowRoot,
      onChange: () => this.placementChanged?.(),
    });
    return this.placeAnnotation(note, options);
  }

  // boxOf is a node's box in top-page CSS px from this frame's own layout:
  // the operator's layout, unscaled (the scale is on the frame element).
  boxOf(id) {
    const n = this.dom.get(id);
    if (!n || (n.nodeType !== 1 && n.nodeType !== 3)) return null;
    let r;
    if (n.nodeType === 1) r = n.getBoundingClientRect();
    else { const range = n.ownerDocument.createRange(); range.selectNodeContents(n); r = range.getBoundingClientRect(); }
    let x = r.left, y = r.top;
    for (let win = n.ownerDocument.defaultView; win && win.frameElement && win !== this.frame.contentWindow; win = win.frameElement.ownerDocument.defaultView) {
      const fe = win.frameElement, fr = fe.getBoundingClientRect();
      x += fr.left + fe.clientLeft; y += fr.top + fe.clientTop;
    }
    const top = this.frame.contentWindow;
    return [x + top.scrollX, y + top.scrollY, r.width, r.height];
  }

  // nodeAt is the id of the drawn node at a top-page point, descending into
  // frames and shadow roots (closed ones too), or 0.
  nodeAt(pageX, pageY) {
    const top = this.frame.contentWindow;
    let doc = this.topDoc, x = pageX - top.scrollX, y = pageY - top.scrollY;
    let el = doc && doc.elementFromPoint(x, y);
    for (let hops = 0; el && hops < 32; hops++) {
      const root = this.shadows.get(this.ids.get(el));
      if (root && root.elementFromPoint) {
        const inner = root.elementFromPoint(x, y);
        if (inner && inner !== el) { el = inner; continue; }
      }
      if ((el.localName === 'iframe' || el.localName === 'frame') && el.contentDocument) {
        const fr = el.getBoundingClientRect();
        x -= fr.left + el.clientLeft; y -= fr.top + el.clientTop;
        const inner = el.contentDocument.elementFromPoint(x, y);
        if (inner) { el = inner; continue; }
      }
      break;
    }
    for (let n = el; n; n = n.parentNode || n.host) if (this.ids.has(n)) return this.ids.get(n);
    return 0;
  }
}

// HOLD_MS is the longest a new document's paint waits for its stylesheets:
// a stylesheet that never arrives (over the asset budget, a failed fetch)
// leaves the page drawn without it.
export const HOLD_MS = 500;

// currentFrame is the frame holding the current document of the page frame
// a renderer was given: frame itself until a paint hold swaps it (#475).
export function currentFrame(frame) {
  return frame.__couchPair ? frame.__couchPair.current : frame;
}

// blankFrames empties the page frame and drops a hold in flight: a fresh
// viewer holds no previous paint.
export function blankFrames(frame) {
  const pair = frame.__couchPair;
  if (pair && pair.hold) { pair.hold.cancel(); pair.hold = null; }
  if (pair) hideHeld(pair);
  try { currentFrame(frame).contentDocument.replaceChildren(); } catch {}
}

// hideHeld hides and empties the frame not showing the current document:
// nothing runs in it.
function hideHeld(pair) {
  for (const f of pair.frames) {
    if (f === pair.current) continue;
    f.style.visibility = 'hidden';
    try { f.contentDocument.replaceChildren(); } catch {}
  }
}

// sheetFailed holds the <link>s whose stylesheet failed to load; a browser
// stops blocking on them.
const sheetFailed = new WeakSet();
const sheetSettled = (l) => !!l.sheet || sheetFailed.has(l);

// framePair is the page frame and its sibling holding the previous paint,
// created beside it the first time, both placed at the box's origin and
// opaque: a tab's canvas is white unless the page paints it, a frame's
// transparent, and the held paint must cover the new document.
function framePair(frame) {
  if (!frame.__couchPair) {
    const back = frame.cloneNode(false);
    back.removeAttribute('id');
    back.style.visibility = 'hidden';
    for (const f of [frame, back]) Object.assign(f.style, { position: 'absolute', left: '0', top: '0', background: '#fff' });
    frame.after(back);
    const pair = { frames: [frame, back], current: frame, hold: null };
    for (const f of pair.frames) Object.defineProperty(f, '__couchPair', { value: pair });
  }
  return frame.__couchPair;
}

// blockingSheets are a document's render-blocking stylesheets, as a
// browser counts them: <link rel=stylesheet>, not alternate or disabled,
// whose media matches.
function blockingSheets(doc) {
  if (!doc) return [];
  const win = doc.defaultView;
  return [...doc.querySelectorAll('link[rel~=stylesheet i]')].filter((l) => {
    const rel = (l.getAttribute('rel') || '').toLowerCase().split(/\s+/);
    if (rel.includes('alternate') || l.hasAttribute('disabled')) return false;
    const media = l.getAttribute('media');
    return !media || !win || win.matchMedia(media).matches;
  });
}

// isURLAttr: tags compare in lower case, as URL_ATTRS lists them (an SVG
// localName keeps its case: feImage).
function isURLAttr(tag, name) {
  const tags = URL_ATTRS[name === 'xlink:href' ? 'href' : name];
  return !!tags && tags.includes(tag.toLowerCase());
}

// setAttr sets (or, for undefined, removes) an attribute, with the XML
// namespaces a name's prefix asks for; a name the DOM refuses is skipped.
function setAttr(el, name, v) {
  const [prefix, local] = name.includes(':') ? name.split(':', 2) : ['', name];
  const ns = prefix === 'xlink' ? NS.xlink : prefix === 'xml' ? NS.xml : (prefix === 'xmlns' || name === 'xmlns') ? NS.xmlns : null;
  try {
    if (v === undefined) {
      if (ns) el.removeAttributeNS(ns, local); else el.removeAttribute(name);
    } else if (ns) el.setAttributeNS(ns, name, v);
    else el.setAttribute(name, v);
  } catch {}
}
