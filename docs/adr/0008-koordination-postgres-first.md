# ADR-0008: Koordination über PostgreSQL statt NATS in v1

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
Master-Instanzen sollen zustandslos und später mehrfach betreibbar sein. Config-Pushes müssen die Instanz erreichen, die den Stream eines Satelliten hält; periodische Jobs dürfen nur einmal laufen.

## Entscheidung
- `satellite_sessions` hält Stream-Zuordnung; Änderungen per `LISTEN/NOTIFY`.
- Periodische und asynchrone Arbeit über eine Jobtabelle mit `FOR UPDATE SKIP LOCKED` bzw. Advisory Locks.
- Benachrichtigungsversand über Outbox-Tabelle.
- Hinter Interfaces (`Bus`, `JobQueue`), damit NATS später austauschbar ist.

## Konsequenzen
- (+) Keine zusätzliche Infrastruktur; transaktional mit den Fachdaten.
- (−) NOTIFY ist nicht persistent → Reconciliation-Schleife nötig (ohnehin vorgesehen).
- (−) Grenze bei sehr vielen Instanzen/Events; dann NATS einführen (neues ADR).
