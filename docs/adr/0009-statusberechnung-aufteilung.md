# ADR-0009: Aufteilung der Statusberechnung Satellit/Master

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
Vorgabe: Satellit ist „dumm“ (keine Alarmlogik, keine Schwellwert-Historie), liefert aber Rohergebnisse **plus Status**. Wiederholungen vor Statuswechsel sollen Flapping vermeiden, ohne die Erkennung stark zu verzögern.

## Entscheidung
- **Satellit:** bewertet jedes Einzelergebnis zustandslos anhand der mitgelieferten Schwellwerte (→ OK/WARNING/CRITICAL/UNKNOWN). Nach Nicht-OK nutzt er `retry_interval` bis `max_attempts` (einziger Zustand pro Check).
- **Master:** Soft/Hard-State, Flapping (Ringpuffer mit Hysterese), Stale/UNKNOWN, Satellit-offline, Abhängigkeiten, Wartung, Alarme, Benachrichtigung. Rohmesswerte werden immer mitgespeichert, sodass der Master später bei Bedarf neu bewerten kann.

## Konsequenzen
- (+) Schnelle Erkennung, Satellit bleibt einfach; gesamte fachliche Logik zentral testbar.
- (−) Schwellwertänderungen wirken erst nach Config-Push (Sekunden).
- (−) Leichte Abweichung von „Satellit völlig zustandslos“.
