import { mountSketch } from './web/sketch.js';
import { createSketchDisplay, paintSketchPaths } from './web/sketch-render.js';
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
  return {metadata:(sketch,stroke)=>display.metadata(sketch,stroke),paths:()=>display.paths({borrow:true}),hasAnchor:(document,node)=>display.hasAnchor(document,node),cursor:()=>delivery.cursor(),receive(message){
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
    transfer.retire();if(!ack.needBootstrap)changed(payload);return ack;
  },retire(){retired=true;transfer.retire();delivery.retire();completed=null;display.install({sketches:[]});}};
}

export function mountOperatorSketch({host,root,box,report,view,annotation,palette,css,shellURL},doc,win){
  const style=doc.createElement('style');style.textContent=css;root.appendChild(style);
  let authority={epoch:0,holder:'operator',ready:false,generation:null},frame=null,ready=null,disposed=false,bypass=false,sceneVersion=0,opening=null;
  const transport=createOperatorSketchTransport(envelope=>report({t:'sketch',request:envelope}),win,()=>authority.generation);
  let tool;
  const receiver=createOperatorSketchReceiver(payload=>{tool?.reconcileSketches({authoritative:true,clear:payload.batch?.patches.some(p=>p.clear===true)});tool?.refresh();});
  const submissionPreview=createOperatorSubmissionPreview({root,receiver},doc);
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
  tool=mountSketch(doc,win,view,{root,box,scene,palette,paths:receiver.paths,sketchMetadata:receiver.metadata,authority:current,showUnavailable:true,unavailableHint:()=>authority.holder==='operator'?'Drawing is getting ready. Wait a moment to sketch.':'A viewer is driving. Ask them to release control so you can sketch.',send:(kind,body)=>ask(kind,body,body.epoch),status:body=>ask('sketch-status',body,body.epoch),onMode(active){bypass=true;try{annotation.setMode('watch');}finally{bypass=false;}if(frame)frame.hidden=!active;},onTool(mode){bypass=true;try{annotation.setMode(mode);}finally{bypass=false;}}});
  const previousBeforeMode=view.beforeAnnotationMode;
  const beforeMode=mode=>disposed||bypass?true:tool.beforeAnnotationMode(mode);
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
  win.__couchcastSketchSubmissionPreview=submissionPreview.publish;
  const resize=()=>{if(disposed)return;layoutScene();tool.refresh({geometry:true});};
  const dispose=()=>{
    if(disposed)return;disposed=true;submissionPreview.dispose();tool.dispose();receiver.retire();transport.close();unRecord?.();style.remove();
    if(win.__couchcastSketchReply===reply)delete win.__couchcastSketchReply;
    if(win.__couchcastSketchDelivery===deliver)delete win.__couchcastSketchDelivery;
    if(win.__couchcastSketchAuthority===updateAuthority)delete win.__couchcastSketchAuthority;
    if(win.__couchcastSketchSubmissionPreview===submissionPreview.publish)delete win.__couchcastSketchSubmissionPreview;
    if(view.beforeAnnotationMode===beforeMode)view.beforeAnnotationMode=previousBeforeMode;
    win.removeEventListener?.('resize',resize);win.removeEventListener?.('pagehide',dispose);
  };
  win.addEventListener('resize',resize);
  win.addEventListener('pagehide',dispose,{once:true});
  return {tool,dispose,transport,receiver,scene,captureBusy:()=>!disposed&&(tool.captureBusy()||submissionPreview.captureBusy())};
}


