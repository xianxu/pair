import test from 'node:test';
import assert from 'node:assert/strict';
import {Model} from '../../cmd/internal/webcast/web/model.js';
import {copyPageLoad,samePageLoad,feedbackReady,onFeedbackPage} from '../../cmd/internal/webcast/web/page-load.js';

const a={target:'A',loader:'opaque:loader',root:'9223372036854775807'},b={...a,target:'B'};
test('page load owns exact opaque strings and canonical positive int64 root',()=>{
 assert.deepEqual(copyPageLoad(a),a);assert.notEqual(copyPageLoad(a),a);
 assert.equal(samePageLoad(a,{...a}),true);assert.equal(samePageLoad(null,null),false);
 assert.equal(samePageLoad(a,b),false);assert.equal(samePageLoad(a,{...a,root:'9007199254740993'}),false);
 for(const root of ['',0,1,'0','01','+1','-1',' 1','1 ','1e3','1.0','9223372036854775808'])assert.equal(copyPageLoad({...a,root}),null,JSON.stringify(root));
 for(const value of [null,{}, {...a,target:''},{...a,loader:'\ud800'}])assert.equal(copyPageLoad(value),null);
 assert.ok(copyPageLoad({target:'x'.repeat(4096),loader:'😀'.repeat(4096),root:'1'}));
});

test('Model retains load membership and history through pause, same-load return, reload, and DOM reset',()=>{
 const m=new Model();m.apply({k:'head',t:0,format:'couch-webcast',v:1,id:'loads',started:'now'});
 let t=1;
 const segment=(seg,load,why='tab')=>{m.apply({k:'seg',t:t++,seg,why,tab:{id:load.target,url:'https://same/'},vp:{w:100,h:100},page_load:load});m.apply({k:'doc',t:t++,doc:seg*10,tree:{id:seg*10,y:'doc'}});};
 const note=id=>m.apply({k:'note',t:t++,id,author:{id:'operator'},region:[1,1,4,4],text:id});
 segment(1,a);note('A');m.apply({k:'stop',t:t++,why:'pause'});
 assert.equal(feedbackReady(m),true);assert.equal(onFeedbackPage(m,1),true);
 segment(2,a,'resume');assert.equal(onFeedbackPage(m,1),true);
 segment(3,b);note('B');assert.equal(onFeedbackPage(m,1),false);assert.equal(onFeedbackPage(m,3),true);
 segment(4,a);assert.equal(onFeedbackPage(m,1),true);assert.equal(onFeedbackPage(m,3),false);
 segment(5,{...a,loader:'reloaded'});assert.equal(onFeedbackPage(m,1),false);
 assert.deepEqual(m.notes.map(x=>x.note.id),['A','B']);
 const copy=m.pageLoadAt(1);copy.loader='mutated';assert.equal(m.pageLoadAt(1).loader,a.loader);
 const current=m.currentPageLoad();current.root='1';assert.equal(m.currentPageLoad().root,a.root);
 m.resetDOM();assert.equal(m.onPageLoad(1,a),true);
 m.apply({k:'seg',t:t++,seg:6,why:'navigate',tab:{id:'A'},vp:{w:100,h:100}});
 assert.equal(feedbackReady(m),false);assert.equal(onFeedbackPage(m,1),false);assert.equal(m.pageLoadAt(6),null);
});

test('invalid segment identity refuses atomically and runtime pending scope fails closed',()=>{
 const m=new Model();m.apply({k:'head',t:0,format:'couch-webcast',v:1,id:'loads',started:'now'});
 for(const page_load of [null,{...a,root:1},{...a,target:'wrong'}]){
  assert.throws(()=>m.apply({k:'seg',t:1,seg:1,why:'live',tab:{id:'A'},vp:{w:100,h:100},page_load}));
  assert.equal(m.seg,0);assert.equal(m.pageLoadAt(1),null);
 }
 for(const model of [{page_load_ready:false,page_load:a},{page_load_ready:true,page_load:null}]){
  assert.equal(feedbackReady(model),false);assert.equal(onFeedbackPage(model,1),false);
 }
 assert.equal(feedbackReady({page_load_ready:true,page_load:a,paused:true}),true);
});
