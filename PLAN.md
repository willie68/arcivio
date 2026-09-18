# Arcivio – SOHO Dokumentenmanagement

Architektur- und Umsetzungsplan: selbsttragendes SOHO-DMS mit Go-Backend, PrimeVue-Frontend, eingebettetem Speicher, eigener AuthN/AuthZ inkl. SSO sowie integriertem, signierbarem Append-Only-Archiv und Bleve (Volltext + Vektoren).

## Offene Arbeitspakete

- [ ] go-micro (Clean/Hexagonal) nach Arcivio kopieren, Address-Demo entfernen, PrimeVue in pkg/web/client
- [ ] Auth auf Template-JWT aufsetzen: lokale Nutzer (Argon2id), Claims/RBAC, Login stellt JWT aus; später OIDC Entra + Apple
- [ ] Dokumenttypen als Schema mit archival vs. fluid Attributen, Blob-Store, Versionen nur bei archival Änderungen, fluider Sidecar-Speicher
- [ ] Selbsttragende DOC-Envelopes, Löschen per DEL-Record im aktuellen Volume (alte Container unverändert), Restore, Siegel/Signatur
- [ ] Auditlog: bereichsweise YAML-Schalter, AUDIT-Envelopes im Auto-Store `_audit`, Admin-Einsicht, Hash-Kette, keine Rekursion
- [ ] Optionale PQC-At-Rest-Verschlüsselung: ML-KEM-Hybrid wrappt AES-256-GCM-DEK für Archiv, Fluid, Working, SQLite (und Bleve); Default aus
- [ ] Extractor-Adapter, Bleve FTS mit KV-Attributen, Query-Parser (Phrase, Wildcard, AND/OR, Range, Klammern), Vector/Hybrid
- [ ] Blobview: native Image/PDF; Outbound-Render wandelt EML/MSG/Office nach PDF als fluides File-Attribut; Frontend PDF-Anzeige
- [ ] PrimeVue: Ablage, Typen, Suche (ein Suchfeld), Archiv-Status, Audit-Store (Admin), Benutzerverwaltung, Blobview

## Annahmen (nach abgebrochenem Fragebogen)

- **Projektname:** Arcivio (Go-Modul `arcivio`)
- **Pfad:** `H:\privat\git-sourcen\Arcivio`
- **Archiv:** pragmatisch **GoBD-tauglich** (Unveränderbarkeit softwareseitig, Hash, Signatur, Audit, Aufbewahrung) – **kein** BSI TR-ESOR
- **SSO:** lokale Konten plus **OIDC** für **Microsoft Entra ID** und **Sign in with Apple**; macOS-Kerberos/Open Directory später optional
- **Go-Gerüst:** Kopie der **aktuellen** `H:\privat\git-sourcen\go-micro`-Clean/Hexagonal-Struktur
- Setup-Defaults: Git, Vue 3 + PrimeVue (Vite), SPA in `pkg/web/client` wie im Template eingebettet

Keine Zertifizierung und keine Rechtsberatung: das System liefert technische Nachvollziehbarkeit; Verfahrensdokumentation bleibt organisatorisch.

## Zielbild

Ein-Binary-DMS für kleine Büros: beliebige Dateien ablegen, frei definierbare Dokumenttypen, Volltext- und KI-Suche, integriertes Archiv in wachsenden Containerdateien, Signatur mit internem oder externem Zertifikat. Extraktion/OCR und Embeddings kommen über Adapter von **externen** Services.

```mermaid
flowchart LR
  subgraph ui [Frontend]
    PrimeVue[PrimeVue SPA]
  end
  subgraph server [Go Binary]
    API[HTTP API]
    Auth[AuthN AuthZ]
    Docs[Dokumentkern]
    Fluid[Fluid Store OCR PreviewPDF]
    Render[Renderengine nach PDF]
    Arch[Archiv Engine]
    Search[Bleve FTS plus Vector]
    Meta[SQLite embedded]
    Audit[Audit Archiv-Store]
  end
  PrimeVue --> API
  API --> Auth
  API --> Docs
  API --> Arch
  API --> Search
  API --> Audit
  Docs --> Meta
  Auth --> Meta
  Docs --> Render
  Render --> Fluid
  Docs --> Extractor[Externe Extraktion]
  Extractor --> Fluid
  Fluid --> Search
  Docs --> Arch
  Auth --> Audit
  Docs --> Audit
  Search --> Audit
  Arch --> Audit
```

