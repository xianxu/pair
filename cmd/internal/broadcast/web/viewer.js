// Couch broadcast viewer (#395). Renders the operator's composed screen,
// view-only, at the sender's grid, with the font scaled to fit the window.
// A view link sends nothing back and keeps nothing; a pointer link (#412)
// posts points and sticky notes (#447) and remembers its viewer's color
// pick (#446).

import { noteAt, noteCell, guardedCell, parseNotes, noteDialog } from './notes.js';

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

// Pointer colors (#446). Couch sends the palette and this stream's assigned
// color in `caps`; the page keeps no colors of its own and posts only an
// index. A viewer's pick is remembered in this browser.
const COLOR_KEY = 'couch.pointerColor';

// pointerColor is the color a page posts: the viewer's pick when it is in
// the palette, else the one Couch assigned.
export function pointerColor(picked, assigned, size) {
  return Number.isInteger(picked) && picked >= 0 && picked < size ? picked : assigned;
}

// colorStore remembers the pick. Storage can be missing or throw (a private
// window, blocked site data); the page then works without remembering.
export function colorStore(storage) {
  return {
    get() {
      try {
        const v = storage ? storage.getItem(COLOR_KEY) : null;
        return /^[0-9]$/.test(v || '') ? Number(v) : null;
      } catch {
        return null;
      }
    },
    set(i) {
      try {
        storage?.setItem(COLOR_KEY, String(i));
      } catch {
        // Not remembered; the pick still holds for this page.
      }
    },
  };
}

// RAIL_INSET is the gap between the rail and the screen's right edge.
export const RAIL_INSET = 6;

// railAt docks the rail (#449) to the drawn screen: its right edge
// RAIL_INSET inside the screen's, its middle on the screen's (the CSS
// translates it up by half its height). rect is the screen's client rect;
// viewW the window's width.
export function railAt(rect, viewW) {
  return {
    top: rect.top + rect.height / 2,
    right: Math.max(0, viewW - rect.right) + RAIL_INSET,
  };
}

