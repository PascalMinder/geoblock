// Package geoblock a Traefik plugin to block requests based on their country of origin.
package geoblock

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	lru "github.com/PascalMinder/geoblock/lrucache"
)

const (
	xForwardedFor        = "X-Forwarded-For"
	xRealIP              = "X-Real-IP"
	countryHeader        = "X-IPCountry"
	numberOfHoursInMonth = 30 * 24
	unknownCountryCode   = "AA"
	countryCodeLength    = 2
)

var (
	infoLogger = log.New(io.Discard, "INFO: GeoBlock: ", log.Ldate|log.Ltime)
)

// Config the plugin configuration.
type Config struct {
	SilentStartUp             bool     `yaml:"silentStartUp"`
	AllowLocalRequests        bool     `yaml:"allowLocalRequests"`
	LogLocalRequests          bool     `yaml:"logLocalRequests"`
	LogAllowedRequests        bool     `yaml:"logAllowedRequests"`
	LogAPIRequests            bool     `yaml:"logApiRequests"`
	API                       string   `yaml:"api"`
	APITimeoutMs              int      `yaml:"apiTimeoutMs"`
	IgnoreAPITimeout          bool     `yaml:"ignoreApiTimeout"`
	CacheSize                 int      `yaml:"cacheSize"`
	ForceMonthlyUpdate        bool     `yaml:"forceMonthlyUpdate"`
	AllowUnknownCountries     bool     `yaml:"allowUnknownCountries"`
	UnknownCountryAPIResponse string   `yaml:"unknownCountryApiResponse"`
	BlackListMode             bool     `yaml:"blacklist"`
	Countries                 []string `yaml:"countries,omitempty"`
	AllowedIPAddresses        []string `yaml:"allowedIPAddresses,omitempty"`
	AddCountryHeader          bool     `yaml:"addCountryHeader"`
}

type ipEntry struct {
	Country   string
	Timestamp time.Time
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// GeoBlock a Traefik plugin.
type GeoBlock struct {
	next                  http.Handler
	silentStartUp         bool
	allowLocalRequests    bool
	logLocalRequests      bool
	logAllowedRequests    bool
	logAPIRequests        bool
	apiURI                string
	apiTimeoutMs          int
	ignoreAPITimeout      bool
	forceMonthlyUpdate    bool
	allowUnknownCountries bool
	unknownCountryCode    string
	blackListMode         bool
	countries             []string
	allowedIPAddresses    []net.IP
	allowedIPRanges       []*net.IPNet
	privateIPRanges       []*net.IPNet
	addCountryHeader      bool
	database              *lru.LRUCache
	name                  string
}

// New created a new GeoBlock plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.API) == 0 || !strings.Contains(config.API, "{ip}") {
		return nil, fmt.Errorf("no api uri given")
	}

	if len(config.Countries) == 0 {
		return nil, fmt.Errorf("no allowed country code provided")
	}

	if config.APITimeoutMs == 0 {
		config.APITimeoutMs = 750
	}

	var allowedIPAddresses []net.IP
	var allowedIPRanges []*net.IPNet
	for _, ipAddressEntry := range config.AllowedIPAddresses {
		ip, ipBlock, err := net.ParseCIDR(ipAddressEntry)
		if err == nil {
			allowedIPAddresses = append(allowedIPAddresses, ip)
			allowedIPRanges = append(allowedIPRanges, ipBlock)
			continue
		}

		ipAddress := net.ParseIP(ipAddressEntry)
		if ipAddress == nil {
			infoLogger.Fatal("Invalid ip address given!")
		}
		allowedIPAddresses = append(allowedIPAddresses, ipAddress)
	}

	infoLogger.SetOutput(os.Stdout)

	if !config.SilentStartUp {
		infoLogger.Printf("allow local IPs: %t", config.AllowLocalRequests)
		infoLogger.Printf("log local requests: %t", config.LogLocalRequests)
		infoLogger.Printf("log allowed requests: %t", config.LogAllowedRequests)
		infoLogger.Printf("log api requests: %t", config.LogAPIRequests)
		infoLogger.Printf("API uri: %s", config.API)
		infoLogger.Printf("API timeout: %d", config.APITimeoutMs)
		infoLogger.Printf("ignore API timeout: %t", config.IgnoreAPITimeout)
		infoLogger.Printf("cache size: %d", config.CacheSize)
		infoLogger.Printf("force monthly update: %t", config.ForceMonthlyUpdate)
		infoLogger.Printf("allow unknown countries: %t", config.AllowUnknownCountries)
		infoLogger.Printf("unknown country api response: %s", config.UnknownCountryAPIResponse)
		infoLogger.Printf("blacklist mode: %t", config.BlackListMode)
		infoLogger.Printf("add country header: %t", config.AddCountryHeader)
		infoLogger.Printf("countries: %v", config.Countries)
	}

	cache, err := lru.NewLRUCache(config.CacheSize)
	if err != nil {
		infoLogger.Fatal(err)
	}

	return &GeoBlock{
		next:                  next,
		silentStartUp:         config.SilentStartUp,
		allowLocalRequests:    config.AllowLocalRequests,
		logLocalRequests:      config.LogLocalRequests,
		logAllowedRequests:    config.LogAllowedRequests,
		logAPIRequests:        config.LogAPIRequests,
		apiURI:                config.API,
		apiTimeoutMs:          config.APITimeoutMs,
		ignoreAPITimeout:      config.IgnoreAPITimeout,
		forceMonthlyUpdate:    config.ForceMonthlyUpdate,
		allowUnknownCountries: config.AllowUnknownCountries,
		unknownCountryCode:    config.UnknownCountryAPIResponse,
		blackListMode:         config.BlackListMode,
		countries:             config.Countries,
		allowedIPAddresses:    allowedIPAddresses,
		allowedIPRanges:       allowedIPRanges,
		privateIPRanges:       initPrivateIPBlocks(),
		database:              cache,
		addCountryHeader:      config.AddCountryHeader,
		name:                  name,
	}, nil
}

