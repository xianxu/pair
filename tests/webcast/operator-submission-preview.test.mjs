import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {createOperatorSketchReceiver,createOperatorSubmissionPreview} from '../../cmd/internal/webcast/operator-sketch.js';

function fixture(n=2){
 const doc={activeElement:null};
 class E extends Element{
  append(...nodes){for(const node of nodes){if(node.parentNode)node.remove();node.parentNode=this;this.children.push(node);}}
  remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(n=>n!==this);this.parentNode=null;}
  focus(){doc.activeElement=this;}
  getContext(){this.paint ||= [];const out={save(){},restore(){},beginPath(){},moveTo(){},lineTo(){},closePath(){},fill(){},arc:(x,y,r)=>this.paint.push({x,y,r})};return out;}
 }
 doc.createElement=tag=>new E(tag);
 const root=new E('root'),editor=new E('textarea');root.append(editor);editor.focus();
 const receiver=createOperatorSketchReceiver(()=>{});
 const sketches=Array.from({length:n},(_,i)=>({begin:{id:`s${i}`,author:{id:'operator',color:'#ff0000'},doc:1,tab:'tab',vp:{w:100,h:100,dpr:1}},phase:'complete',label:`note ${i}`,strokes:[{begin:{id:'stroke',width:4,at:{node:1,box:[0,0,100,100]}},pts:[[10,10,.1],[40,40,.9]],visible:true,sealed:true,truncated:false}]}));
 const bytes=Buffer.from(JSON.stringify({bootstrap:{epoch:'cast',seq:7,display:{sketches}}}));
 receiver.receive({phase:'begin',handle:'h',epoch:'cast',total:bytes.length});receiver.receive({phase:'chunk',handle:'h',epoch:'cast',total:bytes.length,offset:0,base64:bytes.toString('base64')});
 const value={host:'scope',generation:'1',revision:'1',operation:'1',stamp:{board:'5',delivered:'2'},items:sketches.map(s=>({recording_id:'rec',id:s.begin.id,next_sequence:5,digest:'a'.repeat(64)})),notes:[],remaining:0,cursor:{epoch:'cast',seq:7}};
 const preview=createOperatorSubmissionPreview({root,receiver},doc);
 return {doc,root,editor,receiver,value,preview};
}
test('private preview shows the selected shapes/count/notes before submission without taking focus',()=>{
 const f=fixture(8);f.value.remaining=3;
 const ack=f.preview.publish(f.value);
 assert.equal(ack.accepted,true);
 assert.equal(ack.revision,'1');assert.equal(ack.operation,'1');
 const panel=f.root.querySelector('.sketch-submission-preview');
 assert.match(panel.querySelector('.sketch-submission-count').textContent,/8 sketches.*3 more/);
 const cards=panel.querySelector('.sketch-submission-cards').children;
 assert.equal(cards.length,8);
 assert.equal(cards[0].querySelector('.sketch-submission-note').textContent,'note 0');
 const canvas=cards[0].children.find(n=>n.tagName==='canvas');
 assert.equal(canvas.paint.length,2);assert.ok(canvas.paint[0].r<canvas.paint[1].r,'pressure painter preserved');
 assert.equal(f.doc.activeElement,f.editor);
 assert.equal(panel.querySelectorAll('button').length,0);
 const first=cards[0];const paint=canvas.paint.length;
 f.preview.publish(structuredClone(f.value));
 assert.equal(panel.querySelector('.sketch-submission-cards').children[0],first);
 assert.equal(canvas.paint.length,paint,'identical projection does not redraw');
 assert.equal(f.doc.activeElement,f.editor);
 f.preview.dispose();assert.equal(f.root.querySelector('.sketch-submission-preview'),undefined);
 assert.throws(()=>f.preview.publish(f.value),/retired/);
});
test('preview refuses missing receiver version and never falsely acknowledges a stale cursor',()=>{
 const f=fixture();
 assert.throws(()=>f.preview.publish({...f.value,cursor:{epoch:'cast',seq:8}}),/caught up|cursor/);
 assert.equal(f.root.querySelector('.sketch-submission-cards').children.length,0);
 assert.throws(()=>f.preview.publish({...f.value,items:[{...f.value.items[0],id:'missing'}]}),/available|missing/);
 assert.equal(f.preview.publish(f.value).accepted,true);
});
test('late generations/revisions cannot replace current cards or ACK another publication',()=>{
 const f=fixture();
 f.preview.publish(f.value);
 const newer={...f.value,revision:'2',operation:'2',items:[f.value.items[1]],remaining:1};
 f.preview.publish(newer);
 const panel=f.root.querySelector('.sketch-submission-cards');const only=panel.children[0];
 const old=f.preview.publish(f.value);
 assert.equal(old.revision,'2');assert.equal(panel.children[0],only);assert.equal(panel.children.length,1);
 const replacement={...newer,host:'new-scope',generation:'2',revision:'3',operation:'3'};
 f.preview.publish(replacement);
 assert.equal(f.preview.publish({...newer,revision:'4',operation:'4'}).generation,'2');
 assert.equal(panel.children[0],only);
 assert.throws(()=>f.preview.publish({...replacement,items:[f.value.items[0]]}),/conflicting/i);
});

