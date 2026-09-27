# ADR-0010: Hypertables in eigenem Schema, Zugriff nur über Views

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
Beim Umsetzen von Phase 1 hat ein Test gezeigt: TimescaleDB kopiert die GRANTs einer Hypertable auf jeden Chunk (`_timescaledb_internal._hyper_*`), **nicht aber die Row-Level-Security**. Eine Rolle mit direkten Rechten auf eine Hypertable kann deshalb per `SELECT * FROM _timescaledb_internal._hyper_1_1_chunk` die Zeilen **aller** Tenants lesen und damit RLS umgehen.

Geprüfte Alternativen:
- `USAGE` auf `_timescaledb_internal` entziehen: bricht auch reguläre Inserts in die Hypertable.
- Rechte an Chunks nachträglich entziehen: Die Chunk-Erzeugung löst keinen Event-Trigger aus, neue Chunks wären bis zum nächsten Aufräumlauf offen.

## Entscheidung
- Hypertables liegen im Schema **`ts`**. `mon_app` hat darauf **keinerlei** Rechte (weder Schema noch Tabelle), daher erben Chunks auch keine Rechte für `mon_app`.
- Die Anwendung greift über gleichnamige **Views in `public`** zu (`public.check_results` → `ts.check_results`). Die Views gehören `mon_migrator`. Weil RLS per `FORCE` auch für den Eigentümer gilt und `app.tenant_ids` pro Transaktion gesetzt wird, filtert die Policy weiterhin nach dem Scope des Aufrufers. Die Views sind automatisch beschreibbar (INSERT/UPDATE/DELETE, auch `ON CONFLICT`).
- `mon_system` (BYPASSRLS) bekommt direkte Rechte auf `ts.*` und muss diese Tabellen auch direkt ansprechen. Über die Views sähe er wegen der RLS des View-Eigentümers nichts.
- Integrationstest `TestAppRoleCannotReachHypertablesOrChunks` prüft, dass `mon_app` keine Rechte am Schema `ts` und an keinem Chunk hat.

## Konsequenzen
- (+) Die Umgehung ist strukturell ausgeschlossen, nicht nur per Aufräumjob.
- (+) Für den Anwendungscode transparent: Tabellennamen bleiben gleich.
- (−) `COPY` direkt in eine View ist nicht möglich. Der Ingest (Phase 3) nutzt `COPY` in eine temporäre Tabelle plus `INSERT … SELECT … ON CONFLICT` über die View.
- (−) Neue Hypertables müssen dieselbe Konstruktion bekommen. Die Regel steht in CLAUDE.md, und der Meta-Test deckt RLS ab.
- Continuous Aggregates (Phase 3) folgen demselben Muster: Materialisierung in `ts`, tenant-filternde View in `public`.
