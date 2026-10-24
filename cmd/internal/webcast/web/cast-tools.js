import { feedbackLayout } from './feedback-layout.js';

// Owns presentation and node placement only. Adapters retain their controls,
// listeners, permission decisions, and hidden/disabled state.
export function mountCastTools(doc, win, {parent, liveButton, layout}) {
  const make = (tag, className) => {
    const node = doc.createElement(tag); node.className = className; return node;
  };
  const root = make('nav', 'cast-tools');
  root.setAttribute('aria-label', 'Cast tools');
  const groups = {};
  for (const name of ['live', 'pointer', 'pen', 'colors', 'note', 'history']) {
    groups[name] = make('div', `cast-tools-group cast-tools-${name}`);
    root.append(groups[name]);
  }
  for (const name of ['pointerColors', 'penColors']) {
    groups[name] = make('div', `cast-tools-group cast-tools-${name}`);
    groups.colors.append(groups[name]);
  }
  parent.append(root);
  let disposed = false;
  const origins = new Map(), adopted = new Map();
  const adopt = (node, group) => {
    if (disposed || !node) return;
    if (!adopted.has(node)) {
      const original = node.parentNode;
      if (original && !origins.has(original)) origins.set(original, [...(original.childNodes || original.children)]);
      adopted.set(node, {original, group});
    } else adopted.get(node).group = group;
    group.append(node);
  };
  const refresh = () => {
    if (disposed) return;
    const bounds = (layout ? layout() : feedbackLayout({width: win.innerWidth, height: win.innerHeight})).toolbar;
    root.style.left = `${bounds.left}px`;
    root.style.top = `${bounds.top}px`;
    root.style.width = `${bounds.width}px`;
    root.style.maxHeight = `${bounds.height}px`;
  };
  // Let native buttons/selects work while keeping scene-level gestures out.
  const stop = event => event.stopPropagation();
  const events = ['pointerdown', 'pointermove', 'pointerup', 'click', 'dblclick', 'wheel'];
  for (const type of events) root.addEventListener(type, stop);
  win.addEventListener('resize', refresh);
  adopt(liveButton, groups.live); refresh();
  return {
    root,
    // Parent may set group.hidden to show only the active tool's controls.
    // Group presentation never modifies a control's own hidden state.
    groups,
    annotation({pointer, note, swatches}) {
      adopt(pointer, groups.pointer); adopt(note, groups.note); adopt(swatches, groups.pointerColors);
    },
    sketch({pen, colors, undo, redo, done, retry, continueDrawing}) {
      adopt(pen, groups.pen); adopt(colors, groups.penColors);
      for (const node of [undo, redo, done, retry, continueDrawing]) adopt(node, groups.history);
    },
    sketchVisibility(active) {
      if (disposed) return;
      groups.penColors.hidden = false; groups.history.hidden = !active;
    },
    refresh,
    dispose() {
      if (disposed) return;
      disposed = true;
      win.removeEventListener('resize', refresh);
      for (const type of events) root.removeEventListener(type, stop);
      // Restore in original sibling order even if controls were adopted in a
      // different order. Controls removed/moved by adapters stay where put.
      for (const [original, siblings] of origins) {
        let before = null;
        for (const node of [...siblings].reverse()) {
          const record = adopted.get(node);
          if (record?.original === original && node.parentNode === record.group) original.insertBefore(node, before);
          if (node.parentNode === original) before = node;
        }
      }
      for (const [node, record] of adopted) {
        if (!record.original && node.parentNode === record.group) node.remove();
      }
      root.remove();
    },
  };
}
