package mapsreview

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed dashboard.html
var dashboardHTML string

// Dashboard tracks scraper progress and serves a live dashboard.
type Dashboard struct {
	mu sync.Mutex

	Phase       string    // "discovery", "scraping", "done"
	Places      int       // total discovered places
	Scraped     int       // number scraped so far
	Todo        int       // total to scrape
	Errors      int       // error count
	Banners     int       // defamation banners found
	StartTime   time.Time // when the scraper started
	ElapsedSecs int       // cached elapsed seconds

	// Log tail
	logLines []string
	// Error details (place → error)
	errorDetails []string
}

// NewDashboard creates a dashboard and starts the HTTP server on the given port.
func NewDashboard(addr string) *Dashboard {
	d := &Dashboard{
		Phase:     "starting",
		StartTime: time.Now(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.serveHTML)
	mux.HandleFunc("/api", d.serveAPI)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Fprintf(os.Stderr, "dashboard server: %v\n", err)
		}
	}()
	return d
}

// SetPhase updates the current phase.
func (d *Dashboard) SetPhase(phase string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Phase = phase
}

// SetDiscoveryCount updates the discovered places count.
func (d *Dashboard) SetDiscoveryCount(n int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Places = n
}

// SetScrapeProgress updates scrape progress.
func (d *Dashboard) SetScrapeProgress(scraped, todo int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Scraped = scraped
	d.Todo = todo
}

// AddError records a scraping error.
func (d *Dashboard) AddError(place string, errMsg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Errors++
	d.errorDetails = append(d.errorDetails, fmt.Sprintf("%s → ERROR: %s", place, errMsg))
	if len(d.errorDetails) > 50 {
		d.errorDetails = d.errorDetails[len(d.errorDetails)-50:]
	}
}

// AddBanner records a found defamation banner.
func (d *Dashboard) AddBanner() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Banners++
}

// LogLine appends a line to the in-memory log tail.
func (d *Dashboard) LogLine(line string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logLines = append(d.logLines, line)
	if len(d.logLines) > 100 {
		d.logLines = d.logLines[len(d.logLines)-100:]
	}
}

// Logf formats and appends a log line.
func (d *Dashboard) Logf(format string, args ...interface{}) {
	d.LogLine(fmt.Sprintf(format, args...))
}

func (d *Dashboard) snapshot() map[string]interface{} {
	d.mu.Lock()
	defer d.mu.Unlock()

	speed := 0.0
	speedHr := 0.0
	eta := ""
	elapsed := int(time.Since(d.StartTime).Seconds())
	d.ElapsedSecs = elapsed

	if d.Scraped > 0 && elapsed > 0 {
		speed = float64(d.Scraped) / (float64(elapsed) / 60.0)
		speedHr = speed * 60
		remaining := d.Todo - d.Scraped
		if speed > 0 {
			etaSec := int((float64(remaining) / speed) * 60)
			if etaSec > 0 {
				h := etaSec / 3600
				m := (etaSec % 3600) / 60
				if h > 0 {
					eta = fmt.Sprintf("%dh %dm", h, m)
				} else {
					eta = fmt.Sprintf("%dm", m)
				}
			}
		}
	}

	elapsedStr := formatSeconds(elapsed)
	tail := ""
	if len(d.logLines) > 50 {
		tail = strings.Join(d.logLines[len(d.logLines)-50:], "\n")
	} else {
		tail = strings.Join(d.logLines, "\n")
	}

	return map[string]interface{}{
		"places":      d.Places,
		"scraped":     d.Scraped,
		"todo":        d.Todo,
		"elapsed":     elapsedStr,
		"elapsed_sec": elapsed,
		"phase":       d.Phase,
		"errors":      d.Errors,
		"banners":     d.Banners,
		"speed":       round1(speed),
		"speed_hr":    int(speedHr + 0.5),
		"eta":         eta,
		"error_lines": d.errorDetails,
		"tail":        tail,
	}
}

func (d *Dashboard) serveAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d.snapshot())
}

func (d *Dashboard) serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

func formatSeconds(s int) string {
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%dm %ds", s/60, s%60)
	}
	return fmt.Sprintf("%dh %dm %ds", s/3600, (s%3600)/60, s%60)
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
