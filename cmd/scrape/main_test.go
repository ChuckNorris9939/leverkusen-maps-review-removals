package main

import (
	"strings"
	"testing"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

func TestIsConsentPage(t *testing.T) {
	if !isConsentPage("Bevor Sie zu Google weitergehen\nAlle akzeptieren") {
		t.Fatal("German consent page not detected")
	}
	if isConsentPage("FranKonya\n4,5\n(173)\nRezensionen") {
		t.Fatal("normal place text detected as consent page")
	}
}

func TestIsRestrictedMapsView(t *testing.T) {
	texts := []string{
		"Die Ansicht ist beschränkt und du siehst nur einen Teil der Google Maps-Daten. Weitere Informationen",
		"This is a limited view and you only see some Google Maps data.",
	}
	for _, text := range texts {
		if !isRestrictedMapsView(text) {
			t.Fatalf("isRestrictedMapsView(%q) = false, want true", text)
		}
	}
	if isRestrictedMapsView("FranKonya\n4,5\n(173)\nRezensionen") {
		t.Fatal("normal place text detected as restricted")
	}
	loadedPlaceWithHiddenMarker := "EAT HAPPY\nÜbersicht\nInfo\nRoutenplaner\nRezension schreiben\nDie Ansicht ist beschränkt"
	if isRestrictedMapsView(loadedPlaceWithHiddenMarker) {
		t.Fatal("loaded place page with hidden restricted marker detected as restricted")
	}
}

func TestParseArgsCustomCityRequiresExplicitPostcodes(t *testing.T) {
	_, err := parseArgs([]string{"--city", "Fürth"})
	if err == nil || !strings.Contains(err.Error(), "--postcodes") {
		t.Fatalf("parseArgs custom city error = %v, want --postcodes error", err)
	}
}

func TestParseArgsCustomCityRejectsAllPostcodes(t *testing.T) {
	_, err := parseArgs([]string{"--city", "Fürth", "--postcodes", "all"})
	if err == nil || !strings.Contains(err.Error(), "explicit --postcodes CSV") {
		t.Fatalf("parseArgs custom city all-postcodes error = %v, want explicit CSV error", err)
	}
}

func TestParseArgsCustomCityPaths(t *testing.T) {
	args, err := parseArgs([]string{"--city", "Fürth", "--postcodes", "90762,90763", "--discovery", "output/fuerth_discovery.json", "--metadata", "output/fuerth_metadata.json"})
	if err != nil {
		t.Fatal(err)
	}
	if args.City != "Fürth" {
		t.Fatalf("City = %q, want Fürth", args.City)
	}
	if got := strings.Join(args.Postcodes, ","); got != "90762,90763" {
		t.Fatalf("Postcodes = %q, want 90762,90763", got)
	}
	if args.Discovery != "output/fuerth_discovery.json" {
		t.Fatalf("Discovery = %q", args.Discovery)
	}
	if args.Metadata != "output/fuerth_metadata.json" {
		t.Fatalf("Metadata = %q", args.Metadata)
	}
}

func TestParseArgsRejectsEmptyCity(t *testing.T) {
	if _, err := parseArgs([]string{"--city"}); err == nil {
		t.Fatal("parseArgs(--city) succeeded, want error")
	}
}

func TestWriteMetadataIncludesCustomPaths(t *testing.T) {
	file := t.TempDir() + "/metadata.json"
	in := args{
		City:      mapsreview.DefaultCity,
		Out:       "custom/places.json",
		CSV:       "custom/places.csv",
		Discovery: "custom/discovery.json",
		Metadata:  file,
	}
	if err := writeMetadata(in, nil, nil); err != nil {
		t.Fatal(err)
	}
	got, err := mapsreview.ReadJSON(file, metadata{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Discovery != in.Discovery || got.Metadata != file {
		t.Fatalf("metadata paths = (%q, %q), want (%q, %q)", got.Discovery, got.Metadata, in.Discovery, file)
	}
}

func TestParseArgsSaveEvery(t *testing.T) {
	args, err := parseArgs([]string{"--save-every", "25"})
	if err != nil {
		t.Fatal(err)
	}
	if args.SaveEvery != 25 {
		t.Fatalf("SaveEvery = %d, want 25", args.SaveEvery)
	}
}

func TestParseArgsPlacesAPI(t *testing.T) {
	args, err := parseArgs([]string{"--places-api-discovery", "--places-api-pages", "2", "--discovery", "output/api.json"})
	if err != nil {
		t.Fatal(err)
	}
	if !args.PlacesAPIDiscovery {
		t.Fatal("PlacesAPIDiscovery = false, want true")
	}
	if args.PlacesAPIPageLimit != 2 {
		t.Fatalf("PlacesAPIPageLimit = %d, want 2", args.PlacesAPIPageLimit)
	}
	if args.Discovery != "output/api.json" {
		t.Fatalf("Discovery = %q", args.Discovery)
	}
}

func TestParseArgsCDPURL(t *testing.T) {
	args, err := parseArgs([]string{"--cdp-url", "ws://127.0.0.1:9333"})
	if err != nil {
		t.Fatal(err)
	}
	if args.CDPURL != "ws://127.0.0.1:9333" {
		t.Fatalf("CDPURL = %q", args.CDPURL)
	}
}

func TestParseArgsBannerAudit(t *testing.T) {
	args, err := parseArgs([]string{"--banner-audit-only", "--notice-attempts", "3"})
	if err != nil {
		t.Fatal(err)
	}
	if !args.BannerAuditOnly {
		t.Fatal("BannerAuditOnly = false, want true")
	}
	if args.NoticeAttempts != 3 {
		t.Fatalf("NoticeAttempts = %d, want 3", args.NoticeAttempts)
	}
}

func TestDirectReviewsTextCanBeParsedWithoutRestrictedOverview(t *testing.T) {
	overview := mapText{Text: "Die Ansicht ist beschränkt und du siehst nur einen Teil der Google Maps-Daten. Route 1,5 km"}
	reviews := mapText{Text: "543214,82.336 Berichte\n21 bis 50 Bewertungen aufgrund von Beschwerden wegen Diffamierung entfernt."}
	if !isRestrictedMapsView(overview.Text) {
		t.Fatal("test fixture should include restricted overview text")
	}

	got := reviews.Text
	if isRestrictedMapsView(got) {
		t.Fatal("direct reviews text includes restricted overview text")
	}
	if notice := mapsreview.ParseNotice(got); notice == nil || notice.Min != 21 {
		t.Fatalf("ParseNotice(%q) = %#v, want deletion banner", got, notice)
	}
}

func TestClassifyPlaceState(t *testing.T) {
	tests := []struct {
		name              string
		rawText           string
		directReviewsText string
		want              string
	}{
		{name: "active", rawText: "FranKonya", directReviewsText: "Rezensionen\n4,5\n173 Berichte\nSortieren", want: mapsreview.PlaceStateActive},
		{name: "no public reviews", rawText: "EAT HAPPY\nÜbersicht", directReviewsText: "EAT HAPPY\nÜbersicht\nRoutenplaner\nRezension schreiben", want: mapsreview.PlaceStateNoPublicReviews},
		{name: "permanently closed", rawText: "Imbiss Nhan\nDauerhaft geschlossen", directReviewsText: "", want: mapsreview.PlaceStatePermanentlyClosed},
		{name: "temporarily closed", rawText: "Snack am Eck\nVorübergehend geschlossen", directReviewsText: "", want: mapsreview.PlaceStateTemporarilyClosed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyPlaceState(test.rawText, test.directReviewsText); got != test.want {
				t.Fatalf("classifyPlaceState() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDirectReviewsTextHasNoPublicReviews(t *testing.T) {
	text := "EAT HAPPY\nÜbersicht\nInfo\nRoutenplaner\nRezension schreiben\nWird auch oft gesucht\nHaDaCo Sushi Thai Wok\n4,5(105)"
	if !directReviewsTextHasNoPublicReviews(text) {
		t.Fatal("no-review-tab place was not detected")
	}
	textWithBerichteInSuggestions := "EAT HAPPY\nÜbersicht\nInfo\nRoutenplaner\nRezension schreiben\nWird auch oft gesucht\nIrgendwo\n4,5\n621 Berichte"
	if !directReviewsTextHasNoPublicReviews(textWithBerichteInSuggestions) {
		t.Fatal("suggestion reviews should not make place look like it has public reviews")
	}
	withReviews := "Zu den zwei goldenen Hirschen\nÜbersicht\nRezensionen\nInfo\n4,6\n447 Berichte\nSortieren"
	if directReviewsTextHasNoPublicReviews(withReviews) {
		t.Fatal("reviews panel detected as no-review page")
	}
}

func TestIsPartialMapsShell(t *testing.T) {
	shell := "Restaurants\nHotels\nGespeichert\nZuletzt verwendet\nApp herunterladen\nRoutenplaner\nSpeichern\nIn der Nähe\nTeilen"
	if !isPartialMapsShell(shell, "Schnitzery Nürnberg") {
		t.Fatal("partial Maps shell was not detected")
	}
	loadedPlace := "Schnitzery Nürnberg\nÜbersicht\nRezensionen\n4,5\n268 Berichte\nSortieren"
	if isPartialMapsShell(loadedPlace, "Schnitzery Nürnberg") {
		t.Fatal("loaded place was detected as partial shell")
	}
}

func TestExtractCategoryUsesDOMCategory(t *testing.T) {
	got := extractCategory("H&B Döner", "Döner / Kebab", "Restaurants in der Nähe\nH&B Döner\n4,7")
	if got == nil || *got != "Döner / Kebab" {
		t.Fatalf("extractCategory = %v, want Döner / Kebab", got)
	}
}

func TestExtractCategoryFallsBackToHeaderText(t *testing.T) {
	text := "Restaurants in der Nähe\nHotels\nH&B Döner\n4,7\n(263)\n\n·10–20 €\nDöner / Kebab\nÜbersicht\nRezensionen\nInfo"
	got := extractCategory("H&B Döner", "", text)
	if got == nil || *got != "Döner / Kebab" {
		t.Fatalf("extractCategory = %v, want Döner / Kebab", got)
	}
}

func TestExtractCategoryDoesNotUseNavigationChips(t *testing.T) {
	text := "Restaurants in der Nähe\nHotels\nMögliche Aktivitäten\nBars\nKaffee\nZum Mitnehmen\nLebensmittel\n2BITES Nürnberg\n4,8\n(100)\nÜbersicht"
	if got := extractCategory("2BITES Nürnberg", "", text); got != nil {
		t.Fatalf("extractCategory = %q, want nil", *got)
	}
}

func TestExtractCategoryCleansInlineIconSuffix(t *testing.T) {
	got := extractCategory("EAT HAPPY", "", "EAT HAPPY\nSushi Takeaway·\nÜbersicht\nInfo")
	if got == nil || *got != "Sushi" {
		t.Fatalf("extractCategory = %v, want Sushi Takeaway", got)
	}
}

func TestApplyNotice(t *testing.T) {
	row := successPlace()
	notice := &mapsreview.Notice{Text: "Zwei bis fünf Bewertungen aufgrund von Beschwerden wegen Diffamierung entfernt.", Min: 2, Max: mapsreview.IntPtr(5), Estimate: 3.5}

	applyNotice(&row, notice)
	mapsreview.ComputeMetrics(&row)
	if !row.HasDefamationNotice || row.RemovedText == nil || *row.RemovedText != notice.Text {
		t.Fatalf("notice was not applied: %#v", row)
	}
	if row.RemovedEstimate == nil || *row.RemovedEstimate != 3.5 {
		t.Fatalf("RemovedEstimate = %v, want 3.5", row.RemovedEstimate)
	}
}

func TestPreservePreviousMetadata(t *testing.T) {
	previous := successPlace()
	previous.Address = mapsreview.StringPtr("Gostenhofer Hauptstraße 20, 90443 Nürnberg")
	previous.Category = mapsreview.StringPtr("Restaurants")
	next := successPlace()
	next.Address = nil
	next.Category = nil

	got := preservePreviousMetadata(previous, next)
	if got.Address == nil || *got.Address != "Gostenhofer Hauptstraße 20, 90443 Nürnberg" {
		t.Fatalf("Address = %v", got.Address)
	}
	if got.Category == nil || *got.Category != "Restaurants" {
		t.Fatalf("Category = %v", got.Category)
	}
}

func TestShouldKeepPreviousRowPreventsBannerClearByDefault(t *testing.T) {
	previous := successPlace()
	previous.HasDefamationNotice = true
	previous.RemovedMin = mapsreview.IntPtr(11)
	previous.RemovedMax = mapsreview.IntPtr(20)
	previous.RemovedEstimate = mapsreview.FloatPtr(15.5)
	previous.RemovedText = mapsreview.StringPtr("11 bis 20 Bewertungen aufgrund von Beschwerden wegen Diffamierung entfernt.")

	next := successPlace()
	keep, reason := shouldKeepPreviousRow(previous, next, true, args{})
	if !keep {
		t.Fatal("shouldKeepPreviousRow = false, want true")
	}
	if !strings.Contains(reason, "clear an existing deletion banner") {
		t.Fatalf("reason = %q, want banner-clear reason", reason)
	}
}

func TestShouldKeepPreviousRowAllowsBannerClearWithFlag(t *testing.T) {
	previous := successPlace()
	previous.HasDefamationNotice = true
	next := successPlace()

	keep, reason := shouldKeepPreviousRow(previous, next, true, args{AllowBannerClears: true})
	if keep {
		t.Fatalf("shouldKeepPreviousRow = true (%q), want false", reason)
	}
}

func TestShouldKeepPreviousRowPreventsStatsRegression(t *testing.T) {
	previous := successPlace()
	next := successPlace()
	next.ReviewCount = nil

	keep, reason := shouldKeepPreviousRow(previous, next, true, args{AllowBannerClears: true})
	if !keep {
		t.Fatal("shouldKeepPreviousRow = false, want true")
	}
	if !strings.Contains(reason, "review count") {
		t.Fatalf("reason = %q, want review-count reason", reason)
	}
}

func TestShouldKeepPreviousRowAllowsNoReviewStatsClear(t *testing.T) {
	previous := successPlace()
	next := successPlace()
	next.Rating = nil
	next.ReviewCount = mapsreview.IntPtr(0)

	keep, reason := shouldKeepPreviousRow(previous, next, true, args{AllowBannerClears: true})
	if keep {
		t.Fatalf("shouldKeepPreviousRow = true (%q), want false", reason)
	}
}

func successPlace() mapsreview.Place {
	return mapsreview.Place{
		ID:          "place-id",
		Name:        "FranKonya",
		Rating:      mapsreview.FloatPtr(4.5),
		ReviewCount: mapsreview.IntPtr(173),
		Status:      "success",
	}
}
