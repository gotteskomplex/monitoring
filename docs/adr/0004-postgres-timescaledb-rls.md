# ADR-0004: PostgreSQL + TimescaleDB mit erzwungener RLS

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
Stammdaten, Zustand und Zeitreihen sollen in einer Datenbank liegen; Mandantentrennung muss auch bei Anwendungsfehlern halten.

## Entscheidung
- PostgreSQL 16 + TimescaleDB 2.x; Hypertables für `check_results`, `metric_points`, `state_events`, `audit_log`; Kompression nach 2 Tagen; Continuous Aggregates (1 h).
- Jede Mandanten-Tabelle: `tenant_id NOT NULL`, `ENABLE` + `FORCE ROW LEVEL SECURITY`, Policy `tenant_id = ANY(current_setting('app.tenant_ids', true)::uuid[])` (fail closed).
- Anwendung verbindet als `mon_app` (kein Eigentümer, kein BYPASSRLS); Tenant-Kontext nur per `SET LOCAL` in `WithTenantScope`. Mandantenübergreifende Jobs über separate Rolle `mon_system`.
- Zusammengesetzte Fremdschlüssel inkl. `tenant_id`.
- Continuous Aggregates (ohne RLS-Unterstützung) nur über `security_barrier`-Views zugänglich.
- Meta-Test: jede Tabelle mit `tenant_id` hat erzwungene RLS.

## Konsequenzen
- (+) Zweite Verteidigungslinie unabhängig vom Anwendungscode.
- (−) Retention ist pro Hypertable, nicht pro Tenant → Stufenmodell (Abweichung A4).
- (−) Timescale-Lizenz (TSL) für Kompression/Caggs: SaaS-Nutzung erlaubt, aber kein Hyperscaler-Managed-Postgres mit diesen Features; juristische Prüfung vor Produktstart.
