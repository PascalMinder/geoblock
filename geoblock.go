// Package geoblock a Traefik plugin to block requests based on their country of origin.
package geoblock

import (
	"context"
	"encoding/gob"
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
	xForwardedFor                      = "X-Forwarded-For"
	xRealIP                            = "X-Real-IP"
	countryHeader                      = "X-IPCountry"
	numberOfHoursInMonth               = 30 * 24
	unknownCountryCode                 = "AA"
	countryCodeLength                  = 2
	defaultDeniedRequestHTTPStatusCode = 403
	defaultCacheWriteCycle             = 15
)

// Config the plugin configuration.
type Config struct {
	SilentStartUp                bool     `yaml:"silentStartUp"`
	AllowLocalRequests           bool     `yaml:"allowLocalRequests"`
	LogLocalRequests             bool     `yaml:"logLocalRequests"`
	LogAllowedRequests           bool     `yaml:"logAllowedRequests"`
	LogAPIRequests               bool     `yaml:"logApiRequests"`
	API                          string   `yaml:"api"`
	APITimeoutMs                 int      `yaml:"apiTimeoutMs"`
	IgnoreAPITimeout             bool     `yaml:"ignoreApiTimeout"`
	IgnoreAPIFailures            bool     `yaml:"ignoreApiFailures"`
	IPGeolocationHTTPHeaderField string   `yaml:"ipGeolocationHttpHeaderField"`
	XForwardedForReverseProxy    bool     `yaml:"xForwardedForReverseProxy"`
	CacheSize                    int      `yaml:"cacheSize"`
	ForceMonthlyUpdate           bool     `yaml:"forceMonthlyUpdate"`
	AllowUnknownCountries        bool     `yaml:"allowUnknownCountries"`
	UnknownCountryAPIResponse    string   `yaml:"unknownCountryApiResponse"`
	BlackListMode                bool     `yaml:"blacklist"`
	Countries                    []string `yaml:"countries,omitempty"`
	AllowedIPAddresses           []string `yaml:"allowedIPAddresses,omitempty"`
	AddCountryHeader             bool     `yaml:"addCountryHeader"`
	HTTPStatusCodeDeniedRequest  int      `yaml:"httpStatusCodeDeniedRequest"`
	RedirectURLIfDenied          string   `yaml:"redirectUrlIfDenied"`
	LogFilePath                  string   `yaml:"logFilePath"`
	IPDatabaseCachePath          string   `yaml:"ipDatabaseCachePath"`
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
	next                         http.Handler
	silentStartUp                bool
	allowLocalRequests           bool
	logLocalRequests             bool
	logAllowedRequests           bool
	logAPIRequests               bool
	apiURI                       string
	apiTimeoutMs                 int
	ignoreAPITimeout             bool
	ignoreAPIFailures            bool
	iPGeolocationHTTPHeaderField string
	xForwardedForReverseProxy    bool
	forceMonthlyUpdate           bool
	allowUnknownCountries        bool
	unknownCountryCode           string
	blackListMode                bool
	countries                    []string
	allowedIPAddresses           []net.IP
	allowedIPRanges              []*net.IPNet
	privateIPRanges              []*net.IPNet
	addCountryHeader             bool
	httpStatusCodeDeniedRequest  int
	database                     *lru.LRUCache
	logFile                      *os.File
	redirectURLIfDenied          string
	name                         string
	infoLogger                   *log.Logger
	ipDatabasePersistence        *CachePersist
}

