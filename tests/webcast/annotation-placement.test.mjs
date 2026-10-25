import {test} from 'node:test';
import assert from 'node:assert/strict';
import {preparePageInk,prepareHistoricalPageInk} from '../../cmd/internal/webcast/web/sketch-page-image.js';
import {createPlacement} from '../../cmd/internal/webcast/web/annotation-placement.js';
function fixture() {
 const win={innerWidth:800,innerHeight:600,scrollX:0,scrollY:0,getComputedStyle:n=>n.style};
 const doc={nodeType:9,childNodes:[],defaultView:win}; win.document=doc;
 const el=(tag,parent=doc,attrs={},text='')=>{
  const n={nodeType:1,localName:tag,ownerDocument:doc,parentNode:parent,childNodes:[],attrs,style:{},rect:{left:10,top:20,width:100,height:50},getAttribute(k){return this.attrs[k]||'';},getBoundingClientRect(){return this.rect;},getClientRects(){return [this.rect];},getRootNode(){let p=this;while(p.parentNode)p=p.parentNode;return p;},get isConnected(){return this.getRootNode()===doc;}};
  parent.childNodes.push(n);if(text)n.childNodes.push({nodeType:3,nodeValue:text,parentNode:n,ownerDocument:doc});return n;
 };
 const main=el('main'), anchor=el('p',main,{},'Distinct article heading');
 const note={at:{tag:'p',text:'Distinct article heading',path:['main','p'],route:'https://example.test/a',box:[10,20,100,50]},region:[20,30,40,20]};
 let route=note.at.route,original=anchor;
 const place=createPlacement({document:()=>doc,route:()=>route,original:()=>original});
 return {doc,win,el,main,anchor,note,place,setRoute:r=>route=r,setOriginal:n=>original=n};
}
test('live region follows connected original geometry; hidden ancestors and missing geometry hide',()=>{
 const f=fixture();f.anchor.rect={left:100,top:100,width:200,height:100};
 assert.deepEqual(f.place(f.note),[120,120,80,40]);
 for(const [key,value] of [['display','none'],['visibility','hidden'],['opacity','0'],['contentVisibility','hidden']]){f.main.style[key]=value;assert.equal(f.place(f.note),null,key);delete f.main.style[key];}
 f.anchor.rect.width=0;assert.equal(f.place(f.note),null);
});

