import {test} from 'node:test';
import assert from 'node:assert/strict';
import {mountOperator} from '../../cmd/internal/webcast/operator.js';

// Exercise the live operator adapter, not createPlacement with a resolver that
// the real adapter never supplies. The page and isolated control root are separate.
function operatorPage(text='Save', {bind=true,sketch=false}={}) {
 const win={innerWidth:800,innerHeight:600,scrollX:0,scrollY:0,location:{href:'https://example.test/form'},getComputedStyle:n=>n.style,addEventListener(){},setTimeout:()=>1,clearTimeout(){}};
 const doc={nodeType:9,childNodes:[],defaultView:win};win.document=doc;
 const el=(tag,parent=null,attrs={},text='')=>{
  const n={nodeType:1,localName:tag,ownerDocument:doc,parentNode:parent,childNodes:[],attrs,style:{},rect:{left:10,top:20,width:100,height:50},appendChild(child){this.childNodes.push(child);child.parentNode=this;},getAttribute(k){return this.attrs[k]||'';},getBoundingClientRect(){return this.rect;},getClientRects(){return [this.rect];},getRootNode(){let p=this;while(p.parentNode)p=p.parentNode;return p;},get isConnected(){return this.getRootNode()===doc;}};
  parent?.childNodes.push(n);if(text)n.childNodes.push({nodeType:3,nodeValue:text,parentNode:n,ownerDocument:doc});return n;
 };
 doc.createElement=tag=>el(tag);
 const main=el('main',doc),anchor=el('button',main,{},text),host=el('couchcast-control',doc),root=el('shadow');
 const note={id:'viewer-note',author:{id:'viewer'},at:{node:7,tag:'button',text,path:['main','button'],route:win.location.href,box:[10,20,100,50]},region:[20,30,40,20]};
 let view;
 mountOperator({host,root,report(){}},{annotate:{mountAnnotation(_doc,_win,surface){view=surface;return{strokes:{clear(){}},setAnnotationAllowed(){},setMode(){}};}},palette:[],css:'',sketch:sketch?{mount(){return {tool:{},receiver:{hasAnchor:(document,id)=>document===1&&id===7}};}}:undefined},doc,win);
 const snapshot=(revision=1,document=1)=>win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},revision,doc:document,seg:document,t:revision,notes:sketch?[]:[note],strokes:[]});
 snapshot();
 if(bind) win.__couchcastOperatorOriginal?.(1,7,anchor);
 return{win,doc,el,main,anchor,note,snapshot,place:()=>view.placementOf(note)};
}

test('operator preserves a viewer-created note on its unchanged common-text Save button',()=>{
 const f=operatorPage();assert.deepEqual(f.place(),f.note.region);
 f.anchor.rect.left=110;assert.deepEqual(f.place(),[120,30,40,20]);
});

test('operator preserves its unchanged anchor beyond the replacement scan cap',()=>{
 const f=operatorPage('Distinct original action title');for(let i=0;i<2100;i++)f.el('i',f.main);
 assert.deepEqual(f.place(),f.note.region);
});

test('operator does not treat a same-shape replacement with common text as original',()=>{
 const f=operatorPage();f.main.childNodes=[];f.anchor.parentNode=null;
 f.el('button',f.main,{},'Save');assert.equal(f.place(),null);
});

test('operator does not promote a reused recording ID after a document switch',()=>{
 const f=operatorPage();f.snapshot(2,2);assert.equal(f.place(),null);
 f.win.location.href='https://example.test/another';assert.equal(f.place(),null);
});


test('operator rejects stale-document and foreign-document identity evidence',()=>{
 const f=operatorPage('Save',{bind:false}),other=operatorPage('Save',{bind:false});
 assert.equal(typeof f.win.__couchcastOperatorOriginal,'function');
 assert.equal(f.win.__couchcastOperatorOriginal(2,7,f.anchor),false);
 assert.equal(f.win.__couchcastOperatorOriginal(1,7,other.anchor),false);
 assert.equal(f.win.__couchcastOperatorOriginal(1,7,f.anchor.childNodes[0]),false);
 assert.equal(f.place(),null);
 assert.equal(f.win.__couchcastOperatorOriginal(1,7,f.anchor),true);
 assert.deepEqual(f.place(),f.note.region);
 f.snapshot(2,2);assert.equal(f.win.__couchcastOperatorOriginal(1,7,f.anchor),false);assert.equal(f.place(),null);
});

test('operator never rebinds an original recording ID to a different same-shape node',()=>{
 const f=operatorPage(),replacement=f.el('button',f.main,{},'Save');
 assert.equal(typeof f.win.__couchcastOperatorOriginal,'function');
 assert.equal(f.win.__couchcastOperatorOriginal(1,7,replacement),false);
 f.main.childNodes=f.main.childNodes.filter(n=>n!==f.anchor);f.anchor.parentNode=null;
 assert.equal(f.win.__couchcastOperatorOriginal(1,7,replacement),false);assert.equal(f.place(),null);
});

test('operator retains trusted identity on same-document snapshots but prunes retired notes',()=>{
 const f=operatorPage();assert.equal(typeof f.win.__couchcastOperatorOriginal,'function');
 f.snapshot(2,1);assert.deepEqual(f.place(),f.note.region);
 f.win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},revision:3,doc:1,seg:1,t:3,notes:[],strokes:[]});
 assert.equal(f.place(),null);
});


test('operator retains paused pins and ignores stale snapshots or unknown-node bindings',()=>{
 const f=operatorPage();assert.equal(f.win.__couchcastOperatorOriginal(1,999,f.anchor),false);
 f.win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},revision:2,doc:1,seg:2,t:2,paused:true,notes:[f.note],strokes:[]});
 assert.deepEqual(f.place(),f.note.region);
 f.snapshot(1,2);assert.deepEqual(f.place(),f.note.region);
 f.anchor.style.visibility='hidden';assert.equal(f.place(),null);
 delete f.anchor.style.visibility;assert.deepEqual(f.place(),f.note.region);
 f.snapshot(3,0);assert.equal(f.place(),null);assert.equal(f.win.__couchcastOperatorOriginal(1,7,f.anchor),false);
});

test('operator binds unchanged sketch anchor without an open note',()=>{
 const f=operatorPage('Save',{sketch:true});
 assert.deepEqual(f.place(),f.note.region);
 f.anchor.rect.left=110;assert.deepEqual(f.place(),[120,30,40,20]);
 f.snapshot(2);assert.deepEqual(f.place(),[120,30,40,20]);
 f.main.childNodes=[];f.anchor.parentNode=null;f.el('button',f.main,{},'Save');assert.equal(f.place(),null);
});
