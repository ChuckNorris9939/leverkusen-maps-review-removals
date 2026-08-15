# Leverkusen Maps Review Removals

Reproduzierbarer lokaler Go-Workflow, um öffentlich sichtbare Google-Maps-Ortsdaten zu sammeln, Hinweise auf entfernte Bewertungen zu erkennen, zum Beispiel:

> „21 bis 50 Bewertungen aufgrund von Beschwerden wegen Diffamierung entfernt.“

…und daraus Leverkusen-Auswertungen sowie ein interaktives Dashboard zu erzeugen.

## Wichtige Hinweise

- Nur für private Recherche / Journalismus gedacht. Google-Maps-Bedingungen und geltendes Recht beachten.
- Der Scraper speichert nur, was zum Scrape-Zeitpunkt öffentlich sichtbar ist. Manuell geprüfte Abweichungen können als Overrides in `internal/mapsreview/data/place_overrides.json` gepflegt werden.
- Kein Banner ≠ definitiv keine entfernten Bewertungen. Es bedeutet nur: Beim Scrape wurde kein passender sichtbarer Hinweis erkannt.
- Das angepasste Rating nimmt an, dass alle entfernten Bewertungen 1-Stern-Bewertungen waren. Das ist ein Worst-Case-Modell, keine Tatsache.
- Langsame Delays verwenden. Wenn Google ein CAPTCHA zeigt: stoppen oder im sichtbaren Browser manuell lösen.

## Einrichtung

Voraussetzungen:

- Go 1.25+
- Chrome oder Chromium im `PATH` oder an einem Standard-Installationsort
- Optional als experimentelles CDP-Backend: Lightpanda
- Optional für Places-API-Discovery: Google Places API (New)-API-Key in `config.toml`
- Optional für PNG-Export: ImageMagick `magick` oder `convert`

```bash
make setup
# oder direkt:
go mod download
```

Für die optionale Places-API-Discovery `config.toml` aus `.config.toml.example` anlegen und `[places_api].api_key` setzen. `config.toml` bleibt lokal/git-ignoriert.

## 1) Daten sammeln

Standardmäßig nutzt der Scraper Chrome. Er liest die normale Google-Maps-Seite für Metadaten und die direkte Rezensionen-URL für Rating, Rezensionszahl und Löschbanner, weil die normale Maps-Ansicht Löschbanner teils nicht im DOM enthält.

Der Workflow ist immer zweistufig:

1. **Discovery** schreibt/erweitert `output/discovery.json`.
2. **Scrape** liest `output/discovery.json`, öffnet die Orte in Google Maps im Browser und schreibt `output/places.json` / `output/places.csv`.

Die Löschbanner-Erkennung passiert in beiden Varianten im Browser auf Google Maps. Die Places API wird nur optional für Discovery verwendet.

### Variante A: Discovery ohne Places API

Diese Variante nutzt nur den Browser: Google-Maps-Suchen werden geöffnet, sichtbare Ergebnislinks gesammelt und danach gescrapt.

```bash
# 1. Orte über Google-Maps-Suchergebnisse finden
make scrape ARGS="--city Leverkusen --postcodes 51371,51373,51375,51377,51379,51381 --discovery-only --headless=false"

# 2. Gefundene Orte im Browser scrapen, inklusive Rezensionen/Löschbanner
make scrape ARGS="--city Leverkusen --scrape-only --headless=false"
```

Vorteile: kein API-Key, keine Google-Cloud-Quota, kein API-Billing-Risiko. Nachteile: langsamer, stärker abhängig von der Google-Maps-Oberfläche und der sichtbaren Ergebnisliste.

### Variante B: Discovery mit Places API

Diese Variante nutzt die offizielle Places API (New) nur für die Ortssuche. Die Text-Search-Anfrage ist bewusst auf ID-only-Felder beschränkt (`places.id,nextPageToken`).

Danach ist der Ablauf identisch: Die gefundenen `ChIJ...`-Place-IDs werden als Google-Maps-URLs in `output/discovery.json` gespeichert und im Browser gescrapt. Beim Scrape löst Google Maps die URL auf eine kanonische `/maps/place/.../data=...` URL auf; diese wird anschließend in `output/places.json` gespeichert, damit spätere Läufe direktere Maps-URLs/IDs haben.

