// Couch broadcast viewer (#395). Renders the operator's composed screen,
// view-only, at the sender's grid, with the font scaled to fit the window.
// It sends nothing back and keeps nothing: closing the tab leaves no trace.

export const MIN_FONT = 4;
export const MAX_FONT = 64;

// nextFontSize is one fit step: scale the font by how much the rendered
// screen (screenW×screenH px at `current`) must grow or shrink to fit the
// viewport, rounded down to half a pixel and clamped. A screen that already
// fits keeps its size, so repeated steps settle.
export function nextFontSize(current, screenW, screenH, viewW, viewH) {
  const measurable = [current, screenW, screenH, viewW, viewH].every((v) => Number.isFinite(v) && v > 0);
  if (!measurable) {
    return current;
  }
  const scale = Math.min(viewW / screenW, viewH / screenH);
  const size = Math.min(MAX_FONT, Math.max(MIN_FONT, Math.floor(current * scale * 2) / 2));
  const fits = screenW <= viewW && screenH <= viewH;
  return fits && size < current ? current : size;
}

function decode(b64) {
  return Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
}

function start() {
  const status = document.getElementById('status');
  const show = (text) => {
    status.textContent = text || '';
    status.classList.toggle('hidden', !text);
  };
  const term = new globalThis.Terminal({
    scrollback: 0,
    disableStdin: true,
    cursorBlink: false,
    fontSize: 16,
    fontFamily: 'ui-monospace, Menlo, Monaco, Consolas, monospace',
  });
  term.open(document.getElementById('term'));
  let size = 16;
  const fit = () => {
    const screen = document.querySelector('#term .xterm-screen');
    if (!screen) {
      return;
    }
    for (let i = 0; i < 3; i++) {
      const r = screen.getBoundingClientRect();
      const next = nextFontSize(size, r.width, r.height, window.innerWidth, window.innerHeight);
      if (next === size) {
        break;
      }
      size = next;
      term.options.fontSize = size;
    }
  };

  const events = new EventSource('events');
  events.addEventListener('frame', (ev) => {
    const m = JSON.parse(ev.data);
    const resized = term.cols !== m.cols || term.rows !== m.rows;
    if (resized) {
      term.resize(m.cols, m.rows);
    }
    term.write(decode(m.b));
    show('');
    if (resized) {
      fit();
    }
  });
  events.addEventListener('end', (ev) => {
    events.close();
    term.reset();
    let reason = '';
    try {
      reason = JSON.parse(ev.data).reason || '';
    } catch {
      // A malformed reason still ends the broadcast.
    }
    show(reason ? `Broadcast ended: ${reason}` : 'Broadcast ended');
  });
  events.onerror = () => {
    if (events.readyState !== EventSource.CLOSED) {
      show('Reconnecting…');
    }
  };
  window.addEventListener('resize', fit);
  fit();
}

if (typeof document !== 'undefined') {
  start();
}
