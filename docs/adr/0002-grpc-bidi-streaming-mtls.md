# ADR-0002: gRPC-Bidi-Streaming mit mTLS, im Master terminiert

**Status:** Angenommen · **Datum:** 2026-09-27

## Kontext
Satelliten dürfen nur ausgehend verbinden (443). Der Master muss Konfiguration und Befehle jederzeit pushen können. Alternativen: gRPC-Streaming (HTTP/2) oder WebSocket.

## Entscheidung
- **gRPC** mit einem langlebigen bidirektionalen Stream `SatelliteService.Connect`, Enrollment als separater unärer Dienst ohne Client-Zertifikat.
- Eigener Hostname `sat.<domain>`; **Traefik leitet per SNI-Passthrough** weiter, **mTLS wird im Master-Prozess terminiert**.
- Unterstützung expliziter HTTP-CONNECT-Proxys; kein WebSocket-Fallback in v1.
- Protobuf-Paket `monitoring.satellite.v1`, nur additive Änderungen (`buf breaking` in CI), zusätzlich `protocol_version` im Hello; Master unterstützt N und N-1.

## Konsequenzen
- (+) Typisierte, geteilte Schnittstelle; Streaming, Keepalive, Kompression eingebaut.
- (+) Keine Vertrauensgrenze im Proxy; Identität kommt direkt aus dem geprüften Zertifikat.
- (−) HTTP/2 + mTLS scheitert an TLS-aufbrechenden Proxys → Ausnahme beim Kunden nötig (Risiko R1). Ein WebSocket-Fallback würde daran nichts ändern.
- (−) Zwei Hostnamen/Zertifikatspfade (öffentlich für `app.`, interne CA für `sat.`).
