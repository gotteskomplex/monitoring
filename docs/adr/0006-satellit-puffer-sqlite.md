# ADR-0006: Satelliten-Puffer in SQLite (CGO-frei)

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
Der Satellit muss bei Master-Ausfall Ergebnisse puffern und Konfiguration persistent halten; das Binary soll statisch gelinkt sein.

## Entscheidung
- SQLite über `modernc.org/sqlite` (reines Go), WAL-Modus.
- Tabellen `results` (FIFO), `latest_result` (je Check), `config` (verschlüsselt), `meta`.
- Limit Standard 500 MB / 7 Tage; bei Überschreitung werden älteste Rohergebnisse verworfen, `latest_result` bleibt; Verwerfungen werden gezählt und gemeldet.
- Nach Reconnect: zuerst `latest_result`, dann Historie mit Ratenbegrenzung.

## Konsequenzen
- (+) Robust, flexibel abfragbar, keine CGO-Abhängigkeit.
- (−) `modernc` ist langsamer als CGO-SQLite; für ~33 Ergebnisse/s irrelevant. Alternative bbolt bleibt möglich hinter einem `Queue`-Interface.