## Tech-Stack

**Go-Teil = aktuelle Vorlage `H:\privat\git-sourcen\go-micro`** – **Clean Architecture** mit hexagonalen Inbound-/Outbound-Adaptern (nicht das alte `internal/services`-Layout). Arcivio ist eine Kopie dieser Struktur.

Schichten (Abhängigkeiten nur nach innen; Domain kennt keine Adapter):

```mermaid
flowchart TB
  subgraph inbound [Inbound Adapter]
    HTTP[adapter inbound http apiv1 JWT]
  end
  subgraph app [Domain Use Cases]
    Doc[domain document]
    ArchD[domain archive]
    SearchD[domain search]
    AuthD[domain identity]
    AuditD[domain audit]
  end
  subgraph outbound [Outbound Adapter]
    SQL[sqlite]
    Vol[archive volumes]
    Bleve[bleve]
    Ext[extract HTTP]
    Rend[render PDF]
  end
  subgraph infra [Infrastructure]
    SHttp[shttp health logging]
    Crypto[cryptofs]
    Boot[bootstrap samber do]
  end
  HTTP --> Doc
  HTTP --> ArchD
  HTTP --> SearchD
  HTTP --> AuthD
  HTTP --> AuditD
  Doc --> SQL
  Doc --> Vol
  SearchD --> Bleve
  AuthD --> SQL
  AuditD --> Vol
  SQL --> Crypto
  Vol --> Crypto
  Boot --> HTTP
  Boot --> SHttp
```

Übernommen:

- Go 1.26, Chi, **samber/do** v2, Wiring in `internal/bootstrap` (`InitServices` + `Provide` je Paket)
- YAML-Config (`internal/config`, `-c`, `${}`, `secretfile`)
- Infrastructure: `shttp`, `health`, `logging` (slog, GELF, VictoriaLogs), OTEL, Prometheus
- Inbound: `adapter/inbound/http` (apiv1, JWT in `.../auth`, Swagger UI)
- Outbound-Muster: Port im Domain-Paket (`AddressStorage`), Implementierung + Factory unter `adapter/outbound/...`, `Provide` registriert die Port-Implementierung
- `internal/shared` (serror, httputils), `pkg/web` Embed, `pkg/pmodel`, `api/` Swagger, `bruno/`, `scripts/`
- Docker-Build des Templates

Nicht übernommen / umgebaut:

- Demo-Bounded-Context **addresses** komplett entfernen (`domain/addresses`, `adapter/outbound/address` inkl. MySQL, Address-Handler, `pkg/client` Address-API)
- **Keine externe DB:** Outbound-Factory analog `address.Provide` mit Typ `sqlite` (`modernc.org/sqlite`); optional `memory` nur für Tests
- Modulname **`github.com/willie68/arcivio`**, Servicename `arcivio`

Ergänzungen (als Domain + Ports + Adapter, nicht als `internal/services`):

- **Frontend:** Vue 3, PrimeVue 4, Vite, TypeScript; Build nach `pkg/web/client`
- **Suche:** Bleve v2 (Text + Vector, Build-Tag `vectors`) als Outbound
- **Krypto:** SHA-256, CMS/PKCS#7; At-Rest PQC in Infrastructure `cryptofs`
- **Auth:** Domain Identity; JWT-Middleware bleibt Inbound wie im Template; Login/OIDC stellen dasselbe JWT aus

## Datenmodell (Kern)

- **User / Role / Permission / Group**
- **DocumentType:** Name, Schema der Metafelder, Default-Aufbewahrung, Archivpflicht
- **Document:** Typ, Status (`inbox` / `active` / `archived`), MIME, Originalname, Verweis auf aktuelle Blob-Version
- **Version:** nur wenn sich **archival** Inhalt oder **archival** Attribute ändern; Blob nie überschreiben
- **RetentionPolicy** und **LegalHold**
- **AuditEvent:** wer/was/wann/Bereich/Ergebnis, Hash-Kette; Speicherung nur im Audit-Archiv-Store, nicht als fluides SQLite-Log
- **ArchiveVolume / ArchiveRecord:** Container, Offset, Länge, Record-Hash, Siegel/Signatur
- **ExtractorJob:** Queue für externe Services
- **SearchDoc:** Bleve-ID = Dokument-UUID; Index speist sich aus archival + aktuellem fluidem Stand

