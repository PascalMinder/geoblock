package GeoBlock_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	GeoBlock "github.com/PascalMinder/GeoBlock"
)

const (
	xForwardedFor          = "X-Forwarded-For"
	CA                     = "99.220.109.148"
	CH                     = "82.220.110.18"
	PrivateRange           = "192.168.1.1"
	Invalid                = "192.168.1.X"
	UnknownCountryIpGoogle = "66.249.93.100"
)

func TestEmptyApi(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = ""
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := GeoBlock.New(ctx, next, cfg, "GeoBlock")

	// expect error
	if err == nil {
		t.Fatal("Empty API uri accepted")
	}
}

func TestMissingIpInApi(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := GeoBlock.New(ctx, next, cfg, "GeoBlock")

	// expect error
	if err == nil {
		t.Fatal("Missing IP block in API uri")
	}
}

func TestEmptyAllowedCountryList(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = make([]string, 0)
	cfg.CacheSize = 10

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := GeoBlock.New(ctx, next, cfg, "GeoBlock")

	// expect error
	if err == nil {
		t.Fatal("Empty country list is not allowed")
	}
}

func TestAllowedContry(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

func TestMultipleAllowedContry(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH", "CA")
	cfg.CacheSize = 10

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

func TestAllowedUnknownContry(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.AllowUnknownCountries = true
	cfg.UnknownCountryAPIResponse = "nil"
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

	req.Header.Add(xForwardedFor, UnknownCountryIpGoogle)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestDenyUnknownContry(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.AllowUnknownCountries = false
	cfg.UnknownCountryAPIResponse = "nil"
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

	req.Header.Add(xForwardedFor, UnknownCountryIpGoogle)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestAllowedContryCacheLookUp(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := GeoBlock.New(ctx, next, cfg, "GeoBlock")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	for i := 0; i < 2; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add(xForwardedFor, CH)

		handler.ServeHTTP(recorder, req)

		assertStatusCode(t, recorder.Result(), http.StatusOK)
	}
}

func TestDeniedContry(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

func TestAllowLocalIP(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = true
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

	req.Header.Add(xForwardedFor, PrivateRange)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestPrivateIPRange(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

	req.Header.Add(xForwardedFor, PrivateRange)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestInvalidIp(t *testing.T) {
	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	cfg.Api = "https://get.geojs.io/v1/ip/country/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

	req.Header.Add(xForwardedFor, Invalid)

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestInvalidApiResponse(t *testing.T) {
	// set up our fake api server
	var apiStub = httptest.NewServer(http.HandlerFunc(apiHandlerInvalid))

	cfg := GeoBlock.CreateConfig()

	cfg.AllowLocalRequests = false
	cfg.LogLocalRequests = false
	fmt.Println(apiStub.URL)
	cfg.Api = apiStub.URL + "/{ip}"
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.CacheSize = 10

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

	// the contry is allowed, but the api response is faulty.
	// therefore the request should be blocked
	req.Header.Add(xForwardedFor, CH)

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

func apiHandlerInvalid(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Invalid Response")
}
