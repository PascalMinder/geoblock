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
	xForwardedFor  = "X-Forwarded-For"
	CountryHeader  = "X-IPCountry"
	caExampleIP    = "99.220.109.148"
	chExampleIP    = "82.220.110.18"
	privateRangeIP = "192.168.1.1"
	invalidIP      = "192.168.1.X"
	unknownCountry = "1.1.1.1"
	apiURI         = "https://get.geojs.io/v1/ip/country/{ip}"
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

	req.Header.Add(xForwardedFor, unknownCountry)

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

	req.Header.Add(xForwardedFor, unknownCountry)

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

func TestAllowBlacklistMode(t *testing.T) {
	cfg := createTesterConfig()
	cfg.BlackListMode = true
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

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestDenyBlacklistMode(t *testing.T) {
	cfg := createTesterConfig()
	cfg.BlackListMode = true
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

func TestExplicitlyAllowedIP(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CH")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, caExampleIP)
	cfg.LogLocalRequests = true

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

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestExplicitlyAllowedIPNoMatch(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CA")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, caExampleIP)

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

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestExplicitlyAllowedIPRangeIPV6(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CA")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "2a00:00c0:2:3::567:8001/128")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "8.8.8.8")

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

	req.Header.Add(xForwardedFor, "2a00:00c0:2:3::567:8001")

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestExplicitlyAllowedIPRangeIPV6NoMatch(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CA")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "2a00:00c0:2:3::567:8001/128")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "8.8.8.8")

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

	req.Header.Add(xForwardedFor, "2a00:00c0:2:3::567:8002")

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestExplicitlyAllowedIPRangeIPV4(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CA")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "178.90.234.0/27")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "8.8.8.8")

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

	req.Header.Add(xForwardedFor, "178.90.234.30")

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusOK)
}

func TestExplicitlyAllowedIPRangeIPV4NoMatch(t *testing.T) {
	cfg := createTesterConfig()
	cfg.Countries = append(cfg.Countries, "CA")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "178.90.234.0/27")
	cfg.AllowedIPAddresses = append(cfg.AllowedIPAddresses, "8.8.8.8")

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

	req.Header.Add(xForwardedFor, "178.90.234.55")

	handler.ServeHTTP(recorder, req)

	assertStatusCode(t, recorder.Result(), http.StatusForbidden)
}

func TestCountryHeader(t *testing.T) {
	cfg := createTesterConfig()
	cfg.AddCountryHeader = true
	cfg.Countries = append(cfg.Countries, "CA")

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

	assertHeader(t, req, CountryHeader, "CA")
}

func assertStatusCode(t *testing.T, req *http.Response, expected int) {
	t.Helper()

	if received := req.StatusCode; received != expected {
		t.Errorf("invalid status code: %d <> %d", expected, received)
	}
}

func assertHeader(t *testing.T, req *http.Request, key string, expected string) {
	t.Helper()

	if received := req.Header.Get(key); received != expected {
		t.Errorf("header value mismatch: %s: %s <> %s", key, expected, received)
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
