package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var legalPageFiles = []string{"methodik.html", "korrektur.html", "impressum.html", "datenschutz.html"}

func legalPagesEnabled(a args) bool {
	return strings.TrimSpace(a.LegalName) != "" || strings.TrimSpace(a.LegalEmail) != "" || len(a.LegalAddressLines) > 0 || strings.TrimSpace(a.LegalNote) != "" || strings.TrimSpace(a.LegalPostHandler) != ""
}

func validateLegalArgs(a args) error {
	if !legalPagesEnabled(a) {
		return nil
	}
	missing := []string{}
	if strings.TrimSpace(a.LegalName) == "" {
		missing = append(missing, "--legal-name")
	}
	if strings.TrimSpace(a.LegalEmail) == "" {
		missing = append(missing, "--legal-email")
	}
	if len(a.LegalAddressLines) == 0 {
		missing = append(missing, "--legal-address-line")
	}
	if len(missing) > 0 {
		return fmt.Errorf("legal pages require %s", strings.Join(missing, ", "))
	}
	return nil
}

func syncLegalPages(a args, outputDir string) error {
	if !legalPagesEnabled(a) {
		for _, name := range legalPageFiles {
			if err := os.Remove(filepath.Join(outputDir, name)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	}
	pages := map[string]string{
		"methodik.html":    methodikPage(a),
		"korrektur.html":   korrekturPage(a),
		"impressum.html":   impressumPage(a),
		"datenschutz.html": datenschutzPage(a),
	}
	for name, body := range pages {
		if err := os.WriteFile(filepath.Join(outputDir, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func legalNavHTML(a args) string {
	if !legalPagesEnabled(a) {
		return ""
	}
	return `<nav aria-label="Weiterführende Hinweise"><a href="methodik.html">Methodik</a><a href="korrektur.html">Korrektur melden</a><a href="impressum.html">Impressum</a><a href="datenschutz.html">Datenschutz</a></nav>`
}

func footerLegalLinksHTML(a args) string {
	if !legalPagesEnabled(a) {
		return ""
	}
	return `<div class="footer-privacy"><a href="methodik.html">Methodik</a> · <a href="korrektur.html">Korrektur melden</a> · <a href="impressum.html">Impressum</a> · <a href="datenschutz.html">Datenschutz</a></div>`
}

func mapConsentHTML(a args) string {
	link := ""
	if legalPagesEnabled(a) {
		link = ` <a href="datenschutz.html">Details in der Datenschutzerklärung</a>.`
	}
	return `<div class="map-consent"><div class="map-consent-inner"><p class="map-consent-text">Die Karte lädt Leaflet von unpkg (Cloudflare, USA – DPF-zertifiziert) und Karten-Kacheln von CARTO (USA/Spanien – ohne DPF-Zertifizierung). Dabei werden IP-Adresse und Browserinformationen an diese Anbieter übertragen. Für CARTO fehlen ein Angemessenheitsbeschluss und Standardvertragsklauseln; ein dem EU-Recht gleichwertiges Datenschutzniveau ist nicht garantiert.</p><button id="mapConsentLoad" type="button" class="map-consent-btn">Karte laden</button><p class="map-consent-link">Einwilligung gilt nur für diese Sitzung.` + link + `</p></div></div>`
}

func legalAddressHTML(a args) string {
	lines := []string{esc(a.LegalName)}
	for _, line := range a.LegalAddressLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, esc(line))
		}
	}
	return strings.Join(lines, "<br>")
}

func legalShell(title string, description string, body string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s · Nürnberg Google-Maps-Bewertungen</title>
  <meta name="description" content="%s">
  <style>body{margin:0;font-family:Arial,Helvetica,sans-serif;color:#333;background:#fff}main{width:min(920px,calc(100vw - 32px));margin:42px auto 70px;line-height:1.6}nav{display:flex;flex-wrap:wrap;gap:12px;margin-bottom:30px}a{color:#cf2a1b;font-weight:700;text-decoration:none}a:hover{text-decoration:underline}h1{font-size:clamp(30px,5vw,48px);line-height:1.1;margin:0 0 18px}h2{margin-top:30px;color:#222}.note,.card{padding:16px;border-left:5px solid #1f6f8b;background:#f6f6f6}@media(prefers-color-scheme:dark){body{color:#e7e2dc;background:#0e0c0b}h2{color:#fff5ec}.note,.card{background:#171412}}</style>
</head>
<body><main>
  <nav aria-label="Navigation"><a href="/">Dashboard</a><a href="methodik.html">Methodik</a><a href="korrektur.html">Korrektur melden</a><a href="impressum.html">Impressum</a><a href="datenschutz.html">Datenschutz</a></nav>
%s
</main></body>
</html>
`, esc(title), escAttr(description), body)
}

func methodikPage(a args) string {
	city := esc(a.City)
	body := fmt.Sprintf(`<h1>Methodik und Einordnung</h1>
  <p class="note">Diese Seite dokumentiert öffentlich sichtbare Google-Hinweise in Unternehmensprofilen zum Abrufzeitpunkt. Sie behauptet nicht, dass Betriebe rechtswidrig gehandelt, Bewertungen missbräuchlich entfernen ließen oder legitime Kritik unterdrückt wurde.</p>
  <h2>Was gemessen wurde</h2>
  <p>Google Maps zeigt auf betroffenen Unternehmensprofilen einen öffentlichen Hinweis, dass Bewertungen aufgrund von Beschwerden wegen Diffamierung entfernt wurden. Nach dem von Google Maps angezeigten Hinweis bezieht sich diese Spanne auf Bewertungen, die <strong>in den letzten 365 Tagen</strong> entfernt wurden. Google nennt dabei keine exakte Zahl, sondern einen Bereich, zum Beispiel <em>2–5</em>, <em>6–10</em>, <em>11–20</em>, <em>21–50</em>, <em>51–100</em>, <em>101–150</em>, <em>151–200</em>, <em>201–250</em> oder <em>über 250</em>.</p>
  <p>Das Dashboard erfasst diese öffentlich sichtbaren Hinweise für Google-Maps-Orte im Raum %s. Es sammelt, speichert und veröffentlicht keine Rezensionstexte, keine Namen von Bewertenden, keine Namen von Beschwerdeführenden und keine Angaben zu Kanzleien in Einzelfällen.</p>
  <h2>Erhebung</h2>
  <p>Die Orte werden über Google-Maps-Suchen mit konfigurierten Kategoriebegriffen und Postleitzahlen gefunden, dedupliziert und anschließend einzeln geprüft. Optional kann die offizielle Google Places API nur für die Ortssuche verwendet werden; die Hinweis-Erkennung selbst erfolgt im Browser auf öffentlich sichtbaren Google-Maps-Seiten.</p>
  <p>Pro Ort werden insbesondere Name, Google-Maps-URL beziehungsweise Place-ID, aktuelle sichtbare Sternebewertung, sichtbare Rezensionszahl, Kategorie, Postleitzahl, Bezirk soweit zuordenbar, sichtbare Google-Hinweisspanne und Abrufzeitpunkt gespeichert.</p>
  <p>Google Maps wurde deutschsprachig ausgewertet. Die Erkennung der Hinweise, Tab-Beschriftungen und Rezensionszahlen ist auf deutsche Google-Maps-Ausgaben ausgelegt.</p>
  <h2>Berechnungen</h2>
  <p>Die sichtbare Rezensionszahl ist die aktuell von Google angezeigte Zahl sichtbarer Rezensionen. Entfernte Rezensionen sind darin nach dieser Modellierung nicht enthalten. Für den Hinweis-Anteil nutzt das Dashboard daher: <code>Näherungswert / (sichtbare Rezensionen + Näherungswert)</code>.</p>
  <p>Für geschlossene Bereiche nutzt das Dashboard den Mittelpunkt der angezeigten Spanne als Näherungswert. Beispiel: Aus <em>21–50</em> wird 35,5. Für <em>über 250</em> ist kein oberes Ende bekannt; das Dashboard verwendet für Sortierung und Näherungswert derzeit 300. Der tatsächliche Wert kann höher liegen.</p>
  <p>Alle aus Spannen abgeleiteten Kennzahlen sind Näherungen. Sie sollen Größenordnungen vergleichbar machen, sind aber keine exakten Zählwerte.</p>
  <h2>Hypothetisches Worst-Case-Rating</h2>
  <p>Das hypothetische Worst-Case-Rating ist eine mathematische Untergrenze, kein realer Wert. Es nimmt an, dass alle in der sichtbaren Google-Spanne enthaltenen entfernten Bewertungen 1-Stern-Bewertungen gewesen wären, und rechnet diese hypothetisch in den Durchschnitt ein.</p>
  <p>Das Worst-Case-Rating ist <strong>keine Tatsachenbehauptung</strong> über die entfernten Bewertungen, <strong>keine Aussage über die Qualität des Betriebs</strong> und keine Bewertung, ob einzelne Beschwerden berechtigt waren. Eine Sortierung nach diesem Wert ist kein Ranking nach Betriebsqualität, sondern nach der rechnerischen Auswirkung einer worst-case-Annahme.</p>
  <h2>Grenzen der Aussage</h2>
  <p>Die Stichprobe entsteht aus Google-Maps-Suchergebnissen. Diese Ergebnisse sind nach Googles eigener Sichtbarkeit, Relevanz und Prominenz sortiert und bilden nicht notwendigerweise alle Betriebe im Raum %s ab. Sichtbare Orte haben tendenziell mehr Online-Präsenz und mehr Rezensionen.</p>
  <p>Ein fehlender Hinweis bedeutet nicht sicher, dass nie Bewertungen entfernt wurden. Ein sichtbarer Hinweis bedeutet nicht, dass eine Beschwerde unberechtigt oder missbräuchlich war. Der Hinweis dokumentiert nur, dass Google zum Abrufzeitpunkt einen entsprechenden öffentlichen Hinweis angezeigt hat.</p>
  <p>Die Daten sind eine Momentaufnahme. Hinweise, Rezensionen und Bewertungen können sich ändern; auch Googles Oberfläche, Spracheinstellungen oder Sichtbarkeitsregeln können die Erkennung beeinflussen.</p>
  <h2>Was die Daten zeigen — und was nicht</h2>
  <p>Die Daten zeigen, bei welchen erfassten Google-Maps-Orten öffentlich sichtbare Hinweise auf entfernte Bewertungen wegen Diffamierungsbeschwerden angezeigt wurden und in welcher Spanne Google diese Hinweise ausweist.</p>
  <p>Die Daten zeigen nicht, ob entfernte Bewertungen legitim waren, ob Beschwerden berechtigt waren oder ob ein Betrieb rechtswidrig gehandelt hat. Ein hoher Hinweiswert ist keine Feststellung von Fehlverhalten. Umgekehrt kann ein Betrieb auch legitime Gründe gehabt haben, gegen falsche oder koordinierte negative Bewertungen vorzugehen.</p>
  <h2>Heilberufe und personenbezogene Einzelpraxen</h2>
  <p>Profile von Ärztinnen und Ärzten, Therapeutinnen und Therapeuten, Anwältinnen und Anwälten und vergleichbaren personenbezogenen Einzelpraxen sind nach der Rechtsprechung des Bundesgerichtshofs (BGH-Jameda-Entscheidungen) besonders schutzwürdig. Sichtbare Google-Hinweise auf solchen Profilen sind Tatsachen aus Google Maps, aber ihre Aufnahme in ein Dashboard berührt die berufliche Reputation der namentlich erfassten Person stärker als bei einem reinen Geschäftsprofil.</p>
  <p>Für diese Profile gilt deshalb besonders: Es wird keine Aussage über fachliche Qualität, Behandlungsstandards oder Berufsausübung getroffen. Betroffene Heilberufler und Einzelpraxen können Einträge bevorzugt über die Korrekturseite prüfen und auf plausible Beanstandung hin vorläufig ausblenden lassen.</p>
  <h2>Quelle und Reproduzierbarkeit</h2>
  <p>Alle Dashboard-Zahlen stammen aus öffentlich sichtbaren Google-Maps-Angaben zum Abrufzeitpunkt und aus daraus abgeleiteten Berechnungen. Quelle für die 365-Tage-Einordnung ist der öffentlich sichtbare Google-Maps-Hinweis im jeweiligen Unternehmensprofil selbst; die betroffenen Google-Maps-Profile sind im Dashboard pro Eintrag verlinkt. Der Hinweis kann sich ändern oder verschwinden, wenn Google die Oberfläche oder den Profilstatus ändert. Statistische Bezirke werden nur zur räumlichen Gruppierung verwendet. Die Veröffentlichung enthält keine Rezensionstexte und keine Rohdaten-CSV.</p>
  <p>Der konkrete Datenstand steht im Dashboard-Footer. Korrekturhinweise können über die Korrekturseite gemeldet werden.</p>
  <h2>Journalistische Sorgfalt (§ 19 MStV)</h2>
  <p>Das Dashboard wird als journalistisch-redaktionelles Telemedienangebot betrieben (siehe Impressum nach § 18 Abs. 2 MStV). Nachrichten werden vor ihrer Veröffentlichung mit der nach den Umständen gebotenen Sorgfalt auf Inhalt, Herkunft und Wahrheit überprüft (§ 19 Abs. 1 Satz 1 MStV). Erkennbare Unrichtigkeiten werden unverzüglich richtiggestellt (§ 19 Abs. 2 MStV); hierzu dient insbesondere die Korrekturseite. Meinungen werden als solche kenntlich gemacht; Tatsachen und Modellrechnungen — namentlich das hypothetische Worst-Case-Rating — sind in dieser Methodik-Seite gesondert ausgewiesen.</p>
  <p><a href="/">Zurück zum Dashboard</a></p>`, city, city)
	return legalShell("Methodik und Einordnung", "Methodik und Grenzen des Dashboards zu öffentlich sichtbaren Google-Maps-Hinweisen.", body)
}

func korrekturPage(a args) string {
	return legalShell("Korrektur melden", "Korrekturhinweise zum Dashboard melden.", fmt.Sprintf(`<h1>Korrektur melden</h1>
  <p class="note">Hinweis fehlerhaft, veraltet oder missverständlich? Bitte sende den Link zum Google-Profil und eine kurze Begründung per E-Mail an <a href="mailto:%s">%s</a>.</p>
  <h2>Prüfung von Hinweisen</h2>
  <p>Einträge werden nicht allein wegen Reputationsinteresse entfernt. Geprüft werden konkrete Fehler, Verwechslungen, veraltete Daten oder nachvollziehbare rechtliche Beanstandungen.</p>
  <h2>Vorläufige Ausblendung</h2>
  <p>Bei plausiblen Beanstandungen kann ein Eintrag vorläufig ausgeblendet werden. Eine vorläufige Ausblendung bedeutet kein Anerkenntnis einer Rechtsverletzung. Nach Prüfung wird korrigiert, entfernt oder wieder angezeigt.</p>
  <h2>Gegendarstellungsanspruch (§ 9 MStV)</h2>
  <p>Soweit das Dashboard als journalistisch-redaktionelles Telemedienangebot Tatsachenbehauptungen über identifizierbare Personen oder Einrichtungen enthält, besteht ein Gegendarstellungsanspruch nach § 9 MStV in Verbindung mit dem jeweiligen Landespressegesetz (für den Verantwortlichen: Art. 10 BayPrG). Voraussetzungen:</p>
  <ul>
    <li>Tatsachenbehauptung — keine Werturteile, keine Meinungsäußerungen</li>
    <li>Berechtigtes Interesse des Betroffenen</li>
    <li>Unverzüglich nach Kenntnis, spätestens innerhalb von drei Monaten ab Veröffentlichung</li>
    <li>Schriftlich, mit eigenhändiger Unterschrift des Betroffenen oder seines gesetzlichen Vertreters</li>
    <li>Angemessener Umfang, kein strafbarer Inhalt</li>
  </ul>
  <p>Eine fristgerechte Gegendarstellung wird in vergleichbarer Aufmachung an entsprechender Stelle des Angebots veröffentlicht (§ 9 Abs. 1 Satz 4 MStV). Das Verlangen ist an die im Impressum genannte Anschrift zu richten.</p>
  <p><a href="/">Zurück zum Dashboard</a></p>`, escAttr(a.LegalEmail), esc(a.LegalEmail)))
}

func impressumPage(a args) string {
	note := ""
	if strings.TrimSpace(a.LegalNote) != "" {
		note = fmt.Sprintf("\n    <p>%s</p>", esc(a.LegalNote))
	}
	body := fmt.Sprintf(`<h1>Impressum</h1>
  <section class="card" aria-label="Verantwortlich">
    <h2>Angaben gemäß § 5 DDG</h2>
    <p><strong>%s</strong><br>%s</p>
    <p>E-Mail: <a href="mailto:%s">%s</a></p>%s
    <h2>Verantwortlich für journalistisch-redaktionelle Inhalte</h2>
    <p>Verantwortlich im Sinne von § 18 Abs. 2 MStV:<br>%s</p>
  </section>
  <h2>Haftung für Inhalte</h2>
  <p>Die Inhalte dieser Website wurden mit Sorgfalt erstellt. Für Richtigkeit, Vollständigkeit und Aktualität der Inhalte kann keine Gewähr übernommen werden. Hinweise auf Fehler können über die Korrekturseite gemeldet werden.</p>
  <p><a href="/">Zurück zum Dashboard</a></p>`, esc(a.LegalName), strings.Join(legalAddressLinesHTML(a), "<br>"), escAttr(a.LegalEmail), esc(a.LegalEmail), note, legalAddressHTML(a))
	return legalShell("Impressum", "Impressum des Dashboards zu Google-Maps-Bewertungen.", body)
}

func legalAddressLinesHTML(a args) []string {
	out := []string{}
	for _, line := range a.LegalAddressLines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, esc(line))
		}
	}
	return out
}

func datenschutzPage(a args) string {
	postHandler := ""
	if strings.TrimSpace(a.LegalPostHandler) != "" {
		postHandler = fmt.Sprintf(`<h2>Zustellanschrift</h2>
  <p>Für die Zustellanschrift kann %s eingehende Post für %s entgegennehmen und weitergeben. Dabei können Absenderdaten, Anschreiben, Zustelldaten und Inhalte der Sendung verarbeitet werden, soweit dies zur Entgegennahme und Weiterleitung erforderlich ist.</p>`, esc(a.LegalPostHandler), esc(a.LegalName))
	}
	note := ""
	if strings.TrimSpace(a.LegalNote) != "" {
		note = "\n  <p>" + esc(a.LegalNote) + "</p>"
	}
	body := fmt.Sprintf(`<h1>Datenschutzerklärung</h1>
  <p class="note">Diese Erklärung beschreibt die Datenverarbeitung auf diesem Dashboard-Subdomain-Projekt.</p>
  <h2>Verantwortlicher</h2>
  <p>%s<br>E-Mail: <a href="mailto:%s">%s</a></p>%s
  <h2>Bereitstellung der Website (Hosting)</h2>
  <p>Die Website wird als statische Website über GitHub Pages veröffentlicht. Anbieter ist die GitHub Inc., 88 Colin P Kelly Jr Street, San Francisco, CA 94107, USA. Beim Aufruf können technisch notwendige Zugriffsdaten verarbeitet werden, insbesondere IP-Adresse, Datum und Uhrzeit, angeforderte Datei, Referrer, Browser-/Geräteinformationen und technische Statusinformationen. Diese Verarbeitung ist notwendig, um die Website auszuliefern, Sicherheit und Stabilität zu gewährleisten und Missbrauch abzuwehren. Rechtsgrundlage ist Art. 6 Abs. 1 lit. f DSGVO; berechtigtes Interesse ist die zuverlässige Auslieferung des Angebots. Die Server-Logs werden gemäß den Voreinstellungen von GitHub Pages aufbewahrt; der Verantwortliche hat darauf keinen direkten Zugriff.</p>
  <p>Da GitHub Inc. ihren Sitz in den USA hat, kann es zu einer Übermittlung personenbezogener Daten in ein Drittland kommen. Die Übermittlung erfolgt auf Grundlage des EU-US Data Privacy Framework (Angemessenheitsbeschluss der EU-Kommission vom 10.07.2023); GitHub Inc. ist nach dem DPF zertifiziert. Der Zertifizierungsstatus ist öffentlich abrufbar unter <a href="https://www.dataprivacyframework.gov/" target="_blank" rel="noopener noreferrer">www.dataprivacyframework.gov</a>.</p>
  <h2>Plausible Analytics</h2>
  <p>Diese Website nutzt Plausible Analytics für datenschutzfreundliche, aggregierte Reichweitenmessung. Anbieter ist die Plausible Insights OÜ, Västriku 2, 50403 Tartu, Estland. Plausible arbeitet ohne Werbe-Cookies und erstellt keine personenbezogenen Nutzerprofile. Verarbeitet werden technische Nutzungsdaten wie Seitenaufrufe, Referrer, Browser, Betriebssystem, Gerätetyp, Land/Region und Zeitpunkt des Besuchs in aggregierter Form. Die Anbieter-Domain für das Skript ist <code>a.patwoz.dev</code>. Die Daten werden innerhalb der EU verarbeitet.</p>
  <p>Rechtsgrundlage ist Art. 6 Abs. 1 lit. f DSGVO. Das berechtigte Interesse liegt in der statistischen Auswertung der Nutzung, der Fehleranalyse und der Verbesserung des Angebots. Aggregierte Statistiken werden so lange aufbewahrt, wie es für die Auswertung erforderlich ist; einzelne Roh-Datenpunkte werden bei Plausible nach kurzer Zeit verworfen und sind ohne Personenbezug.</p>
  <h2>Lokale Speicherung im Browser</h2>
  <p>Für den Hell-/Dunkel-Modus speichert die Website die gewählte Darstellung lokal im Browser mit dem Schlüssel <code>dashboardTheme</code>. Dabei wird kein Cookie gesetzt und diese Information wird nicht an den Betreiber übermittelt. Die Speicherung ist nach § 25 Abs. 2 Nr. 2 TDDDG für die vom Nutzer gewünschte Darstellungsfunktion unbedingt erforderlich. Sie kann über die Browser-Einstellungen gelöscht werden.</p>
  <h2>Karte – Leaflet, unpkg und CARTO</h2>
  <p>Die interaktive Karte wird erst nach einem aktiven Klick auf „Karte laden“ aktiviert. Vorher findet kein Aufruf der Karten-Anbieter statt. Erst nach diesem Klick werden das Open-Source-Skript Leaflet von <code>unpkg.com</code> (betrieben von Cloudflare, Inc., 101 Townsend Street, San Francisco, CA 94107, USA) und die Karten-Kacheln von der CARTO-Basemap-CDN (CARTO DB Inc., 201 Avenue of the Americas, New York, NY 10013, USA, sowie CartoDB Spain SL) geladen. Dabei werden technisch notwendige Zugriffsdaten, insbesondere IP-Adresse und Browserinformationen, an diese Anbieter übertragen. Die Karten-Daten basieren auf OpenStreetMap; OpenStreetMap-Mitwirkende sind im Attributionshinweis der Karte genannt.</p>
  <p>Rechtsgrundlage für das Laden der Ressourcen ist Art. 6 Abs. 1 lit. a DSGVO (Einwilligung durch aktives Klicken auf „Karte laden“). Die Einwilligung gilt nur für die laufende Sitzung und kann durch Neuladen der Seite zurückgenommen werden. Da Cloudflare, Inc. und CARTO DB Inc. ihren Sitz in den USA haben, ist mit dem Laden eine Übermittlung in ein Drittland verbunden.</p>
  <p>Cloudflare, Inc. ist nach dem EU-US Data Privacy Framework zertifiziert (Status verifizierbar unter <a href="https://www.dataprivacyframework.gov/" target="_blank" rel="noopener noreferrer">www.dataprivacyframework.gov</a>); die Übermittlung an Cloudflare ist dadurch auf Grundlage des Angemessenheitsbeschlusses vom 10.07.2023 zulässig.</p>
  <p>Für CARTO DB Inc. ist nach derzeitigem Kenntnisstand keine DPF-Zertifizierung verzeichnet; ein Auftragsverarbeitungsvertrag mit Standardvertragsklauseln nach Art. 46 DSGVO besteht zwischen dem Verantwortlichen und CARTO nicht. Die Übermittlung erfolgt deshalb auf Grundlage von Art. 49 Abs. 1 lit. a DSGVO: ausdrückliche Einwilligung in einen Drittlandtransfer, nachdem über das mit dem Fehlen eines Angemessenheitsbeschlusses und geeigneter Garantien verbundene Risiko informiert wurde (möglicher Behördenzugriff im Empfängerland, kein dem Unionsrecht gleichwertiges Datenschutzniveau). Die Einwilligung wird ausschließlich für das laufende Browsersession-Laden der Karte erteilt und ist nicht für systematische oder wiederholte Transfers vorgesehen.</p>
  <h2>Standortfunktion „In meiner Nähe“</h2>
  <p>Die Standortfunktion wird nur nach aktiver Auswahl von „In meiner Nähe“ verwendet. Der Browser fragt dann nach einer Standortfreigabe. Der Standort wird ausschließlich lokal im Browser genutzt, um Entfernungen zu berechnen und die Tabelle/Karte zu sortieren. Der Standort wird vom Dashboard nicht an den Betreiber gesendet und nicht dauerhaft gespeichert.</p>
  <h2>Google-Maps-Links</h2>
  <p>Einträge im Dashboard verlinken auf Google Maps. Erst beim Anklicken eines solchen Links wird Google Maps in einem neuen Tab geöffnet. Dann gelten die Datenschutzbedingungen von Google für den dortigen Seitenaufruf.</p>
  <h2>Öffentliche Unternehmensprofildaten</h2>
  <p>Das Dashboard enthält Daten aus öffentlich sichtbaren Google-Unternehmensprofilen, zum Beispiel Name des Profils, Kategorie, Ort, Postleitzahl, Bewertungszahl, Bewertung, öffentlich sichtbare Google-Hinweisspanne und Abrufzeitpunkt. Einzelne Rezensionstexte, Nutzerprofile oder entfernte Bewertungen werden nicht veröffentlicht.</p>
  <p>Bei Einzelunternehmen, freiberuflichen Praxen oder personenbezogenen Profilnamen können diese Angaben personenbezogene Daten sein. Die Verarbeitung dient der Dokumentation und Einordnung öffentlich sichtbarer Google-Hinweise als regionales Transparenz- und Informationsangebot. Rechtsgrundlage ist Art. 6 Abs. 1 lit. f DSGVO. Betroffene können über die Korrekturseite eine Prüfung verlangen.</p>
  %s
  <h2>Kontakt und Korrekturanfragen</h2>
  <p>Wenn du Kontakt aufnimmst oder eine Korrektur meldest, werden die übermittelten Angaben zur Bearbeitung der Anfrage verarbeitet, insbesondere Kontaktdaten, Inhalt der Nachricht, Zeitpunkt und betroffene Google-Profil-Links. Die Kontaktaufnahme erfolgt per E-Mail an <a href="mailto:%s">%s</a>. Korrespondenz wird so lange aufbewahrt, wie es zur Bearbeitung und Dokumentation der Anfrage erforderlich ist, und danach gelöscht, sofern keine gesetzlichen Aufbewahrungspflichten entgegenstehen.</p>
  <h2>Betroffenenrechte und Aufsichtsbehörde</h2>
  <p>Betroffene Personen haben nach Maßgabe der DSGVO insbesondere Rechte auf Auskunft (Art. 15), Berichtigung (Art. 16), Löschung (Art. 17), Einschränkung der Verarbeitung (Art. 18), Datenübertragbarkeit (Art. 20) und Widerspruch (Art. 21). Für Verarbeitungen auf Basis von Einwilligung kann diese jederzeit mit Wirkung für die Zukunft widerrufen werden (Art. 7 Abs. 3 DSGVO).</p>
  <p>Es besteht ein Beschwerderecht bei einer Datenschutzaufsichtsbehörde (Art. 77 DSGVO). Zuständige Aufsichtsbehörde für den Verantwortlichen: Bayerisches Landesamt für Datenschutzaufsicht (BayLDA), Promenade 18, 91522 Ansbach, <a href="https://www.lda.bayern.de" target="_blank" rel="noopener noreferrer">www.lda.bayern.de</a>.</p>
  <h2>Medienprivileg (§ 23 MStV)</h2>
  <p>Das Dashboard versteht sich als journalistisch-redaktionelles Telemedienangebot im Sinne des Medienstaatsvertrags (siehe Impressum nach § 18 Abs. 2 MStV). Soweit personenbezogene Daten zu journalistisch-redaktionellen Zwecken verarbeitet werden, gilt für diese Verarbeitung das Medienprivileg nach § 23 MStV. Davon unberührt bleiben die Pflichten zur Datensicherheit sowie die Verantwortlichkeit nach §§ 5, 7 und 24 BDSG. Korrektur- und Gegendarstellungsanfragen können über die Korrekturseite gestellt werden.</p>
  <h2>Änderungen</h2>
  <p>Diese Datenschutzerklärung kann angepasst werden, wenn sich Technik, Dienste oder rechtliche Anforderungen ändern.</p>
  <p><a href="/">Zurück zum Dashboard</a></p>`, legalAddressHTML(a), escAttr(a.LegalEmail), esc(a.LegalEmail), note, postHandler, escAttr(a.LegalEmail), esc(a.LegalEmail))
	return legalShell("Datenschutzerklärung", "Datenschutzerklärung des Dashboards zu Google-Maps-Bewertungen.", body)
}
