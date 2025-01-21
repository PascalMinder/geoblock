// Package geoblock a Traefik plugin to block requests based on their country of origin.
package geoblock

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	filePermissions                    = fs.FileMode(0666)
)

var (
	infoLogger = log.New(io.Discard, "INFO: GeoBlock: ", log.Ldate|log.Ltime)
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
	LogFilePath                  string   `yaml:"logFilePath"`
	RedirectURLIfDenied          string   `yaml:"redirectUrlIfDenied"`
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
}

// New created a new GeoBlock plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
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
		printConfiguration(config, infoLogger)
	}

	// create LRU cache for IP lookup
	cache, err := lru.NewLRUCache(config.CacheSize)
	if err != nil {
		infoLogger.Fatal(err)
	}

	// create custom log target if needed
	logFile, err := initializeLogFile(config.LogFilePath, infoLogger)
	if err != nil {
		infoLogger.Printf("Error initializing log file: %v\n", err)
	}

	// Set up a goroutine to close the file when the context is done
	if logFile != nil {
		go func(logger *log.Logger) {
			<-ctx.Done() // Wait for context cancellation
			logger.SetOutput(os.Stdout)
			logFile.Close()
			logger.Printf("Log file closed for middleware: %s\n", name)
		}(infoLogger)
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
	}, nil
}

func (a *GeoBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	requestIPAddresses, err := a.collectRemoteIP(req)
	if err != nil {
		// if one of the ip addresses could not be parsed, return status forbidden
		infoLogger.Println(err)
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
				a.next.ServeHTTP(rw, req)
			} else {
				rw.WriteHeader(a.httpStatusCodeDeniedRequest)
				a.next.ServeHTTP(rw, req)
			}
		}
	}

	a.next.ServeHTTP(rw, req)
}

