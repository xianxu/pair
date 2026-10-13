const assert = require('assert');
const fs = require('fs');
const inspect = eval('(' + fs.readFileSync('capture_ready.js','utf8') + ')');
const box=[20,120,40,30];
global.scrollX=10;global.scrollY=100;global.innerWidth=800;global.innerHeight=600;global.devicePixelRatio=2;
let animations=[];
global.getComputedStyle=(node,pseudo)=>({content:node.pseudoContent?.[pseudo] ?? "none"});
global.document={getAnimations:()=>animations,documentElement:{}};
function target(rect={left:10,top:20,width:40,height:30}) {return {nodeType:1,getBoundingClientRect:()=>({...rect,right:rect.left+rect.width,bottom:rect.top+rect.height}),getRootNode:()=>document};}
function animation(state='running',node=target(),timing={iterations:1,duration:1000,endTime:1000}) {
 return {playState:state,playbackRate:1,effect:{target:node,getComputedTiming:()=>timing},finish(){throw Error('mutated finish')},cancel(){throw Error('mutated cancel')},pause(){throw Error('mutated pause')}};
}
let result=inspect(box);assert.deepEqual(result,{viewport:[10,100,800,600],dpr:2,pending:false,reason:''});
animations=[animation()];assert.equal(inspect(box).pending,true);assert.equal(inspect(box).reason,'');
animations[0].playState='finished';assert.equal(inspect(box).pending,false);
animations[0].playState='idle';assert.equal(inspect(box).pending,false);
animations=[animation('paused')];assert.equal(inspect(box).reason,'paused animation');
animations=[animation('running',target(),{iterations:Infinity,duration:1000,endTime:Infinity})];assert.equal(inspect(box).reason,'infinite animation');
animations=[animation('running',target({left:500,top:500,width:20,height:20}))];assert.equal(inspect(box).pending,false);
animations=[animation('running',target(),{iterations:1,delay:800,duration:100,endTime:900})];assert.equal(inspect(box).pending,true);
animations[0].playbackRate=-1;assert.equal(inspect(box).pending,true);
// The sequence represents a transition completing and another starting between polls.
animations=[animation()];assert.equal(inspect(box).pending,true);animations=[];assert.equal(inspect(box).pending,false);animations=[animation()];assert.equal(inspect(box).pending,true);
// Pseudo-element effects are attached to their originating host element.
animations=[animation('running',{element:target(),type:'::before'})];assert.equal(inspect(box).pending,true);
animations=Array.from({length:513},()=>animation('finished'));assert.equal(inspect(box).reason,'inspection limit');
document.getAnimations=()=>{throw Error('not observable')};assert.equal(inspect(box).reason,'inspection unavailable');
assert.deepEqual(inspect(box,false),{viewport:[10,100,800,600],dpr:2,pending:false,reason:''});
console.log('capture readiness fixtures passed');
document.getAnimations=()=>animations;
const host=target({left:0,top:0,width:800,height:600});
const shadowTarget=target();const knownRoot={mode:'open',host,firstChild:shadowTarget,getAnimations:()=>[]};host.shadowRoot=knownRoot;document.documentElement=host;shadowTarget.getRootNode=()=>knownRoot;
animations=[animation('running',shadowTarget)];assert.equal(inspect(box).pending,true);assert.equal(inspect(box).reason,'');
shadowTarget.getRootNode=()=>({mode:'closed',host});assert.equal(inspect(box).reason,'inspection unavailable');
// Ordinary ancestor opacity/transform animations enclose the crop.
animations=[animation('running',host)];assert.equal(inspect(box).pending,true);
animations=[animation('running')];animations[0].playbackRate=0;assert.equal(inspect(box).reason,'paused animation');
animations=[animation('running',null)];assert.equal(inspect(box).reason,'inspection unavailable');
animations=[animation()];animations[0].effect.getComputedTiming=()=>{throw Error('expired')};assert.equal(inspect(box).reason,'inspection unavailable');
const cycle=target();cycle.nextSibling=cycle;document.documentElement=cycle;animations=[];assert.equal(inspect(box).reason,'inspection limit');document.documentElement={};
animations=[animation()];assert.equal(inspect([1000,120,10,10]).pending,false);
console.log('shadow and failure readiness fixtures passed');
// Real Chrome omits shadow-root animations from document.getAnimations().
animations=[];
const localTarget=target(), localHost=target();
const localAnimation=animation('running',localTarget);
const localRoot={mode:'open',host:localHost,firstChild:localTarget,getAnimations:()=>[localAnimation]};
localHost.shadowRoot=localRoot;localTarget.getRootNode=()=>localRoot;document.documentElement=localHost;
assert.equal(inspect(box).pending,true,'open-shadow animation must be enumerated at its root');
assert.equal(inspect(box).reason,'');
localAnimation.playState='finished';assert.equal(inspect(box).pending,false);
animations=Array.from({length:300},()=>animation('finished'));
localRoot.getAnimations=()=>Array.from({length:213},()=>animation('finished'));
assert.equal(inspect(box).reason,'inspection limit','animation cap is aggregate across roots');
animations=[];delete localRoot.getAnimations;assert.equal(inspect(box).reason,'inspection unavailable');
localRoot.getAnimations=()=>{throw Error('shadow inspection denied')};assert.equal(inspect(box).reason,'inspection unavailable');
assert.deepEqual(inspect(box,false),{viewport:[10,100,800,600],dpr:2,pending:false,reason:''});
let rootEnumerations=0;
localRoot.getAnimations=()=>{rootEnumerations++;return []};
const nestedTarget=target(),nestedAnimation=animation('running',nestedTarget);
const nestedRoot={mode:'open',host:localTarget,firstChild:nestedTarget,getAnimations:()=>{rootEnumerations++;return [nestedAnimation]}};
localTarget.shadowRoot=nestedRoot;nestedTarget.getRootNode=()=>nestedRoot;
assert.equal(inspect(box).pending,true);assert.equal(rootEnumerations,2,'each open shadow root enumerated once');
nestedAnimation.playState='finished';assert.equal(inspect(box).pending,false);
console.log('separate shadow-root enumeration fixtures passed');

