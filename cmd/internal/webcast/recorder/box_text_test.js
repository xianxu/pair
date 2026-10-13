const assert = require('assert');
const fs = require('fs');
const collect = eval('(' + fs.readFileSync('box_text.js', 'utf8') + ')');
let elements;
function text(s, x=0, y=0) { return {nodeType:3,nodeValue:s,x,y,childNodes:[]}; }
function el(tag, attrs={}, kids=[], x=0, y=0, w=150, h=10) {
 const n={clientLeft:0,clientTop:0,clientWidth:w,clientHeight:h,offsetWidth:w,offsetHeight:h,nodeType:1,localName:tag,childNodes:kids,attrs,style:{display:'block',visibility:'visible'},getAttribute:k=>attrs[k]||null,
 getBoundingClientRect:()=>({left:x,top:y,right:x+w,bottom:y+h,width:w,height:h}), getClientRects(){return [this.getBoundingClientRect()]}};
 kids.forEach((k,i)=>{k.parentElement=n;k.nextSibling=kids[i+1]||null}); n.firstChild=kids[0]||null; elements.push(n); return n;
}
function setup(kids) {
 global.scrollX=global.scrollY=0; global.innerWidth=200; global.innerHeight=200;
 global.CSS={escape:s=>s}; global.getComputedStyle=n=>n.style;
 const root=el('body',{},kids); root.parentElement=null;
 global.document={documentElement:root,getElementById:id=>elements.find(n=>n.attrs.id===id),createRange:()=>({
 setStart(n,i){this.n=n;this.a=i},setEnd(n,i){this.b=i},selectNodeContents(n){this.n=n;this.a=0;this.b=n.nodeValue.length},
 getClientRects(){return [{left:this.n.x+this.a,top:this.n.y,right:this.n.x+this.b,bottom:this.n.y+10,width:this.b-this.a,height:10}]}
 })};
}
elements=[];
const later=el('p',{id:'later'},[text('second',0,30)],0,30);
const first=el('button',{id:'first','aria-label':'Save exact',title:'Tip'},[text('first',0,10)],0,10);
const hidden=el('p',{'aria-hidden':'true'},[text('secret',0,20)],0,20);
const off=el('p',{},[text('offscreen',0,300)],0,300);
setup([later,first,hidden,off]);
let out=collect([0,0,100,100]);
assert.deepEqual(out.visible_text.map(x=>x.text),['first','second']);
assert(out.accessible_names.some(x=>x.kind==='aria-label'&&x.text==='Save exact'));
assert(out.accessible_names.some(x=>x.kind==='role'&&x.role==='button'&&x.text==='Save exact'));
assert(out.accessible_names.some(x=>x.kind==='title'&&x.text==='Tip'));
assert(!JSON.stringify(out).includes('secret'));
assert.equal(out.truncated,false);
elements=[]; setup([el('p',{id:'clip'},[text('abcdefgh',0,0)])]);
out=collect([2,0,3,10]); assert.equal(out.visible_text.map(x=>x.text).join(''),'cde');
elements=[]; const many=[]; for(let i=0;i<300;i++)many.push(el('p',{},[text('x',0,i%100)],0,i%100)); setup(many);
out=collect([0,0,200,200]); assert(out.truncated); assert(out.visible_text.length+out.accessible_names.length<=256);
elements=[]; setup([el('p',{},[text('a'.repeat(30000),0,0)])]); out=collect([0,0,200,200]); assert(out.truncated);
console.log('box text fixtures passed');
elements=[];
const nested=el('button',{id:'nested'},[el('span',{},[text('Nested exact')])]);
const label=el('label',{id:'label'},[text('Referenced name',0,20)],0,20);
const labelled=el('input',{'aria-label':'Referenced name',type:'checkbox',id:'field'},[],0,40);
const gone=el('p',{},[text('display secret')]);gone.style.display='none';
const invisible=el('p',{},[text('visibility secret')]);invisible.style.visibility='hidden';
const zero=el('p',{},[text('zero secret')],0,0,0,0);
setup([nested,label,labelled,gone,invisible,zero]);out=collect([0,0,200,200]);
assert(out.accessible_names.some(x=>x.role==='button'&&x.text==='Nested exact'));
assert(out.accessible_names.some(x=>x.role==='checkbox'&&x.text==='Referenced name'));
assert(!JSON.stringify(out).includes('secret'));
elements=[];const host=el('x-host',{},[]);host.shadowRoot={nodeType:11,firstChild:el('p',{id:'shadow'},[text('Shadow exact')])};
setup([host,el('iframe',{},[]),el('object',{},[])]);out=collect([0,0,200,200]);
assert(out.visible_text.some(x=>x.text==='Shadow exact'));assert(out.truncated);assert(out.skipped.includes('cross-origin frame'));assert(out.skipped.includes('object/embed'));
elements=[];const dense=[];for(let i=0;i<21000;i++)dense.push(el('p',{},[]));setup(dense);out=collect([0,0,200,200]);assert(out.truncated);
elements=[];setup([el('p',{'aria-label':'é'.repeat(8192)},[])]);out=collect([0,0,200,200]);assert(out.truncated);
assert(Buffer.byteLength(out.accessible_names.map(x=>Object.values(x).join('')).join(''))<=8192);

elements=[];const slotted=el('span',{id:'slotted'},[text('Slot exact')]);const slotHost=el('x-host',{id:'host'},[slotted]);slotHost.shadowRoot={nodeType:11,firstChild:el('slot',{},[])};slotHost.shadowRoot.firstChild.assignedNodes=()=>[slotted];setup([slotHost]);out=collect([0,0,200,200]);assert(out.visible_text.some(x=>x.text==='Slot exact'));

