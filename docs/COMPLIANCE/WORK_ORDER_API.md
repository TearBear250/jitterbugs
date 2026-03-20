# Work Order API — Security Partner Integration

## Overview

This document specifies the work-order lifecycle, webhook schema, authentication model,
and API conventions that security partners integrate with when jitterbugs generates an
escalation event.

Work orders are the primary integration artifact between jitterbugs and partner
security-response companies.  A work order is created when an escalation event exceeds
the configured threshold and a human operator (or an auto-escalation policy) authorises
dispatch.

---

## Work Order Lifecycle States

```
CREATED → ASSIGNED → ENROUTE → ONSCENE → RESOLVED → CLOSED
                                              ↑
                                    (partner uploads evidence)
```

| State      | Description                                                               |
|------------|---------------------------------------------------------------------------|
| `CREATED`  | Escalation event generated; work order record created, not yet sent.      |
| `ASSIGNED` | Work order delivered to partner and acknowledgement signature received.    |
| `ENROUTE`  | Partner confirms a response unit is en route to the site.                 |
| `ONSCENE`  | Partner confirms responder has arrived at the site.                       |
| `RESOLVED` | Incident concluded; partner submits resolution summary and evidence URLs. |
| `CLOSED`   | jitterbugs system confirms evidence received and marks record closed.     |

State transitions outside the sequence above are rejected with HTTP 409
`INVALID_STATE_TRANSITION`.

---

## Work Order Schema Summary

See `schemas/work_order.json` for the authoritative JSON Schema.  Key top-level fields:

| Field                  | Type    | Required | Description                                        |
|------------------------|---------|----------|----------------------------------------------------|
| `work_order_id`        | string  | yes      | UUID v4, jitterbugs-assigned                       |
| `event_id`             | string  | yes      | Source escalation event UUID                       |
| `created_at_utc`       | string  | yes      | ISO 8601 creation timestamp                        |
| `priority`             | string  | yes      | `P1` / `P2` / `P3` — response urgency             |
| `required_response_level` | string | yes  | `IMMEDIATE` / `STANDARD` / `ADVISORY`              |
| `location`             | object  | yes      | See Location sub-object below                      |
| `site_contact`         | object  | no       | On-site contact name and phone                     |
| `evidence`             | object  | no       | Presigned clip URL and SHA-256 hash                |
| `sla_deadline_utc`     | string  | yes      | ISO 8601 — partner must reach ONSCENE by this time |
| `server_signature`     | string  | yes      | Server ed25519 signature over canonical JSON       |

### Location sub-object

| Field             | Type   | Required | Description                                      |
|-------------------|--------|----------|--------------------------------------------------|
| `lat`             | number | yes      | WGS-84 latitude                                  |
| `lon`             | number | yes      | WGS-84 longitude                                 |
| `accuracy_m`      | number | yes      | Horizontal accuracy in metres                    |
| `source`          | string | yes      | Location source ID (see DEVICE_LOCATION.md)      |
| `reverse_geocode` | string | no       | Human-readable address (best-effort)             |

---

## Authentication

### Mutual TLS (mTLS)

All partner-facing endpoints require client certificates issued by the jitterbugs
Certificate Authority.  Partner certificates are issued during onboarding and renewed
annually.

```
TLS 1.3 minimum
Cipher suite: TLS_AES_256_GCM_SHA384 preferred
Client cert: X.509 v3, RSA-4096 or ECDSA P-384
Server cert: issued by jitterbugs Intermediate CA
```

### Signed Payloads

Every webhook delivery includes an `X-JB-Signature` HTTP header containing an
ed25519 signature over `SHA-256(request_body)`, Base64url-encoded.  Partners MUST
verify this signature using the jitterbugs webhook public key published at
`GET /v1/pubkeys/webhook`.

```
X-JB-Signature: <Base64url ed25519 sig>
X-JB-Timestamp: <Unix epoch seconds>
X-JB-Nonce:     <32-byte random, Base64url>
```

Partners MUST reject deliveries where `X-JB-Timestamp` is more than 300 seconds old
or where the nonce has been seen before (replay protection).

---

## Acknowledgement Flow

When a work order is delivered via webhook, the partner MUST respond with an
**acknowledgement payload** signed with its own private key:

```json
{
  "work_order_id": "wo_01HV…",
  "partner_id": "acme-security",
  "acknowledged_at_utc": "2026-03-20T12:01:00Z",
  "acceptance": "ACCEPTED",
  "partner_signature": "Base64url-ed25519-sig-over-canonical-ack-body"
}
```

The partner signature is verified server-side.  An unsigned or unverifiable
acknowledgement is treated as a delivery failure and retried.

`acceptance` values:

| Value      | Meaning                                                                |
|------------|------------------------------------------------------------------------|
| `ACCEPTED` | Partner confirms receipt and will respond within SLA.                  |
| `DECLINED` | Partner cannot respond (e.g., capacity issue); jitterbugs re-routes.   |

---

## Presigned Evidence URL Policy

