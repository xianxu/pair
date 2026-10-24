import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {createFeedbackMask} from '../../cmd/internal/webcast/web/feedback-mask.js';
import {mountAnnotation} from '../../cmd/internal/webcast/web/annotate.js';
import {mountOperator} from '../../cmd/internal/webcast/operator.js';
test('live note mask redraws pins, list and open dialog without modifying note state',()=>{
 class E extends Element{append(...nodes){for(const n of nodes){n.parent=this;super.append(n);}}remove(){this.parent.children=this.parent.children.filter(n=>n!==this);}getContext(){return {setTransform(){},clearRect(){},beginPath(){},stroke(){},moveTo(){},lineTo(){},arc(){},fill(){}};}}
 const root=new E('root'),box=new E('box');root.append(box);const frames=[],hooks=new Set();
 const doc={body:root,createElement:t=>new E(t)};
 const win={innerWidth:800,innerHeight:600,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:fn=>frames.push(fn),addEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
 const note={id:'n',text:'keep me in replay',author:{name:'Viewer',color:'#112233'},region:[0,0,30,30]};
 const entry={note,seg:1,state:'open',revision:0};const v={model:{seg:1,docSeg:1,notes:[entry]},onRecord(){},onFeedback(fn){hooks.add(fn);},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y],placementOf:()=>[0,0,30,30],boxOf:()=>null};
 v.feedbackMask=createFeedbackMask({changed:()=>{for(const fn of hooks)fn();}});
 const ui=mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#112233'],annotate:false,who:{id:()=> 'viewer',name:()=> 'Viewer',viewer:()=>({id:'viewer',name:'Viewer'})},colors:{get:()=>0,set(){}},post:()=>assert.fail('display cannot post')});
 const flush=()=>{while(frames.length)frames.shift()();};flush();
 const panel=root.querySelector('.feedback-panel');
 const list=root.querySelector('.feedback-rows');assert.ok(list,'rows live in the shared scrolling panel body');
 assert.equal(list.children.length,1);
 const click={preventDefault(){},stopPropagation(){},detail:0};
 panel.querySelector('.floating-panel-minimize').listeners.click[0](click);
 assert.equal(panel.classList.contains('minimized'),true);
 assert.equal(root.querySelector('.pins').children.length,1,'minimizing preserves anchored feedback');
 assert.equal(entry.revision,0);assert.equal(entry.state,'open');
 list.children[0].querySelector('.feedback-open').listeners.click[0]({stopPropagation(){}});assert.equal(ui.dialog.open,true);
 assert.equal(panel.classList.contains('minimized'),false,'opening a note makes the adopted editor reachable');
 v.feedbackMask.install({scope:'cast',revision:'1',sketches:[],notes:[{id:'n',revision:'0'}]});flush();
 assert.equal(list.children.length,0);assert.equal(root.querySelector('.pins').children.length,0);assert.equal(ui.dialog.open,false);
 assert.equal(entry.state,'open');assert.equal(entry.note,note);
 entry.revision=1;for(const fn of hooks)fn();flush();assert.equal(list.children.length,1,'newer note revision reappears');
 assert.equal(panel.classList.contains('minimized'),false,'updates preserve the expanded mode selected by opening a note');
 assert.equal(panel.hidden,false);
 assert.equal(panel.querySelector('.floating-panel-restore').textContent,'Notes (1) · Status');
 assert.ok(root.querySelector('.feedback-status').textContent,'pending editor status remains discoverable');
 v.feedbackMask.install({scope:'cast',revision:'2',sketches:[],notes:[{id:'n',revision:'2'}]});
 for(const revision of [0,1,2]){entry.revision=revision;for(const fn of hooks)fn();flush();assert.equal(list.children.length,0,'old reconnect prefix must not flash a known submitted note');}
 entry.revision=3;for(const fn of hooks)fn();flush();assert.equal(list.children.length,1,'current newer note remains visible');
});
test('operator snapshot installs exact sketch mask but trusts filtered note projection',()=>{
 const node=()=>({style:{},appendChild(){}}),win={addEventListener(){},setTimeout:()=>1,clearTimeout(){}};let view,changes=0;
 mountOperator({host:node(),root:node(),report(){}},{annotate:{mountAnnotation(_d,_w,v){view=v;v.onFeedback(()=>changes++);return {strokes:{clear(){}},setAnnotationAllowed(){},setMode(){}};}},palette:[],css:''},{createElement:node},win);
 const mask={scope:'cast',revision:'1',sketches:[{id:'s',next_sequence:'5',digest:'a'.repeat(64)}],notes:[{id:'n',revision:'0'}]};
 win.__couchcastOperatorView({revision:1,doc:1,seg:1,t:1,notes:[{id:'n',text:'newer server-approved note'}],strokes:[],feedback_mask:mask});
 assert.equal(view.feedbackMask.sketchVisible('s',{next_sequence:5,digest:'a'.repeat(64)}),false);assert.equal(changes,1);
 assert.equal(view.model.notes.length,1,'console cannot apply fabricated note revision0');
});

