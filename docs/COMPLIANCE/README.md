# Compliance Documentation

This directory contains compliance, privacy, and partner-integration documentation for
the jitterbugs platform.

## Files in this directory

| File | Description |
|------|-------------|
| [DEVICE_LOCATION.md](DEVICE_LOCATION.md) | Device location attestation policy — how location is sourced, signed, validated, and shared. |
| [WORK_ORDER_API.md](WORK_ORDER_API.md) | Work-order lifecycle and webhook/API specification for security partners. |
| [TERMINAL_BOX_HW_REQS.md](TERMINAL_BOX_HW_REQS.md) | Hardware requirements for jitterbugs terminal boxes (GNSS, SE/TPM, IMU, power, fallback). |

Related artefacts in the repository root:

| Path | Description |
|------|-------------|
| `schemas/device_manifest.json` | JSON Schema for the signed device manifest. |
| `schemas/work_order.json` | JSON Schema for work orders dispatched to partners. |
| `compliance/verify_location_signature.py` | Python utility for verifying device manifest signatures (ed25519). |

---

## Default Policy Choices (this PR)

The following defaults are in effect as of this PR and apply to all sites unless
explicitly overridden by a site-level policy document:

### 1. Coarse location by default (precise is opt-in)

Terminal boxes report **coarse location** (latitude and longitude truncated to two
decimal places, approximately 1 km resolution) by default.  Precise location (full
GNSS accuracy) is only transmitted if a site operator has explicitly enabled the
`precise_location` flag in the site policy and accepted the associated data-handling
addendum.

_Rationale_: Minimises privacy exposure for all users while still providing enough
geographic information for dispatch routing and compliance reporting.

### 2. Law-enforcement escalation requires human confirmation

When an escalation event triggers a potential law-enforcement notification, the system
requires a human operator to click "Confirm escalation" in the operations dashboard
before precise device location is transmitted to law enforcement.

Sites that have signed the **auto-escalation addendum** may set `auto_escalation: true`
in their site policy to allow automated escalation without per-event confirmation.
This setting requires annual renewal and is logged in the audit trail.

_Rationale_: Reduces risk of false-positive law-enforcement dispatches and ensures
accountability for every escalation decision.

---

## Overriding Defaults

Site-level policy overrides are managed by site operators in the jitterbugs management
portal.  Available overrides:

| Policy key              | Default | Allowed values | Description |
|-------------------------|---------|----------------|-------------|
| `precise_location`      | `false` | `true` / `false` | Enable full-resolution location reporting. |
| `auto_escalation`       | `false` | `true` / `false` | Enable automated law-enforcement escalation (requires signed addendum). |
| `location_retention_days` | `7`  | `7`–`90`       | How long location fixes are retained. |

---

## Legal and Jurisdictional Cautions

> **Important**: Sharing precise device location and escalating to law enforcement are
> regulated activities in many jurisdictions.  Before enabling precise location or
> auto-escalation in a site policy, operators **must**:
>
> 1. Obtain explicit, informed consent from all individuals whose location may be
>    captured or reported.
> 2. Consult legal counsel regarding applicable privacy, surveillance, and
>    emergency-dispatch laws in the relevant jurisdiction (e.g., GDPR, CCPA, ECPA,
>    state wiretapping statutes).
> 3. Ensure security partners hold required certifications (ISO/IEC 27001 or SOC 2
>    Type II) and applicable regulatory approvals before sharing precise location data.
> 4. Renew auto-escalation authorisation annually and maintain auditable records of
>    all escalation decisions.

---

## How to Test

### Verify a device manifest signature

```bash
pip install cryptography
python compliance/verify_location_signature.py manifest.json device_pubkey.pem
```

Run the built-in self-test (field validation only, no real key required):

```bash
python compliance/verify_location_signature.py
```

### Simulate a work-order webhook with curl and mTLS

```bash
# Replace paths and values with real certificate files and a live server
curl --cert partner.crt \
     --key  partner.key \
     --cacert jitterbugs-ca.crt \
     -X POST https://partner.example/webhooks/jitterbugs \
     -H "Content-Type: application/json" \
     -H "X-JB-Signature: <Base64url-sig>" \
     -H "X-JB-Timestamp: $(date +%s)" \
     -H "X-JB-Nonce: $(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=')" \
     -d @/tmp/sample_work_order.json
```

### Partner acknowledgement of a work order

POST the acknowledgement JSON (see WORK_ORDER_API.md §Acknowledgement Flow) to the
webhook endpoint with the partner's signed payload.  The server will verify the partner
signature before marking the work order as `ASSIGNED`.