Dokumenttypen sind **Daten**, kein Hardcode. UI rendert Felder aus dem Schema.

### Archival vs. fluide Attribute

Jedes Feld (Metadatum) am Typ hat `persistence: archival | fluid`.

**Archival (archivierungswürdig)** – Teil der revisionssicheren Identität:

- Originaldatei (Bytes), Content-Hash, Dateiname/MIME zum Archivzeitpunkt
- Geschäftliche Festschreibungen: Rechnungsnummer, Betrag, Vertragsparteien, Belegdatum, Aufbewahrungsfrist, Dokumenttyp
- Gehen ins Archiv-Volume (`META` + `BLOB`); Änderung **nach** Archivierung erzeugt eine **neue Version** und einen neuen Archiv-Record (oder ist verboten, je nach Typ/Rolle)

**Fluid** – Ableitungen und Hilfsdaten, nicht identitätsstiftend:

- OCR-/Extractor-Volltext, Konfidenz, Sprachcode, Embedding-Vektoren, KI-Zusammenfassung, nachträglich korrigierte Suchbegriffe, UI-Tags ohne Rechtswirkung
- **Fluide Dateiattribute** (eigene Dateien im Fluid Store, nicht im Archiv): insbesondere **Prerender-PDF** (`previewPdf`) für die Blob-Anzeige; gebunden an Content-Hash des Quell-Blobs, bei neuer Engine überschreibbar ohne Dokumentversion
- Aus **Nutzersicht** am Dokument sichtbar (eine Maske, ein Datensatz)
- Technisch **eigener Nicht-Archiv-Speicher** (SQLite + optional Dateien unter `data/derived/<doc-id>/`), keyed nur über Dokument-ID, **ohne** Volume-Append
- Nach Archivierung **beliebig ersetzbar** (neue OCR, bessere KI): kein neues Dokument, keine Blob-Version, Archiv-Siegel unberührt
- Bleve wird bei fluidem Update neu geschrieben; Suchtreffer folgen dem aktuellen fluiden Stand

Schema-Beispiel (konzeptuell): `invoiceNumber` archival, `ocrFulltext` / `embedding` fluid. Systemfelder `fulltext` und `embedding` sind immer fluid, auch ohne Typ-Definition.

Regel: Versionierung entscheidet sich am **archival Diff**, nicht am OCR-Ergebnis.

## Speicherung und Archiv

Drei Schichten:

1. **Working Store:** Originale als Content-Addressed Blobs (`data/blobs/ab/cd/<sha256>`), **archival** Metadaten in SQLite. Alltag, Vorschau, erneutes Archivieren.
2. **Fluid Store (kein Archiv):** Volltexte, Embeddings, Ableitungen und **Prerender-PDFs** (`data/derived/…` + SQLite-Tabellen `document_fluid`). Überschreibbar; optional Historie der letzten N Läufe nur für Nachvollziehbarkeit, **nicht** siegelrelevant.
3. **Archiv:** Append-Only-**Volumes**, z. B. `archive/vol-000042.arc`. Pro archivierter Dokumentversion **ein selbsttragendes Objekt** (Envelope). Fluid wird **nicht** in den Container geschrieben.

**Selbsttragendes Archivobjekt:** Ein Record enthält alles, was nötig ist, das Dokument **allein aus den Containern** wiederherzustellen – ohne SQLite, ohne Working-Blobs, ohne Fluid Store:

- Dokument-ID, Versionsnummer, Archivzeitpunkt, Retention/LegalHold
- **Snapshot des Dokumenttyps** (ID, Name, Schema der archival Felder zum Archivzeitpunkt), damit Felder ohne aktuelle Typ-Registry lesbar bleiben
- alle **archival** Attribute (JSON)
- Originalbytes, MIME, Originaldateiname, Content-Hash
- optional Vorgänger-Versions-ID

Kein Verweis nach außen (kein Blob-Pfad, keine DB-ID als einzige Quelle). Ein Dokument darf nicht auf mehrere Records/Volumes aufgeteilt werden, außer der Envelope selbst liegt vollständig in einem Volume; sehr große Dateien: ein Envelope mit eingebettetem Payload, notfalls ein Volume nur für dieses Objekt. SQLite ist ein **Index/Cache** über die Container, nicht die Quelle der Wahrheit für archivierte Dokumente.