// New created a new GeoBlock plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	gob.Register(ipEntry{})

	infoLogger := log.New(io.Discard, "INFO: GeoBlock: ", log.Ldate|log.Ltime)

	// check geolocation API uri
	if len(config.API) == 0 || !strings.Contains(config.API, "{ip}") {
		return nil, fmt.Errorf("no api uri given")
	}

	// check if at least one allowed country is provided
	if len(config.Countries) == 0 {
		return nil, fmt.Errorf("no allowed country code provided")
	}

	// set default API timeout if non is given
	if config.APITimeoutMs == 0 {
		config.APITimeoutMs = 750
	}

	// set default HTTP status code for denied requests if non other is supplied
	deniedRequestHTTPStatusCode, err := getHTTPStatusCodeDeniedRequest(config.HTTPStatusCodeDeniedRequest)
	if err != nil {
		return nil, err
	}
	config.HTTPStatusCodeDeniedRequest = deniedRequestHTTPStatusCode

	// build allowed IP and IP ranges lists
	allowedIPAddresses, allowedIPRanges := parseAllowedIPAddresses(config.AllowedIPAddresses, infoLogger)

	infoLogger.SetOutput(os.Stdout)

	// output configuration of the middleware instance
	if !config.SilentStartUp {
		infoLogger.Printf("%s: Staring middleware", name)
		printConfiguration(name, config, infoLogger)
	}

	// create custom log target if needed
	var logFile *os.File
	if len(config.LogFilePath) > 0 {
		logTarget, err := CreateCustomLogTarget(ctx, infoLogger, name, config.LogFilePath)
		if err != nil {
			infoLogger.Fatal(err)
		}
		logFile = logTarget
	}

	// initialize local IP lookup cache
	cacheOptions := Options{
		CacheSize:       config.CacheSize,
		CachePath:       config.IPDatabaseCachePath,
		PersistInterval: defaultCacheWriteCycle,
		Logger:          infoLogger,
		Name:            name,
	}
	cache, ipDB, err := InitializeCache(ctx, cacheOptions)
	if err != nil {
		infoLogger.Fatal(err)
	}

	return &GeoBlock{
		next:                         next,
		silentStartUp:                config.SilentStartUp,
		allowLocalRequests:           config.AllowLocalRequests,
		logLocalRequests:             config.LogLocalRequests,
		logAllowedRequests:           config.LogAllowedRequests,
		logAPIRequests:               config.LogAPIRequests,
		apiURI:                       config.API,
		apiTimeoutMs:                 config.APITimeoutMs,
		ignoreAPITimeout:             config.IgnoreAPITimeout,
		ignoreAPIFailures:            config.IgnoreAPIFailures,
		iPGeolocationHTTPHeaderField: config.IPGeolocationHTTPHeaderField,
		xForwardedForReverseProxy:    config.XForwardedForReverseProxy,
		forceMonthlyUpdate:           config.ForceMonthlyUpdate,
		allowUnknownCountries:        config.AllowUnknownCountries,
		unknownCountryCode:           config.UnknownCountryAPIResponse,
		blackListMode:                config.BlackListMode,
		countries:                    config.Countries,
		allowedIPAddresses:           allowedIPAddresses,
		allowedIPRanges:              allowedIPRanges,
		privateIPRanges:              initPrivateIPBlocks(),
		database:                     cache,
		addCountryHeader:             config.AddCountryHeader,
		httpStatusCodeDeniedRequest:  config.HTTPStatusCodeDeniedRequest,
		logFile:                      logFile,
		redirectURLIfDenied:          config.RedirectURLIfDenied,
		name:                         name,
		infoLogger:                   infoLogger,
		ipDatabasePersistence:        ipDB, // may be nil => feature OFF
	}, nil
}