test('paint witness uses the same chosen node and geometry without a second matcher',()=>{
 const f=fixture(),chosen=[];
 const place=createPlacement({document:()=>f.doc,route:()=>f.note.at.route,original:()=>f.anchor,locator:node=>{chosen.push(node);return [0,-1,2];}});
 assert.deepEqual(place(f.note,{withWitness:true}),{box:f.note.region,clip:[0,0,800,600],locator:[0,-1,2]});
 assert.deepEqual(chosen,[f.anchor]);assert.deepEqual(place(f.note),f.note.region);
 const unavailable=createPlacement({document:()=>f.doc,route:()=>f.note.at.route,original:()=>f.anchor,locator:()=>{throw Error('not recorded');}});
 assert.deepEqual(unavailable(f.note,{withWitness:true}),{box:f.note.region,clip:[0,0,800,600],locator:null},'unavailable witness must preserve ordinary painting geometry');
});
test('SPA replacement must be unique, same route and conflict checked; hidden original never jumps',()=>{
 const f=fixture(); f.main.childNodes=[]; f.anchor.parentNode=null;
 const replacement=f.el('p',f.main,{},'Distinct article heading');replacement.rect.left=200;
 assert.deepEqual(f.place(f.note),[210,30,40,20]);
 const duplicate=f.el('p',f.main,{},'Distinct article heading');assert.equal(f.place(f.note),null);
 f.main.childNodes.pop();duplicate.parentNode=null;f.setRoute('https://example.test/b');assert.equal(f.place(f.note),null);
 f.setRoute(f.note.at.route);f.setOriginal(replacement);replacement.style.display='none';f.el('p',f.main,{},'Distinct article heading');assert.equal(f.place(f.note),null);
 replacement.style.display='';replacement.childNodes[0].nodeValue='A different article';assert.equal(f.place(f.note),null);
});
test('operator matching uses evidence, cap refuses a falsely unique truncated scan',()=>{
 const f=fixture();f.setOriginal(null);assert.deepEqual(f.place(f.note),f.note.region);
 for(let i=0;i<2000;i++)f.el('i',f.main);
 assert.equal(f.place(f.note),null);
});
test('clip and viewport intersection hide invisible regions, including wholly clipped offset regions',()=>{
 const f=fixture();f.main.style.overflow='hidden';f.main.rect={left:0,top:0,width:5,height:5};assert.equal(f.place(f.note),null);
 delete f.main.style.overflow;f.note.region=[1000,1000,20,20];assert.equal(f.place(f.note),null);
});
test('missing modern identity cannot reanchor and historical absolute regions never substitute',()=>{
 const f=fixture();f.setOriginal(null);delete f.note.at.tag;assert.equal(f.place(f.note),null);
});
test('connected recorded anchor remains usable on a page above the replacement scan cap',()=>{
 const f=fixture();for(let i=0;i<2100;i++)f.el('i',f.main);
 assert.deepEqual(f.place(f.note),f.note.region);
});
test('a text budget cannot turn a long different value into a false unique match',()=>{
 const f=fixture();f.setOriginal(null);f.anchor.childNodes[0].nodeValue='Distinct article heading'+' '.repeat(5000)+'different ending';
 assert.equal(f.place(f.note),null);
});
test('replacement scopes reject frame documents and shadow trees inside frames',()=>{
 const f=fixture();f.setOriginal(null);const frame=f.el('iframe',f.main);f.doc.querySelectorAll=()=>[frame];
 for(const sel of ['iframe >>> p','iframe >>> section >>> p']){f.note.at.sel=sel;assert.equal(f.place(f.note),null);}
});
test('shadow replacements require a unique host and cannot escape their original scope',()=>{
 const f=fixture();f.setOriginal(null);const host=f.el('section',f.main),root={nodeType:11,host,childNodes:[],ownerDocument:f.doc};host.shadowRoot=root;
 const child=f.el('p',root,{},'Distinct article heading');Object.defineProperty(child,'isConnected',{get:()=>host.isConnected});
 f.doc.querySelectorAll=()=>[host];f.note.at.sel='section >>> p';f.note.at.path=['main','section','p'];
 assert.deepEqual(f.place(f.note),f.note.region);
 f.doc.querySelectorAll=()=>[host,host];assert.equal(f.place(f.note),null);
});
test('observer invalidates cached candidate facts before the next synchronous placement and requests redraw',()=>{
 const f=fixture();let observer,changed=0;
 f.win.MutationObserver=class {constructor(callback){this.callback=callback;observer=this;this.records=[];}observe(){}disconnect(){}takeRecords(){const r=this.records;this.records=[];return r;}};
 const place=createPlacement({document:()=>f.doc,route:()=>f.note.at.route,onChange:()=>changed++});
 assert.deepEqual(place(f.note),f.note.region);
 f.anchor.childNodes[0].nodeValue='Other text';observer.records.push({type:'characterData'});assert.equal(place(f.note),null);
 observer.callback([]);assert.equal(changed,1);
});
test('Renderer wires recorded identity into the same strict adapter',async()=>{
 const {Renderer}=await import('../../cmd/internal/webcast/web/render.js');const f=fixture();f.note.at.node=7;
 const renderer=Object.create(Renderer.prototype);Object.assign(renderer,{frame:{contentDocument:f.doc},model:{url:f.note.at.route},dom:new Map([[7,f.anchor]]),ids:new WeakMap([[f.anchor,7]]),shadows:new Map()});
 assert.deepEqual(renderer.placementOf(f.note),f.note.region);
 f.main.style.display='none';assert.equal(renderer.placementOf(f.note),null);
});
test('connected frame anchor composes geometry but detached frame owners hide',()=>{
 const f=fixture(),inner=fixture(),frame=f.el('iframe',f.main);
 inner.win.frameElement=frame;f.setOriginal(inner.anchor);
 assert.deepEqual(f.place(f.note),[30,50,40,20]);
 frame.parentNode=null;f.main.childNodes=f.main.childNodes.filter(n=>n!==frame);
 assert.equal(f.place(f.note),null);
});
test('Renderer never treats a reused ID on a later document as the original anchor',async()=>{
 const {Renderer}=await import('../../cmd/internal/webcast/web/render.js');const f=fixture();f.note.id='old';f.note.at.node=7;f.note.at.text='Save';f.anchor.childNodes[0].nodeValue='Save';
 const renderer=Object.create(Renderer.prototype);Object.assign(renderer,{frame:{contentDocument:f.doc},model:{url:f.note.at.route,docSeg:2,notes:[{seg:1,note:f.note}]},dom:new Map([[7,f.anchor]]),ids:new WeakMap([[f.anchor,7]]),shadows:new Map()});
 assert.equal(renderer.placementOf(f.note),null);
 renderer.model.docSeg=1;assert.deepEqual(renderer.placementOf(f.note),f.note.region);
});
test('unsupported clip paths, masks and legacy clipping hide positive-size anchors conservatively',()=>{
 const f=fixture();for(const [key,value] of [['clipPath','inset(100%)'],['maskImage','linear-gradient(transparent,transparent)'],['webkitMaskImage','url(mask.svg)'],['clip','rect(0px,0px,0px,0px)']]){f.main.style[key]=value;assert.equal(f.place(f.note),null,key);delete f.main.style[key];}
});

