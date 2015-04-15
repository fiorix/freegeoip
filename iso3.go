package freegeoip

var IsoAlpha3 = map[string]string{
	"AD": "AND", // Andorra
	"AE": "ARE", // United Arab Emirates
	"AF": "AFG", // Afghanistan
	"AG": "ATG", // Antigua and Barbuda
	"AI": "AIA", // Anguilla
	"AL": "ALB", // Albania
	"AM": "ARM", // Armenia
	"AO": "AGO", // Angola
	"AQ": "ATA", // Antarctica
	"AR": "ARG", // Argentina
	"AS": "ASM", // American Samoa
	"AT": "AUT", // Austria
	"AU": "AUS", // Australia
	"AW": "ABW", // Aruba
	"AX": "ALA", // Aland Islands
	"AZ": "AZE", // Azerbaijan
	"BA": "BIH", // Bosnia and Herzegovina
	"BB": "BRB", // Barbados
	"BD": "BGD", // Bangladesh
	"BE": "BEL", // Belgium
	"BF": "BFA", // Burkina Faso
	"BG": "BGR", // Bulgaria
	"BH": "BHR", // Bahrain
	"BI": "BDI", // Burundi
	"BJ": "BEN", // Benin
	"BL": "BLM", // Saint Barthelemy
	"BM": "BMU", // Bermuda
	"BN": "BRN", // Brunei
	"BO": "BOL", // Bolivia
	"BQ": "BES", // Bonaire, Saint Eustatius and Saba
	"BR": "BRA", // Brazil
	"BS": "BHS", // Bahamas
	"BT": "BTN", // Bhutan
	"BV": "BVT", // Bouvet Island
	"BW": "BWA", // Botswana
	"BY": "BLR", // Belarus
	"BZ": "BLZ", // Belize
	"CA": "CAN", // Canada
	"CC": "CCK", // Cocos Islands
	"CD": "COD", // Democratic Republic of the Congo
	"CF": "CAF", // Central African Republic
	"CG": "COG", // Republic of the Congo
	"CH": "CHE", // Switzerland
	"CI": "CIV", // Ivory Coast
	"CK": "COK", // Cook Islands
	"CL": "CHL", // Chile
	"CM": "CMR", // Cameroon
	"CN": "CHN", // China
	"CO": "COL", // Colombia
	"CR": "CRI", // Costa Rica
	"CU": "CUB", // Cuba
	"CV": "CPV", // Cape Verde
	"CW": "CUW", // Curacao
	"CX": "CXR", // Christmas Island
	"CY": "CYP", // Cyprus
	"CZ": "CZE", // Czech Republic
	"DE": "DEU", // Germany
	"DJ": "DJI", // Djibouti
	"DK": "DNK", // Denmark
	"DM": "DMA", // Dominica
	"DO": "DOM", // Dominican Republic
	"DZ": "DZA", // Algeria
	"EC": "ECU", // Ecuador
	"EE": "EST", // Estonia
	"EG": "EGY", // Egypt
	"EH": "ESH", // Western Sahara
	"ER": "ERI", // Eritrea
	"ES": "ESP", // Spain
	"ET": "ETH", // Ethiopia
	"FI": "FIN", // Finland
	"FJ": "FJI", // Fiji
	"FK": "FLK", // Falkland Islands
	"FM": "FSM", // Micronesia
	"FO": "FRO", // Faroe Islands
	"FR": "FRA", // France
	"GA": "GAB", // Gabon
	"GB": "GBR", // United Kingdom
	"GD": "GRD", // Grenada
	"GE": "GEO", // Georgia
	"GF": "GUF", // French Guiana
	"GG": "GGY", // Guernsey
	"GH": "GHA", // Ghana
	"GI": "GIB", // Gibraltar
	"GL": "GRL", // Greenland
	"GM": "GMB", // Gambia
	"GN": "GIN", // Guinea
	"GP": "GLP", // Guadeloupe
	"GQ": "GNQ", // Equatorial Guinea
	"GR": "GRC", // Greece
	"GS": "SGS", // South Georgia and the South Sandwich Islands
	"GT": "GTM", // Guatemala
	"GU": "GUM", // Guam
	"GW": "GNB", // Guinea-Bissau
	"GY": "GUY", // Guyana
	"HK": "HKG", // Hong Kong
	"HM": "HMD", // Heard Island and McDonald Islands
	"HN": "HND", // Honduras
	"HR": "HRV", // Croatia
	"HT": "HTI", // Haiti
	"HU": "HUN", // Hungary
	"ID": "IDN", // Indonesia
	"IE": "IRL", // Ireland
	"IL": "ISR", // Israel
	"IM": "IMN", // Isle of Man
	"IN": "IND", // India
	"IO": "IOT", // British Indian Ocean Territory
	"IQ": "IRQ", // Iraq
	"IR": "IRN", // Iran
	"IS": "ISL", // Iceland
	"IT": "ITA", // Italy
	"JE": "JEY", // Jersey
	"JM": "JAM", // Jamaica
	"JO": "JOR", // Jordan
	"JP": "JPN", // Japan
	"KE": "KEN", // Kenya
	"KG": "KGZ", // Kyrgyzstan
	"KH": "KHM", // Cambodia
	"KI": "KIR", // Kiribati
	"KM": "COM", // Comoros
	"KN": "KNA", // Saint Kitts and Nevis
	"KP": "PRK", // North Korea
	"KR": "KOR", // South Korea
	"XK": "XKX", // Kosovo
	"KW": "KWT", // Kuwait
	"KY": "CYM", // Cayman Islands
	"KZ": "KAZ", // Kazakhstan
	"LA": "LAO", // Laos
	"LB": "LBN", // Lebanon
	"LC": "LCA", // Saint Lucia
	"LI": "LIE", // Liechtenstein
	"LK": "LKA", // Sri Lanka
	"LR": "LBR", // Liberia
	"LS": "LSO", // Lesotho
	"LT": "LTU", // Lithuania
	"LU": "LUX", // Luxembourg
	"LV": "LVA", // Latvia
	"LY": "LBY", // Libya
	"MA": "MAR", // Morocco
	"MC": "MCO", // Monaco
	"MD": "MDA", // Moldova
	"ME": "MNE", // Montenegro
	"MF": "MAF", // Saint Martin
	"MG": "MDG", // Madagascar
	"MH": "MHL", // Marshall Islands
	"MK": "MKD", // Macedonia
	"ML": "MLI", // Mali
	"MM": "MMR", // Myanmar
	"MN": "MNG", // Mongolia
	"MO": "MAC", // Macao
	"MP": "MNP", // Northern Mariana Islands
	"MQ": "MTQ", // Martinique
	"MR": "MRT", // Mauritania
	"MS": "MSR", // Montserrat
	"MT": "MLT", // Malta
	"MU": "MUS", // Mauritius
	"MV": "MDV", // Maldives
	"MW": "MWI", // Malawi
	"MX": "MEX", // Mexico
	"MY": "MYS", // Malaysia
	"MZ": "MOZ", // Mozambique
	"NA": "NAM", // Namibia
	"NC": "NCL", // New Caledonia
	"NE": "NER", // Niger
	"NF": "NFK", // Norfolk Island
	"NG": "NGA", // Nigeria
	"NI": "NIC", // Nicaragua
	"NL": "NLD", // Netherlands
	"NO": "NOR", // Norway
	"NP": "NPL", // Nepal
	"NR": "NRU", // Nauru
	"NU": "NIU", // Niue
	"NZ": "NZL", // New Zealand
	"OM": "OMN", // Oman
	"PA": "PAN", // Panama
	"PE": "PER", // Peru
	"PF": "PYF", // French Polynesia
	"PG": "PNG", // Papua New Guinea
	"PH": "PHL", // Philippines
	"PK": "PAK", // Pakistan
	"PL": "POL", // Poland
	"PM": "SPM", // Saint Pierre and Miquelon
	"PN": "PCN", // Pitcairn
	"PR": "PRI", // Puerto Rico
	"PS": "PSE", // Palestinian Territory
	"PT": "PRT", // Portugal
	"PW": "PLW", // Palau
	"PY": "PRY", // Paraguay
	"QA": "QAT", // Qatar
	"RE": "REU", // Reunion
	"RO": "ROU", // Romania
	"RS": "SRB", // Serbia
	"RU": "RUS", // Russia
	"RW": "RWA", // Rwanda
	"SA": "SAU", // Saudi Arabia
	"SB": "SLB", // Solomon Islands
	"SC": "SYC", // Seychelles
	"SD": "SDN", // Sudan
	"SS": "SSD", // South Sudan
	"SE": "SWE", // Sweden
	"SG": "SGP", // Singapore
	"SH": "SHN", // Saint Helena
	"SI": "SVN", // Slovenia
	"SJ": "SJM", // Svalbard and Jan Mayen
	"SK": "SVK", // Slovakia
	"SL": "SLE", // Sierra Leone
	"SM": "SMR", // San Marino
	"SN": "SEN", // Senegal
	"SO": "SOM", // Somalia
	"SR": "SUR", // Suriname
	"ST": "STP", // Sao Tome and Principe
	"SV": "SLV", // El Salvador
	"SX": "SXM", // Sint Maarten
	"SY": "SYR", // Syria
	"SZ": "SWZ", // Swaziland
	"TC": "TCA", // Turks and Caicos Islands
	"TD": "TCD", // Chad
	"TF": "ATF", // French Southern Territories
	"TG": "TGO", // Togo
	"TH": "THA", // Thailand
	"TJ": "TJK", // Tajikistan
	"TK": "TKL", // Tokelau
	"TL": "TLS", // East Timor
	"TM": "TKM", // Turkmenistan
	"TN": "TUN", // Tunisia
	"TO": "TON", // Tonga
	"TR": "TUR", // Turkey
	"TT": "TTO", // Trinidad and Tobago
	"TV": "TUV", // Tuvalu
	"TW": "TWN", // Taiwan
	"TZ": "TZA", // Tanzania
	"UA": "UKR", // Ukraine
	"UG": "UGA", // Uganda
	"UM": "UMI", // United States Minor Outlying Islands
	"US": "USA", // United States
	"UY": "URY", // Uruguay
	"UZ": "UZB", // Uzbekistan
	"VA": "VAT", // Vatican
	"VC": "VCT", // Saint Vincent and the Grenadines
	"VE": "VEN", // Venezuela
	"VG": "VGB", // British Virgin Islands
	"VI": "VIR", // U.S. Virgin Islands
	"VN": "VNM", // Vietnam
	"VU": "VUT", // Vanuatu
	"WF": "WLF", // Wallis and Futuna
	"WS": "WSM", // Samoa
	"YE": "YEM", // Yemen
	"YT": "MYT", // Mayotte
	"ZA": "ZAF", // South Africa
	"ZM": "ZMB", // Zambia
	"ZW": "ZWE", // Zimbabwe
	"CS": "SCG", // Serbia and Montenegro
	"AN": "ANT", // Netherlands Antilles
}