func (a *GeoBlock) allowDenyIPAddress(requestIPAddr *net.IP, req *http.Request) bool {
	// check if the request IP address is a local address and if those are allowed
	if isPrivateIP(*requestIPAddr, a.privateIPRanges) {
		if a.allowLocalRequests {
			if a.logLocalRequests {
				infoLogger.Printf("%s: request allowed [%s] since local IP addresses are allowed", a.name, requestIPAddr)
			}
			return true
		}

		if a.logLocalRequests {
			infoLogger.Printf("%s: request denied [%s] since local IP addresses are denied", a.name, requestIPAddr)
		}
		return false
	}

	// check if the request IP address is explicitly allowed
	if ipInSlice(*requestIPAddr, a.allowedIPAddresses) {
		if a.logAllowedRequests {
			infoLogger.Printf("%s: request allowed [%s] since the IP address is explicitly allowed", a.name, requestIPAddr)
		}
		return true
	}

	// check if the request IP address is contained within one of the explicitly allowed IP address ranges
	for _, ipRange := range a.allowedIPRanges {
		if ipRange.Contains(*requestIPAddr) {
			if a.logLocalRequests {
				infoLogger.Printf("%s: request allowed [%s] since the IP address is explicitly allowed", a.name, requestIPAddr)
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
	cacheEntry, ok := a.database.Get(ipAddressString)

	var entry ipEntry
	var err error
	if !ok {
		entry, err = a.createNewIPEntry(req, ipAddressString)
		if err != nil && !(os.IsTimeout(err) && a.ignoreAPITimeout) {
			return false, ""
		} else if os.IsTimeout(err) && a.ignoreAPITimeout {
			infoLogger.Printf("%s: request allowed [%s] due to API timeout", a.name, requestIPAddr)
			// TODO: this was previously an immediate response to the client
			return true, ""
		}
	} else {
		entry = cacheEntry.(ipEntry)
	}

	if a.logAPIRequests {
		infoLogger.Println("Loaded from database: ", entry)
	}

	// check if existing entry was made more than a month ago, if so update the entry
	if time.Since(entry.Timestamp).Hours() >= numberOfHoursInMonth && a.forceMonthlyUpdate {
		entry, err = a.createNewIPEntry(req, ipAddressString)

		if err != nil {
			return false, ""
		}
	}

	// check if we are in black/white-list mode and allow/deny based on country code
	isAllowed := (stringInSlice(entry.Country, a.countries) != a.blackListMode) ||
		(entry.Country == unknownCountryCode && a.allowUnknownCountries)

	if !isAllowed {
		infoLogger.Printf("%s: request denied [%s] for country [%s]", a.name, requestIPAddr, entry.Country)
		return false, entry.Country
	}

	if a.logAllowedRequests {
		infoLogger.Printf("%s: request allowed [%s] for country [%s]", a.name, requestIPAddr, entry.Country)
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

func (a *GeoBlock) createNewIPEntry(req *http.Request, ipAddressString string) (ipEntry, error) {
	var entry ipEntry

	country, err := a.getCountryCode(req, ipAddressString)
	if err != nil {
		return entry, err
	}

	entry = ipEntry{Country: country, Timestamp: time.Now()}
	a.database.Add(ipAddressString, entry)

	if a.logAPIRequests {
		infoLogger.Println("Added to database: ", entry)
	}

	return entry, nil
}

func (a *GeoBlock) getCountryCode(req *http.Request, ipAddressString string) (string, error) {
	if len(a.iPGeolocationHTTPHeaderField) != 0 {
		country, err := a.readIPGeolocationHTTPHeader(req, a.iPGeolocationHTTPHeaderField)
		if err == nil {
			return country, nil
		}

		if a.logAPIRequests {
			infoLogger.Print("Failed to read country from HTTP header field [",
				a.iPGeolocationHTTPHeaderField,
				"], continuing with API lookup.")
		}
	}

	country, err := a.callGeoJS(ipAddressString)
	if err != nil {
		if !(os.IsTimeout(err) || a.ignoreAPITimeout) {
			infoLogger.Println(err)
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
		infoLogger.Printf("Country [%s] for ip %s fetched from %s", countryCode, ipAddress, apiURI)
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

func printConfiguration(config *Config, logger *log.Logger) {
	logger.Printf("allow local IPs: %t", config.AllowLocalRequests)
	logger.Printf("log local requests: %t", config.LogLocalRequests)
	logger.Printf("log allowed requests: %t", config.LogAllowedRequests)
	logger.Printf("log api requests: %t", config.LogAPIRequests)
	if len(config.IPGeolocationHTTPHeaderField) == 0 {
		logger.Printf("use custom HTTP header field for country lookup: %t", false)
	} else {
		logger.Printf("use custom HTTP header field for country lookup: %t [%s]", true, config.IPGeolocationHTTPHeaderField)
	}
	logger.Printf("API uri: %s", config.API)
	logger.Printf("API timeout: %d", config.APITimeoutMs)
	logger.Printf("ignore API timeout: %t", config.IgnoreAPITimeout)
	logger.Printf("cache size: %d", config.CacheSize)
	logger.Printf("force monthly update: %t", config.ForceMonthlyUpdate)
	logger.Printf("allow unknown countries: %t", config.AllowUnknownCountries)
	logger.Printf("unknown country api response: %s", config.UnknownCountryAPIResponse)
	logger.Printf("blacklist mode: %t", config.BlackListMode)
	logger.Printf("add country header: %t", config.AddCountryHeader)
	logger.Printf("countries: %v", config.Countries)
	logger.Printf("Denied request status code: %d", config.HTTPStatusCodeDeniedRequest)
	logger.Printf("Log file path: %s", config.LogFilePath)
	if len(config.RedirectURLIfDenied) != 0 {
		logger.Printf("Redirect URL on denied requests: %s", config.RedirectURLIfDenied)
	}
}

func initializeLogFile(logFilePath string, logger *log.Logger) (*os.File, error) {
	if len(logFilePath) == 0 {
		return nil, nil
	}

	writable, err := isFolder(logFilePath)
	if err != nil {
		logger.Println(err)
		return nil, err
	} else if !writable {
		logger.Println("Specified log folder is not writable")
		return nil, fmt.Errorf("folder is not writable: %s", logFilePath)
	}

	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, filePermissions)
	if err != nil {
		logger.Printf("Failed to open log file: %v\n", err)
		return nil, err
	}

	logger.SetOutput(logFile)
	return logFile, nil
}

func isFolder(filePath string) (bool, error) {
	dirPath := filepath.Dir(filePath)
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("path does not exist")
		}
		return false, fmt.Errorf("error checking path: %w", err)
	}

	if !info.IsDir() {
		return false, fmt.Errorf("folder does not exist")
	}

	return true, nil
}
