// The cast's in-page control (#454). Chrome evaluates this in the "couchcast"
// isolated world of every document of the cast Chrome: it shares the page's
// DOM but not its JavaScript, so the page can't see __couchcast or call it.
//
// It draws a Go live / Pause button inside a closed shadow root, so page CSS
// can't restyle it; the host carries data-couchcast-ui so the recorder can
// leave it out. It reports the page's visibility and focus (the cast follows
// the active tab) and its title (Chrome reports a new <title> late), and asks
// for the current state on load ("hello"); Couch answers through
// __couchcastSet(live, driver). While a viewer drives (#467) it shows who,
// above the button.
(() => {
  if (window !== window.top || globalThis.__couchcastInstalled) return;
  globalThis.__couchcastInstalled = true;

  const report = (m) => {
    try { __couchcast(JSON.stringify(m)); } catch (_) { /* binding gone: Chrome is closing */ }
  };
  let live = false;
  let controlRevision = -1;
  let driver = '';
  let button = null, operatorUI = null;
  let banner = null;
  let host = null;
  let guarded = true; // a replacement document waits for its target's guard state
  let tokens = new Map();
  const suppress = (event) => {
    const next = globalThis.__couchcastSuppressionPolicy(tokens, event, Date.now());
    tokens = next.tokens;
    if (host) {
      const hidden = guarded || next.hidden, display = hidden ? 'none' : '';
      if (host.hidden !== hidden) host.hidden = hidden;
      if (host.style.display !== display) host.style.display = display;
    }
    return next.confirmed;
  };
  globalThis.__couchcastUI = (op, token, until) => {
    if (op === 'acquire-background') {
      if (globalThis.__couchcastCaptureBusy?.()) return false;
      op = 'acquire';
    }
    const confirmed = suppress({ op, token, until });
    if (op === 'acquire' && until > 0) setTimeout(() => suppress({ op: 'check' }), Math.max(1, until - Date.now()));
    return confirmed;
  };

  const draw = () => {
    if (!button) return;
    const text = driver ? `${driver} is driving` : '', label = live ? '❚❚ Pause' : '● Go live';
    if (banner.textContent !== text) banner.textContent = text;
    if (banner.hidden !== (!driver || !!operatorUI)) banner.hidden = !driver || !!operatorUI;
    if (button.textContent !== label) button.textContent = label;
    const pressed = live ? 'true' : 'false', cls = live ? 'cast-button live' : 'cast-button';
    if (button.getAttribute('aria-pressed') !== pressed) button.setAttribute('aria-pressed', pressed);
    if (button.className !== cls) button.className = cls;
  };
  globalThis.__couchcastSet = (v, who, guard = false, revision = controlRevision + 1) => { if (revision < controlRevision) return; controlRevision = revision; live = !!v; driver = typeof who === 'string' ? who : ''; guarded = !!guard; suppress({ op: 'check' }); draw(); };

  const mount = () => {
    if (button || !document.documentElement) return;
    host = document.createElement('couchcast-control');
    host.setAttribute('data-couchcast-ui', '');
    host.style.cssText = 'all:initial;position:fixed;z-index:2147483647;right:12px;bottom:12px;';
    const root = host.attachShadow({ mode: 'closed' });
    const style = document.createElement('style');
    style.textContent =
      '.cast-button{font:600 12px/1 system-ui,sans-serif;padding:7px 11px;border-radius:14px;border:0;' +
      'cursor:pointer;color:#fff;background:rgba(40,40,40,.72);box-shadow:0 1px 4px rgba(0,0,0,.3)}' +
      '.cast-button:hover{background:rgba(40,40,40,.95)}.cast-button.live{background:#c62828}' +
      '.cast-banner{font:600 12px/1.2 system-ui,sans-serif;margin:0 0 6px;padding:6px 10px;border-radius:10px;' +
      'color:#fff;background:#1565c0;box-shadow:0 1px 4px rgba(0,0,0,.3);text-align:right}.cast-banner[hidden]{display:none}';
    banner = document.createElement('div');
    banner.setAttribute('role', 'status');
    banner.className = 'cast-banner';
    button = document.createElement('button');
    button.type = 'button';
    button.addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      report({ t: 'toggle' });
    });
    root.appendChild(style);
    root.appendChild(banner);
    root.appendChild(button);
    document.documentElement.appendChild(host);
    if (globalThis.__couchcastMountOperator) {
      operatorUI = globalThis.__couchcastMountOperator({ host, root, report, liveButton: button, banner });
      // The initial hello can precede DOMContentLoaded and the operator hook.
      report({ t: 'hello' });
    }
    suppress({ op: 'check' });
    draw();
  };

  const visibility = () => report({
    t: 'vis', visible: document.visibilityState === 'visible', focused: document.hasFocus(),
  });
  document.addEventListener('visibilitychange', visibility);
  window.addEventListener('focus', visibility);
  window.addEventListener('blur', visibility);

  let lastTitle = null;
  const title = () => {
    const t = document.title;
    if (t === lastTitle) return;
    lastTitle = t;
    report({ t: 'title', title: t });
  };
  // A <title> lives in the head, which changes rarely; observing it costs
  // nothing on a busy body.
  let observed = null;
  const observeTitle = () => {
    const target = document.head || document.documentElement;
    if (!target || target === observed || typeof MutationObserver === 'undefined') return;
    observed = target;
    new MutationObserver(title).observe(target, { subtree: true, childList: true, characterData: true });
  };

  const loaded = () => { mount(); observeTitle(); title(); };
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', loaded);
  } else {
    loaded();
  }
  window.addEventListener('load', title);
  report({ t: 'hello' });
  visibility();
  title();
})();
