package couchmessage

import (
	"reflect"
	"testing"
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
