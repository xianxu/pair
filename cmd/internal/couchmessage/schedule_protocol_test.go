package couchmessage

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func TestScheduleStatusRequestHasClosedEmptyShape(t *testing.T) {
	if err := ValidateRequest(Request{Op: "schedule-status"}); err != nil {
		t.Fatal(err)
	}
	base := Request{Op: "schedule-status"}
	typeOf := reflect.TypeOf(base)
	for i := 0; i < typeOf.NumField(); i++ {
		if typeOf.Field(i).Name == "Op" {
			continue
		}
		request := base
		field := reflect.ValueOf(&request).Elem().Field(i)
		switch field.Kind() {
		case reflect.String:
			field.SetString("extra")
		case reflect.Bool:
			field.SetBool(true)
		case reflect.Int:
			field.SetInt(1)
		case reflect.Pointer:
			field.Set(reflect.New(field.Type().Elem()))
		default:
			t.Fatalf("new request kind needs an adversarial value: %s", field.Kind())
		}
		if ValidateRequest(request) == nil {
			t.Errorf("accepted foreign field %s", typeOf.Field(i).Name)
		}
	}
}

func TestScheduleStatusProtocolPreservesRefusalWithoutRoutingIt(t *testing.T) {
	snapshot := couchschedule.BuildStatus([]couchschedule.TaskState{
		{Key: couchschedule.TaskKey{Hub: "/hub", Task: "good"}, Allowed: true, Revision: "good-revision"},
		{Key: couchschedule.TaskKey{Hub: "/hub", Task: "BadTask"}, Refusal: "invalid name", Revision: "bad-revision"},
	}, "UTC")
	socket := transportSocket(t)
	server, err := StartServer(t.Context(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var request Request
		if err := strictjson.Decode(raw, &request); err != nil {
			return nil, err
		}
		if err := ValidateRequest(request); err != nil {
			return nil, err
		}
		return json.Marshal(Response{Code: "ok", Schedule: &snapshot})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var response Response
	if err = Call(t.Context(), socket, Request{Op: "schedule-status"}, &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "ok" || response.Schedule == nil {
		t.Fatalf("response %#v", response)
	}
	if err = response.Schedule.Validate(); err != nil {
		t.Fatalf("refused entry poisoned decoded protocol: %v", err)
	}
	good, bad := 0, 0
	for _, row := range response.Schedule.Tasks {
		if row.Task == "good" {
			good++
		}
		if row.State == "refused" {
			bad++
			if row.Task != "" || row.Entry == "" || row.EntryID == "" {
				t.Fatalf("refused entry retained routing %#v", row)
			}
		}
	}
	if good != 1 || bad != 1 {
		t.Fatalf("lost mixed rows: %#v", response.Schedule.Tasks)
	}
}
