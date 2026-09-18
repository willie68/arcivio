# Arcivio

**Work in progress.** Arcivio is an early-stage project. The public API, data formats and UI will change. Do not use it in production yet.

**WIP – kein Produktionseinsatz.** Arcivio ist ein SOHO-Dokumentenmanagementsystem in aktiver Entwicklung. Schnittstellen, Speicherformate und Oberfläche sind noch nicht stabil. Es gibt **keine Gewähr** für Vollständigkeit, Datensicherheit oder revisionssichere Archivierung.

## Ziel

Kleines, selbsttragendes DMS für Einzelbüros und kleine Teams:

- Ablage beliebiger Dateitypen
- frei definierbare Dokumenttypen (Rechnung, Vertrag, Lohnabrechnung, …)
- integriertes Append-Only-Archiv mit selbsttragenden Dokumenten
- Volltext- und spätere KI-Suche
- eigene Authentifizierung inkl. internem IdP, später SSO (Microsoft / Apple)
- eine Binary: Go-Server und eingebettetes PrimeVue-Frontend
- eingebettetes SQLite, keine externe Datenbank

OCR, Volltext-Extraktion und Embeddings kommen über **externe** Dienste, nicht ins Kernprodukt.

Der technische Entwurf steht in [PLAN.md](PLAN.md). Das ist ein Architekturplan, keine fertige Spezifikation und **keine Rechtsberatung**. GoBD-Tauglichkeit ist ein Designziel, keine Zertifizierung.

## Aktueller Stand (Phase 1)

Vorhanden:

- Clean-/Hexagonal-Gerüst (Go 1.26, Chi, samber/do)
- Health (`/livez`, `/readyz`), optional Metrics, Swagger unter `/swagger/`
- eingebettetes SQLite (`storage.type: sqlite`)
- Platzhalter-SPA (Vue 3 + PrimeVue) unter `/`

Noch nicht vorhanden (geplant): Dokumentablage, Archiv-Volumes, Auditlog, Suche, Auth, Blob-Anzeige, optionale At-Rest-Verschlüsselung.

## Mitmachen

Forks und Weiterentwicklung sind ausdrücklich erwünscht. Issues und Pull Requests gerne auf [github.com/willie68/arcivio](https://github.com/willie68/arcivio).

Änderungen an Forks liegen bei den jeweiligen Autorinnen und Autoren. Der ursprüngliche Autor übernimmt dafür keine Verantwortung.

## Lokal starten

Voraussetzungen: Go 1.26+, für das Frontend-Rebuild Node.js (optional, die SPA liegt bereits in `pkg/web/client`).

```text
go test ./...
go run ./cmd/service -c ./configs/service_local.yaml
```

Mit der lokalen Beispielconfig:

- HTTP (Health/Metrics): [http://127.0.0.1:9480/livez](http://127.0.0.1:9480/livez)
- HTTPS (API/UI): [https://127.0.0.1:9443/](https://127.0.0.1:9443/) — Testzertifikat, Browser-Warnung ist erwartet

Konfiguration über `-c` oder Standarddatei unter dem Benutzer-Configverzeichnis. Platzhalter `${name}` kommen aus der Umgebung. Laufzeitdaten liegen unter `data/` (nicht im Git).

Frontend neu bauen:

```text
cd web
npm install
npm run build
```

## Lizenz und Haftung

[MIT License](LICENSE) — Copyright 2026 Wilfried Klaas.

Die Software wird **unverändert („as is“)** bereitgestellt, ohne Gewährleistung. Soweit gesetzlich zulässig, haftet der Autor nicht für Schäden aus Nutzung, Nichtnutzung oder Weiterentwicklung — auch nicht beim eigenen Einsatz. Zwingende gesetzliche Haftung bleibt unberührt.