// renderSwatches draws one button per palette color into el, marking the
// current one. make(tag) creates an element.
export function renderSwatches(el, palette, current, pick, make) {
  el.replaceChildren(...palette.map((hex, i) => {
    const b = make('button');
    b.type = 'button';
    b.className = i === current ? 'swatch on' : 'swatch';
    b.style.background = hex;
    b.setAttribute('aria-label', `Pointer color ${i + 1}`);
    b.title = `Pointer color ${i + 1}`;
    b.setAttribute('aria-pressed', String(i === current));
    b.addEventListener('click', () => pick(i));
    return b;
  }));
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
// show(text); setStale(bool); later(fn, ms); caps(capabilities) and
// notes(list), optional.
export function connect(deps) {
  let attempt = 0;
  let ended = false;
  // A pointer page hears caps; a view page never does. The end puts each
  // back to its own join state.
  let sawCaps = false;
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
      sawCaps = true;
      if (deps.caps) {
        deps.caps(JSON.parse(ev.data));
      }
    });
    events.addEventListener('notes', (ev) => {
      if (deps.notes) {
        deps.notes(JSON.parse(ev.data));
      }
    });
    events.addEventListener('end', (ev) => {
      ended = true;
      events.close();
      deps.reset();
      // Everything the stream gave a page goes back to the page's join
      // state (#447): no notes to tap, and on a pointer page pointing off
      // (which closes a dialog and hides the rail). The server sends end,
      // caps and notes in no fixed order, so the end can't rely on a last
      // caps or notes event, and it never replays an event this page
      // variant doesn't receive.
      if (sawCaps) {
        deps.caps?.({ pointer: false });
      }
      deps.notes?.({ guard: 0, notes: [] });
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

const SYMBOLS = '"Couch Symbols"';

// SYMBOL_SAMPLE is a code point in the symbol font's unicode-range, so loading
// it fetches the face. xterm.js measures each character once, the first time
// it draws it, and keeps a letter-spacing that squeezes the measured glyph
// into one cell. Measured before the lazily fetched face arrived, a symbol
// keeps the fallback's correction, collapses to no width, and swallows the
// next column (#415 iPad smoke: `⏺The`, `⏸` outside LIVE's background). So it
// loads up front with the rest, not on first use.
export const SYMBOL_SAMPLE = '⏺';

// loadFont waits, briefly, for every face of the packed fonts, so xterm.js
// measures and draws with them rather than a fallback. If they don't load,
// the stack falls back and the screen still works.
async function loadFont() {
  try {
    await Promise.race([
      Promise.all([
        ...FONT_FACES.map((face) => document.fonts.load(`${face}16px ${FONT}`)),
        document.fonts.load(`16px ${SYMBOLS}`, SYMBOL_SAMPLE),
      ]),
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
    fontFamily: `${FONT}, ${SYMBOLS}, ui-monospace, Menlo, Monaco, Consolas, monospace`,
    // The Unicode API is "proposed" in xterm.js; it is what sets widths.
    allowProposedApi: true,
  });
  // Unicode 11 widths: emoji are two columns, as Couch draws them (#412).
  term.loadAddon(new globalThis.Unicode11Addon.Unicode11Addon());
  term.unicode.activeVersion = '11';
  term.open(document.getElementById('term'));
  let size = 16;
  const rail = document.getElementById('rail');
  // fit scales the font to the stage, then docks the rail to the screen it
  // drew.
  const fit = () => {
    const screen = document.querySelector('#term .xterm-screen');
    if (!screen) {
      return;
    }
    for (let i = 0; i < 3; i++) {
      const r = screen.getBoundingClientRect();
      const next = nextFontSize(size, r.width, r.height, stage.clientWidth, stage.clientHeight);
      if (next === size) {
        break;
      }
      size = next;
      term.options.fontSize = size;
    }
    const at = railAt(screen.getBoundingClientRect(), document.documentElement.clientWidth);
    rail.style.top = `${at.top}px`;
    rail.style.right = `${at.right}px`;
  };

  let storage = null;
  try {
    storage = window.localStorage;
  } catch {
    // Blocked site data: picks aren't remembered.
  }
  const swatches = document.getElementById('swatches');
  const modeButton = document.getElementById('mode');
  let notes = { guard: 0, notes: [] };
  let pointer = null;
  const byId = (id) => document.getElementById(id);
  const dialog = noteDialog({
    root: byId('note'), text: byId('note-text'), input: byId('note-input'), count: byId('note-count'),
    msg: byId('note-msg'), save: byId('note-save'), cancel: byId('note-cancel'), resolve: byId('note-resolve'),
  }, { post: postJSON, color: () => pointer.color() });
  pointer = pointerMode(term, stage, {
    notes: () => notes,
    dialog,
    modeButton: {
      show: (mode, toggle, tip) => {
        modeButton.textContent = mode === 'comment' ? '💭' : '👆';
        modeButton.title = tip;
        modeButton.setAttribute('aria-label', tip);
        modeButton.onclick = toggle;
        modeButton.classList.remove('hidden');
      },
      hide: () => modeButton.classList.add('hidden'),
    },
    store: colorStore(storage),
    swatches: {
      show: (palette, current, pick) => {
        renderSwatches(swatches, palette, current, pick, (tag) => document.createElement(tag));
        swatches.classList.remove('hidden');
      },
      hide: () => swatches.classList.add('hidden'),
    },
  });
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
    setStale: (on) => {
      stage.classList.toggle('stale', on);
      rail.classList.toggle('stale', on);
    },
    later: (fn, ms) => setTimeout(fn, ms),
    caps: (c) => {
      pointer.setColors(c.palette, c.color);
      pointer.setOn(Boolean(c.pointer));
    },
    notes: (data) => {
      notes = parseNotes(data);
      dialog.update(notes);
    },
  });
  window.addEventListener('resize', fit);
  fit();
}

// POST_ROUTES is every request this page makes besides its own assets and
// stream (#412, #447).
const POST_ROUTES = ['point', 'note', 'resolve'];

// postJSON is the page's one request function: same origin, a relative
// route from POST_ROUTES, no credentials, no referrer. It resolves to the
// status and the response text (a refusal sentence, for the note dialog).
function postJSON(route, body) {
  if (!POST_ROUTES.includes(route)) {
    return Promise.reject(new Error('unknown route'));
  }
  return fetch(route, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    credentials: 'omit',
    referrerPolicy: 'no-referrer',
    cache: 'no-store',
  }).then(async (r) => ({ status: r.status, text: await r.text() }));
}