**Wiederherstellung (Disaster Recovery / Migration):** Volumes in ID-/Zeitreihenfolge scannen → `DOC` anwenden → spätere `DEL` für dieselbe Dokument-/Versions-ID ausblenden → Working Store und Metadaten neu aufbauen. **Fluide Attribute werden nicht wiederhergestellt**. Nach Restore: nicht gelöschte Dokumente und archival Felder vorhanden, OCR/Volltext/Embeddings leer. Bei At-Rest-Verschlüsselung ist der KEK/Passphrase nötig.

**Löschen im Archiv (Tombstone, Append-Only):** Es wird **nicht** im Volume geändert, das den `DOC`-Record trägt (wichtig, wenn versiegelte Container auf Read-only-/WORM-Medien liegen). Stattdessen hängt das System an das **aktuell offene** Volume einen **`DEL`-Record** (Dokument-ID, Versions-ID, Zeitpunkt, Akteur-Referenz, Grund/Retention). Ist kein Volume offen (alles versiegelt), wird ein neues Volume nur für Folge-Records inkl. `DEL` eröffnet. Replay: letzter `DEL` gewinnt gegenüber älterem `DOC`. Die Originalbytes bleiben im alten Container; Speicherfreigabe höchstens später durch optionales Umkopieren in neue Volumes, nie durch Patch des alten Files. Working Store/SQLite/Index: Dokument als gelöscht führen. Fluid darf mitgelöscht werden (kein Archiv). Steuerrelevante Typen: Vier-Augen vor dem `DEL`-Append.

**Audit:** Jedes Archiv-Löschen wird **immer** als Audit-Event geschrieben (nicht abschaltbar), unabhängig von `document.cud`.

**Containerformat (einfach, inkrementell):** eigenes Binary-Format, nicht ZIP (Append bei ZIP ist unzuverlässig).

- Dateikopf: Magic `ARCV`, Version, Volume-ID
- Records nacheinander: `len | type | payload | sha256`
- Record-Typen: `DOC` (Dokument-Envelope), `DEL` (Löschdatensatz, verweist auf DOC-ID/Version), `AUDIT` (nur Store `_audit`), `MANIFEST`, `SEAL`
- Offenes Volume wächst nur am Ende (O_APPEND)
- Ab Schwelle (z. B. 256–512 MiB) oder Zeitplan: **SEAL** (Manifest aller Record-Hashes + Merkle-Root) und **CMS-Signatur** mit internem oder importiertem Zertifikat
- Danach Volume read-only; neues Volume. Read-only-Medien: nur **versiegelte** Volumes auslagern; `DEL` lebt danach nur auf beschreibbaren Folgevolumes.

Integrität: Record-Hashes, Volume-Merkle-Root, Signatur, periodische Verify-Jobs. Fluide Daten können mit dem Dokument logisch gelöscht oder unabhängig neu erzeugt werden.

Signierung: interne CA (erster Admin erzeugt Root+Signing-Cert, Root offline/exportierbar) oder PKCS#12/PEM-Import. Signiert wird das **Siegel**; jedes `DOC`- und `DEL`-Envelope ist über Hash im Manifest gebunden. Fluide Attribute sind **nicht** Teil der Signatur.

## At-Rest-Verschlüsselung (optional, PQC)

Alle Dateien auf dem Speicherlaufwerk (Working-Blobs, Fluid/`data/derived`, Archiv-Volumes inkl. `_audit`, **SQLite-Datei**) können **optional** verschlüsselt werden. Default: **aus**. Ein Schalter pro Server in `service.yaml`; Schlüssel nur in `secretfile`.

**Ziel:** Schutz ruhender Daten (Diebstahl/Backup-Medium), nicht TLS. TLS bleibt klassisch oder separat.

**Konstruktion (hybrid, post-quantum-tauglich):**

- **DEK:** AES-256-GCM (symmetrisch mit 256-bit gilt als PQ-resistent genug)
- **Wrap der DEK:** Hybrid-KEM **ML-KEM-768** (FIPS 203) plus klassisch **X25519**, z. B. über Cloudflare CIRCL; beide Shares kombinieren (z. B. HKDF), bevor die DEK entschlüsselt wird
- Unlock beim Start: Passphrase und/oder Keyfile aus `secret.yaml` (nicht in Git)
- Dateiformat: kurzer Klartext-Header (`ARCE`, Algorithmen-IDs, KEM-Ciphertexts, Nonce-Salt), danach Ciphertext
- Ein DEK für die Instanz, rotiierbar (Re-Wrap; optionales Re-Encrypt der Dateien als Admin-Job)

