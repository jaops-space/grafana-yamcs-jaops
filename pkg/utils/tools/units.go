package tools

import (
	"strings"
	"unicode"
)

// ConvertYamcsUnitToGrafanaUnit maps common Yamcs/XTCE engineering unit
// strings to Grafana unit ids.
//
// Grafana unit ids are not always the same as the rendered unit symbol. The
// important example is "m": Yamcs/XTCE commonly uses it for meters, while in
// Grafana "m" is the time unit minute. Grafana's meter id is "lengthm".
//
// Unknown units are returned unchanged, so project-specific units still show as
// custom unit text instead of being hidden.
func ConvertYamcsUnitToGrafanaUnit(unit string) string {
	trimmed := strings.TrimSpace(unit)
	if trimmed == "" {
		return ""
	}

	if grafanaUnit, ok := yamcsCaseSensitiveUnitAliases[trimmed]; ok {
		return grafanaUnit
	}

	if grafanaUnit, ok := yamcsUnitAliases[normalizeYamcsUnit(trimmed)]; ok {
		return grafanaUnit
	}

	return trimmed
}

func normalizeYamcsUnit(unit string) string {
	unit = strings.TrimSpace(unit)
	unit = strings.ReplaceAll(unit, "µ", "u")
	unit = strings.ReplaceAll(unit, "μ", "u")
	unit = strings.ReplaceAll(unit, "²", "^2")
	unit = strings.ReplaceAll(unit, "³", "^3")
	unit = strings.ReplaceAll(unit, "·", "*")
	unit = strings.ReplaceAll(unit, "°", "deg")

	var b strings.Builder
	for _, r := range unit {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

var yamcsCaseSensitiveUnitAliases = map[string]string{
	// SI prefixes are case-sensitive. Keep these exact aliases before the
	// lower-case fallback so mW (milliwatt) does not become MW (megawatt), and
	// H (henry) does not become h (hour).
	"W":  "watt",
	"kW": "kwatt",
	"MW": "megwatt",
	"GW": "gwatt",
	"mW": "mwatt",

	"Wh":  "watth",
	"kWh": "kwatth",
	"MWh": "mwatth",
	"Ah":  "amph",
	"kAh": "kamph",
	"mAh": "mamph",

	"A":  "amp",
	"kA": "kamp",
	"mA": "mamp",
	"V":  "volt",
	"kV": "kvolt",
	"mV": "mvolt",

	"Ω":  "ohm",
	"kΩ": "kohm",
	"MΩ": "Mohm",
	"mΩ": "mohm",

	"F":  "farad",
	"µF": "µfarad",
	"μF": "µfarad",
	"uF": "µfarad",
	"nF": "nfarad",
	"pF": "pfarad",
	"fF": "ffarad",

	"H":  "henry",
	"mH": "mhenry",
	"µH": "µhenry",
	"μH": "µhenry",
	"uH": "µhenry",

	"Hz":  "hertz",
	"kHz": "rotkhz",
	"MHz": "rotmhz",
	"GHz": "rotghz",

	"B": "bytes",

	"°C": "celsius",
	"°F": "fahrenheit",
}

var yamcsUnitAliases = map[string]string{
	// Misc.
	"%":       "percent",
	"percent": "percent",
	"%h":      "humidity",
	"db":      "dB",
	"cd":      "candela",
	"px":      "pixel",
	"bool":    "bool",
	"boolean": "bool",

	// Acceleration.
	"m/s^2":     "accMS2",
	"m/sec^2":   "accMS2",
	"meter/s^2": "accMS2",
	"metre/s^2": "accMS2",
	"ms^-2":     "accMS2",
	"ft/s^2":    "accFS2",
	"feet/s^2":  "accFS2",
	"gee":       "accG",
	"g0":        "accG",

	// Angles and rotation.
	"deg":       "degree",
	"degree":    "degree",
	"degrees":   "degree",
	"rad":       "radian",
	"radian":    "radian",
	"radians":   "radian",
	"grad":      "grad",
	"arcmin":    "arcmin",
	"arcsec":    "arcsec",
	"rpm":       "rotrpm",
	"rev/min":   "rotrpm",
	"r/min":     "rotrpm",
	"hz":        "hertz",
	"khz":       "rotkhz",
	"mhz":       "rotmhz",
	"ghz":       "rotghz",
	"rad/s":     "rotrads",
	"deg/s":     "rotdegs",
	"degree/s":  "rotdegs",
	"degrees/s": "rotdegs",

	// Area.
	"m^2":  "areaM2",
	"sqm":  "areaM2",
	"ft^2": "areaF2",
	"mi^2": "areaMI2",
	"ac":   "acres",
	"acre": "acres",
	"ha":   "hectares",

	// Data and data rate.
	"b":        "bits",
	"byte":     "bytes",
	"bytes":    "bytes",
	"bit":      "bits",
	"bits":     "bits",
	"b/s":      "Bps",
	"byte/s":   "Bps",
	"bytes/s":  "Bps",
	"bit/s":    "bps",
	"bits/s":   "bps",
	"kb/s":     "KBs",
	"kbyte/s":  "KBs",
	"kbytes/s": "KBs",
	"mb/s":     "MBs",
	"mbyte/s":  "MBs",
	"mbytes/s": "MBs",
	"gb/s":     "GBs",
	"gbyte/s":  "GBs",
	"gbytes/s": "GBs",
	"packet/s": "pps",
	"pkt/s":    "pps",

	// Electrical / energy.
	"w":        "watt",
	"watt":     "watt",
	"kw":       "kwatt",
	"mwatt":    "mwatt",
	"mw":       "megwatt",
	"gw":       "gwatt",
	"w/m^2":    "Wm2",
	"va":       "voltamp",
	"kva":      "kvoltamp",
	"var":      "voltampreact",
	"kvar":     "kvoltampreact",
	"wh":       "watth",
	"kwh":      "kwatth",
	"mwh":      "mwatth",
	"ah":       "amph",
	"kah":      "kamph",
	"mah":      "mamph",
	"j":        "joule",
	"joule":    "joule",
	"ev":       "ev",
	"a":        "amp",
	"amp":      "amp",
	"ampere":   "amp",
	"ka":       "kamp",
	"ma":       "mamp",
	"v":        "volt",
	"volt":     "volt",
	"kv":       "kvolt",
	"mv":       "mvolt",
	"dbm":      "dBm",
	"ohm":      "ohm",
	"ω":        "ohm",
	"kohm":     "kohm",
	"kω":       "kohm",
	"megaohm":  "Mohm",
	"milliohm": "mohm",
	"f":        "farad",
	"farad":    "farad",
	"uf":       "µfarad",
	"ufarad":   "µfarad",
	"nf":       "nfarad",
	"pf":       "pfarad",
	"ff":       "ffarad",
	"h":        "h",
	"henry":    "henry",
	"mh":       "mhenry",
	"uh":       "µhenry",
	"lm":       "lumens",

	// Flow / volume.
	"gpm":      "flowgpm",
	"m^3/s":    "flowcms",
	"m3/s":     "flowcms",
	"ft^3/s":   "flowcfs",
	"ft3/s":    "flowcfs",
	"ft^3/min": "flowcfm",
	"ft3/min":  "flowcfm",
	"l/h":      "litreh",
	"l/min":    "flowlpm",
	"ml/min":   "flowmlpm",
	"lx":       "lux",
	"ml":       "mlitre",
	"l":        "litre",
	"litre":    "litre",
	"liter":    "litre",
	"m^3":      "m3",
	"m3":       "m3",
	"nm^3":     "Nm3",
	"nm3":      "Nm3",
	"dm^3":     "dm3",
	"dm3":      "dm3",
	"gal":      "gallons",

	// Force.
	"nm":  "forceNm",
	"knm": "forcekNm",
	"n":   "forceN",
	"kn":  "forcekN",

	// Length. "m" is intentionally mapped to Grafana's meter id, not the
	// Grafana minute id.
	"mm":     "lengthmm",
	"cm":     "lengthcm",
	"in":     "lengthin",
	"inch":   "lengthin",
	"ft":     "lengthft",
	"foot":   "lengthft",
	"feet":   "lengthft",
	"m":      "lengthm",
	"meter":  "lengthm",
	"meters": "lengthm",
	"metre":  "lengthm",
	"metres": "lengthm",
	"km":     "lengthkm",
	"mi":     "lengthmi",

	// Mass.
	"mg": "massmg",
	"g":  "massg",
	"kg": "masskg",
	"lb": "masslb",
	"t":  "masst",

	// Pressure.
	"mbar": "pressurembar",
	"bar":  "pressurebar",
	"kbar": "pressurekbar",
	"pa":   "pressurepa",
	"hpa":  "pressurehpa",
	"kpa":  "pressurekpa",
	"inhg": "pressurehg",
	"psi":  "pressurepsi",

	// Temperature.
	"degc":       "celsius",
	"celsius":    "celsius",
	"degf":       "fahrenheit",
	"fahrenheit": "fahrenheit",
	"k":          "kelvin",
	"kelvin":     "kelvin",

	// Time. These are real Grafana time unit ids.
	"ns":    "ns",
	"us":    "µs",
	"ms":    "ms",
	"s":     "s",
	"sec":   "s",
	"secs":  "s",
	"min":   "m",
	"mins":  "m",
	"hr":    "h",
	"hrs":   "h",
	"hour":  "h",
	"hours": "h",
	"day":   "d",
	"days":  "d",

	// Velocity.
	"m/s":     "velocityms",
	"ms^-1":   "velocityms",
	"meter/s": "velocityms",
	"metre/s": "velocityms",
	"km/h":    "velocitykmh",
	"kph":     "velocitykmh",
	"mph":     "velocitymph",
	"kt":      "velocityknot",
	"knot":    "velocityknot",
	"knots":   "velocityknot",
}
