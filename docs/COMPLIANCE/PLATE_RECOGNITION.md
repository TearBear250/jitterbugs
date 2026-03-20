# License Plate Recognition (LPR) — Compliance Policy

> **Status:** Draft — requires legal review before production deployment.

---

## Purpose

This document defines the policy, pipeline overview, legal cautions, privacy defaults, evidence-packaging guidance, and operational recommendations for the optional License Plate Recognition (LPR) feature within jitterbugs.

LPR is designed to be **conservative and privacy-forward**. The goal is to enable site operators to detect and record vehicle activity only when lawfully permitted, with strong defaults that require explicit human action before any plate data is shared or cross-referenced.

---

## Key Policy Decisions

| Setting | Default | Notes |
|---|---|---|
| `lpr.enabled_by_default` | **`false`** | LPR is opt-in per site; no plates are processed until a site administrator explicitly enables the feature. |
| `require_human_review_before_lookup` | **`true`** | A human operator must review and approve any external plate lookup (e.g., DMV, law-enforcement databases) before it occurs. Automated lookups are disabled by default. |
| `store_plain_plate_text` | **`false`** | Raw plate strings are not persisted. Only a keyed SHA-256 hash (`sha256(plate_text + site_salt)`) is stored for analytics and de-duplication. |
| `store_plate_hash` | **`true`** | The salted hash is stored to support repeat-sighting alerts and watchlist matching without retaining raw PII. |
| `allow_external_lookup` | **`false`** | All external lookups (vehicle registration, law-enforcement databases) are disabled by default. Enabling requires legal counsel, explicit opt-in, and audit trail configuration. |
| `require_opt_in` | **`true`** | Sites must explicitly opt in before any LPR processing is enabled. |

---

## Pipeline Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│  Camera Frame                                                        │
│         │                                                            │
│         ▼                                                            │
│  [1] Object Detector (YOLO or equivalent)                            │
│      → locates bounding box of licence-plate regions                 │
│         │                                                            │
│         ▼                                                            │
│  [2] Plate Crop + Pre-processing                                     │
│      → crop, grayscale, contrast normalisation                       │
│         │                                                            │
│         ▼                                                            │
│  [3] OCR Engine (EasyOCR or equivalent)                              │
│      → raw text string + confidence score                            │
│         │                                                            │
│         ▼                                                            │
│  [4] Text Normalisation                                              │
│      → strip whitespace, uppercase, remove ambiguous chars           │
│         │                                                            │
│         ▼                                                            │
│  [5] plate_hash = sha256(normalised_plate + SITE_SALT)               │
│      (plain plate text is NOT stored unless policy permits)          │
│         │                                                            │
│         ▼                                                            │
│  [6] plate_event JSON package assembled                              │
│      (see schemas/plate_event.json)                                  │
│         │                                                            │
│         ▼                                                            │
│  [7] Human Review Queue                                              │
│      → operator reviews event; may approve/dismiss                   │
│         │                                                            │
│         ▼                                                            │
│  [8] (Optional, policy-gated) External Lookup                        │
│      → only if human approved AND allow_external_lookup=true          │
└─────────────────────────────────────────────────────────────────────┘
```

**All steps after [6] require explicit policy enablement and human approval.** Steps [7] and [8] are never executed automatically.

---

## Security & Audit

- **Device signing:** every `plate_event` package includes a `signed_by_device` field (device private-key signature over the event payload). See `schemas/plate_event.json` for the full field list.
- **Device manifest reference:** each event references the device manifest (`device_manifest_ref`) so the verifying server can validate hardware identity and software version.
- **Audit log:** every human review action (approve / dismiss / escalate) must be logged with operator identity, timestamp, and the event ID. Audit logs must be retained for the duration defined by `retention_days.lookup_log`.
- **Site salt rotation:** the `site_salt` used for plate hashing must be rotated at the interval specified in `config/policies/plate_policy.yaml`. Rotate salts when site personnel change.
- **Access control:** the LPR configuration endpoint and audit log are accessible only to authenticated administrators.

---

## Legal & Compliance Warnings

> ⚠️ **Read before enabling LPR in any environment.**

1. **Jurisdiction-specific law:** LPR is subject to varying and complex legal frameworks. In many jurisdictions, recording, storing, or querying vehicle plate data requires specific legal authority (e.g., a warrant, court order, or explicit statutory authorisation). **Obtain qualified legal advice for each jurisdiction of deployment before enabling LPR.**
2. **External lookups are high-risk:** Querying DMV, DVLA, law-enforcement, or other third-party databases using plate data may constitute regulated data access. `allow_external_lookup` is `false` by default and must not be enabled without legal review and appropriate data-sharing agreements.
3. **Automated enforcement is not implemented:** jitterbugs does not implement automated enforcement actions (e.g., gate control, alerting law enforcement) from LPR results. Any enforcement action must be taken by a human operator following their own legal obligations.
4. **Retention limits:** plate data (even hashed) may be subject to data-protection regulations (GDPR, CCPA, PIPEDA, etc.). Configure `retention_days` conservatively and honour deletion requests promptly.
5. **CCTV / surveillance disclosure:** deploying LPR-capable cameras may require posting visible notice to the public. Check local signage requirements.
6. **Bias & accuracy:** OCR accuracy varies significantly with lighting, angle, plate style, and vehicle speed. Never rely solely on automated OCR output for enforcement or identification without human verification.

---

## Safe Operational Recommendations

1. **Keep `lpr.enabled_by_default = false`** and require a documented, accountable request from a site administrator before enabling LPR at any location.
2. **Require human review** for every plate event before taking any action. The queue in step [7] of the pipeline is mandatory.
3. **Do not store plain plate text** unless there is a documented legal basis and it is explicitly enabled in policy. Use the salted hash for de-duplication and repeat-sighting rules.
4. **Rotate site salts** regularly and immediately after personnel changes to prevent correlation attacks.
5. **Limit data sharing:** do not share plate hashes or event packages with third parties without a signed Data Processing Agreement (DPA) and legal review.
6. **Conduct periodic audits** of the LPR audit log. Review false-positive rates and escalation patterns.
7. **Test with synthetic data** in CI using the demo script (`tooling/lpr_demo.py`) to verify pipeline outputs without real plate data.
8. **Train operators** on the human review process, legal obligations, and escalation procedures before granting access to the review queue.

---

## Reference

- **Event schema:** [`schemas/plate_event.json`](../../schemas/plate_event.json)
- **Site policy:** [`config/policies/plate_policy.yaml`](../../config/policies/plate_policy.yaml)
- **Demo script:** [`tooling/lpr_demo.py`](../../tooling/lpr_demo.py)
- **Threat model:** [`docs/THREAT_MODEL.md`](../THREAT_MODEL.md)
- **Safety & scope policy:** [`docs/SAFETY-SCOPE-POLICY.md`](../SAFETY-SCOPE-POLICY.md)
