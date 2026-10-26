package countries

import (
	"regexp"
	"strings"
)

// Country holds metadata for an ISO 3166-1 country.
type Country struct {
	Alpha2 string
	Alpha3 string
	Name   string
}

// All canonical ISO 3166-1 countries.
var countriesList = []Country{
	{"AD", "AND", "Andorra"},
	{"AE", "ARE", "United Arab Emirates"},
	{"AF", "AFG", "Afghanistan"},
	{"AG", "ATG", "Antigua and Barbuda"},
	{"AI", "AIA", "Anguilla"},
	{"AL", "ALB", "Albania"},
	{"AM", "ARM", "Armenia"},
	{"AO", "AGO", "Angola"},
	{"AQ", "ATA", "Antarctica"},
	{"AR", "ARG", "Argentina"},
	{"AS", "ASM", "American Samoa"},
	{"AT", "AUT", "Austria"},
	{"AU", "AUS", "Australia"},
	{"AW", "ABW", "Aruba"},
	{"AX", "ALA", "Åland Islands"},
	{"AZ", "AZE", "Azerbaijan"},
	{"BA", "BIH", "Bosnia and Herzegovina"},
	{"BB", "BRB", "Barbados"},
	{"BD", "BGD", "Bangladesh"},
	{"BE", "BEL", "Belgium"},
	{"BF", "BFA", "Burkina Faso"},
	{"BG", "BGR", "Bulgaria"},
	{"BH", "BHR", "Bahrain"},
	{"BI", "BDI", "Burundi"},
	{"BJ", "BEN", "Benin"},
	{"BL", "BLM", "Saint Barthélemy"},
	{"BM", "BMU", "Bermuda"},
	{"BN", "BRN", "Brunei"},
	{"BO", "BOL", "Bolivia"},
	{"BQ", "BES", "Bonaire, Sint Eustatius and Saba"},
	{"BR", "BRA", "Brazil"},
	{"BS", "BHS", "Bahamas"},
	{"BT", "BTN", "Bhutan"},
	{"BV", "BVT", "Bouvet Island"},
	{"BW", "BWA", "Botswana"},
	{"BY", "BLR", "Belarus"},
	{"BZ", "BLZ", "Belize"},
	{"CA", "CAN", "Canada"},
	{"CC", "CCK", "Cocos (Keeling) Islands"},
	{"CD", "COD", "Democratic Republic of the Congo"},
	{"CF", "CAF", "Central African Republic"},
	{"CG", "COG", "Republic of the Congo"},
	{"CH", "CHE", "Switzerland"},
	{"CI", "CIV", "Ivory Coast"},
	{"CK", "COK", "Cook Islands"},
	{"CL", "CHL", "Chile"},
	{"CM", "CMR", "Cameroon"},
	{"CN", "CHN", "China"},
	{"CO", "COL", "Colombia"},
	{"CR", "CRI", "Costa Rica"},
	{"CU", "CUB", "Cuba"},
	{"CV", "CPV", "Cape Verde"},
	{"CW", "CUW", "Curaçao"},
	{"CX", "CXR", "Christmas Island"},
	{"CY", "CYP", "Cyprus"},
	{"CZ", "CZE", "Czech Republic"},
	{"DE", "DEU", "Germany"},
	{"DJ", "DJI", "Djibouti"},
	{"DK", "DNK", "Denmark"},
	{"DM", "DMA", "Dominica"},
	{"DO", "DOM", "Dominican Republic"},
	{"DZ", "DZA", "Algeria"},
	{"EC", "ECU", "Ecuador"},
	{"EE", "EST", "Estonia"},
	{"EG", "EGY", "Egypt"},
	{"EH", "ESH", "Western Sahara"},
	{"ER", "ERI", "Eritrea"},
	{"ES", "ESP", "Spain"},
	{"ET", "ETH", "Ethiopia"},
	{"FI", "FIN", "Finland"},
	{"FJ", "FJI", "Fiji"},
	{"FK", "FLK", "Falkland Islands"},
	{"FM", "FSM", "Micronesia"},
	{"FO", "FRO", "Faroe Islands"},
	{"FR", "FRA", "France"},
	{"GA", "GAB", "Gabon"},
	{"GB", "GBR", "United Kingdom"},
	{"GD", "GRD", "Grenada"},
	{"GE", "GEO", "Georgia"},
	{"GF", "GUF", "French Guiana"},
	{"GG", "GGY", "Guernsey"},
	{"GH", "GHA", "Ghana"},
	{"GI", "GIB", "Gibraltar"},
	{"GL", "GRL", "Greenland"},
	{"GM", "GMB", "Gambia"},
	{"GN", "GIN", "Guinea"},
	{"GP", "GLP", "Guadeloupe"},
	{"GQ", "GNQ", "Equatorial Guinea"},
	{"GR", "GRC", "Greece"},
	{"GS", "SGS", "South Georgia"},
	{"GT", "GTM", "Guatemala"},
	{"GU", "GUM", "Guam"},
	{"GW", "GNB", "Guinea-Bissau"},
	{"GY", "GUY", "Guyana"},
	{"HK", "HKG", "Hong Kong"},
	{"HM", "HMD", "Heard Island and McDonald Islands"},
	{"HN", "HND", "Honduras"},
	{"HR", "HRV", "Croatia"},
	{"HT", "HTI", "Haiti"},
	{"HU", "HUN", "Hungary"},
	{"ID", "IDN", "Indonesia"},
	{"IE", "IRL", "Ireland"},
	{"IL", "ISR", "Israel"},
	{"IM", "IMN", "Isle of Man"},
	{"IN", "IND", "India"},
	{"IO", "IOT", "British Indian Ocean Territory"},
	{"IQ", "IRQ", "Iraq"},
	{"IR", "IRN", "Iran"},
	{"IS", "ISL", "Iceland"},
	{"IT", "ITA", "Italy"},
	{"JE", "JEY", "Jersey"},
	{"JM", "JAM", "Jamaica"},
	{"JO", "JOR", "Jordan"},
	{"JP", "JPN", "Japan"},
	{"KE", "KEN", "Kenya"},
	{"KG", "KGZ", "Kyrgyzstan"},
	{"KH", "KHM", "Cambodia"},
	{"KI", "KIR", "Kiribati"},
	{"KM", "COM", "Comoros"},
	{"KN", "KNA", "Saint Kitts and Nevis"},
	{"KP", "PRK", "North Korea"},
	{"KR", "KOR", "South Korea"},
	{"KW", "KWT", "Kuwait"},
	{"KY", "CYM", "Cayman Islands"},
	{"KZ", "KAZ", "Kazakhstan"},
	{"LA", "LAO", "Laos"},
	{"LB", "LBN", "Lebanon"},
	{"LC", "LCA", "Saint Lucia"},
	{"LI", "LIE", "Liechtenstein"},
	{"LK", "LKA", "Sri Lanka"},
	{"LR", "LBR", "Liberia"},
	{"LS", "LSO", "Lesotho"},
	{"LT", "LTU", "Lithuania"},
	{"LU", "LUX", "Luxembourg"},
	{"LV", "LVA", "Latvia"},
	{"LY", "LBY", "Libya"},
	{"MA", "MAR", "Morocco"},
	{"MC", "MCO", "Monaco"},
	{"MD", "MDA", "Moldova"},
	{"ME", "MNE", "Montenegro"},
	{"MF", "MAF", "Saint Martin"},
	{"MG", "MDG", "Madagascar"},
	{"MH", "MHL", "Marshall Islands"},
	{"MK", "MKD", "North Macedonia"},
	{"ML", "MLI", "Mali"},
	{"MM", "MMR", "Myanmar"},
	{"MN", "MNG", "Mongolia"},
	{"MO", "MAC", "Macau"},
	{"MP", "MNP", "Northern Mariana Islands"},
	{"MQ", "MTQ", "Martinique"},
	{"MR", "MRT", "Mauritania"},
	{"MS", "MSR", "Montserrat"},
	{"MT", "MLT", "Malta"},
	{"MU", "MUS", "Mauritius"},
	{"MV", "MDV", "Maldives"},
	{"MW", "MWI", "Malawi"},
	{"MX", "MEX", "Mexico"},
	{"MY", "MYS", "Malaysia"},
	{"MZ", "MOZ", "Mozambique"},
	{"NA", "NAM", "Namibia"},
	{"NC", "NCL", "New Caledonia"},
	{"NE", "NER", "Niger"},
	{"NF", "NFK", "Norfolk Island"},
	{"NG", "NGA", "Nigeria"},
	{"NI", "NIC", "Nicaragua"},
	{"NL", "NLD", "Netherlands"},
	{"NO", "NOR", "Norway"},
	{"NP", "NPL", "Nepal"},
	{"NR", "NRU", "Nauru"},
	{"NU", "NIU", "Niue"},
	{"NZ", "NZL", "New Zealand"},
	{"OM", "OMN", "Oman"},
	{"PA", "PAN", "Panama"},
	{"PE", "PER", "Peru"},
	{"PF", "PYF", "French Polynesia"},
	{"PG", "PNG", "Papua New Guinea"},
	{"PH", "PHL", "Philippines"},
	{"PK", "PAK", "Pakistan"},
	{"PL", "POL", "Poland"},
	{"PM", "SPM", "Saint Pierre and Miquelon"},
	{"PN", "PCN", "Pitcairn"},
	{"PR", "PRI", "Puerto Rico"},
	{"PS", "PSE", "Palestine"},
	{"PT", "PRT", "Portugal"},
	{"PW", "PLW", "Palau"},
	{"PY", "PRY", "Paraguay"},
	{"QA", "QAT", "Qatar"},
	{"RE", "REU", "Réunion"},
	{"RO", "ROU", "Romania"},
	{"RS", "SRB", "Serbia"},
	{"RU", "RUS", "Russia"},
	{"RW", "RWA", "Rwanda"},
	{"SA", "SAU", "Saudi Arabia"},
	{"SB", "SLB", "Solomon Islands"},
	{"SC", "SYC", "Seychelles"},
	{"SD", "SDN", "Sudan"},
	{"SE", "SWE", "Sweden"},
	{"SG", "SGP", "Singapore"},
	{"SH", "SHN", "Saint Helena"},
	{"SI", "SVN", "Slovenia"},
	{"SJ", "SJM", "Svalbard and Jan Mayen"},
	{"SK", "SVK", "Slovakia"},
	{"SL", "SLE", "Sierra Leone"},
	{"SM", "SMR", "San Marino"},
	{"SN", "SEN", "Senegal"},
	{"SO", "SOM", "Somalia"},
	{"SR", "SUR", "Suriname"},
	{"SS", "SSD", "South Sudan"},
	{"ST", "STP", "São Tomé and Príncipe"},
	{"SV", "SLV", "El Salvador"},
	{"SX", "SXM", "Sint Maarten"},
	{"SY", "SYR", "Syria"},
	{"SZ", "SWZ", "Eswatini"},
	{"TC", "TCA", "Turks and Caicos Islands"},
	{"TD", "TCD", "Chad"},
	{"TF", "ATF", "French Southern Territories"},
	{"TG", "TGO", "Togo"},
	{"TH", "THA", "Thailand"},
	{"TJ", "TJK", "Tajikistan"},
	{"TK", "TKL", "Tokelau"},
	{"TL", "TLS", "Timor-Leste"},
	{"TM", "TKM", "Turkmenistan"},
	{"TN", "TUN", "Tunisia"},
	{"TO", "TON", "Tonga"},
	{"TR", "TUR", "Turkey"},
	{"TT", "TTO", "Trinidad and Tobago"},
	{"TV", "TUV", "Tuvalu"},
	{"TW", "TWN", "Taiwan"},
	{"TZ", "TZA", "Tanzania"},
	{"UA", "UKR", "Ukraine"},
	{"UG", "UGA", "Uganda"},
	{"UM", "UMI", "United States Minor Outlying Islands"},
	{"US", "USA", "United States"},
	{"UY", "URY", "Uruguay"},
	{"UZ", "UZB", "Uzbekistan"},
	{"VA", "VAT", "Vatican City"},
	{"VC", "VCT", "Saint Vincent and the Grenadines"},
	{"VE", "VEN", "Venezuela"},
	{"VG", "VGB", "British Virgin Islands"},
	{"VI", "VIR", "U.S. Virgin Islands"},
	{"VN", "VNM", "Vietnam"},
	{"VU", "VUT", "Vanuatu"},
	{"WF", "WLF", "Wallis and Futuna"},
	{"WS", "WSM", "Samoa"},
	{"YE", "YEM", "Yemen"},
	{"YT", "MYT", "Mayotte"},
	{"ZA", "ZAF", "South Africa"},
	{"ZM", "ZMB", "Zambia"},
	{"ZW", "ZWE", "Zimbabwe"},
}

