package storagegc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const validRuntimeRow = `{"id":"first","process":{"pid":1,"birth":"darwin:123.4"},"role":"couch-runtime"}`

func TestRuntimeRegistryRejectsMalformedKnownFields(t *testing.T) {
	valid := `{"version":1,"processes":[` + validRuntimeRow + `]}`
	cases := []string{
		`null`, `[]`, `{}`, `{"version":null,"processes":[]}`, `{"version":"1","processes":[]}`, `{"version":2,"processes":[]}`, `{"version":1.0,"processes":[]}`,
		`{"version":1,"processes":null}`, `{"version":1,"processes":{}}`, `{"version":1,"processes":[null]}`, `{"version":1,"processes":[{}]}`,
		`{"version":1,"version":1,"processes":[]}`, valid + ` {}`,
		`{"version":1,"processes":[` + validRuntimeRow + `,` + validRuntimeRow + `]}`,
	}
	for _, field := range []string{`"id":null`, `"id":1`, `"id":""`, `"process":null`, `"process":[]`, `"process":{}`, `"role":null`, `"role":1`, `"role":""`, `"role":"` + strings.Repeat("x", 129) + `"`, `"target":null`, `"target":1`, `"target":"owner"`, `"diagnostic_protocol":null`, `"diagnostic_protocol":"1"`, `"diagnostic_protocol":1.5`, `"diagnostic_protocol":true`} {
		var row map[string]json.RawMessage
		if err := json.Unmarshal([]byte(validRuntimeRow), &row); err != nil {
			t.Fatal(err)
		}
		var replacement map[string]json.RawMessage
		if err := json.Unmarshal([]byte("{"+field+"}"), &replacement); err != nil {
			t.Fatal(err)
		}
		for key, value := range replacement {
			row[key] = value
		}
		raw, _ := json.Marshal(row)
		cases = append(cases, `{"version":1,"processes":[`+string(raw)+`]}`)
	}
	for _, process := range []string{`{"pid":null,"birth":"darwin:123.4"}`, `{"pid":0,"birth":"darwin:123.4"}`, `{"pid":1.5,"birth":"darwin:123.4"}`, `{"pid":"1","birth":"darwin:123.4"}`, `{"pid":1}`, `{"birth":"darwin:123.4"}`, `{"pid":1,"birth":null}`, `{"pid":1,"birth":"bogus"}`, `{"pid":1,"pid":2,"birth":"darwin:123.4"}`} {
		cases = append(cases, strings.Replace(valid, `{"pid":1,"birth":"darwin:123.4"}`, process, 1))
	}
	cases = append(cases, strings.Replace(valid, `"role":"couch-runtime"`, `"role":"couch-runtime","future":{"x":1,"x":2}`, 1))
	// Distinct IDs cannot disguise a duplicate process/role identity.
	cases = append(cases, `{"version":1,"processes":[`+validRuntimeRow+`,`+strings.Replace(validRuntimeRow, `"first"`, `"second"`, 1)+`]}`)
	rows := make([]string, 257)
	for i := range rows {
		rows[i] = fmt.Sprintf(`{"id":"%d","process":{"pid":%d,"birth":"darwin:123.4"},"role":"r"}`, i, i+1)
	}
	cases = append(cases, `{"version":1,"processes":[`+strings.Join(rows, ",")+`]}`)
	for i, raw := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var registry runtimeRegistry
			if err := json.Unmarshal([]byte(raw), &registry); err == nil {
				t.Fatalf("accepted malformed registry: %s", raw)
			}
		})
	}
}

func FuzzRuntimeRegistryCodec(f *testing.F) {
	for _, raw := range []string{`{"version":1,"processes":[]}`, `{"version":1,"future":{"a":[true,null]},"processes":[` + strings.Replace(validRuntimeRow, `"pid":1`, `"pid":1,"future":[]`, 1) + `]}`, `{"version":1,"processes":[` + validRuntimeRow + `]}`, `{"version":1,"processes":[` + strings.Replace(validRuntimeRow, `"role":"couch-runtime"`, `"role":"couch-runtime","target":"","diagnostic_protocol":0,"future":{"a":[true,null]}`, 1) + `]}`, `{"version":1,"processes":[` + strings.Replace(validRuntimeRow, `"role":"couch-runtime"`, `"role":"couch-runtime","diagnostic_protocol":17`, 1) + `]}`, `null`, `{"version":1,"version":1,"processes":[]}`} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		var registry runtimeRegistry
		if json.Unmarshal([]byte(raw), &registry) != nil {
			return
		}
		encoded, err := json.Marshal(registry)
		if err != nil {
			t.Fatal(err)
		}
		var again runtimeRegistry
		if err := json.Unmarshal(encoded, &again); err != nil {
			t.Fatal(err)
		}
		var before, after any
		if err := json.Unmarshal([]byte(raw), &before); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &after); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("round trip changed wire semantics: %s -> %s", raw, encoded)
		}
	})
}
