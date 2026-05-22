package main

import (
	"strings"
	"testing"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

func TestMakeHTMLEscapesCity(t *testing.T) {
	page := makeHTML(args{City: `<script>alert("x")</script>`}, nil)
	if strings.Contains(page, `<script>alert`) {
		t.Fatal("city was inserted as raw HTML")
	}
	if !strings.Contains(page, `&lt;script&gt;alert`) {
		t.Fatal("escaped city is missing from HTML")
	}
}

func TestMakeHTMLCustomCityOmitsNurembergBoundaries(t *testing.T) {
	page := makeHTML(args{City: "Fürth"}, nil)
	if !strings.Contains(page, `<script id="bezirkData" type="application/json">[]</script>`) {
		t.Fatal("custom-city dashboard should not include Nürnberg district boundaries")
	}
}

func TestParseArgsRejectsEmptyCity(t *testing.T) {
	if _, err := parseArgs([]string{"--city"}); err == nil {
		t.Fatal("parseArgs(--city) succeeded, want error")
	}
}

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
