import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {mountFeedbackPanel} from '../../cmd/internal/webcast/web/feedback-panel.js';
import {feedbackLayout} from '../../cmd/internal/webcast/web/feedback-layout.js';

class Node extends Element {
 constructor(tag,cost){super(tag);this.cost=cost;this.hidden=false;this._text='';this.style=new Proxy({}, {set:(o,k,v)=>{cost.mutations++;o[k]=v;return true;}});}
 set textContent(v){this.cost.mutations++;this._text=v;} get textContent(){return this._text;}
 setAttribute(k,v){this.attributes??={};this.attributes[k]=String(v);}
 getAttribute(k){return this.attributes?.[k]??null;}
 set innerHTML(v){throw Error('markup forbidden');}
 append(...nodes){for(const node of nodes){node.remove();node.parentNode=this;this.children.push(node);this.cost.mutations++;}}
 insertBefore(node,before){node.remove();node.parentNode=this;this.children.splice(this.children.indexOf(before),0,node);this.cost.mutations++;}
 remove(){if(this.parentNode){const p=this.parentNode;p.children=p.children.filter(x=>x!==this);this.parentNode=null;this.cost.mutations++;}}
 removeEventListener(k,f){this.listeners[k]=(this.listeners[k]||[]).filter(x=>x!==f);}
 getBoundingClientRect(){this.cost.geometry++;return {width:300,height:320};}
 contains(other){return other===this||this.children.some(c=>c.contains(other));}
 fire(type,extra={}){const e={target:this,preventDefault(){},stopPropagation(){this.stopped=true;},...extra};for(let n=this;n;n=n.parentNode){for(const f of [...(n.listeners[type]||[])])f(e);if(e.stopped)break;}return e;}
}
function fixture(layout=()=>null){const cost={mutations:0,geometry:0},doc={createElement:tag=>new Node(tag,cost)};const win=new Node('window',cost);win.innerWidth=800;win.innerHeight=600;const parent=doc.createElement('main');return {...mountFeedbackPanel(doc,win,{parent,layout}),doc,win,cost,parent};}
const rows=f=>f.body.querySelector('.feedback-rows').children;
const button=(row,cls)=>row.querySelector('.feedback-'+cls);
test('latest typed keys preserve rows and route latest sources and handlers',()=>{
 const f=fixture(),calls=[];
 f.setNotes([{id:'x',text:'<img onerror=x>'}],{open:n=>calls.push(n.text),reveal:n=>calls.push(n.id)});
 f.setSketches([{sketch:'x',label:'draft',editable:true}],{open:s=>calls.push(s.label)});
 const [note,sketch]=rows(f);assert.equal(note.querySelector('.feedback-text').textContent,'<img onerror=x>');
 assert.equal(button(note,'open').textContent,'Open note');assert.equal(button(sketch,'open').textContent,'View note');
 f.setNotes([{id:'x',text:'latest'}],{open:n=>calls.push('new '+n.text),reveal:n=>calls.push(n.id)});
 f.setSketches([{sketch:'x',label:'',editable:true}],{open:s=>calls.push('new '+s.label)});
 assert.equal(rows(f)[0],note);assert.equal(rows(f)[1],sketch);assert.equal(button(sketch,'open').textContent,'Add note');
 button(note,'open').fire('click');button(note,'reveal').fire('click');button(sketch,'open').fire('click');
 assert.deepEqual(calls,['new latest','x','new ']);
 f.setNotes([]);assert.equal(rows(f).length,1);button(note,'open').fire('click');assert.equal(calls.length,3);
});
test('finished sketches show read-only text and never displace a visible adopted editor',()=>{
 const f=fixture(),old=f.doc.createElement('aside'),editor=f.doc.createElement('div');old.append(editor);
 editor.className='note-dialog hidden';editor.value='unsaved';f.adoptEditor(editor);
 assert.equal(editor.parentNode,f.body);assert.equal(editor.classList.contains('feedback-editor'),true);
 f.setSketches([{sketch:'done',label:'<b>finished</b>'}],{open:()=>assert.fail('finished sketch must be readonly')});
 button(rows(f)[0],'open').fire('click');const readonly=f.body.querySelector('.feedback-readonly');
 assert.equal(readonly.hidden,false);assert.equal(readonly.querySelector('.feedback-readonly-text').textContent,'<b>finished</b>');
 readonly.querySelector('.feedback-readonly-close').fire('click');assert.equal(readonly.hidden,true);
 editor.classList.remove('hidden');assert.equal(f.showReadonly('blocked'),false);assert.equal(editor.value,'unsaved');
 f.dispose();assert.equal(editor.parentNode,old);assert.equal(editor.value,'unsaved');
});
test('hover and focus have no DOM work, survive refresh and clear on leave, minimize, disable and dispose',()=>{
 const f=fixture(),calls=[],handlers={hover:n=>calls.push(n?.id??null)};
 f.setNotes([{id:'n',text:'note'}],handlers);const row=rows(f)[0];
 let before=f.cost.mutations;row.fire('mouseenter');row.fire('focusin');assert.equal(f.cost.mutations,before);assert.equal(calls.at(-1),'n');
 row.fire('mouseleave');assert.equal(calls.at(-1),null);
 row.fire('focusin');row.fire('focusout',{relatedTarget:button(row,'open')});assert.equal(calls.at(-1),'n');
 row.fire('focusout');assert.equal(calls.at(-1),null);
 row.fire('mouseenter');f.setNotes([{id:'n',text:'new'}],handlers);assert.equal(calls.at(-1),'n');
 for(const action of [()=>f.root.querySelector('.floating-panel-minimize').fire('click'),()=>f.setEnabled(false),()=>f.dispose()]){row.fire('mouseenter');action();assert.equal(calls.at(-1),null);}
 const snapshot={...f.cost};f.setNotes([{id:'z'}]);row.fire('mouseenter');f.dispose();assert.deepEqual(f.cost,snapshot);
});
test('fixed title/status priority and layout reserve toolbar even during keyboard movement',()=>{
 let width=800;const f=fixture(()=>feedbackLayout({width,height:600}));assert.equal(f.root.hidden,false);
 assert.equal(f.header.querySelector('.floating-panel-drag').textContent,'Notes (0)');
 f.setStatus('preview','preview');f.setStatus('editor','editing');f.setStatus('control','control');
 assert.equal(f.status.textContent,'editing · preview');f.setStatus('editor','');assert.equal(f.status.textContent,'control · preview');
 const handle=f.header.querySelector('.floating-panel-drag');for(let i=0;i<100;i++)handle.fire('keydown',{key:'ArrowRight',shiftKey:true});
 const left=Number.parseFloat(f.root.style.inset.split(' ')[3]);assert.ok(left+300<=720);
 width=10;f.refresh();assert.equal(f.root.hidden,true);f.setNotes([{id:'n'}]);assert.equal(f.root.hidden,true);
 width=800;f.refresh();assert.equal(f.root.hidden,false);f.setEnabled(false);assert.equal(f.root.hidden,true);f.win.fire('resize');assert.equal(f.root.hidden,true);
});
test('reconciliation scales linearly with stable identities and geometry stays constant at 32/64',()=>{
 const run=n=>{const f=fixture();const source=Array.from({length:n},(_,i)=>({id:String(i),text:'note'}));f.setNotes(source);const saved=[...rows(f)],start={...f.cost};f.setNotes(source.map(x=>({...x,text:'changed'})));assert.deepEqual(rows(f),saved);return {mutations:f.cost.mutations-start.mutations,geometry:f.cost.geometry-start.geometry};};
 const a=run(32),b=run(64);assert.equal(a.geometry,b.geometry);assert.ok(b.geometry<=1);assert.ok(b.mutations<=2*a.mutations+10);
});

