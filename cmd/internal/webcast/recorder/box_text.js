function(region) {
  // Current text evidence in the admitted isolated world. No page mutation,
  // unbounded selector queries, or descendant innerText duplication.
  const out = {region, visible_text: [], accessible_names: [], truncated: false,
    scope: 'top-document-and-open-shadow-roots', skipped: []};
  const encoder = new TextEncoder();
  const maxBytes = 8192, maxEntries = 256, maxNodes = 20000;
  let visited = 0, characterWork = 0, entries = 0;
  const bytes = {visible_text: 0, accessible_names: 0};
  const candidates = [];
  const area = {left: Math.max(0, region[0] - scrollX), top: Math.max(0, region[1] - scrollY),
    right: Math.min(innerWidth, region[0] + region[2] - scrollX), bottom: Math.min(innerHeight, region[1] + region[3] - scrollY)};
  const intersects = r => r.width > 0 && r.height > 0 && r.right > area.left && r.left < area.right && r.bottom > area.top && r.top < area.bottom;
  const bounded = s => { s = String(s || ''); if(s.length > maxBytes) {out.truncated = true; return s.slice(0,maxBytes);} return s; };
  function push(group, item, rect) {
    if (!item.text) return;
    if (entries >= maxEntries) { out.truncated = true; return; }
    let size = Object.values(item).reduce((n,s)=>n+encoder.encode(s).length,0);
    if (size > maxBytes - bytes[group]) { out.truncated = true; return; }
    bytes[group] += size; entries++;
    candidates.push({group,item,y:rect.top,x:rect.left,order:candidates.length});
  }
  function roleOf(n) {
    const explicit=n.getAttribute('role'); if(explicit)return explicit;
    if(n.localName==='input')return ({checkbox:'checkbox',radio:'radio',button:'button',submit:'button',reset:'button',range:'slider',number:'spinbutton',search:'searchbox',email:'textbox',tel:'textbox',url:'textbox',text:'textbox'}[(n.getAttribute('type')||'text').toLowerCase()]||'');
    return ({button:'button',a:n.getAttribute('href')?'link':'',img:'img',textarea:'textbox',select:'combobox'}[n.localName] || '');
  }
  const roles=[]; let nameWork=0;
  const stack = [{node: document.documentElement, selector:'html', hidden:false, owner:null}];
  const seen = new WeakSet();
  while (stack.length) {
    if (++visited > maxNodes) { out.truncated = true; break; }
    const {node:n, selector:path, hidden, owner:inheritedOwner} = stack.pop();
    let owner=inheritedOwner;
    if (seen.has(n)) continue; seen.add(n);
    if (n.nodeType === 3) {
      if(hidden) continue;
      const raw=bounded(n.nodeValue); if(!raw)continue;
      const range=document.createRange(); range.setStart(n,0);range.setEnd(n,raw.length);
      const rects=Array.from(range.getClientRects()); const rect=rects.find(intersects);
      if(!rect)continue;
      let text='';
      // The whole node fits: preserve exact characters without per-character work.
      if(rects.every(r=>r.left>=area.left&&r.right<=area.right&&r.top>=area.top&&r.bottom<=area.bottom)) text=raw;
      else for(let i=0;i<raw.length;i++) {
        if(++characterWork>maxNodes){out.truncated=true;break;}
        range.setStart(n,i);range.setEnd(n,i+1);
        if(Array.from(range.getClientRects()).some(intersects))text+=raw[i];
      }
      push('visible_text',{selector:path,text},rect);
      for(let r=owner;r;r=r.parent){if(++nameWork>maxNodes){out.truncated=true;break;}if(!r.explicit){const room=maxBytes-r.text.length;if(text.length>room)out.truncated=true;r.text+=text.slice(0,room);}}
      continue;
    }
    if(n.nodeType!==1 && n.nodeType!==11)continue;
    let skip=hidden, selector=path, rect=null;
    if(n.nodeType===1) {
      const style=getComputedStyle(n);
      if(style.display==='none'||n.getAttribute('aria-hidden')==='true')continue;
      skip=style.visibility==='hidden'||style.visibility==='collapse';
      const id=n.getAttribute('id');
      if(id && id.length<=128){const shadow=path.lastIndexOf(' >>> ');selector=(shadow<0?'':path.slice(0,shadow+5))+'#'+CSS.escape(id);}
      if(selector.length>512){selector=selector.slice(0,512);out.truncated=true;}
      rect=n.getBoundingClientRect();
      if((rect.width<=0||rect.height<=0)&&style.display!=='contents')skip=true;
      if(!skip && intersects(rect)) {
        const attrs={}; for(const kind of ['aria-label','alt','title']) {
          attrs[kind]=bounded(n.getAttribute(kind));
          if(attrs[kind])push('accessible_names',{selector,text:attrs[kind],kind},rect);
        }
        const role=bounded(roleOf(n));
        if(role) {
          const explicit=attrs['aria-label']||attrs.alt||'';
          if(roles.length<maxEntries){owner={selector,role,text:explicit,explicit:!!explicit,title:attrs.title,rect,parent:owner};roles.push(owner);}
          else out.truncated=true;
        }
        if(['iframe','frame','object','embed'].includes(n.localName)){out.truncated=true;let kind='object/embed';if(n.localName==='iframe'||n.localName==='frame'){try{kind=n.contentDocument?'frame':'cross-origin frame';}catch(_){kind='cross-origin frame';}}if(!out.skipped.includes(kind))out.skipped.push(kind);}
      }
    }
    const root=n;
    if(n.shadowRoot)stack.push({node:n.shadowRoot,selector:selector+' >>> ',hidden:skip,owner});
    // Iterate children with a cursor, never materialize an unbounded child list.
    // The stack stays bounded by the same visited-node budget.
    const kids=[];
    let index=0;
    for(let child=root.firstChild;child;child=child.nextSibling) {
      if(kids.length+stack.length+visited>=maxNodes){out.truncated=true;break;}
      if(child.nodeType===1)index++;
      const childPath=selector+(n.nodeType===11?'':' > ')+(child.localName||'*')+':nth-child('+index+')';
      kids.push({node:child,selector:child.nodeType===3?selector:childPath,hidden:skip,owner});
    }
    for(let i=kids.length-1;i>=0;i--)stack.push(kids[i]);
  }
  for(const r of roles)push('accessible_names',{selector:r.selector,text:r.text||r.title,kind:'role',role:r.role},r.rect);
  candidates.sort((a,b)=>a.y-b.y||a.x-b.x||a.order-b.order);
  for(const c of candidates)out[c.group].push(c.item);
  return out;
}
