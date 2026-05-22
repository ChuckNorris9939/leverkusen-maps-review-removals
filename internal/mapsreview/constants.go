package mapsreview

import "strings"

const (
	OutputDir     = "output"
	ResultsJSON   = "output/places.json"
	ResultsCSV    = "output/places.csv"
	DiscoveryJSON = "output/discovery.json"
	MetadataJSON  = "output/metadata.json"
)

var DefaultCity = "Nürnberg"

func IsDefaultCity(city string) bool {
	return strings.EqualFold(strings.TrimSpace(city), DefaultCity)
}

var NurembergPostcodes = []string{
	"90402", "90403", "90408", "90409", "90411", "90419", "90425", "90427",
	"90429", "90431", "90439", "90441", "90443", "90449", "90451", "90453",
	"90455", "90459", "90461", "90469", "90471", "90473", "90475", "90478",
	"90480", "90482", "90489", "90491",
}

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
