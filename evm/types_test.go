package evm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScalarEncoding(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		size      int
		valid     bool
	}{
		{"address", "0x" + strings.Repeat("12", 20), 20, true},
		{"address-short", "0x12", 20, false},
		{"address-no-prefix", strings.Repeat("12", 20), 20, false},
		{"hash", "0x" + strings.Repeat("ab", 32), 32, true}, {"uppercase-hash", "0x" + strings.Repeat("AB", 32), 32, true},
		{"hash-long", "0x" + strings.Repeat("ab", 33), 32, false},
		{"hash-nonhex", "0x" + strings.Repeat("zz", 32), 32, false},
		{"empty-data", "0x", 0, true}, {"byte-data", "0x00ff", 0, true},
		{"odd-data", "0x0", 0, false}, {"nonhex-data", "0xgg", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var value any
			var err error
			switch tc.size {
			case 20:
				value, err = ParseAddress(tc.raw)
			case 32:
				value, err = ParseHash(tc.raw)
			default:
				value, err = ParseData(tc.raw)
			}
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if tc.valid {
				b, e := json.Marshal(value)
				if e != nil || string(b) != `"`+strings.ToLower(tc.raw)+`"` {
					t.Errorf("wire=%s err=%v", b, e)
				}
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
