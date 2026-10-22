// Live placement only: historical replay deliberately uses its recorded boxes.
// Both browser adapters share one bounded walk, evidence grammar and visibility rule.
import { MAX_ANCHOR_CANDIDATES, normalizeAnchorText, scoreAnchor, chooseAnchor, projectRegion } from './annotation-anchor.js';

const containers = new Set('header nav main aside footer form section article dialog fieldset figure table tr ul ol dl li details menu search'.split(' '));
const roles = new Set('banner navigation main complementary contentinfo form search region dialog alertdialog article list listitem table row grid group toolbar menu menubar tablist tabpanel figure'.split(' '));
const attr = (n, k) => n.getAttribute?.(k) || '';
const parent = n => n.parentNode?.nodeType === 1 ? n.parentNode : n.parentNode?.host || null;
const step = n => n.localName + (attr(n,'id') && attr(n,'id').length <= 40 ? '#' + attr(n,'id') : attr(n,'role') && attr(n,'role').length <= 40 ? '[' + attr(n,'role') + ']' : '');
function container(n) {
  if (containers.has(n.localName) || roles.has(attr(n,'role')) || attr(n,'aria-modal') === 'true') return true;
  if (!('children' in n ? n.children.length : Array.prototype.some.call(n.childNodes, c => c.nodeType === 1))) return false;
  const classes = attr(n,'class').toLowerCase().split(/\s+/).map(c => c.split(':').at(-1));
  return classes.some(c => /^(card|panel|modal|dialog|popover|toolbar|navbar|sidebar|drawer|tile)/.test(c)) || (classes.some(c => c.startsWith('rounded')) && classes.some(c => /^(border$|shadow$|shadow-)/.test(c)));
}
function facts(n, text) {
  const path = [];
  for (let p=n, depth=0; p && depth<128; p=parent(p),depth++) {
    if ((p.localName==='body'||p.localName==='html') && !p.getRootNode?.().host) break;
    if (p===n || container(p)) path.unshift(step(p));
  }
  const stable = {};
  for (const k of ['id','data-testid','data-test','data-cy']) if (attr(n,k)) stable[k]=attr(n,k);
  return {node:n,tag:n.localName,text:normalizeAnchorText(text),stable,path:path.slice(-6),classes:attr(n,'class').split(/\s+/).filter(c=>c&&c.length<=80).slice(0,16)};
}
function intersect(a,b) {
  const x=Math.max(a[0],b[0]), y=Math.max(a[1],b[1]);
  const w=Math.min(a[0]+a[2],b[0]+b[2])-x, h=Math.min(a[1]+a[3],b[1]+b[3])-y;
  return w>0&&h>0 ? [x,y,w,h] : null;
}
const rect = n => { const r=n.getBoundingClientRect(); return [r.left,r.top,r.width,r.height]; };
// Return the anchor's top-page box and its visible clipping rectangle.
function geometry(n, topDocument) {
  if (!n.isConnected || !n.getClientRects?.().length) return null;
  let box=rect(n), clip=null, current=n, depth=0;
  if (!(box[2]>0 && box[3]>0)) return null;
  while (current && depth++<128) {
    if (!current.isConnected || !current.getClientRects?.().length) return null;
    const ownBox = rect(current);
    if (!(ownBox[2] > 0 && ownBox[3] > 0)) return null;
    const win=current.ownerDocument.defaultView;
    let viewport=[0,0,win.innerWidth,win.innerHeight];
    clip=clip ? intersect(clip,viewport) : viewport;
    if (!clip) return null;
    for (let p=current; p; p=parent(p)) {
      if (++depth>128) return null;
      const s=win.getComputedStyle(p);
      if ((s.clipPath && s.clipPath !== 'none') || (s.maskImage && s.maskImage !== 'none') ||
          (s.webkitMaskImage && s.webkitMaskImage !== 'none') || (s.clip && s.clip !== 'auto')) return null;
      if (s.display==='none'||s.visibility==='hidden'||s.visibility==='collapse'||s.contentVisibility==='hidden'||s.opacity==='0') return null;
      const b=rect(p), overflow=s.overflow || '';
      const x=/hidden|clip|scroll|auto/.test(s.overflowX||overflow), y=/hidden|clip|scroll|auto/.test(s.overflowY||overflow);
      if (x || y) {
        const boundary=[x?b[0]:clip[0],y?b[1]:clip[1],x?b[2]:clip[2],y?b[3]:clip[3]];
        clip=intersect(clip,boundary);if(!clip)return null;
      }
    }
    if (current.ownerDocument===topDocument) {
      if (!intersect(box,clip)) return null;
      const dx=win.scrollX||0,dy=win.scrollY||0;
      return {box:[box[0]+dx,box[1]+dy,box[2],box[3]],clip:[clip[0]+dx,clip[1]+dy,clip[2],clip[3]]};
    }
    const frame=win.frameElement;
    if(!frame)return null;
    const transform=frame.ownerDocument.defaultView.getComputedStyle(frame).transform;
    if(transform && transform !== 'none') return null;
    const fr=rect(frame),dx=fr[0]+(frame.clientLeft||0),dy=fr[1]+(frame.clientTop||0);
    clip=intersect(clip,[0,0,frame.clientWidth ?? fr[2],frame.clientHeight ?? fr[3]]);
    if(!clip)return null;
    box=[box[0]+dx,box[1]+dy,box[2],box[3]];
    clip=[clip[0]+dx,clip[1]+dy,clip[2],clip[3]];
    current=frame;
  }
  return null;
}