**SQLite:** nicht unverschlüsselt daneben legen. Page-Level-VFS (AES-256-GCM pro Seite, DEK wie oben), damit die DB weiter normale Datei-I/O nutzt. Kein SQLCipher (nur AES, kein PQC-KEM). WAL/SHM liegen hinter demselben VFS.

**Bleve-Index:** ebenfalls über denselben verschlüsselten Dateizugriff, sonst läge Volltext klartextlich neben den Stores.

Modul: `internal/infrastructure/cryptofs` (oder Outbound-File-Port). Archive-/Fluid-/Working-/Store-Backends schreiben nur darüber. Unverschlüsseltes Alt-Verzeichnis: einmalige Migration beim ersten Einschalten.

```yaml
storage:
  encryption:
    enabled: false          # Default aus
    kem: ml-kem-768-x25519  # Hybrid
    dataCipher: aes-256-gcm
    # kekFile / passphrase: nur secret.yaml
```

Ohne `enabled` bleiben Dateien Klartext wie bisher. Umschalten auf an ohne Migration ist ein Fehlerstart (Header fehlt bzw. Klartext erkannt).

## Extraktion und Suche

- Adapter-Interface: HTTP-Callback oder Webhook (`POST /extract` an konfigurierbare URL, Ergebnis `text` + optionale `embedding[]`)
- DMS enthält **keinen** OCR/Parser
- Pipeline: Upload → Blob → Job → Ergebnis in **Fluid Store** → Bleve-Index
- Erneute Extraktion nach Archivierung: gleicher Dokument- und Blob-Hash, Fluid überschreiben, Index aktualisieren, **keine** neue Version
- **Index-Mapping ohne Field-Explosion:** keine dynamischen Top-Level-Felder pro Metadatum. Indexierbare Attribute (archival + fluid, soweit suchbar) als Liste:

```json
"Attributes": [
  {"Key": "color", "Value": "red"},
  {"Key": "size", "Value": "XL"},
  {"Key": "weight", "Value": "10kg"}
]
```

Nested/Bleve: `Attributes` wiederholt, `Key` keyword, `Value` text (Analyzer + n-gram/reverse für führende Wildcards), zusätzlich typisiert `ValueNum` / `ValueDate` wenn das Schema Zahl oder Datum ist. `fulltext` und `embedding` bleiben eigene Felder. Attributsuche: Conjunction `Key=<name>` AND (`Value`/`ValueNum`/`ValueDate` je Query).
- Query-Parser in `domain/search` (eigene Grammatik, nicht Bleve-Query-String), Outbound Bleve übersetzt Boolean/Wildcard/Phrase/Range/Nested
- GET Dokument: eine Ressource; Response trennt intern `archival` und `fluid`, UI zeigt beides zusammen

## Use Case Blobview

Der Client zeigt den **aktuellen Original-Blob** (archival) an. Die Anzeige hängt vom MIME/Typ ab; das Original bleibt unverändert.

**Nativ (kein Prerender):**

- Bilder: JPEG, PNG, GIF, WebP, BMP – HTML-`<img>` / einfacher Image-Viewer
- PDF: nativer PDF-Viewer im Frontend (z. B. pdf.js)

**Prerender (fluides Dateiattribut `previewPdf`):**

- EML, MSG, Office (mindestens DOCX, ODT; analog ODS/ODP/XLSX/PPTX sofern die Engine sie kann)
- der Render-Use-Case wandelt den Original-Blob nach **PDF** und legt das Ergebnis als **fluides File-Attribut** ab (`data/derived/<doc-id>/preview.pdf` plus Metadaten: MIME, Größe, Quell-Hash, Engine-Version)
- Frontend zeigt dieses PDF im **gleichen PDF-Viewer** wie native PDFs
- Fehlt das Prerender oder ist der Quell-Hash veraltet: Job anstoßen, UI „wird vorbereitet“; nach Archivierung erneutes Rendern **ohne** neue Dokumentversion
- Nach Restore aus Containern fehlt `previewPdf` (fluid) – Rendern läuft neu

