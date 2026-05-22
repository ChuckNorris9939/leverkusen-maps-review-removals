package main

import (
	"strings"
	"testing"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

func TestMakeClientRowsSkipsRowsWithoutRating(t *testing.T) {
	rows := []mapsreview.Place{
		{ID: "with-rating", Name: "Rated", Rating: mapsreview.FloatPtr(4.5), ReviewCount: mapsreview.IntPtr(10), Status: "success"},
		{ID: "no-rating", Name: "No rating", Rating: nil, ReviewCount: mapsreview.IntPtr(0), Status: "success", PlaceState: mapsreview.PlaceStateNoPublicReviews},
	}

	got := makeClientRows(rows)
	if len(got) != 1 {
		t.Fatalf("len(makeClientRows) = %d, want 1", len(got))
	}
	if got[0].ID != "with-rating" {
		t.Fatalf("row ID = %q, want with-rating", got[0].ID)
	}
}

func TestMakeHTMLIncludesSEOMetadataAndSummary(t *testing.T) {
	data := []clientRow{{
		ID:              "test-place",
		Name:            "Café <Test>",
		Postcode:        "90402",
		Rating:          mapsreview.FloatPtr(4.5),
		ReviewCount:     mapsreview.IntPtr(120),
		HasBanner:       true,
		RemovedRange:    "21 bis 50",
		RemovedEstimate: 35.5,
		URL:             "https://example.com/maps",
		ReadAt:          "2026-05-04T02:20:07Z",
	}}

	html := makeHTML(data)
	checks := []string{
		`<meta name="description" content="Interaktives Nürnberg-Dashboard`,
		`<link rel="canonical" href="https://nuernberg-maps-review-removals.patwoz.dev/">`,
		`<meta property="og:type" content="website">`,
		`<script type="application/ld+json">`,
		`<h1 class="hero-title">Nürnberg Google-Maps-Bewertungen</h1>`,
		`Eigene Bewertung entfernt?`,
		`Google-Einspruch`,
		`Dashboard basiert auf der MIT-lizenzierten Vorlage von Patrick Wozniak.`,
		`Einordnung der Löschquote`,
		`ratioHistogram`,
		`Weitere regionale Dashboards`,
		`https://bewertungsradar-saar.de/`,
		`https://berlintrustindex.com/`,
		`Top-Orte nach geschätzten entfernten Bewertungen`,
		`Café &lt;Test&gt;`,
	}
	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Fatalf("makeHTML missing %q", check)
		}
	}
	for _, unwanted := range []string{`Streitbeilegung`, `platform-control.com`} {
		if strings.Contains(html, unwanted) {
			t.Fatalf("makeHTML contains unwanted %q", unwanted)
		}
	}
}
