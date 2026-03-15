package apisix

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewClient("http://127.0.0.1:9180", "")
	if err == nil {
		t.Fatal("expected error when API key is empty")
	}
}

func TestDoAlwaysSetsAPIKeyHeader(t *testing.T) {
	t.Parallel()

	const expectedAPIKey = "real-admin-key"

	client, err := NewClient("http://127.0.0.1:9180", expectedAPIKey)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	client.client.SetTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get(HeaderAPIKey); got != expectedAPIKey {
			t.Fatalf("expected %s header %q, got %q", HeaderAPIKey, expectedAPIKey, got)
		}
		if got := r.Header.Get("X-Real-IP"); got != "" {
			t.Fatalf("expected X-Real-IP header to be removed, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-For"); got != "" {
			t.Fatalf("expected X-Forwarded-For header to be removed, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Proto"); got != "" {
			t.Fatalf("expected X-Forwarded-Proto header to be removed, got %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    r,
		}, nil
	}))

	_, err = client.Do(context.Background(), http.MethodGet, "/apisix/admin/routes", Request{
		Headers: map[string]string{
			HeaderAPIKey:       "wrong-key",
			"X-Real-IP":        "127.0.0.1",
			"X-Forwarded-For":  "203.0.113.10",
			"X-Forwarded-Proto": "https",
		},
	})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
}

func TestMarshalHeadersForLogMasksAPIKey(t *testing.T) {
	t.Parallel()

	got := marshalHeadersForLog(map[string][]string{
		HeaderAPIKey:   {"secret-key"},
		"Content-Type": {"application/json"},
	})

	want := `{"Content-Type":["application/json"],"x-api-key":["***"]}`
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected log headers: got %s want %s", got, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
