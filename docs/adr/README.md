# Architecture Decision Records

Format: Kontext, Entscheidung, Konsequenzen. Status: `Vorgeschlagen` → `Angenommen` / `Abgelehnt` / `Ersetzt durch ADR-XXXX`.
Neue ADRs fortlaufend nummerieren; angenommene ADRs werden nicht umgeschrieben, sondern durch neue ersetzt.

| Nr. | Titel | Status |
|---|---|---|
| [0001](0001-go-monorepo-modularer-monolith.md) | Go-Monorepo, Master als modularer Monolith | Angenommen |
| [0002](0002-grpc-bidi-streaming-mtls.md) | gRPC-Bidi-Streaming mit mTLS, im Master terminiert | Angenommen |
| [0003](0003-interne-ca-kurzlebige-zertifikate.md) | Interne zweistufige CA, 90-Tage-Zertifikate, Sperrung per DB | Angenommen |
| [0004](0004-postgres-timescaledb-rls.md) | PostgreSQL + TimescaleDB mit erzwungener RLS | Angenommen |
| [0005](0005-envelope-encryption-crypto-shredding.md) | Envelope Encryption mit DEK pro Tenant | Angenommen |
| [0006](0006-satellit-puffer-sqlite.md) | Satelliten-Puffer in SQLite (CGO-frei) | Angenommen |
| [0007](0007-config-snapshots-versioniert.md) | Konfiguration als versionierter Voll-Snapshot | Angenommen |
| [0008](0008-koordination-postgres-first.md) | Koordination über PostgreSQL statt NATS in v1 | Angenommen |
| [0009](0009-statusberechnung-aufteilung.md) | Aufteilung der Statusberechnung Satellit/Master | Angenommen |
| [0010](0010-hypertables-eigenes-schema.md) | Hypertables in eigenem Schema, Zugriff nur über Views | Angenommen |
