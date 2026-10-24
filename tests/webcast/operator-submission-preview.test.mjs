import {test} from 'node:test';
import assert from 'node:assert/strict';
import {mountFloatingPanel} from '../../cmd/internal/webcast/web/floating-panel.js';
import {Element} from './dom-fixture.mjs';
import {createOperatorSketchReceiver,createOperatorSubmissionPreview,mountOperatorSketch} from '../../cmd/internal/webcast/operator-sketch.js';

function fixture(n=2){
 const created=[],statuses=[];const doc={activeElement:null};
 class E extends Element{
  append(...nodes){for(const node of nodes){if(node.parentNode)node.remove();node.parentNode=this;this.children.push(node);}}
  appendChild(node){this.append(node);return node;}
  remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(n=>n!==this);this.parentNode=null;}
  focus(){doc.activeElement=this;}
  getContext(){return {setTransform(){},clearRect(){},save(){},restore(){},beginPath(){},moveTo(){},lineTo(){},closePath(){},fill(){},arc(){}};}
 }
 doc.createElement=tag=>{created.push(tag);return new E(tag);};
 const root=new E('root'),editor=new E('textarea');root.append(editor);editor.focus();
 const receiver=createOperatorSketchReceiver(()=>{});
 const sketches=Array.from({length:n},(_,i)=>({begin:{id:`s${i}`,author:{id:'operator',color:'#ff0000'},doc:1,tab:'tab',vp:{w:100,h:100,dpr:1}},phase:'complete',label:`note ${i}`,strokes:[{begin:{id:'stroke',width:4,at:{node:1,box:[0,0,100,100]}},pts:[[10,10,.1],[40,40,.9]],visible:true,sealed:true,truncated:false}]}));
 const bytes=Buffer.from(JSON.stringify({bootstrap:{epoch:'cast',seq:7,display:{sketches}}}));
 receiver.receive({phase:'begin',handle:'h',epoch:'cast',total:bytes.length});receiver.receive({phase:'chunk',handle:'h',epoch:'cast',total:bytes.length,offset:0,base64:bytes.toString('base64')});
 const value={host:'scope',generation:'1',revision:'1',operation:'1',stamp:{board:'5',delivered:'2'},items:sketches.map(s=>({recording_id:'rec',id:s.begin.id,next_sequence:5,digest:'a'.repeat(64)})),notes:[],remaining:0,cursor:{epoch:'cast',seq:7}};
 const preview=createOperatorSubmissionPreview({root,receiver,setStatus:(key,text)=>statuses.push({key,text})},doc);
 return {doc,root,editor,receiver,value,preview,created,statuses,status:()=>statuses.at(-1)?.text};
}
test('preview routes exact counts to the unified header without creating a panel or canvas',()=>{
 const f=fixture(8);f.value.remaining=3;f.value.note_remaining=4;
 const ack=f.preview.publish(f.value);
 assert.equal(ack.accepted,true);assert.equal(ack.revision,'1');assert.equal(ack.operation,'1');
 assert.deepEqual(f.statuses,[{key:'preview',text:'Pending feedback: 8 sketches, 0 annotations, 3 more sketches, 4 more annotations'}]);
 assert.deepEqual(f.created,[]);assert.deepEqual(f.root.children,[f.editor]);assert.equal(f.doc.activeElement,f.editor);
 const statuses=f.statuses.length;assert.deepEqual(f.preview.publish(structuredClone(f.value)),ack);assert.equal(f.statuses.length,statuses);
 f.preview.dispose();assert.equal(f.status(),'');assert.throws(()=>f.preview.publish(f.value),/retired/);
});

test('preview refuses stale cursor and unavailable ink without changing status or ack',()=>{
 const f=fixture();
 assert.throws(()=>f.preview.publish({...f.value,cursor:{epoch:'cast',seq:8}}),/caught up|cursor/);
 assert.throws(()=>f.preview.publish({...f.value,items:[{...f.value.items[0],id:'missing'}]}),/available|missing/);
 assert.equal(f.statuses.length,0);assert.equal(f.preview.captureBusy(),false);
 assert.equal(f.preview.publish(f.value).accepted,true);
});