test('finished empty sketches remain view actions and reordered sources retain their nodes',()=>{
 const f=fixture();f.setSketches([{sketch:'a',label:'',editable:false},{sketch:'b',label:'b'}]);
 const [a,b]=rows(f);assert.equal(button(a,'open').textContent,'View note');
 f.setSketches([{sketch:'b',label:'latest'},{sketch:'a',label:''}]);
 assert.deepEqual(rows(f),[b,a]);assert.equal(b.querySelector('.feedback-text').textContent,'latest');
});

test('minimized intrinsic width stays inside protected content after count growth',()=>{
 const f=fixture(()=>feedbackLayout({width:800,height:600}));
 f.root.getBoundingClientRect=()=>({width:230,height:44});
 f.setNotes([{id:'n'}]);f.root.querySelector('.floating-panel-minimize').fire('click');
 const restore=f.root.querySelector('.floating-panel-restore');for(let i=0;i<50;i++)restore.fire('keydown',{key:'ArrowRight',shiftKey:true});
 f.setNotes(Array.from({length:64},(_,i)=>({id:String(i)})));
 assert.ok(Number.parseFloat(f.root.style.inset.split(' ')[3])+230<=720);
});

test('dropping protected layout restores standalone sizing and teardown removes all listeners',()=>{
 let layout=feedbackLayout({width:800,height:600});const f=fixture(()=>layout);
 f.setNotes([{id:'one'}]);
 assert.equal(f.root.style.width,'300px');layout=null;f.refresh();assert.equal(f.root.style.width,'');
 const nodes=[];const collect=n=>{nodes.push(n);for(const child of n.children)collect(child);};collect(f.root);
 f.dispose();const snapshot={...f.cost};f.win.fire('resize');assert.deepEqual(f.cost,snapshot);
 for(const node of [...nodes,f.win])assert.equal(Object.values(node.listeners).flat().length,0);
 assert.equal(f.parent.children.length,0);
});

