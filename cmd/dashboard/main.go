package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

//go:embed app.js
var dashboardJS string

const (
	defaultInput  = mapsreview.ResultsJSON
	defaultOutput = "output/charts/nuernberg_dashboard.html"

	siteURL = "https://nuernberg-maps-review-removals.patwoz.dev/"
)

type args struct {
	City   string
	Input  string
	Output string
}

type clientRow struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Postcode           string   `json:"postcode"`
	Lat                *float64 `json:"lat,omitempty"`
	Lng                *float64 `json:"lng,omitempty"`
	BezirkID           string   `json:"bezirkId"`
	BezirkName         string   `json:"bezirkName"`
	BezirkLabel        string   `json:"bezirkLabel"`
	Rating             *float64 `json:"rating"`
	ReviewCount        *int     `json:"reviewCount"`
	Category           string   `json:"category"`
	ParentCategory     string   `json:"parentCategory"`
	HasBanner          bool     `json:"hasBanner"`
	RemovedRange       string   `json:"removedRange"`
	RemovedMin         *int     `json:"removedMin"`
	RemovedMax         *int     `json:"removedMax"`
	RemovedEstimate    float64  `json:"removedEstimate"`
	DeletionRatioPct   *float64 `json:"deletionRatioPct"`
	RealRatingAdjusted *float64 `json:"realRatingAdjusted"`
	RemovedText        string   `json:"removedText"`
	URL                string   `json:"url"`
	Address            string   `json:"address"`
	ReadAt             string   `json:"readAt"`
	PlaceState         string   `json:"placeState,omitempty"`
}

type seoStats struct {
	Total           int
	Banners         int
	Clean           int
	RemovedEstimate int
	Snapshot        string
	Top             []clientRow
}

type dashboardMetadata struct {
	SiteName        string
	PageTitle       string
	PageDescription string
	CanonicalURL    string
	SocialImageURL  string
	SocialImageAlt  string
}

func dashboardMeta(city string) dashboardMetadata {
	return dashboardMetadata{
		SiteName:        fmt.Sprintf("%s Maps Review Removals", city),
		PageTitle:       fmt.Sprintf("%s Google-Maps-Bewertungen: Löschbanner-Dashboard", city),
		PageDescription: fmt.Sprintf("Interaktives %s-Dashboard zu sichtbaren Google-Maps-Hinweisen auf entfernte Bewertungen: Löschbanner, Löschquoten, Karte und Daten-Explorer.", city),
		CanonicalURL:    siteURL,
		SocialImageURL:  siteURL + "charts/" + dashboardChartFilePrefix(city) + "_overall_summary.png",
		SocialImageAlt:  fmt.Sprintf("Diagramm zur Auswertung entfernter Google-Maps-Bewertungen in %s", city),
	}
}

func dashboardChartFilePrefix(city string) string {
	if mapsreview.IsDefaultCity(city) {
		return "nuernberg"
	}
	city = strings.ToLower(strings.TrimSpace(city))
	city = strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
	).Replace(city)
	var b strings.Builder
	lastDash := false
	for _, r := range city {
		isASCIIAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isASCIIAlnum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	prefix := strings.Trim(b.String(), "-")
	if prefix == "" {
		return "city"
	}
	return prefix
}

