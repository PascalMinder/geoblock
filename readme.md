# GeoBlock

Simple plugin for [Traefik](https://github.com/containous/traefik) to block or allow requests based on their country of origin. Uses [GeoJs.io](https://www.geojs.io/).

## Configuration

It is possible to install the [plugin locally](https://traefik.io/blog/using-private-plugins-in-traefik-proxy-2-5/) or to install it through [Traefik Pilot](https://pilot.traefik.io/plugins).

### Configuration as local plugin

Depending on your setup, the installation steps might differ from the one described here. This example assumes that your Traefik instance runs in a Docker container and uses the [official image](https://hub.docker.com/_/traefik/).

Download the latest release of the plugin and save it to a location the Traefik container can reach. Below is an example of a possible setup. Notice how the plugin source is mapped into the container (`/plugin/geoblock:/plugins-local/src/github.com/PascalMinder/geoblock/`) via a volume bind mount:

#### `docker-compose.yml`

````yml
version: "3.7"

services:
  traefik:
    image: traefik

    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /docker/config/traefik/traefik.yml:/etc/traefik/traefik.yml
      - /docker/config/traefik/dynamic-configuration.yml:/etc/traefik/dynamic-configuration.yml
      - /docker/config/traefik/plugin/geoblock:/plugins-local/src/github.com/PascalMinder/geoblock/

    ports:
      - "80:80"

  hello:
    image: containous/whoami
    labels:
      - traefik.enable=true
      - traefik.http.routers.hello.entrypoints=http
      - traefik.http.routers.hello.rule=Host(`hello.localhost`)
      - traefik.http.services.hello.loadbalancer.server.port=80
      - traefik.http.routers.hello.middlewares=my-plugin@file
````

To complete the setup, the Traefik configuration must be extended with the plugin. For this you must create the `traefik.yml` and the dynamic-configuration.yml` files if not present already.

````yml
log:
  level: INFO

experimental:
  localPlugins:
    geoblock:
      moduleName: github.com/PascalMinder/geoblock
````

#### `dynamic-configuration.yml`

````yml
http:
  middlewares:
    geoblock-ch:
      plugin:
        geoblock:
          silentStartUp: false
          allowLocalRequests: true
          logLocalRequests: false
          logAllowedRequests: false
          logApiRequests: true
          api: "https://get.geojs.io/v1/ip/country/{ip}"
          apiTimeoutMs: 750                                 # optional
          cacheSize: 15
          forceMonthlyUpdate: true
          allowUnknownCountries: false
          unknownCountryApiResponse: "nil"
          countries:
            - CH
````

### Traefik Plugin registry

This procedure will install the plugin via the [Traefik Plugin registry](https://plugins.traefik.io/install).

Add the following to your `traefik-config.yml`

```yml
experimental:
  plugins:
    geoblock:
      moduleName: "github.com/PascalMinder/geoblock"
      version: "v0.2.5"

# other stuff you might have in your traefik-config
entryPoints:
  http:
    address: ":80"
  https:
    address: ":443"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
  file:
    filename: "/etc/traefik/dynamic-configuration.yml"
```

In your dynamic configuration add the following:

```yml
http:
  middlewares:
    my-GeoBlock:
      plugin:
        geoblock:
          silentStartUp: false
          allowLocalRequests: true
          logLocalRequests: false
          logAllowedRequests: false
          logApiRequests: false
          api: "https://get.geojs.io/v1/ip/country/{ip}"
          apiTimeoutMs: 500
          cacheSize: 25
          forceMonthlyUpdate: true
          allowUnknownCountries: false
          unknownCountryApiResponse: "nil"
          countries:
            - CH
```

And some example docker file for traefik:

```yml
version: "3"
networks:
  proxy:
    external: true # specifies that this network has been created outside of Compose, raises an error if it doesn’t exist
services:
  traefik:
    image: traefik:latest
    container_name: traefik
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    networks:
      proxy:
        aliases:
          - traefik
    ports:
      - 80:80
      - 443:443
    volumes:
      - "/etc/timezone:/etc/timezone:ro"
      - "/etc/localtime:/etc/localtime:ro"
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
      - "/a/docker/config/traefik/data/traefik.yml:/etc/traefik/traefik.yml:ro"
      - "/a/docker/config/traefik/data/dynamic-configuration.yml:/etc/traefik/dynamic-configuration.yml"
```

This configuration might not work. It's just to give you an idea how to configure it.

## Full plugin sample configuration

- `allowLocalRequests`: If set to true, will not block request from [Private IP Ranges](https://de.wikipedia.org/wiki/Private_IP-Adresse)
- `logLocalRequests`: If set to true, will log every connection from any IP in the private IP range
- `api`: API URI used for querying the country associated with the connecting IP
- `countries`: list of allowed countries
- `blackListMode`: set to `false` so the plugin is running in `whitelist mode`

````yml
my-GeoBlock:
    plugin:
        GeoBlock:
            silentStartUp: false
            allowLocalRequests: false
            logLocalRequests: false
            logAllowedRequests: false
            logApiRequests: false
            api: "https://get.geojs.io/v1/ip/country/{ip}"
            apiTimeoutMs: 750                                 # optional
            cacheSize: 15
            forceMonthlyUpdate: false
            allowUnknownCountries: false
            unknownCountryApiResponse: "nil"
            blackListMode: false
            addCountryHeader: false
            countries:
                - AF # Afghanistan
                - AL # Albania
                - DZ # Algeria
                - AS # American Samoa
                - AD # Andorra
                - AO # Angola
                - AI # Anguilla
                - AQ # Antarctica
                - AG # Antigua and Barbuda
                - AR # Argentina
                - AM # Armenia
                - AW # Aruba
                - AU # Australia
                - AT # Austria
                - AZ # Azerbaijan
                - BS # Bahamas (the)
                - BH # Bahrain
                - BD # Bangladesh
                - BB # Barbados
                - BY # Belarus
                - BE # Belgium
                - BZ # Belize
                - BJ # Benin
                - BM # Bermuda
                - BT # Bhutan
                - BO # Bolivia (Plurinational State of)
                - BQ # Bonaire, Sint Eustatius and Saba
                - BA # Bosnia and Herzegovina
                - BW # Botswana
                - BV # Bouvet Island
                - BR # Brazil
                - IO # British Indian Ocean Territory (the)
                - BN # Brunei Darussalam
                - BG # Bulgaria
                - BF # Burkina Faso
                - BI # Burundi
                - CV # Cabo Verde
                - KH # Cambodia
                - CM # Cameroon
                - CA # Canada
                - KY # Cayman Islands (the)
                - CF # Central African Republic (the)
                - TD # Chad
                - CL # Chile
                - CN # China
                - CX # Christmas Island
                - CC # Cocos (Keeling) Islands (the)
                - CO # Colombia
                - KM # Comoros (the)
                - CD # Congo (the Democratic Republic of the)
                - CG # Congo (the)
                - CK # Cook Islands (the)
                - CR # Costa Rica
                - HR # Croatia
                - CU # Cuba
                - CW # Curaçao
                - CY # Cyprus
                - CZ # Czechia
                - CI # Côte d'Ivoire
                - DK # Denmark
                - DJ # Djibouti
                - DM # Dominica
                - DO # Dominican Republic (the)
                - EC # Ecuador
                - EG # Egypt
                - SV # El Salvador
                - GQ # Equatorial Guinea
                - ER # Eritrea
                - EE # Estonia
                - SZ # Eswatini
                - ET # Ethiopia
                - FK # Falkland Islands (the) [Malvinas]
                - FO # Faroe Islands (the)
                - FJ # Fiji
                - FI # Finland
                - FR # France
                - GF # French Guiana
                - PF # French Polynesia
                - TF # French Southern Territories (the)
                - GA # Gabon
                - GM # Gambia (the)
                - GE # Georgia
                - DE # Germany
                - GH # Ghana
                - GI # Gibraltar
                - GR # Greece
                - GL # Greenland
                - GD # Grenada
                - GP # Guadeloupe
                - GU # Guam
                - GT # Guatemala
                - GG # Guernsey
                - GN # Guinea
                - GW # Guinea-Bissau
                - GY # Guyana
                - HT # Haiti
                - HM # Heard Island and McDonald Islands
                - VA # Holy See (the)
                - HN # Honduras
                - HK # Hong Kong
                - HU # Hungary
                - IS # Iceland
                - IN # India
                - ID # Indonesia
                - IR # Iran (Islamic Republic of)
                - IQ # Iraq
                - IE # Ireland
                - IM # Isle of Man
                - IL # Israel
                - IT # Italy
                - JM # Jamaica
                - JP # Japan
                - JE # Jersey
                - JO # Jordan
                - KZ # Kazakhstan
                - KE # Kenya
                - KI # Kiribati
                - KP # Korea (the Democratic People's Republic of)
                - KR # Korea (the Republic of)
                - KW # Kuwait
                - KG # Kyrgyzstan
                - LA # Lao People's Democratic Republic (the)
                - LV # Latvia
                - LB # Lebanon
                - LS # Lesotho
                - LR # Liberia
                - LY # Libya
                - LI # Liechtenstein
                - LT # Lithuania
                - LU # Luxembourg
                - MO # Macao
                - MG # Madagascar
                - MW # Malawi
                - MY # Malaysia
                - MV # Maldives
                - ML # Mali
                - MT # Malta
                - MH # Marshall Islands (the)
                - MQ # Martinique
                - MR # Mauritania
                - MU # Mauritius
                - YT # Mayotte
                - MX # Mexico
                - FM # Micronesia (Federated States of)
                - MD # Moldova (the Republic of)
                - MC # Monaco
                - MN # Mongolia
                - ME # Montenegro
                - MS # Montserrat
                - MA # Morocco
                - MZ # Mozambique
                - MM # Myanmar
                - NA # Namibia
                - NR # Nauru
                - NP # Nepal
                - NL # Netherlands (the)
                - NC # New Caledonia
                - NZ # New Zealand
                - NI # Nicaragua
                - NE # Niger (the)
                - NG # Nigeria
                - NU # Niue
                - NF # Norfolk Island
                - MP # Northern Mariana Islands (the)
                - NO # Norway
                - OM # Oman
                - PK # Pakistan
                - PW # Palau
                - PS # Palestine, State of
                - PA # Panama
                - PG # Papua New Guinea
                - PY # Paraguay
                - PE # Peru
                - PH # Philippines (the)
                - PN # Pitcairn
                - PL # Poland
                - PT # Portugal
                - PR # Puerto Rico
                - QA # Qatar
                - MK # Republic of North Macedonia
                - RO # Romania
                - RU # Russian Federation (the)
                - RW # Rwanda
                - RE # Réunion
                - BL # Saint Barthélemy
                - SH # Saint Helena, Ascension and Tristan da Cunha
                - KN # Saint Kitts and Nevis
                - LC # Saint Lucia
                - MF # Saint Martin (French part)
                - PM # Saint Pierre and Miquelon
                - VC # Saint Vincent and the Grenadines
                - WS # Samoa
                - SM # San Marino
                - ST # Sao Tome and Principe
                - SA # Saudi Arabia
                - SN # Senegal
                - RS # Serbia
                - SC # Seychelles
                - SL # Sierra Leone
                - SG # Singapore
                - SX # Sint Maarten (Dutch part)
                - SK # Slovakia
                - SI # Slovenia
                - SB # Solomon Islands
                - SO # Somalia
                - ZA # South Africa
                - GS # South Georgia and the South Sandwich Islands
                - SS # South Sudan
                - ES # Spain
                - LK # Sri Lanka
                - SD # Sudan (the)
                - SR # Suriname
                - SJ # Svalbard and Jan Mayen
                - SE # Sweden
                - CH # Switzerland
                - SY # Syrian Arab Republic
                - TW # Taiwan (Province of China)
                - TJ # Tajikistan
                - TZ # Tanzania, United Republic of
                - TH # Thailand
                - TL # Timor-Leste
                - TG # Togo
                - TK # Tokelau
                - TO # Tonga
                - TT # Trinidad and Tobago
                - TN # Tunisia
                - TR # Turkey
                - TM # Turkmenistan
                - TC # Turks and Caicos Islands (the)
                - TV # Tuvalu
                - UG # Uganda
                - UA # Ukraine
                - AE # United Arab Emirates (the)
                - GB # United Kingdom of Great Britain and Northern Ireland (the)
                - UM # United States Minor Outlying Islands (the)
                - US # United States of America (the)
                - UY # Uruguay
                - UZ # Uzbekistan
                - VU # Vanuatu
                - VE # Venezuela (Bolivarian Republic of)
                - VN # Viet Nam
                - VG # Virgin Islands (British)
                - VI # Virgin Islands (U.S.)
                - WF # Wallis and Futuna
                - EH # Western Sahara
                - YE # Yemen
                - ZM # Zambia
                - ZW # Zimbabwe
                - AX # Åland Islands
````

## Configuration options

### Silent start-up: `silentStartUp`

If set to true, the configuration is not written to the output upon the start-up of the plugin.

### Allow local requests: `allowLocalRequests`

If set to true, will not block request from [Private IP Ranges](https://en.wikipedia.org/wiki/Private_network).

### Log local requests: `logLocalRequests`

If set to true, will show a log message when some one accesses the service over a private ip address.

### Log allowed requests `logAllowedRequests`

If set to true, will show a log message with the IP and the country of origin if a request is allowed.

### Log API requests `logApiRequests`

If set to true, will show a log message for every API hit.

### API `api`

Defines the API URL for the IP to Country resolution. The IP to fetch can be added with `{ip}` to the URL.

### API Timeout `apiTimeoutMs`

Timeout for the call to the api uri.

### Ignore the API timeout error `ignoreAPITimeout`

If the `ignoreAPITimeout` option is set to `true`, a request is allowed even if the API could not be reached.

### Set custom HTTP header field to retrieve the country code from `ipGeolocationHttpHeaderField`

Allow setting the name of a custom HTTP header field to retrieve the country code from. E.g. `cf-ipcountry` for Cloudflare.

### Cache size `cacheSize`

Defines the max size of the [LRU](https://en.wikipedia.org/wiki/Cache_replacement_policies#Least_recently_used_(LRU)) (least recently used) cache.

### Force monthly update `forceMonthlyUpdate`

Even if an IP stays in the cache for a period of a month (about 30 x 24 hours), it must be fetch again after a month.

### Allow unknown countries `allowUnknownCountries`

Some IP addresses have no country associated with them. If this option is set to true, all IPs with no associated country are also allowed.  

### Unknown country api response `unknownCountryApiResponse`

The API uri can be customized. This options allows to customize the response string of the API when a IP with no associated country is requested.

### Black list mode `blackListMode`

When set to `true` the filter logic is inverted, i.e. requests originating from countries listed in the [`countries`](#countries-countries) list are **blocked**. Default: `false`.

### Countries `countries`

A list of country codes from which connections to the service should be allowed. Logic can be inverted by using the [`blackListMode`](#black-list-mode-blacklistmode).

### Allowed IP addresses `allowedIPAddresses`

A list of explicitly allowed IP addresses or IP address ranges. IP addresses and ranges added to this list will always be allowed.

```yaml
allowedIPAddresses:
  - 192.0.2.10          # single IPv4 address
  - 203.0.113.0/24      # IPv4 range in CIDR format  
  - 2001:db8:1234:/48   # IPv6 range in CIDR format
```

### Add Header to request with Country Code: `addCountryHeader`

If set to `true`, adds the X-IPCountry header to the HTTP request header. The header contains the two letter country code returned by cache or API request.

### Customize denied request status code `httpStatusCodeDeniedRequest`

Allows customizing the HTTP status code returned if the request was denied.

### Customize denied request HTTP headers `httpAddHeadersDeniedRequest`

If defined, adds HTTP headers to denied requests.  You can pair this with `httpStatusCodeDeniedRequest` to redirect blocked sessions.
Multiple headers can be seperated by a `;`

```yaml
httpStatusCodeDeniedRequest: 302
httpAddHeadersDeniedRequest: Location=https://blockedyo.com;Bad=yes
```

### Define a custom log file `logFilePath`

Allows to define a target for the logs of the middleware. The path must look like the following: `logFilePath: "/log/geoblock.log"`. Make sure the folder is writeable.
