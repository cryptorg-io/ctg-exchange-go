package ctgexchange

import (
	"encoding/json"
	"os"
	"testing"
)

// The signing implementation must reproduce the cross-language vectors
// in testdata/signing-vectors.json — the canonical fixture shared by
// every CTG.EXCHANGE SDK. This is what proves this SDK signs identically to
// the others and to the server.

type signingVectors struct {
	Credentials struct {
		Secret string `json:"secret"`
	} `json:"credentials"`
	REST []struct {
		Name              string `json:"name"`
		Ts                int64  `json:"ts"`
		Method            string `json:"method"`
		RequestURI        string `json:"request_uri"`
		Body              string `json:"body"`
		BodySHA256        string `json:"body_sha256"`
		CanonicalString   string `json:"canonical_string"`
		ExpectedSignature string `json:"expected_signature"`
	} `json:"rest"`
	WSAuth []struct {
		Name              string `json:"name"`
		Ts                int64  `json:"ts"`
		ExpectedSignature string `json:"expected_signature"`
	} `json:"ws_auth"`
}

func loadVectors(t *testing.T) signingVectors {
	t.Helper()
	data, err := os.ReadFile("testdata/signing-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v signingVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRESTSigningVectors(t *testing.T) {
	v := loadVectors(t)
	for _, c := range v.REST {
		t.Run(c.Name, func(t *testing.T) {
			if got := SHA256Hex(c.Body); got != c.BodySHA256 {
				t.Errorf("body hash: got %s want %s", got, c.BodySHA256)
			}
			got := RESTCanonicalString(c.Ts, c.Method, c.RequestURI, c.Body)
			if got != c.CanonicalString {
				t.Errorf("canonical: got %q want %q", got, c.CanonicalString)
			}
			sig := SignREST(v.Credentials.Secret, c.Ts, c.Method, c.RequestURI, c.Body)
			if sig != c.ExpectedSignature {
				t.Errorf("signature: got %s want %s", sig, c.ExpectedSignature)
			}
		})
	}
}

func TestWSAuthSigningVectors(t *testing.T) {
	v := loadVectors(t)
	for _, c := range v.WSAuth {
		t.Run(c.Name, func(t *testing.T) {
			if got := SignWSAuth(v.Credentials.Secret, c.Ts); got != c.ExpectedSignature {
				t.Errorf("got %s want %s", got, c.ExpectedSignature)
			}
		})
	}
}

func TestEmptyBodyHash(t *testing.T) {
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := SHA256Hex(""); got != want {
		t.Errorf("got %s want %s", got, want)
	}
}