func (a *GeoBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	requestIPAddresses, err := a.collectRemoteIP(req)
	if err != nil {
		// if one of the ip addresses could not be parsed, return status forbidden
		a.infoLogger.Printf("%s: %s", a.name, err)
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	// only keep the first IP address, which should be the client (if the proxy behaves itself), to check if allowed or denied
	if a.xForwardedForReverseProxy {
		requestIPAddresses = requestIPAddresses[:1]
	}

	for _, requestIPAddress := range requestIPAddresses {
		if !a.allowDenyIPAddress(requestIPAddress, req) {
			if len(a.redirectURLIfDenied) != 0 {
				rw.Header().Set("Location", a.redirectURLIfDenied)
				rw.WriteHeader(http.StatusFound)
				return
			}

			rw.WriteHeader(a.httpStatusCodeDeniedRequest)
			return
		}
	}

	a.next.ServeHTTP(rw, req)
}

func (a *GeoBlock) allowDenyIPAddress(requestIPAddr *net.IP, req *http.Request) bool {
	// check if the request IP address is a local address and if those are allowed
	if isPrivateIP(*requestIPAddr, a.privateIPRanges) {
		if a.allowLocalRequests {
			if a.logLocalRequests {
				a.infoLogger.Printf("%s: request allowed [%s] since local IP addresses are allowed", a.name, requestIPAddr)
			}
			return true
		}

		if a.logLocalRequests {
			a.infoLogger.Printf("%s: request denied [%s] since local IP addresses are denied", a.name, requestIPAddr)
		}
		return false
	}

	// check if the request IP address is explicitly allowed
	if ipInSlice(*requestIPAddr, a.allowedIPAddresses) {
		if a.addCountryHeader {
			ok, countryCode := a.cachedRequestIP(requestIPAddr, req)
			if ok && len(countryCode) > 0 {
				req.Header.Set(countryHeader, countryCode)
			}
		}
		if a.logAllowedRequests {
			a.infoLogger.Printf("%s: request allowed [%s] since the IP address is explicitly allowed", a.name, requestIPAddr)
		}
		return true
	}

	// check if the request IP address is contained within one of the explicitly allowed IP address ranges
	for _, ipRange := range a.allowedIPRanges {
		if ipRange.Contains(*requestIPAddr) {
			if a.addCountryHeader {
				ok, countryCode := a.cachedRequestIP(requestIPAddr, req)
				if ok && len(countryCode) > 0 {
					req.Header.Set(countryHeader, countryCode)
				}
			}
			if a.logLocalRequests {
				a.infoLogger.Printf("%s: request allowed [%s] since the IP address is explicitly allowed", a.name, requestIPAddr)
			}
			return true
		}
	}

	// check if the GeoIP database contains an entry for the request IP address
	allowed, countryCode := a.allowDenyCachedRequestIP(requestIPAddr, req)

	if a.addCountryHeader && len(countryCode) > 0 {
		req.Header.Set(countryHeader, countryCode)
	}

	return allowed
}

func (a *GeoBlock) allowDenyCachedRequestIP(requestIPAddr *net.IP, req *http.Request) (bool, string) {
	ipAddressString := requestIPAddr.String()
	cacheEntry, cacheHit := a.database.Get(ipAddressString)

	var entry ipEntry
	var err error
	if !cacheHit {
		entry, err = a.createNewIPEntry(req, ipAddressString)
		if err != nil {
			if a.ignoreAPIFailures {
				a.infoLogger.Printf("%s: request allowed [%s] due to API failure", a.name, requestIPAddr)
				return true, ""
			}

			if os.IsTimeout(err) && a.ignoreAPITimeout {
				a.infoLogger.Printf("%s: request allowed [%s] due to API timeout", a.name, requestIPAddr)
				// TODO: this was previously an immediate response to the client
				return true, ""
			}

			a.infoLogger.Printf("%s: request denied [%s] due to error: %s", a.name, requestIPAddr, err)
			return false, ""
		}
	} else {
		entry = cacheEntry.(ipEntry)
		// order has changed
		a.ipDatabasePersistence.MarkDirty()
	}

	if a.logAPIRequests {
		a.infoLogger.Printf("%s: [%s] loaded from database: %s", a.name, requestIPAddr, entry)
	}

	// check if existing entry was made more than a month ago, if so update the entry
	if time.Since(entry.Timestamp).Hours() >= numberOfHoursInMonth && a.forceMonthlyUpdate {
		entry, err = a.createNewIPEntry(req, ipAddressString)
		if err != nil {
			if a.ignoreAPIFailures {
				a.infoLogger.Printf("%s: request allowed [%s] due to API failure", a.name, requestIPAddr)
				return true, ""
			}
			a.infoLogger.Printf("%s: request denied [%s] due to error: %s", a.name, requestIPAddr, err)
			return false, ""
		}
	}

	// check if we are in black/white-list mode and allow/deny based on country code
	isUnknownCountry := entry.Country == unknownCountryCode
	isCountryAllowed := stringInSlice(entry.Country, a.countries) != a.blackListMode
	isAllowed := isCountryAllowed || (isUnknownCountry && a.allowUnknownCountries)

	if !isAllowed {
		switch {
		case isUnknownCountry && !a.allowUnknownCountries:
			a.infoLogger.Printf(
				"%s: request denied [%s] for country [%s] due to: unknown country",
				a.name,
				requestIPAddr,
				entry.Country)
		case !isCountryAllowed:
			a.infoLogger.Printf(
				"%s: request denied [%s] for country [%s] due to: country is not allowed",
				a.name,
				requestIPAddr,
				entry.Country)
		default:
			a.infoLogger.Printf(
				"%s: request denied [%s] for country [%s]",
				a.name,
				requestIPAddr,
				entry.Country)
		}

		return false, entry.Country
	}

	if a.logAllowedRequests {
		a.infoLogger.Printf("%s: request allowed [%s] for country [%s]", a.name, requestIPAddr, entry.Country)
	}

	return true, entry.Country
}

func (a *GeoBlock) cachedRequestIP(requestIPAddr *net.IP, req *http.Request) (bool, string) {
	ipAddressString := requestIPAddr.String()
	cacheEntry, ok := a.database.Get(ipAddressString)

	var entry ipEntry
	var err error
	if !ok {
		entry, err = a.createNewIPEntry(req, ipAddressString)
		if err != nil {
			return false, ""
		}
	} else {
		entry = cacheEntry.(ipEntry)
		// order has changed
		a.ipDatabasePersistence.MarkDirty()
	}

	if a.logAPIRequests {
		a.infoLogger.Printf("%s: [%s] Loaded from database: %s", a.name, ipAddressString, entry)
	}

	// check if existing entry was made more than a month ago, if so update the entry
	if time.Since(entry.Timestamp).Hours() >= numberOfHoursInMonth && a.forceMonthlyUpdate {
		entry, err = a.createNewIPEntry(req, ipAddressString)
		if err != nil {
			return false, ""
		}
	}

	return true, entry.Country
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
		value = strings.Trim(value, " ")
		ipAddress, err := parseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		ipList = append(ipList, &ipAddress)
	}

	for _, value := range xRealIPList {
		value = strings.Trim(value, " ")
		ipAddress, err := parseIP(value)
		if err != nil {
			return ipList, fmt.Errorf("parsing failed: %s", err)
		}

		ipList = append(ipList, &ipAddress)
	}

	return ipList, nil
}

