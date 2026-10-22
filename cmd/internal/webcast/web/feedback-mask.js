// Display-only confirmed-delivery mask. Never edits vectors, receipts or notes.
const uint=(value,max)=>typeof value==='string'&&/^(0|[1-9][0-9]*)$/.test(value)&&value.length<=20&&BigInt(value)<=max;
export const FEEDBACK_MASK_BYTES=64*1024*1024;
const U64=18446744073709551615n,SAFE=9007199254740991n;
const decimal=(value)=>Number.isSafeInteger(value)&&value>=0?String(value):value;
const id=value=>typeof value==='string'&&value.length>0&&new TextEncoder().encode(value).length<=128&&!/[\p{Cc}\p{Cf}\p{Cs}]/u.test(value);
const need=(ok,why)=>{if(!ok)throw Error(`feedback mask: ${why}`);};
export function createFeedbackMask({changed=()=>{}}={}){
 let current=null,signature='',stream=null;
 const clear=()=>{const had=!!current;current=null;signature='';if(had)changed();};
 function install(value){
  need(value&&id(value.scope)&&uint(value.revision,U64),'scope or revision');
  need(Array.isArray(value.sketches)&&value.sketches.length<=256&&Array.isArray(value.notes),'item bounds');
  need(new TextEncoder().encode(JSON.stringify(value)).length<=FEEDBACK_MASK_BYTES,'serialized byte budget');
  const sketches=new Map(),notes=new Map();
  for(const item of value.sketches){
   need(item&&id(item.id)&&!sketches.has(item.id)&&uint(item.next_sequence,SAFE)&&item.next_sequence!=='0'&&typeof item.digest==='string'&&/^[0-9a-f]{64}$/.test(item.digest),'sketch version');
   sketches.set(item.id,{next_sequence:item.next_sequence,digest:item.digest});
  }
  for(const item of value.notes){need(item&&id(item.id)&&!notes.has(item.id)&&uint(item.revision,U64),'note version');notes.set(item.id,item.revision);}
  const nextSignature=JSON.stringify([value.scope,value.revision,[...sketches].sort(([a],[b])=>a<b?-1:a>b?1:0),[...notes].sort(([a],[b])=>a<b?-1:a>b?1:0)]);
  if(current?.scope===value.scope){
   if(BigInt(value.revision)<BigInt(current.revision))return false;
   if(value.revision===current.revision){need(signature===nextSignature,'conflicting revision');return false;}
  }
  current={scope:value.scope,revision:value.revision,sketches,notes};signature=nextSignature;changed();return true;
 }
 return {
  install,
  beginStream(){stream={};clear();return stream;},
  receive(value,token){if(!stream||token!==stream||current&&current.scope!==value?.scope)return false;return install(value);},
  retire(){stream=null;clear();},
  sketchVisible(key,metadata){const sent=current?.sketches.get(key);return !(sent&&metadata&&sent.next_sequence===decimal(metadata.next_sequence)&&sent.digest===metadata.digest);},
  noteVisible(key,revision){const sent=current?.notes.get(key);return sent===undefined||sent!==decimal(revision);},
 };
}