export function createPlacement({document:documentOf,route,original=()=>null,shadow=n=>n.shadowRoot,exclude=()=>false,onChange=()=>{}}) {
  let observed,observer,cache;
  const invalidate=()=>{cache=null;onChange();};
  const scan=(doc,root=doc)=>{
    if(observed!==doc){observer?.disconnect();observed=doc;cache=null;const Observer=doc.defaultView?.MutationObserver;if(Observer){observer=new Observer(invalidate);observer.observe(doc,{subtree:true,childList:true,attributes:true,characterData:true});}
      doc.defaultView?.addEventListener?.('scroll',invalidate,true);
      doc.defaultView?.addEventListener?.('resize',invalidate);
      doc.defaultView?.addEventListener?.('transitionend',invalidate,true);
      doc.defaultView?.addEventListener?.('animationend',invalidate,true);
    }
    if(observer?.takeRecords().length)cache=null;
    if(root===doc&&cache&&observer)return cache;
    const nodes=[],texts=new Map(),stack=[{node:root,next:0,entered:false}];
    let truncated=false;
    while(stack.length){
      const entry=stack.at(-1),n=entry.node;
      if(!entry.entered){
        entry.entered=true;
        if(exclude(n)){stack.pop();continue;}
        if(nodes.length>=MAX_ANCHOR_CANDIDATES){truncated=true;break;}
        nodes.push(n);
        if(n.nodeType===3){const full=String(n.nodeValue||'');texts.set(n,{text:full.slice(0,4096),overflow:full.length>4096});}
        const sr=n.nodeType===1&&shadow(n);if(sr){observer?.observe(sr,{subtree:true,childList:true,attributes:true,characterData:true});entry.shadow=sr;}
      }
      const children=n.childNodes||[];
      if(entry.next<children.length){stack.push({node:children[entry.next++],next:0,entered:false});continue;}
      if(entry.shadow){const sr=entry.shadow;entry.shadow=null;stack.push({node:sr,next:0,entered:false});continue;}
      stack.pop();
      if(n.nodeType!==3){let text='',overflow=false;for(const c of children){const value=texts.get(c);if(!value)continue;overflow ||= value.overflow || text.length+value.text.length>4096;text=(text+value.text).slice(0,4096);}texts.set(n,{text,overflow});}
      const value=texts.get(n);
      // A truncated whitespace-heavy prefix is not evidence of equal text.
      if(value?.overflow && Array.from(normalizeAnchorText(value.text)).length<200){truncated=true;break;}
    }
    const result={truncated,candidates:truncated?[]:nodes.filter(n=>n.nodeType===1).map(n=>facts(n,texts.get(n)?.text||''))};
    if(root===doc)cache=result;return result;
  };
  return (note,{withClip=false}={})=>{
    try {
      const at=note.at,doc=documentOf();if(!at||!doc||(at.route&&at.route!==route()))return null;
      const node=original(at.node,note);
      let chosen;
      if(node?.isConnected){
        // A reused recording ID with conflicting evidence never selects elsewhere.
        const own=scan(node.ownerDocument,node);if(own.truncated)return null;
        chosen=own.candidates.find(c=>c.node===node);
        if(!chosen||scoreAnchor(at,chosen)<0)return null;
      }else{
        const data=scan(doc);if(data.truncated)return null;
        let scope=doc;
        const parts=(at.sel||'').split(' >>> ');
        for(const selector of parts.slice(0,-1)){
          const hosts=scope.querySelectorAll(selector);if(hosts.length!==1)return null;
          // Never cross frame documents without frozen frame-route evidence.
          if(hosts[0].localName==='iframe'||hosts[0].localName==='frame')return null;
          scope=shadow(hosts[0]);if(!scope)return null;
        }
        chosen=chooseAnchor(at,data.candidates.filter(c=>c.node.getRootNode()===scope),{route:route()});
        if(!chosen)return null;
      }
      const g=geometry(chosen.node,doc);if(!g)return null;
      const box=note.region?projectRegion(note.region,at.box,g.box):g.box;
      return box&&intersect(box,g.clip)?(withClip?{box,clip:g.clip}:box):null;
    }catch{return null;}
  };
}
