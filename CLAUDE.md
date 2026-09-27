# CLAUDE.md

Leitfaden für die Arbeit in diesem Repository. Bei jeder Phase aktuell halten.

## Projekt
Mandantenfähiges Monitoring-System (Master/Satellit) für einen MSP. Master (EU-gehostet) verwaltet Tenants, Konfiguration, Auswertung, Alarmierung, UI. Satelliten im Kunden-LAN führen Checks aus und liefern nur Ergebnisse.
Architektur: `docs/ARCHITECTURE.md` · Entscheidungen: `docs/adr/`

## Aktueller Stand
**Planungsphase** – Architektur zur Freigabe vorgelegt, noch kein Produktivcode. Nächster Schritt nach Freigabe: Phase 1 (Fundament).

## Befehle (Zielzustand ab Phase 1, noch nicht vorhanden)
```
make gen          # buf generate, sqlc generate, oapi-codegen, openapi-typescript
make lint         # golangci-lint, buf lint, eslint, prettier --check
make test         # go test ./... (Unit) + web: vitest
make test-int     # Integrationstests mit Testcontainers (Docker erforderlich)
make build        # Binaries: master, satellite, satellite-launcher (linux amd64/arm64, windows amd64)
make run          # docker compose -f deploy/compose/docker-compose.yml up --build
make migrate      # master migrate up
```

## Verbindliche Regeln
1. **Mandantentrennung**
   - Tenant-Kontext nur aus Session/API-Token bzw. Satelliten-Zertifikat, **nie** aus Request-Parametern.
   - DB-Zugriff nur über `store.WithTenantScope`; kein direkter Pool-Zugriff im Request-Pfad.
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
- Tests: Tabellentests für Kernlogik; Integrationstests mit Testcontainers unter `test/integration`.
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
