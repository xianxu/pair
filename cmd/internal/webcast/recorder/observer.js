// The recorder's observer (#458). Chrome evaluates it in the "couchcast"
// isolated world of every frame of the cast Chrome: it shares the page's DOM
// but not its JavaScript, so the page neither sees nor can disturb it, and
// nothing runs in the page's main world.
//
// It reports what CDP's DOM domain doesn't: form values (also those set by
// script, found by a scan), scroll offsets, the selection, focus, the
// element under the pointer and a sampled pointer position, clicks, the
// viewport, and CSSOM-only stylesheets. Nodes are named by path (child
// indexes from this frame's document; -1 enters an open shadow root), which
// the recorder resolves against its mirror of CDP's DOM.
//
// Reports go through the __couchcast binding as {t:"obs", items:[…]}. A
// click is sent at once, in its handler, after whatever was pending: a click
// that navigates is reported before the document goes (#457: a polled
// observer lost it). Everything else is coalesced per animation frame. It
// reports only while the recorder has switched it on (__couchcastRec), so a
// page that isn't cast pays only idle listeners.
//
// Remote driving (#467) holds it around a viewer's injected input:
// __couchcastHold(by) returns what is pending (the operator's or the
// page's) and starts the hold; everything the observer sends during it
// carries by (a click at once, before a navigation can take its document),
// and __couchcastTake sends what is still coalesced, tagged, and ends it.
// Every driven item goes through the binding, so they keep their order.
// The recorder calls both inside one queue input. A hold no take ends
// expires after HOLD_MAX: what comes after is sent as usual, unattributed.
// It can never stay held.
(() => {
  // The opaque private scene and its inherited frames must never install a
  // recording observer. This reserved URL exists before document scripts run,
  // unlike a DOM marker. Cross-origin ordinary parents end the ancestry walk.
  try {
    for (let frame = window, depth = 0; depth < 128; depth++) {
      if (frame.location.href.startsWith('data:text/html;couch-sketch=1,')) return;
      if (frame.parent === frame) break;
      frame = frame.parent;
    }
  } catch (_) { /* ordinary cross-origin ancestor */ }
  if (globalThis.__couchcastRecInstalled) return;
  globalThis.__couchcastRecInstalled = true;

  const SHADOW = -1;
  const PTR_EVERY = 250; // ms between pointer samples (a few per second)
  const SCAN_EVERY = 250; // ms between scans for values set by script
  const HOLD_MAX = 1000; // ms a hold lasts without a take
  const send = (items, by) => {
    try { __couchcast(JSON.stringify(by ? { t: 'obs', items, by } : { t: 'obs', items })); } catch (_) { /* binding gone */ }
  };

  let on = false;
  let pending = [];
  let scheduled = false;
  const scrolls = new Map(); // target -> true, coalesced per frame
  let selDirty = false;
  let held = null; // the hold's expiry timer while held
  let heldBy = ''; // whose input the hold is around

  const isControlUI = (n) => n && n.nodeType === 1 && n.hasAttribute && n.hasAttribute('data-couchcast-ui');

  // path names a node from its document: child indexes, entering open
  // shadow roots. null when it isn't in this frame's document.
  const path = (n) => {
    const steps = [];
    while (n && n.nodeType !== 9) {
      const p = n.parentNode;
      if (!p) return null;
      if (isControlUI(n)) return null;
      steps.push(Array.prototype.indexOf.call(p.childNodes, n));
      if (p.nodeType === 11 && p.host) {
        steps.push(SHADOW);
        n = p.host;
      } else {
        n = p;
      }
    }
    return n === document ? steps.reverse() : null;
  };
  globalThis.__couchcastPath = path;
  const target = (e) => (e.composedPath && e.composedPath()[0]) || e.target;

  // collect turns coalesced scrolls and the selection into pending items.
  const collect = () => {
    for (const t of scrolls.keys()) scrollItem(t);
    scrolls.clear();
    if (selDirty) { selDirty = false; selectionItem(); }
  };
  const flush = () => {
    scheduled = false;
    if (!on) { pending = []; scrolls.clear(); selDirty = false; return; }
    collect();
    // While held, what goes is sent at once, tagged with whose input the
    // hold is around: a click that navigates must leave before its document.
    if (pending.length) { const items = pending; pending = []; send(items, held ? heldBy : ''); }
  };
  const schedule = () => {
    if (scheduled) return;
    scheduled = true;
    // Whichever comes first: a frame, or a short timeout (no frames are
    // painted while the page is hidden).
    let done = false;
    const go = () => { if (!done) { done = true; flush(); } };
    requestAnimationFrame(go);
    setTimeout(go, 50);
  };
  const push = (item) => { if (on) { pending.push(item); schedule(); } };

  // Form controls.
  const last = new WeakMap(); // control -> what was last reported
  const controlItem = (el) => {
    const p = path(el);
    if (!p) return null;
    const tag = el.tagName;
    if (tag === 'INPUT' && (el.type || '').toLowerCase() === 'password') {
      return { k: 'input', p, secret: el.value !== '' }; // never the value
    }
    if (tag === 'INPUT' && (el.type === 'checkbox' || el.type === 'radio')) return { k: 'input', p, checked: el.checked };
    if (tag === 'SELECT') {
      const sel = [];
      for (let i = 0; i < el.options.length; i++) if (el.options[i].selected) sel.push(i);
      return { k: 'input', p, sel };
    }
    return { k: 'input', p, v: String(el.value) };
  };
  const report = (el, force) => {
    const it = controlItem(el);
    if (!it) return;
    const key = JSON.stringify([it.v, it.checked, it.sel, it.secret]);
    if (!force && last.get(el) === key) return;
    last.set(el, key);
    push(it);
  };
  const isControl = (el) => el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT');
  const controls = (root, out) => {
    root.querySelectorAll('input,textarea,select').forEach((el) => out.push(el));
    root.querySelectorAll('*').forEach((el) => { if (el.shadowRoot) controls(el.shadowRoot, out); });
    return out;
  };
  for (const type of ['input', 'change']) {
    document.addEventListener(type, (e) => { const t = target(e); if (on && isControl(t)) report(t); }, true);
  }
  setInterval(() => { if (on && document.documentElement) controls(document, []).forEach((el) => report(el)); }, SCAN_EVERY);

  // Scroll: event-driven, coalesced per frame (#460: polling trailed the tab).
  const scrollItem = (t) => {
    if (t === document || t === document.scrollingElement || t === document.documentElement) {
      push({ k: 'scroll', p: [], x: window.scrollX, y: window.scrollY });
      return;
    }
    const p = path(t);
    if (p) push({ k: 'scroll', p, x: t.scrollLeft, y: t.scrollTop });
  };
  document.addEventListener('scroll', (e) => { if (on) { scrolls.set(target(e), true); schedule(); } }, true);

  // Selection.
  const selectionItem = () => {
    const s = document.getSelection();
    if (!s || s.rangeCount === 0 || !s.anchorNode) { push({ k: 'sel' }); return; }
    const a = path(s.anchorNode), f = path(s.focusNode);
    if (!a || !f) return;
    push({ k: 'sel', a: [a, s.anchorOffset], f: [f, s.focusOffset] });
  };
  document.addEventListener('selectionchange', () => { if (on) { selDirty = true; schedule(); } });

  // Focus.
  document.addEventListener('focusin', (e) => { const p = path(target(e)); if (p) push({ k: 'focus', p }); }, true);
  document.addEventListener('focusout', (e) => { if (!e.relatedTarget) push({ k: 'focus', p: null }); }, true);

  // Hover and the pointer.
  let lastPtr = 0;
  document.addEventListener('pointerover', (e) => { const p = path(target(e)); if (p) push({ k: 'hover', p }); }, true);
  document.addEventListener('pointerout', (e) => { if (!e.relatedTarget) push({ k: 'hover', p: null }); }, true);
  document.addEventListener('pointermove', (e) => {
    const now = Date.now();
    if (!on || now - lastPtr < PTR_EVERY) return;
    lastPtr = now;
    push({ k: 'ptr', x: e.clientX, y: e.clientY });
  }, true);

  // Clicks: at once, after anything pending, so the order holds.
  document.addEventListener('click', (e) => {
    if (!on) return;
    const p = path(target(e));
    if (!p) return;
    pending.push({ k: 'click', p, x: e.clientX, y: e.clientY });
    flush();
  }, true);

  // The viewport (top frame only).
  const vp = () => { if (window === window.top) push({ k: 'vp', w: window.innerWidth, h: window.innerHeight, dpr: window.devicePixelRatio }); };
  window.addEventListener('resize', vp);

  // CSSOM-only stylesheets: a <style> whose rules were added by insertRule
  // with no text (CSS-in-JS in production mode). Changes after this come
  // from CDP's styleSheetChanged.
  const sheets = () => {
    for (const sh of Array.from(document.styleSheets)) {
      const owner = sh.ownerNode;
      if (!owner || owner.tagName !== 'STYLE') continue;
      let rules;
      try { rules = Array.from(sh.cssRules, (r) => r.cssText).join('\n'); } catch (_) { continue; }
      if (rules && owner.textContent.trim() === '') {
        const p = path(owner);
        if (p) push({ k: 'sheet', p, text: rules });
      }
    }
  };

  // resync reports everything the recorder can't read from CDP, after a new
  // snapshot (each segment, each new frame document).
  const resync = () => {
    vp();
    controls(document, []).forEach((el) => report(el, true));
    scrollItem(document);
    document.querySelectorAll('*').forEach((el) => { if (el.scrollTop || el.scrollLeft) scrollItem(el); });
    const a = document.activeElement;
    if (a && a !== document.body) { const p = path(a); if (p) push({ k: 'focus', p }); }
    selDirty = true;
    sheets();
    schedule();
  };

  const take = () => {
    if (!on) { pending = []; return '[]'; }
    collect();
    return JSON.stringify(pending.splice(0));
  };
  const release = () => {
    if (held) clearTimeout(held);
    held = null;
  };
  globalThis.__couchcastHold = (by) => {
    const out = take();
    release();
    heldBy = typeof by === 'string' ? by : '';
    held = setTimeout(() => { held = null; schedule(); }, HOLD_MAX);
    return out;
  };
  // The take sends what is left through the binding, after anything the
  // hold already sent, so the driver's items arrive in the order they
  // happened, all tagged; it returns nothing.
  globalThis.__couchcastTake = () => {
    if (on) {
      collect();
      if (pending.length) send(pending.splice(0), heldBy);
    } else {
      pending = [];
    }
    release();
    return '[]';
  };

  globalThis.__couchcastRec = (v, sync) => {
    on = !!v;
    if (on && sync) resync();
  };
  send([{ k: 'hello' }]);
})();
