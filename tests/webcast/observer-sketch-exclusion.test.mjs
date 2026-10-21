import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const observer=fs.readFileSync(new URL('../../cmd/internal/webcast/recorder/observer.js',import.meta.url),'utf8');
const prefix='data:text/html;couch-sketch=1,';
function bootstrap(frame){
 const sentinel=new Error('observer reached page APIs');
 const scope={window:frame,location:frame.location};
 Object.defineProperty(scope,'document',{get(){throw sentinel;}});
 try{vm.runInNewContext(observer,scope);}catch(e){if(e!==sentinel && !scope.__couchcastRecInstalled)throw e;}
 return scope;
}
function frame(url,parent){const f={location:{href:url}};f.parent=parent||f;return f;}
test('private shell observer returns before installation or page APIs',()=>{
 assert.equal(bootstrap(frame(prefix+'private')).__couchcastRecInstalled,undefined);
});
test('nested private frames skip observer through accessible ancestor',()=>{
 const shell=frame(prefix+'private');
 assert.equal(bootstrap(frame('about:blank',frame('about:blank',shell))).__couchcastRecInstalled,undefined);
});
test('normal page and similar nonreserved URL still install observer',()=>{
 for(const url of ['https://example.test/','data:text/html;couch-sketch=10,normal'])
  assert.equal(bootstrap(frame(url)).__couchcastRecInstalled,true);
});
test('cross origin inaccessible parent does not exclude ordinary page',()=>{
 const parent={};Object.defineProperty(parent,'location',{get(){throw new Error('SecurityError');}});
 assert.equal(bootstrap(frame('https://example.test/',parent)).__couchcastRecInstalled,true);
});
