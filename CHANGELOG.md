# Changelog

## Phase 1: Fundament (2026-09-27)

**Fertig**
- Monorepo-Struktur, Makefile, CLAUDE.md, golangci-lint (v2), EditorConfig.
- Protokoll `monitoring.satellite.v1` (Enrollment, bidirektionaler Stream, Config-Snapshot, Ergebnisse, Befehle) mit `buf lint` und `buf breaking` in der CI. Generierter Go-Code eingecheckt.
- DB: Bootstrap (Rollen `mon_migrator`/`mon_app`/`mon_system`, TimescaleDB), goose-Migrationen für Tenants, Sites, Satelliten, Enrollment-Tokens, Zertifikate, Hosts, Abhängigkeiten, Checks, Check-State, Hypertables `check_results`/`metric_points`. Erzwungene RLS auf allen Mandanten-Tabellen, zusammengesetzte Fremdschlüssel.
- **Sicherheitsbefund:** TimescaleDB-Chunks erben GRANTs, aber keine RLS. Behoben durch Schema `ts` plus Views (ADR-0010) und per Test abgesichert.
- `store.WithTenantScope` / `store.WithSystem`, sqlc-Queries.
- Integrationstests (Testcontainers, TimescaleDB): RLS-Meta-Test, Chunk-Zugriff, Rechteausweitung, **Tenant A kann B weder lesen noch schreiben**, Migrationen down/up, Bootstrap-Idempotenz. Gegenprobe durchgeführt: Tests schlagen bei entfernter RLS fehl.
- Master: `serve` (`/healthz`, `/readyz`, `/metrics`, `/api/v1/version`, SPA-Auslieferung, Security-Header), `migrate`, `bootstrap`, `healthcheck`. JSON-Logging mit Secret-Redaction. Konfiguration per Env/`_FILE`.
- OpenAPI-Vertrag mit generiertem Go-Server und TS-Client.
- Web-Stub (React 19, Vite, i18n DE/EN, Vitest).
- Docker-Images (Distroless), Compose mit Traefik (TLS für `app.`, SNI-Passthrough für `sat.`).
- GitHub Actions: Lint, Unit- und Integrationstests, Prüfung des generierten Codes, Proto-Breaking-Check, Web, Cross-Builds, Multi-Arch-Images (Push nach GHCR auf dem Default-Branch und bei Tags).

**Noch offen / Nächstes (Phase 2)**
- Interne CA, Enrollment-API, gRPC-Gateway mit mTLS, Satellit mit Scheduler, Worker-Pool, Ping/TCP/HTTP-Checks, SQLite-Puffer, Dev-Satellit in Compose.
- API-Isolationstests folgen mit den ersten mandantenbezogenen Endpunkten (Phase 4).
