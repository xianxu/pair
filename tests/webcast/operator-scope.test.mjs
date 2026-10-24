import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mountOperator, operatorTransport } from '../../cmd/internal/webcast/operator.js';
import { Element } from './dom-fixture.mjs';
import { mountAnnotation } from '../../cmd/internal/webcast/web/annotate.js';
import { createOperatorSketchReceiver, createOperatorSubmissionPreview, mountOperatorSketch } from '../../cmd/internal/webcast/operator-sketch.js';

function harness() {
  const sent = [], timers = new Map();
  let serial = 0;
  const win = {
    setTimeout(fn) { timers.set(++serial, fn); return serial; },
    clearTimeout(id) { timers.delete(id); },
    addEventListener() {}, scrollX: 0, scrollY: 0,
  };
  const report = (message) => {if(message.t==='operator')sent.push(message.request);};
  const expire = () => { const [id, fn] = timers.entries().next().value; timers.delete(id); fn(); };
  return { sent, timers, win, report, expire };
}
const note = (draft) => ({ draft, color: 0, text: draft });

for (const replacement of [0, 2, 'ended']) {
  test(`scope ${replacement} retires pending requests without accepting or resending writes`, async () => {
    const h = harness(), transport = operatorTransport(h.report, h.win);
  transport.setAnnotationGeneration('1');
    transport.setScope(1);
    const old = transport.mutate('save-note', note('old'));
    const prepare = transport.ask('prepare-note', { seg: 1 });
    transport.setScope(replacement === 'ended' ? 1 : replacement, replacement === 'ended');
    assert.equal(h.timers.size, 0);
    assert.equal((await old).status, 409);
    assert.match((await prepare).error, /document|stale/i);
    assert.equal(h.sent.length, 2, 'retirement submitted a write');
    transport.setScope(1);
    const fresh = transport.mutate('save-note', note('fresh'));
    const freshID = h.sent.at(-1).request;
    assert.equal(h.sent.at(-1).op, 'save-note');
    assert.notEqual(freshID, h.sent[0].request);
    transport.reply({ request: h.sent[0].request, outcome: 'accepted' });
    assert.equal(h.timers.size, 1, 'old reply settled the fresh request');
    transport.reply({ request: freshID, outcome: 'accepted' });
    assert.equal((await fresh).status, 204);
  });
}

test('retired accepted completion cannot accept a stale caller or clear the new unsettled write', async () => {
  const h = harness(), transport = operatorTransport(h.report, h.win);
  transport.setAnnotationGeneration('1');
  transport.setScope(1);
  const old = transport.mutate('save-note', note('old'));
  transport.reply({ request: h.sent[0].request, outcome: 'accepted' });
  transport.setScope(0);
  transport.setScope(1);
  const fresh = transport.mutate('save-note', note('fresh'));
  assert.equal((await old).status, 409);
  h.expire();
  assert.equal((await fresh).status, 409);
  const retry = transport.mutate('save-note', note('fresh'));
  assert.equal(h.sent.at(-1).op, 'status');
  assert.equal(h.sent.at(-1).request, h.sent[1].request);
  transport.reply({ request: h.sent.at(-1).request, outcome: 'accepted' });
  assert.equal((await retry).status, 204);
});

function mounted() {
  const h = harness();
  let options, receive = () => {};
  const node = () => ({ style: {}, appendChild() {} });
  mountOperator({ host: node(), root: node(), report: h.report }, {
    annotate: { mountAnnotation(doc, win, view, opts) {
      options = opts;
      view.onRecord((...args) => receive(...args));
      return { strokes:{clear(){}}, setAnnotationAllowed() {}, setMode() {} };
    } }, palette: [], css: '',
  }, { createElement: node }, h.win);
  h.win.__couchcastAnnotationAllowed(true, '1');
  const snapshot = (revision, doc, extra = {}) => h.win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},
    revision, doc, seg: revision, t: revision, notes: [], strokes: [], ...extra,
  });
  return { ...h, options, snapshot, onView(fn) { receive = fn; } };
}