When evidence (video clip) is available, the work order `evidence` object contains:

```json
{
  "clip_url": "https://evidence.jitterbugs.example/…?X-Amz-Signature=…",
  "clip_hash": "sha256:abc123…"
}
```

- URLs are presigned and valid for **60 minutes** from work order delivery.
- Partners MUST verify `clip_hash` (SHA-256 of the downloaded clip) before use.
- URLs may not be shared outside the partner organisation.
- After URL expiry, partners may request a refresh token via
  `POST /v1/work-orders/{id}/evidence/refresh`.

---

## Status Update Examples

Partners push status updates to:

```
POST /v1/work-orders/{work_order_id}/status
Content-Type: application/json
(mTLS client cert required)
```

### ENROUTE update

```json
{
  "work_order_id": "wo_01HV…",
  "state": "ENROUTE",
  "updated_at_utc": "2026-03-20T12:05:00Z",
  "eta_minutes": 12,
  "responder_id": "unit-42"
}
```

### RESOLVED update

```json
{
  "work_order_id": "wo_01HV…",
  "state": "RESOLVED",
  "updated_at_utc": "2026-03-20T12:45:00Z",
  "resolution_summary": "False alarm — motion triggered by HVAC unit.",
  "evidence_urls": [
    "https://partner.example/evidence/wo_01HV_photo1.jpg"
  ]
}
```

---

## Error Handling

| HTTP Status | Error Code                   | Meaning                                          |
|-------------|------------------------------|--------------------------------------------------|
| 400         | `INVALID_PAYLOAD`            | Malformed JSON or missing required field         |
| 401         | `AUTH_FAILED`                | mTLS cert invalid or signature verification fail |
| 404         | `WORK_ORDER_NOT_FOUND`       | Unknown `work_order_id`                          |
| 409         | `INVALID_STATE_TRANSITION`   | State change not permitted from current state    |
| 409         | `DUPLICATE_NONCE`            | Replay detected                                  |
| 410         | `EVIDENCE_URL_EXPIRED`       | Presigned URL no longer valid                    |
| 429         | `RATE_LIMITED`               | Too many requests; back off and retry            |
| 500         | `SERVER_ERROR`               | Internal error; safe to retry with exponential backoff |

Partners MUST implement exponential backoff with jitter for 429 and 500 responses,
starting at 5 s and capping at 300 s.

---

## Example Webhook Request/Response

### Webhook delivery (server → partner)

```bash
curl --cert partner.crt \
     --key  partner.key \
     --cacert jitterbugs-ca.crt \
     -X POST https://partner.example/webhooks/jitterbugs \
     -H "Content-Type: application/json" \
     -H "X-JB-Signature: <Base64url-sig>" \
     -H "X-JB-Timestamp: 1742472000" \
     -H "X-JB-Nonce: <Base64url-nonce>" \
     -d '{
           "work_order_id": "wo_01HV…",
           "event_id": "evt_XYZ…",
           "created_at_utc": "2026-03-20T12:00:00Z",
           "priority": "P1",
           "required_response_level": "IMMEDIATE",
           "location": {
             "lat": 37.7749,
             "lon": -122.4194,
             "accuracy_m": 8.5,
             "source": "AssistedGNSS",
             "reverse_geocode": "Market St, San Francisco, CA"
           },
           "sla_deadline_utc": "2026-03-20T12:20:00Z",
           "server_signature": "<Base64url-server-ed25519-sig>"
         }'
```

### Partner acknowledgement response

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "work_order_id": "wo_01HV…",
  "partner_id": "acme-security",
  "acknowledged_at_utc": "2026-03-20T12:00:03Z",
  "acceptance": "ACCEPTED",
  "partner_signature": "<Base64url-partner-ed25519-sig>"
}
```

### Posting a status update (partner → server)

```bash
curl --cert partner.crt \
     --key  partner.key \
     --cacert jitterbugs-ca.crt \
     -X POST https://api.jitterbugs.example/v1/work-orders/wo_01HV…/status \
     -H "Content-Type: application/json" \
     -d '{"work_order_id":"wo_01HV…","state":"ONSCENE","updated_at_utc":"2026-03-20T12:18:00Z"}'
```

---

## SLA Fields

| Priority | `required_response_level` | Target time to ONSCENE | SLA breach action                              |
|----------|---------------------------|------------------------|------------------------------------------------|
| `P1`     | `IMMEDIATE`               | 20 minutes             | Automatic re-route + operator alert            |
| `P2`     | `STANDARD`                | 60 minutes             | Operator alert                                 |
| `P3`     | `ADVISORY`                | 240 minutes            | Logged only                                    |

`sla_deadline_utc` in the work order is calculated as `created_at_utc + target_time`.
Partners are notified via a `sla_warning` webhook 5 minutes before the deadline if the
work order has not yet reached `ONSCENE`.

SLA performance metrics are reviewed monthly.  Partners consistently missing P1 SLAs
may have their integration suspended pending review.
