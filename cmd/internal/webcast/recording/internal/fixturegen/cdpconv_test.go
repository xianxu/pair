package main

import (
	"encoding/json"
	"strings"
	"testing"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// feed runs raw lines (JSON objects, t relative to 1000) through a
// converter and returns it.
func feed(t *testing.T, lines ...string) *converter {
	t.Helper()
	c := newConverter("t", 1000)
	for i, l := range lines {
		var r raw
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		r.T += 1000
		r.Extra = []byte(l)
		if err := c.event(r); err != nil && err != errSkip {
			t.Fatalf("line %d: %v", i, err)
		}
	}
	if len(c.dropped) > 0 {
		t.Fatalf("dropped %s: %v", c.summary(), c.examples)
	}
	return c
}

// doc is a CDP document: <html><head/><body>body</body></html>, nodeIds
// from base, backendNodeIds from 100.
func doc(url, body string) string {
	return `{"nodeId":1,"backendNodeId":100,"nodeType":9,"nodeName":"#document","documentURL":"` + url + `","children":[` +
		`{"nodeId":2,"backendNodeId":101,"nodeType":1,"nodeName":"HTML","localName":"html","attributes":[],"children":[` +
		`{"nodeId":3,"backendNodeId":102,"nodeType":1,"nodeName":"HEAD","localName":"head","attributes":[]},` +
		`{"nodeId":4,"backendNodeId":103,"nodeType":1,"nodeName":"BODY","localName":"body","attributes":[],"children":[` + body + `]}]}]}`
}

func kinds(c *converter) string {
	var ks []string
	for _, r := range c.out {
		ks = append(ks, r.Kind())
	}
	return strings.Join(ks, ",")
}

func TestSnapshotsMakeSegmentsAndKeepIDs(t *testing.T) {
	div := `{"nodeId":5,"backendNodeId":104,"nodeType":1,"nodeName":"DIV","localName":"div","attributes":["id","a"]}`
	// The same document snapshotted again under new nodeIds keeps its ids.
	div2 := strings.Replace(div, `"nodeId":5`, `"nodeId":55`, 1)
	c := feed(t,
		`{"t":0,"s":"","type":"snapshot","frameId":"F","root":`+doc("about:blank", "")+`}`,
		`{"t":5,"s":"","type":"snapshot","frameId":"F","root":`+strings.Replace(doc("http://x/", div), `"backendNodeId":100`, `"backendNodeId":200`, 1)+`}`,
		`{"t":6,"s":"","type":"snapshot","frameId":"F","root":`+strings.Replace(doc("http://x/", div2), `"backendNodeId":100`, `"backendNodeId":200`, 1)+`}`,
		`{"t":7,"s":"","type":"attributeModified","nodeId":55,"name":"class","value":"on"}`,
	)
	if got := kinds(c); got != "head,seg,doc,seg,doc,kids,attr" {
		t.Fatalf("records %s", got)
	}
	if s := c.out[3].(*rec.Seg); s.Why != rec.SegNavigate || s.Seg != 2 {
		t.Fatalf("the second document is a navigation: %+v", s)
	}
	d := c.st.Top.Kids[0].Kids[1].Kids[0]
	if d.A["class"] != "on" || d.ID != c.out[4].(*rec.Doc).Tree.C[0].C[1].C[0].ID {
		t.Fatal("the re-snapshotted div must keep its id")
	}
}

func TestUserAgentShadowAndPseudoElementsAreDropped(t *testing.T) {
	input := `{"nodeId":5,"backendNodeId":104,"nodeType":1,"nodeName":"INPUT","localName":"input","attributes":[],` +
		`"shadowRoots":[{"nodeId":6,"backendNodeId":105,"nodeType":11,"nodeName":"#document-fragment","shadowRootType":"user-agent","children":[` +
		`{"nodeId":7,"backendNodeId":106,"nodeType":1,"nodeName":"DIV","localName":"div","attributes":[]}]}]}`
	c := feed(t,
		`{"t":0,"s":"","type":"snapshot","frameId":"F","root":`+doc("http://x/", input)+`}`,
		`{"t":1,"s":"","type":"childNodeInserted","parentNodeId":7,"previousNodeId":0,"node":{"nodeId":8,"backendNodeId":107,"nodeType":3,"nodeName":"#text","nodeValue":"x"}}`,
		`{"t":2,"s":"","type":"attributeModified","nodeId":7,"name":"style","value":"x"}`,
		`{"t":3,"s":"","type":"value","nodeId":5,"v":"typed"}`,
	)
	if got := kinds(c); got != "head,seg,doc,input" {
		t.Fatalf("records %s (events inside a UA shadow tree must not appear)", got)
	}
	if c.st.Top.Kids[0].Kids[1].Kids[0].Shadow != nil {
		t.Fatal("a user-agent shadow root is not page content")
	}
}

func TestOutOfProcessFrameAttachesWhenItsOwnerAppears(t *testing.T) {
	frameDoc := `{"nodeId":1,"backendNodeId":10,"nodeType":9,"nodeName":"#document","documentURL":"http://y/","children":[` +
		`{"nodeId":2,"backendNodeId":11,"nodeType":1,"nodeName":"HTML","localName":"html","attributes":[]}]}`
	c := feed(t,
		`{"t":0,"s":"","type":"snapshot","frameId":"F","root":`+doc("http://x/", "")+`}`,
		// The frame's session snapshots before its iframe is in the page.
		`{"t":1,"s":"S2","type":"snapshot","frameId":"G","root":`+frameDoc+`}`,
		`{"t":2,"s":"S2","type":"attributeModified","nodeId":2,"name":"lang","value":"en"}`,
		`{"t":3,"s":"","type":"childNodeInserted","parentNodeId":4,"previousNodeId":0,"node":{"nodeId":9,"backendNodeId":120,"nodeType":1,"nodeName":"IFRAME","localName":"iframe","attributes":["src","http://y/"],"frameId":"G"}}`,
		`{"t":4,"s":"S2","type":"attributeModified","nodeId":2,"name":"dir","value":"ltr"}`,
	)
	if got := kinds(c); got != "head,seg,doc,ins,doc,attr" {
		t.Fatalf("records %s", got)
	}
	fd := c.out[4].(*rec.Doc)
	if fd.Host != c.out[3].(*rec.Ins).Node.ID || fd.URL != "http://y/" {
		t.Fatalf("frame doc %+v", fd)
	}
	html := c.st.Node(fd.Doc).Kids[0]
	if html.A["lang"] != "en" || html.A["dir"] != "ltr" {
		t.Fatalf("the frame's changes before and after attaching: %v", html.A)
	}
}

func TestSetChildNodesStyleTextRulesAndObserver(t *testing.T) {
	style := `{"nodeId":5,"backendNodeId":104,"nodeType":1,"nodeName":"STYLE","localName":"style","attributes":[],"children":[` +
		`{"nodeId":6,"backendNodeId":105,"nodeType":3,"nodeName":"#text","nodeValue":"cut"}]}`
	box := `{"nodeId":7,"backendNodeId":106,"nodeType":1,"nodeName":"INPUT","localName":"input","attributes":["type","checkbox"]}`
	c := feed(t,
		`{"t":0,"s":"","type":"snapshot","frameId":"F","root":`+doc("http://x/", style+","+box)+`}`,
		`{"t":1,"s":"","type":"styleText","ownerNode":104,"text":"`+strings.Repeat("a", 12000)+`"}`,
		`{"t":2,"s":"","type":"rules","ownerNode":104,"rules":[".a{}",".b{}"]}`,
		`{"t":3,"s":"","type":"value","nodeId":7,"v":"1"}`,
		`{"t":4,"s":"","type":"setChildNodes","parentId":4,"nodes":[{"nodeId":8,"backendNodeId":107,"nodeType":1,"nodeName":"P","localName":"p","attributes":[]}]}`,
		`{"t":5,"s":"","type":"layout","docs":[{"url":"http://x/","backend":[100,107],"layoutNode":[1],"bounds":[[1,2,3.333,4]]}]}`,
		`{"t":6,"s":"","type":"scroll","nodeId":2,"v":"0,300"}`,
		`{"t":7,"s":"","type":"click","nodeId":8,"v":"10,20"}`,
	)
	if got := kinds(c); got != "head,seg,doc,text,sheet,input,kids,layout,scroll,click" {
		t.Fatalf("records %s", got)
	}
	if len(c.out[3].(*rec.Text).V) != 12000 {
		t.Fatal("a <style> over the 10k cut gets its full text")
	}
	if sh := c.out[4].(*rec.Sheet); sh.Text != ".a{}\n.b{}" || sh.Owner == 0 {
		t.Fatalf("rules: %+v", sh)
	}
	if in := c.out[5].(*rec.Input); in.Checked == nil || !*in.Checked {
		t.Fatalf("a checkbox's value is checked: %+v", in)
	}
	p := c.out[6].(*rec.Kids).C[0].ID
	if b := c.st.Boxes[p]; b != (rec.Box{1, 2, 3.33, 4}) {
		t.Fatalf("layout box %v", b)
	}
	if sc := c.out[8].(*rec.Scroll); sc.ID != c.st.Top.ID || sc.Y != 300 {
		t.Fatalf("scrolling the html element scrolls the viewport: %+v", sc)
	}
	if cl := c.out[9].(*rec.Click); cl.Y != 320 {
		t.Fatalf("a click is in page coordinates (viewport 20 + scroll 300): %+v", cl)
	}
}

func TestAssetsAndExclusions(t *testing.T) {
	c := newConverter("t", 0)
	c.exclude = []string{"/late.ttf"}
	c.event(raw{T: 0, S: "", Type: "snapshot", Extra: []byte(`{"frameId":"F","root":` + doc("http://x/", "") + `}`)})
	for _, l := range []string{
		`{"t":1,"s":"","type":"assetBody","url":"http://x/a.png","mime":"image/png","b64":"AAEC"}`,
		`{"t":1,"s":"","type":"assetBody","url":"http://x/s.css","mime":"text/css","b64":"AAEC"}`,
		`{"t":1,"s":"","type":"asset","url":"http://x/s.css","text":"body{}"}`,
		`{"t":1,"s":"","type":"assetBody","url":"http://x/late.ttf","mime":"font/ttf","b64":"AAEC"}`,
	} {
		var r raw
		json.Unmarshal([]byte(l), &r)
		r.Extra = []byte(l)
		if err := c.event(r); err != nil {
			t.Fatal(err)
		}
	}
	if len(c.st.Assets) != 2 || c.st.Assets["http://x/s.css"].Size != 6 || c.st.Assets["http://x/late.ttf"] != nil {
		t.Fatalf("assets %v", c.st.Assets)
	}
}