// Common aliases, localized names, and alternate spellings mapped to Alpha2.
var countryAliases = map[string]string{
	// United States
	"usa":                      "US",
	"united states of america": "US",
	"america":                  "US",
	"u.s.":                     "US",
	"u.s.a.":                   "US",

	// United Kingdom
	"uk":             "GB",
	"u.k.":           "GB",
	"great britain":  "GB",
	"britain":        "GB",
	"england":        "GB",
	"scotland":       "GB",
	"wales":          "GB",
	"northern ireland": "GB",

	// Belgium
	"belgique": "BE",
	"belgië":   "BE",
	"belgien":  "BE",

	// Netherlands
	"holland":   "NL",
	"nederland": "NL",
	"pays-bas":  "NL",

	// Germany
	"deutschland": "DE",
	"allemagne":   "DE",

	// France
	"francia":    "FR",
	"frankreich": "FR",

	// Spain
	"españa":  "ES",
	"espagne": "ES",
	"spanien": "ES",

	// Italy
	"italia": "IT",
	"italie": "IT",
	"italien": "IT",

	// Switzerland
	"schweiz":  "CH",
	"suisse":   "CH",
	"svizzera": "CH",

	// Austria
	"österreich": "AT",
	"autriche":   "AT",

	// Ireland
	"eire":    "IE",
	"irland":  "IE",
	"irlande": "IE",

	// Russia
	"russian federation": "RU",
	"russie":             "RU",
	"russland":           "RU",

	// China
	"people's republic of china": "CN",
	"prc":                        "CN",

	// South Korea
	"korea":             "KR",
	"republic of korea": "KR",

	// Czech Republic
	"czechia":        "CZ",
	"tschechien":     "CZ",
	"république tchèque": "CZ",

	// Turkey
	"türkiye": "TR",
	"turkei":  "TR",
	"turquie": "TR",

	// Poland
	"polska": "PL",
	"pologne": "PL",
	"polen":   "PL",

	// Sweden
	"sverige": "SE",
	"suède":   "SE",
	"schweden": "SE",

	// Norway
	"norge":    "NO",
	"norvège":  "NO",
	"norwegen": "NO",

	// Denmark
	"danmark":  "DK",
	"danemark": "DK",
	"dänemark": "DK",

	// Finland
	"suomi":    "FI",
	"finlande": "FI",
	"finnland": "FI",

	// Greece
	"hellas": "GR",
	"grèce":  "GR",
	"griechenland": "GR",

	// Portugal
	"portogallo": "PT",

	// Brazil
	"brasil": "BR",
	"brésil": "BR",

	// UAE
	"uae": "AE",
	"emirates": "AE",
}

