# ADR-0001: Go-Monorepo, Master als modularer Monolith

**Status:** Vorgeschlagen · **Datum:** 2026-09-27

## Kontext
Master und Satellit teilen Protokolltypen. Für v1 genügt eine Master-Instanz (50 Tenants / 100 Satelliten / 50.000 Checks), horizontale Skalierung soll aber möglich bleiben.

## Entscheidung
- Ein Repository, **ein Go-Modul** für `cmd/master`, `cmd/satellite`, `cmd/satellite-launcher`; Frontend in `web/`.
- Der Master ist ein **modularer Monolith**: Module (`api`, `gateway`, `ingest`, `eval`, `alert`, `notify`, `jobs`) kommunizieren über Go-Interfaces, nicht über Netzwerk. Per `--roles` einzeln aktivierbar.
- Modulgrenzen werden durch Paketstruktur (`internal/master/<modul>`) und einen Architekturtest (verbotene Importe) abgesichert.

## Konsequenzen
- (+) Ein Binary, einfache Entwicklung, Deployment und Fehlersuche.
- (+) Aufteilung später durch Rollen-Flags statt Neuschreiben.
- (−) Disziplin nötig, damit Module nicht über gemeinsame Tabellen „heimlich“ koppeln.