// A read-only projection of the manager's frozen candidate identities. The
// receiver owns accepted paths; this view never sends, selects an agent or
// acknowledges a delivery. Go additionally fences its exact board generation.
export function createOperatorSubmissionPreview({root,receiver},doc){
  const panel=doc.createElement('section'),heading=doc.createElement('div'),list=doc.createElement('div');
  panel.className='sketch-submission-preview';heading.className='sketch-submission-count';list.className='sketch-submission-cards';
  panel.setAttribute('aria-label','Pending feedback');
  panel.style.cssText='position:fixed;right:12px;bottom:90px;width:224px;max-height:42vh;overflow:auto;z-index:36;pointer-events:auto;background:#202124;color:#fff;border:1px solid #777;border-radius:6px;padding:8px;font:12px sans-serif';
  panel.hidden=true;panel.append(heading,list);root.append(panel);
  let disposed=false,last=null,fingerprint=null,cards=new Map(),noteCount=0;
  const uint=value=>typeof value==='string'&&/^(0|[1-9][0-9]{0,19})$/.test(value)&&BigInt(value)<=18446744073709551615n;
  const copyAck=()=>structuredClone(last);
  function card(key,item,paths,note){
    const node=doc.createElement('div'),canvas=doc.createElement('canvas'),label=doc.createElement('div'),index=doc.createElement('div');
    node.className='sketch-submission-card';node.dataset.version=key;
    node.style.cssText='margin-top:6px;border-top:1px solid #555;padding-top:6px';
    canvas.width=200;canvas.height=88;canvas.setAttribute('aria-label',`Sketch ${item.id}`);
    canvas.style.cssText='display:block;width:200px;height:88px;background:#fff;border-radius:3px';
    label.className='sketch-submission-note';label.textContent=note;label.style.cssText='white-space:pre-wrap;overflow-wrap:anywhere;margin-top:4px';
    const g=canvas.getContext?.('2d');if(!g)throw Error('Sketch preview image unavailable.');
    let left=Infinity,top=Infinity,right=-Infinity,bottom=-Infinity;
    for(const path of paths)for(const point of path.pts){const r=path.width/2;left=Math.min(left,point[0]-r);top=Math.min(top,point[1]-r);right=Math.max(right,point[0]+r);bottom=Math.max(bottom,point[1]+r);}
    const width=Math.max(1,right-left),height=Math.max(1,bottom-top),scale=Math.min(184/width,72/height);
    const offsetX=(200-width*scale)/2,offsetY=(88-height*scale)/2;
    paintSketchPaths(g,paths,{scale,toClient:(x,y)=>[offsetX+(x-left)*scale,offsetY+(y-top)*scale]});
    index.className='sketch-submission-index';node.append(index,canvas,label);return node;
  }
  function publish(value){
    if(disposed)throw Error('Submission preview context retired.');
    if(typeof value?.host!=='string'||!value.host||!uint(value.generation)||!uint(value.revision)||!uint(value.operation))throw Error('Invalid submission preview identity.');
    if(last){
      if(BigInt(value.generation)<BigInt(last.generation))return copyAck();
      if(value.generation===last.generation){
        if(value.host!==last.host)throw Error('Conflicting submission preview host.');
        if(BigInt(value.revision)<BigInt(last.revision)||BigInt(value.operation)<BigInt(last.operation))return copyAck();
      }
    }
    if(!uint(value.stamp?.board)||!uint(value.stamp?.delivered)||!Array.isArray(value.items)||value.items.length>8||!Number.isSafeInteger(value.remaining)||value.remaining<0||value.remaining>256)throw Error('Invalid submission preview selection.');
    const noteRemaining=value.note_remaining===undefined?0:value.note_remaining;
    if(!Array.isArray(value.notes)||!Number.isSafeInteger(noteRemaining)||noteRemaining<0)throw Error('Invalid annotation preview selection.');
    const noteIDs=new Set();
    for(const note of value.notes){
      if(typeof note?.id!=='string'||!note.id||!uint(note.revision)||noteIDs.has(note.id))throw Error('Invalid annotation preview revision.');
      noteIDs.add(note.id);
    }
    const seen=new Set();
    for(const item of value.items){
      if(typeof item?.recording_id!=='string'||!item.recording_id||typeof item.id!=='string'||!item.id||!Number.isSafeInteger(item.next_sequence)||item.next_sequence<1||typeof item.digest!=='string'||!(/^[0-9a-f]{64}$/i.test(item.digest))||seen.has(item.id))throw Error('Invalid submission preview version.');
      seen.add(item.id);
    }
    const ack={notes:value.notes.map(note=>({id:note.id,revision:note.revision})),page:value.page===undefined?null:structuredClone(value.page),host:value.host,generation:value.generation,revision:value.revision,operation:value.operation,stamp:{board:value.stamp.board,delivered:value.stamp.delivered},items:value.items.map(item=>({recording_id:item.recording_id,id:item.id,next_sequence:item.next_sequence,digest:item.digest})),remaining:value.remaining,note_remaining:noteRemaining,cursor:structuredClone(value.cursor),accepted:true};
    const nextFingerprint=JSON.stringify(ack);
    if(last&&value.generation===last.generation&&(value.revision===last.revision||value.operation===last.operation)){
      if(nextFingerprint!==fingerprint)throw Error('Conflicting submission preview publication.');
      return copyAck();
    }
    const cursor=receiver.cursor();
    if(typeof value.cursor?.epoch!=='string'||cursor.epoch!==value.cursor.epoch||cursor.seq!==value.cursor.seq)throw Error('Sketch preview receiver has not caught up to the required cursor.');
    const paths=receiver.paths(),chosen=[];
    for(const [position,item] of value.items.entries()){
      const meta=receiver.metadata(item.id),ink=paths.filter(path=>path.sketch===item.id);
      if(!meta||!['complete','interrupted'].includes(meta.phase)||!meta.label?.trim()||!ink.length)throw Error('Selected sketch preview is unavailable.');
      const key=JSON.stringify([item.recording_id,item.id,item.next_sequence,item.digest]);
      const node=cards.get(key)||card(key,item,ink,meta.label),index=String(position+1),badge=node.querySelector('.sketch-submission-index');
      if(node.dataset.index!==index)node.dataset.index=index;
      if(badge.textContent!==`Sketch ${index}`)badge.textContent=`Sketch ${index}`;
      chosen.push({key,node});
    }
    const nextCards=new Map(chosen.map(({key,node})=>[key,node]));
    for(const [key,node] of cards)if(!nextCards.has(key))node.remove();
    // Do not move stable keyed nodes on an identical metadata refresh.
    if(chosen.some(({node},i)=>list.children[i]!==node))for(const {node} of chosen)list.append(node);
    const count=`Pending feedback: ${chosen.length} sketches, ${value.notes.length} annotation${value.notes.length===1?'':'s'}${value.remaining?`, ${value.remaining} more sketches`:''}${noteRemaining?`, ${noteRemaining} more annotations`:''}`;
    if(heading.textContent!==count)heading.textContent=count;
    noteCount=value.notes.length;panel.hidden=chosen.length===0&&noteCount===0&&value.remaining===0&&noteRemaining===0;cards=nextCards;last=ack;fingerprint=nextFingerprint;return copyAck();
  }
  return {publish,captureBusy:()=>!disposed&&!panel.hidden&&(cards.size>0||noteCount>0),dispose(){if(disposed)return;disposed=true;cards.clear();last=null;panel.remove();}};
}
