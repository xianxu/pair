import { feedbackEntries } from './feedback-entries.js';
import { mountFloatingPanel } from './floating-panel.js';

// Projection and presentation only. Annotation/sketch owners supply all actions.
export function mountFeedbackPanel(doc, win, {parent, layout = () => null}) {
  const element = (tag, cls, text) => {
    const node = doc.createElement(tag); node.className = cls;
    if (text !== undefined) node.textContent = text;
    return node;
  };
  const body = element('div', 'feedback-body');
  const list = element('div', 'feedback-rows');
  const readonly = element('div', 'feedback-readonly');
  const readonlyText = element('div', 'feedback-readonly-text');
  const readonlyClose = element('button', 'feedback-readonly-close', 'Back');
  readonlyClose.type = 'button'; readonly.hidden = true;
  readonly.append(readonlyText, readonlyClose); body.append(list, readonly);
  let disposed = false, enabled = true, floating;
  let notes = [], sketches = [], latest = new Map(), activeHover = null, activeKey = null, readonlyKey = null;
  const handlers = {note:{}, sketch:{}};
  const rows = new Map(), editors = new Map(), statuses = new Map(), removers = [];
  const listen = (node, type, fn, owned = removers) => {
    node.addEventListener(type, fn);
    owned.push(() => node.removeEventListener?.(type, fn));
  };
  const clearHover = () => {
    const previous = activeHover; activeHover = null; activeKey = null;
    previous?.(null);
  };
  const geometry = mode => {
    const bounds = layout();
    if (!bounds) return null;
    return {viewport:bounds.content, size:mode === 'minimized' ? undefined : bounds.panel,
      left:bounds.initial.left, top:bounds.initial.top};
  };
  floating = mountFloatingPanel(doc, win, {parent, body, title:'Notes', className:'feedback-panel',
    showEmpty:true, geometry,
    onPresentation({minimized, hidden}) {
      if (minimized || hidden || !enabled) clearHover();
      if (!enabled && floating) floating.root.hidden = true;
    }});
  const status = element('div', 'feedback-status', '');
  status.setAttribute('role', 'status'); status.setAttribute('aria-live', 'polite');
  floating.header.append(status);
  const editorOpen = () => [...editors.keys()].some(node => node.parentNode === body && !node.hidden && !node.classList.contains('hidden'));
  const closeReadonly = () => {
    readonly.hidden = true; readonlyKey = null;
    if (readonlyText.textContent) readonlyText.textContent = '';
  };
  const showReadonly = (text, key = null) => {
    if (disposed || editorOpen()) return false;
    clearHover(); readonlyKey = key; readonlyText.textContent = String(text ?? ''); readonly.hidden = false;
    return true;
  };
  listen(readonlyClose, 'click', () => { if (!disposed) closeReadonly(); });
  const createRow = key => {
    const node = element('div', 'feedback-row'); node.dataset.key = key;
    const icon = element('span', 'feedback-icon'); icon.setAttribute('aria-hidden', 'true');
    const text = element('div', 'feedback-text');
    const location = element('div', 'feedback-location', '');
    const open = element('button', 'feedback-open'); open.type = 'button';
    const reveal = element('button', 'feedback-reveal', 'Show on page'); reveal.type = 'button';
    const actions = element('div', 'feedback-actions'); actions.append(open, reveal); node.append(icon, text, location, actions);
    const owned = [];
    listen(open, 'click', () => {
      if (disposed || !enabled) return;
      const row = latest.get(key); if (!row) return;
      clearHover();
      if (row.kind === 'sketch' && !row.editable) showReadonly(row.text, row.key);
      else { closeReadonly(); handlers[row.kind].open?.(row.source); }
    }, owned);
    listen(reveal, 'click', () => {
      if (disposed || !enabled) return;
      const row = latest.get(key); if (row) handlers[row.kind].reveal?.(row.source);
    }, owned);
    const hover = () => {
      if (disposed || !enabled || floating.root.hidden || floating.root.classList.contains('minimized')) return;
      const row = latest.get(key); if (!row) return;
      clearHover(); activeKey = key; activeHover = handlers[row.kind].hover || null;
      activeHover?.(row.source);
    };
    listen(node, 'mouseenter', hover, owned); listen(node, 'focusin', hover, owned);
    listen(node, 'mouseleave', clearHover, owned);
    listen(node, 'focusout', event => { if (!node.contains(event.relatedTarget)) clearHover(); }, owned);
    return {node, icon, text, location, open, dispose:() => { for (const remove of owned) remove(); node.remove(); }};
  };
  const reconcile = () => {
    const hoveredKey = activeKey;
    const projected = feedbackEntries(notes, sketches);
    latest = new Map(projected.map(row => [row.key, row]));
    if (!readonly.hidden) {
      const selected = latest.get(readonlyKey);
      if (!selected) closeReadonly();
      else if (readonlyText.textContent !== selected.text) readonlyText.textContent = selected.text;
    }
    for (const [key, row] of rows) if (!latest.has(key)) { row.dispose(); rows.delete(key); }
    for (let index = 0; index < projected.length; index++) {
      const entry = projected[index]; let row = rows.get(entry.key);
      if (!row) { row = createRow(entry.key); rows.set(entry.key, row); }
      const icon = entry.kind === 'note' ? '▤' : '✎';
      const action = entry.kind === 'note' ? 'Open note' : entry.text || !entry.editable ? 'View note' : 'Add note';
      if (row.icon.textContent !== icon) row.icon.textContent = icon;
      if (row.text.textContent !== entry.text) row.text.textContent = entry.text;
      if (row.open.textContent !== action) row.open.textContent = action;
      const available = handlers[entry.kind].available?.(entry.source);
      const placement = available === true ? 'visible' : available === false ? 'unavailable' : 'unknown';
      if (row.node.dataset.placement !== placement) row.node.dataset.placement = placement;
      const location = available === false ? 'not on this view' : '';
      if (row.location.textContent !== location) row.location.textContent = location;
      if (list.children[index] !== row.node) {
        if (list.children[index]) list.insertBefore(row.node, list.children[index]);
        else list.append(row.node);
      }
    }
    floating.setCount(projected.length);
    if (!enabled) floating.root.hidden = true;
    if (hoveredKey) {
      clearHover();
      const entry = latest.get(hoveredKey);
      if (entry && enabled && !floating.root.hidden && !floating.root.classList.contains('minimized')) {
        activeKey = hoveredKey; activeHover = handlers[entry.kind].hover || null; activeHover?.(entry.source);
      }
    }
  };
  return {root:floating.root, body, header:floating.header, status,
    setNotes(value, actions = {}) { if (disposed) return; notes = value || []; handlers.note = actions; reconcile(); },
    setSketches(value, actions = {}) { if (disposed) return; sketches = value || []; handlers.sketch = actions; reconcile(); },
    setStatus(source, text) {
      if (disposed) return;
      statuses.set(source, String(text ?? ''));
      const primary = ['editor','sketch','control'].map(key => statuses.get(key)).find(Boolean);
      const textValue = [...new Set([primary, statuses.get('preview')].filter(Boolean))].join(' · ');
      if (status.textContent !== textValue) status.textContent = textValue;
    },
    setEnabled(value) { if (disposed || enabled === !!value) return; enabled = !!value; clearHover(); floating.refresh(); if (!enabled) floating.root.hidden = true; },
    refresh() { if (!disposed) { clearHover(); floating.refresh(); if (!enabled) floating.root.hidden = true; } },
    adoptEditor(node) {
      if (disposed || !node || editors.has(node)) return;
      editors.set(node, {parent:node.parentNode, next:node.nextSibling, hadClass:node.classList.contains('feedback-editor')});
      node.classList.add('feedback-editor'); body.append(node); closeReadonly(); clearHover();
    },
    showReadonly,
    clearHighlight() { if (!disposed) clearHover(); },
    dispose() {
      if (disposed) return;
      clearHover(); disposed = true;
      for (const row of rows.values()) row.dispose(); rows.clear(); latest.clear();
      for (const remove of removers) remove();
      for (const [node, original] of editors) {
        if (!original.hadClass) node.classList.remove('feedback-editor');
        if (original.parent) {
          if (original.next?.parentNode === original.parent) original.parent.insertBefore(node, original.next);
          else original.parent.append(node);
        } else node.remove();
      }
      editors.clear(); statuses.clear(); notes = []; sketches = []; handlers.note = {}; handlers.sketch = {};
      floating.dispose();
    }};
}
