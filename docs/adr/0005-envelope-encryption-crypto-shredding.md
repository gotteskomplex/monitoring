# ADR-0005: Envelope Encryption mit DEK pro Tenant

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
SNMP-Communities, v3-Credentials und HTTP-Auth müssen gespeichert und an Satelliten übertragen werden. Der Schlüssel darf nicht in der DB liegen; Tenant-Löschung soll DSGVO-konform sein.

## Entscheidung
- KEK außerhalb der DB über Schnittstelle `KeyProvider` (v1: eingehängte Datei/Docker Secret; später Vault/KMS/HSM).
- Pro Tenant ein DEK (AES-256-GCM), KEK-verpackt in `tenant_keys`.
- AAD = `tenant_id|credential_id`.
- Secrets im UI nur schreibbar; Entschlüsselung ausschließlich beim Bau des Config-Snapshots für den zuständigen Satelliten.
- Tenant-Löschung vernichtet den DEK (**Crypto-Shredding**, wirkt auch auf Backups).

## Konsequenzen
- (+) DB-Dump allein gibt keine Secrets preis; KEK-Rotation ohne Neuverschlüsselung aller Daten.
- (−) KEK-Verlust = Verlust aller Credentials → KEK getrennt und redundant sichern.
- (−) Auf dem Satelliten liegen Secrets zwangsläufig nutzbar vor (Risiko R3).
