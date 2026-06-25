package ctgexchange

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// REST client wiring — offline, against an httptest server. These tests
// prove the client signs the exact request it sends: the canonical
// request-uri includes the query string, and the body hash covers the
// exact bytes on the wire.

func TestPublicRequestIsUnsigned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Signature") != "" {
				t.Error("public request must not be signed")
			}
			_, _ = w.Write([]byte(`[{"symbol":"CTGUSDT"}]`))
		}))
	defer srv.Close()

	syms, err := NewClient(Config{BaseURL: srv.URL}).GetSymbols(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 1 || syms[0].Symbol != "CTGUSDT" {
		t.Errorf("unexpected: %+v", syms)
	}
}

func TestSignedGetIncludesQueryInCanonicalURI(t *testing.T) {
	var uri, sig, ts string
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			uri = r.URL.RequestURI()
			sig = r.Header.Get("X-Signature")
			ts = r.Header.Get("X-Timestamp")
			_, _ = w.Write([]byte(`[]`))
		}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "ak_test", APISecret: "sk_test", BaseURL: srv.URL})
	_, err := c.GetOrders(context.Background(), "CTGUSDT",
		OrderQuery{Status: "open", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}

	want := "/api/v1/me/orders/CTGUSDT?limit=50&status=open"
	if uri != want {
		t.Errorf("uri: got %s want %s", uri, want)
	}
	tsN, _ := strconv.ParseInt(ts, 10, 64)
	if sig != SignREST("sk_test", tsN, "GET", uri, "") {
		t.Error("signature mismatch")
	}
}

func TestSignedPostBodyHash(t *testing.T) {
	var uri, sig, ts, body string
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			body, uri = string(b), r.URL.RequestURI()
			sig, ts = r.Header.Get("X-Signature"), r.Header.Get("X-Timestamp")
			_, _ = w.Write([]byte(`{"order":{"id":"o1"},"trades":[]}`))
		}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "ak_test", APISecret: "sk_test", BaseURL: srv.URL})
	_, err := c.PlaceOrder(context.Background(), "CTGUSDT", PlaceOrderParams{
		Side: "buy", Type: "limit", Price: "100.5", Qty: "2", ClientOrderID: "cid-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	tsN, _ := strconv.ParseInt(ts, 10, 64)
	if sig != SignREST("sk_test", tsN, "POST", uri, body) {
		t.Error("signature does not cover the exact body sent")
	}
}

func TestModifyOrderUnwrapsEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(
				`{"order":{"id":"o1","status":"open","price":"10.50"},"trades":[]}`))
		}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "ak_test", APISecret: "sk_test", BaseURL: srv.URL})
	o, err := c.ModifyOrder(context.Background(), "CTGUSDT", "o1",
		ModifyOrderParams{NewPrice: "10.50", NewQty: "0.12"})
	if err != nil {
		t.Fatal(err)
	}
	if o.ID != "o1" || o.Price != "10.50" {
		t.Errorf("envelope not unwrapped: %+v", o)
	}
}

func TestPrivateWithoutCredentials(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://example.invalid"})
	_, err := c.GetBalances(context.Background())
	if !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("want ErrMissingCredentials, got %v", err)
	}
}

func TestErrorStatusMapsToAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(
				`{"error":"bad","message":"nope","request_id":"r1"}`))
		}))
	defer srv.Close()

	_, err := NewClient(Config{BaseURL: srv.URL}).GetSymbols(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.StatusCode != 400 || apiErr.RequestID != "r1" || !apiErr.IsBadRequest() {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestRateLimitExposesRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate"}`))
		}))
	defer srv.Close()

	_, err := NewClient(Config{BaseURL: srv.URL}).GetSymbols(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !apiErr.IsRateLimited() || apiErr.RetryAfter != 7 {
		t.Fatalf("unexpected: %v", err)
	}
}
