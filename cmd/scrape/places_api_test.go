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

func TestLoadDotEnvSetsUnsetValues(t *testing.T) {
	t.Setenv("GOOGLE_MAPS_API_KEY", "")
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# comment\nGOOGLE_MAPS_API_KEY=test-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOOGLE_MAPS_API_KEY"); got != "test-key" {
		t.Fatalf("GOOGLE_MAPS_API_KEY = %q, want test-key", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingEnv(t *testing.T) {
	t.Setenv("GOOGLE_MAPS_API_KEY", "from-env")
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GOOGLE_MAPS_API_KEY=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOOGLE_MAPS_API_KEY"); got != "from-env" {
		t.Fatalf("GOOGLE_MAPS_API_KEY = %q, want from-env", got)
	}
}
