import { projectSketchPoint } from './sketch-render.js';

// Reuse the painter's accepted placement pass; inspecting one sample is enough
// to position one private control without walking a sketch's point history.
export function sketchNoteEntries(paths, targets, { metadata, editable, toClient = (x,y)=>[x,y] }) {
  const entries = new Map();
  for (let i=0; i<paths.length; i++) {
    const p=paths[i];
    if (!p || typeof p.sketch!=='string' || entries.has(p.sketch)) continue;
    const target=targets[i]?.box || targets[i];
    if (!Array.isArray(target) || target.length!==4 || !target.every(Number.isFinite) || target[2]<=0 || target[3]<=0) continue;
    const point=p.pts?.[0], projected=point && projectSketchPoint(point,p.at?.box,target);
    if (!projected) continue;
    const xy=toClient(projected[0],projected[1]);
    if (!Array.isArray(xy) || xy.length!==2 || !xy.every(Number.isFinite)) continue;
    const info=metadata(p.sketch), label=typeof info?.label==='string' ? info.label : '';
    const canAdd=!label && editable.has(p.sketch) && ['drawing','interrupted','label-required'].includes(info?.phase);
    if (!label && !canAdd) continue;
    entries.set(p.sketch,{sketch:p.sketch,x:xy[0],y:xy[1],label,editable:canAdd});
  }
  return [...entries.values()];
}

export function mountSketchNoteControls(doc,{parent,onAdd}) {
  const make=(tag,cls,owner=parent)=>{const e=doc.createElement(tag);e.className=cls;owner.appendChild(e);return e;};
  const root=make('div','sketch-note-controls');
  const view=make('div','sketch-note-view');view.hidden=true;view.setAttribute('role','dialog');view.setAttribute('aria-label','Sketch note');
  const text=make('div','sketch-note-text',view),close=make('button','sketch-note-close',view);
  close.type='button';close.textContent='Close';
  let current=new Map(),buttons=new Map(),selected=null,disposed=false;
  close.addEventListener('click',()=>{view.hidden=true;selected=null;});
  for(const type of ['pointerdown','pointerup','click','keydown']) {
    root.addEventListener(type,e=>e.stopPropagation());view.addEventListener(type,e=>e.stopPropagation());
  }
  return {
    update(entries) {
      if(disposed)return;
      current=new Map(entries.map(e=>[e.sketch,e]));
      for(const [key,b] of buttons) if(!current.has(key)){b.remove();buttons.delete(key);}
      for(const entry of entries) {
        let b=buttons.get(entry.sketch);
        if(!b) {
          b=make('button','sketch-note-control',root);b.type='button';b.dataset.sketch=entry.sketch;
          const key=entry.sketch;
          b.addEventListener('click',()=>{
            const latest=current.get(key);if(disposed||!latest)return;
            if(latest.label){selected=key;text.textContent=latest.label;view.hidden=false;close.focus();}
            else if(latest.editable)onAdd(key);
          });buttons.set(entry.sketch,b);
        }
        b.textContent=entry.label?'View note':'Add note';
        b.style.left=`${entry.x}px`;// Keep the 44px control above the ink with an 8px gap. Near the
        // top edge place it below instead, so the control stays reachable.
        b.style.top=`${entry.y < 52 ? entry.y + 8 : entry.y - 52}px`;
      }
      if(selected) {
        const latest=current.get(selected);
        if(!latest?.label){view.hidden=true;selected=null;} else text.textContent=latest.label;
      }
    },
    dispose(){if(disposed)return;disposed=true;current.clear();buttons.clear();root.remove();view.remove();},
  };
}
