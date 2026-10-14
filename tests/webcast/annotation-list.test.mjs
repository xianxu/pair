import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {openNotes,noteList,pinLayer} from '../../cmd/internal/webcast/web/annotate.js';
const doc={createElement:tag=>new Element(tag)};
const note={id:'n1',text:'Earlier route',author:{name:'Ana',color:'#112233'},region:[0,0,10,10]};
test('all open notes remain listed across documents, unavailable placement is explicit and latest text opens',()=>{
 const model={docSeg:8,notes:[{seg:1,state:'open',note},{seg:9,state:'resolved',note:{id:'done'}}]};
 assert.deepEqual(openNotes(model),[note]);
 const root=new Element('div'),opened=[];const list=noteList(doc,root,n=>opened.push(n));
 list.draw(openNotes(model),()=>null);
 const button=root.children[0];assert.match(button.textContent,/not on this view/);
 const edited={...note,text:'Edited latest text'};
 list.draw([edited],()=>[0,0,10,10]);assert.equal(root.children[0],button);assert.doesNotMatch(button.textContent,/not on this view/);
 button.listeners.click[0]({stopPropagation(){}});assert.equal(opened[0],edited);
});
test('live pins never use historical regions when placement fails and latest record is opened',()=>{
 const root=new Element('div'),opened=[];const pins=pinLayer(doc,root,n=>opened.push(n));
 pins.draw([note],()=>null,()=>assert.fail('must hide'),{placementOf:()=>null});
 assert.equal(root.children[0].hidden,true);
 const edited={...note,text:'Newest'};
 pins.draw([edited],()=>null,()=>{}, {placementOf:()=>[1,1,10,10]});
 assert.equal(root.children[0].hidden,false);
 root.children[0].children[0].listeners.click[0]({stopPropagation(){}});assert.equal(opened[0],edited);
});