test('sketch source identity proves an original node without note membership',async()=>{
 const {Renderer}=await import('../../cmd/internal/webcast/web/render.js');const f=fixture();
 f.note.at.node=7;f.note.at.text='Save';f.anchor.childNodes[0].nodeValue='Save';delete f.note.region;
 f.note.source={doc:1,tab:'t',t:3};
 const renderer=Object.create(Renderer.prototype);Object.assign(renderer,{frame:{contentDocument:f.doc},model:{url:f.note.at.route,docSeg:1,tab:{id:'t'},t:4,notes:[]},dom:new Map([[7,f.anchor]]),ids:new WeakMap([[f.anchor,7]]),shadows:new Map()});
 assert.deepEqual(renderer.placementOf(f.note),[10,20,100,50]);
 renderer.model.docSeg=2;assert.equal(renderer.placementOf(f.note),null);
 renderer.model.docSeg=1;renderer.model.tab.id='other';assert.equal(renderer.placementOf(f.note),null);
});

test('historical originalOnly refuses replacement matching while live placement retains it',async()=>{
 const {Renderer}=await import('../../cmd/internal/webcast/web/render.js');const f=fixture();
 f.note.at.node=7;delete f.note.region;f.note.source={doc:1,tab:'t',t:3};
 const renderer=Object.create(Renderer.prototype);Object.assign(renderer,{frame:{contentDocument:f.doc},model:{url:f.note.at.route,docSeg:1,tab:{id:'t'},t:4,notes:[]},dom:new Map([[7,f.anchor]]),ids:new WeakMap([[f.anchor,7]]),shadows:new Map()});
 const strict=()=>renderer.placementOf(f.note,{withClip:true,originalOnly:true});
 assert.deepEqual(strict(),{box:[10,20,100,50],clip:[0,0,800,600]});
 f.main.childNodes=[];f.anchor.parentNode=null;const replacement=f.el('p',f.main,{},'Distinct article heading');replacement.rect.left=200;
 assert.deepEqual(renderer.placementOf(f.note),[200,20,100,50]);
 assert.throws(()=>preparePageInk({scene:{doc:1,tab:'t',t:4},region:[0,0,800,600],scale:1,width:800,height:600},[{index:1,id:'s',source:f.note.source,paths:[{id:'p',at:f.note.at,width:4,pts:[[30,30,1],[50,40,1]]}]}],strict),/index 1.*unavailable/);
 assert.equal(strict(),null);
});

test('historical page production adapter requires connected original identity in every case',async()=>{
 const {Renderer}=await import('../../cmd/internal/webcast/web/render.js');
 for(const failure of ['connected','missing','disconnected','hidden','conflicting','wrongdoc','wrongtab']){
  const f=fixture();f.note.at.node=7;delete f.note.region;f.note.source={doc:1,tab:'t',t:3};
  const renderer=Object.create(Renderer.prototype);Object.assign(renderer,{frame:{contentDocument:f.doc},model:{url:f.note.at.route,docSeg:1,tab:{id:'t'},t:4,notes:[]},dom:new Map([[7,f.anchor]]),ids:new WeakMap([[f.anchor,7]]),shadows:new Map()});
  const replacement=f.el('p',f.main,{},'Distinct article heading');replacement.rect.left=200;
  if(failure==='missing')renderer.dom.delete(7);
  if(failure==='disconnected'){f.main.childNodes=f.main.childNodes.filter(n=>n!==f.anchor);f.anchor.parentNode=null;}
  if(failure==='hidden')f.anchor.style.display='none';
  if(failure==='conflicting')f.anchor.childNodes[0].nodeValue='Different heading';
  if(failure==='wrongdoc')renderer.model.docSeg=2;
  if(failure==='wrongtab')renderer.model.tab.id='other';
  const run=()=>prepareHistoricalPageInk({scene:{doc:1,tab:'t',t:4},region:[0,0,800,600],scale:1,width:800,height:600},[{index:1,id:'s',source:f.note.source,paths:[{id:'p',at:f.note.at,width:4,pts:[[30,30,1],[50,40,1]]}]}],renderer);
  if(failure==='connected')assert.deepEqual(run().shapes[0].paths[0].projected,[[30,30,1],[50,40,1]]);
  else assert.throws(run,/index 1.*path p unavailable/,failure);
 }
});