test('adapter retires a lost save reply on switch and return before delivering the new view', async () => {
  const h = mounted();
  h.snapshot(1, 10);
  const old = h.options.post('note', note('old'));
  h.expire();
  assert.equal((await old).status, 409);
  h.snapshot(2, 0);
  let fresh;
  h.onView(() => { fresh = h.options.post('note', note('fresh')); });
  h.snapshot(3, 10);
  assert.equal(h.sent.at(-1).op, 'save-note', 'fresh document retried the dead operation');
  assert.notEqual(h.sent.at(-1).request, h.sent[0].request);
  h.win.__couchcastOperatorReply({ request: h.sent[0].request, outcome: 'accepted' });
  assert.equal(h.timers.size, 1);
  h.win.__couchcastOperatorReply({ request: h.sent.at(-1).request, outcome: 'accepted' });
  assert.equal((await fresh).status, 204);
});

test('adapter retains reconciliation through pause, driving and rejected older snapshots', async () => {
  const h = mounted();
  h.snapshot(1, 10);
  const old = h.options.post('note', note('old'));
  h.expire();
  assert.equal((await old).status, 409);
  h.snapshot(2, 10, { paused: true });
  h.snapshot(3, 10, { driver: 'Ana' });
  h.snapshot(2, 0);
  h.snapshot(4, 10);
  const retry = h.options.post('note', note('old'));
  assert.equal(h.sent.at(-1).op, 'status');
  assert.equal(h.sent.at(-1).request, h.sent[0].request);
  h.win.__couchcastOperatorReply({ request: h.sent[0].request, outcome: 'accepted' });
  assert.equal((await retry).status, 204);
});

test('adapter end retires requests even when the document number is retained', async () => {
  const h = mounted();
  h.snapshot(1, 10);
  const old = h.options.post('note', note('old'));
  h.snapshot(2, 10, { ended: true });
  assert.equal(h.timers.size, 0);
  assert.equal((await old).status, 409);
});

test('operator edit adapter preserves uncertain edit identity and submits only edit fields',async()=>{
 const h=mounted();h.snapshot(1,10);
 const body={seg:1,id:'note',text:'revised'};
 const first=h.options.post('note-edit',body);
 assert.equal(h.sent.at(-1).op,'edit-note');assert.deepEqual(h.sent.at(-1).body,body);
 const id=h.sent.at(-1).request;h.expire();await first;
 const retry=h.options.post('note-edit',body);assert.equal(h.sent.at(-1).op,'status');assert.equal(h.sent.at(-1).request,id);
 h.win.__couchcastOperatorReply({request:id,outcome:'accepted'});assert.equal((await retry).status,204);
});

test('operator shared sketch and annotation record subscribers coexist',()=>{
 const h=harness();const counts=[0,0];const node=()=>({style:{},appendChild(){}});
 mountOperator({host:node(),root:node(),report:h.report},{
  annotate:{mountAnnotation(doc,win,view){view.onRecord(()=>counts[0]++);return {strokes:{clear(){}},setAnnotationAllowed(){},setMode(){}};}},
  palette:[],css:'',sketch:{css:'',shellURL:'data:text/html,private',mount({view}){view.onRecord(()=>counts[1]++);return {tool:{}};}},
 },{createElement:node},h.win);
 h.win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},revision:1,doc:1,seg:1,t:1,notes:[],strokes:[]});
 assert.deepEqual(counts,[1,1]);
});