test('late generations/revisions retain the current exact ack and header counts',()=>{
 const f=fixture();f.preview.publish(f.value);
 const newer={...f.value,revision:'2',operation:'2',items:[f.value.items[1]],remaining:1};
 const ack=f.preview.publish(newer),status=f.status();
 assert.deepEqual(f.preview.publish(f.value),ack);assert.equal(f.status(),status);
 const replacement={...newer,host:'new-scope',generation:'2',revision:'3',operation:'3'};
 f.preview.publish(replacement);
 assert.equal(f.preview.publish({...newer,revision:'4',operation:'4'}).generation,'2');assert.equal(f.status(),status);
 assert.throws(()=>f.preview.publish({...replacement,items:[f.value.items[0]]}),/conflicting/i);
 assert.throws(()=>f.preview.publish({...replacement,host:'other'}),/host/i);
});

test('accepted selected sketches or notes defer capture; deferred counts alone do not',()=>{
 const f=fixture();assert.equal(f.preview.captureBusy(),false);f.preview.publish(f.value);assert.equal(f.preview.captureBusy(),true);
 f.preview.publish(structuredClone(f.value));assert.equal(f.preview.captureBusy(),true);
 f.preview.publish({...f.value,revision:'2',operation:'2',items:[],notes:[{id:'historic',revision:'3'}]});assert.equal(f.preview.captureBusy(),true);
 f.preview.publish({...f.value,revision:'3',operation:'3',items:[],remaining:4,note_remaining:2});
 assert.equal(f.preview.captureBusy(),false);assert.equal(f.status(),'Pending feedback: 0 sketches, 0 annotations, 4 more sketches, 2 more annotations');
 f.preview.publish({...f.value,revision:'4',operation:'4',items:[]});assert.equal(f.status(),'');
 f.preview.dispose();assert.equal(f.preview.captureBusy(),false);
});

test('pending feedback acknowledges exact owned note revisions and both deferred counts',()=>{
 const f=fixture(1);f.value.notes=[{id:'historical',revision:'18446744073709551615'}];f.value.remaining=3;f.value.note_remaining=49;
 const ack=f.preview.publish(f.value);assert.deepEqual(ack.notes,f.value.notes);assert.equal(ack.note_remaining,49);
 assert.equal(f.status(),'Pending feedback: 1 sketches, 1 annotation, 3 more sketches, 49 more annotations');
 assert.throws(()=>f.preview.publish({...f.value,note_remaining:48}),/conflicting/i);
 ack.notes[0].revision='0';ack.note_remaining=0;assert.deepEqual(f.preview.publish(f.value).notes,f.value.notes);assert.equal(f.preview.publish(f.value).note_remaining,49);
});

test('invalid selection metadata cannot become accepted header information',()=>{
 const mutations=[v=>v.note_remaining=-1,v=>v.note_remaining=1.5,v=>v.note_remaining='1',v=>v.note_remaining=Number.MAX_SAFE_INTEGER+1,v=>v.notes=[{id:'x',revision:'-1'}],v=>v.notes=[{id:'x',revision:'1'},{id:'x',revision:'1'}],v=>v.items[0].digest='bad',v=>v.items.push(v.items[0]),v=>v.remaining=-1,v=>v.stamp.board='-1',v=>v.generation='01'];
 for(const mutate of mutations){const f=fixture();mutate(f.value);assert.throws(()=>f.preview.publish(f.value),/Invalid/i);assert.equal(f.statuses.length,0);assert.equal(f.preview.captureBusy(),false);}
 for(const metadata of [undefined,{phase:'open',label:'label'},{phase:'complete',label:'   '}]){
  const f=fixture();const preview=createOperatorSubmissionPreview({receiver:{cursor:f.receiver.cursor,paths:f.receiver.paths,metadata:()=>metadata},setStatus:()=>assert.fail('invalid metadata reported')},f.doc);
  assert.throws(()=>preview.publish(f.value),/unavailable/);
 }
 const f=fixture();const preview=createOperatorSubmissionPreview({receiver:{cursor:f.receiver.cursor,paths:()=>[],metadata:f.receiver.metadata}},f.doc);assert.throws(()=>preview.publish(f.value),/unavailable/);
});

test('stale status retains accepted identity and capture fence without acknowledging freshness',()=>{
 const f=fixture();assert.equal(f.preview.stale('not ready'),false);const ack=f.preview.publish(f.value);
 assert.equal(f.preview.stale('manager unavailable'),true);assert.equal(f.status(),'Pending feedback is out of date: manager unavailable');assert.equal(f.preview.captureBusy(),true);
 assert.deepEqual(f.preview.publish(structuredClone(f.value)),ack);assert.match(f.status(),/out of date/);
 assert.throws(()=>f.preview.stale(' '),/Invalid/);
 f.preview.publish({...f.value,revision:'2',operation:'2',items:[],remaining:2});assert.equal(f.preview.captureBusy(),false);assert.equal(f.preview.stale('retry'),true);
 f.preview.publish({...f.value,revision:'3',operation:'3',items:[]});assert.equal(f.preview.stale('empty'),false);
 f.preview.dispose();assert.equal(f.preview.stale('retired'),false);assert.equal(f.preview.captureBusy(),false);
});

