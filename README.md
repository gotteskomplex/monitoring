# Monitoring (Master/Satellit)

Mandantenfähiges Monitoring-System für Managed Service Provider: Ein zentraler Master (EU) verwaltet Mandanten, Konfiguration, Auswertung und Alarmierung. Satelliten im Kundennetz führen die Checks aus und verbinden sich nur ausgehend (gRPC, mTLS, 443).

- Architektur: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Entscheidungen (ADRs): [docs/adr/](docs/adr/)
- Arbeitsleitfaden, Befehle, Konventionen: [CLAUDE.md](CLAUDE.md)
- Änderungen je Phase: [CHANGELOG.md](CHANGELOG.md)

## Schnellstart (lokal)

Voraussetzungen: Docker mit Compose, Go 1.25, Node 22.

```bash
make run                      # Postgres/TimescaleDB, Bootstrap, Migration, Master, Traefik
curl localhost:8080/readyz    # {"status":"ready"}
open https://app.localhost    # Web-UI über Traefik (selbstsigniertes Zertifikat)
```

Tests:

```bash
make test       # Unit-Tests (ohne Docker)
make test-int   # Integrationstests inkl. Mandantentrennung gegen echte TimescaleDB
make lint
```

## Stand

Phase 1 (Fundament) ist abgeschlossen: Protokoll v1, DB-Schema mit erzwungener Row-Level-Security, Isolationstests, Master-Grundgerüst, Web-Stub, Compose-Umgebung, CI. Als Nächstes folgt Phase 2 (Satellit-Kern und Gateway).