var (
	alpha2Map = make(map[string]Country)
	alpha3Map = make(map[string]string)
	nameMap   = make(map[string]string)
)

func init() {
	for _, c := range countriesList {
		alpha2Map[c.Alpha2] = c
		alpha3Map[c.Alpha3] = c.Alpha2
		nameMap[strings.ToLower(c.Name)] = c.Alpha2
	}
	for alias, code := range countryAliases {
		nameMap[strings.ToLower(alias)] = code
	}
}

// cleanInput trims whitespace and common surrounding quotes/brackets.
func cleanInput(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'“”‘’[]()`)
	return strings.TrimSpace(s)
}

// NormalizeCode takes any country representation (2-letter ISO code, 3-letter ISO code,
// English country name, common alias, or local name) and resolves it to a canonical
// 2-letter uppercase ISO 3166-1 alpha-2 code (e.g. "BE", "FR", "US").
// Returns (code, true) if resolved, or ("", false) if not recognized.
func NormalizeCode(input string) (string, bool) {
	s := cleanInput(input)
	if s == "" {
		return "", false
	}

	upper := strings.ToUpper(s)
	// Check if already valid Alpha-2
	if _, ok := alpha2Map[upper]; ok {
		return upper, true
	}

	// Check Alpha-3
	if a2, ok := alpha3Map[upper]; ok {
		return a2, true
	}

	// Check name and aliases (case-insensitive)
	lower := strings.ToLower(s)
	if a2, ok := nameMap[lower]; ok {
		return a2, true
	}

	return "", false
}

