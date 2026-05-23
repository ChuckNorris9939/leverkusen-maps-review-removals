package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

type args struct {
	Config             string
	City               string
	Postcodes          []string
	Queries            []string
	MaxResults         int
	Headless           bool
	CDPURL             string
	Discovery          string
	Metadata           string
	PlacesAPIDiscovery bool
	PlacesAPIPageLimit int
	DiscoveryOnly      bool
	ScrapeOnly         bool
	RescrapeAll        bool
	BannerAuditOnly    bool
	AllowBannerClears  bool
	NoticeAttempts     int
	ScrapeStart        int
	ScrapeLimit        int
	SaveEvery          int
	DelayMin           int
	DelayMax           int
	Out                string
	CSV                string
	DashboardAddr      string
}

func parseArgs(argv []string) (args, error) {
	csvSet := false
	postcodesSet := false
	postcodesValue := ""
	out := args{
		Config:             "config.toml",
		City:               mapsreview.DefaultCity,
		Postcodes:          mapsreview.NurembergPostcodes,
		Queries:            mapsreview.DefaultQueries,
		Headless:           false,
		DelayMin:           2500,
		DelayMax:           6000,
		SaveEvery:          1,
		NoticeAttempts:     2,
		DashboardAddr:      ":8081",
		Discovery:          mapsreview.DiscoveryJSON,
		Metadata:           mapsreview.MetadataJSON,
		PlacesAPIPageLimit: 1,
		Out:                mapsreview.ResultsJSON,
		CSV:                mapsreview.ResultsCSV,
		MaxResults:         0,
		ScrapeStart:        1,
	}

	for i := 0; i < len(argv); i++ {
		key, value, consume := mapsreview.SplitArg(argv, i)
		switch key {
		case "--config":
			out.Config = value
		case "--city":
			out.City = value
		case "--postcodes":
			postcodesSet = true
			postcodesValue = strings.TrimSpace(value)
			if value == "" || value == "all" {
				out.Postcodes = mapsreview.NurembergPostcodes
			} else {
				out.Postcodes = splitCSV(value)
			}
		case "--queries":
			out.Queries = splitCSV(value)
		case "--max-results":
			out.MaxResults = mapsreview.Atoi(value)
		case "--headless":
			out.Headless = mapsreview.ParseBool(value, true)
		case "--cdp-url":
			out.CDPURL = value
		case "--discovery":
			out.Discovery = value
		case "--metadata":
			out.Metadata = value
		case "--places-api-discovery":
			out.PlacesAPIDiscovery = true
			consume = false
		case "--places-api-pages":
			out.PlacesAPIPageLimit = max(1, mapsreview.Atoi(value))
		case "--discovery-only":
			out.DiscoveryOnly = true
			consume = false
		case "--scrape-only":
			out.ScrapeOnly = true
			consume = false
		case "--rescrape-all", "--all":
			out.RescrapeAll = true
			consume = false
		case "--banner-audit-only":
			out.BannerAuditOnly = true
			consume = false
		case "--allow-banner-clears":
			out.AllowBannerClears = true
			consume = false
		case "--scrape-start", "--resume-from":
			out.ScrapeStart = max(1, mapsreview.Atoi(value))
		case "--scrape-limit":
			out.ScrapeLimit = max(0, mapsreview.Atoi(value))
		case "--save-every":
			out.SaveEvery = max(1, mapsreview.Atoi(value))
		case "--notice-attempts":
			out.NoticeAttempts = max(1, mapsreview.Atoi(value))
		case "--delay-min":
			out.DelayMin = mapsreview.Atoi(value)
		case "--delay-max":
			out.DelayMax = mapsreview.Atoi(value)
		case "--out":
			out.Out = value
		case "--csv":
			out.CSV = value
			csvSet = true
		case "--dashboard":
			out.DashboardAddr = value
		case "--help", "-h":
			printHelp()
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
	if !mapsreview.IsDefaultCity(out.City) && (!postcodesSet || postcodesValue == "" || strings.EqualFold(postcodesValue, "all")) {
		return out, fmt.Errorf("explicit --postcodes CSV is required when --city is not %s", mapsreview.DefaultCity)
	}
	if !csvSet && out.Out != "" {
		out.CSV = strings.TrimSuffix(out.Out, filepath.Ext(out.Out)) + ".csv"
	}
	return out, nil
}

func printHelp() {
	fmt.Printf(`Usage:
  go run ./cmd/scrape --postcodes all --headless=false
  go run ./cmd/scrape --postcodes 90402,90403 --queries restaurant,café,imbiss

Options:
  --config <path>           TOML config path for optional Places API key. Default: config.toml.
  --city <name>             City name for discovery. Default: %s. For other cities, pass --postcodes explicitly.
  --postcodes <all|csv>     PLZ list. Default: all known Nürnberg PLZ.
  --queries <csv>           Google Maps search terms. Default: %s.
  --max-results <n>         Stop after n discovered places. 0 = unlimited.
  --headless <true|false>   Chrome headless mode. Default: false; safer for consent/CAPTCHA.
  --cdp-url <ws-url>        Experimental: use an existing CDP browser instead of Chrome, e.g. Lightpanda on ws://127.0.0.1:9333.
  --discovery <path>        Discovery JSON path. Default: output/discovery.json.
  --metadata <path>         Metadata JSON path. Default: output/metadata.json.
  --places-api-discovery    Use official Places API Text Search ID-only discovery. Reads [places_api].api_key from config.toml.
  --places-api-pages <n>    Places API result pages per postcode/query. Default: 1 (default searches stay under 1,000 requests/day).
  --discovery-only          Only create/update the discovery JSON.
  --scrape-only             Skip discovery; scrape the discovery JSON.
  --rescrape-all, --all     Re-read every discovered place, including existing success rows.
  --banner-audit-only       Scan existing no-banner success rows for missed banners; only newly found banners are written.
  --allow-banner-clears     Allow a re-scrape to remove a previously seen deletion banner. Default: keep old banner until manually verified.
  --notice-attempts <n>     Direct-reviews attempts for banner-clear verification and banner audit. Default: 2.
  --scrape-start <n>        Start scraping at 1-based position within the todo list. Default: 1.
  --resume-from <n>         Alias for --scrape-start.
  --scrape-limit <n>        Scrape at most n todo rows. 0 = unlimited.
  --save-every <n>          Persist results every n changed rows. Default: 1.
  --delay-min <ms>          Minimum delay between place pages. Default: 2500.
  --delay-max <ms>          Maximum delay between place pages. Default: 6000.
  --out <path>              Results JSON path. Default: output/places.json.
  --csv <path>              Results CSV path. Default: output/places.csv.
  --dashboard <addr>        Scrape dashboard listen address. Default: :8081.
`, mapsreview.DefaultCity, strings.Join(mapsreview.DefaultQueries, ","))
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
