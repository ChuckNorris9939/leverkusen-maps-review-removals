package main

import (
	"os"
	"path/filepath"
	"testing"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

func TestPlacesTextSearchFieldMaskStaysIDOnly(t *testing.T) {
	if placesTextSearchIDOnlyFieldMask != "places.id,nextPageToken" {
		t.Fatalf("Places API discovery must stay ID-only/no-cost; got field mask %q", placesTextSearchIDOnlyFieldMask)
	}
}

func TestDiscoverySeenMatchesScrapedSearchResultAlias(t *testing.T) {
	seen := map[string]bool{}
	markDiscoverySeen(seen, mapsreview.Discovery{
		ID:  "0x479f57a73350aed5:0xef0321790f9cee83",
		URL: "https://www.google.com/maps/place/FranKonya/data=!4m7!3m6!1s0x479f57a73350aed5:0xef0321790f9cee83!8m2!3d49.4471632!4d11.0647079!16s%2Fg%2F11t1h2jrkw!19sChIJ1a5QM6dXn0cRg-6cD3khA-8?authuser=0&hl=de&rclk=1",
	})
	if !discoverySeen(seen, discoveryFromAPIPlace(placesAPIPlace{ID: "ChIJ1a5QM6dXn0cRg-6cD3khA-8"}, "90402", "restaurant", "restaurant 90402 Nürnberg")) {
		t.Fatal("API discovery did not match existing scraped discovery alias")
	}
}

func TestPlacesAPIKeyFromConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[places_api]\napi_key = \"test-key\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := placesAPIKeyFromConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "test-key" {
		t.Fatalf("placesAPIKeyFromConfig = %q, want test-key", got)
	}
}

func TestPlacesAPIKeyFromConfigMissingFile(t *testing.T) {
	got, err := placesAPIKeyFromConfig(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("placesAPIKeyFromConfig missing file = %q, want empty", got)
	}
}
