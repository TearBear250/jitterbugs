# Device Location Attestation Policy

## Overview

This document defines how jitterbugs terminal boxes source, validate, attest, and
share device location data.  It covers privacy defaults, partner access rules, audit
requirements, and the cryptographic attestation chain that proves a reported location
originated from a specific, unmodified device.

All location data handling follows the principle of **coarse-by-default, precise
opt-in**: a device reports only a reduced-resolution position unless the site operator
has explicitly enabled precise location sharing.  Law-enforcement escalation that
requires sharing a precise position requires human operator confirmation unless the
site has separately enabled auto-escalation in its site policy.

---

## Location Sources

| Source ID      | Description                                         | Typical accuracy |
|----------------|-----------------------------------------------------|-----------------|
| `GNSS`         | On-device GPS/GLONASS/Galileo/BeiDou receiver       | 3–10 m CEP      |
| `AssistedGNSS` | GNSS with network-provided assistance data (A-GPS)  | 3–10 m CEP      |
| `WiFi`         | Wi-Fi positioning via observed BSSIDs               | 10–50 m         |
| `BLE`          | Bluetooth Low Energy beacon trilateration           | 1–5 m (indoor)  |
| `Cell`         | Cellular network cell-ID / RSSI-based positioning   | 100–5 000 m     |
| `Manual`       | Operator-entered coordinates stored in site config  | Site-defined    |

A device selects the highest-confidence source available at reporting time and records
`source` in the manifest.  `AssistedGNSS` is preferred over `GNSS` alone when network
connectivity permits.

---

## Location Fields and Accuracy

Every location fix included in a device manifest MUST contain:

| Field          | Type    | Required | Description                                         |
|----------------|---------|----------|-----------------------------------------------------|
| `latitude`     | number  | yes      | WGS-84 decimal degrees, –90 to +90                 |
| `longitude`    | number  | yes      | WGS-84 decimal degrees, –180 to +180               |
| `accuracy_m`   | number  | yes      | Estimated 1-σ horizontal accuracy in metres         |
| `source`       | string  | yes      | One of the Source IDs listed above                  |
| `hdop`         | number  | no       | GNSS Horizontal Dilution of Precision (when known)  |

Devices MUST NOT report a fix with `accuracy_m > 5000`.  If no source meets this
threshold the `location` object is omitted and the manifest field `location_unavailable`
is set to `true`.

**Coarse mode**: when a site has not opted in to precise location, latitude and
longitude are truncated to two decimal places (~1 km resolution) before signing and
transmission.  The `coarse` boolean field is set to `true` in the manifest.

---

## Attestation and Signing

### Device key provisioning

Each terminal box is provisioned at manufacture with an ed25519 key pair stored in a
hardware-backed Secure Element or TPM.  The private key never leaves the secure
enclave.  The associated `device_key_id` (SHA-256 fingerprint of the public key) is
registered in the jitterbugs device registry before deployment.

### Manifest signing

Before transmission, the device:

1. Builds the canonical UTF-8 JSON manifest (keys sorted, no extra whitespace).
2. Computes `SHA-256(canonical_json)` as the message digest.
3. Signs the digest with its ed25519 private key to produce `signature` (Base64url).
4. Includes a fresh `nonce` (32 random bytes, Base64url) to bind the manifest to the
   current challenge/response cycle.

The server verifies the signature against the registered public key before accepting
any manifest.

### Nonce / challenge flow

```
Device                           Server
  |  GET /location/challenge        |
  |-------------------------------->|
  |  { nonce, expires_utc }         |
  |<--------------------------------|
  |                                 |
  |  POST /location/attest          |
  |  { manifest (includes nonce) }  |
  |-------------------------------->|
  |  { accepted: true, token }      |
  |<--------------------------------|
```

Nonces expire after 120 seconds.  A manifest submitted with an expired or
already-used nonce is rejected with HTTP 409.

---

## Time Synchronisation

`timestamp_utc` in the manifest MUST be within ±30 seconds of the server's NTP-synced
clock at the time of receipt.  Manifests outside this window are rejected with HTTP
400 and error code `TIMESTAMP_SKEW`.

Devices synchronise time via NTP (or PTP where available) at boot and at least once
every 15 minutes.  The secure element enforces a monotonic clock that cannot be rolled
back by software.

---

## Tamper Sensors

The `tamper_flags` field is a bitmask reported by the device's IMU and enclosure sensors:

