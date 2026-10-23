import { projectPoint, pressureWidth } from './annotation-anchor.js';

const U64=18446744073709551615n,SAFE=9007199254740991n;
const decimal=v=>Number.isSafeInteger(v)&&v>=0?String(v):v;
const uint=(v,max=U64)=>typeof v==='string'&&/^(0|[1-9][0-9]{0,19})$/.test(v)&&BigInt(v)<=max;
const box=b=>Array.isArray(b)&&b.length===4&&b.every(Number.isFinite)&&b[2]>0&&b[3]>0;
const intersects=(a,b)=>a[0]<b[0]+b[2]&&a[0]+a[2]>b[0]&&a[1]<b[1]+b[3]&&a[1]+a[3]>b[1];
const same=(a,b)=>JSON.stringify(a)===JSON.stringify(b);
const fail=why=>{throw Error(`Feedback visibility unavailable: ${why}`);};

// One host owns one generation. Completed paint data is copied, never a DOM,
// mutable path, note, receiver or renderer reference. Notifications invalidate
// immediately and wake again on completed paint without changing its revision.
export function createFeedbackVisibility({context,mask,busy=()=>false,notify=()=>{}}){
 let revision=0n,notes=null,sketches=null,ready=null,error='',listeners=new Set();
 const frame=()=>{
  const scene=context(),identity=mask();
  if(!scene||!Number.isSafeInteger(scene.doc)||scene.doc<=0||!Number.isSafeInteger(scene.record_index)||scene.record_index<0||!Number.isSafeInteger(scene.t)||scene.t<0||!scene.recording_id||!scene.tab||!scene.url||typeof scene.title!=='string'||!box([...(scene.scroll||[]),scene.vp?.w,scene.vp?.h])||!Number.isFinite(scene.vp?.dpr)||scene.vp.dpr<=0)fail('source context is not ready');
  if(!identity?.scope||!uint(identity.revision))fail('confirmed mask is not ready');
  return {scene:structuredClone(scene),mask_scope:identity.scope,mask_revision:identity.revision};
 };
 const token=()=>String(revision);
 const invalidate=()=>{
  if(revision===U64)fail('paint revision exhausted');
  revision++;notes=null;sketches=null;ready=null;error='';notify(token());
  for(const fn of listeners)fn();return token();
 };
 const complete=()=>{
  if(!notes||!sketches||error||busy())return;
  if(ready)return;
  const current=frame();
  if(!same(current,notes.frame)||!same(current,sketches.frame)){invalidate();return;}
  ready={revision:token(),...current,sketches:sketches.items,notes:notes.items};
  if(new TextEncoder().encode(JSON.stringify(ready)).length>64*1024*1024){ready=null;error='serialized paint exceeds 64 MiB';return;}
  notify(token());
 };
 const commit=(stamp,kind,build)=>{
  if(stamp!==token()||revision===0n)return false;
  try{
   const current=frame(),items=build(current.scene);
   if(kind==='notes')notes={frame:current,items};else sketches={frame:current,items};
   complete();return true;
  }catch(e){ready=null;if(error!==e.message){error=e.message;notify(token());}return false;}
 };
 return {
  token,invalidate,
  onInvalidate(fn){listeners.add(fn);return()=>listeners.delete(fn);},
  notesPaint(stamp,painted){return commit(stamp,'notes',scene=>{
   const viewport=[...scene.scroll,scene.vp.w,scene.vp.h],items=[];
   for(const n of painted){if(!box(n.box)||!intersects(n.box,viewport))continue;if(!n.id||!uint(n.revision))fail('painted annotation revision is missing');items.push({id:n.id,revision:n.revision});}
   return items;
  });},
  sketchesPaint(stamp,paths,targets,metadata){return commit(stamp,'sketches',scene=>{
   const viewport=[...scene.scroll,scene.vp.w,scene.vp.h],groups=new Map();
   for(let i=0;i<paths.length;i++){
    const p=paths[i],meta=metadata(p.sketch);
    if(!['complete','interrupted'].includes(meta?.phase)||!meta.label?.trim())continue;
    let group=groups.get(p.sketch);if(!group){group={meta,paths:[],missing:0};groups.set(p.sketch,group);}
    const target=targets[i];
    if(!box(target?.box)){group.missing++;continue;}
    let left=Infinity,top=Infinity,right=-Infinity,bottom=-Infinity,valid=true;
    for(const raw of p.pts){const point=projectPoint(raw,p.at?.box,target.box),radius=pressureWidth(p.width,point?.[2])/2;if(!point||!Number.isFinite(radius)){valid=false;break;}left=Math.min(left,point[0]-radius);top=Math.min(top,point[1]-radius);right=Math.max(right,point[0]+radius);bottom=Math.max(bottom,point[1]+radius);}
    if(!valid||!box([left,top,right-left,bottom-top])||!intersects([left,top,right-left,bottom-top],viewport)){group.missing++;continue;}
    if(!box(target.clip)||!same(target.clip,viewport))fail(`sketch ${p.sketch}: element clipping is not supported by the live painter`);
    if(!Array.isArray(target.locator)||!target.locator.length||target.locator.length>128||!target.locator.every(n=>Number.isSafeInteger(n)&&n>=-1&&n<=1048576))fail(`sketch ${p.sketch}: recorder locator is unavailable`);
    group.paths.push({id:p.id,locator:[...target.locator],box:[...target.box],clip:[...target.clip]});
   }
   const items=[];
   for(const [id,g]of groups){
    if(!g.paths.length)continue;
    if(g.missing)fail(`sketch ${id}: partial path placement`);
    const sequence=decimal(g.meta.next_sequence);
    if(!uint(sequence,SAFE)||sequence==='0'||!/^[0-9a-f]{64}$/.test(g.meta.digest||''))fail(`sketch ${id}: accepted version is unavailable`);
    items.push({id,next_sequence:sequence,digest:g.meta.digest,paths:g.paths});
   }
   return items;
  });},
  unavailable(stamp,reason){if(stamp!==token())return;ready=null;if(error!==reason){error=reason;notify(token());}},
  read(){
   if(busy())fail('drawing or a draft is busy');
   if(error)fail(error);
   if(!ready)fail('both paints must complete');
   const current=frame();
   if(!same(current,{scene:ready.scene,mask_scope:ready.mask_scope,mask_revision:ready.mask_revision})){invalidate();fail('source or mask changed; stale paint');}
   return structuredClone(ready);
  },
  dispose(){listeners.clear();ready=null;error='context retired';},
 };
}
