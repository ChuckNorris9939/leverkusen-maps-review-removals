# Nürnberg Google-Bewertungen: Diffamierungs-Löschbanner

Reproduzierbarer lokaler Go-Workflow, um öffentlich sichtbare Google-Maps-Ortsdaten zu sammeln, Hinweise auf entfernte Bewertungen zu erkennen, zum Beispiel:

> „21 bis 50 Bewertungen aufgrund von Beschwerden wegen Diffamierung entfernt.“

…und daraus Nürnberg-Auswertungen sowie ein interaktives Dashboard zu erzeugen.

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
- Optional für Places-API-Discovery: Google Places API (New)-API-Key in `.env` (siehe `.env.example`)
- Optional für PNG-Export: ImageMagick `magick` oder `convert`

```bash
make setup
# oder direkt:
go mod download
```

Für die optionale Places-API-Discovery:

```bash
cp .env.example .env
# GOOGLE_MAPS_API_KEY in .env setzen
```

`.env` wird nur beim Lauf mit `--places-api-discovery` gelesen und bleibt lokal/git-ignoriert.

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
make scrape ARGS="--discovery-only --postcodes all --headless=false"

# 2. Gefundene Orte im Browser scrapen, inklusive Rezensionen/Löschbanner
make scrape ARGS="--scrape-only --headless=false"
```

Vorteile: kein API-Key, keine Google-Cloud-Quota, kein API-Billing-Risiko. Nachteile: langsamer, stärker abhängig von der Google-Maps-Oberfläche und der sichtbaren Ergebnisliste.

### Variante B: Discovery mit Places API

Diese Variante nutzt die offizielle Places API (New) nur für die Ortssuche. Die Text-Search-Anfrage ist bewusst auf ID-only-Felder beschränkt:

```text
places.id,nextPageToken
```

Danach ist der Ablauf identisch: Die gefundenen `ChIJ...`-Place-IDs werden als Google-Maps-URLs in `output/discovery.json` gespeichert und im Browser gescrapt. Beim Scrape löst Google Maps die URL auf eine kanonische `/maps/place/.../data=...` URL auf; diese wird anschließend in `output/places.json` gespeichert, damit spätere Läufe direktere Maps-URLs/IDs haben.

```bash
# 1. Orte über Places API Text Search finden
make scrape ARGS="--places-api-discovery --discovery-only --places-api-pages 1"