func main() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args args) error {
	rows, err := mapsreview.ReadJSON(args.Input, []mapsreview.Place{})
	if err != nil {
		return err
	}
	data := makeClientRows(rows)
	if err := os.MkdirAll(filepath.Dir(args.Output), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(args.Output, []byte(makeHTML(args, data)), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", args.Output)
	return nil
}

func parseArgs(argv []string) (args, error) {
	out := args{City: mapsreview.DefaultCity, Input: defaultInput, Output: defaultOutput}
	for i := 0; i < len(argv); i++ {
		key, value, consume := splitArg(argv, i)
		switch key {
		case "--city":
			out.City = value
		case "--input":
			out.Input = value
		case "--output":
			out.Output = value
		case "--help", "-h":
			fmt.Printf(`Usage:
  go run ./cmd/dashboard
  go run ./cmd/dashboard --input output/places.json --output output/charts/nuernberg_dashboard.html

Options:
  --city <name>       Displayed city name. Default: %s.
  --input <path>      Scrape results JSON. Default: %s.
  --output <path>     Dashboard HTML path. Default: %s.
`, mapsreview.DefaultCity, defaultInput, defaultOutput)
			os.Exit(0)
		default:
			return out, fmt.Errorf("unknown argument: %s", argv[i])
		}
		if consume {
			i++
		}
	}
	out.City = strings.TrimSpace(out.City)
	if out.City == "" {
		return out, fmt.Errorf("--city must not be empty")
	}
	return out, nil
}

func splitArg(argv []string, index int) (key string, value string, consume bool) {
	arg := argv[index]
	if before, after, ok := strings.Cut(arg, "="); ok {
		return before, after, false
	}
	if index+1 < len(argv) && !strings.HasPrefix(argv[index+1], "--") {
		return arg, argv[index+1], true
	}
	return arg, "", false
}

func makeClientRows(rows []mapsreview.Place) []clientRow {
	out := make([]clientRow, 0, len(rows))
	for _, row := range rows {
		if row.Status != "success" || row.Name == "" || row.Rating == nil {
			continue
		}
		mapsreview.EnrichPlaceLocation(&row)
		mapsreview.ApplyPlaceOverrides(&row)
		removedEstimate := 0.0
		if row.HasDefamationNotice {
			removedEstimate = mapsreview.RemovedSortValue(row)
		}
		lat := row.Lat
		lng := row.Lng
		bezirkID := mapsreview.StringValue(row.BezirkID)
		bezirkName := mapsreview.StringValue(row.BezirkName)
		bezirkLabel := ""
		if bezirkID != "" && bezirkName != "" {
			bezirkLabel = bezirkID + " " + bezirkName
		}
		out = append(out, clientRow{
			ID:                 row.ID,
			Name:               row.Name,
			Postcode:           mapsreview.StringValue(row.Postcode),
			Lat:                lat,
			Lng:                lng,
			BezirkID:           bezirkID,
			BezirkName:         bezirkName,
			BezirkLabel:        bezirkLabel,
			Rating:             row.Rating,
			ReviewCount:        row.ReviewCount,
			Category:           mapsreview.StringValue(row.Category),
			ParentCategory:     mapsreview.StringValue(row.ParentCategory),
			HasBanner:          row.HasDefamationNotice,
			RemovedRange:       mapsreview.RemovedRange(row),
			RemovedMin:         row.RemovedMin,
			RemovedMax:         row.RemovedMax,
			RemovedEstimate:    removedEstimate,
			DeletionRatioPct:   row.DeletionRatioPct,
			RealRatingAdjusted: row.RealRatingAdjusted,
			RemovedText:        mapsreview.StringValue(row.RemovedText),
			URL:                row.URL,
			Address:            mapsreview.StringValue(row.Address),
			ReadAt:             row.ReadAt,
			PlaceState:         row.PlaceState,
		})
	}
	return out
}

func makeHTML(args args, data []clientRow) string {
	city := strings.TrimSpace(args.City)
	if city == "" {
		city = mapsreview.DefaultCity
	}
	postcodes := uniqueSorted(data, func(row clientRow) string { return row.Postcode })
	bezirke := []string{}
	bezirkBoundaries := []mapsreview.BezirkBoundary{}
	if mapsreview.IsDefaultCity(city) {
		bezirke = allBezirkLabels()
		bezirkBoundaries = mapsreview.BezirkBoundaries()
	}
	if len(bezirke) == 0 {
		bezirke = uniqueSorted(data, func(row clientRow) string { return row.BezirkLabel })
	}
	ranges := uniqueSorted(data, func(row clientRow) string { return row.RemovedRange })
	sort.SliceStable(ranges, func(i, j int) bool {
		return maxEstimateForRange(data, ranges[i]) > maxEstimateForRange(data, ranges[j])
	})
	snapshot := snapshotTime(data)
	snapshotDisplay := snapshot.Format("02.01.2006")
	stats := makeSEOStats(data, snapshotDisplay)
	meta := dashboardMeta(city)
	structuredData := structuredDataJSON(stats, snapshot, city, meta)

	jsonText := compactClientDataJSON(data, city)
	bezirkText := compactBezirkDataJSON(bezirkBoundaries)
	configText := safeJSON(map[string]string{"city": city})

	postcodeOptions := ""
	for _, postcode := range postcodes {
		postcodeOptions += fmt.Sprintf(`<option value="%s">%s</option>`, escAttr(postcode), esc(postcode))
	}
	bezirkOptions := ""
	if countRows(data, func(row clientRow) bool { return row.BezirkLabel == "" }) > 0 {
		bezirkOptions += `<option value="__none__">Ohne Bezirk</option>`
	}
	for _, bezirk := range bezirke {
		bezirkOptions += fmt.Sprintf(`<option value="%s">%s</option>`, escAttr(bezirk), esc(bezirk))
	}
	rangeOptions := ""
	for _, r := range ranges {
		if r != "" {
			rangeOptions += fmt.Sprintf(`<option value="%s">%s</option>`, escAttr(r), esc(r))
		}
	}
	// Build hierarchical options: parent categories first (selectable), then their sub-categories
	type catEntry struct{ parent, name string }
	parentMap := map[string][]string{}
	for _, row := range data {
		if row.Category != "" && row.ParentCategory != "" {
			parentMap[row.ParentCategory] = append(parentMap[row.ParentCategory], row.Category)
		}
	}
	// Sort parents by total count desc, then alphabetically
	parentOrder := make([]string, 0, len(parentMap))
	for parent := range parentMap {
		parentOrder = append(parentOrder, parent)
	}
	sort.SliceStable(parentOrder, func(i, j int) bool {
		ci, cj := 0, 0
		for _, row := range data {
			if row.ParentCategory == parentOrder[i] {
				ci++
			}
			if row.ParentCategory == parentOrder[j] {
				cj++
			}
		}
		if ci != cj {
			return ci > cj
		}
		return parentOrder[i] < parentOrder[j]
	})
	categoryOptions := ""
	for _, parent := range parentOrder {
		// Parent category as selectable option
		categoryOptions += fmt.Sprintf(`<option value="parent:%s">%s</option>`, escAttr(parent), esc(parent))
		// Deduplicate children
		seen := map[string]bool{}
		for _, child := range parentMap[parent] {
			if !seen[child] {
				seen[child] = true
				categoryOptions += fmt.Sprintf(`<option value="%s">&nbsp;&nbsp;%s</option>`, escAttr(child), esc(child))
			}
		}
	}

	page := `<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>__PAGE_TITLE__</title>
  <meta name="description" content="__PAGE_DESCRIPTION__">
  <meta name="robots" content="index,follow,max-image-preview:large">
  <meta name="author" content="Patrick Wozniak">
  <meta name="theme-color" content="#ffffff" media="(prefers-color-scheme: light)">
  <meta name="theme-color" content="#0e0c0b" media="(prefers-color-scheme: dark)">
  <link rel="canonical" href="__CANONICAL_URL__">
  <link rel="alternate" hreflang="de" href="__CANONICAL_URL__">
  <link rel="alternate" hreflang="x-default" href="__CANONICAL_URL__">
  <meta property="og:type" content="website">
  <meta property="og:locale" content="de_DE">
  <meta property="og:site_name" content="__SITE_NAME__">
  <meta property="og:title" content="__PAGE_TITLE__">
  <meta property="og:description" content="__PAGE_DESCRIPTION__">
  <meta property="og:url" content="__CANONICAL_URL__">
  <meta property="og:image" content="__SOCIAL_IMAGE__">
  <meta property="og:image:width" content="1800">
  <meta property="og:image:height" content="2500">
  <meta property="og:image:alt" content="__SOCIAL_IMAGE_ALT__">
  <meta property="og:updated_time" content="__MODIFIED_TIME__">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="__PAGE_TITLE__">
  <meta name="twitter:description" content="__PAGE_DESCRIPTION__">
  <meta name="twitter:image" content="__SOCIAL_IMAGE__">
  <meta name="twitter:image:alt" content="__SOCIAL_IMAGE_ALT__">
  <script type="application/ld+json">
__STRUCTURED_DATA__
  </script>
__ANALYTICS__
  <script>
    (function () {
      try {
        const savedTheme = localStorage.getItem('dashboardTheme');
        if (savedTheme === 'dark' || savedTheme === 'light') document.documentElement.dataset.theme = savedTheme;
      } catch (_) {}
    }());
  </script>
  <style>
    :root {
      color-scheme: light;
      --red: #cf2a1b;
      --red-dark: #9f2017;
      --text: #3f3f3f;
      --heading: #333;
      --muted: #6f6f6f;
      --line: #d6d6d6;
      --soft: #f4f4f4;
      --blue: #1f6f8b;
      --green: #2d7b3f;
      --orange: #d97a1d;
      --bg: #fff;
      --surface: #fff;
      --surface-raised: #fff;
      --surface-muted: #f6f6f6;
      --surface-subtle: #fafafa;
      --control-bg: #333;
      --control-text: #fff;
      --control-muted: rgba(255,255,255,.78);
      --input-bg: #fff;
      --input-text: #333;
      --nav-text: #666;
      --hero-bg: linear-gradient(180deg, rgba(38,120,155,.25), rgba(38,120,155,.25)), linear-gradient(135deg, #b7d5e1 0%, #e1edf2 45%, #bdc8cf 100%);
      --hero-title-bg: rgba(55,55,55,.78);
      --track-bg: #e7e7e7;
      --table-head-bg: #f3f3f3;
      --table-head-hover: #ececec;
      --row-hover: #fff4f2;
      --target-row: #fff0cc;
      --pill-bg: #e8f2ea;
      --pill-bad-bg: #fde6e2;
      --map-bg: #f4f4f4;
      --hint-bg: rgba(51,51,51,.88);
      --focus-blue: rgba(31,111,139,.13);
      --focus-red: rgba(207,42,27,.35);
      --shadow: 0 2px 10px rgba(0,0,0,.18);
      --sans: Arial, Helvetica, sans-serif;
    }
    @media (prefers-color-scheme: dark) {
      :root:not([data-theme="light"]) {
        color-scheme: dark;
        --red: #ff5b49;
        --red-dark: #e23c2b;
        --text: #e7e2dc;
        --heading: #fff5ec;
        --muted: #b8afa6;
        --line: #332e2b;
        --soft: #171412;
        --blue: #6ab8d8;
        --green: #72c382;
        --orange: #ff9d42;
        --bg: #0e0c0b;
        --surface: #171412;
        --surface-raised: #211c19;
        --surface-muted: #211c19;
        --surface-subtle: #141110;
        --control-bg: #fff5ec;
        --control-text: #171412;
        --control-muted: rgba(23,20,18,.68);
        --input-bg: #120f0e;
        --input-text: #fff5ec;
        --nav-text: #d4c8bd;
        --hero-bg: radial-gradient(circle at 18% 18%, rgba(255,91,73,.18), transparent 34%), linear-gradient(180deg, rgba(14,12,11,.32), rgba(14,12,11,.68)), linear-gradient(135deg, #102b35 0%, #171412 48%, #3b1c18 100%);
        --hero-title-bg: rgba(14,12,11,.82);
        --track-bg: #302925;
        --table-head-bg: #211c19;
        --table-head-hover: #2b2420;
        --row-hover: #2a1714;
        --target-row: #382b12;
        --pill-bg: rgba(114,195,130,.16);
        --pill-bad-bg: rgba(255,91,73,.16);
        --map-bg: #151210;
        --hint-bg: rgba(14,12,11,.92);
        --focus-blue: rgba(106,184,216,.22);
        --focus-red: rgba(255,91,73,.32);
        --shadow: 0 18px 40px rgba(0,0,0,.42);
      }
    }
    :root[data-theme="dark"] {
      color-scheme: dark;
      --red: #ff5b49;
      --red-dark: #e23c2b;
      --text: #e7e2dc;
      --heading: #fff5ec;
      --muted: #b8afa6;
      --line: #332e2b;
      --soft: #171412;
      --blue: #6ab8d8;
      --green: #72c382;
      --orange: #ff9d42;
      --bg: #0e0c0b;
      --surface: #171412;
      --surface-raised: #211c19;
      --surface-muted: #211c19;
      --surface-subtle: #141110;
      --control-bg: #fff5ec;
      --control-text: #171412;
      --control-muted: rgba(23,20,18,.68);
      --input-bg: #120f0e;
      --input-text: #fff5ec;
      --nav-text: #d4c8bd;
      --hero-bg: radial-gradient(circle at 18% 18%, rgba(255,91,73,.18), transparent 34%), linear-gradient(180deg, rgba(14,12,11,.32), rgba(14,12,11,.68)), linear-gradient(135deg, #102b35 0%, #171412 48%, #3b1c18 100%);
      --hero-title-bg: rgba(14,12,11,.82);
      --track-bg: #302925;
      --table-head-bg: #211c19;
      --table-head-hover: #2b2420;
      --row-hover: #2a1714;
      --target-row: #382b12;
      --pill-bg: rgba(114,195,130,.16);
      --pill-bad-bg: rgba(255,91,73,.16);
      --map-bg: #151210;
      --hint-bg: rgba(14,12,11,.92);
      --focus-blue: rgba(106,184,216,.22);
      --focus-red: rgba(255,91,73,.32);
      --shadow: 0 18px 40px rgba(0,0,0,.42);
    }
    * { box-sizing: border-box; }
    html { scroll-behavior: smooth; }
    body { margin: 0; min-height: 100vh; font-family: var(--sans); color: var(--text); background: var(--bg); }
    .sitebar { height: 76px; background: var(--surface-raised); border-bottom: 1px solid var(--line); box-shadow: var(--shadow); }
    .sitebar-inner { position: relative; width: min(1320px, calc(100vw - 32px)); height: 100%; margin: 0 auto; display: flex; align-items: center; gap: 42px; }
    .top-icons { display: flex; gap: 28px; color: var(--nav-text); font-size: 33px; line-height: 1; }
    .theme-toggle { display: inline-flex; align-items: center; gap: 8px; height: 42px; margin-left: auto; margin-right: 190px; padding: 0 14px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface); color: var(--heading); box-shadow: 0 1px 4px rgba(0,0,0,.12); font-weight: 700; cursor: pointer; }
    .theme-toggle:hover, .theme-toggle:focus-visible { border-color: var(--red); color: var(--red); outline: none; }
    .theme-toggle-icon { width: 1.1em; text-align: center; }
    .n-logo { position: absolute; right: 0; top: 0; width: 170px; height: 128px; padding: 66px 14px 10px; background: var(--red); color: #fff; font-size: 24px; font-weight: 700; letter-spacing: .04em; text-transform: uppercase; z-index: 5; }
    .n-logo::before { content: ""; position: absolute; left: 14px; right: 14px; top: 58px; height: 2px; background: #fff; opacity: .9; }
    .n-logo::after { content: "⌂⌂"; position: absolute; right: 13px; top: 20px; color: #fff; font-size: 36px; letter-spacing: -12px; transform: scaleX(1.4); }
    .hero { min-height: 380px; margin: 0 0 30px; background: var(--hero-bg); display: flex; align-items: end; }
    .hero-inner { width: min(1320px, calc(100vw - 32px)); margin: 0 auto; padding: 140px 0 42px; }
    .hero-title { width: min(760px, 100%); margin: 0; padding: 24px 28px; background: var(--hero-title-bg); color: #fff; font-size: clamp(32px, 4vw, 52px); line-height: 1.12; font-weight: 400; }
    .hero-subtitle { width: min(760px, 100%); margin-top: 14px; padding: 18px 22px; background: var(--surface-raised); border-radius: 5px; box-shadow: var(--shadow); color: var(--muted); font-size: 20px; line-height: 1.45; }
    main { width: min(1320px, calc(100vw - 32px)); margin: 0 auto 70px; }
    .controls { position: sticky; top: 0; z-index: 2000; display: grid; grid-template-columns: minmax(200px, 1fr) 120px 170px 130px 130px 130px 95px auto; gap: 12px; align-items: end; padding: 16px; margin: 0 0 24px; background: var(--surface-raised); border: 1px solid var(--line); box-shadow: 0 2px 8px rgba(0,0,0,.12); }
    .chips { display: flex; flex-wrap: wrap; gap: 8px; margin: 0 0 14px; }
    .chip { display: inline-flex; align-items: center; gap: 5px; padding: 7px 16px; border: 1px solid var(--line); border-radius: 20px; background: var(--surface); color: var(--text); font-size: 13px; cursor: pointer; transition: .12s; }
    .chip:hover { border-color: var(--red); background: var(--row-hover); }
    .chip.active { background: var(--red); border-color: var(--red); color: #fff; }
    .sub-chips { display: flex; flex-wrap: wrap; gap: 6px; margin: -8px 0 14px; min-height: 30px; }
    .sub-chip { padding: 4px 12px; border: 1px solid var(--line); border-radius: 14px; background: var(--surface-muted); color: var(--muted); font-size: 12px; cursor: pointer; transition: .12s; }
    .sub-chip:hover { border-color: var(--red); color: var(--text); background: var(--surface); }
    .sub-chip.active { background: var(--red); border-color: var(--red); color: #fff; }
    .filter-toggle { display: none; }
    label { display: block; margin-bottom: 6px; color: var(--muted); font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; }
    input, select, button { font: inherit; }
    input, select { width: 100%; height: 44px; padding: 0 12px; border: 1px solid var(--line); border-radius: 5px; background: var(--input-bg); color: var(--input-text); outline: none; }
    input:focus, select:focus { border-color: var(--blue); box-shadow: 0 0 0 3px var(--focus-blue); }
    .reset { height: 34px; border: 0; border-radius: 5px; padding: 0 10px; background: var(--control-bg); color: var(--control-text); font-size: 13px; font-weight: 700; cursor: pointer; align-self: center; }
    .grid { display: grid; gap: 16px; }
    .kpis { grid-template-columns: repeat(5, minmax(0, 1fr)); }
    .card { background: var(--surface); border: 1px solid var(--line); overflow: hidden; }
    .seo-summary { padding: 18px; margin-top: 18px; line-height: 1.55; }
    .seo-summary h2, .seo-top h3 { margin: 0 0 8px; color: var(--heading); }
    .seo-summary p { margin: 0 0 12px; color: var(--muted); }
    .seo-facts { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin: 14px 0; padding: 0; list-style: none; }
    .seo-facts li { margin: 0; padding: 12px; border: 1px solid var(--line); background: var(--surface-muted); }
    .seo-facts strong { display: block; color: var(--heading); font-size: 22px; line-height: 1.1; }
    .seo-top { margin-top: 14px; }
    .seo-top ol { margin: 8px 0 0 22px; padding: 0; }
    .seo-top li { margin: 6px 0; }
    .seo-meta { color: var(--muted); }
    .kpi { padding: 18px; border-top: 5px solid var(--red); }
    .kpi:nth-child(2) { border-top-color: var(--orange); }
    .kpi:nth-child(3) { border-top-color: var(--blue); }
    .kpi:nth-child(4) { border-top-color: var(--red-dark); }
    .kpi:nth-child(5) { border-top-color: var(--green); }
    .kpi .value { display: block; color: var(--heading); font-size: clamp(30px, 3vw, 44px); font-weight: 400; line-height: 1; }
    .kpi .label { display: block; margin-top: 8px; color: var(--muted); font-size: 14px; }
    .panel-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); align-items: stretch; margin-top: 18px; }
    .panel { padding: 18px; min-height: 330px; }
    .panel h2, .dist h2 { margin: 0 0 8px; color: var(--heading); font-size: 22px; font-weight: 700; }
    .panel p { margin: 0 0 16px; color: var(--muted); font-size: 13px; }
    .bars { display: grid; gap: 10px; }
    .bar-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 7px 10px; align-items: center; font-size: 12px; }
    .bar-link { width: 100%; padding: 0; border: 0; background: transparent; color: inherit; text-align: left; text-decoration: none; cursor: pointer; }
    .bar-link:hover .bar-name, .bar-link:focus .bar-name { color: var(--red); text-decoration: underline; }
    .bar-link:focus { outline: 2px solid var(--focus-red); outline-offset: 3px; }
    .bar-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 700; }
    .bar-value { color: var(--muted); font-variant-numeric: tabular-nums; white-space: nowrap; }
    .track { grid-column: 1 / -1; height: 8px; background: var(--track-bg); overflow: hidden; }
    .fill { height: 100%; background: var(--red); }
    .fill.orange { background: var(--orange); }
    .fill.green { background: var(--green); }
    .dist { padding: 18px; margin-top: 18px; }
    .dist-row { display: grid; grid-template-columns: 130px minmax(0, 1fr) 90px; gap: 12px; align-items: center; margin: 8px 0; font-size: 13px; }
    .bezirk-summary { padding: 18px; margin-top: 18px; }
    .bezirk-summary h2 { margin: 0 0 8px; color: var(--heading); font-size: 22px; font-weight: 700; }
    .bezirk-summary p { margin: 0 0 14px; color: var(--muted); font-size: 13px; }
    .bezirk-list { display: grid; gap: 8px; }
    .bezirk-row { display: grid; grid-template-columns: minmax(0, 1fr) 76px 80px 86px; gap: 12px; align-items: center; width: 100%; padding: 10px 12px; border: 1px solid var(--line); background: var(--surface); color: var(--heading); text-align: left; cursor: pointer; }
    .bezirk-row:hover, .bezirk-row:focus { border-color: var(--red); background: var(--row-hover); outline: none; }
    .bezirk-row strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .bezirk-row span { color: var(--muted); font-variant-numeric: tabular-nums; text-align: right; }
    .dist-track { height: 12px; background: var(--track-bg); overflow: hidden; }
    .dist-fill { height: 100%; background: var(--red); }
    .map-panel { margin-top: 18px; padding: 18px; }
    .map-panel h2 { margin: 0 0 8px; color: var(--heading); font-size: 22px; font-weight: 700; }
    .map-panel p { margin: 0 0 16px; color: var(--muted); font-size: 13px; }
    #placesMap { position: relative; z-index: 0; height: 520px; border: 1px solid var(--line); background: var(--map-bg); }
    #placesMap.map-needs-key::after, #placesMap.map-active::after { position: absolute; left: 50%; top: 18px; z-index: 1000; transform: translateX(-50%); padding: 10px 14px; border-radius: 4px; background: var(--hint-bg); color: #fff; font-size: 13px; font-weight: 700; box-shadow: 0 2px 10px rgba(0,0,0,.22); pointer-events: none; }
    #placesMap.map-needs-key::after { content: "Strg/⌘ halten, um mit dem Mausrad zu zoomen"; }
    #placesMap.map-active::after { content: "Karten-Zoom aktiv"; }
    @media (pointer: coarse) { #placesMap.map-needs-key::after { content: "Zwei Finger zum Zoomen und Bewegen der Karte"; } }
    .map-empty { display: grid; place-items: center; height: 100%; padding: 20px; color: var(--muted); text-align: center; }
    .map-legend { display: flex; flex-wrap: wrap; gap: 16px; margin-top: 10px; color: var(--muted); font-size: 13px; }
    .legend-dot { display: inline-block; width: 12px; height: 12px; margin-right: 6px; border-radius: 50%; vertical-align: -1px; }
    .legend-area { display: inline-block; width: 16px; height: 12px; margin-right: 6px; border: 2px solid var(--red); background: rgba(207,42,27,.12); vertical-align: -2px; }
    .leaflet-container { background: var(--map-bg); color: var(--text); }
    .leaflet-bar a, .leaflet-bar a:hover, .leaflet-control-attribution { background: var(--surface-raised); color: var(--text); border-color: var(--line); }
    .leaflet-control-attribution a, .leaflet-tooltip a { color: var(--red); }
    .leaflet-tooltip, .leaflet-popup-content-wrapper, .leaflet-popup-tip { background: var(--surface-raised); color: var(--text); border-color: var(--line); box-shadow: var(--shadow); }
    .tabs { display: flex; flex-wrap: wrap; gap: 8px; margin: 22px 0 14px; }
    .tab { border: 1px solid var(--line); border-radius: 5px; padding: 10px 14px; background: var(--surface-muted); color: var(--heading); font-weight: 700; cursor: pointer; }
    .tab.active { background: var(--red); border-color: var(--red); color: #fff; }
    .tab:disabled { opacity: .65; cursor: progress; }
    .nearby-status { margin: -6px 0 14px; color: var(--muted); font-size: 13px; }
    .nearby-status.error { color: var(--red); font-weight: 700; }
    .nearby-status[hidden] { display: none; }
    .table-head { display: flex; justify-content: space-between; align-items: center; margin: 0 0 10px; color: var(--muted); font-size: 14px; }
    .table-head strong { color: var(--heading); font-size: 22px; }
    .table-wrap { overflow: auto; background: var(--surface); border: 1px solid var(--line); }
    table { width: 100%; min-width: 1570px; border-collapse: collapse; table-layout: fixed; }
    col.rank { width: 70px; } col.name { width: 360px; } col.bezirk { width: 210px; } col.plz { width: 90px; } col.rating { width: 95px; } col.reviews { width: 125px; } col.banner { width: 100px; } col.removed { width: 120px; } col.estimate { width: 125px; } col.ratio { width: 120px; } col.real { width: 160px; } col.checked { width: 130px; } col.category { width: 175px; }
    th { position: sticky; top: 0; z-index: 2; padding: 0; background: var(--table-head-bg); border-bottom: 2px solid var(--line); color: var(--heading); font-size: 13px; text-align: left; }
    th button { display: flex; align-items: center; gap: 5px; width: 100%; min-height: 44px; padding: 12px; border: 0; background: transparent; color: inherit; font: inherit; font-weight: 700; text-align: inherit; cursor: pointer; }
    th.num button, th.rank button { justify-content: flex-end; text-align: right; }
    th button:hover { color: var(--red); background: var(--table-head-hover); }
    .arrow { width: 1em; color: var(--muted); }
    button.active .arrow { color: var(--red); }
    td { padding: 12px; border-bottom: 1px solid var(--line); vertical-align: top; font-size: 14px; }
    tbody tr:nth-child(even) { background: var(--surface-subtle); }
    tbody tr:hover { background: var(--row-hover); }
    tbody tr.target-row { background: var(--target-row); box-shadow: inset 5px 0 0 var(--red); }
    td.num, td.rank { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
    td.name { overflow-wrap: anywhere; font-weight: 700; }
    .entry-address { display: block; margin-top: 4px; color: var(--muted); font-size: 12px; font-weight: 400; line-height: 1.35; }
    a { color: var(--red); text-decoration: none; }
    a:hover { text-decoration: underline; }
    .pill { display: inline-flex; align-items: center; border-radius: 3px; padding: 3px 7px; background: var(--pill-bg); color: var(--green); font-weight: 700; font-size: 12px; }
    .pill.bad { background: var(--pill-bad-bg); color: var(--red); }
    .show-more { display: block; min-width: 220px; height: 42px; margin: 14px auto 0; padding: 0 18px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface-raised); color: var(--heading); font-weight: 700; cursor: pointer; }
    .show-more:hover, .show-more:focus-visible { border-color: var(--red); color: var(--red); outline: none; }
    .show-more[hidden] { display: none; }
    footer { margin-top: 18px; color: var(--muted); font-size: 13px; line-height: 1.5; }
    .footer-privacy, .footer-credit { margin-top: 6px; }
    .footer-credit a { font-weight: 700; }
    @media (max-width: 1200px) { .kpis, .panel-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .controls { grid-template-columns: 1fr 1fr 1fr; } .search { grid-column: 1 / -1; } .theme-toggle { margin-left: auto; margin-right: 0; } .n-logo { position: relative; height: 76px; width: 150px; margin-left: 0; padding-top: 48px; } .n-logo::before { top: 40px; } .n-logo::after { top: 4px; } }
    @media (max-width: 720px) {
      .sitebar-inner, main, .hero-inner { width: min(100vw - 20px, 1320px); }
      .sitebar-inner { gap: 14px; }
      .top-icons { display: none; }
      .theme-toggle { width: 42px; padding: 0; justify-content: center; }
      .theme-toggle-text { display: none; }
      .n-logo { width: 128px; font-size: 18px; padding-left: 10px; padding-right: 10px; }
      .kpis, .panel-grid, .seo-facts { grid-template-columns: 1fr; }
      .controls { position: sticky; top: 0; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 9px 10px; padding: 10px; margin-bottom: 14px; }
      .filter-toggle { grid-column: 1 / -1; display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; min-height: 42px; padding: 8px 12px; border: 0; border-radius: 5px; background: var(--control-bg); color: var(--control-text); text-align: left; cursor: pointer; }
      .filter-toggle strong { display: block; font-size: 15px; line-height: 1.1; }
      .filter-summary { display: block; max-width: 380px; overflow: hidden; color: var(--control-muted); font-size: 12px; font-weight: 400; line-height: 1.3; text-overflow: ellipsis; white-space: nowrap; }
      .filter-toggle-icon { font-size: 16px; transition: transform .18s ease; }
      .controls:not(.is-collapsed) .filter-toggle-icon { transform: rotate(180deg); }
      .controls.is-collapsed .control, .controls.is-collapsed .reset { display: none; }
      .search { grid-column: 1 / -1; }
      label { margin-bottom: 4px; font-size: 10px; letter-spacing: .04em; }
      input, select { height: 38px; padding: 0 9px; font-size: 15px; }
      .reset { height: 34px; padding: 0 10px; font-size: 13px; }
      .hero { min-height: 300px; }
      .hero-inner { padding-top: 92px; }
      .hero-title { font-size: 32px; padding: 18px; }
      .hero-subtitle { font-size: 16px; }
    }
  </style>
</head>
<body>
  <div class="sitebar" role="banner">
    <div class="sitebar-inner">
      <div class="top-icons" aria-hidden="true"><span>●</span><span>☝</span><span>▰</span></div>
      <button class="theme-toggle" id="themeToggle" type="button" aria-label="Dunkles Design aktivieren" aria-pressed="false"><span class="theme-toggle-icon" aria-hidden="true">☾</span><span class="theme-toggle-text">Dunkel</span></button>
      <div class="n-logo">__CITY__</div>
    </div>
  </div>

  <section class="hero" aria-label="Seitentitel">
    <div class="hero-inner">
      <h1 class="hero-title">__CITY__ Google-Maps-Bewertungen</h1>
      <div class="hero-subtitle">Interaktives Daten-Dashboard zu sichtbaren Hinweisen auf entfernte Bewertungen wegen Diffamierungsbeschwerden.</div>
    </div>
  </section>

  <main>
    <section class="chips" aria-label="Quick-Filter"><button type="button" class="chip" data-chip="banner">🔴 Mit Löschbanner</button><button type="button" class="chip" data-chip="gastro">🍽️ Gastronomie</button><button type="button" class="chip" data-chip="nachtleben">🎉 Nachtleben</button><button type="button" class="chip" data-chip="beauty">💇 Beauty &amp; Wellness</button><button type="button" class="chip" data-chip="hotels">🏨 Beherbergung</button><button type="button" class="chip" data-chip="gesundheit">🏥 Gesundheit</button><button type="button" class="chip" data-chip="altstadt">🗺️ Altstadt</button></section>
    <section class="controls is-collapsed" id="dashboardFilterControls" aria-label="Dashboard-Filter">
      <button class="filter-toggle" id="filterToggle" type="button" aria-expanded="false" aria-controls="dashboardFilterControls"><span><strong>Filter</strong><span class="filter-summary" id="filterSummary">Keine aktiven Filter</span></span><span class="filter-toggle-icon" aria-hidden="true">▾</span></button>
      <div class="control search"><label for="searchInput">Suche</label><input id="searchInput" type="search" placeholder="Name, PLZ, Kategorie, Löschbereich …" autocomplete="off"></div>
      <div class="control"><label for="postcodeFilter">PLZ</label><select id="postcodeFilter"><option value="">Alle PLZ</option>__POSTCODE_OPTIONS__</select></div>
      <div class="control"><label for="bezirkFilter">Bezirk</label><select id="bezirkFilter"><option value="">Alle Bezirke</option>__BEZIRK_OPTIONS__</select></div>
      <div class="control"><label for="bannerFilter">Banner</label><select id="bannerFilter"><option value="all">Alle</option><option value="banner">Mit Banner</option><option value="clean">Ohne Banner</option></select></div>
      <div class="control"><label for="rangeFilter">Gelöscht</label><select id="rangeFilter"><option value="">Alle Bereiche</option>__RANGE_OPTIONS__</select></div>
      <div class="control"><label for="categoryFilter">Kategorie</label><select id="categoryFilter"><option value="">Alle Kategorien</option>__CATEGORY_OPTIONS__</select></div>
      <div class="control"><label for="minReviews">Min. Rezensionen</label><input id="minReviews" type="number" min="0" step="1" value="0"></div>
      <button class="reset" id="resetFilters" type="button">Reset</button>
    </section>

    <section class="grid kpis" aria-label="Kennzahlen">
      <div class="card kpi"><span class="value" id="kpiPlaces">–</span><span class="label">Orte im Filter</span></div>
      <div class="card kpi"><span class="value" id="kpiBanners">–</span><span class="label">mit sichtbarem Banner</span></div>
      <div class="card kpi"><span class="value" id="kpiBannerPct">–</span><span class="label">Banner-Anteil</span></div>
      <div class="card kpi"><span class="value" id="kpiRemoved">–</span><span class="label">geschätzt entfernt</span></div>
      <div class="card kpi"><span class="value" id="kpiClean">–</span><span class="label">ohne sichtbaren Banner</span></div>
    </section>

__SEO_SUMMARY__

    <section class="grid panel-grid" aria-label="Top-Rankings">
      <article class="card panel"><h2>Meiste entfernte Bewertungen</h2><p>Sortiert nach geschätztem Mittelpunkt.</p><div class="bars" id="barsRemoved"></div></article>
      <article class="card panel"><h2>Höchste Lösch-Quote</h2><p>Entfernte / sichtbare + entfernte Bewertungen.</p><div class="bars" id="barsRatio"></div></article>
      <article class="card panel"><h2>Schlechtestes Worst-Case-Rating</h2><p>Modell: alle entfernten Bewertungen waren 1★.</p><div class="bars" id="barsWorst"></div></article>
      <article class="card panel"><h2>Beste Orte ohne Löschbanner</h2><p>Ohne sichtbaren Diffamierungs-Löschbanner, Rating zuerst — ab 100 Rezensionen.</p><div class="bars" id="barsClean"></div></article>
    </section>

    <section class="card dist" aria-label="Verteilung"><h2>Verteilung der Lösch-Stufen</h2><div id="distribution"></div></section>

    <section class="card bezirk-summary" aria-label="Bezirks-Gruppen"><h2>Gruppierung nach statistischem Bezirk</h2><p>Top-Bezirke im aktuellen Filter, sortiert nach Banner-Anteil. Anklicken setzt den Bezirksfilter.</p><div class="bezirk-list" id="bezirkSummary"></div></section>

    <section class="card bezirk-summary" aria-label="Kategorie-Gruppen"><h2>Gruppierung nach Kategorie</h2><p>Übergeordnete Kategorie-Gruppen im aktuellen Filter, sortiert nach Banner-Anteil.</p><div class="bezirk-list" id="parentSummary"></div></section>

    <section class="card map-panel" aria-label="Karte">
      <h2>Karte der erfassten Orte</h2>
      <p><span id="mapCount">–</span> Orte mit Koordinaten im aktuellen Filter. Marker anklicken markiert Einträge; Bezirksflächen anklicken setzt den Bezirkfilter.</p>
      <div id="placesMap"><div class="map-empty">Karte wird geladen …</div></div>
      <div class="map-legend"><span><i class="legend-area"></i>Bezirk, klickbar</span><span><i class="legend-dot" style="background:#1f6f8b"></i>dein Standort</span><span><i class="legend-dot" style="background:#c9332c"></i>hohe Lösch-Quote</span><span><i class="legend-dot" style="background:#ef7d16"></i>sichtbarer Banner</span><span><i class="legend-dot" style="background:#2d7b3f"></i>kein sichtbarer Banner</span></div>
    </section>

    <nav class="tabs" aria-label="Tabellen-Presets">
      <button class="tab" data-mode="removed">Meiste entfernt</button>
      <button class="tab active" data-mode="ratio">Höchste Lösch-Quote</button>
      <button class="tab" data-mode="worst">Worst-Case-Rating</button>
      <button class="tab" data-mode="clean">Ohne Löschbanner</button>
      <button class="tab" data-mode="nearby">In meiner Nähe</button>
    </nav>
    <div class="sub-chips" id="subChips"></div>
    <div class="nearby-status" id="nearbyStatus" role="status" aria-live="polite" hidden></div>

    <div class="table-head"><strong id="tableTitle">Höchste Lösch-Quote</strong><span id="resultCount">–</span></div>
    <section class="table-wrap" aria-label="Daten-Explorer">
      <table id="placesTable">
        <colgroup><col class="rank"><col class="name"><col class="bezirk"><col class="plz"><col class="rating"><col class="reviews"><col class="banner"><col class="removed"><col class="estimate"><col class="ratio"><col class="real"><col class="checked"><col class="category"></colgroup>
        <thead><tr>
          <th class="rank"><button data-sort="rank">Rang <span class="arrow"></span></button></th>
          <th><button data-sort="name">Name / Google Maps <span class="arrow"></span></button></th>
          <th><button data-sort="bezirkLabel">Bezirk <span class="arrow"></span></button></th>
          <th><button data-sort="postcode">PLZ <span class="arrow"></span></button></th>
          <th class="num"><button data-sort="rating">Rating <span class="arrow"></span></button></th>
          <th class="num"><button data-sort="reviewCount">Rezensionen <span class="arrow"></span></button></th>
          <th><button data-sort="hasBanner">Banner <span class="arrow"></span></button></th>
          <th class="num"><button data-sort="removedEstimate">Gelöscht <span class="arrow"></span></button></th>
          <th class="num"><button data-sort="removedEstimate">Schätzwert <span class="arrow"></span></button></th>
          <th class="num"><button data-sort="deletionRatioPct">Löschquote <span class="arrow"></span></button></th>
          <th class="num"><button data-sort="realRatingAdjusted">Worst-Case <span class="arrow"></span></button></th>
          <th><button data-sort="readAt">Geprüft <span class="arrow"></span></button></th>
          <th><button data-sort="category">Kategorie <span class="arrow"></span></button></th>
        </tr></thead>
        <tbody></tbody>
      </table>
    </section>
    <button class="show-more" id="showMoreRows" type="button" hidden>Mehr Zeilen anzeigen</button>
    <footer>
      <div>Quelle: Google Maps, öffentlich sichtbare Banner. „Kein Banner“ heißt nur: im Scrape war kein passender Hinweis sichtbar. Snapshot: __SNAPSHOT__.</div>
__ANALYTICS_PRIVACY__
      <div class="footer-credit">© 2026 Patrick Wozniak · <a href="https://patwoz.dev" target="_blank" rel="noopener noreferrer">patwoz.dev</a></div>
    </footer>
  </main>

  <script id="dashboardConfig" type="application/json">__CONFIG__</script>
  <script id="placesData" type="application/json">__DATA__</script>
  <script id="bezirkData" type="application/json">__BEZIRK_DATA__</script>
  <script>
__DASHBOARD_JS__
  </script>
</body>
</html>`

	return strings.NewReplacer(
		"__PAGE_TITLE__", esc(meta.PageTitle),
		"__PAGE_DESCRIPTION__", escAttr(meta.PageDescription),
		"__CANONICAL_URL__", escAttr(meta.CanonicalURL),
		"__SITE_NAME__", esc(meta.SiteName),
		"__SOCIAL_IMAGE__", escAttr(meta.SocialImageURL),
		"__SOCIAL_IMAGE_ALT__", escAttr(meta.SocialImageAlt),
		"__MODIFIED_TIME__", snapshot.Format(time.RFC3339),
		"__STRUCTURED_DATA__", structuredData,
		"__CITY__", esc(city),
		"__SEO_SUMMARY__", seoSummaryHTML(stats, city),
		"__POSTCODE_OPTIONS__", postcodeOptions,
		"__BEZIRK_OPTIONS__", bezirkOptions,
		"__RANGE_OPTIONS__", rangeOptions,
		"__CATEGORY_OPTIONS__", categoryOptions,
		"__DASHBOARD_JS__", dashboardJS,
		"__ANALYTICS__", plausibleAnalyticsSnippet(),
		"__ANALYTICS_PRIVACY__", plausiblePrivacyNotice(),
		"__SNAPSHOT__", snapshotDisplay,
		"__CONFIG__", configText,
		"__DATA__", jsonText,
		"__BEZIRK_DATA__", bezirkText,
	).Replace(page)
}

type compactClientData struct {
	Dictionaries [][]string      `json:"d"`
	Rows         [][]interface{} `json:"r"`
}

type stringDictionary struct {
	values []string
	index  map[string]int
}

const compactCoordScale = 100000.0

func compactClientDataJSON(data []clientRow, city string) string {
	postcodes := newStringDictionary()
	bezirke := newStringDictionary()
	categories := newStringDictionary()
	parentCategories := newStringDictionary()
	ranges := newStringDictionary()

	rows := make([][]interface{}, 0, len(data))
	for _, row := range data {
		compactRow := []interface{}{
			compactCID(row.ID),
			row.Name,
			postcodes.Add(row.Postcode),
			compactCoordinatePtr(row.Lat, 49),
			compactCoordinatePtr(row.Lng, 11),
			bezirke.Add(row.BezirkLabel),
			compactFloatPtr(row.Rating, 1),
			compactIntPtr(row.ReviewCount),
			categories.Add(row.Category),
			parentCategories.Add(row.ParentCategory),
			compactAddress(row.Address, row.Postcode, city),
			readAtMinute(row.ReadAt),
		}
		if row.HasBanner {
			compactRow = append(compactRow,
				ranges.Add(row.RemovedRange),
				compactFloat(row.RemovedEstimate, 1),
				compactFloatPtr(row.DeletionRatioPct, 2),
				compactFloatPtr(row.RealRatingAdjusted, 3),
			)
		}
		rows = append(rows, compactRow)
	}
	return safeJSON(compactClientData{
		Dictionaries: [][]string{postcodes.values, bezirke.values, categories.values, parentCategories.values, ranges.values},
		Rows:         rows,
	})
}

func compactBezirkDataJSON(boundaries []mapsreview.BezirkBoundary) string {
	rows := make([][]interface{}, 0, len(boundaries))
	for _, boundary := range boundaries {
		polygons := make([][]int, 0, len(boundary.Polygons))
		for _, polygon := range boundary.Polygons {
			flat := make([]int, 0, len(polygon)*2)
			prevLat := 0
			prevLng := 0
			for _, point := range polygon {
				if len(point) < 2 {
					continue
				}
				lat := compactCoordinate(point[0], 49)
				lng := compactCoordinate(point[1], 11)
				if len(flat) == 0 {
					flat = append(flat, lat, lng)
				} else {
					flat = append(flat, lat-prevLat, lng-prevLng)
				}
				prevLat = lat
				prevLng = lng
			}
			if len(flat) > 0 {
				polygons = append(polygons, flat)
			}
		}
		rows = append(rows, []interface{}{boundary.Label, polygons})
	}
	return safeJSON(rows)
}

func newStringDictionary() *stringDictionary {
	return &stringDictionary{index: map[string]int{}}
}

func (d *stringDictionary) Add(value string) int {
	if index, ok := d.index[value]; ok {
		return index
	}
	index := len(d.values)
	d.index[value] = index
	d.values = append(d.values, value)
	return index
}

func compactCID(id string) string {
	if _, after, ok := strings.Cut(id, ":"); ok {
		return strings.TrimPrefix(after, "0x")
	}
	return strings.TrimPrefix(id, "0x")
}

func compactCoordinatePtr(value *float64, base float64) interface{} {
	if value == nil {
		return nil
	}
	return compactCoordinate(*value, base)
}

func compactCoordinate(value float64, base float64) int {
	return int(math.Round((value - base) * compactCoordScale))
}

func compactAddress(address string, postcode string, cityName string) string {
	if address == "" {
		return ""
	}
	city := strings.TrimSpace(postcode + " " + cityName)
	if address == city {
		return ""
	}
	if before, ok := strings.CutSuffix(address, ", "+city); ok {
		return before
	}
	return "!" + address
}

func compactFloatPtr(value *float64, decimals int) interface{} {
	if value == nil {
		return nil
	}
	return compactFloat(*value, decimals)
}

func compactFloat(value float64, decimals int) interface{} {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	return roundDecimal(value, decimals)
}

func compactIntPtr(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func roundDecimal(value float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(value*factor) / factor
}

func readAtMinute(value string) int64 {
	if value == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, value)
	}
	if err != nil {
		return 0
	}
	return parsed.Unix() / 60
}

func safeJSON(value interface{}) string {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return strings.ReplaceAll(string(jsonData), "<", "\\u003c")
}

func snapshotTime(data []clientRow) time.Time {
	latest := time.Time{}
	for _, row := range data {
		if row.ReadAt == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, row.ReadAt)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339Nano, row.ReadAt)
		}
		if err == nil && parsed.After(latest) {
			latest = parsed
		}
	}
	if latest.IsZero() {
		return time.Now().UTC()
	}
	return latest.UTC()
}

