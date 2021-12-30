# GeoBlock
Simple plugin for [Traefik](https://github.com/containous/traefik) to block request based on their country of origin. Uses [GeoJs.io](https://www.geojs.io/).

## Sample configuration
- `allowlocalrequests`: If set to true, will not block request from [Private IP Ranges](https://de.wikipedia.org/wiki/Private_IP-Adresse)
- `loglocalrequests`: If set to true, will log every connection from any IP in the private IP range
- `api`: API URI used for querying the country associated with the connecting IP
- `countries`: list of allowed countries

````
my-GeoBlock:
    plugin:
        GeoBlock:
            allowlocalrequests: false
            loglocalrequests: false
            logallowedrequests: false
            logapirequests: false
            api: "https://get.geojs.io/v1/ip/country/{ip}"
            cachesize: 15
            forcemonthlyupdate: false
            allowunknowncountries: false
            unknowncountrycode: "nil"
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
    
### Allow local requests: `allowlocalrequests`
If set to true, will not block request from [Private IP Ranges](https://en.wikipedia.org/wiki/Private_network).

### Log local requests: `loglocalrequests`
If set to true, will show a log message when some one accesses the service over a private ip address.

### Log allowed requests `logallowedrequests`
If set to true, will show a log message with the IP and the country of origin if a request is allowed.

### Log API requests `logapirequests`
If set to true, will show a log message for every API hit.

### API
Defines the API URL for the IP to Country resolution. The IP to fetch can be added with `{ip}` to the URL.

### Cache size `cachesize`
Defines the max size of the [LRU](https://en.wikipedia.org/wiki/Cache_replacement_policies#Least_recently_used_(LRU)) (least recently used) cache.

### Force monthly update `forcemonthlyupdate`
Even if an IP stays in the cache for a period of a month (about 30 x 24 hours), it must be fetch again after a month.

### Allow unknown countries `allowunknowncountries`
Some IP addresses have no country associated with them. If this option is set to true, all IPs with no associated country are also allowed.  

### Unknown country api response`unknowncountryapiresponse`
The API uri can be customized. This options allows to customize the response string of the API when a IP with no associated country is requested.

### Countries
A list of country codes from which connections to the service should be allowed 
