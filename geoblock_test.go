package geoblock_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	geoblock "github.com/PascalMinder/geoblock"
)

const (
	xForwardedFor          = "X-Forwarded-For"
	caExampleIP            = "99.220.109.148"
	chExampleIP            = "82.220.110.18"
	privateRangeIP         = "192.168.1.1"
	invalidIP              = "192.168.1.X"
	unknownCountryIPGoogle = "66.249.93.100"
	apiURI                 = "https://get.geojs.io/v1/ip/country/{ip}"
)

func TestEmptyApi(t *testing.T) {
	cfg := createTesterConfig()
	cfg.API = ""
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := geoblock.New(ctx, next, cfg, "GeoBlock")

	// expect error
	if err == nil {
		t.Fatal("Empty API uri accepted")
	}
}

func TestMissingIpInApi(t *testing.T) {
	cfg := createTesterConfig()
	cfg.API = "https://get.geojs.io/v1/ip/country/"
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := geoblock.New(ctx, next, cfg, "GeoBlock")

	// expect error
	if err == nil {
		t.Fatal("Missing IP block in API uri")
	}
}

func TestEmptyAllowedCountryList(t *testing.T) {
	cfg := createTesterConfig()

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := geoblock.New(ctx, next, cfg, "GeoBlock")

	// expect error
	if err == nil {
		t.Fatal("Empty country list is not allowed")
	}
}

func TestAllowedCountry(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, chExampleIP)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestMultipleAllowedCountry(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH", "CA")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, chExampleIP)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestAllowedUnknownCountry(t *testing.T) {
	cfg := createTesterConfig()

	cfg.Countries = append(cfg.Countries, "CH")
	cfg.AllowUnknownCountries = true

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, unknownCountryIPGoogle)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestDenyUnknownCountry(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, unknownCountryIPGoogle)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestAllowedCountryCacheLookUp(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	for i := 0; i < 2; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add(xForwardedFor, chExampleIP)

		handler.ServeHTTP(recorder, req)

		assertStatusCode(t, recorder.Result(), http.StatusOK)
	}
}

func TestDeniedCountry(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, caExampleIP)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestAllowLocalIP(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.AllowLocalRequests = true

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, privateRangeIP)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestPrivateIPRange(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, privateRangeIP)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestInvalidIp(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(xForwardedFor, invalidIP)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestInvalidApiResponse(t *testing.T) {
	// set up our fake api server
	var apiStub = httptest.NewServer(http.HandlerFunc(apiHandlerInvalid))

	cfg := createTesterConfig()
	fmt.Println(apiStub.URL)
	cfg.API = apiStub.URL + "/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	// the country is allowed, but the api response is faulty.
	// therefore the request should be blocked
	req.Header.Add(xForwardedFor, chExampleIP)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func assertStatusCode(t *testing.T, req *http.Response, expected int) {
	t.Helper()

	if received := req.StatusCode; received != expected {
		t.Errorf("invalid status code: %d <> %d", expected, received)
	}
}

func apiHandlerInvalid(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Invalid Response")
}

func createTesterConfig() *geoblock.Config {
	cfg := geoblock.CreateConfig()

	cfg.API = apiURI
	cfg.APITimeoutMs = 750
	cfg.AllowLocalRequests = false
	cfg.AllowUnknownCountries = false
	cfg.CacheSize = 10
	cfg.Countries = make([]string, 0)
	cfg.ForceMonthlyUpdate = true
	cfg.LogAPIRequests = false
	cfg.LogAllowedRequests = false
	cfg.LogLocalRequests = false
	cfg.UnknownCountryAPIResponse = "nil"

	return cfg
}
