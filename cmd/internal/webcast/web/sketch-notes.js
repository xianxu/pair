// Bounds come from the existing painter/cache, never another point projection.
export function sketchNoteEntries(paths, paintedBounds, {metadata,editable,viewport}) {
  const entries=new Map();
  for(let i=0;i<paths.length;i++) {
    const id=paths[i]?.sketch,b=paintedBounds[i];
    if(typeof id!=='string'||!b||![b.left,b.top,b.right,b.bottom].every(Number.isFinite)||b.left>b.right||b.top>b.bottom)continue;
    const bounds={left:Math.max(0,b.left),top:Math.max(0,b.top),right:Math.min(viewport.width,b.right),bottom:Math.min(viewport.height,b.bottom)};
    if(!(bounds.left<bounds.right&&bounds.top<bounds.bottom))continue;
    const existing=entries.get(id);
    if(existing) {
      existing.bounds.left=Math.min(existing.bounds.left,bounds.left);existing.bounds.top=Math.min(existing.bounds.top,bounds.top);
      existing.bounds.right=Math.max(existing.bounds.right,bounds.right);existing.bounds.bottom=Math.max(existing.bounds.bottom,bounds.bottom);
      continue;
    }
    const info=metadata(id),label=typeof info?.label==='string'?info.label:'';
    const canAdd=!label&&editable.has(id)&&['drawing','interrupted','label-required'].includes(info?.phase);
    if(label||canAdd)entries.set(id,{sketch:id,bounds,label,editable:canAdd});
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
    update(entries, viewport) {
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
        // Measure the whole labeled target once. Prefer above/below/right/left
        // with an 8px gap; clamp the complete button when space is exhausted.
        const available=viewport || parent.getBoundingClientRect();
        const width=Math.max(0,available.width),height=Math.max(0,available.height);
        b.style.maxWidth=`${width}px`;b.style.maxHeight=`${height}px`;
        const measured=b.getBoundingClientRect(),w=Math.min(width,measured.width),h=Math.min(height,measured.height),ink=entry.bounds;
        const clamp=(value,max)=>Math.max(0,Math.min(Math.max(0,max),value));
        let x=clamp(ink.left,width-w),y;
        if(ink.top>=h+8)y=ink.top-h-8;
        else if(ink.bottom+h+8<=height)y=ink.bottom+8;
        else if(ink.right+w+8<=width){x=ink.right+8;y=clamp(ink.top,height-h);}
        else if(ink.left>=w+8){x=ink.left-w-8;y=clamp(ink.top,height-h);}
        else y=clamp(ink.top-h-8,height-h);
        b.style.left=`${clamp(x,width-w)}px`;b.style.top=`${clamp(y,height-h)}px`;
      }
      if(selected) {
        const latest=current.get(selected);
        if(!latest?.label){view.hidden=true;selected=null;} else text.textContent=latest.label;
      }
    },
    dispose(){if(disposed)return;disposed=true;current.clear();buttons.clear();root.remove();view.remove();},
  };
}
