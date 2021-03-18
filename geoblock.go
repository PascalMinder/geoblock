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
	Api       string   `yaml:"api"`
	Countries []string `yaml:"countries,omitempty"`
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
	apiUri    string
	countries []string
	database  map[string]IpEntry
	name      string
}

// New created a new GeoBlock plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Api) == 0 || !strings.Contains(config.Api, "{ip}") {
		return nil, fmt.Errorf("no api uri given")
	}

	if len(config.Countries) == 0 {
		return nil, fmt.Errorf("no allowed country code provided")
	}

	log.Println("allowed countries: ", config.Countries)

	return &GeoBlock{
		next:      next,
		apiUri:    config.Api,
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
			country, err := a.CallGeoJS(ipAddressString)
			if err != nil {
				log.Println(err)
				rw.WriteHeader(http.StatusForbidden)
				return
			}

			entry = IpEntry{Country: country, Timestamp: time.Now()}
			a.database[ipAddressString] = entry
			log.Println("Added to database: ", entry)
		} else {
			log.Println("Loaded from database: ", entry)
		}

		var isAllowed bool = StringInSlice(entry.Country, a.countries)

		if !isAllowed {
			log.Printf("%s: request denied [%s] for country [%s]", a.name, ipAddress, entry.Country)
			rw.WriteHeader(http.StatusForbidden)

			return
		} else {
			log.Printf("%s: request allowed [%s] for country [%s]", a.name, ipAddress, entry.Country)
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

	for _, value := range xForwardedForIPs {
		ipAddress, err := ParseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		ipList = append(ipList, &ipAddress)
	}

	for _, value := range xRealIpIPs {
		ipAddress, err := ParseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		ipList = append(ipList, &ipAddress)
	}

	return ipList, nil
}

func (a *GeoBlock) CallGeoJS(ipAddress string) (string, error) {
	geoJsClient := http.Client{
		Timeout: time.Millisecond * 750, // Timeout after 150 milliseconds
	}

	apiUri := strings.Replace(a.apiUri, "{ip}", ipAddress, 1)

	req, err := http.NewRequest(http.MethodGet, apiUri, nil)
	if err != nil {
		return "", err
	}

	res, err := geoJsClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	sb := string(body)
	countryCode := strings.TrimSuffix(sb, "\n")

	if len([]rune(countryCode)) != 2 {
		return "", fmt.Errorf("API response has more than 2 characters")
	}

	log.Printf("Country [%s] for ip %s fetched from %s", countryCode, ipAddress, apiUri)

	return countryCode, nil
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

func ParseIP(addr string) (net.IP, error) {
	ipAddress := net.ParseIP(addr)

	if ipAddress == nil {
		return nil, fmt.Errorf("unable parse IP address from address [%s]", addr)
	}

	return ipAddress, nil
}
