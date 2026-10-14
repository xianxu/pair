package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
)

func TestNoteEditLiveViewerOwnership(t *testing.T) {
	a, b := headlessPage(t), headlessPage(t)
	c := newBoardCast(t)
	drafts := replayNoteDraftService(t, c.bd, c.w)
	base, locks := lockedCapturedNoteEditServer(t, c, drafts)
	for _, p := range []*chromePage{a, b} {
		p.call(t, p.session, "Page.navigate", map[string]any{"url": base + "/tok/"}, nil)
		waitEval(t, p, "lock page", `String(!!document.querySelector('script[src="lock.js"]'))`, "true")
		p.eval(t, fmt.Sprintf(`(async()=>{const r=await fetch('unlock',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({code:%q})});if(!r.ok)throw Error('unlock '+r.status);return 'ok'})()`, locks.LockFor("tok").Code()))
		p.call(t, p.session, "Page.reload", map[string]any{}, nil)
		waitEval(t, p, "annotation mount", `String(!!window.couchViewer && !!document.querySelector('.annotate-rail') && !document.querySelector('.annotate-rail').hidden)`, "true")
	}
	prepare, _ := json.Marshal(map[string]any{"request": "draw1", "seg": 1, "node": c.p.ID, "box": []int{1, 2, 30, 40}, "region": []int{1, 2, 30, 40}})
	if got := a.evalString(t, fmt.Sprintf(`(async()=>{
      const post=async(route,body)=>{const r=await fetch(route,{method:'POST',headers:{'Content-Type':'application/json','X-Couch-Annotation-Generation':'1'},body:JSON.stringify(body)});if(!r.ok)throw Error(route+' '+r.status+' '+await r.text());return r.json()};
      let status=await post('prepare-note',JSON.parse(%q));
      const deadline=Date.now()+10000;
      while(status.state==='pending'&&Date.now()<deadline){await new Promise(r=>setTimeout(r,20));status=await post('draft-status',{request:'draw1'})}
      if(status.state!=='ready')throw Error(JSON.stringify(status));
      const draft=status.draft;
      status=await post('save-note',{request:'save1',draft,viewer:{id:'same-public-id',name:'Ana'},color:0,text:'Original note'});
      while(status.state==='pending'&&Date.now()<deadline){await new Promise(r=>setTimeout(r,20));status=await post('draft-status',{request:'save1',draft})}
      return status.state;
    })()`, string(prepare))); got != "saved" {
		t.Fatalf("capture/save %s", got)
	}
	original, ok := c.bd.ObservedNote(c.bd.Live()[0].ID)
	if !ok || original.Capture == "" {
		t.Fatal("viewer note missing captured evidence")
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
	updated, ok := c.bd.ObservedNote(original.ID)
	if !ok || updated.Capture != original.Capture || !reflect.DeepEqual(updated.At, original.At) || !reflect.DeepEqual(updated.Region, original.Region) {
		t.Fatalf("edit replaced frozen evidence: before %+v after %+v", original, updated)
	}
	if got := c.bd.Live(); len(got) != 1 || got[0].Text != "Changed note" || got[0].Author.Name != "Ana" {
		t.Fatalf("notes %+v", got)
	}
}

func lockedCapturedNoteEditServer(t *testing.T, c *boardCast, drafts *notedraft.Service) (string, *broadcast.LinkLocks) {
	t.Helper()
	locks := &broadcast.LinkLocks{}
	for _, token := range []string{"view", "tok"} {
		if err := locks.Mint(token, token, 4, nil); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(locks.End)
	server := NewServer("view", NewFakeBrowser(), time.Second, c.hub, nil).WithBoard(c.bd).WithPointer("tok").WithLocks(locks).WithNoteDrafts(drafts, func(r *http.Request) string { return broadcast.NoteOwnerFromContext(r.Context()) })
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: time.Second})
	mount, err := host.Mount(context.Background(), server, "tok/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mount.Unmount)
	return mount.Base(), locks
}