func (a *GeoBlock) createNewIPEntry(req *http.Request, ipAddressString string) (ipEntry, error) {
	var entry ipEntry

	country, err := a.getCountryCode(req, ipAddressString)
	if err != nil {
		return entry, err
	}

	entry = ipEntry{Country: country, Timestamp: time.Now()}
	a.database.Add(ipAddressString, entry)
	a.ipDatabasePersistence.MarkDirty() // new entry in the cache

	if a.logAPIRequests {
		a.infoLogger.Printf("%s: [%s] added to database: %s", a.name, ipAddressString, entry)
	}

	return entry, nil
}

func (a *GeoBlock) getCountryCode(req *http.Request, ipAddressString string) (string, error) {
	if len(a.iPGeolocationHTTPHeaderField) != 0 {
		country, err := a.readIPGeolocationHTTPHeader(req, a.iPGeolocationHTTPHeaderField)
		if err == nil {
			return country, nil
		}

		a.infoLogger.Printf(
			"%s: Failed to read country from HTTP header field [%s], continuing with API lookup.",
			a.name,
			a.iPGeolocationHTTPHeaderField,
		)
	}

	country, err := a.callGeoJS(ipAddressString)
	if err != nil {
		if !(os.IsTimeout(err) || a.ignoreAPITimeout) {
			a.infoLogger.Printf("%s: %s", a.name, err)
		}
		return "", err
	}

	return country, nil
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

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API response status code: %d", res.StatusCode)
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
		return "", fmt.Errorf("API response has more or less than 2 characters")
	}

	if a.logAPIRequests {
		a.infoLogger.Printf("%s: Country [%s] for ip %s fetched from %s", a.name, countryCode, ipAddress, apiURI)
	}

	return countryCode, nil
}