func (a *GeoBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reqIPAddr, err := a.collectRemoteIP(req)
	if err != nil {
		// if one of the ip addresses could not be parsed, return status forbidden
		infoLogger.Println(err)
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	for _, ipAddress := range reqIPAddr {
		var entry ipEntry
		ipAddressString := ipAddress.String()
		privateIP := isPrivateIP(*ipAddress, a.privateIPRanges)

		if privateIP {
			if a.allowLocalRequests {
				if a.logLocalRequests {
					infoLogger.Println("Local ip allowed: ", ipAddress)
				}
				a.next.ServeHTTP(rw, req)
			} else {
				if a.logLocalRequests {
					infoLogger.Println("Local ip denied: ", ipAddress)
				}
				rw.WriteHeader(http.StatusForbidden)
			}

			return
		}

		if ipInSlice(*ipAddress, a.allowedIPAddresses) {
			if a.logLocalRequests {
				infoLogger.Println("Allow explicitly allowed ip: ", ipAddress)
			}
			a.next.ServeHTTP(rw, req)
			return
		}

		for _, ipRange := range a.allowedIPRanges {
			if ipRange.Contains(*ipAddress) {
				if a.logLocalRequests {
					infoLogger.Println("Allow explicitly allowed ip: ", ipAddress)
				}
				a.next.ServeHTTP(rw, req)
				return
			}
		}

		cacheEntry, ok := a.database.Get(ipAddressString)

		if !ok {
			entry, err = a.createNewIPEntry(ipAddressString, a.ignoreAPITimeout)

			if err != nil && !(os.IsTimeout(err) && a.ignoreAPITimeout) {
				rw.WriteHeader(http.StatusForbidden)
				return
			} else if os.IsTimeout(err) && a.ignoreAPITimeout {
				infoLogger.Printf("%s: request allowed [%s] due to API timeout!", a.name, ipAddress)
				a.next.ServeHTTP(rw, req)
				return
			}
		} else {
			entry = cacheEntry.(ipEntry)

			if a.logAPIRequests {
				infoLogger.Println("Loaded from database: ", entry)
			}

			// check if existing entry was made more than a month ago, if so update the entry
			if time.Since(entry.Timestamp).Hours() >= numberOfHoursInMonth && a.forceMonthlyUpdate {
				entry, err = a.createNewIPEntry(ipAddressString, a.ignoreAPITimeout)

				if err != nil {
					rw.WriteHeader(http.StatusForbidden)
					return
				}
			}
		}

		isAllowed := (stringInSlice(entry.Country, a.countries) != a.blackListMode) ||
			(entry.Country == unknownCountryCode && a.allowUnknownCountries)

		if !isAllowed {
			infoLogger.Printf("%s: request denied [%s] for country [%s]", a.name, ipAddress, entry.Country)
			rw.WriteHeader(http.StatusForbidden)

			return
		} else if a.logAllowedRequests {
			infoLogger.Printf("%s: request allowed [%s] for country [%s]", a.name, ipAddress, entry.Country)
		}

		if a.addCountryHeader {
			req.Header.Set(countryHeader, entry.Country)
		}
	}

	a.next.ServeHTTP(rw, req)
}

func (a *GeoBlock) collectRemoteIP(req *http.Request) ([]*net.IP, error) {
	var ipList []*net.IP

	splitFn := func(c rune) bool {
		return c == ','
	}

	xForwardedForValue := req.Header.Get(xForwardedFor)
	xForwardedForIPs := strings.FieldsFunc(xForwardedForValue, splitFn)

	xRealIPValue := req.Header.Get(xRealIP)
	xRealIPList := strings.FieldsFunc(xRealIPValue, splitFn)

	for _, value := range xForwardedForIPs {
		ipAddress, err := parseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		ipList = append(ipList, &ipAddress)
	}

	for _, value := range xRealIPList {
		ipAddress, err := parseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		ipList = append(ipList, &ipAddress)
	}

	return ipList, nil
}

func (a *GeoBlock) createNewIPEntry(ipAddressString string, ignoreApiTimeout bool) (ipEntry, error) {
	var entry ipEntry

	country, err := a.callGeoJS(ipAddressString)
	if err != nil {
		if !(os.IsTimeout(err) || ignoreApiTimeout) {
			infoLogger.Println(err)
		}
		return entry, err
	}

	entry = ipEntry{Country: country, Timestamp: time.Now()}
	a.database.Add(ipAddressString, entry)

	if a.logAPIRequests {
		infoLogger.Println("Added to database: ", entry)
	}

	return entry, nil
}

func (a *GeoBlock) callGeoJS(ipAddress string) (string, error) {
	geoJsClient := http.Client{
		Timeout: time.Millisecond * time.Duration(a.apiTimeoutMs),
	}

	apiURI := strings.Replace(a.apiURI, "{ip}", ipAddress, 1)

	req, err := http.NewRequest(http.MethodGet, apiURI, nil)
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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	sb := string(body)
	countryCode := strings.TrimSuffix(sb, "\n")

	// api response for unknown country
	if len([]rune(countryCode)) == len(a.unknownCountryCode) && countryCode == a.unknownCountryCode {
		return unknownCountryCode, nil
	}

	// this could possible cause a DoS attack
	if len([]rune(countryCode)) != countryCodeLength {
		return "", fmt.Errorf("API response has more than 2 characters")
	}

	if a.logAPIRequests {
		infoLogger.Printf("Country [%s] for ip %s fetched from %s", countryCode, ipAddress, apiURI)
	}

	return countryCode, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

func ipInSlice(a net.IP, list []net.IP) bool {
	for _, b := range list {
		if b.Equal(a) {
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

// https://stackoverflow.com/questions/41240761/check-if-ip-address-is-in-private-network-space
func initPrivateIPBlocks() []*net.IPNet {
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

func isPrivateIP(ip net.IP, privateIPBlocks []*net.IPNet) bool {
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
