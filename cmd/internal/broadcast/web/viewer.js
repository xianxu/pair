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

// The operator's palette arrives as `event: theme`. xterm.js names the 16
// ANSI colours; index order is SGR 30–37 then 90–97.
const ANSI_NAMES = [
  'black', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'white',
  'brightBlack', 'brightRed', 'brightGreen', 'brightYellow',
  'brightBlue', 'brightMagenta', 'brightCyan', 'brightWhite',
];
const HEX_COLOR = /^#[0-9a-fA-F]{6}$/;

// xtermTheme maps the operator's palette onto xterm.js theme keys. Only
// well-formed #rrggbb values pass; anything else (unknown, or not a colour)
// is left out, so xterm.js keeps its default for it.
export function xtermTheme(palette) {
  const theme = {};
  const put = (key, value) => {
    if (typeof value === 'string' && HEX_COLOR.test(value)) {
      theme[key] = value;
    }
  };
  put('foreground', palette.foreground);
  put('background', palette.background);
  (palette.ansi || []).forEach((value, i) => {
    if (i < ANSI_NAMES.length) {
      put(ANSI_NAMES[i], value);
    }
  });
  return theme;
}

// Backoff for a connection closed for good (the server refused it or is
// gone). A broadcast that ended says so with `end` and is never retried.
export const RETRY_DELAYS = [2000, 4000, 8000, 15000, 30000];

const EVENTSOURCE_CLOSED = 2;

function endText(data) {
  try {
    const reason = JSON.parse(data).reason || '';
    return reason ? `Broadcast ended: ${reason}` : 'Broadcast ended';
  } catch {
    return 'Broadcast ended';
  }
}

// connect runs the viewer's connection state machine. The screen is never
// left looking live while it isn't: any lost connection dims it and says so.
// deps: open() → EventSource-like; render(frame); theme(palette); reset();
// show(text); setStale(bool); later(fn, ms).
export function connect(deps) {
  let attempt = 0;
  let ended = false;
  const open = () => {
    const events = deps.open();
    events.addEventListener('frame', (ev) => {
      attempt = 0;
      deps.setStale(false);
      deps.show('');
      deps.render(JSON.parse(ev.data));
    });
    events.addEventListener('theme', (ev) => {
      deps.theme(JSON.parse(ev.data));
    });
    events.addEventListener('end', (ev) => {
      ended = true;
      events.close();
      deps.reset();
      deps.setStale(false);
      deps.show(endText(ev.data));
    });
    events.onerror = () => {
      if (ended) {
        return;
      }
      deps.setStale(true);
      if (events.readyState !== EVENTSOURCE_CLOSED) {
        // The browser is reconnecting by itself.
        deps.show('Reconnecting…');
        return;
      }
      events.close();
      if (attempt >= RETRY_DELAYS.length) {
        deps.show('Disconnected');
        return;
      }
      const delay = RETRY_DELAYS[attempt++];
      deps.show(`Disconnected — retrying in ${delay / 1000}s`);
      deps.later(open, delay);
    };
  };
  open();
}

function decode(b64) {
  return Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
}

const FONT = '"JetBrains Mono"';

// FONT_FACES is every face the screen can draw in. All of them must be
// loaded before the first frame: xterm.js lays a row's styled runs out in
// flow, so a run drawn in a fallback face with a different advance (say,
// italic text before the italic face arrives) shifts the rest of its row.
// (#395 M5 smoke: Claude's italic recap lines pushed the pane border left.)
export const FONT_FACES = ['', 'bold ', 'italic ', 'italic bold '];

// loadFont waits, briefly, for every face of the packed font, so xterm.js
// measures and draws with it rather than a fallback. If it doesn't load, the
// stack falls back and the screen still works.
async function loadFont() {
  try {
    await Promise.race([
      Promise.all(FONT_FACES.map((face) => document.fonts.load(`${face}16px ${FONT}`))),
      new Promise((resolve) => setTimeout(resolve, 3000)),
    ]);
  } catch {
    // Fall back to the rest of the stack.
  }
}

async function start() {
  await loadFont();
  const status = document.getElementById('status');
  const stage = document.getElementById('stage');
  const show = (text) => {
    status.textContent = text || '';
    status.classList.toggle('hidden', !text);
  };
  const term = new globalThis.Terminal({
    scrollback: 0,
    disableStdin: true,
    cursorBlink: false,
    fontSize: 16,
    fontFamily: `${FONT}, ui-monospace, Menlo, Monaco, Consolas, monospace`,
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

  connect({
    open: () => new EventSource('events'),
    render: (m) => {
      const resized = term.cols !== m.cols || term.rows !== m.rows;
      if (resized) {
        term.resize(m.cols, m.rows);
      }
      term.write(decode(m.b));
      if (resized) {
        fit();
      }
    },
    theme: (palette) => {
      const theme = xtermTheme(palette);
      term.options.theme = theme;
      if (theme.background) {
        document.documentElement.style.setProperty('--bg', theme.background);
      }
    },
    reset: () => term.reset(),
    show,
    setStale: (on) => stage.classList.toggle('stale', on),
    later: (fn, ms) => setTimeout(fn, ms),
  });
  window.addEventListener('resize', fit);
  fit();
}

if (typeof document !== 'undefined') {
  start();
}
