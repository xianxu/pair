import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {mountAnnotation} from '../../cmd/internal/webcast/web/annotate.js';

function mount({annotate = true, operator = false} = {}) {
  const root = new Element('body'), box = new Element('box'); root.append(box);
  const doc = {body:root, createElement:tag=>new Element(tag)};
  const win = {innerWidth:800, innerHeight:600, crypto:{getRandomValues:b=>b.fill(1)}, performance:{now:()=>0}, requestAnimationFrame(){}, addEventListener(){}, setTimeout:()=>1, clearTimeout(){}, clearInterval(){}};
  let receive;
  const view = {model:{seg:1,docSeg:1,notes:[]}, onRecord:fn=>{receive=fn;}, toPage:(x,y)=>[x,y], fromPage:(x,y)=>[x,y]};
  const ui = mountAnnotation(doc,win,view,{root,box,stage:box,palette:['#112233'],annotate,operator,annotationAllowed:false,
    who:{id:()=> 'viewer',name:()=> 'Viewer',viewer:()=>({id:'viewer',name:'Viewer'})},colors:{get:()=>0,set(){}},post:()=>assert.fail('a reason must not grant or post')});
  return {root,ui,view,refresh:()=>receive({k:'view',t:1},view.model),status:root.querySelector('.feedback-status')};
}

test('viewer Annotation OFF visibly explains how to request annotation in Couch',()=>{
  const f=mount();
  assert.equal(f.root.querySelector('.feedback-panel').hidden,false);
  assert.equal(f.root.querySelector('.floating-panel-header').hidden,false);
  assert.equal(f.status.textContent,'Annotation is off. Ask the cast owner to enable viewer annotation in Couch.');
  assert.equal(f.ui.controls.note.disabled,true);
  f.ui.setMode('note'); assert.equal(f.ui.mode,'watch');
  f.ui.setAnnotationAllowed(true);
  assert.equal(f.status.textContent,'');
  assert.equal(f.ui.controls.note.disabled,false);
  f.view.model.paused=true; f.refresh();
  assert.equal(f.status.textContent,'The cast is paused.');
  assert.equal(f.ui.controls.note.disabled,true);
});

test('view-only link explains that an annotating link is needed even when Annotation is granted',()=>{
  const f=mount({annotate:false});
  const reason='This link is read-only. Ask the cast owner for an annotating link.';
  assert.equal(f.status.textContent,reason);
  assert.equal(f.root.querySelector('.feedback-panel').hidden,false);
  assert.equal(f.root.querySelector('.floating-panel-header').hidden,false);
  f.ui.setAnnotationAllowed(true);
  assert.equal(f.status.textContent,reason);
  f.ui.setMode('point'); assert.equal(f.ui.mode,'watch');
  assert.equal(f.ui.controls.pointer.disabled,true);
});

test('viewer grant guidance preserves cleanup and owner driving explanations',()=>{
  const f=mount();
  f.ui.setAnnotationAllowed(false,false,'retry');
  assert.equal(f.status.textContent,'Annotation is off. Waiting for the operator to retry.');
  f.ui.setAnnotationAllowed(false,false,'broken');
  assert.equal(f.status.textContent,'Cast broken. Wait for the operator to end the cast.');
  const owner=mount({operator:true});
  assert.equal(owner.status.textContent,'Annotation is off.');
  owner.ui.setAnnotationAllowed(true);
  owner.view.model.driver='Ana'; owner.refresh();
  assert.equal(owner.status.textContent,'Ana is driving.');
  assert.equal(owner.ui.controls.note.disabled,true);
});
