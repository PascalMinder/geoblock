// Package geoblock a Traefik plugin to block requests based on their country of origin.
package GeoBlock

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	xForwardedFor = "X-Forwarded-For"
)

// Config the plugin configuration.
type Config struct {
	Countries []string `json:"countries,omitempty"`
}

type IpEntry struct {
	Country   string
	Timestamp time.Time
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// GeoBlock a Traefik plugin.
type GeoBlock struct {
	next      http.Handler
	countries []string
	database  map[string]IpEntry
	name      string
}

// New created a new GeoBlock plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Countries) == 0 {
		return nil, fmt.Errorf("no allowed country code provided")
	}

	for _, country := range config.Countries {
		log.Println("Allowed country: ", country)
	}

	return &GeoBlock{
		next:      next,
		countries: config.Countries,
		database:  make(map[string]IpEntry),
		name:      name,
	}, nil
}

func (a *GeoBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reqIPAddr := a.CollectRemoteIP(req)

	for _, ipAddress := range reqIPAddr {
		var entry IpEntry

		entry, ok := a.database[ipAddress]

		if !ok {
			country := a.CallGeoJS(ipAddress)
			entry = IpEntry{Country: country, Timestamp: time.Now()}
			a.database[ipAddress] = entry
			log.Println("Added to database: ", entry)
		} else {
			log.Println("Loaded from database: ", entry)
		}

		for _, country := range a.countries {
			log.Println("Allowed country: ", country, " -> ", entry.Country)
		}

		var isAllowed bool = stringInSlice(entry.Country, a.countries)

		if !isAllowed {
			log.Printf("%s: request denied [%s]", a.name, ipAddress)
			rw.WriteHeader(http.StatusForbidden)

			return
		} else {
			log.Printf("%s: request denied [%s]", a.name, ipAddress)
		}
	}

	a.next.ServeHTTP(rw, req)
}

func (a *GeoBlock) CollectRemoteIP(req *http.Request) []string {
	var ipList []string

	xForwardedForValue := req.Header.Get(xForwardedFor)
	xForwardedForIPs := strings.Split(xForwardedForValue, ",")

	for key, value := range xForwardedForIPs {
		log.Println(key, value)
		ipList = append(ipList, value)
	}

	return ipList
}

func (a *GeoBlock) CallGeoJS(ipAddress string) string {
	geoJsClient := http.Client{
		Timeout: time.Second * 1, // Timeout after 1 seconds
	}

	req, err := http.NewRequest(http.MethodGet, "https://get.geojs.io/v1/ip/country/"+ipAddress, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := geoJsClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	sb := string(body)
	countryCode := strings.TrimSuffix(sb, "\n")

	log.Printf("Contry [%s] for ip %s fetched from GeoJs.io", countryCode, ipAddress)

	return countryCode
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
