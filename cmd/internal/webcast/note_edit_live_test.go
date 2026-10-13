package webcast

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNoteEditLiveViewerOwnership(t *testing.T) {
	a, b := headlessPage(t), headlessPage(t)
	c := newBoardCast(t)
	base, locks := lockedNoteEditServer(t, c)
	for _, p := range []*chromePage{a, b} {
		p.call(t, p.session, "Page.navigate", map[string]any{"url": base + "/tok/"}, nil)
		waitEval(t, p, "lock page", `String(!!document.querySelector('script[src="lock.js"]'))`, "true")
		p.eval(t, fmt.Sprintf(`(async()=>{const r=await fetch('unlock',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({code:%q})});if(!r.ok)throw Error('unlock '+r.status);return 'ok'})()`, locks.LockFor("tok").Code()))
		p.call(t, p.session, "Page.reload", map[string]any{}, nil)
		waitEval(t, p, "annotation mount", `String(!!window.couchViewer && !!document.querySelector('.annotate-rail') && !document.querySelector('.annotate-rail').hidden)`, "true")
	}
	body, _ := json.Marshal(map[string]any{"viewer": map[string]string{"id": "same-public-id", "name": "Ana"}, "seg": 1, "node": c.p.ID, "box": []int{1, 2, 30, 40}, "region": []int{1, 2, 30, 40}, "color": 0, "text": "Original note"})
	if got := a.evalString(t, fmt.Sprintf(`(async()=>String((await fetch('note',{method:'POST',headers:{'Content-Type':'application/json','X-Couch-Annotation-Generation':'1'},body:%q})).status))()`, string(body))); got != "204" {
		t.Fatalf("create %s", got)
	}
	for _, p := range []*chromePage{a, b} {
		waitEval(t, p, "note pin", `String(!!document.querySelector('.pin .chip'))`, "true")
	}
	b.eval(t, `document.querySelector('.pin .chip').click(); 'ok'`)
	if got := b.evalString(t, `String(document.querySelector('.note-dialog').classList.contains('view'))`); got != "true" {
		t.Fatal("other session opened editor", got)
	}
	a.eval(t, `document.querySelector('.pin .chip').click(); 'ok'`)
	if got := a.evalString(t, `document.querySelector('.note-input').value`); got != "Original note" {
		t.Fatal("own editor not populated", got)
	}
	a.eval(t, `(()=>{const e=document.querySelector('.note-input');e.value='discard';e.dispatchEvent(new Event('input',{bubbles:true}));[...document.querySelectorAll('.note-buttons button')].find(b=>b.textContent==='Cancel').click();return 'ok'})()`)
	if c.bd.Live()[0].Text != "Original note" {
		t.Fatal("cancel wrote edit")
	}
	a.eval(t, `document.querySelector('.pin .chip').click(); 'ok'`)
	a.eval(t, `(()=>{const e=document.querySelector('.note-input');e.value='Changed note';e.dispatchEvent(new Event('input',{bubbles:true}));document.querySelector('.note-buttons').lastElementChild.click();return 'ok'})()`)
	waitEval(t, a, "edit acknowledgment", `String(document.querySelector('.note-dialog').classList.contains('hidden'))`, "true")
	waitEval(t, b, "edited readonly text", `document.querySelector('.note-text').textContent`, "Changed note (edited)")
	if got := c.bd.Live(); len(got) != 1 || got[0].Text != "Changed note" || got[0].Author.Name != "Ana" {
		t.Fatalf("notes %+v", got)
	}
}