test('hover refresh uses the latest source and handler while explicit scope clearing survives refresh',()=>{
 const f=fixture(),calls=[];f.setNotes([{id:'n',text:'old'}],{hover:n=>calls.push(n?.text??null)});
 const row=rows(f)[0];row.fire('mouseenter');
 f.setNotes([{id:'n',text:'new'}],{hover:n=>calls.push(n?'latest '+n.text:null)});
 assert.equal(calls.at(-1),'latest new');assert.equal(rows(f)[0],row);
 f.clearHighlight();assert.equal(calls.at(-1),null);const before=calls.length;
 f.setNotes([{id:'n',text:'later'}],{hover:n=>calls.push(n?.text??null)});assert.equal(calls.length,before);
 row.fire('mouseenter');f.setNotes([]);assert.equal(calls.at(-1),null);
});

test('repeated enable sync preserves active hover and avoids geometry work',()=>{
 const f=fixture(),calls=[],handlers={hover:n=>calls.push(n?.text??null)};
 f.setNotes([{id:'n',text:'before'}],handlers);rows(f)[0].fire('mouseenter');
 const before={...f.cost};f.setEnabled(true);assert.equal(calls.at(-1),'before');assert.deepEqual(f.cost,before);
 f.setNotes([{id:'n',text:'after'}],handlers);f.setEnabled(1);assert.equal(calls.at(-1),'after');
 f.setEnabled(false);assert.equal(calls.at(-1),null);const disabled={...f.cost};f.setEnabled(0);assert.deepEqual(f.cost,disabled);
});
test('selected read-only sketch follows latest text and closes when its row disappears',()=>{
 const f=fixture();f.setSketches([{sketch:'s',label:'old'}]);button(rows(f)[0],'open').fire('click');
 const pane=f.body.querySelector('.feedback-readonly'),text=pane.querySelector('.feedback-readonly-text');assert.equal(pane.hidden,false);
 f.setNotes([{id:'n',text:'ordinary sync'}]);assert.equal(pane.hidden,false);
 f.setSketches([{sketch:'s',label:'<b>latest</b>'}]);assert.equal(text.textContent,'<b>latest</b>');
 f.setSketches([]);assert.equal(pane.hidden,true);
 f.showReadonly('transient');assert.equal(pane.hidden,false);f.setNotes([]);assert.equal(pane.hidden,true);
});

