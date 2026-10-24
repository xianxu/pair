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
