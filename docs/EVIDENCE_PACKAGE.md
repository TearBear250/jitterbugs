# Evidence Package Specification

> **Not legal advice.** This document defines a technical standard for evidence export. Consult qualified legal counsel and your local law enforcement for chain-of-custody requirements that apply in your jurisdiction.

---

## 1. Purpose

The evidence package ("one-shot police handoff") is a self-contained export that a Jitterbugs operator can provide to law enforcement, legal counsel, or an insurer to support an incident investigation.

The package is created **locally** on the NVR/terminal and written to a portable medium (USB drive, microSD card). No data is transmitted to any third party during package creation. Network sharing requires explicit operator opt-in.

---

## 2. Package Contents

| Item | Description | Required |
|---|---|---|
| `incident_summary.txt` | Plain-text factual summary using the template in [`docs/OSINT_INCIDENT_RESPONSE.md`](OSINT_INCIDENT_RESPONSE.md) | Yes |
| `timeline.csv` | Machine-readable timeline of events (see §4) | Yes |
| `video/` directory | Original, unmodified video clips covering the incident window | Yes |
| `audit_log_excerpt.txt` | Timestamped system audit log entries covering the incident window plus 1 hour before and after | Yes |
| `system_health.json` | Snapshot of system state at time of export (see §5) | Yes |
| `manifest.sha256` | SHA-256 checksums of all files in the package | Yes |
| `manifest.sha256.sig` | RSA-4096 signature over `manifest.sha256` (operator key) | Recommended |
| `README_CHAIN_OF_CUSTODY.txt` | Human-readable chain-of-custody record (see §6) | Yes |

---

## 3. Package Naming Convention

```
jitterbugs_evidence_<SITE_ID>_<YYYYMMDD>_<HHMMSS>_<UTC_OFFSET>.zip
```

Example:
```
jitterbugs_evidence_MAIN_20260310_143000_UTC+0000.zip
```

Where:
- `SITE_ID` is the operator-configured site identifier (alphanumeric, no spaces)
- `YYYYMMDD` and `HHMMSS` are the export timestamp in local time
- `UTC_OFFSET` is the UTC offset at the time of export (e.g., `UTC+0000`, `UTC-0500`)

The package must also include a file `package_timestamp_utc.txt` containing the export time in ISO 8601 UTC format (e.g., `2026-03-10T14:30:00Z`).

---

## 4. Timeline File Format (`timeline.csv`)

```csv
timestamp_utc,camera_id,event_type,description
2026-03-10T14:00:00Z,CAM-01,motion_start,"Motion detected in zone A"
2026-03-10T14:00:45Z,CAM-01,motion_end,"Motion ended in zone A"
2026-03-10T14:01:10Z,CAM-02,motion_start,"Motion detected at rear entrance"
```

- Timestamps must be in ISO 8601 UTC format.
- `event_type` values: `motion_start`, `motion_end`, `recording_start`, `recording_end`, `alarm`, `operator_note`, `system_event`.
- `description` must contain only factual, observable information. No identity claims or inferences.

---

## 5. System Health Snapshot (`system_health.json`)

```json
{
  "export_timestamp_utc": "2026-03-10T14:30:00Z",
  "system_id": "jitterbugs-MAIN",
  "firmware_version": "x.y.z",
  "time_source": "pool.ntp.org",
  "time_sync_status": "synced",
  "time_offset_ms": 12,
  "storage_volumes": [
    {
      "label": "video_storage",
      "encryption": "aes-256",
      "free_bytes": 107374182400,
      "total_bytes": 1099511627776
    }
  ],
  "cameras": [
    {
      "id": "CAM-01",
      "status": "online",
      "resolution": "1920x1080",
      "fps": 15
    }
  ],
  "audit_log_integrity": "verified"
}
```

---

## 6. Chain-of-Custody Record (`README_CHAIN_OF_CUSTODY.txt`)

```
CHAIN OF CUSTODY — Jitterbugs Evidence Package
===============================================

Package filename  : jitterbugs_evidence_MAIN_20260310_143000_UTC+0000.zip
SHA-256 (package) : <hash of the zip file itself>
Export timestamp  : 2026-03-10T14:30:00Z (UTC)
Exported by       : <operator name / role>
Export medium     : USB drive (serial: ____________)
Seal / label      : <describe physical labelling if any>

Transfer log:
  Date       | Transferred by    | Transferred to          | Method
  -----------|-------------------|-------------------------|--------
  2026-03-10 | <operator>        | <law enforcement agency> | In person
  ...

Notes:
  - Original footage on NVR has NOT been deleted.
  - No edits have been made to any video file in this package.
  - This package was created entirely on the local NVR/terminal.
  - No data was uploaded to external servers during export.
```

---

## 7. Integrity Protection

### SHA-256 Manifest

The `manifest.sha256` file must contain one line per file in the package in the format:

```
<sha256hex>  <relative_path>
```

Example:
```
a3f1...  incident_summary.txt
7b2d...  timeline.csv
c4e9...  video/clip_001.mp4
...
```

Generate with:
```bash
find . -type f ! -name 'manifest.sha256' ! -name 'manifest.sha256.sig' \
  | sort | xargs sha256sum > manifest.sha256
```

### RSA-4096 Signature (Recommended)

Sign `manifest.sha256` with the operator's RSA-4096 private key:

```bash
openssl dgst -sha256 -sign operator_private.pem \
  -out manifest.sha256.sig manifest.sha256
```

The corresponding public key (`operator_public.pem`) should be provided separately to the recipient (law enforcement, legal counsel) to allow verification:

```bash
openssl dgst -sha256 -verify operator_public.pem \
  -signature manifest.sha256.sig manifest.sha256
```

The operator private key must be stored securely (not on the NVR storage volume itself). A hardware security module (HSM), smart card, or offline key store is recommended.

---

## 8. Encryption for Transit (Optional)

If the evidence package must be transmitted electronically (e.g., to legal counsel), encrypt it before transmission:

```bash
gpg --recipient <recipient_key_id> --encrypt \
  jitterbugs_evidence_MAIN_20260310_143000_UTC+0000.zip
```

The passphrase or recipient key must be communicated out-of-band (e.g., by phone).

---

## 9. Privacy-by-Default Constraints

- Package creation is entirely local. No data leaves the device during the export process.
- Network sharing of the package requires **explicit operator action** (e.g., manually attaching it to an email or using the export-over-network feature if available).
- The package must not be uploaded to any cloud service, public URL, or third-party storage without explicit operator decision and appropriate legal basis.

---

## 10. What to Do After Export

1. Label the storage medium (USB/microSD) with the case reference and your name.
2. Keep your own copy on a separate medium.
3. Provide the package to law enforcement or legal counsel upon request.
4. Do **not** post footage or package contents publicly during an active investigation.
5. Retain originals on the NVR until the investigation is resolved and you receive written confirmation that you may delete them.

See [`docs/OSINT_INCIDENT_RESPONSE.md`](OSINT_INCIDENT_RESPONSE.md) for the full incident-response workflow.
