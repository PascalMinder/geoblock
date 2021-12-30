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

	lru "github.com/PascalMinder/GeoBlock/lrucache"
)

const (
	xForwardedFor        = "X-Forwarded-For"
	xRealIp              = "X-Real-IP"
	NumberOfHoursInMonth = 30 * 24
)

// Config the plugin configuration.
type Config struct {
	AllowLocalRequests bool     `yaml:"allowlocalrequests"`
	LogLocalRequests   bool     `yaml:"loglocalrequests"`
	LogAllowedRequests bool     `yaml:"logallowedrequests"`
	LogAPIRequests     bool     `yaml:"logapirequests"`
	Api                string   `yaml:"api"`
	CacheSize          int      `yaml:"cachesize"`
	ForceMonthlyUpdate bool     `yaml:"forcemonthlyupdate"`
	Countries          []string `yaml:"countries,omitempty"`
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
	next               http.Handler
	allowLocalRequests bool
	logLocalRequests   bool
	logAllowedRequests bool
	logAPIRequests     bool
	apiUri             string
	ForceMonthlyUpdate bool
	countries          []string
	privateIPRanges    []*net.IPNet
	database           *lru.LRUCache
	name               string
}

// New created a new GeoBlock plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Api) == 0 || !strings.Contains(config.Api, "{ip}") {
		return nil, fmt.Errorf("no api uri given")
	}

	if len(config.Countries) == 0 {
		return nil, fmt.Errorf("no allowed country code provided")
	}

	log.Println("API uri: ", config.Api)
	log.Println("allow local IPs: ", config.AllowLocalRequests)
	log.Println("log local requests: ", config.LogLocalRequests)
	log.Println("log allowed requests: ", config.LogAllowedRequests)
	log.Println("log api requests: ", config.LogAPIRequests)
	log.Println("allowed countries: ", config.Countries)

	cache, err := lru.NewLRUCache(config.CacheSize)
	if err != nil {
		log.Fatal(err)
	}

	return &GeoBlock{
		next:               next,
		allowLocalRequests: config.AllowLocalRequests,
		logLocalRequests:   config.LogLocalRequests,
		logAllowedRequests: config.LogAllowedRequests,
		logAPIRequests:     config.LogAPIRequests,
		apiUri:             config.Api,
		ForceMonthlyUpdate: config.ForceMonthlyUpdate,
		countries:          config.Countries,
		privateIPRanges:    InitPrivateIPBlocks(),
		database:           cache,
		name:               name,
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
		isPrivateIp := IsPrivateIP(*ipAddress, a.privateIPRanges)

		if isPrivateIp {
			if a.allowLocalRequests {
				if a.logLocalRequests {
					log.Println("Local ip allowed: ", ipAddress)
				}
				a.next.ServeHTTP(rw, req)
			} else {
				if a.logLocalRequests {
					log.Println("Local ip denied: ", ipAddress)
				}
				rw.WriteHeader(http.StatusForbidden)
			}

			return
		}

		cacheEntry, ok := a.database.Get(ipAddressString)

		if !ok {
			entry, err = a.CreateNewIPEntry(ipAddressString)

			if err != nil {
				rw.WriteHeader(http.StatusForbidden)
				return
			}
		} else {
			entry = cacheEntry.(IpEntry)

			log.Println("Loaded from database: ", entry)

			// check if existing entry was made more than a month ago, if so update the entry
			if time.Since(entry.Timestamp).Hours() >= NumberOfHoursInMonth && a.ForceMonthlyUpdate {
				entry, err = a.CreateNewIPEntry(ipAddressString)

				if err != nil {
					rw.WriteHeader(http.StatusForbidden)
					return
				}
			}
		}

		var isAllowed bool = StringInSlice(entry.Country, a.countries)

		if !isAllowed {
			log.Printf("%s: request denied [%s] for country [%s]", a.name, ipAddress, entry.Country)
			rw.WriteHeader(http.StatusForbidden)

			return
		} else {
			if a.logAllowedRequests {
				log.Printf("%s: request allowed [%s] for country [%s]", a.name, ipAddress, entry.Country)
			}
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

func (a *GeoBlock) CreateNewIPEntry(ipAddressString string) (IpEntry, error) {
	var entry IpEntry

	country, err := a.CallGeoJS(ipAddressString)
	if err != nil {
		log.Println(err)
		return entry, err
	}

	entry = IpEntry{Country: country, Timestamp: time.Now()}
	a.database.Add(ipAddressString, entry)
	log.Println("Added to database: ", entry)

	return entry, nil
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

	// this could possible cause a DoS attack
	if len([]rune(countryCode)) != 2 {
		return "", fmt.Errorf("API response has more than 2 characters")
	}

	if a.logAPIRequests {
		log.Printf("Country [%s] for ip %s fetched from %s", countryCode, ipAddress, apiUri)
	}

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

// https://stackoverflow.com/questions/41240761/check-if-ip-address-is-in-private-network-space
func InitPrivateIPBlocks() []*net.IPNet {
	var privateIPBlocks []*net.IPNet

	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}

	return privateIPBlocks
}

func IsPrivateIP(ip net.IP, privateIPBlocks []*net.IPNet) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}
