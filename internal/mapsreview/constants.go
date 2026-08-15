package mapsreview

import "strings"

const (
	OutputDir     = "output"
	ResultsJSON   = "output/places.json"
	ResultsCSV    = "output/places.csv"
	DiscoveryJSON = "output/discovery.json"
	MetadataJSON  = "output/metadata.json"
)

var DefaultCity = "Leverkusen"

func IsDefaultCity(city string) bool {
	return strings.EqualFold(strings.TrimSpace(city), DefaultCity)
}

var LeverkusenPostcodes = []string{
	"51371", "51373", "51375", "51377", "51379", "51381",
}

// NurembergPostcodes remains for backward compatibility in tests
var NurembergPostcodes = LeverkusenPostcodes

var DefaultQueries = []string{
	// Gastro (original)
	"restaurant", "café", "imbiss", "pizzeria", "bäckerei",
	"döner", "burger", "sushi", "schnitzel", "frühstück", "brunch",
	// Bars & Nightlife
	"bar", "kneipe", "pub", "biergarten", "brauerei",
	"cocktail bar", "lounge", "weinstube",
	"club", "nachtclub", "diskothek",
	// Hotels
	"hotel",
	// Beauty & Wellness
	"friseur", "barbier", "barbershop",
	"fitnessstudio", "fitness",
	// Shopping & Daily
	"supermarkt", "metzgerei",
	"apotheke",
	// Services
	"tankstelle",
}

var NurembergPostcodeSet = func() map[string]bool {
	set := make(map[string]bool, len(NurembergPostcodes))
	for _, postcode := range NurembergPostcodes {
		set[postcode] = true
	}
	return set
}()

var LeverkusenPostcodeSet = NurembergPostcodeSet