test('header-only publication groups accepted paths once without inspecting vertices',()=>{
 const f=fixture(),paths=f.receiver.paths();let reads=0,groupReads=0;
 const guarded=paths.map(path=>({...path,get sketch(){groupReads++;return path.sketch;},get pts(){assert.fail('preview must not inspect vertices');}}));
 const receiver={cursor:f.receiver.cursor,metadata:f.receiver.metadata,paths:()=>{reads++;return guarded;}};
 const preview=createOperatorSubmissionPreview({receiver,setStatus(){}},f.doc);
 assert.equal(preview.publish(f.value).accepted,true);assert.equal(reads,1);assert.equal(groupReads,paths.length);assert.deepEqual(f.created,[]);
});

test('notes panel presentation leaves the exact selection and stamps intact with no extra panel',()=>{
 const f=fixture(),before=f.preview.publish(f.value);
 const panel=mountFloatingPanel(f.doc,{innerWidth:800,innerHeight:600,addEventListener(){},removeEventListener(){}},{parent:f.root});
 panel.setCount(12);const e={preventDefault(){},stopPropagation(){},detail:0};
 panel.root.querySelector('.floating-panel-minimize').listeners.click[0](e);panel.setCount(13);
 const button=panel.root.querySelector('.floating-panel-restore');button.listeners.keydown[0]({...e,key:'ArrowLeft'});button.listeners.click[0](e);
 assert.deepEqual(f.preview.publish(structuredClone(f.value)),before);assert.equal(f.root.querySelector('.sketch-submission-preview'),undefined);assert.ok(!f.created.includes('canvas'));
 assert.throws(()=>f.preview.publish({...f.value,items:[f.value.items[1]]}),/conflicting/i);panel.dispose();
});

test('operator mount routes preview status and sketch editor into its shared Notes panel',()=>{
 const f=fixture(0),adopted=[],sketchRows=[];
 const notes={setStatus:(key,text)=>f.statuses.push({key,text}),adoptEditor:node=>adopted.push(node),setSketches:rows=>sketchRows.push(rows)};
 const win={innerWidth:800,innerHeight:600,requestAnimationFrame:()=>1,cancelAnimationFrame(){},setTimeout:()=>1,clearTimeout(){},addEventListener(){},removeEventListener(){}};
 const view={model:{seg:1,docSeg:1,paused:false,ended:false},onRecord:()=>()=>{}};
 const adapter=mountOperatorSketch({root:f.root,box:f.root,report(){},view,annotation:{notes,setMode(){}},palette:['#ff0000'],css:'',shellURL:'data:text/html,private'},f.doc,win);
 assert.equal(adopted.length,1);assert.equal(adopted[0].className,'sketch-label');
 assert.equal(f.root.querySelector('.floating-notes'),undefined);assert.equal(f.root.querySelector('.sketch-submission-preview'),undefined);
 const bytes=Buffer.from(JSON.stringify({bootstrap:{epoch:'cast',seq:7,display:{sketches:[]}}}));
 adapter.receiver.receive({phase:'begin',handle:'mount',epoch:'cast',total:bytes.length});
 adapter.receiver.receive({phase:'chunk',handle:'mount',epoch:'cast',total:bytes.length,offset:0,base64:bytes.toString('base64')});
 const value={...f.value,notes:[{id:'note',revision:'3'}]};
 assert.equal(win.__couchcastSketchSubmissionPreview(value).accepted,true);
 assert.equal(f.status(),'Pending feedback: 0 sketches, 1 annotation');
 assert.equal(win.__couchcastSketchSubmissionPreviewStale('disconnected'),true);assert.equal(f.status(),'Pending feedback is out of date: disconnected');
 assert.equal(adapter.captureBusy(),true);adapter.dispose();assert.equal(adapter.captureBusy(),false);
 assert.equal(f.statuses.filter(s=>s.key==='preview').at(-1).text,'');
 assert.equal(win.__couchcastSketchSubmissionPreview,undefined);assert.equal(win.__couchcastSketchSubmissionPreviewStale,undefined);
 assert.ok(sketchRows.length>0);
});