test('visible nonempty preview defers background capture until empty or disposed',()=>{
 const f=fixture();
 assert.equal(f.preview.captureBusy(),false);
 f.preview.publish(f.value);
 assert.equal(f.preview.captureBusy(),true);
 const card=f.root.querySelector('.sketch-submission-card');
 f.preview.publish(structuredClone(f.value));
 assert.equal(f.preview.captureBusy(),true);
 assert.equal(f.root.querySelector('.sketch-submission-card'),card);
 f.preview.publish({...f.value,revision:'2',operation:'2',items:[],remaining:0});
 assert.equal(f.preview.captureBusy(),false);
 assert.equal(f.root.querySelector('.sketch-submission-preview').hidden,true);
 f.preview.dispose();assert.equal(f.preview.captureBusy(),false);
});

test('pending feedback shows both counts and acknowledges exact note revisions',()=>{
 const f=fixture();
 f.value.notes=[{id:'old-page-note',revision:'3'}];
 const ack=f.preview.publish(f.value);
 const panel=f.root.querySelector('.sketch-submission-preview');
 assert.match(panel.querySelector('.sketch-submission-count').textContent,/Pending feedback: 2 sketches, 1 annotation/);
 assert.deepEqual(ack.notes,f.value.notes);
 const card=panel.querySelector('.sketch-submission-card');
 assert.equal(card.dataset.index,'1');
 assert.match(card.querySelector('.sketch-submission-index').textContent,/1/);
 const next={...f.value,revision:'2',operation:'2',notes:[{id:'old-page-note',revision:'4'}]};
 assert.deepEqual(f.preview.publish(next).notes,next.notes);
 assert.equal(panel.querySelector('.sketch-submission-card'),card);
 assert.equal(f.doc.activeElement,f.editor);
});

test('note-only feedback is visible and capture-private until its exact list empties',()=>{
 const f=fixture(0);
 f.value.notes=[{id:'historical',revision:'18446744073709551615'}];
 const ack=f.preview.publish(f.value);
 assert.deepEqual(ack.notes,f.value.notes);
 const panel=f.root.querySelector('.sketch-submission-preview');
 assert.equal(panel.hidden,false);
 assert.equal(f.preview.captureBusy(),true);
 assert.match(panel.querySelector('.sketch-submission-count').textContent,/0 sketches, 1 annotation/);
 assert.equal(panel.querySelectorAll('canvas').length,0);
 f.preview.publish({...f.value,revision:'2',operation:'2',notes:[]});
 assert.equal(panel.hidden,true);
 assert.equal(f.preview.captureBusy(),false);
});


test('pending feedback preserves both deferred counts in its exact acknowledgement',()=>{
 const f=fixture(1);
 f.value.remaining=3;f.value.note_remaining=49;
 const ack=f.preview.publish(f.value);
 assert.equal(ack.note_remaining,49);
 const heading=f.root.querySelector('.sketch-submission-count');
 assert.match(heading.textContent,/1 sketches, 0 annotations, 3 more sketches, 49 more annotations/);
 assert.throws(()=>f.preview.publish({...f.value,note_remaining:48}),/conflicting/i);
 ack.note_remaining=0;
 assert.equal(f.preview.publish(f.value).note_remaining,49,'acknowledgements are owned snapshots');
 for(const invalid of [-1,1.5,'1',Number.MAX_SAFE_INTEGER+1]){
   const other=fixture(0);
   assert.throws(()=>other.preview.publish({...other.value,note_remaining:invalid}),/annotation.*selection/i);
 }
});

test('only deferred feedback keeps its count reachable without blocking capture',()=>{
 const f=fixture(0);
 f.value.note_remaining=2;
 assert.equal(f.preview.publish(f.value).note_remaining,2);
 const panel=f.root.querySelector('.sketch-submission-preview');
 assert.equal(panel.hidden,false);
 assert.match(panel.querySelector('.sketch-submission-count').textContent,/2 more annotations/);
 assert.equal(f.preview.captureBusy(),false);
 f.preview.publish({...f.value,revision:'2',operation:'2',note_remaining:0});
 assert.equal(panel.hidden,true);
});