func (a *GeoBlock) readIPGeolocationHTTPHeader(req *http.Request, name string) (string, error) {
	countryCode := req.Header.Get(name)

	if len([]rune(countryCode)) != countryCodeLength {
		return "", fmt.Errorf("API response has more or less than 2 characters")
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

func getHTTPStatusCodeDeniedRequest(code int) (int, error) {
	if code != 0 {
		// check if given status code is valid
		if len(http.StatusText(code)) == 0 {
			return 0, fmt.Errorf("invalid denied request status code supplied")
		}

		return code, nil
	}

	return defaultDeniedRequestHTTPStatusCode, nil
}

func parseAllowedIPAddresses(entries []string, logger *log.Logger) ([]net.IP, []*net.IPNet) {
	var allowedIPAddresses []net.IP
	var allowedIPRanges []*net.IPNet

	for _, ipAddressEntry := range entries {
		ipAddressEntry = strings.Trim(ipAddressEntry, " ")
		// Attempt to parse as CIDR
		ip, ipBlock, err := net.ParseCIDR(ipAddressEntry)
		if err == nil {
			allowedIPAddresses = append(allowedIPAddresses, ip)
			allowedIPRanges = append(allowedIPRanges, ipBlock)
			continue
		}

		// Attempt to parse as a single IP address
		ipAddress := net.ParseIP(ipAddressEntry)
		if ipAddress == nil {
			logger.Fatal("Invalid IP address provided:", ipAddressEntry)
		}
		allowedIPAddresses = append(allowedIPAddresses, ipAddress)
	}

	return allowedIPAddresses, allowedIPRanges
}

func printConfiguration(name string, config *Config, logger *log.Logger) {
	logger.Printf("%s: allow local IPs: %t", name, config.AllowLocalRequests)
	logger.Printf("%s: log local requests: %t", name, config.LogLocalRequests)
	logger.Printf("%s: log allowed requests: %t", name, config.LogAllowedRequests)
	logger.Printf("%s: log api requests: %t", name, config.LogAPIRequests)
	if len(config.IPGeolocationHTTPHeaderField) == 0 {
		logger.Printf("%s: use custom HTTP header field for country lookup: %t", name, false)
	} else {
		logger.Printf("%s: use custom HTTP header field for country lookup: %t [%s]",
			name,
			true,
			config.IPGeolocationHTTPHeaderField,
		)
	}
	logger.Printf("%s: API uri: %s", name, config.API)
	logger.Printf("%s: API timeout: %d", name, config.APITimeoutMs)
	logger.Printf("%s: ignore API timeout: %t", name, config.IgnoreAPITimeout)
	logger.Printf("%s: cache size: %d", name, config.CacheSize)
	logger.Printf("%s: force monthly update: %t", name, config.ForceMonthlyUpdate)
	logger.Printf("%s: allow unknown countries: %t", name, config.AllowUnknownCountries)
	logger.Printf("%s: unknown country api response: %s", name, config.UnknownCountryAPIResponse)
	logger.Printf("%s: blacklist mode: %t", name, config.BlackListMode)
	logger.Printf("%s: add country header: %t", name, config.AddCountryHeader)
	logger.Printf("%s: countries: %v", name, config.Countries)
	logger.Printf("%s: Denied request status code: %d", name, config.HTTPStatusCodeDeniedRequest)
	logger.Printf("%s: Log file path: %s", name, config.LogFilePath)
	if len(config.RedirectURLIfDenied) != 0 {
		logger.Printf("%s: Redirect URL on denied requests: %s", name, config.RedirectURLIfDenied)
	}
}