func makeSEOStats(data []clientRow, snapshot string) seoStats {
	stats := seoStats{Total: len(data), Snapshot: snapshot}
	for _, row := range data {
		if row.HasBanner {
			stats.Banners++
			stats.RemovedEstimate += int(row.RemovedEstimate + 0.5)
		} else {
			stats.Clean++
		}
	}

	top := make([]clientRow, 0, len(data))
	for _, row := range data {
		if row.HasBanner && row.RemovedEstimate > 0 {
			top = append(top, row)
		}
	}
	sort.SliceStable(top, func(i, j int) bool {
		if top[i].RemovedEstimate != top[j].RemovedEstimate {
			return top[i].RemovedEstimate > top[j].RemovedEstimate
		}
		return top[i].Name < top[j].Name
	})
	if len(top) > 8 {
		top = top[:8]
	}
	stats.Top = top
	return stats
}

func seoSummaryHTML(stats seoStats, city string) string {
	return fmt.Sprintf(`<section class="card seo-summary" aria-labelledby="data-overview-title">
      <h2 id="data-overview-title">Datenstand: Google-Maps-Bewertungen und Löschbanner in %s</h2>
      <p>Dieses Dashboard macht öffentlich sichtbare Hinweise auf wegen Diffamierungsbeschwerden entfernte Google-Maps-Bewertungen in %s durchsuchbar. Die Karte, Filter und Ranglisten zeigen Löschbanner, geschätzte entfernte Bewertungen, Löschquoten und Worst-Case-Ratings je Ort.</p>
      <ul class="seo-facts">
        <li><strong>%s</strong> erfasste Orte</li>
        <li><strong>%s</strong> mit sichtbarem Löschbanner</li>
        <li><strong>%s</strong> geschätzt entfernte Bewertungen</li>
        <li><strong>%s</strong> ohne sichtbaren Banner</li>
      </ul>
      <p class="seo-meta">Datenstand: %s. Kein sichtbarer Banner bedeutet nur, dass beim Scrape kein passender Hinweis sichtbar war.</p>
%s
    </section>`,
		esc(city),
		esc(city),
		mapsreview.FormatGermanInt(stats.Total),
		mapsreview.FormatGermanInt(stats.Banners),
		mapsreview.FormatGermanInt(stats.RemovedEstimate),
		mapsreview.FormatGermanInt(stats.Clean),
		esc(stats.Snapshot),
		seoTopListHTML(stats.Top),
	)
}

