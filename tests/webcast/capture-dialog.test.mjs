import {test} from 'node:test';
import assert from 'node:assert/strict';
import {noteDialog} from '../../cmd/internal/broadcast/web/collab.js';
import {Element} from './dom-fixture.mjs';
function fixture(){const els=Object.fromEntries(['root','text','input','count','msg','save','cancel','resolve','retry'].map(k=>[k,new Element(k)]));const discarded=[];let release;const d=noteDialog(els,{color:()=>0,post:()=>Promise.resolve({status:204}),saveDraft:()=>new Promise(r=>release=r),onDiscard:p=>discarded.push(p),onRetry:()=>{}});return{els,d,discarded,release:r=>release(r),type(text){els.input.value=text;els.input.listeners.input[0]();},click(k){els[k].listeners.click[0]({});}};}
test('capture projection preserves editable text, gates Save pending only, and fences stale requests',()=>{
 const f=fixture();f.d.draft({request:'r',region:[1,2,3,4],capturePending:true});f.type('authored during capture');assert.equal(f.els.save.disabled,true);assert.ok(!f.els.input.readOnly);
 assert.equal(f.d.projectDraft('old',{payload:{draft:'bad'},pending:false}),false);
 f.d.projectDraft('r',{payload:{draft:'signed'},pending:false,blocked:false,message:'✓ Captured',capturedAt:1});assert.equal(f.els.input.value,'authored during capture');assert.equal(f.els.save.disabled,false);assert.equal(f.els.msg.textContent,'✓ Captured');
 f.d.projectDraft('r',{pending:false,blocked:false,message:'Capture unavailable: queue full',unavailable:'queue full'});assert.equal(f.els.save.disabled,false);
});
test('capture save is one bounded operation; failure releases editability; success does not discard',async()=>{
 const f=fixture();f.d.draft({request:'r',draft:'signed'});f.type('body');f.click('save');assert.equal(f.els.input.readOnly,true);f.type('body');assert.equal(f.els.save.disabled,true);
 f.release({status:409,text:'Store failed'});await new Promise(r=>setTimeout(r,0));assert.equal(f.els.input.readOnly,false);assert.equal(f.els.input.value,'body');assert.equal(f.els.msg.textContent,'Store failed');
 f.click('save');f.release({status:204});await new Promise(r=>setTimeout(r,0));assert.equal(f.d.open,false);assert.equal(f.discarded.length,0);
 f.d.draft({request:'next'});f.click('cancel');assert.equal(f.discarded.length,1);
});
test('shared browser mount starts capture at mouseup, retains typing, and new gesture cancels before its mouseup',async()=>{
 const {mountAnnotation}=await import('../../cmd/internal/webcast/web/annotate.js');
 const root=new Element('body'),box=new Element('box');root.append(box);const doc={body:root,createElement:t=>new Element(t)};
 let serial=0,release;const calls=[];
 const win={innerWidth:100,crypto:{getRandomValues:b=>b.fill(++serial)},performance:{now:()=>0},requestAnimationFrame(){},addEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
 const v={model:{seg:1,docSeg:1,notes:[]},onRecord(){},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y],nodeAt:()=>7,boxOf:()=>[0,0,100,100]};
 const ui=mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#ff0000'],who:{name:()=> 'Named',viewer:()=>({id:'v',name:'Named'})},colors:{get:()=>0},post:async()=>({status:204}),captureCall:(op,body)=>{calls.push([op,body]);return op==='prepare-note'?new Promise(r=>release=r):Promise.resolve({state:'cancelled'});}});
 ui.setMode('note');const layer=root.querySelector('.annotate'),pointer=kind=>layer.listeners[kind][0]({target:layer,pointerId:1,pointerType:'mouse',clientX:40,clientY:40,preventDefault(){}});
 pointer('pointerdown');await pointer('pointerup');assert.equal(ui.dialog.open,true);assert.equal(root.querySelector('.note-msg').textContent,'Capturing…');
 const input=root.querySelector('.note-input');input.value='typing before capture';input.listeners.input[0]();assert.ok(!input.readOnly);
 const save=root.querySelector('.note-buttons').children.at(-1);assert.equal(save.disabled,true);
 release({state:'ready',draft:'signed',captured_at:123});for(let i=0;i<12;i++)await Promise.resolve();assert.equal(input.value,'typing before capture');assert.equal(save.disabled,false);
 pointer('pointerdown');for(let i=0;i<8;i++)await Promise.resolve();assert.ok(calls.some(([op])=>op==='cancel-draft'));assert.equal(ui.dialog.open,false);
});
