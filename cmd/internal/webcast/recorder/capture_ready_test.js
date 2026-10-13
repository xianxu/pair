const assert = require('assert');
const fs = require('fs');
const inspect = eval('(' + fs.readFileSync('capture_ready.js','utf8') + ')');
const box=[20,120,40,30];
global.scrollX=10;global.scrollY=100;global.innerWidth=800;global.innerHeight=600;global.devicePixelRatio=2;
let animations=[];
global.document={getAnimations:()=>animations};
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
const shadowTarget=target();shadowTarget.getRootNode=()=>({mode:'open',host});
animations=[animation('running',shadowTarget)];assert.equal(inspect(box).pending,true);assert.equal(inspect(box).reason,'');
shadowTarget.getRootNode=()=>({mode:'closed',host});assert.equal(inspect(box).reason,'inspection unavailable');
// Ordinary ancestor opacity/transform animations enclose the crop.
animations=[animation('running',host)];assert.equal(inspect(box).pending,true);
animations=[animation('running')];animations[0].playbackRate=0;assert.equal(inspect(box).reason,'paused animation');
animations=[animation('running',null)];assert.equal(inspect(box).reason,'inspection unavailable');
animations=[animation()];animations[0].effect.getComputedTiming=()=>{throw Error('expired')};assert.equal(inspect(box).reason,'inspection unavailable');
const cycle=target();cycle.getRootNode=()=>({mode:'open',host:cycle});animations=[animation('running',cycle)];assert.equal(inspect(box).reason,'inspection limit');
animations=[animation()];assert.equal(inspect([1000,120,10,10]).pending,false);
console.log('shadow and failure readiness fixtures passed');
