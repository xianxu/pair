import { mountSketch } from './web/sketch.js';
import { createSketchDisplay } from './web/sketch-render.js';
import { createSketchDelivery, createSketchTransfer } from './web/sketch-delivery.js';

// One context owns this transport. Its operation permissions survive timeout
// and document-only updates; destruction closes every pending promise.
export function createOperatorSketchTransport(report, win, generation) {
  const pending=new Map(), origins=new Map();let serial=0,closed=false;
  const key=b=>`${b.epoch}:${b.sequence===0?'':b.sketch||''}:${b.operation}`;
  function ask(kind,body={},epoch=body.epoch,timeout=5000){
    if(closed)return Promise.reject(Error('Sketch context retired.'));
    if(pending.size>=8)return Promise.reject(Error('Sketch requests are busy.'));
    const operation=kind.startsWith('sketch-')&&body.operation?key(body):null;
    if(operation&&!origins.has(operation)){
      if(origins.size>=64)return Promise.reject(Error('Sketch confirmation is pending.'));
      origins.set(operation,generation());
    }
    const origin=operation?origins.get(operation):generation();
    if(typeof origin!=='string'||!/^(0|[1-9][0-9]{0,19})$/.test(origin)||BigInt(origin)>18446744073709551615n||!Number.isSafeInteger(epoch)||epoch<0)return Promise.reject(Error('Sketch authority is unavailable.'));
    const request=`sketch-${++serial}`;
    return new Promise((resolve,reject)=>{
      const timer=win.setTimeout(()=>{pending.delete(request);reject(Error('Sketch confirmation is pending.'));},timeout);
      pending.set(request,{resolve,reject,timer,operation});
      report({request,kind,generation:origin,epoch,body:structuredClone(body)});
    });
  }
  return {ask,reply(reply){
    const p=pending.get(reply?.request);if(!p)return false;
    pending.delete(reply.request);win.clearTimeout(p.timer);
    if(reply.ok){if(p.operation&&['accepted','refused'].includes(reply.value?.outcome))origins.delete(p.operation);p.resolve(reply.value);}
    else p.reject(Error(reply.error||'Sketch request was not confirmed.'));
    return true;
  },close(){closed=true;for(const p of pending.values()){win.clearTimeout(p.timer);p.reject(Error('Sketch context retired.'));}pending.clear();origins.clear();}};
}

export function createOperatorSketchReceiver(changed){
  const display=createSketchDisplay(),transfer=createSketchTransfer();let completed=null,retired=false;
  const delivery=createSketchDelivery({installBootstrap:value=>display.install(value),applyBatch:patches=>display.applyBatch(patches)});
  return {paths:()=>display.paths({borrow:true}),hasAnchor:(document,node)=>display.hasAnchor(document,node),cursor:()=>delivery.cursor(),receive(message){
    if(retired)throw Error('Sketch context retired.');
    if(message.phase==='begin'){
      const current=transfer.cursor();
      if(current?.handle===message.handle&&current.epoch===message.epoch&&current.total===message.total)return {offset:current.offset,...delivery.cursor()};
      if(completed?.handle===message.handle&&completed.epoch===message.epoch&&completed.total===message.total)return completed.ack;
      transfer.retire();completed=null;
      return transfer.begin(message);
    }
    if(message.phase!=='chunk'||typeof message.base64!=='string'||message.base64.length>43692)throw Error('Invalid private delivery chunk.');
    if(completed?.handle===message.handle&&completed.epoch===message.epoch&&completed.total===message.total&&completed.offset===message.offset){if(completed.base64!==message.base64)throw Error('Conflicting private delivery.');return completed.ack;}
    const text=atob(message.base64),bytes=Uint8Array.from(text,c=>c.charCodeAt(0));
    const result=transfer.append({...message,bytes});
    if(!result.complete)return {offset:result.offset};
    const payload=JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(result.value));
    let ack;
    if(payload.bootstrap&&!payload.batch)ack=delivery.receiveBootstrap(payload.bootstrap);
    else if(payload.batch&&!payload.bootstrap)ack=delivery.receiveBatch(payload.batch);
    else throw Error('Invalid private drawing delivery.');
    ack={offset:result.offset,...ack};completed={handle:message.handle,epoch:message.epoch,total:message.total,offset:message.offset,base64:message.base64,ack};
    transfer.retire();changed();return ack;
  },retire(){retired=true;transfer.retire();delivery.retire();completed=null;display.install({sketches:[]});}};
}

