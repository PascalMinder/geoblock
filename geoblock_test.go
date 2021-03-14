package GeoBlock_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	GeoBlock "github.com/PascalMinder/GeoBlock"
)

const (
	xForwardedFor = "X-Forwarded-For"
	CA            = "99.220.109.148"
	CH            = "82.220.110.18"
)

func TestAllowedContry(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := GeoBlock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, CH)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestDeniedContry(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := GeoBlock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, CA)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func assertStatusCode(t *testing.T, req *http.Response, expected int) {
	t.Helper()

	received := req.StatusCode

	if received != expected {
		t.Errorf("invalid status code: %d <> %d", expected, received)
	}
}