// Compose the real operator, annotation placement/list/pins and retained sketch
// receiver. Only the existing DOM fixture's geometry/canvas surface is supplied;
// source snapshots and private sketch delivery use the production adapters.
function feedbackRoundtrip(text,{actualSketch=false}={}) {
  const frames=[],sent=[],events=new Map(),visibilityNotices=[],previewStatuses=new Map();
  const win={innerWidth:800,innerHeight:600,scrollX:0,scrollY:0,location:{href:'https://example.test/a'},getComputedStyle:n=>n.style,
    crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:fn=>frames.push(fn),
    addEventListener(event,fn){if(!events.has(event))events.set(event,[]);events.get(event).push(fn);},removeEventListener(){},cancelAnimationFrame(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
  const doc={nodeType:9,childNodes:[],defaultView:win,activeElement:null};win.document=doc;
  class E extends Element {
    constructor(tag){super(tag);this.nodeType=1;this.localName=tag;this.ownerDocument=doc;this.attrs={};this.rect={left:10,top:20,width:100,height:50};}
    get childNodes(){return this.children;}
    append(...nodes){for(const n of nodes){n.parentNode=this;super.append(n);}}
    appendChild(n){this.append(n);}
    remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(n=>n!==this);this.parentNode=null;}
    getAttribute(k){return this.attrs[k]||'';}
    getBoundingClientRect(){return this.className==='operator-box'?{left:0,top:0,width:800,height:600}:this.rect;}
    getClientRects(){return [this.rect];}
    getRootNode(){let n=this;while(n.parentNode)n=n.parentNode;return n;}
    get isConnected(){return this.getRootNode()===doc;}
    focus(){doc.activeElement=this;}
    getContext(){this.paint ||= [];return {setTransform(){},clearRect:()=>{this.paint=[];},beginPath(){},stroke(){},moveTo(){},lineTo(){},closePath(){},arc:(x,y,r)=>this.paint.push({x,y,r}),fill(){},save(){},restore(){}};}
  }
  doc.createElement=tag=>new E(tag);
  const main=new E('main'),anchor=new E('button'),host=new E('couchcast-control'),root=new E('shadow');
  main.parentNode=doc;host.parentNode=doc;doc.childNodes.push(main,host);main.append(anchor);doc.body=main;
  anchor.append({nodeType:3,nodeValue:text,ownerDocument:doc});
  win.__couchcastPath=node=>node===anchor?[0,0]:null;
  const note={id:'unsent-note',text:'Keep this annotation reachable',author:{id:'operator',name:'Operator',color:'#112233'},
    at:{node:7,tag:'button',text,path:['main','button'],route:win.location.href,box:[10,20,100,50]},region:[20,30,40,20]};
  let view,receiver,preview;
  const ui=mountOperator({host,root,report:m=>{if(m.t==='operator')sent.push(m);if(m.t==='feedback-view')visibilityNotices.push(m);}},{annotate:{mountAnnotation},palette:['#112233'],css:'',sketch:{mount(options){
    view=options.view;
    if(actualSketch){const mounted=mountOperatorSketch(options,doc,win);receiver=mounted.receiver;preview={publish:value=>win.__couchcastSketchSubmissionPreview(value)};return mounted;}
    receiver=createOperatorSketchReceiver(()=>{});preview=createOperatorSubmissionPreview({root,receiver,setStatus:(source,text)=>previewStatuses.set(source,text)},doc);
    return {tool:{},receiver,captureBusy:()=>preview.captureBusy()};
  }}},doc,win);
  const digest='a'.repeat(64),mask={scope:'cast',revision:'9',sketches:[{id:'submitted-shape',next_sequence:'5',digest}],notes:[]};
  const shapes=['unsent-shape','submitted-shape'].map(id=>({begin:{id,author:{id:'operator',color:'#112233'},doc:1,tab:'A',vp:{w:800,h:600,dpr:1}},
    next_sequence:'5',digest,phase:'complete',label:`Retained ${id}`,strokes:[{begin:{id:'stroke',width:4,at:note.at},pts:[[20,30,.1],[40,40,.9]],visible:true,sealed:true,truncated:false}]}));
  const bytes=Buffer.from(JSON.stringify({bootstrap:{epoch:'cast',seq:7,display:{sketches:shapes}}}));
  receiver.receive({phase:'begin',handle:'history',epoch:'cast',total:bytes.length});
  receiver.receive({phase:'chunk',handle:'history',epoch:'cast',total:bytes.length,offset:0,base64:bytes.toString('base64')});
  const previewValue={host:'A',generation:'1',revision:'1',operation:'1',stamp:{board:'1',delivered:'1'},items:[{recording_id:'rec',id:'unsent-shape',next_sequence:5,digest}],notes:[{id:note.id,revision:'0'}],remaining:0,cursor:{epoch:'cast',seq:7}};
  const previewAck=preview.publish(previewValue);
  const flush=()=>{let budget=30;while(frames.length){assert.ok(budget-->0,'annotation redraw did not settle');frames.shift()();}};
  const snapshot=(revision,document,url,notes)=>{win.location.href=url;win.__couchcastOperatorView({revision,doc:document,seg:document,t:revision,notes,strokes:[],feedback_mask:structuredClone(mask),
    ...(actualSketch?{feedback_context:{recording_id:'rec',t:revision,record_index:revision*10,doc:document,tab:url.endsWith('/a')?'A':'B',url,title:'Page',vp:{w:800,h:600,dpr:1},scroll:[0,0]},note_revisions:Object.fromEntries(notes.map(n=>[n.id,'0']))}:{})});flush();};
  const rows=()=>Array.from(root.querySelector('.feedback-rows').children).filter(row=>row.dataset.key.startsWith('note:'));
  const pin=()=>root.querySelector('.pins').children[0];
  const settle=async()=>{for(let i=0;i<20;i++){flush();await Promise.resolve();}};
  return {win,root,anchor,note,ui,view,receiver,preview,snapshot,flush,settle,rows,pin,sent,digest,events,visibilityNotices,previewStatuses,previewValue,previewAck};
}

for(const text of ['Save','Distinct original action for the operator roundtrip']) test(`source tab roundtrip keeps unsent feedback reachable and restores only safe placement (${text==='Save'?'common':'distinct'})`,()=>{
  const h=feedbackRoundtrip(text),original=structuredClone(h.note);
  h.snapshot(1,1,'https://example.test/a',[h.note]);
  assert.equal(h.win.__couchcastOperatorOriginal(1,7,h.anchor),true);h.flush();
  assert.equal(h.rows().length,1);assert.equal(h.pin().hidden,false);assert.equal(h.rows()[0].dataset.placement,'visible');
  const history=structuredClone(h.receiver.paths()),metadata=structuredClone(h.receiver.metadata('unsent-shape'));
  assert.equal(h.root.querySelector('.sketch-submission-cards'),undefined,'preview must not paint a second feedback surface');
  assert.equal(h.previewStatuses.get('preview'),'Pending feedback: 1 sketches, 1 annotation');
  assert.equal(h.previewAck.accepted,true);assert.equal(h.preview.captureBusy(),true);
  h.snapshot(2,0,'https://example.test/a',[]);
  assert.equal(h.rows().length,0,'inactive target must retire its page projection');
  assert.equal(h.view.feedbackMask.sketchVisible('submitted-shape',{next_sequence:5,digest:h.digest}),false,'inactive view forgot confirmed history');
  assert.equal(h.win.__couchcastOperatorOriginal(1,7,h.anchor),false,'inactive scope accepted a stale identity binding');
  h.snapshot(3,2,'https://example.test/b',[h.note]);
  assert.equal(h.rows().length,1);assert.match(h.rows()[0].querySelector('.feedback-location').textContent,/not on this view/);assert.equal(h.pin().hidden,true);
  assert.equal(h.view.placementOf(h.receiver.paths()[0]),null,'historical sketch anchor must not project onto another route');
  h.rows()[0].querySelector('.feedback-open').listeners.click[0]({stopPropagation(){}});
  assert.equal(h.ui.dialog.open,true,'unplaced note must still open');
  assert.equal(h.root.querySelector('.note-input').value,h.note.text);h.ui.dialog.close();
  // Returning A is a fresh recording document within the same isolated world;
  // the retained sketch still owns the original connected DOM binding.
  h.snapshot(4,3,'https://example.test/a',[h.note]);
  assert.equal(h.rows().length,1);
  assert.doesNotMatch(h.rows()[0].querySelector('.feedback-location').textContent,/not on this view/);assert.equal(h.pin().hidden,false,'retained original failed to restore the pin');
  assert.ok(h.view.placementOf(h.receiver.paths()[0]),'safe retained sketch anchor did not recover placement');
  assert.deepEqual(h.receiver.paths(),history,'tab switch deleted retained complete sketch history');
  assert.equal(h.receiver.metadata('unsent-shape').phase,'complete');
  assert.deepEqual(h.receiver.metadata('unsent-shape'),metadata,'tab switch changed retained sketch evidence');
  assert.deepEqual(h.preview.publish(h.previewValue),h.previewAck,'tab switch changed the exact validated preview acknowledgement');
  assert.equal(h.previewStatuses.get('preview'),'Pending feedback: 1 sketches, 1 annotation');
  assert.equal(h.view.feedbackMask.sketchVisible('submitted-shape',{next_sequence:5,digest:h.digest}),false);
  assert.equal(h.view.feedbackMask.sketchVisible('unsent-shape',h.receiver.metadata('unsent-shape')),true);
  assert.deepEqual(h.note,original,'placement mutated recorded note evidence');assert.equal(h.sent.length,0,'display roundtrip emitted a mutation');
});

test('trusted retained sketch anchor survives inactive and new recording document without replacement matching',()=>{
  const h=feedbackRoundtrip('Save');
  h.snapshot(1,1,'https://example.test/a',[h.note]);
  assert.equal(h.win.__couchcastOperatorOriginal(1,7,h.anchor),true);h.flush();
  const path=h.receiver.paths()[0];
  h.snapshot(2,0,'https://example.test/a',[]);
  h.snapshot(3,2,'https://example.test/b',[h.note]);
  assert.equal(h.view.placementOf(path),null,'source route must still constrain a trusted original');
  h.snapshot(4,3,'https://example.test/a',[h.note]);
  assert.deepEqual(h.view.placementOf(path),[10,20,100,50],'connected trusted original was forgotten when recorder document1 became3');
  // A real geometry invalidation must recompute using the original reference;
  // merely finding old pixels immediately after a tab switch misses this bug.
  h.anchor.rect.left=110;h.snapshot(5,3,'https://example.test/a',[h.note]);
  assert.deepEqual(h.view.placementOf(path),[110,20,100,50]);
  assert.equal(h.pin().hidden,false);assert.equal(h.rows()[0].dataset.placement,'visible');
});

test('retained binding cannot promote an ambiguous replacement and actual clear releases its proof',()=>{
  const h=feedbackRoundtrip('Save');
  h.snapshot(1,1,'https://example.test/a',[h.note]);
  assert.equal(h.win.__couchcastOperatorOriginal(1,7,h.anchor),true);h.flush();
  const path=h.receiver.paths()[0],parent=h.anchor.parentNode,doc=h.anchor.ownerDocument;
  h.snapshot(2,0,'https://example.test/a',[]);
  h.anchor.remove();
  const replacement=doc.createElement('button');replacement.append({nodeType:3,nodeValue:'Save',ownerDocument:doc});parent.append(replacement);
  h.snapshot(3,3,'https://example.test/a',[h.note]);
  assert.equal(h.win.__couchcastOperatorOriginal(3,7,replacement),false,'historical ID rebound to a same-shape node');
  assert.equal(h.view.placementOf(path),null);
  assert.match(h.rows()[0].querySelector('.feedback-location').textContent,/not on this view/);assert.equal(h.pin().hidden,true);
  // The accepted annotation-clear delta drops the historical anchor index.
  const bytes=Buffer.from(JSON.stringify({batch:{epoch:'cast',from:8,to:8,patches:[{clear:true}]}}));
  h.receiver.receive({phase:'begin',handle:'clear',epoch:'cast',total:bytes.length});
  h.receiver.receive({phase:'chunk',handle:'clear',epoch:'cast',total:bytes.length,offset:0,base64:bytes.toString('base64')});
  h.snapshot(4,3,'https://example.test/a',[]);
  replacement.remove();parent.append(h.anchor);
  assert.equal(h.receiver.paths().length,0);
  assert.equal(h.win.__couchcastOperatorOriginal(3,7,h.anchor),false,'clear left an authority to bind the retired ID');
  assert.equal(h.view.placementOf(path),null,'clear retained an old connected original proof');
});

test('isolated context retirement releases a retained trusted original',()=>{
  const h=feedbackRoundtrip('Save');h.snapshot(1,1,'https://example.test/a',[h.note]);
  assert.equal(h.win.__couchcastOperatorOriginal(1,7,h.anchor),true);h.flush();
  const path=h.receiver.paths()[0];
  assert.ok(h.view.placementOf(path));
  for(const fn of h.events.get('pagehide')||[])fn();
  assert.equal(h.view.placementOf(path),null,'context retirement retained a connected original proof');
});

test('actual console painters publish only completed current note and accepted sketch placements',async()=>{
  const h=feedbackRoundtrip('Save',{actualSketch:true});
  h.snapshot(1,1,'https://example.test/a',[h.note]);
  assert.equal(h.win.__couchcastOperatorOriginal(1,7,h.anchor),true);
  assert.throws(()=>h.win.__couchcastFeedbackVisibility(),/paint|unavailable/i);
  await h.settle();
  const painted=h.win.__couchcastFeedbackVisibility();
  assert.equal(painted.scene.doc,1);assert.equal(painted.scene.tab,'A');assert.equal(painted.mask_scope,'cast');
  assert.deepEqual(painted.notes,[{id:'unsent-note',revision:'0'}]);
  assert.equal(painted.sketches.length,1);assert.equal(painted.sketches[0].id,'unsent-shape');
  assert.deepEqual(painted.sketches[0].paths[0],{id:'stroke',locator:[0,0],box:[10,20,100,50],clip:[0,0,800,600]});
  assert.equal(h.visibilityNotices.at(-1).revision,painted.revision,'completed paint did not wake the host');
  h.anchor.rect.left=110;for(const fn of h.events.get('scroll')||[])fn();
  assert.throws(()=>h.win.__couchcastFeedbackVisibility(),/paint|unavailable/i);await h.settle();
  assert.deepEqual(h.win.__couchcastFeedbackVisibility().sketches[0].paths[0].box,[110,20,100,50]);
  h.rows()[0].querySelector('.feedback-open').listeners.click[0]({stopPropagation(){}});assert.throws(()=>h.win.__couchcastFeedbackVisibility(),/busy|drawing/i);
  h.ui.dialog.close();await h.settle();assert.equal(h.win.__couchcastFeedbackVisibility().notes.length,1);
  h.snapshot(2,0,'https://example.test/a',[]);await h.settle();assert.throws(()=>h.win.__couchcastFeedbackVisibility(),/context|unavailable/i);
});

test('late source placement cannot paint or publish after synchronous geometry invalidation',async()=>{
  const h=feedbackRoundtrip('Save',{actualSketch:true});
  h.snapshot(1,1,'https://example.test/a',[h.note]);h.win.__couchcastOperatorOriginal(1,7,h.anchor);await h.settle();
  const place=h.view.placementOf;let release,held=false;
  h.view.placementOf=(path,options)=>{const target=place(path,options);if(path.sketch&&!held){held=true;return new Promise(resolve=>{release=()=>resolve(target);});}return target;};
  for(const fn of h.events.get('scroll')||[])fn();h.flush();await Promise.resolve();await Promise.resolve();assert.equal(typeof release,'function');
  const observer=h.root.querySelector('.sketch-observer'),before=structuredClone(observer.paint);
  h.anchor.rect.left=210;for(const fn of h.events.get('scroll')||[])fn();
  assert.throws(()=>h.win.__couchcastFeedbackVisibility(),/paint|unavailable/i);
  release();for(let i=0;i<5;i++)await Promise.resolve();
  assert.deepEqual(observer.paint,before,'late old placement painted after invalidation before fresh animation frame');
  await h.settle();assert.deepEqual(h.win.__couchcastFeedbackVisibility().sketches[0].paths[0].box,[210,20,100,50]);
});

test('accepted receiver version changes retire a completed visibility snapshot before repaint',async()=>{
  const h=feedbackRoundtrip('Save',{actualSketch:true});h.snapshot(1,1,'https://example.test/a',[h.note]);h.win.__couchcastOperatorOriginal(1,7,h.anchor);await h.settle();
  const old=h.win.__couchcastFeedbackVisibility();
  const bytes=Buffer.from(JSON.stringify({batch:{epoch:'cast',from:8,to:8,patches:[{sketch:'unsent-shape',header:{phase:'complete',label:'Updated saved label',next_sequence:'6',digest:'b'.repeat(64)}}]}}));
  h.receiver.receive({phase:'begin',handle:'new-version',epoch:'cast',total:bytes.length});
  h.receiver.receive({phase:'chunk',handle:'new-version',epoch:'cast',total:bytes.length,offset:0,base64:bytes.toString('base64')});
  assert.throws(()=>h.win.__couchcastFeedbackVisibility(),/paint|unavailable/i,'new accepted version left the old completed snapshot available');
  await h.settle();const latest=h.win.__couchcastFeedbackVisibility();
  assert.ok(BigInt(latest.revision)>BigInt(old.revision));assert.equal(latest.sketches[0].next_sequence,'6');assert.equal(latest.sketches[0].digest,'b'.repeat(64));
});