function postPoint(body) {
  postJSON('point', body).catch(() => {});
}

// MODE_TIPS explain the mode button on hover (#447 M3): the page has no hint
// bar, so the screen keeps the whole window. Touch devices show no tooltips;
// the icons stand alone there.
export const MODE_TIPS = {
  point: 'Pointer mode: tap or drag to point. Click for comment mode.',
  comment: 'Comment mode: tap the screen to leave a note. Click for pointer mode.',
};

// pointerMode routes the viewer's taps. A tap on a 💭 opens that note, on
// any page (#447). While the broadcast says pointing is on, the mode button
// switches between pointer mode, where taps and drags become point batches
// (flushed every 50ms), and comment mode, where a tap on an empty cell opens
// a draft. deps: notes() → {guard, notes}, dialog (noteDialog), modeButton
// {show(mode, toggle, tip), hide()}, store (colorStore), swatches
// {show(palette, current, pick), hide()}, and for tests post(body),
// every(fn, ms) → id, stop(id).
export function pointerMode(term, stage, deps = {}) {
  const send = deps.post || postPoint;
  const every = deps.every || ((fn, ms) => setInterval(fn, ms));
  const stop = deps.stop || ((id) => clearInterval(id));
  const store = deps.store || colorStore(null);
  const swatches = deps.swatches || { show() {}, hide() {} };
  const notes = deps.notes || (() => ({ guard: 0, notes: [] }));
  const dialog = deps.dialog || { draft() {}, view() {}, close() {} };
  const modeButton = deps.modeButton || { show() {}, hide() {} };
  let mode = 'point';
  let on = false;
  let stroke = null;
  let timer = null;
  let palette = [];
  let assigned = 0;
  let picked = store.get();
  const color = () => pointerColor(picked, assigned, palette.length);
  const drawMode = () => {
    if (on) {
      modeButton.show(mode, () => {
        mode = mode === 'point' ? 'comment' : 'point';
        drawMode();
      }, MODE_TIPS[mode]);
    } else {
      modeButton.hide();
    }
    stage.classList.toggle('comment', on && mode === 'comment');
  };
  const drawSwatches = () => {
    if (on && palette.length) {
      swatches.show(palette, color(), (i) => {
        picked = i;
        store.set(i);
        drawSwatches();
      });
    } else {
      swatches.hide();
    }
  };
  const screen = () => stage.querySelector('.xterm-screen');
  const post = (down, points) => send({ cols: term.cols, rows: term.rows, down, color: color(), points });
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
    if (timer !== null) {
      stop(timer);
    }
    timer = null;
    stroke = null;
  };
  stage.addEventListener('pointerdown', (e) => {
    const c = cell(e);
    if (!c) {
      return;
    }
    const hit = noteAt(notes(), c[0], c[1], term.cols, term.rows);
    if (hit) {
      e.preventDefault();
      end();
      dialog.view(hit, { resolvable: on });
      return;
    }
    if (!on) {
      return;
    }
    e.preventDefault();
    if (mode === 'comment') {
      const at = noteCell(c, term.cols);
      if (!guardedCell(at[0], at[1], term.rows, notes().guard)) {
        dialog.draft(at, { cols: term.cols, rows: term.rows });
      }
      return;
    }
    stage.setPointerCapture?.(e.pointerId);
    end();
    stroke = { last: null, pending: [c] };
    flush(true);
    timer = every(() => flush(true), 50);
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
        mode = 'point';
        dialog.close();
      }
      stage.classList.toggle('pointer', on);
      drawMode();
      drawSwatches();
    },
    // color is the palette index this viewer posts with.
    color,
    // setColors takes the palette and assigned color from caps. A palette
    // with anything but #rrggbb values offers no swatches.
    setColors(p, a) {
      const valid = Array.isArray(p) && p.every((v) => typeof v === 'string' && HEX_COLOR.test(v));
      palette = valid ? p : [];
      assigned = pointerColor(a, 0, palette.length);
      drawSwatches();
    },
  };
}

if (typeof document !== 'undefined') {
  start();
}
