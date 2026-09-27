# CLAUDE.md

Leitfaden für die Arbeit in diesem Repository. Bei jeder Phase aktuell halten.

## Projekt
Mandantenfähiges Monitoring-System (Master/Satellit) für einen MSP. Master (EU-gehostet) verwaltet Tenants, Konfiguration, Auswertung, Alarmierung, UI. Satelliten im Kunden-LAN führen Checks aus und liefern nur Ergebnisse.
Architektur: `docs/ARCHITECTURE.md` · Entscheidungen: `docs/adr/`

## Aktueller Stand
**Phase 1 (Fundament) abgeschlossen.** Architektur freigegeben (2026-09-27). Nächster Schritt: Phase 2 (Satellit-Kern + Gateway: CA, Enrollment, gRPC/mTLS, Scheduler, Ping/TCP/HTTP, SQLite-Puffer). Änderungshistorie: `CHANGELOG.md`.

## Befehle
```
make tools        # gepinnte Generatoren + golangci-lint installieren (einmalig)
make gen          # buf generate, sqlc generate, oapi-codegen, openapi-typescript
make check-gen    # schlägt fehl, wenn generierter Code veraltet ist (CI)
make lint         # golangci-lint, buf lint, eslint, prettier --check, tsc
make test         # Unit-Tests Go (-race) + web (vitest), ohne Docker
make test-int     # Integrationstests (build tag "integration", Testcontainers, Docker nötig)
make build        # bin/master, bin/satellite (Host-Plattform)
make build-all    # Satellit linux amd64/arm64 + windows amd64, Master linux amd64/arm64
make web          # Web-UI bauen und nach internal/master/webui/dist kopieren (go:embed)
make run          # docker compose up --build (Postgres/Timescale, Bootstrap, Migration, Master, Traefik)
```
Lokal: http://localhost:8080 (Master direkt) bzw. https://app.localhost (über Traefik, selbstsigniert).
Web-Dev-Server: `cd web && npm run dev` (Proxy `/api` → localhost:8080).

## Master-Befehle und Konfiguration (Env, jeweils auch als `NAME_FILE`)
- `master bootstrap`: Rollen + Extension anlegen (Superuser, idempotent). Braucht `MON_DB_SUPERUSER_DSN` und `MON_DB_{MIGRATOR,APP,SYSTEM}_PASSWORD`.
- `master migrate`: goose-Migrationen als `mon_migrator` (`MON_DB_MIGRATOR_DSN`).
- `master serve`: HTTP (`MON_HTTP_ADDR`, Standard `:8080`), `MON_DB_APP_DSN`, optional `MON_DB_SYSTEM_DSN`, `MON_LOG_LEVEL`.
- `master healthcheck`: Container-Healthcheck (Distroless-Image hat keine Shell).
- Endpunkte: `/healthz`, `/readyz` (DB + Schemaversion == eingebettete Version), `/metrics`, `/api/v1/*`, `/` (SPA).

## Verbindliche Regeln
1. **Mandantentrennung**
   - Tenant-Kontext nur aus Session/API-Token bzw. Satelliten-Zertifikat, **nie** aus Request-Parametern.
   - DB-Zugriff nur über `store.WithTenantScope`; die Pools sind in `store.DB` nicht exportiert. `store.WithSystem` (BYPASSRLS) nur für echte Plattform-Jobs, mit Begründung.
   - Hypertables liegen im Schema `ts` ohne Rechte für `mon_app`, Zugriff über gleichnamige Views in `public` (ADR-0010). `WithSystem` muss `ts.*` direkt ansprechen. Neue Hypertables: gleiches Muster plus `GRANT … TO mon_system`.
   - Jede neue Mandanten-Tabelle: `tenant_id NOT NULL`, `ENABLE`+`FORCE ROW LEVEL SECURITY`, Policy, zusammengesetzte FKs. Der RLS-Meta-Test muss grün bleiben.
   - Satelliten dürfen nur Ergebnisse für **ihnen zugeordnete** Checks liefern.
