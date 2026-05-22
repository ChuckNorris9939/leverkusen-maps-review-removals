package main

import (
	"os"
	"strings"
	"testing"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

func TestFilter(t *testing.T) {
	rows := []mapsreview.Place{
		{Name: "A", HasDefamationNotice: true},
		{Name: "B", HasDefamationNotice: false},
		{Name: "C", HasDefamationNotice: true},
	}
	withBanner := filter(rows, func(row mapsreview.Place) bool { return row.HasDefamationNotice })
	if len(withBanner) != 2 {
		t.Fatalf("filter = %d rows, want 2", len(withBanner))
	}
}

func TestTake(t *testing.T) {
	rows := make([]mapsreview.Place, 5)
	if got := len(take(rows, 3)); got != 3 {
		t.Fatalf("take = %d, want 3", got)
	}
	if got := len(take(rows, 0)); got != 5 {
		t.Fatalf("take(0) = %d, want 5", got)
	}
	if got := len(take(rows, 10)); got != 5 {
		t.Fatalf("take(10) = %d, want 5", got)
	}
}

func TestCountRows(t *testing.T) {
	rows := []mapsreview.Place{
		{Name: "A", HasDefamationNotice: true},
		{Name: "B", HasDefamationNotice: false},
	}
	if got := countRows(rows, func(row mapsreview.Place) bool { return row.HasDefamationNotice }); got != 1 {
		t.Fatalf("countRows = %d, want 1", got)
	}
}

func TestMaxFloat(t *testing.T) {
	if got := maxFloat(1, []float64{}); got != 1 {
		t.Fatalf("maxFloat fallback = %v, want 1", got)
	}
	if got := maxFloat(1, []float64{3, 5, 2}); got != 5 {
		t.Fatalf("maxFloat = %v, want 5", got)
	}
}

func TestIntCSV(t *testing.T) {
	if got := intCSV(nil); got != "" {
		t.Fatalf("intCSV(nil) = %q, want \"\"", got)
	}
	if got := intCSV(mapsreview.IntPtr(42)); got != "42" {
		t.Fatalf("intCSV(42) = %q, want \"42\"", got)
	}
}

func TestFloatCSV(t *testing.T) {
	if got := floatCSV(nil); got != "" {
		t.Fatalf("floatCSV(nil) = %q, want \"\"", got)
	}
	if got := floatCSV(mapsreview.FloatPtr(3.5)); got != "3.5" {
		t.Fatalf("floatCSV(3.5) = %q, want \"3.5\"", got)
	}
}

func TestEsc(t *testing.T) {
	if got := esc("<script>alert('xss')</script>"); got == "<script>" {
		t.Fatal("esc did not escape HTML")
	}
}

func TestWriteMostRemovedMDEscapesCity(t *testing.T) {
	file := t.TempDir() + "/most.md"
	if err := writeMostRemovedMD(file, nil, args{City: `<img src=x onerror=alert(1)>`}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(data)
	if strings.Contains(markdown, `<img src=x`) {
		t.Fatal("city was inserted as raw Markdown HTML")
	}
	if !strings.Contains(markdown, `&lt;img src=x`) {
		t.Fatal("escaped city is missing from Markdown")
	}
}

func TestWriteMostRemovedHTMLEscapesCity(t *testing.T) {
	file := t.TempDir() + "/most.html"
	if err := writeMostRemovedHTML(file, nil, nil, args{City: `<img src=x onerror=alert(1)>`}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	if strings.Contains(html, `<img src=x`) {
		t.Fatal("city was inserted as raw HTML")
	}
	if !strings.Contains(html, `&lt;img src=x`) {
		t.Fatal("escaped city is missing from HTML")
	}
}

func TestChartFilePrefix(t *testing.T) {
	tests := []struct {
		city string
		want string
	}{
		{city: mapsreview.DefaultCity, want: "nuernberg"},
		{city: "Fürth", want: "fuerth"},
		{city: "München Süd", want: "muenchen-sued"},
		{city: "***", want: "city"},
	}
	for _, test := range tests {
		if got := chartFilePrefix(test.city); got != test.want {
			t.Fatalf("chartFilePrefix(%q) = %q, want %q", test.city, got, test.want)
		}
	}
}

func TestParseArgsChartsRejectsEmptyCity(t *testing.T) {
	if _, err := parseArgs([]string{"--city"}); err == nil {
		t.Fatal("parseArgs(--city) succeeded, want error")
	}
}

func TestParseArgsChartsDefaults(t *testing.T) {
	args, err := parseArgs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if args.City != mapsreview.DefaultCity {
		t.Fatalf("City = %q, want %q", args.City, mapsreview.DefaultCity)
	}
	if args.Top != 30 {
		t.Fatalf("Top = %d, want 30", args.Top)
	}
	if args.MinCleanReviews != 100 {
		t.Fatalf("MinCleanReviews = %d, want 100", args.MinCleanReviews)
	}
}

func TestParseArgsChartsCustom(t *testing.T) {
	args, err := parseArgs([]string{"--top", "50", "--min-clean-reviews", "200"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Top != 50 {
		t.Fatalf("Top = %d, want 50", args.Top)
	}
	if args.MinCleanReviews != 200 {
		t.Fatalf("MinCleanReviews = %d, want 200", args.MinCleanReviews)
	}
}

func TestMakeChartSmoke(t *testing.T) {
	rows := []mapsreview.Place{
		{Name: "Test Place", Postcode: mapsreview.StringPtr("90402"), Rating: mapsreview.FloatPtr(4.5), ReviewCount: mapsreview.IntPtr(100), Status: "success"},
	}
	args := args{Top: 5, MinCleanReviews: 10}
	svg := makeChart(rows, "overall", args)
	if svg == "" {
		t.Fatal("makeChart returned empty SVG")
	}
	if !contains(svg, "<svg") || !contains(svg, "Test Place") {
		t.Fatal("SVG is missing expected content")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
