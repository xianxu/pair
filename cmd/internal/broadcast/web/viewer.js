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

// The pointer link (#412): a helper's taps and drags become cell coordinates
// posted to the relative URL `point`. Couch only draws them as fading marks.
export const MAX_POINTS = 64;

// cellAt maps a viewport point to the grid cell under it, or null outside the
// screen.
export function cellAt(x, y, rect, cols, rows) {
  if (!(rect.width > 0 && rect.height > 0) || !Number.isFinite(x) || !Number.isFinite(y)) {
    return null;
  }
  const col = Math.floor(((x - rect.left) / rect.width) * cols);
  const row = Math.floor(((y - rect.top) / rect.height) * rows);
  if (col < 0 || col >= cols || row < 0 || row >= rows) {
    return null;
  }
  return [col, row];
}

// chunkStroke cuts a stroke's new points into requests of at most MAX_POINTS.
// Each request starts at the point the previous one ended on (or `last`, the
// end of the previous batch), so Couch's line fill joins them seamlessly.
export function chunkStroke(last, points) {
  if (points.length === 0) {
    return [];
  }
  const chunks = [];
  let prev = last;
  let i = 0;
  while (i < points.length) {
    const room = prev ? MAX_POINTS - 1 : MAX_POINTS;
    const part = points.slice(i, i + room);
    chunks.push(prev ? [prev, ...part] : part);
    prev = part[part.length - 1];
    i += part.length;
  }
  return chunks;
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
// show(text); setStale(bool); later(fn, ms); caps(capabilities), optional.
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
    events.addEventListener('caps', (ev) => {
      if (deps.caps) {
        deps.caps(JSON.parse(ev.data));
      }
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
  const hint = document.getElementById('hint');
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
    // The Unicode API is "proposed" in xterm.js; it is what sets widths.
    allowProposedApi: true,
  });
  // Unicode 11 widths: emoji are two columns, as Couch draws them (#412).
  term.loadAddon(new globalThis.Unicode11Addon.Unicode11Addon());
  term.unicode.activeVersion = '11';
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

  const pointer = pointerMode(term, stage, hint);
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
    caps: (c) => pointer.setOn(Boolean(c.pointer)),
  });
  window.addEventListener('resize', fit);
  fit();
}

// pointerMode turns the helper's taps and drags into point batches while
// the broadcast says pointing is on. Strokes are flushed every 50ms.
function pointerMode(term, stage, hint) {
  let on = false;
  let stroke = null;
  let timer = null;
  const screen = () => stage.querySelector('.xterm-screen');
  const post = (down, points) => {
    // The one request this page makes besides its own assets and stream:
    // same origin, relative URL, no credentials, no referrer.
    fetch('point', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cols: term.cols, rows: term.rows, down, points }),
      credentials: 'omit',
      referrerPolicy: 'no-referrer',
      cache: 'no-store',
    }).catch(() => {});
  };
  const flush = (down) => {
    if (!stroke) {
      return;
    }
    for (const chunk of chunkStroke(stroke.last, stroke.pending)) {
      post(down, chunk);
    }
    if (stroke.pending.length) {
      stroke.last = stroke.pending[stroke.pending.length - 1];
    }
    stroke.pending = [];
  };
  const cell = (e) => {
    const el = screen();
    return el ? cellAt(e.clientX, e.clientY, el.getBoundingClientRect(), term.cols, term.rows) : null;
  };
  const end = () => {
    if (stroke) {
      flush(false);
    }
    clearInterval(timer);
    timer = null;
    stroke = null;
  };
  stage.addEventListener('pointerdown', (e) => {
    if (!on) {
      return;
    }
    const c = cell(e);
    if (!c) {
      return;
    }
    e.preventDefault();
    stage.setPointerCapture?.(e.pointerId);
    end();
    stroke = { last: null, pending: [c] };
    flush(true);
    timer = setInterval(() => flush(true), 50);
  });
  stage.addEventListener('pointermove', (e) => {
    if (!stroke) {
      return;
    }
    const c = cell(e);
    const tail = stroke.pending[stroke.pending.length - 1] || stroke.last;
    if (c && !(tail && tail[0] === c[0] && tail[1] === c[1])) {
      stroke.pending.push(c);
    }
  });
  stage.addEventListener('pointerup', end);
  stage.addEventListener('pointercancel', end);
  return {
    setOn(value) {
      on = value;
      if (!on) {
        end();
      }
      stage.classList.toggle('pointer', on);
      hint.textContent = on ? 'You can point: tap or drag on the screen' : 'Pointing is off';
      hint.classList.remove('hidden');
    },
  };
}

if (typeof document !== 'undefined') {
  start();
}