# 2. Gefundene Orte im Browser scrapen, inklusive Rezensionen/Löschbanner
make scrape ARGS="--scrape-only --headless=false"
```

Für tiefere Discovery, wenn die Tagesquota entsprechend gesetzt ist:

```bash
make scrape ARGS="--places-api-discovery --discovery-only --places-api-pages 2"
make scrape ARGS="--scrape-only --headless=false"
```

Vorteile: bessere und stabilere Discovery-Abdeckung. Nachteile: API-Key, Quota-Management und Billing-Monitoring nötig. Die API liefert keine Löschbanner; dafür bleibt immer der Browser-Scrape nötig.

Vollständiger Nürnberg-Lauf mit der Standard-Browser-Discovery:

```bash
make scrape ARGS="--postcodes all --headless=false"
```

Kleiner Testlauf:

```bash
make scrape ARGS="--postcodes 90402 --queries restaurant,café --max-results 20 --headless=false"
```

Ausgaben:

- `output/discovery.json` — gefundene Google-Maps-Orte
- `output/places.json` — gescrapte Daten inklusive Koordinaten und, sofern zuordenbar, `bezirkId` / `bezirkName`
- `output/places.csv` — CSV-Export für Tabellenkalkulationen
- `output/metadata.json` — Scrape-Einstellungen, Zählwerte, Zeitstempel und User-Agent

Nützliche Optionen:

```bash
--city Nürnberg
--postcodes 90402,90403
--queries restaurant,café,imbiss,pizzeria,bäckerei
--discovery-only
--places-api-discovery --discovery-only   # experimentell: offizielle Places API Text Search, ID-only/no-cost-SKU laut Google-Preisliste; liest GOOGLE_MAPS_API_KEY aus Umgebung oder .env
--places-api-pages 1                       # API-Seiten pro PLZ/Suche; Default 1 hält die Standardsuchen unter 1.000 Requests/Tag
--scrape-only
--scrape-only --rescrape-all   # alle gefundenen Orte erneut lesen, auch bereits erfolgreiche
--scrape-only --rescrape-all --allow-banner-clears   # zuvor erkannte Banner nach manueller Prüfung entfernen lassen
--scrape-only --banner-audit-only --notice-attempts 2   # no-banner-Zeilen gezielt auf übersehene Banner prüfen; schreibt nur neu gefundene Banner
--scrape-only --rescrape-all --resume-from 1288   # vollständigen Rescan an 1-basierter Todo-Position fortsetzen
--scrape-only --rescrape-all --resume-from 1288 --scrape-limit 200   # sichereren Teil-Scan ausführen
--delay-min 4000 --delay-max 9000
--out output/places.json --csv output/places.csv
--discovery output/discovery.json --metadata output/metadata.json
```

Für andere Städte kann `--city` den Suchort und die Darstellung anpassen. Dann muss die PLZ-Liste explizit per `--postcodes` übergeben werden, damit nicht versehentlich die Nürnberger Standard-PLZ genutzt werden. Die statistischen Bezirke und Kartenflächen sind Nürnberg-spezifisch; bei anderen Städten zeigt das Dashboard keine Nürnberger Bezirksflächen. Diagrammdateien nutzen für andere Städte einen City-Slug als Präfix, z. B. `fuerth_overall_summary.svg`.

Optional kann der Scraper über CDP gegen einen bereits laufenden Browser wie Lightpanda laufen. Das ist experimentell; Chrome bleibt der Standard und war in Stichproben schneller:

```bash
LIGHTPANDA_DISABLE_TELEMETRY=true lightpanda serve --host 127.0.0.1 --port 9333
make scrape ARGS="--scrape-only --rescrape-all --cdp-url ws://127.0.0.1:9333 --save-every 25 --delay-min 4000 --delay-max 9000"
```

Lightpanda ist als Vergleichs- oder Fallback-Backend nützlich, sollte aber mit Stichproben gegen Chrome geprüft werden, bevor seine Ergebnisse übernommen werden.

Nach einem Refresh kann ein konservativer Banner-Audit helfen, zuvor übersehene Löschbanner zu finden. Der Audit prüft nur bestehende erfolgreiche Zeilen ohne Banner und schreibt ausschließlich neu gefundene Banner; bestehende Banner werden dabei nie entfernt:

```bash
make scrape ARGS="--scrape-only --banner-audit-only --notice-attempts 2 --save-every 25 --delay-min 4000 --delay-max 9000"
```

## 2) Datenqualität verbessern

Fehlende Adressen nachtragen:

```bash
make backfill ARGS="--headless=true --concurrency 4"
```

Scrape-Ergebnis validieren:

```bash
make validate
go run ./cmd/validate --strict-nuremberg
```

Die Validierung meldet fehlende Adressen, fehlende Ratings/Rezensionszahlen, fehlende Nürnberg-Bezirkszuordnungen, Nicht-Nürnberger Postleitzahlen, doppelte URLs/IDs und Banner-Zeilen mit Parse-Problemen.

## 3) Diagramme und Dashboard erzeugen

```bash
make charts ARGS="--png"
make dashboard
```

Ausgaben:

- `output/charts/nuernberg_dashboard.html` — interaktive App mit KPIs, Filtern, Karte, sortierbarer Explorer-Tabelle und Google-Maps-Links
- `output/charts/nuernberg_overall_summary.svg/.png`
- `output/charts/nuernberg_90402_summary.svg/.png`
- `output/charts/nuernberg_most_removed.csv`
- `output/charts/nuernberg_most_removed.md`
- `output/charts/nuernberg_most_removed.html`

Wenn ImageMagick nicht installiert ist, überspringt `--png` die PNG-Dateien und schreibt weiterhin SVGs.

Die erzeugten Diagramm- und Dashboard-Dateien unter `output/charts/` werden von git ignoriert. Im Repository bleiben nur die Scrape-Snapshots (`output/places.json`, `output/places.csv`, `output/metadata.json`, optional `output/discovery.json`) versioniert; `make site` baut daraus `public/` für GitHub Pages neu.

Die Dashboard-Karte nutzt Leaflet mit CARTO-Kartenkacheln auf Basis von OpenStreetMap-Daten. Beim Öffnen der HTML-Datei ist deshalb Internetzugriff für Kartenkacheln nötig. Das Dashboard gruppiert, filtert und überlagert Einträge außerdem nach Nürnberger statistischem Bezirk (`Bezirk`).

## Veröffentlichung mit GitHub Pages

GitHub Pages ist auf den Branch `gh-pages` konfiguriert. Der Branch enthält nur das generierte `public/`-Artefakt; die Quell- und Snapshot-Dateien bleiben auf `main`.

Öffentliche URL: <https://nuernberg-maps-review-removals.patwoz.dev/>

Lokale Vorschau des Veröffentlichungs-Artefakts:

```bash
make site
python3 -m http.server --directory public 8080
```

Optionale Plausible-Analytics werden nur eingebunden, wenn die Umgebungsvariable gesetzt ist:

```bash
DASHBOARD_ANALYTICS_SRC="https://a.patwoz.dev/js/script.js" \
DASHBOARD_ANALYTICS_DOMAIN="nuernberg-maps-review-removals.patwoz.dev" \
make site
```

Veröffentlichen:

```bash
make deploy-pages
```

Im GitHub-Repository muss dafür **Settings → Pages → Source: Deploy from a branch**, Branch `gh-pages`, Ordner `/` aktiv sein.

## GitHub Actions

Der Workflow `.github/workflows/refresh-and-deploy.yml` baut und veröffentlicht GitHub Pages bei jedem Push auf `main` neu.

Ein Daten-Refresh läuft bewusst nur manuell über **Actions → Refresh data and deploy site → Run workflow** mit aktivierter Option `refresh_data`. Standardmäßig wird dann der vorhandene Discovery-Snapshot komplett neu gescrapt:

```bash
--scrape-only --rescrape-all --save-every 25 --delay-min 4000 --delay-max 9000 --headless=true
```

Falls Google ein CAPTCHA oder eine eingeschränkte Ansicht ausliefert, kann der Action-Lauf fehlschlagen oder unvollständige Daten liefern; dann lokal mit sichtbarem Browser neu laufen lassen. Zuvor erkannte Löschbanner werden bei automatischen Re-Scrapes standardmäßig nicht entfernt; dafür ist nach manueller Prüfung `--allow-banner-clears` nötig.

## Tests / Checks

```bash
make test
make check
# oder direkt:
go test ./...
go run ./cmd/validate
```

## Was die Diagramme zeigen

1. **Höchste Lösch-Quote**  
   `removed_midpoint / (visible_reviews + removed_midpoint)`

2. **Schlechtestes „echtes“ Rating**  
   Annahme: Jede entfernte Bewertung war eine 1-Stern-Bewertung.

3. **Beste Orte ohne Löschbanner**  
   Ohne sichtbaren Diffamierungs-Löschbanner, sortiert nach Rating und danach Rezensionszahl.

4. **Verteilung der Lösch-Stufen**  
   Zählt Orte nach Googles sichtbaren Löschbereichen.

## Nürnberger statistische Bezirke

Einträge mit Koordinaten werden über die offizielle Bezirksatlas-Geometrie von `online-service2.nuernberg.de/geoinf/ia_bezirksatlas/` den Nürnberger statistischen Bezirken zugeordnet. Die Geometrie liegt in `internal/mapsreview/data/nuernberg_statistische_bezirke.json`.

Punkte in nicht bewohnten Lücken dieser Quelle werden nur dann dem nächstgelegenen statistischen Bezirk zugeordnet, wenn die Zeile eine Nürnberger Postleitzahl hat. Nicht-Nürnberger Postleitzahlen bleiben ohne Bezirkszuordnung.

## Standardmäßig enthaltene Nürnberger PLZ

`90402, 90403, 90408, 90409, 90411, 90419, 90425, 90427, 90429, 90431, 90439, 90441, 90443, 90449, 90451, 90453, 90455, 90459, 90461, 90469, 90471, 90473, 90475, 90478, 90480, 90482, 90489, 90491`

## Hinweise zur Vollständigkeit

Die Google-Maps-Suche ist kein vollständiger Datenbankexport. Für bessere Abdeckung mehrere Suchbegriffe pro PLZ verwenden und Ergebnisse deduplizieren. Die Standard-Suchbegriffe sind:

`restaurant, café, imbiss, pizzeria, bäckerei, döner, burger, sushi, schnitzel, frühstück, brunch`

Für einen strengeren „nur Restaurants“-Datensatz nur `--queries restaurant` verwenden und `output/places.csv` anschließend manuell filtern.

## Lizenz

MIT, siehe [`LICENSE`](LICENSE).
