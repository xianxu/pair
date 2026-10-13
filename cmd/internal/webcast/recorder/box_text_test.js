const assert = require('assert');
const fs = require('fs');
const collect = eval('(' + fs.readFileSync('box_text.js', 'utf8') + ')');
let elements;
function text(s, x=0, y=0) { return {nodeType:3,nodeValue:s,x,y,childNodes:[]}; }
function el(tag, attrs={}, kids=[], x=0, y=0, w=150, h=10) {
 const n={nodeType:1,localName:tag,childNodes:kids,attrs,style:{display:'block',visibility:'visible'},getAttribute:k=>attrs[k]||null,
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

elements=[];const slotted=el('span',{id:'slotted'},[text('Slot exact')]);const slotHost=el('x-host',{id:'host'},[slotted]);slotHost.shadowRoot={nodeType:11,firstChild:el('slot',{},[])};setup([slotHost]);out=collect([0,0,200,200]);assert(out.visible_text.some(x=>x.text==='Slot exact'));

elements=[];const zeroParent=el('div',{},[el('p',{},[text('Visible positioned child')])],0,0,0,0);const hiddenParent=el('div',{},[el('span',{},[text('Visible override')])]);hiddenParent.style.visibility='hidden';setup([zeroParent,hiddenParent]);out=collect([0,0,200,200]);assert(out.visible_text.some(x=>x.text==='Visible positioned child'));assert(out.visible_text.some(x=>x.text==='Visible override'));
elements=[];const plainHost=el('x-host',{id:'host'},[]);plainHost.shadowRoot={nodeType:11,firstChild:el('span',{},[text('Anonymous shadow')])};setup([plainHost]);out=collect([0,0,200,200]);
assert.equal(out.visible_text.find(x=>x.text==='Anonymous shadow').selector,'#host >>> span:nth-child(1)');