// Name returns the canonical English display name for an ISO 3166-1 alpha-2 code (e.g. "BE" -> "Belgium").
// If the code is not recognized, it returns the input code as-is.
func Name(code string) string {
	upper := strings.ToUpper(strings.TrimSpace(code))
	if c, ok := alpha2Map[upper]; ok {
		return c.Name
	}
	return upper
}

var (
	// Matches "<query> in <country>", "<query>, <country>", "<query> - <country>"
	reInCountry    = regexp.MustCompile(`(?i)^(.*?)\s+(?:in|within|inside|near)\s+([a-zA-Z\s\.\-']+)$`)
	reCommaCountry = regexp.MustCompile(`(?i)^(.*?)\s*,\s*([a-zA-Z\s\.\-']+)$`)
)

// ExtractFromQuery inspects a free-text search query and extracts any referenced country.
// It handles patterns like:
// - "airports in belgium" -> cleaned: "airports", country: "BE", found: true
// - "brussels, belgium"   -> cleaned: "brussels", country: "BE", found: true
// - "hotels in france"    -> cleaned: "hotels",   country: "FR", found: true
// - "belgium"             -> cleaned: "",         country: "BE", found: true
// - "belgium airports"    -> cleaned: "airports", country: "BE", found: true
// - "brussels"            -> cleaned: "brussels", country: "",   found: false
func ExtractFromQuery(q string) (cleanedQuery string, countryCode string, found bool) {
	raw := cleanInput(q)
	if raw == "" {
		return "", "", false
	}

	// 1. Is the entire query a country name or code?
	if code, ok := NormalizeCode(raw); ok {
		return "", code, true
	}

	// 2. Pattern: "<something> in <country>"
	if matches := reInCountry.FindStringSubmatch(raw); len(matches) == 3 {
		targetQuery := strings.TrimSpace(matches[1])
		countryPart := strings.TrimSpace(matches[2])
		if code, ok := NormalizeCode(countryPart); ok {
			return targetQuery, code, true
		}
	}

	// 3. Pattern: "<something>, <country>"
	if matches := reCommaCountry.FindStringSubmatch(raw); len(matches) == 3 {
		targetQuery := strings.TrimSpace(matches[1])
		countryPart := strings.TrimSpace(matches[2])
		if code, ok := NormalizeCode(countryPart); ok {
			return targetQuery, code, true
		}
	}

	// 4. Pattern: "<country> <something>" (e.g. "belgium airports", "france hotels")
	words := strings.Fields(raw)
	if len(words) >= 2 {
		// Try first 1, 2, or 3 words as country name
		for i := 1; i <= 3 && i < len(words); i++ {
			candidate := strings.Join(words[:i], " ")
			if code, ok := NormalizeCode(candidate); ok {
				rest := strings.TrimSpace(strings.Join(words[i:], " "))
				return rest, code, true
			}
		}
	}

	return raw, "", false
}
