package live

import (
	"context"
	"strings"
	"testing"
)

// Independent raw-byte SHA-256 literals; never computed by the SUT.
const fixtureManifest = `{"case_ids":["read-a","read-b"],"network":"fixture","version":1}`
const fixtureDigest = "09a1f48523edd0927fb1bdf2ae2eedd49568f51d5621855c8b73d6e51dfdd7cb"

func TestHarnessManifestRawIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, raw, digest string
		valid             bool
	}{
		{"fixed", fixtureManifest, fixtureDigest, true},
		{"empty_object", `{}`, "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a", true},
		{"whitespace", " " + fixtureManifest, fixtureDigest, false},
		{"newline", fixtureManifest + "\n", fixtureDigest, false},
		{"key_order", `{"version":1,"network":"fixture","case_ids":["read-a","read-b"]}`, fixtureDigest, false},
		{"array_order", strings.Replace(fixtureManifest, `"read-a","read-b"`, `"read-b","read-a"`, 1), fixtureDigest, false},
		{"metadata", strings.TrimSuffix(fixtureManifest, "}") + `,"observed_at":"2099-01-01"}`, fixtureDigest, false},
		{"missing", "", fixtureDigest, false},
		{"missing_digest", fixtureManifest, "", false},
		{"bad_digest", fixtureManifest, "bad", false},
		// Matching hashes isolate JSON validation from digest mismatch.
		{"malformed", `{`, "021fb596db81e6d02bf3d2586ee3981fe519f275c0ac9ca76bbcf2ebb4097d96", false},
		{"null", `null`, "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b", false},
		{"array", `[]`, "4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945", false},
		{"trailing", `{}{}`, "b51f08b698d88d8027a935d9db649774949f5fb41a0c559bfee6a9a13225c72d", false},
		{"duplicate", `{"a":1,"a":2}`, "1c53ee0df7b12fd4d65b976120c7fa6b847dc41dffd7f0331c3237a1ceab1756", false},
		{"escaped_duplicate", `{"a":1,"\u0061":2}`, "c95ba945b310a34906bbfc28ce1346b87741d5a2b3066d1d3ecfa0acbd86392c", false},
		{"secret_parser", `{"token":"SYNTHETIC_SECRET",}`, "43654d1891cba58665d53135afb70f56e9ccf3e4e8736c79166a597852abb1bb", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			digest, err := verifyManifest([]byte(tc.raw), tc.digest)
			if tc.valid {
				if err != nil || digest != tc.digest {
					t.Error("raw-byte identity was not accepted and returned")
				}
			} else if err == nil {
				t.Error("invalid manifest accepted")
			}
			assertRedacted(t, err, "SYNTHETIC_SECRET")
			deps := fixtureDeps()
			deps.LoadManifest = func() ([]byte, string, error) { return []byte(tc.raw), tc.digest, nil }
			calls := 0
			c := harnessCase{ID: "read-a", Run: func(context.Context, harnessRequest) (string, error) { calls++; return "PASS", nil }}
			got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
			if tc.valid {
				checkStatus(t, got, "PASS")
				if calls != 1 || got.Digest != tc.digest {
					t.Error("report not bound to accepted manifest")
				}
			} else {
				checkStatus(t, got, "BLOCKED")
				if calls != 0 || got.Requests != 0 {
					t.Error("invalid manifest reached case")
				}
			}
			assertRedacted(t, got, "SYNTHETIC_SECRET")
		})
	}
}
