// A gesture owns one server capture. UI request IDs correlate operations;
// only the opaque server draft token authorizes Save.
export function captureDraftController({call, project, id, now=()=>Date.now(), setTimeout:later=globalThis.setTimeout, clearTimeout:clear=globalThis.clearTimeout}) {
  let current = null;
  const rpc = async (op, body, deadline, context) => {
    const timeoutMs = Math.min(4000, deadline-now());
    if (timeoutMs <= 0) throw Error('Capture timed out. Retry capture.');
    let timer;
    try {
      return await Promise.race([
        Promise.resolve().then(()=>call(op,body,{timeoutMs,context})),
        new Promise((_,reject)=>{timer=later(()=>reject(Error('Capture request timed out. Retry capture.')),timeoutMs);}),
      ]);
    } finally { clear(timer); }
  };
  const notify = (entry, update) => { if(current===entry)project(entry.request,update); };
  const failure = (entry, error) => {
    entry.state='failed';
    notify(entry,{state:'failed',pending:false,blocked:true,retry:true,message:error?.message || String(error)});
  };
  const pause = deadline => new Promise((resolve,reject)=>{
    const left=deadline-now();
    if(left<=0){reject(Error('Capture timed out. Retry capture.'));return;}
    later(resolve,Math.min(1000,left));
  });
  const statusBody = entry => ({request:entry.request,...(entry.draft?{draft:entry.draft}:{})});
  const settle = async (entry, answer, deadline) => {
    while(current===entry && answer?.state==='pending') {
      if(answer.draft)entry.draft=answer.draft;
      await pause(deadline);
      if(current!==entry)return null;
      answer=await rpc('draft-status',statusBody(entry),deadline,entry.context);
    }
    return current===entry?answer:null;
  };
  const prepare = async entry => {
    try {
      const deadline=now()+36000;
      const answer=await settle(entry,await rpc('prepare-note',{...entry.body,request:entry.request},deadline,entry.context),deadline);
      if(!answer)return;
      if(answer.state!=='ready'||!answer.draft)throw Error(answer.error||answer.unavailable||'Could not capture this note. Retry capture.');
      entry.draft=answer.draft;entry.state='ready';
      const timestamp=typeof answer.captured_at === 'number' ? answer.captured_at : Date.parse(answer.captured_at);
      const capturedAt=timestamp > 0 ? answer.captured_at : undefined;
      notify(entry,{state:'ready',pending:false,blocked:false,retry:false,payload:{draft:answer.draft},capturedAt,
        unavailable:answer.unavailable||'',message:answer.unavailable?`Capture unavailable: ${answer.unavailable}`:capturedAt?'✓ Captured':'Capture ready'});
    } catch(error) { if(current===entry)failure(entry,error); }
  };
  const cancel = () => {
    const old=current;current=null;
    if(old&&old.state!=='saved')void rpc('cancel-draft',statusBody(old),now()+4000,old.context).catch(()=>{});
  };
  const start = (body,request=id()) => {
    cancel();
    const entry={body,request,context:call.context?.(),state:'pending',draft:null,saving:null};current=entry;
    notify(entry,{state:'pending',pending:true,blocked:true,retry:false,message:'Capturing…'});
    void prepare(entry);return request;
  };
  return {
    start,cancel,
    retry() {
      const old=current;if(!old||old.state!=='failed')return;
      const request=id();
      // Rekey the existing editor before starting; authored text is preserved.
      project(old.request,{payload:{request},pending:true,blocked:true,retry:false,message:'Capturing…'});
      start(old.body,request);
    },
    save(body) {
      const entry=current;
      if(!entry||entry.state!=='ready'||!entry.draft)return Promise.resolve({status:409,text:'Capture is not ready. Retry capture.'});
      if(entry.saving)return entry.saving;
      entry.saving=(async()=>{
        try {
          const deadline=now()+36000;
          const authored={viewer:body.viewer,color:body.color,text:body.text};
          const same=()=>JSON.stringify(entry.intent?.authored)===JSON.stringify(authored);
          const confirmed=()=>{
            entry.state='saved';
            if(!same()) {
              const message='The original note was saved. Your edited text is still here; draw a new note to send it.';
              notify(entry,{state:'saved',pending:false,blocked:true,message});
              return {status:409,text:message};
            }
            return {status:204};
          };
          // An uncertain Save keeps its identity and payload. Check whether the
          // server has proved it absent before accepting newly authored text.
          if(entry.intent) {
            const prior=await rpc('draft-status',statusBody(entry),deadline,entry.context);
            if(current!==entry)return {status:409,text:'This draft was cancelled.'};
            if(prior.state==='saved')return confirmed();
            if(prior.state==='ready')entry.intent=null;
            else if(prior.state!=='pending')throw Error(prior.error||'Save was not confirmed. Try again.');
          }
          if(!entry.intent)entry.intent={request:id(),authored};
          const answer=await settle(entry,await rpc('save-note',{request:entry.intent.request,draft:entry.draft,...entry.intent.authored},deadline,entry.context),deadline);
          if(!answer)return {status:409,text:'This draft was cancelled.'};
          if(answer.state==='saved')return confirmed();
          if(answer.state==='ready')entry.intent=null; // acknowledged writer refusal
          throw Error(answer.error||'Save was not confirmed. Try again.');
        } catch(error) {return {status:409,text:error.message||'Save was not confirmed. Try again.'};}
        finally {entry.saving=null;}
      })();
      return entry.saving;
    },
  };
}