```bash
# 1. Orte über Places API Text Search finden
make scrape ARGS="--city Leverkusen --postcodes 51371,51373,51375,51377,51379,51381 --places-api-discovery --discovery-only --places-api-pages 1"

# 2. Gefundene Orte im Browser scrapen, inklusive Rezensionen/Löschbanner
make scrape ARGS="--city Leverkusen --scrape-only --headless=false"
```

Vollständiger Leverkusen-Lauf:

```bash
make scrape ARGS="--city Leverkusen --postcodes 51371,51373,51375,51377,51379,51381 --headless=false"
```

Kleiner Testlauf:

```bash
make scrape ARGS="--city Leverkusen --postcodes 51373 --queries restaurant,café --max-results 20 --headless=false"
```

Ausgaben:

- `output/discovery.json` — gefundene Google-Maps-Orte
- `output/places.json` — gescrapte Daten inklusive Koordinaten
- `output/places.csv` — CSV-Export für Tabellenkalkulationen
- `output/metadata.json` — Scrape-Einstellungen, Zählwerte, Zeitstempel und User-Agent

Nützliche Optionen:

```bash
--city Leverkusen
--postcodes 51371,51373,51375,51377,51379,51381
--queries restaurant,café,imbiss,pizzeria,bäckerei
--discovery-only
--scrape-only
--scrape-only --rescrape-all
--scrape-only --banner-audit-only --notice-attempts 2
--delay-min 4000 --delay-max 9000
--out output/places.json --csv output/places.csv --discovery output/discovery.json --metadata output/metadata.json
```

## 2) Datenqualität verbessern

Fehlende Adressen nachtragen:

```bash
make backfill ARGS="--headless=true --concurrency 4"
```

Scrape-Ergebnis validieren:

```bash
make validate
```

## 3) Diagramme und Dashboard erzeugen

```bash
make charts ARGS="--city Leverkusen --png"
make dashboard
```

### Dashboard-Konfiguration

Optional kann `cmd/dashboard` eine lokale TOML-Konfiguration laden. Wenn `config.toml` im Projektroot existiert, wird sie automatisch verwendet:

```bash
cp .config.toml.example config.toml
$EDITOR config.toml

go run ./cmd/dashboard --config config.toml
# oder implizit, wenn config.toml existiert:
go run ./cmd/dashboard
```

Beispiel:

```toml
city = "Leverkusen"
input = "output/places.json"
output = "output/charts/leverkusen_dashboard.html"
site_domain = "leverkusen-maps-review-removals.de"
site_url = "https://leverkusen-maps-review-removals.de"
site_output = "public"

[legal]
enabled = true
name = "Dein Name"
email = "kontakt@leverkusen-maps-review-removals.de"
address_lines = [
  "Musterstraße 1",
  "51373 Leverkusen",
  "Deutschland",
]
note = "Angaben gemäß § 5 TMG"
post_handler = "Dein Name"
```

Ausgaben:

- `output/charts/leverkusen_dashboard.html` — interaktive App mit KPIs, Filtern, Karte, sortierbarer Explorer-Tabelle und Google-Maps-Links
- `output/charts/leverkusen_overall_summary.svg/.png`
- `output/charts/leverkusen_most_removed.csv`
- `output/charts/leverkusen_most_removed.md`
- `output/charts/leverkusen_most_removed.html`

## Veröffentlichung mit GitHub Pages

GitHub Pages ist auf den Branch `gh-pages` konfiguriert. Der Branch enthält nur das generierte `public/`-Artefakt; die Quell- und Snapshot-Dateien bleiben auf `main`.

Lokale Vorschau des Veröffentlichungs-Artefakts:

```bash
make site
python3 -m http.server --directory public 8080
```

Für GitHub Actions muss der Inhalt der lokalen `config.toml` als Repository-Secret `DASHBOARD_CONFIG_TOML` hinterlegt werden. Der Workflow baut bei jedem Push auf `main` automatisch das Dashboard und aktualisiert `gh-pages`.

Veröffentlichen:

```bash
make deploy-pages
```

Im GitHub-Repository muss dafür **Settings → Pages → Source: Deploy from a branch**, Branch `gh-pages`, Ordner `/` aktiv sein.

## GitHub Actions

Der Workflow `.github/workflows/refresh-and-deploy.yml` baut und veröffentlicht GitHub Pages bei jedem Push auf `main` neu.

## Standardmäßig enthaltene Leverkusen PLZ

`51371, 51373, 51375, 51377, 51379, 51381`

## Lizenz

MIT, siehe [`LICENSE`](LICENSE).