func seoTopListHTML(rows []clientRow) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="seo-top"><h3>Top-Orte nach geschätzten entfernten Bewertungen</h3><ol>`)
	for _, row := range rows {
		label := esc(row.Name)
		if row.URL != "" {
			label = fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener noreferrer nofollow">%s</a>`, escAttr(row.URL), label)
		}
		meta := []string{}
		if row.RemovedRange != "" {
			meta = append(meta, "Bereich "+row.RemovedRange)
		}
		if row.Postcode != "" {
			meta = append(meta, "PLZ "+row.Postcode)
		}
		if row.BezirkLabel != "" {
			meta = append(meta, row.BezirkLabel)
		}
		detail := ""
		if len(meta) > 0 {
			detail = esc(strings.Join(meta, " · ")) + " · "
		}
		fmt.Fprintf(&b, `<li>%s <span class="seo-meta">%sSchätzwert ca. %s entfernte Bewertungen</span></li>`,
			label,
			detail,
			mapsreview.FormatGermanInt(int(row.RemovedEstimate+0.5)),
		)
	}
	b.WriteString(`</ol></div>`)
	return b.String()
}

func structuredDataJSON(stats seoStats, snapshot time.Time, city string, meta dashboardMetadata) string {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@graph": []map[string]interface{}{
			{
				"@type":      "WebSite",
				"@id":        meta.CanonicalURL + "#website",
				"url":        meta.CanonicalURL,
				"name":       meta.SiteName,
				"inLanguage": "de-DE",
				"publisher": map[string]interface{}{
					"@type": "Person",
					"name":  "Patrick Wozniak",
					"url":   "https://patwoz.dev",
				},
			},
			{
				"@type":       "WebPage",
				"@id":         meta.CanonicalURL + "#webpage",
				"url":         meta.CanonicalURL,
				"name":        meta.PageTitle,
				"description": meta.PageDescription,
				"isPartOf": map[string]interface{}{
					"@id": meta.CanonicalURL + "#website",
				},
				"mainEntity": map[string]interface{}{
					"@id": meta.CanonicalURL + "#dataset",
				},
				"primaryImageOfPage": map[string]interface{}{
					"@type": "ImageObject",
					"url":   meta.SocialImageURL,
				},
				"dateModified": snapshot.Format("2006-01-02"),
				"inLanguage":   "de-DE",
			},
			{
				"@type":                "Dataset",
				"@id":                  meta.CanonicalURL + "#dataset",
				"name":                 fmt.Sprintf("%s Google-Maps-Bewertungen mit sichtbaren Löschbanner-Hinweisen", city),
				"description":          meta.PageDescription,
				"url":                  meta.CanonicalURL,
				"dateModified":         snapshot.Format("2006-01-02"),
				"temporalCoverage":     snapshot.Format("2006-01-02"),
				"measurementTechnique": "Scrape öffentlich sichtbarer Google-Maps-Ortsseiten",
				"keywords": []string{
					city,
					"Google Maps Bewertungen",
					"entfernte Bewertungen",
					"Löschbanner",
					"Diffamierungsbeschwerden",
				},
				"spatialCoverage": map[string]interface{}{
					"@type": "City",
					"name":  city,
				},
				"variableMeasured": []string{
					"Orte",
					"sichtbare Löschbanner",
					"geschätzte entfernte Bewertungen",
					"Löschquote",
					"Worst-Case-Rating",
				},
				"size": fmt.Sprintf("%d Orte, %d sichtbare Löschbanner", stats.Total, stats.Banners),
				"creator": map[string]interface{}{
					"@type": "Person",
					"name":  "Patrick Wozniak",
					"url":   "https://patwoz.dev",
				},
			},
		},
	}
	jsonData, _ := json.MarshalIndent(data, "  ", "  ")
	return strings.ReplaceAll(string(jsonData), "<", "\\u003c")
}