2. **Protokoll** (`proto/monitoring/satellite/v1`): nur additive Änderungen, Feldnummern nie wiederverwenden (`reserved`). `buf breaking` muss grün sein. Master unterstützt Protokollversion N und N-1.
3. **Migrationen** (`migrations/`, goose): nach Merge nie ändern, nur neue anlegen.
4. **Secrets**: nie loggen, nie im API-Response zurückgeben (write-only), nie im Audit-Log im Klartext.
5. **Generierter Code** (`internal/gen/`, `web/src/gen/`) wird nie von Hand geändert.
6. **Satellit bleibt „dumm“**: keine Alarmlogik, keine Historie außer Retry-Zähler (ADR-0009).
7. Jede Phase: Tests grün, Doku + CLAUDE.md aktualisiert, wichtige Entscheidungen als ADR.

## Konventionen
- Sprache: Code, Bezeichner, Kommentare, Commit-Messages auf **Englisch**; Doku (`docs/`) und UI-Texte zuerst **Deutsch**, Englisch als zweite Sprache (i18n-Schlüssel, keine hartkodierten Texte).
- Go: gofumpt, golangci-lint; Fehler mit `%w` wrappen; `context.Context` als erster Parameter; Logging nur über `log/slog` (JSON).
- IDs: UUIDv7. Zeiten: UTC, `timestamptz`.
- SQL über `sqlc`, kein ORM.
- REST-Vertrag zuerst in `api/openapi.yaml` ändern, dann generieren.
- Tests: Tabellentests für Kernlogik; Integrationstests mit Testcontainers unter `test/integration` (Build-Tag `integration`, Datenbank über `internal/testutil/pgtest`).
- Sicherheitsrelevante Tests brauchen eine Gegenprobe: einmal prüfen, dass der Test bei entfernter Schutzmaßnahme wirklich fehlschlägt.
- DB-Statuswerte = Protobuf-Enum-Werte (1 OK, 2 WARNING, 3 CRITICAL, 4 UNKNOWN), nie umnummerieren.
- Web: React 19, TypeScript **5.9** (6/7 erst, wenn typescript-eslint/openapi-typescript es unterstützen), Texte nur über i18n (`web/src/locales/{de,en}.json`). Der API-Client (`web/src/api/client.ts`) ist aus OpenAPI generiert.
- Commits: Conventional Commits (`feat:`, `fix:`, `docs:`, …).

## Kernentscheidungen (Kurzfassung, Details in ADRs)
- Go-Monorepo, Master als modularer Monolith mit Rollen-Flags (ADR-0001)
- gRPC-Bidi-Stream, mTLS im Master terminiert, Traefik SNI-Passthrough (ADR-0002)
- Interne CA, 90-Tage-Zertifikate, Sperrung per DB-Abgleich (ADR-0003)
- PostgreSQL + TimescaleDB, erzwungene RLS, Caggs nur über filternde Views (ADR-0004)
- Envelope Encryption, DEK pro Tenant, Crypto-Shredding (ADR-0005)
- Satelliten-Puffer SQLite CGO-frei (ADR-0006)
- Config als versionierter Voll-Snapshot (ADR-0007)
- Koordination über Postgres (LISTEN/NOTIFY, SKIP LOCKED), NATS später (ADR-0008)
- Statusberechnung: Satellit bewertet Einzelergebnis, Master alles mit Historie (ADR-0009)
- Hypertables im Schema `ts`, Zugriff nur über Views (Chunks erben keine RLS) (ADR-0010)

## Stolpersteine
- In der Claude-Code-Cloud-Sandbox scheitern Docker-Builds an `go mod download`/`npm ci` (Proxy-CA im Build nicht vertraut). Zum lokalen Prüfen die Binaries auf dem Host bauen und per Compose-Override ein Minimal-Image verwenden. In CI oder auf normalen Rechnern ist das nicht nötig.
- Docker-Daemon in der Sandbox ggf. zuerst starten: `dockerd > /tmp/dockerd.log 2>&1 &`.