| Bit | Meaning                                           |
|-----|---------------------------------------------------|
| 0   | Enclosure opened since last attestation           |
| 1   | Unexpected tilt / orientation change detected     |
| 2   | Vibration above threshold in last 60 s            |
| 3   | GPS antenna continuity failure                    |
| 4   | Secure element reported integrity violation       |

A non-zero `tamper_flags` value does NOT automatically invalidate the manifest but MUST
be surfaced in the operations dashboard and triggers an alert to the site operator.

---

## Anti-Spoofing Checks

The server performs the following checks on every submitted manifest:

1. **Signature verification** — ed25519 signature must verify against registered device
   public key.
2. **Nonce freshness** — nonce must not have been seen before and must not be expired.
3. **Timestamp skew** — `timestamp_utc` within ±30 s of server time.
4. **Location plausibility** — reported location must be within the device's registered
   deployment region (configurable bounding box per site).  Outliers beyond 50 km of
   the registered site location are rejected unless the device has a `mobile` flag.
5. **Accuracy sanity** — `accuracy_m` must be positive and ≤ 5000 m.
6. **Source consistency** — if `source` is `GNSS` or `AssistedGNSS`, `hdop` should be
   present; if absent, a warning is logged.

---

## Consent and Privacy Defaults

| Setting                      | Default       | Override mechanism                              |
|------------------------------|---------------|-------------------------------------------------|
| Location resolution          | Coarse (~1 km)| Site operator enables `precise_location` in site policy |
| Location sharing with partners | Disabled    | Partner RBAC grant required per-site            |
| Retention period             | 7 days        | Site policy may extend up to 90 days            |
| Law-enforcement disclosure   | Requires human confirmation | Site may enable `auto_escalation` (see below) |

**Precise location opt-in**: A site operator may enable precise location by updating
the site policy and confirming acceptance of the associated data-handling addendum.

**Law-enforcement escalation**: By default, transmitting a precise device location to
law enforcement requires an operator to click "Confirm escalation" in the dashboard.
Sites that have signed the auto-escalation addendum may set `auto_escalation: true` in
their site policy, which allows the system to transmit location automatically when an
escalation event is triggered.  This setting requires annual renewal.

---

## Retention and Deletion

- Location fixes attached to non-escalated events are deleted after 7 days by default.
- Location fixes attached to escalated events are retained for 90 days or until the
  case is closed, whichever is later.
- Site operators may request deletion at any time via the data-deletion API; deletion
  is completed within 72 hours and confirmed by a signed deletion receipt.
- Backup copies are purged within the same retention window.

---

## Partner Access and RBAC

Access to location data is gated by the jitterbugs RBAC model:

| Role                  | Access                                                  |
|-----------------------|---------------------------------------------------------|
| `site_operator`       | Own-site location, full resolution if opted in          |
| `partner_viewer`      | Coarse location for assigned sites only                 |
| `partner_responder`   | Precise location for active work orders on assigned sites|
| `compliance_auditor`  | Audit log access, no raw location data                  |
| `law_enforcement`     | Precise location for authorised case, time-limited token|

Partner tokens are short-lived (1 hour, renewable) and scoped to a specific work order
ID.  Token issuance is logged in the audit trail.

---

## Audit Logs

Every access to location data generates an immutable audit record:

```json
{
  "audit_id": "aud_01HV…",
  "event_type": "location_access",
  "actor": "partner_id:acme-security",
  "resource": "device_id:dev_ABC123 / work_order_id:wo_XYZ",
  "action": "read",
  "resolution": "precise",
  "timestamp_utc": "2026-03-20T12:00:00Z",
  "source_ip": "203.0.113.42"
}
```

Audit logs are append-only, cryptographically chained (each record includes the
SHA-256 of the previous record), and retained for a minimum of 3 years.

---

## Example Device Manifest

```json
{
  "device_id": "dev_ABC123",
  "timestamp_utc": "2026-03-20T12:00:00Z",
  "nonce": "dGhpcyBpcyBhIHRlc3Qgbm9uY2U",
  "location": {
    "latitude": 37.7749,
    "longitude": -122.4194,
    "accuracy_m": 8.5,
    "source": "AssistedGNSS",
    "hdop": 1.2
  },
  "coarse": false,
  "firmware_version": "2.4.1",
  "model": "JB-T300",
  "device_key_id": "sha256:a1b2c3d4e5f6…",
  "tamper_flags": 0,
  "signature": "Base64url-encoded-ed25519-signature"
}
```

See `schemas/device_manifest.json` for the full JSON Schema definition and
`compliance/verify_location_signature.py` for a verification utility.