test('row availability is display-only, explicit and refreshed for notes and sketches',()=>{
 const f=fixture(),note={id:'n',text:'original'},sketch={sketch:'s',label:'shape',editable:true};
 const originals=structuredClone([note,sketch]);let available=false;
 f.setNotes([note],{available:()=>available});f.setSketches([sketch]);
 const [n,s]=rows(f);assert.equal(n.dataset.placement,'unavailable');assert.equal(n.querySelector('.feedback-location').textContent,'not on this view');
 assert.equal(s.dataset.placement,'unknown');assert.equal(s.querySelector('.feedback-location').textContent,'');
 available=true;f.setNotes([note],{available:()=>available});assert.equal(rows(f)[0],n);assert.equal(n.dataset.placement,'visible');assert.equal(n.querySelector('.feedback-location').textContent,'');
 f.setSketches([sketch],{available:()=>false});assert.equal(s.dataset.placement,'unavailable');assert.equal(s.querySelector('.feedback-location').textContent,'not on this view');
 f.setNotes([note]);assert.equal(n.dataset.placement,'unknown');assert.deepEqual([note,sketch],originals);
});

test('pending count remains visible beside control status without repeated DOM writes',()=>{
 const f=fixture();f.setStatus('control','Annotation is off.');f.setStatus('preview','Pending feedback: 1 sketches, 0 annotations');
 assert.equal(f.status.textContent,'Annotation is off. · Pending feedback: 1 sketches, 0 annotations');
 const before=f.cost.mutations;f.setStatus('control','Annotation is off.');f.setStatus('preview','Pending feedback: 1 sketches, 0 annotations');assert.equal(f.cost.mutations,before);
 f.setStatus('preview','');assert.equal(f.status.textContent,'Annotation is off.');
});

test('minimized Notes visibly conveys status and restores the full message without losing position',()=>{
 const f=fixture(()=>feedbackLayout({width:800,height:600}));f.setNotes([{id:'n',text:'note'}]);
 f.root.querySelector('.floating-panel-minimize').fire('click');const restore=f.root.querySelector('.floating-panel-restore');
 const label=restore.textContent;f.setStatus('preview','Pending feedback is out of date.');
 assert.notEqual(restore.textContent,label);assert.match(restore.textContent,/Status/);
 assert.match(restore.getAttribute('aria-label'),/Pending feedback is out of date/);
 assert.equal(restore.title,'Pending feedback is out of date.');
 restore.fire('click');assert.equal(f.header.hidden,false);assert.equal(f.status.textContent,'Pending feedback is out of date.');
 f.setStatus('preview','');f.root.querySelector('.floating-panel-minimize').fire('click');assert.equal(restore.textContent,label);
 assert.equal(restore.title,'');assert.equal(restore.getAttribute('aria-label'),label);
});

test('opening an adopted editor restores minimized Notes and retires read-only view',()=>{
 const f=fixture(),editor=f.doc.createElement('div');editor.hidden=true;f.adoptEditor(editor);
 f.setSketches([{sketch:'s',label:'finished'}]);button(rows(f)[0],'open').fire('click');
 f.root.querySelector('.floating-panel-minimize').fire('click');assert.equal(f.body.hidden,true);
 assert.equal(f.showEditor(editor),true);editor.hidden=false;
 assert.equal(f.body.hidden,false);assert.equal(f.header.hidden,false);
 assert.equal(f.body.querySelector('.feedback-readonly').hidden,true);
 assert.equal(f.showEditor(f.doc.createElement('div')),false);
 f.dispose();assert.equal(f.showEditor(editor),false);
});
