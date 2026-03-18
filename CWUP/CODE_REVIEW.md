# Code Review — ForgeCrowdBook v1.0

**Datum:** 2026-03-18
**Reviewer:** Claude (claude-sonnet-4-6)
**Stand:** alle Phasen abgeschlossen, vor erstem Release-Tag

---

## Methodik

Vollständige Analyse aller Go-Quelldateien (`main.go`, `internal/**`, `integration_test.go`), Templates (`templates/*.html`) und statischer Assets (`static/preview.js`). Befunde nach Schweregrad gruppiert.

---

## Befunde und Maßnahmen

### KRITISCH — vor Tag behoben

**CR-01: Open Redirect via Referer-Header** (`internal/handler/middleware.go`)
Der `/lang`-Endpoint verwendete den HTTP-Referer-Header ohne Validierung als Redirect-Ziel. Ein Angreifer konnte durch einen manipulierten Referer auf externe Seiten umleiten.
*Fix:* Nur den Pfadteil des Referrers verwenden (`url.Parse` → `RequestURI()`), Host wird verworfen.

**CR-02: UTF-8-Truncation in `excerpt()`** (`internal/handler/helpers.go`)
`clean[:max]` schnitt auf Byte-Ebene ab. Bei Multi-Byte-Zeichen (Umlaute, Unicode) entstand ungültiges UTF-8 in OG-Meta-Tags.
*Fix:* Truncation über `[]rune` auf Zeichen-Ebene.

---

### MITTEL — vor Tag behoben

**CR-03: Kein CSRF-Schutz** (alle POST-Endpoints)
Sämtliche zustandsverändernden Operationen (Logout, Sprachauswahl, Chapter-Erstellung, Admin-Aktionen) waren ohne CSRF-Schutz. SameSite=Lax allein ist kein vollständiger Schutz.
*Fix:* Neues Paket `internal/csrf` mit Double-Submit-Cookie-Pattern. Middleware global in `main.go` eingehängt. Alle HTML-Forms erhalten `<input type="hidden" name="_csrf">`, AJAX-Calls senden `X-CSRF-Token`-Header (gelesen aus `<meta name="csrf-token">`).

**CR-04: Session Secret ohne Mindestlängen-Validierung** (`internal/config/config.go`)
Schwache oder leere Secrets wurden still akzeptiert. HMAC-Signaturen auf Basis kurzer Secrets sind kompromittierbar.
*Fix:* `LoadConfig()` prüft `len(session_secret) >= 32`, gibt andernfalls Fehler zurück.

**CR-05: Lang-Cookie ohne MaxAge** (`internal/handler/middleware.go`)
Fehlender `MaxAge` machte den Sprachcookie zu einem Session-Cookie — Sprachpräferenz ging beim Browser-Schließen verloren.
*Fix:* `MaxAge: 365 * 24 * 60 * 60` (1 Jahr).

---

### NIEDRIG — vor Tag behoben

**CR-06: Fehlendes Logging in Template-Render-Fehlerpfaden** (`internal/handler/helpers.go`)
Fehler bei Template-Parse und -Ausführung wurden nicht geloggt, nur ein generisches HTTP 500 zurückgegeben. Erschwert Produktions-Debugging erheblich.
*Fix:* `log.Printf` in allen Fehlerpfaden von `renderPage()`.

**CR-07: JSON-Encode-Fehler ignoriert** (`internal/handler/api.go`)
`_ = json.NewEncoder(w).Encode(...)` im Like-Endpoint ignorierte Encode-Fehler stillschweigend.
*Fix:* Fehler wird geloggt.

---

## False Positives (nicht gefixt)

**FP-01: XSS in OG-Meta-Tags**
Go's `html/template` escaped Attributwerte context-sensitiv korrekt. Kein Handlungsbedarf.

**FP-02: User-Enumeration via Login-Timing**
Der Login-Endpoint zeigt identische Bestätigungsseite für bekannte und unbekannte E-Mail-Adressen (spec-konform). Timing-Unterschiede durch E-Mail-Versand sind bei Magic-Link-Flows inhärent und akzeptiert.

---

## Abgleich mit RISKS.md

| Risiko | Typ | Status |
|--------|-----|--------|
| R-01 SMTP/TLS-Konfiguration | Deployment | unverändert, keine Code-Maßnahme erforderlich |
| R-02 SQLite Concurrency | Architektur | bewusst akzeptiert für v1 |
| R-03 Mail-Delivery/Spam | Deployment | unverändert, Dokumentationsaufgabe |
| R-04 Markdown-Encoding | Code | durch `r.FormValue()` abgedeckt, TC-04 testet explizit |
| R-05 Session Secret Rotation | Ops | dokumentiert; jetzt zusätzlich code-seitig (CR-04) abgesichert |

---

## Ergebnis

Alle 7 Code-Findings behoben. Alle Tests grün. Projekt ist release-bereit.