test('reveal scrolls only a confirmed original and recomputes strict placement',()=>{
 const f=fixture(),calls=[];f.anchor.rect.top=900;
 f.anchor.scrollIntoView=options=>{calls.push(options);f.anchor.rect.top=200;};
 assert.equal(f.place(f.note,{originalOnly:true}),null);
 assert.equal(f.place.reveal(f.note),true);
 assert.deepEqual(calls,[{block:'center',inline:'nearest'}]);
 assert.deepEqual(f.place(f.note,{originalOnly:true}),[20,210,40,20]);
});

test('reveal refuses untrusted, unsafe and foreign originals without scrolling',()=>{
 for(const failure of ['missing','detached','conflicting','wrongroute','hidden','mask','transform','empty','foreign','clipped']){
  const f=fixture();let calls=0;f.anchor.scrollIntoView=()=>calls++;
  f.el('p',f.main,{},'Distinct article heading'); // A matching replacement is never authority.
  if(failure==='missing')f.setOriginal(null);
  if(failure==='detached'){f.main.childNodes=f.main.childNodes.filter(n=>n!==f.anchor);f.anchor.parentNode=null;}
  if(failure==='conflicting')f.anchor.childNodes[0].nodeValue='Different article';
  if(failure==='wrongroute')f.setRoute('https://example.test/b');
  if(failure==='hidden')f.main.style.visibility='hidden';
  if(failure==='mask')f.main.style.maskImage='url(mask.svg)';
  if(failure==='transform')f.main.style.transform='rotate(30deg)';
  if(failure==='empty')f.anchor.rect.width=0;
  if(failure==='foreign'){const other=fixture();other.anchor.scrollIntoView=()=>calls++;f.setOriginal(other.anchor);}
  if(failure==='clipped'){f.main.style.overflow='hidden';f.main.rect={left:0,top:0,width:5,height:5};}
  assert.equal(f.place.reveal(f.note),false,failure);assert.equal(calls,0,failure);
 }
});

test('reveal does not claim success before post-scroll strict geometry is available',()=>{
 const f=fixture();let calls=0;f.anchor.rect.top=900;f.anchor.scrollIntoView=()=>calls++;
 assert.equal(f.place.reveal(f.note),false);assert.equal(calls,1);
});

test('reveal traverses an offscreen frame only through connected safe owners',()=>{
 for(const failure of ['none','detached','transformed','masked']){
  const f=fixture(),inner=fixture(),frame=f.el('iframe',f.main);let calls=0;
  frame.rect.top=900;inner.win.frameElement=frame;f.setOriginal(inner.anchor);
  inner.anchor.scrollIntoView=()=>{calls++;frame.rect.top=100;};
  if(failure==='detached'){f.main.childNodes=[];frame.parentNode=null;}
  if(failure==='transformed')frame.style.transform='rotate(30deg)';
  if(failure==='masked')frame.style.maskImage='url(mask.svg)';
  assert.equal(f.place.reveal(f.note),failure==='none',failure);
  assert.equal(calls,failure==='none'?1:0,failure);
 }
});

test('Renderer reveal retains source guards and current-note ownership on repeated URLs',async()=>{
 const {Renderer}=await import('../../cmd/internal/webcast/web/render.js');
 for(const scenario of ['current','historic','unknown','source','wrongdoc','wrongtab','future']){
  const f=fixture();f.note.id='n';f.note.at.node=7;let calls=0;
  const renderer=Object.create(Renderer.prototype);Object.assign(renderer,{frame:{contentDocument:f.doc},model:{url:f.note.at.route,docSeg:2,tab:{id:'t'},t:4,notes:[{seg:2,note:f.note}]},dom:new Map([[7,f.anchor]]),ids:new WeakMap([[f.anchor,7]]),shadows:new Map()});
  f.anchor.rect.top=900;f.anchor.scrollIntoView=()=>{calls++;f.anchor.rect.top=200;};
  if(scenario==='historic')renderer.model.notes[0].seg=1;
  if(scenario==='unknown')renderer.model.notes=[];
  if(['source','wrongdoc','wrongtab','future'].includes(scenario)){
   renderer.model.notes=[];f.note.source={doc:scenario==='wrongdoc'?1:2,tab:scenario==='wrongtab'?'other':'t',t:scenario==='future'?5:3};
  }
  const allowed=['current','source'].includes(scenario);
  assert.equal(renderer.reveal(f.note),allowed,scenario);assert.equal(calls,allowed?1:0,scenario);
 }
});
