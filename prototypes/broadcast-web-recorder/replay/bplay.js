// Prototype B's replayer: rebuild the CDP DOM stream in a sandboxed, script-free frame and apply changes by node id.
// window.bwLoad(events) then await window.bwSeek(t) renders the page as of t (epoch ms).
(function () {
  const SVG = 'http://www.w3.org/2000/svg', MATHML = 'http://www.w3.org/1998/Math/MathML';
  let EV = [], host, stats;

  async function freshFrame(parent, w, h) {
    const f = parent.ownerDocument.createElement('iframe');
    f.setAttribute('sandbox', 'allow-same-origin');
    f.style.cssText = `border:0;width:${w}px;height:${h}px;display:block`;
    f.srcdoc = '<!doctype html><html><head></head><body></body></html>';
    const loaded = new Promise(r => f.addEventListener('load', r, { once: true }));
    parent.appendChild(f); await loaded; return f;
  }

  // One replay state per CDP session (top target, each out-of-process iframe target).
  function Session(sid) { return { sid, nodes: new Map(), backend: new Map(), doc: null, frameEl: null, sheets: new Map(), ua: new Set(), scrolls: new Map() }; }

  function build(S, n, doc, inSvg) {
    let el;
    if (n.nodeType === 1) {
      const name = n.localName || n.nodeName.toLowerCase();
      const svg = n.isSVG || name === 'svg' || (inSvg && name !== 'foreignobject');
      el = name === 'math' ? doc.createElementNS(MATHML, name) : svg ? doc.createElementNS(SVG, n.localName || n.nodeName) : doc.createElement(name === 'script' ? 'script' : name);
      const a = n.attributes || [];
      for (let i = 0; i < a.length; i += 2) setAttr(el, a[i], a[i + 1]);
      if (name === 'script') el.setAttribute('type', 'bw/inert');
      if (name === 'iframe' || name === 'frame') { el.removeAttribute('src'); el.removeAttribute('srcdoc'); el.setAttribute('sandbox', 'allow-same-origin'); el.__bwFrame = n; }
      if (n.shadowRoots) for (const r of n.shadowRoots) attachShadow(S, el, r, doc, svg);
      if (n.templateContent) buildChildrenInto(S, el.content, n.templateContent.children || [], doc, false), S.nodes.set(n.templateContent.nodeId, el.content);
      buildChildrenInto(S, el, n.children || [], doc, svg && name !== 'foreignobject');
    } else if (n.nodeType === 3) el = doc.createTextNode(n.nodeValue);
    else if (n.nodeType === 8) el = doc.createComment(n.nodeValue);
    else if (n.nodeType === 11) { el = doc.createDocumentFragment(); buildChildrenInto(S, el, n.children || [], doc, inSvg); }
    else return null; // doctype, processing instructions
    S.nodes.set(n.nodeId, el); if (n.backendNodeId) S.backend.set(n.backendNodeId, el);
    if (n.adoptedStyleSheets) el.__bwAdopted = n.adoptedStyleSheets;
    return el;
  }
  // Captured stylesheet bodies replace origin fetches (blob URLs stand in for the asset proxy).
  const assetBlobs = new Map(); let curBase = '';
  function assetFor(el, v) { try { return assetBlobs.get(new URL(v, el.__bwBase || curBase).href); } catch { return null; } }
  function setAttr(el, k, v) {
    if (/^on/i.test(k)) return;
    if (k === 'href' && el.localName === 'link') { el.__bwHref = v; el.__bwBase = curBase; const b = assetFor(el, v); if (b) v = b; }
    if ((k === 'href' || k === 'src' || k === 'action' || k === 'xlink:href') && /^\s*javascript:/i.test(v)) return;
    try { if (k.startsWith('xlink:')) el.setAttributeNS('http://www.w3.org/1999/xlink', k, v); else el.setAttribute(k, v); } catch {}
  }
  function attachShadow(S, el, r, doc, svg) {
    if (r.shadowRootType === 'user-agent') { markUA(S, r); return; }
    let root; try { root = el.attachShadow({ mode: 'open' }); } catch { return; }
    S.nodes.set(r.nodeId, root);
    if (r.adoptedStyleSheets) root.__bwAdopted = r.adoptedStyleSheets;
    buildChildrenInto(S, root, r.children || [], doc, svg);
  }
  // The browser's own shadow trees (input editors, details slots) are not page content: their ids are ignored.
  function markUA(S, n) { S.ua.add(n.nodeId); for (const c of n.children || []) markUA(S, c); for (const r of n.shadowRoots || []) markUA(S, r); }
  function buildChildrenInto(S, parent, kids, doc, svg) { for (const k of kids) { const c = build(S, k, doc, svg); if (c) parent.appendChild(c); } }

  function snapshot(S, root, frameEl) {
    if (!S.keep) { S.nodes.clear(); S.backend.clear(); }
    const doc = frameEl.contentDocument; S.doc = doc; S.frameEl = frameEl;
    S.nodes.set(root.nodeId, doc);
    if (root.adoptedStyleSheets) doc.__bwAdopted = root.adoptedStyleSheets;
    const html = (root.children || []).find(c => c.nodeType === 1);
    const base = doc.createElement('base'); base.href = curBase = root.baseURL || root.documentURL;
    if (html) { const el = build(S, html, doc, false); doc.replaceChild(el, doc.documentElement);
      const head = el.querySelector('head') || el; head.insertBefore(base, head.firstChild); }
    hydrateFrames(S);
    applyAdopted(S);
  }
  // Same-process iframes come inside the tree (contentDocument); out-of-process ones arrive as their own session.
  function hydrateFrames(S) {
    const frames = [...S.doc.querySelectorAll('iframe,frame')];
    S.doc.querySelectorAll('*').forEach(e => { if (e.shadowRoot) frames.push(...e.shadowRoot.querySelectorAll('iframe')); });
    for (const f of frames) {
      const n = f.__bwFrame; if (!n || f.__bwHydrated) continue; f.__bwHydrated = true;
      if (!f.contentDocument) continue;
      f.contentDocument.open(); f.contentDocument.write('<!doctype html><html><head></head><body></body></html>'); f.contentDocument.close();
      if (n.contentDocument) snapshot(Object.assign(Session(S.sid), { nodes: S.nodes, backend: S.backend, sheets: S.sheets, keep: true }), n.contentDocument, f);
      if (n.frameId) host.frameOwners.set(n.frameId, f);
    }
  }
  function applyAdopted(S) {
    const all = [S.doc, ...[...S.nodes.values()].filter(x => x instanceof ShadowRoot)];
    for (const d of all) if (d.__bwAdopted) {
      const win = S.doc.defaultView;
      d.adoptedStyleSheets = d.__bwAdopted.map(id => { let sh = S.sheets.get(id); if (!sh) { sh = new win.CSSStyleSheet(); S.sheets.set(id, sh); } return sh; });
    }
  }

  function apply(S, e) {
    const N = id => S.nodes.get(id);
    const ua = id => S.ua.has(id);
    if (ua(e.parentNodeId) || ua(e.parentId) || ua(e.nodeId) || ua(e.hostId)) {
      if (e.node) markUA(S, e.node); if (e.nodes) e.nodes.forEach(n => markUA(S, n)); stats.ua++; return; }
    switch (e.type) {
      case 'childNodeInserted': { const p = N(e.parentNodeId); if (!p) return miss(e); const c = build(S, e.node, S.doc, p.namespaceURI === SVG);
        if (!c) return; const prev = e.previousNodeId ? N(e.previousNodeId) : null; p.insertBefore(c, prev ? prev.nextSibling : p.firstChild); hydrateFrames(S); return; }
      case 'childNodeRemoved': { const c = N(e.nodeId); if (!c) return miss(e); c.remove(); S.nodes.delete(e.nodeId); return; }
      case 'setChildNodes': { const p = N(e.parentId); if (!p) return miss(e); const kids = e.nodes.map(n => build(S, n, S.doc, p.namespaceURI === SVG)).filter(Boolean);
        (p.content && p instanceof S.doc.defaultView.HTMLTemplateElement ? p.content : p).replaceChildren(...kids); hydrateFrames(S); return; }
      case 'attributeModified': { const el = N(e.nodeId); if (!el) return miss(e); setAttr(el, e.name, e.value); return; }
      case 'attributeRemoved': { const el = N(e.nodeId); if (!el) return miss(e); el.removeAttribute(e.name); return; }
      case 'characterDataModified': { const el = N(e.nodeId); if (!el) return miss(e); el.data = e.characterData; return; }
      case 'shadowRootPushed': { const el = N(e.hostId); if (!el) return miss(e); if (!el.shadowRoot) attachShadow(S, el, e.root, S.doc, false); return; }
      case 'shadowRootPopped': return;
      case 'adoptedStyleSheetsModified': { const el = N(e.nodeId); if (!el) return miss(e); el.__bwAdopted = e.adoptedStyleSheets; applyAdopted(S); return; }
      case 'sheet': { if (!e.text && !(e.header && e.header.isConstructed)) return; let sh = S.sheets.get(e.styleSheetId);
        if (!sh) { sh = new S.doc.defaultView.CSSStyleSheet(); S.sheets.set(e.styleSheetId, sh); } try { sh.replaceSync(e.text || ''); } catch {} return; }
      case 'rules': { const el = S.backend.get(e.ownerNode); if (!el || !el.sheet || !e.rules) return miss(e);
        while (el.sheet.cssRules.length) el.sheet.deleteRule(0); e.rules.forEach((r, i) => { try { el.sheet.insertRule(r, i); } catch {} }); return; }
      case 'styleText': { const el = S.backend.get(e.ownerNode); if (!el) return miss(e);
        const tn = [...el.childNodes].find(n => n.nodeType === 3); if (tn) tn.data = e.text; else el.textContent = e.text; return; }
      case 'asset': { if (!assetBlobs.has(e.url)) assetBlobs.set(e.url, URL.createObjectURL(new Blob([e.text], { type: 'text/css' })));
        for (const Sx of host.sessions.values()) Sx.doc.querySelectorAll('link').forEach(l => { if (l.__bwHref && assetFor(l, l.__bwHref) && l.href !== assetFor(l, l.__bwHref)) l.href = assetFor(l, l.__bwHref); }); return; }
      case 'font': { const f = e.font; try { const ff = new S.doc.defaultView.FontFace(f.fontFamily, `url(${f.src})`, { style: f.fontStyle, weight: f.fontWeight, stretch: f.fontStretch });
        S.doc.fonts.add(ff); ff.load().catch(() => {}); } catch {} return; }
      case 'value': { const el = N(e.nodeId); if (!el) return miss(e);
        if (el.type === 'checkbox' || el.type === 'radio') el.checked = e.v === '1'; else if (el.tagName === 'SELECT') el.selectedIndex = +e.v; else el.value = e.v; return; }
      case 'scroll': { const el = N(e.nodeId); if (!el) return miss(e); const [x, y] = e.v.split(',').map(Number);
        S.scrolls.set(el, [x, y]); scrollTo(S, el, x, y); return; }
      default: stats.ignored[e.type] = (stats.ignored[e.type] || 0) + 1;
    }
  }
  function scrollTo(S, el, x, y) { if (el === S.doc.documentElement || el === S.doc.body) S.doc.defaultView.scrollTo(x, y); else { el.scrollLeft = x; el.scrollTop = y; } }
  function miss(e) { stats.missing[e.type] = (stats.missing[e.type] || 0) + 1; if (!stats.ex) stats.ex = JSON.stringify(e).slice(0, 300); }

  window.bwLoad = function (events) { EV = events; };
  // Rebuild from the newest top snapshot at or before t, then apply everything up to t.
  window.bwSeek = async function (t, w, h) {
    stats = { applied: 0, ua: 0, ignored: {}, missing: {} };
    const stage = document.getElementById('stage'); stage.replaceChildren();
    host = { frameOwners: new Map(), sessions: new Map() };
    let start = -1;
    for (let i = 0; i < EV.length && EV[i].t <= t; i++) if (EV[i].type === 'snapshot' && !EV[i].s) start = i;
    if (start < 0) return stats;
    // Captured assets outlive the snapshot they arrived under (the proxy's store is per recording, not per document).
    for (let i = 0; i < EV.length && EV[i].t <= t; i++) if (EV[i].type === 'asset' && !assetBlobs.has(EV[i].url))
      assetBlobs.set(EV[i].url, URL.createObjectURL(new Blob([EV[i].text], { type: 'text/css' })));
    const top = Session(''); host.sessions.set('', top);
    const f = await freshFrame(stage, w, h);
    snapshot(top, EV[start].root, f);
    for (let i = start + 1; i < EV.length && EV[i].t <= t; i++) {
      const e = EV[i];
      if (e.type === 'snapshot') {
        const owner = host.frameOwners.get(e.frameId); if (!owner) continue;
        owner.contentDocument.open(); owner.contentDocument.write('<!doctype html><html><head></head><body></body></html>'); owner.contentDocument.close();
        const S = Session(e.s); host.sessions.set(e.s, S); snapshot(S, e.root, owner); continue;
      }
      const S = host.sessions.get(e.s); if (!S) continue;
      apply(S, e); stats.applied++;
    }
    // Sheets of constructed stylesheets carry over between seeks only through events; re-assert adoption at the end.
    for (const S of host.sessions.values()) applyAdopted(S);
    // Scroll offsets only stick once the stylesheets that make an element scrollable have loaded: wait, then re-apply.
    const links = []; for (const S of host.sessions.values()) links.push(...S.doc.querySelectorAll('link[rel=stylesheet]'));
    await Promise.race([Promise.all(links.filter(l => !l.sheet).map(l => new Promise(r => { l.addEventListener('load', r); l.addEventListener('error', r); }))), new Promise(r => setTimeout(r, 1500))]);
    for (const S of host.sessions.values()) for (const [el, [x, y]] of S.scrolls) scrollTo(S, el, x, y);
    await Promise.race([Promise.all([...host.sessions.values()].map(S => S.doc.fonts.ready)), new Promise(r => setTimeout(r, 1500))]);
    return stats;
  };
})();
