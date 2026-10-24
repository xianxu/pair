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
  const root=make('div','sketch-note-controls'),summary=make('button','sketch-notes-summary',root);
  summary.type='button';summary.hidden=true;summary.setAttribute('aria-expanded','false');
  const list=make('div','sketch-notes-list'),header=make('div','sketch-notes-list-header',list),title=make('span','',header),listClose=make('button','sketch-notes-list-close',header),body=make('div','sketch-notes-list-body',list);
  list.hidden=true;list.setAttribute('role','group');list.setAttribute('aria-label','Sketch notes');
  listClose.type='button';listClose.textContent='Close';
  const view=make('div','sketch-note-view');view.hidden=true;view.setAttribute('role','dialog');view.setAttribute('aria-label','Sketch note');
  const text=make('div','sketch-note-text',view),close=make('button','sketch-note-close',view);
  close.type='button';close.textContent='Close';
  let current=new Map(),buttons=new Map(),selected=null,disposed=false;
  const closeList=()=>{list.hidden=true;summary.setAttribute('aria-expanded','false');};
  summary.addEventListener('click',()=>{
    if(disposed||!current.size)return;
    list.hidden=!list.hidden;summary.setAttribute('aria-expanded',String(!list.hidden));
    if(!list.hidden){view.hidden=true;selected=null;listClose.focus();}
  });
  listClose.addEventListener('click',()=>{closeList();summary.focus();});
  close.addEventListener('click',()=>{view.hidden=true;selected=null;summary.focus();});
  for(const owner of [root,list,view])for(const type of ['pointerdown','pointerup','click','keydown'])owner.addEventListener(type,e=>e.stopPropagation());
  return {
    update(entries,viewport) {
      if(disposed)return;
      current=new Map(entries.map(e=>[e.sketch,e]));
      for(const [key,b] of buttons)if(!current.has(key)){b.remove();buttons.delete(key);}
      summary.hidden=!entries.length;
      summary.textContent=`Sketch notes (${entries.length})`;title.textContent=summary.textContent;
      const ordered=[];let ink=null;
      for(let index=0;index<entries.length;index++) {
        const entry=entries[index];let b=buttons.get(entry.sketch);
        if(!b) {
          b=make('button','sketch-note-control',body);b.type='button';b.dataset.sketch=entry.sketch;
          const key=entry.sketch;
          b.addEventListener('click',()=>{
            const latest=current.get(key);if(disposed||!latest)return;
            closeList();
            if(latest.label){selected=key;text.textContent=latest.label;view.hidden=false;close.focus();}
            else if(latest.editable)onAdd(key);
          });buttons.set(entry.sketch,b);
        }
        b.textContent=`${index+1}. ${entry.label?'View note':'Add note'} — ${entry.sketch}${entry.label?': '+entry.label.slice(0,100):''}`;
        ordered.push(b);
        const bounds=entry.bounds;
        ink=ink?{left:Math.min(ink.left,bounds.left),top:Math.min(ink.top,bounds.top),right:Math.max(ink.right,bounds.right),bottom:Math.max(ink.bottom,bounds.bottom)}:{...bounds};
      }
      if(body.children.length!==ordered.length||ordered.some((b,i)=>body.children[i]!==b))body.replaceChildren(...ordered);
      if(!entries.length){closeList();view.hidden=true;selected=null;return;}
      // Only the single summary needs geometry. Rows live in normal scrolling
      // flow, so coincident sketches never require collision packing.
      const available=viewport||parent.getBoundingClientRect(),width=Math.max(0,available.width),height=Math.max(0,available.height);
      summary.style.maxWidth=`${width}px`;summary.style.maxHeight=`${height}px`;summary.style.minHeight=`${Math.min(44,height)}px`;
      const measured=summary.getBoundingClientRect(),w=Math.min(width,measured.width),h=Math.min(height,measured.height);
      const clamp=(value,max)=>Math.max(0,Math.min(Math.max(0,max),value));
      let x=clamp(ink.left,width-w),y;
      if(ink.top>=h+8)y=ink.top-h-8;
      else if(ink.bottom+h+8<=height)y=ink.bottom+8;
      else if(ink.right+w+8<=width){x=ink.right+8;y=clamp(ink.top,height-h);}
      else if(ink.left>=w+8){x=ink.left-w-8;y=clamp(ink.top,height-h);}
      else y=clamp(ink.top-h-8,height-h);
      x=clamp(x,width-w);y=clamp(y,height-h);
      summary.style.left=`${x}px`;summary.style.top=`${y}px`;
      const listWidth=Math.min(300,width),listHeight=Math.min(320,height);
      list.style.width=`${listWidth}px`;list.style.height=`${listHeight}px`;
      list.style.left=`${clamp(x,width-listWidth)}px`;
      list.style.top=`${clamp(y+h+8,height-listHeight)}px`;
      if(selected) {
        const latest=current.get(selected);
        if(!latest?.label){view.hidden=true;selected=null;}else text.textContent=latest.label;
      }
    },
    dispose(){if(disposed)return;disposed=true;current.clear();buttons.clear();root.remove();list.remove();view.remove();},
  };
}
