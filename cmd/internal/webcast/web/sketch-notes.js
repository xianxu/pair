// Accepted identities/metadata own membership. Optional viewport bounds come
// from the painter/cache and describe placement only, never another projection.
export function sketchNoteEntries(paths, paintedBounds, {metadata,editable,viewport}) {
  const entries=new Map(),seen=new Set();
  for(let i=0;i<paths.length;i++) {
    const id=paths[i]?.sketch,b=paintedBounds[i];
    if(typeof id!=='string'||!id)continue;
    if(!seen.has(id)){
      seen.add(id);
      const info=metadata(id),label=typeof info?.label==='string'?info.label:'';
      const canAdd=!label&&editable.has(id)&&['drawing','interrupted','label-required'].includes(info?.phase);
      if(label||canAdd)entries.set(id,{sketch:id,bounds:null,label,editable:canAdd});
    }
    const existing=entries.get(id);
    if(!existing||!b||![b.left,b.top,b.right,b.bottom].every(Number.isFinite)||b.left>b.right||b.top>b.bottom)continue;
    const bounds={left:Math.max(0,b.left),top:Math.max(0,b.top),right:Math.min(viewport.width,b.right),bottom:Math.min(viewport.height,b.bottom)};
    if(!(bounds.left<bounds.right&&bounds.top<bounds.bottom))continue;
    if(existing.bounds) {
      existing.bounds.left=Math.min(existing.bounds.left,bounds.left);existing.bounds.top=Math.min(existing.bounds.top,bounds.top);
      existing.bounds.right=Math.max(existing.bounds.right,bounds.right);existing.bounds.bottom=Math.max(existing.bounds.bottom,bounds.bottom);
      continue;
    }
    existing.bounds=bounds;
  }
  return [...entries.values()];
}
