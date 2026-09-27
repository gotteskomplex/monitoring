# ADR-0003: Interne zweistufige CA, 90-Tage-Zertifikate, Sperrung per DB

**Status:** Vorgeschlagen · **Datum:** 2026-09-27

## Kontext
Jeder Satellit braucht eine eigene, rotier- und widerrufbare Identität. Der einzige Prüfer ist der Master.

## Entscheidung
- Offline-Root-CA → Online-Intermediate im Master (Schlüssel KEK-verschlüsselt).
- Schlüssel wird **auf dem Satelliten erzeugt** (ECDSA P-256), Enrollment per CSR + Einmal-Token (24 h, nur Hash gespeichert).
- Client-Zertifikate 90 Tage, automatische Erneuerung ab Tag 30 über den bestehenden Stream.
- Identität: `CN=<satellite_id>`, URI-SAN `urn:monitoring:tenant:<tenant_id>:satellite:<satellite_id>`.
- **Sperrung per DB-Abgleich** der Seriennummer bei jedem Handshake (Cache 30 s); aktive Streams werden bei Sperrung getrennt. Keine CRL/OCSP.

## Konsequenzen
- (+) Einfach, sofort wirksame Sperrung, keine CRL-Verteilung.
- (−) Satellit > 90 Tage offline muss neu enrollt werden (dokumentiert).
- (−) Verlust des Intermediate-Schlüssels ⇒ Neu-Enrollment aller Satelliten; daher getrenntes Backup des Schlüsselmaterials.
