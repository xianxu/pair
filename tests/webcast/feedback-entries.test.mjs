import test from 'node:test';
import assert from 'node:assert/strict';
import { feedbackEntries } from '../../cmd/internal/webcast/web/feedback-entries.js';

test('defaults to empty projection', () => {
  assert.deepEqual(feedbackEntries(), []);
});

test('projects notes before sketches, preserving source and keeping policy outside helper', () => {
  const note = Object.freeze({id:'same',text:'annotation',author:'other',editable:true});
  const bounds = Object.freeze({x:1,y:2,width:3,height:4});
  const sketch = Object.freeze({sketch:'same',label:'drawing',editable:true,bounds});
  assert.deepEqual(feedbackEntries([note], [sketch]), [
    {key:'note:same',kind:'note',id:'same',text:'annotation',editable:false,source:note},
    {key:'sketch:same',kind:'sketch',id:'same',text:'drawing',editable:true,source:sketch},
  ]);
  const row = feedbackEntries([], [sketch])[0];
  assert.strictEqual(row.source.bounds, bounds);
  assert.equal(Object.hasOwn(row, 'bounds'), false);
});

test('preserves arbitrary string IDs and exact note text without interpreting markup', () => {
  const ids = ['__proto__', 'constructor', 'a:b', '<img onerror=x>', ' ', '雪\n💬', '0'];
  const texts = ['', '<script>x</script>', '  text\n ', '📝', 'line\nline', 'a:b', '0'];
  const notes = ids.map((id,i) => ({id,text:texts[i]}));
  const rows = feedbackEntries(notes);
  assert.deepEqual(rows.map(({id,key,text}) => ({id,key,text})), ids.map((id,i) => ({id,key:'note:'+id,text:texts[i]})));
});

test('skips absent, empty and non-string IDs and keeps first typed key only', () => {
  const notes = [null, undefined, {}, {id:''}, {id:0}, {id:false}, {id:Symbol('id')}, {id:{}}, {id:'x',text:'first'}, {id:'x',text:'second'}, {id:'y',text:'third'}];
  const sketches = [null, {}, {sketch:''}, {sketch:1}, {sketch:'x',label:'first'}, {sketch:'x',label:'second'}];
  const rows = feedbackEntries(notes, sketches);
  assert.deepEqual(rows.map(row => [row.key,row.text]), [['note:x','first'],['note:y','third'],['sketch:x','first']]);
});

test('notes accept string text only; sketch labels convert to text and editability is strict', () => {
  const notes = [undefined,null,0,false,{},['text'],Symbol('text')].map((text,i) => ({id:String(i),text}));
  assert.deepEqual(feedbackEntries(notes).map(row => row.text), notes.map(() => ''));
  const labels = [undefined,null,0,false,42,['a','b'],{toString:()=> 'custom'},Symbol('label')];
  const flags = [true,false,1,'true',null,undefined,{},true];
  const sketches = labels.map((label,i) => ({sketch:String(i),label,editable:flags[i]}));
  const rows = feedbackEntries([], sketches);
  assert.deepEqual(rows.map(row => row.text), ['','','0','false','42','a,b','custom','Symbol(label)']);
  assert.deepEqual(rows.map(row => row.editable), [true,false,false,false,false,false,false,true]);
});

test('does not mutate frozen inputs or reuse a previous projection', () => {
  const note = Object.freeze({id:'n',text:'original',metadata:Object.freeze({value:1})});
  const sketch = Object.freeze({sketch:'s',label:'drawing',editable:false});
  const notes = Object.freeze([note]);
  const sketches = Object.freeze([sketch]);
  const first = feedbackEntries(notes, sketches);
  const second = feedbackEntries(notes, sketches);
  assert.notStrictEqual(first, second);
  assert.notStrictEqual(first[0], second[0]);
  assert.strictEqual(first[0].source, note);
  assert.strictEqual(first[1].source, sketch);
  first[0].text = 'projection only';
  assert.equal(note.text, 'original');
  assert.equal(second[0].text, 'original');
});

function measuredProjection(size) {
  const counts = {iterations:0,yields:0,reads:0};
  function input(kind) {
    const rows = Array.from({length:size}, (_,i) => new Proxy(kind === 'note' ? {id:String(i),text:'n'} : {sketch:String(i),label:'s',editable:true}, {
      get(target,key) { counts.reads++; return target[key]; },
    }));
    return {
      [Symbol.iterator]() {
        counts.iterations++;
        let index = 0;
        return {next() { if(index === rows.length) return {done:true}; counts.yields++; return {done:false,value:rows[index++]}; }};
      },
    };
  }
  assert.equal(feedbackEntries(input('note'), input('sketch')).length, size*2);
  return counts;
}

test('consumes each input once with linear property reads at 32 and 64 entries', () => {
  const small = measuredProjection(32);
  const large = measuredProjection(64);
  assert.equal(small.iterations, 2);
  assert.equal(large.iterations, 2);
  assert.equal(small.yields, 64);
  assert.equal(large.yields, 128);
  assert.ok(small.reads > 0);
  assert.ok(small.reads <= 32*8);
  assert.equal(large.reads, small.reads*2);
});