export function mountOperatorSketch({host,root,box,report,view,annotation,palette,css,shellURL},doc,win){
  const style=doc.createElement('style');style.textContent=css;root.appendChild(style);
  let authority={epoch:0,holder:'operator',ready:false,generation:null},frame=null,ready=null,disposed=false,bypass=false,sceneVersion=0,opening=null;
  const transport=createOperatorSketchTransport(envelope=>report({t:'sketch',request:envelope}),win,()=>authority.generation);
  let tool;
  const receiver=createOperatorSketchReceiver(()=>tool?.refresh());
  const ask=(kind,body={},epoch=authority.epoch)=>transport.ask(kind,body,epoch,kind==='scene-prepare'?60000:5000);
  const bounds=()=>box.getBoundingClientRect();
  const scale=()=>ready?.vp?Math.min(bounds().width/ready.vp.w,bounds().height/ready.vp.h):1;
  const origin=()=>{const b=bounds(),s=scale();return [b.left+(b.width-(ready?.vp.w||b.width)*s)/2,b.top+(b.height-(ready?.vp.h||b.height)*s)/2];};
  function layoutScene(){if(!frame||!ready)return;const [left,top]=origin(),s=scale();Object.assign(frame.style,{left:left+'px',top:top+'px',width:ready.vp.w+'px',height:ready.vp.h+'px',transformOrigin:'0 0',transform:`scale(${s})`});}
  const scene={
    get ready(){return !!ready;},
    preflight(){
      if(disposed)return Promise.reject(Error('Sketch context retired.'));
      if(opening)return opening.promise;
      if(frame)return Promise.resolve();
      const next=doc.createElement('iframe'),version=sceneVersion;
      next.name='opaque-private-scene';next.setAttribute('sandbox','allow-same-origin');next.setAttribute('title','Recorded sketch scene');next.className='sketch-private-scene';next.hidden=true;
      next.style.cssText='position:fixed;inset:0;width:100%;height:100%;border:0;pointer-events:none';
      const attempt={};
      const canceled=new Promise((_,reject)=>{attempt.cancel=()=>reject(Error('Sketch context retired.'));});
      let timeout;
      const loaded=new Promise((resolve,reject)=>{timeout=win.setTimeout(()=>reject(Error('Recorded scene did not load.')),5000);next.onload=resolve;next.onerror=()=>reject(Error('Recorded scene is blocked.'));});
      const check=()=>{if(disposed||version!==sceneVersion||frame!==next)throw Error('Sketch context retired.');};
      frame=next;opening=attempt;win.__couchcastPrivateSceneFrame=next;
      attempt.promise=(async()=>{
        try{
          await Promise.race([loaded,canceled]);check();
          await Promise.race([ask('scene-open'),canceled]);check();
        }catch(error){
          next.remove();
          if(frame===next)frame=null;
          if(win.__couchcastPrivateSceneFrame===next)delete win.__couchcastPrivateSceneFrame;
          throw error;
        }finally{
          win.clearTimeout(timeout);next.onload=null;next.onerror=null;
          if(opening===attempt)opening=null;
        }
      })();
      next.src=shellURL;root.insertBefore(next,box);
      return attempt.promise;
    },
    async prepare(source,{sketch,epoch}){const version=sceneVersion;const result=await ask('scene-prepare',{sketch,source},epoch);if(disposed||version!==sceneVersion)throw Error('Sketch context retired.');ready=result;layoutScene();},
    hitTest:point=>ask('scene-hit',{point}),
    placement:path=>ask('scene-placement',{path:{source:path.source,at:path.at}}),
    toPage(x,y){const [left,top]=origin(),s=scale(),scroll=ready?.scroll||[0,0];return [(x-left)/s+scroll[0],(y-top)/s+scroll[1]];},
    fromPage(x,y){const [left,top]=origin(),s=scale(),scroll=ready?.scroll||[0,0];return [left+(x-scroll[0])*s,top+(y-scroll[1])*s];},
    dispose(){sceneVersion++;ready=null;opening?.cancel();opening=null;if(frame){const owned=frame;frame=null;owned.remove();if(win.__couchcastPrivateSceneFrame===owned)delete win.__couchcastPrivateSceneFrame;ask('scene-dispose').catch(()=>{});}},
  };
  const current=()=>({allowed:authority.ready&&authority.holder==='operator',epoch:authority.epoch,live:!view.model.paused&&!view.model.ended&&view.model.seg>0,ended:!!view.model.ended});
  tool=mountSketch(doc,win,view,{root,box,scene,palette,paths:receiver.paths,authority:current,send:(kind,body)=>ask(kind,body,body.epoch),status:body=>ask('sketch-status',body,body.epoch),onMode(active){bypass=true;try{annotation.setMode('watch');}finally{bypass=false;}if(frame)frame.hidden=!active;},onTool(mode){bypass=true;try{annotation.setMode(mode);}finally{bypass=false;}}});
  const previousBeforeMode=view.beforeAnnotationMode;
  const beforeMode=mode=>{if(disposed||bypass||!tool.state().active)return true;tool.leave(mode);return false;};
  view.beforeAnnotationMode=beforeMode;
  const unRecord=view.onRecord(()=>{if(!disposed)tool.setAuthority(current());});
  const reply=reply=>transport.reply(reply);
  const deliver=payload=>receiver.receive(payload);
  let authorityRevision=null;
  const authorityReceipt=()=>({revision:authorityRevision??'0',generation:authority.generation,epoch:authority.epoch,ready:authority.ready});
  const authorityUint=value=>typeof value==='string'&&/^(0|[1-9][0-9]{0,19})$/.test(value)&&BigInt(value)<=18446744073709551615n;
  const updateAuthority=value=>{
    if(disposed)return {revision:authorityRevision??'0',epoch:0,generation:null,ready:false};
    if(!authorityUint(value?.revision))throw Error('Invalid sketch authority revision.');
    if(authorityRevision!==null&&BigInt(value.revision)<=BigInt(authorityRevision))return authorityReceipt();
    if(!authorityUint(value.generation)||!Number.isSafeInteger(value.epoch)||value.epoch<0||typeof value.holder!=='string'||typeof value.ready!=='boolean'||typeof value.active!=='boolean')throw Error('Invalid sketch authority projection.');
    authority={...value,ready:value.ready&&value.active&&value.holder==='operator'};
    authorityRevision=value.revision;
    tool.setAuthority(current());
    return authorityReceipt();
  };
  win.__couchcastSketchReply=reply;
  win.__couchcastSketchDelivery=deliver;
  win.__couchcastSketchAuthority=updateAuthority;
  const resize=()=>{if(disposed)return;layoutScene();tool.refresh({geometry:true});};
  const dispose=()=>{
    if(disposed)return;disposed=true;tool.dispose();receiver.retire();transport.close();unRecord?.();style.remove();
    if(win.__couchcastSketchReply===reply)delete win.__couchcastSketchReply;
    if(win.__couchcastSketchDelivery===deliver)delete win.__couchcastSketchDelivery;
    if(win.__couchcastSketchAuthority===updateAuthority)delete win.__couchcastSketchAuthority;
    if(view.beforeAnnotationMode===beforeMode)view.beforeAnnotationMode=previousBeforeMode;
    win.removeEventListener?.('resize',resize);win.removeEventListener?.('pagehide',dispose);
  };
  win.addEventListener('resize',resize);
  win.addEventListener('pagehide',dispose,{once:true});
  return {tool,dispose,transport,receiver,scene};
}
