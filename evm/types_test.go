package evm

import (
	"encoding/json"
	"reflect"
	"testing"
)

// This assertion keeps the tests compilable before the approved API migration.
func rawString[T any](t *testing.T, s string) T {
	t.Helper()
	var v T
	r := reflect.ValueOf(&v).Elem()
	if r.Kind() != reflect.String {
		t.Fatalf("raw RPC API: %s underlying kind = %s; want string", r.Type(), r.Kind())
	}
	r.SetString(s)
	return v
}

func TestRawStringAPI(t *testing.T) {
	for _, v := range []any{*new(Address), *new(Hash), *new(Data)} {
		t.Run(reflect.TypeOf(v).Name(), func(t *testing.T) {
			r := reflect.New(reflect.TypeOf(v)).Elem()
			if r.Kind() != reflect.String {
				t.Fatalf("underlying kind = %s; want string", r.Kind())
			}
			r.SetString("0xAbCD")
			s, ok := r.Interface().(interface{ ToString() string })
			if !ok || s.ToString() != "0xAbCD" {
				t.Fatal("ToString must preserve raw spelling")
			}
		})
	}
}

func TestBlockRefEncoding(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  BlockRef
		want string
	}{
		{"latest", Latest(), `"latest"`}, {"pending", Pending(), `"pending"`},
		{"zero", AtNumber(0), `"0x0"`}, {"number", AtNumber(16), `"0x10"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, e := json.Marshal(tc.ref)
			if e != nil || string(b) != tc.want {
				t.Fatalf("got %s, %v; want %s", b, e, tc.want)
			}
		})
	}
}

func TestRawRequestFieldTypes(t *testing.T) {
	assertFields(t, CallRequest{}, map[string]any{
		"From": "", "To": "", "Gas": "", "Value": "", "GasPrice": "", "MaxFeePerGas": "", "MaxPriorityFeePerGas": "", "Data": "",
	})
	assertFields(t, LogFilter{}, map[string]any{
		"FromBlock": BlockRef{}, "ToBlock": BlockRef{}, "Addresses": []string(nil), "Topics": [][]string(nil),
	})
}

func assertFields(t *testing.T, dto any, want map[string]any) {
	t.Helper()
	v := reflect.ValueOf(dto)
	if v.NumField() != len(want) {
		t.Errorf("%s has %d fields; want %d raw DTO fields", v.Type(), v.NumField(), len(want))
	}
	for name, expected := range want {
		field := v.FieldByName(name)
		if !field.IsValid() {
			t.Errorf("%s missing %s", v.Type(), name)
			continue
		}
		if !reflect.DeepEqual(field.Interface(), expected) {
			t.Errorf("%s.%s = %#v (%s); want %#v (%T)", v.Type(), name, field.Interface(), field.Type(), expected, expected)
		}
	}
}
