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

## Aktueller Stand (Phase 2 Auth)

Vorhanden:

- Clean-/Hexagonal-Gerüst (Go 1.26, Chi, samber/do)
- Health (`/livez`, `/readyz`), optional Metrics, Swagger unter `/swagger/`
- eingebettetes SQLite (`storage.type: sqlite`)
- interner OIDC-IdP unter `/auth` (Authorization Code + PKCE, Argon2id, RS256 JWT)
- SPA-Login (Vue 3 + PrimeVue) unter `/`; `GET /api/v1/me` mit Bearer-Token

Noch nicht vorhanden (geplant): Dokumentablage, Archiv-Volumes, Auditlog, Suche, RBAC-Durchsetzung, Blob-Anzeige, optionale At-Rest-Verschlüsselung, SSO (Entra/Apple).

## Mitmachen

Forks und Weiterentwicklung sind ausdrücklich erwünscht. Issues und Pull Requests gerne auf [github.com/willie68/arcivio](https://github.com/willie68/arcivio).

Änderungen an Forks liegen bei den jeweiligen Autorinnen und Autoren. Der ursprüngliche Autor übernimmt dafür keine Verantwortung.

## Lokal starten

Voraussetzungen: Go 1.26+, für das Frontend-Rebuild Node.js (optional, die SPA liegt bereits in `backend/pkg/web/client`).

```text
cd backend
go test ./...
go run ./cmd/service -c ./configs/service_local.yaml
```

Mit der lokalen Beispielconfig (`backend/configs/service_local.yaml`):

- HTTP (Health/Metrics): [http://127.0.0.1:9480/livez](http://127.0.0.1:9480/livez)
- HTTPS (API/UI): [https://127.0.0.1:9443/](https://127.0.0.1:9443/) — Testzertifikat, Browser-Warnung ist erwartet

Beim ersten Start (leere Nutzertabelle) legt der Dienst den User `admin` mit Passwort `admin` an. Die SPA verlangt danach einen Passwortwechsel (mindestens 8 Zeichen, ungleich dem alten). `/api/v1/*` ohne JWT antwortet mit 401. Bootstrap zurücksetzen: `backend/data/arcivio.db` löschen und den Dienst neu starten.

Konfiguration über `-c` oder Standarddatei unter dem Benutzer-Configverzeichnis. Platzhalter `${name}` kommen aus der Umgebung. Laufzeitdaten liegen unter `data/` (nicht im Git).

Frontend neu bauen (landet in `backend/pkg/web/client`):

```text
cd frontend
npm install
npm run build
```

Oder `scripts/build.cmd` (Frontend, Swagger, Binary).

Frontend-Dev (Vite, Port 5173) braucht denselben laufenden Go-Dienst; `/auth` und `/api` werden nach `https://127.0.0.1:9443` geproxyt:

```text
cd frontend
npm run dev
```

Dann [http://localhost:5173/](http://localhost:5173/).

## Lizenz und Haftung

[MIT License](LICENSE) — Copyright 2026 Wilfried Klaas.

Die Software wird **unverändert („as is“)** bereitgestellt, ohne Gewährleistung. Soweit gesetzlich zulässig, haftet der Autor nicht für Schäden aus Nutzung, Nichtnutzung oder Weiterentwicklung — auch nicht beim eigenen Einsatz. Zwingende gesetzliche Haftung bleibt unberührt.