// BR-1: geometry of an animation target does not bound its painted influence.
document.documentElement={};animations=[];
for (const kind of ['child','shadow','pseudo','pseudo-object','pseudo-effect']) {
 const parent=target({left:500,top:500,width:0,height:0});
 if(kind==='child') parent.firstChild=target();
 if(kind==='shadow') parent.shadowRoot={mode:'open'};
 if(kind==='pseudo') parent.pseudoContent={'::before':'"visible"'};
 const effectTarget=kind==='pseudo-object'?{element:parent,type:'::before'}:parent;
 for(const state of ['running','paused','infinite']) {
  animations=[animation(state==='paused'?'paused':'running',effectTarget,
   {iterations:state==='infinite'?Infinity:1,endTime:state==='infinite'?Infinity:1000})];
  if(kind==='pseudo-effect') animations[0].effect.pseudoElement='::after';
  const observed=inspect(box);
  assert.equal(observed.pending,true,kind+' '+state+' overflow cannot be settled');
  assert.equal(observed.reason,state==='paused'?'paused animation':state==='infinite'?'infinite animation':'');
 }
 animations[0].playState='finished';assert.equal(inspect(box).pending,false);
}
console.log('overflow influence readiness fixtures passed');

// A leaf can also paint into the crop through a shadow outside its border box.
document.documentElement={};
const shadowLeaf=target({left:-100,top:20,width:40,height:30});
const priorStyle=getComputedStyle;
global.getComputedStyle=(node,pseudo)=>node===shadowLeaf&&!pseudo?
 {boxShadow:'rgb(255, 255, 255) 110px 0px 0px 0px',filter:'none'}:priorStyle(node,pseudo);
for(const state of ['running','paused','infinite']) {
 animations=[animation(state==='paused'?'paused':'running',shadowLeaf,
  {iterations:state==='infinite'?Infinity:1,endTime:state==='infinite'?Infinity:1000})];
 assert.equal(inspect(box).pending,true,'leaf shadow '+state+' cannot falsely settle');
}