func plausibleAnalyticsSnippet() string {
	src := plausibleAnalyticsSrc()
	if src == "" {
		return ""
	}
	domain := strings.TrimSpace(os.Getenv("DASHBOARD_ANALYTICS_DOMAIN"))
	if domain != "" {
		return fmt.Sprintf(`  <!-- Privacy-friendly analytics by Plausible -->
  <script defer data-domain="%s" src="%s"></script>`, escAttr(domain), escAttr(src))
	}
	return fmt.Sprintf(`  <!-- Privacy-friendly analytics by Plausible -->
  <script async src="%s"></script>
  <script>
    window.plausible=window.plausible||function(){(plausible.q=plausible.q||[]).push(arguments)},plausible.init=plausible.init||function(i){plausible.o=i||{}};
    plausible.init()
  </script>`, escAttr(src))
}

func plausiblePrivacyNotice() string {
	src := plausibleAnalyticsSrc()
	if src == "" {
		return ""
	}
	host := analyticsHost(src)
	hostText := ""
	if host != "" {
		hostText = fmt.Sprintf(` Anbieter-Domain: <code>%s</code>.`, esc(host))
	}
	return fmt.Sprintf(`<div class="footer-privacy">Diese Website nutzt Plausible Analytics, eine datenschutzfreundliche Webanalyse ohne Cookies. Die Auswertung erfolgt aggregiert und ohne personenbezogene Nutzerprofile.%s</div>`, hostText)
}

