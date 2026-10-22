import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createFeedbackMask,FEEDBACK_MASK_BYTES} from '../../cmd/internal/webcast/web/feedback-mask.js';
const digest='a'.repeat(64);
const dto=(revision='1',scope='cast')=>({scope,revision,sketches:[{id:'s',next_sequence:'5',digest}],notes:[{id:'n',revision:'18446744073709551615'}]});
test('exact versions hide only confirmed matching display facts, with atomic validation',()=>{
 let changes=0;const mask=createFeedbackMask({changed:()=>changes++});
 assert.equal(mask.sketchVisible('s',{next_sequence:5,digest}),true);
 mask.install(dto());assert.equal(changes,1);
 assert.equal(mask.sketchVisible('s',{next_sequence:5,digest}),false);
 assert.equal(mask.sketchVisible('s',{next_sequence:'5',digest}),false);
 for(const meta of [undefined,{}, {next_sequence:6,digest},{next_sequence:5,digest:'b'.repeat(64)}])assert.equal(mask.sketchVisible('s',meta),true);
 assert.equal(mask.noteVisible('n','18446744073709551615'),false);
 assert.equal(mask.noteVisible('n',0),true);assert.equal(mask.noteVisible('n',undefined),true);
 for(const change of [v=>v.sketches.push(v.sketches[0]),v=>v.revision='01',v=>v.revision='18446744073709551616',v=>v.sketches[0].next_sequence='9007199254740992',v=>v.notes[0].revision=null,v=>v.sketches[0].digest='bad',v=>v.notes=Array(49).fill(v.notes[0])]){
  const value=dto('2');change(value);assert.throws(()=>mask.install(value));assert.equal(mask.sketchVisible('s',{next_sequence:5,digest}),false);
 }
 assert.equal(changes,1);
});
test('duplicates, stale revisions, conflicting duplicates and stream replacement are fenced',()=>{
 const mask=createFeedbackMask();const first=mask.beginStream();mask.receive(dto('9'),first);
 assert.equal(mask.receive(dto('8'),first),false);assert.equal(mask.receive(dto('9'),first),false);
 assert.throws(()=>mask.receive({...dto('9'),sketches:[]},first));
 const second=mask.beginStream();assert.equal(mask.sketchVisible('s',{next_sequence:5,digest}),true);
 assert.equal(mask.receive(dto('10','old'),first),false);
 mask.receive(dto('0','new'),second);assert.equal(mask.sketchVisible('s',{next_sequence:5,digest}),false);
 mask.retire();assert.equal(mask.receive(dto('1','new'),second),false);assert.equal(mask.noteVisible('n','18446744073709551615'),true);
});

test('historical notes across pages are not limited by one live page palette',()=>{
 const mask=createFeedbackMask(),value={scope:'cast',revision:'1',sketches:[],notes:Array.from({length:49},(_,i)=>({id:`n${i}`,revision:'0'}))};
 mask.install(value);for(const item of value.notes)assert.equal(mask.noteVisible(item.id,0),false);
});

test('shared byte envelope admits boundary UTF8 and atomically refuses oversized masks',()=>{
 const mask=createFeedbackMask();mask.install(dto());
 const value={...dto('2'),padding:''},base=new TextEncoder().encode(JSON.stringify(value)).length,room=FEEDBACK_MASK_BYTES-base;
 value.padding='é'.repeat(Math.floor(room/2))+(room%2?'x':'');
 assert.equal(new TextEncoder().encode(JSON.stringify(value)).length,FEEDBACK_MASK_BYTES);
 assert.equal(mask.install(value),true);
 value.revision='3';value.padding+='x';value.sketches=[];
 // Removing the one sketch saves bytes: refill those bytes before crossing.
 value.padding+='x'.repeat(FEEDBACK_MASK_BYTES-new TextEncoder().encode(JSON.stringify(value)).length+1);
 assert.throws(()=>mask.install(value),/byte budget/);
 assert.equal(mask.sketchVisible('s',{next_sequence:5,digest}),false);
 assert.equal(mask.install(dto('2')),false,'over-budget input must leave prior revision intact');
});