test('operator keeps its whole confirmed projection when a mask is missing or invalid',()=>{
 const node=()=>({style:{},appendChild(){}}),win={addEventListener(){},setTimeout:()=>1,clearTimeout(){}};let view,changes=0;
 mountOperator({host:node(),root:node(),report(){}},{annotate:{mountAnnotation(_d,_w,v){view=v;v.onFeedback(()=>changes++);return {strokes:{clear(){}},setAnnotationAllowed(){},setMode(){}};}},palette:[],css:''},{createElement:node},win);
 const mask={scope:'cast',revision:'1',sketches:[{id:'s',next_sequence:'5',digest:'a'.repeat(64)}],notes:[]};
 const initial={revision:1,doc:1,seg:1,t:1,notes:[],strokes:[],feedback_mask:mask};win.__couchcastOperatorView(initial);
 const model=view.model;
 for(const feedback_mask of [undefined,null,{scope:'',revision:'0',sketches:null,notes:null}]){
  assert.throws(()=>win.__couchcastOperatorView({...initial,revision:2,notes:[{id:'old',text:'unproven revision'}],feedback_mask}));
  assert.equal(view.model,model);assert.equal(changes,1);assert.equal(view.feedbackMask.sketchVisible('s',{next_sequence:5,digest:'a'.repeat(64)}),false);
 }
 win.__couchcastOperatorView({...initial,revision:2,feedback_mask:{...mask,revision:'2',sketches:[]}});
 assert.notEqual(view.model,model,'refused view must not consume the revision of a valid retry');
 assert.equal(view.feedbackMask.sketchVisible('s',{next_sequence:5,digest:'a'.repeat(64)}),true);
});

test('operator rejects a stale mask together with its unproven note projection',()=>{
 const node=()=>({style:{},appendChild(){}}),win={addEventListener(){},setTimeout:()=>1,clearTimeout(){}};let view;
 mountOperator({host:node(),root:node(),report(){}},{annotate:{mountAnnotation(_d,_w,v){view=v;return {strokes:{clear(){}},setAnnotationAllowed(){},setMode(){}};}},palette:[],css:''},{createElement:node},win);
 const mask={scope:'cast',revision:'10',sketches:[],notes:[{id:'n',revision:'2'}]},frame={revision:1,doc:1,seg:1,t:1,notes:[],strokes:[],feedback_mask:mask};
 win.__couchcastOperatorView(frame);const model=view.model;
 assert.throws(()=>win.__couchcastOperatorView({...frame,revision:2,notes:[{id:'n',text:'old submitted note'}],feedback_mask:{...mask,revision:'9',notes:[]}}));
 assert.equal(view.model,model);
 win.__couchcastOperatorView({...frame,revision:2,notes:[{id:'n',text:'server-approved newer revision'}]});
 assert.equal(view.model.notes[0].note.text,'server-approved newer revision');
});
