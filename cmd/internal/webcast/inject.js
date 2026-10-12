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
  let driver = '';
  let button = null;
  let banner = null;

  const draw = () => {
    if (!button) return;
    banner.textContent = driver ? `${driver} is driving` : '';
    banner.hidden = !driver;
    button.textContent = live ? '❚❚ Pause' : '● Go live';
    button.setAttribute('aria-pressed', live ? 'true' : 'false');
    button.className = live ? 'live' : '';
  };
  globalThis.__couchcastSet = (v, who) => { live = !!v; driver = typeof who === 'string' ? who : ''; draw(); };

  const mount = () => {
    if (button || !document.documentElement) return;
    const host = document.createElement('couchcast-control');
    host.setAttribute('data-couchcast-ui', '');
    host.style.cssText = 'all:initial;position:fixed;z-index:2147483647;right:12px;bottom:12px;';
    const root = host.attachShadow({ mode: 'closed' });
    const style = document.createElement('style');
    style.textContent =
      'button{font:600 12px/1 system-ui,sans-serif;padding:7px 11px;border-radius:14px;border:0;' +
      'cursor:pointer;color:#fff;background:rgba(40,40,40,.72);box-shadow:0 1px 4px rgba(0,0,0,.3)}' +
      'button:hover{background:rgba(40,40,40,.95)}button.live{background:#c62828}' +
      'div{font:600 12px/1.2 system-ui,sans-serif;margin:0 0 6px;padding:6px 10px;border-radius:10px;' +
      'color:#fff;background:#1565c0;box-shadow:0 1px 4px rgba(0,0,0,.3);text-align:right}div[hidden]{display:none}';
    banner = document.createElement('div');
    banner.setAttribute('role', 'status');
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
