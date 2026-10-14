import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {chooseAnchor, scoreAnchor, projectRegion} from '../../cmd/internal/webcast/web/annotation-anchor.js';
const cases=JSON.parse(readFileSync(new URL('../../cmd/internal/webcast/notes/testdata/anchor-candidates.json',import.meta.url),'utf8'));
for(const c of cases)test(c.name,()=>assert.equal(chooseAnchor(c.anchor,c.candidates,{truncated:c.truncated,route:c.route})?.key||'',c.want));
test('same-node evidence rejects conflicts',()=>assert.equal(scoreAnchor(cases[0].anchor,{...cases[0].candidates[0],text:'Delete report'}),-1));
test('projection follows translation and resize',()=>assert.deepEqual(projectRegion([20,30,40,20],[10,10,100,50],[50,100,200,100]),[70,140,80,40]));
test('projection rejects absent, nonfinite and degenerate geometry',()=>{
 for(const b of [null,[0,0,0,10],[0,0,10,-1],[Infinity,0,10,10]])assert.equal(projectRegion([1,1,2,2],b,[0,0,10,10]),null);
 assert.equal(projectRegion([1,1,NaN,2],[0,0,10,10],[0,0,10,10]),null);
});