elements=[];const zeroParent=el('div',{},[el('p',{},[text('Visible positioned child')])],0,0,0,0);const hiddenParent=el('div',{},[el('span',{},[text('Visible override')])]);hiddenParent.style.visibility='hidden';setup([zeroParent,hiddenParent]);out=collect([0,0,200,200]);assert(out.visible_text.some(x=>x.text==='Visible positioned child'));assert(out.visible_text.some(x=>x.text==='Visible override'));
elements=[];const plainHost=el('x-host',{id:'host'},[]);plainHost.shadowRoot={nodeType:11,firstChild:el('span',{},[text('Anonymous shadow')])};setup([plainHost]);out=collect([0,0,200,200]);
assert.equal(out.visible_text.find(x=>x.text==='Anonymous shadow').selector,'#host >>> span:nth-child(1)');
// Regression BR-4: raw range/element rects do not prove visibility.
elements=[];
const clippedChild=el('button',{'aria-label':'Clipped name'},[text('Clipped text',30,0)],30,0,50,10);
const clipParent=el('div',{},[clippedChild],0,0,10,10);clipParent.style.overflowX=clipParent.style.overflowY='hidden';
setup([clipParent]);out=collect([0,0,100,100]);
assert.deepEqual(out.visible_text,[],'overflow hidden text is invisible');assert.deepEqual(out.accessible_names,[],'overflow hidden names are invisible');
elements=[];setup([el('div',{'aria-label':'Wide ancestor'},[],0,0,1000,1000)]);out=collect([300,300,10,10]);
assert.deepEqual(out.visible_text,[]);assert.deepEqual(out.accessible_names,[],'empty note/viewport intersection is not a rectangle');
elements=[];
const clippedLine=el('span',{'aria-label':'Line'},[text('abcdefghij',0,0)],0,0,10,10);
const innerClip=el('div',{},[clippedLine],2,-5,3,20);innerClip.style.overflowX='scroll';innerClip.style.overflowY='visible';
const outerClip=el('div',{},[innerClip],0,0,20,5);outerClip.style.overflowX='visible';outerClip.style.overflowY='auto';
setup([outerClip]);out=collect([0,0,100,100]);assert.equal(out.visible_text.map(x=>x.text).join(''),'cde');assert(out.accessible_names.some(x=>x.text==='Line'));
elements=[];const transparent=el('div',{},[el('button',{'aria-label':'Transparent'},[text('Transparent')])]);transparent.style.opacity='0';setup([transparent]);out=collect([0,0,100,100]);assert.equal(out.visible_text.length+out.accessible_names.length,0);
// Client scrollports exclude borders/scrollbars and inherit transform scaling.
elements=[];
const borderLine=el('span',{},[text('abcdefghijklmnop',0,0)],0,0,30,10);
const scaledClip=el('div',{},[borderLine],0,0,20,20);scaledClip.offsetWidth=10;scaledClip.offsetHeight=10;scaledClip.clientLeft=2;scaledClip.clientTop=0;scaledClip.clientWidth=3;scaledClip.clientHeight=10;scaledClip.style.overflowX='hidden';scaledClip.style.transform='matrix(2, 0, 0, 2, 0, 0)';
setup([scaledClip]);out=collect([0,0,100,100]);assert.equal(out.visible_text.map(x=>x.text).join(''),'efghij');
elements=[];const painted=el('div',{},[el('span',{'aria-label':'Paint clipped'},[text('Paint clipped',30,0)],30,0)],0,0,10,10);painted.style.contain='paint';setup([painted]);out=collect([0,0,100,100]);assert.equal(out.visible_text.length+out.accessible_names.length,0);
elements=[];const slotNode=el('slot',{},[]);const slottedClipped=el('button',{'aria-label':'Slot clipped'},[text('Slot clipped',30,0)],30,0);slotNode.assignedNodes=()=>[slottedClipped];
const shadowClip=el('div',{},[slotNode],0,0,10,10);shadowClip.style.overflowX='hidden';
const clippedHost=el('x-host',{},[slottedClipped]);clippedHost.shadowRoot={nodeType:11,firstChild:shadowClip};setup([clippedHost]);out=collect([0,0,100,100]);assert.equal(out.visible_text.length+out.accessible_names.length,0,'slot uses composed shadow clip');
for(const styles of [{clipPath:'circle(5px)'},{maskImage:'url(mask.png)'},{clip:'rect(0px, 2px, 2px, 0px)'},{transform:'matrix(0, 1, -1, 0, 0, 0)'},{overflowX:'hidden',borderTopLeftRadius:'10px'}]) {
 elements=[];const shaped=el('div',{'aria-label':'Unsupported'},[text('Unsupported')]);Object.assign(shaped.style,styles);setup([shaped]);out=collect([0,0,100,100]);assert.equal(out.visible_text.length+out.accessible_names.length,0);assert(out.truncated);assert(out.skipped.includes('unsupported clipping'));
}
// A fixed descendant escapes an ordinary overflow ancestor; no second traversal.
elements=[];const fixed=el('button',{'aria-label':'Fixed visible'},[text('Fixed visible',30,0)],30,0);fixed.style.position='fixed';const flowClip=el('div',{},[fixed],0,0,10,10);flowClip.style.overflowX='hidden';setup([flowClip]);out=collect([0,0,100,100]);assert(out.visible_text.some(x=>x.text==='Fixed visible'));
console.log('effective clip fixtures passed');
// Refuse before assignedNodes materializes a host list beyond the visit budget.
elements=[];const hugeSlot=el('slot',{},[]);hugeSlot.getRootNode=()=>({host:{childNodes:{length:21000}}});hugeSlot.assignedNodes=()=>{throw new Error('unbounded slot enumeration')};setup([hugeSlot]);out=collect([0,0,100,100]);assert(out.truncated);
