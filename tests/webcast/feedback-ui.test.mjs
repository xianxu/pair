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
 const win={innerWidth:100,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:fn=>frames.push(fn),addEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
 const note={id:'n',text:'keep me in replay',author:{name:'Viewer',color:'#112233'},region:[0,0,30,30]};
 const entry={note,seg:1,state:'open',revision:0};const v={model:{seg:1,docSeg:1,notes:[entry]},onRecord(){},onFeedback(fn){hooks.add(fn);},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y],placementOf:()=>[0,0,30,30],boxOf:()=>null};
 v.feedbackMask=createFeedbackMask({changed:()=>{for(const fn of hooks)fn();}});
 const ui=mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#112233'],annotate:false,who:{id:()=> 'viewer',name:()=> 'Viewer',viewer:()=>({id:'viewer',name:'Viewer'})},colors:{get:()=>0,set(){}},post:()=>assert.fail('display cannot post')});
 const flush=()=>{while(frames.length)frames.shift()();};flush();
 const list=root.querySelector('.annotation-notes');assert.equal(list.children.length,1);
 list.children[0].listeners.click[0]({stopPropagation(){}});assert.equal(ui.dialog.open,true);
 v.feedbackMask.install({scope:'cast',revision:'1',sketches:[],notes:[{id:'n',revision:'0'}]});flush();
 assert.equal(list.children.length,0);assert.equal(root.querySelector('.pins').children.length,0);assert.equal(ui.dialog.open,false);
 assert.equal(entry.state,'open');assert.equal(entry.note,note);
 entry.revision=1;for(const fn of hooks)fn();flush();assert.equal(list.children.length,1,'newer note revision reappears');
});
test('operator snapshot installs exact sketch mask but trusts filtered note projection',()=>{
 const node=()=>({style:{},appendChild(){}}),win={addEventListener(){},setTimeout:()=>1,clearTimeout(){}};let view,changes=0;
 mountOperator({host:node(),root:node(),report(){}},{annotate:{mountAnnotation(_d,_w,v){view=v;v.onFeedback(()=>changes++);return {strokes:{clear(){}},setAnnotationAllowed(){},setMode(){}};}},palette:[],css:''},{createElement:node},win);
 const mask={scope:'cast',revision:'1',sketches:[{id:'s',next_sequence:'5',digest:'a'.repeat(64)}],notes:[{id:'n',revision:'0'}]};
 win.__couchcastOperatorView({revision:1,doc:1,seg:1,t:1,notes:[{id:'n',text:'newer server-approved note'}],strokes:[],feedback_mask:mask});
 assert.equal(view.feedbackMask.sketchVisible('s',{next_sequence:5,digest:'a'.repeat(64)}),false);assert.equal(changes,1);
 assert.equal(view.model.notes.length,1,'console cannot apply fabricated note revision0');
});
