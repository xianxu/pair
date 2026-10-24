import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {mountAnnotation} from '../../cmd/internal/webcast/web/annotate.js';

function mount(scaleX,scaleY) {
 const frames=[];
 class E extends Element {
  getBoundingClientRect(){return this.tagName==='box'?{left:20,top:30,right:420,bottom:330,width:400,height:300}:super.getBoundingClientRect();}
  getContext(){return this.context ||= {rectangles:[],strokes:[],setTransform(){},clearRect(){},beginPath(){},stroke(){},moveTo(){},lineTo(){},arc(){},fill(){},save(){},restore(){},clip(){},rect(...box){this.rectangles.push(box);},strokeRect(...box){this.strokes.push(box);}};}
 }
 const root=new E('body'),box=new E('box');root.append(box);
 const doc={body:root,createElement:tag=>new E(tag)};
 const win={innerWidth:800,innerHeight:600,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:fn=>frames.push(fn),addEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
 const note={id:'scaled',text:'An anchored note',author:{name:'Viewer',color:'#112233'},region:[40,60,80,40]};
 const original=structuredClone(note),queries=[];
 const view={model:{seg:1,docSeg:1,notes:[{note,seg:1,state:'open',revision:0}]},onRecord(){},toPage:(x,y)=>[(x-20)/scaleX,(y-30)/scaleY],fromPage:(x,y)=>[20+x*scaleX,30+y*scaleY],
  placementOf(_note,options){queries.push(options);return options?.withClip?{box:[40,60,80,40],clip:[50,70,40,20]}:[40,60,80,40];},boxOf:()=>null};
 const ui=mountAnnotation(doc,win,view,{root,box,stage:box,palette:['#112233'],annotate:false,who:{id:()=> 'viewer',name:()=> 'Viewer',viewer:()=>({id:'viewer',name:'Viewer'})},colors:{get:()=>0,set(){}},post:()=>assert.fail('hover must not post')});
 while(frames.length)frames.shift()();
 return {root,box,ui,note,original,queries};
}
for(const [sx,sy,expectedBox,expectedClip] of [[.5,.75,[20,45,40,30],[25,52.5,20,15]],[2,1.5,[80,90,160,60],[100,105,80,30]]]) {
 test(`annotation row hover transforms both geometry corners at scale ${sx}/${sy}`,()=>{
  const f=mount(sx,sy),row=f.root.querySelector('.feedback-rows').children[0];
  row.listeners.mouseenter[0]();
  const context=f.root.querySelector('.feedback-glow').context;
  assert.deepEqual(context.strokes,[expectedBox]);assert.deepEqual(context.rectangles,[expectedClip]);
  assert.deepEqual(f.queries.at(-1),{originalOnly:true,withClip:true});assert.deepEqual(f.note,f.original);
 });
}
test('opening an annotation from a minimized panel expands its adopted editor',()=>{
 const f=mount(1,1),panel=f.root.querySelector('.feedback-panel'),body=panel.querySelector('.feedback-body');
 const event={preventDefault(){},stopPropagation(){},detail:0};panel.querySelector('.floating-panel-minimize').listeners.click[0](event);
 assert.equal(body.hidden,true);f.root.querySelector('.feedback-open').listeners.click[0](event);
 assert.equal(f.ui.dialog.open,true);assert.equal(panel.classList.contains('minimized'),false);assert.equal(body.hidden,false);
});