**Renderengine:** Domain-Port + Outbound `adapter/outbound/render` (LibreOffice/soffice, EML/MSG→PDF). Nicht der externe OCR-Extractor. Konfigurierbarer Binary-Pfad in `service.yaml`. Formate ohne Viewer und ohne Engine: Download-only.

Audit: Anzeige folgt `document.blobRead` (Default aus).

## Use Case einfache Suche

Ein Suchfeld. Die Zeichenkette wird vom Parser interpretiert (Groß/Kleinschreibung der Operatoren wie `+` und `=` streng; Terme default case-insensitive). Audit: `index.search` (Default aus).

**Volltext** (Feld `fulltext`, OCR + suchbare Texte):

- `Wilfried` – Term
- `"Wilfried Klaas"` – Phrase
- `Wilf*` – Prefix-Wildcard (Wörter mit Präfix `wilf`)
- `Wilf?` – genau ein Zeichen nach `wilf` (`Wilf1`, `Wilfr`, `wilfa`; nicht `wilf12`, nicht `Wilfried`)
- `*fried` / `?fried` – führende Wildcard (über n-gram/reverse-Index, nicht naives Leading-Wildcard-Scan)

**Verknüpfung ohne Feld:**

- `Wilfried Klaas` – **ODER** (Dokumente mit Wilfried **oder** Klaas)
- `Wilfried +Klaas` – **UND** (`+` = Muss-Term)

**Attributsuche** (`Key=…` gegen `Attributes`; Phrase und Wildcards analog Volltext):

- `Name=Wilfried` – Feld `Name` enthält Term Wilfried
- `Name="Wilfried Klaas"` – Phrase im Feld
- `Name=Wilf*` / `Name=*fried` – Wildcards im Feld
- `Vorname=Wilfried +Name=Klaas` – UND über zwei Felder
- `Name=Klaas Name=Maier` – ODER im selben oder in Feldern (Juxtaposition = ODER)

**Klammern:** `(Name=Klaas +Vorname=Wilfried) +(Datum=16.02.26)` – Gruppe UND Datum. Datum `DD.MM.YY` oder `DD.MM.YYYY`.

**Bereiche** (`von..bis`, inklusiv):

- `16.02.26..17.02.26` – irgendein indexiertes Datum im Intervall
- `1234..1340` – irgendeine indexierte Zahl im Intervall
- `Datum=16.02.26..17.02.26` – nur Attribut `Datum`
- `Rechnungsnummer=1234..1340` – nur dieses Attribut

Parser-Fehler: HTTP 400, UI zeigt die Stelle. KI/Vektor-Suche ist ein **anderer** Modus (Toggle), nicht diese Grammatik.

## AuthN / AuthZ

Baut auf Template-JWT auf (`adapter/inbound/http/auth`, Config-Block `auth` in `service.yaml`). Use Cases in `domain/identity`; Persistenz Outbound SQLite.

**Interner IdP** (lokal, kein Keycloak): Benutzername + Argon2id-Passwort. SSO (Entra/Apple) ist zusätzlich; nach OIDC entsteht dasselbe interne JWT.

- JWT im Header oder Cookie; CSRF wenn Cookie-basiert
- RBAC: `admin`, `archivist`, `clerk`, `reader`; Feingranular `document.create`, `archive.seal`, `retention.dispose`
- Nutzerfeld `mustChangePassword` (boolean)

**Bootstrap beim ersten Start:** Ist die Nutzertabelle leer, wird genau ein User `admin` mit Passwort `admin` und Rolle `admin` angelegt, `mustChangePassword=true`. Kein zweites automatisches Anlegen, wenn schon Nutzer existieren.

**Passwortänderungspflicht (interner IdP):**

- Login mit gültigem Passwort, aber `mustChangePassword`: kein volles API-JWT. Nur Wechsel-Token bzw. `POST /api/v1/auth/change-password` (alt + neu). UI zwingt zur Änderung **bei/sofort nach** der ersten Anmeldung (Admin mit `admin` ebenso).
- Nach erfolgreichem Wechsel: `mustChangePassword=false`, normales JWT.
- Legt ein User einen anderen User an: der interne IdP setzt ein **zufälliges** Einmalpasswort (nur einmal an den Anleger zurück, nicht persistiert im Klartext), Zieluser `mustChangePassword=true`. Gleiche Login-Regel.