func plausibleAnalyticsSrc() string {
	return strings.TrimSpace(os.Getenv("DASHBOARD_ANALYTICS_SRC"))
}

func analyticsHost(src string) string {
	if _, after, ok := strings.Cut(src, "://"); ok {
		src = after
	}
	host, _, _ := strings.Cut(src, "/")
	return host
}

func allBezirkLabels() []string {
	bezirke := mapsreview.AllBezirke()
	out := make([]string, 0, len(bezirke))
	for _, bezirk := range bezirke {
		out = append(out, bezirk.ID+" "+bezirk.Name)
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(data []clientRow, value func(clientRow) string) []string {
	set := map[string]bool{}
	for _, row := range data {
		v := value(row)
		if v != "" {
			set[v] = true
		}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func countRows(data []clientRow, keep func(clientRow) bool) int {
	count := 0
	for _, row := range data {
		if keep(row) {
			count++
		}
	}
	return count
}

func maxEstimateForRange(data []clientRow, r string) float64 {
	max := 0.0
	for _, row := range data {
		if row.RemovedRange == r && row.RemovedEstimate > max {
			max = row.RemovedEstimate
		}
	}
	return max
}

func esc(value string) string     { return html.EscapeString(value) }
func escAttr(value string) string { return html.EscapeString(value) }
