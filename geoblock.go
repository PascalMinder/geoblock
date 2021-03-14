// Package geoblock a Traefik plugin to block requests based on their country of origin.
package GeoBlock

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	xForwardedFor = "X-Forwarded-For"
	xRealIp       = "X-Real-IP"
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
	reqIPAddr, err := a.CollectRemoteIP(req)
	if err != nil {
		// if one of the ip addresses could not be parsed, return status forbidden
		log.Println(err)

		rw.WriteHeader(http.StatusForbidden)

		return
	}

	for _, ipAddress := range reqIPAddr {
		var entry IpEntry
		ipAddressString := ipAddress.String()

		entry, ok := a.database[ipAddressString]

		if !ok {
			country := a.CallGeoJS(ipAddressString)
			entry = IpEntry{Country: country, Timestamp: time.Now()}
			a.database[ipAddressString] = entry
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
			log.Printf("%s: request allowed [%s]", a.name, ipAddress)
		}
	}

	a.next.ServeHTTP(rw, req)
}

func (a *GeoBlock) CollectRemoteIP(req *http.Request) ([]*net.IP, error) {
	var ipList []*net.IP

	splitFn := func(c rune) bool {
		return c == ','
	}

	xForwardedForValue := req.Header.Get(xForwardedFor)
	xForwardedForIPs := strings.FieldsFunc(xForwardedForValue, splitFn)

	xRealIpValue := req.Header.Get(xRealIp)
	xRealIpIPs := strings.FieldsFunc(xRealIpValue, splitFn)

	for key, value := range xForwardedForIPs {
		ipAddress, err := parseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		log.Printf("appending ip address (%d): %s", key, ipAddress)
		ipList = append(ipList, &ipAddress)
	}

	for key, value := range xRealIpIPs {
		ipAddress, err := parseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		log.Printf("appending ip address (%d): %s", key, ipAddress)
		ipList = append(ipList, &ipAddress)
	}

	return ipList, nil
}

func (a *GeoBlock) CallGeoJS(ipAddress string) string {
	geoJsClient := http.Client{
		Timeout: time.Millisecond * 750, // Timeout after 150 milliseconds
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

func parseIP(addr string) (net.IP, error) {
	ipAddress := net.ParseIP(addr)

	if ipAddress == nil {
		return nil, fmt.Errorf("unable parse IP address from address [%s]", addr)
	}

	return ipAddress, nil
}