OIDC-Nutzer ohne lokales Passwort: kein `mustChangePassword` über dieses Passwort-Protokoll. TOTP später optional.

## Auditlog

Alle fachlichen Vorgänge können in ein **Auditlog** geschrieben werden. Das Log liegt **nicht** in SQLite als führendem Speicher, sondern direkt in einem **Archiv-Store** (gleiche Container-/Envelope-Technik wie Dokumente, Record-Typ `AUDIT`).

**System-Store `_audit`:** Beim ersten Start automatisch angelegt (Volumes unter `archive/_audit/`). Nur Rolle `admin` sieht und durchsucht ihn in der UI; kein normales Dokumentfach. Restore der Audit-Container stellt die Protokolle wieder her (selbsttragende Events inkl. Hash-Vorgänger). Schreiben ins Auditlog wird **nicht** rekursiv auditiert; Lesen des Audit-Stores folgt den Document-Read-Schaltern nicht, sondern einem eigenen Admin-Pfad ohne Extra-Audit (sonst Rekursion).

Jedes Event: Zeit (UTC), Akteur, Bereich, Aktion, Objekt-IDs, Ergebnis (ok/deny/fail), optionale Diff-Skizze (keine Passwörter, keine Blob-Bytes), `prevHash` + eigener Hash.

Schalter in `configs/service.yaml` (pro Server, nicht pro Tenant):

```yaml
audit:
  authn:
    logins: false          # erfolgreiche Logins (SSO und lokal); Default aus
    loginAttempts: false   # zusätzlich alle Versuche (Fail/Success); nur sinnvoll wenn logins oder dieses Flag an
    userManagement: true   # User Create/Update/Delete; Default ein
  authz:
    permissionManagement: true  # Create/Update/Delete von Rollen/Rechten; Default ein
  document:
    cud: true              # Dokument CUD; Default ein
    read: false            # Dokument Read; Default aus
    blobCud: true          # Blob CUD; Default ein
    blobRead: false        # Blob Read/Download; Default aus
  index:
    cud: false             # Index anlegen/ändern/löschen; Default aus
    search: false          # Suchanfragen; Default aus
```

`loginAttempts: true` bei `logins: false` protokolliert nur Fehlversuche (und konfigurierbar alle Versuche); bei beiden an: jeder Versuch plus Erfolg.

Fluide OCR-Updates zählen als Document-Update nur wenn `document.cud` an ist (kein Blob-CUD, Datei unverändert). Index-Rebuild nach OCR: `index.cud`, Default aus.

## Frontend (PrimeVue)

Selbsttragend: Vite-Build nach `pkg/web/client`; `shttp` liefert SPA + `/api/v1` wie im Template.

Oberflächen (MVP+): Login/SSO, Inbox/Upload, Dokumentliste + Filter, Dokumentdetail mit **Blobview** (Image/PDF nativ, sonst Prerender-PDF), Typschablonen-Editor, **ein Suchfeld** (Volltext/Attribute/Range), Suche-KI-Toggle, Archiv-Volumes (Status, Verify, Siegel), **Audit-Store (Admin)**, Benutzer/Rollen, Extraktor-URL.

## Repository-Struktur (Clean / hexagonal, wie go-micro)

Aus der Vorlage belassen: `cmd/service`, `internal/bootstrap`, `internal/config`, `internal/infrastructure/{shttp,health,logging}`, `internal/adapter/inbound/http/{api,apiv1,auth}`, `internal/shared`, `pkg/web`, `pkg/pmodel`, `api/`, `bruno/`, `configs/`, `scripts/`.

**Domain (Use Cases + Ports), statt `internal/services`:**

- `internal/domain/document` – Typen, Versionen (archival Diff), Ports für Blob/Metadaten
- `internal/domain/fluid` – OCR/Volltext/Embeddings/Prerender-PDF
- `internal/domain/archive` – selbsttragende Envelopes, `DEL`-Tombstones im offenen Volume, Restore, Siegel
- `internal/domain/audit` – Schalter, Hash-Kette, Port zum Audit-Store
- `internal/domain/search` – Query-Parser, Index-/Such-Port
- `internal/domain/identity` – interner IdP, mustChangePassword, Bootstrap-Admin, OIDC-Orchestrierung
- `internal/domain/extract` – Extraktionsjobs (Port nach außen)
- `internal/domain/render` – Prerender nach PDF

**Outbound (Implementierungen der Ports), analog `adapter/outbound/address`:**

