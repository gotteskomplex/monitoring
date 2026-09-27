# ADR-0007: Konfiguration als versionierter Voll-Snapshot

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
Konfigurationsänderungen sollen versioniert an Satelliten gehen und bestätigt werden; Verbindungsabbrüche und verlorene Benachrichtigungen dürfen nicht zu Drift führen.

## Entscheidung
- Pro Satellit monotone `config_version`; jede relevante Änderung erhöht sie.
- Master sendet immer den **vollständigen Snapshot** (inkl. benötigter Secrets); Satellit bestätigt mit `ConfigAck(version, Fehler je Check)`.
- Satellit meldet seine Version in Hello und Heartbeat; Master gleicht periodisch ab (Reconciliation).

## Konsequenzen
- (+) Idempotent und selbstheilend, keine Delta-Reihenfolgeprobleme.
- (−) Größere Nachrichten (≈ 100 KB gzip bei 2.000 Checks) – unkritisch; Deltas später additiv möglich.