- `adapter/outbound/store/sqlite` – Factory `type: sqlite` (kein MySQL)
- `adapter/outbound/blob` – Working-Blobs
- `adapter/outbound/archive` – Volume-Dateien
- `adapter/outbound/fluid` – derived files
- `adapter/outbound/search/bleve`
- `adapter/outbound/extract/http`
- `adapter/outbound/render`
- `adapter/outbound/identity/sqlite`

**Inbound:** REST-Handler in `adapter/inbound/http/apiv1` rufen nur Domain-Interfaces auf (wie `Address` im Template), kein direkter Store-Zugriff.

**Infrastructure:** `cryptofs` für At-Rest; Wiring in `bootstrap.InitServices` (`Provide` der Outbounds, dann `domain/*.Provide`, dann `shttp.Provide`).

Frontend-Quellen `web/`, Output `pkg/web/client/`. Laufzeit `data/` gitignored.

## Umsetzungsphasen

### Phase 1 – Gerüst (für den Start ausreichend definiert)

Ziel: lauffähiger Arcivio-Binary-Stub auf Basis des **aktuellen** go-micro, ohne Address-Demo, ohne DMS-Fachlogik.

**Vorgehen (Ordner existiert schon, `PLAN.md` behalten):**

1. Inhalt von `H:\privat\git-sourcen\go-micro` nach `H:\privat\git-sourcen\Arcivio` kopieren (ohne `.git` der Vorlage, **`PLAN.md` nicht überschreiben**).
2. Sofort Workspace-Root setzen.
3. Git in Arcivio initialisieren, falls noch keins.
4. Rename: `go.mod` → `github.com/willie68/arcivio`, `config.Servicename` → `arcivio`, Swagger-Titel, Scripts/Docker-Namen, alle Importpfade.
5. Address-Bounded-Context entfernen: `domain/addresses`, `adapter/outbound/address`, Address-Handler in `apiv1`, `pkg/client` (Address-API), zugehörige Tests/Testdaten/`addressstorage` in YAML.
6. **Kein Tenant-Header** (SOHO, keine Mandanten): Middleware/Pflicht aus go-micro streichen.
7. Config: `addressstorage` ersetzen durch `storage.type: sqlite` und Pfad z. B. `${configdir}/data/arcivio.db` (noch leeres Schema / nur `schema_migrations`).
8. `bootstrap.InitServices`: Health + shttp; optional leeres `domain/document` + Outbound SQLite `Provide` (Ping/Open).
9. Frontend: Vite + Vue 3 + PrimeVue in `web/`, Build nach `pkg/web/client` (Platzhalter-Seite „Arcivio“). Template-`index.html` ersetzen.
10. `data/` in `.gitignore`. Ports lokal wie Vorlage (HTTP/HTTPS aus `service_local.yaml`).

**Fertig, wenn:** `go test ./...` grün ohne Address-Pakete; Binary startet mit `-c configs/service_local.yaml`; `/livez`, `/readyz`, `/swagger/` erreichbar; `/api/v1/addresses` weg; SPA vom Binary ausgeliefert.

### Weitere Phasen (Kurz)

2. **Auth lokal + RBAC** – interner IdP, Bootstrap `admin`/`admin` + Pflichtwechsel, Zufallspasswort bei Anlage durch andere; JWT-Signing-Key in `secret.yaml`
3. **Dokumente + Typen + Blob-Store**
4. **Archiv-Volumes:** selbsttragende Envelopes, `DEL` im aktuellen Container, Restore, Siegel/Signatur
5. **Auditlog:** Store `_audit` auto-anlegen, YAML-Schalter, Hash-Kette
6. **At-Rest-Crypto:** optional ML-KEM-Hybrid + AES-256-GCM für alle Stores inkl. SQLite/Bleve
7. **Extractor-Adapter + Bleve FTS** (KV-Attribute, Query-Parser)
8. **Vektorfeld + Hybrid-Suche**
9. **OIDC Entra + Apple**
10. **Blobview:** native Viewer + Outbound-Render + fluides `previewPdf`
11. **PrimeVue-UI** der Kernflows, Single-Binary-Release

## Bewusst nicht im ersten Wurf

TR-ESOR/ERS, Hardware-WORM, S/MIME je Datei, Mandantenfähigkeit, Cluster, externe DB, eingebaute OCR.
