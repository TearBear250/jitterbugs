# Data Flows

This document describes every category of data that Jitterbugs handles, where it is stored, how long it is retained, and the conditions under which it may leave the device.

> **Privacy-by-default:** No data leaves the device unless the operator takes an explicit action or enables an explicit opt-in feature. There is no background telemetry, no cloud sync, and no third-party data sharing in the default configuration.

---

## 1. Data Types and Storage

| Data type | Description | Where stored | Encryption |
|---|---|---|---|
| Video footage | Raw camera streams recorded to disk | Local video storage volume (NVR/terminal) | AES-256 at rest (AES-128 compatibility fallback) |
| Motion / alarm events | Timestamped records of detected motion or alarm triggers | Local database on NVR/terminal | Same volume encryption as above |
| Audit logs | System events: auth, recording start/stop, exports, config changes | Separate local log volume or log directory | Volume-level encryption |
| System configuration | Camera settings, user accounts, retention policy, network config | Local filesystem | Volume-level encryption |
| Evidence packages | Exported incident packages (ZIP with manifest) | Local first; written to USB/microSD on export | AES-256 when encrypted for transit |
| System health snapshots | Snapshot of system state at time of evidence export | Included in evidence package; not separately stored | N/A (part of export) |

---

## 2. Retention

| Data type | Default retention | Notes |
|---|---|---|
| Video footage | Operator-configured (suggested default: 30 days) | Overwriting is blocked for footage flagged as incident-relevant |
| Motion / alarm events | Same as associated video | Deleted when associated video is deleted |
| Audit logs | Minimum 90 days (recommended); never shorter than video retention | Do not delete logs while associated footage is retained |
| System configuration | Retained indefinitely (no automatic expiry) | Backed up at operator discretion |
| Evidence packages (on USB/microSD) | Operator responsibility | Retain until investigation resolved; consult legal counsel |

---

## 3. Conditions for Data Leaving the Device

Data leaves the device **only** under the following conditions, all of which require explicit operator action:

| Condition | Trigger | Data involved | Notes |
|---|---|---|---|
| Evidence export to USB/microSD | Operator initiates export via UI | Evidence package (video clips, timeline, audit excerpt, system health, manifest) | Primary evidence handoff mechanism; see [`docs/EVIDENCE_PACKAGE.md`](EVIDENCE_PACKAGE.md) |
| Remote access via VPN | Operator enables and connects VPN | Live view / UI interaction only; no bulk data transfer by default | Requires explicit VPN configuration; see [`docs/SECURITY_ARCHITECTURE.md`](SECURITY_ARCHITECTURE.md) §5.2 |
| Remote access via Tor onion service | Operator explicitly enables Tor (advanced, opt-in) | Live view / UI interaction only | Advanced option; see [`docs/SECURITY_ARCHITECTURE.md`](SECURITY_ARCHITECTURE.md) §6 |
| Manual network share | Operator explicitly attaches/sends package file | Evidence package or specific clip | No automatic upload; operator must initiate each transfer |
| OS/firmware updates | Operator initiates or approves update | Software package download only; no user data transmitted | Outbound only; no footage or logs uploaded |
| NTP time sync | Automatic (system function) | Time query only; no footage or personal data | Outbound UDP/123 only |

---

## 4. What Never Leaves the Device

The following data is **never** transmitted, uploaded, or shared without explicit operator action:

- Raw video footage (no background cloud sync)
- Motion/alarm events
- Audit logs
- System configuration including credentials
- Operator identity or location information
- Any analytics, telemetry, or usage statistics

There is no built-in telemetry, analytics, or "phone home" capability.

---

## 5. Third-Party Data Sharing

| Scenario | Default | Opt-in available |
|---|---|---|
| Cloud backup of footage | Disabled | No built-in cloud backup; operator may implement manually |
| Analytics / telemetry to developer | Disabled | No |
| Law enforcement remote access | Disabled (never built-in) | No — evidence is provided via export package, not remote access |
| Third-party monitoring services | Disabled | No built-in integration |

---

## 6. Network Traffic Summary

| Direction | Purpose | Protocol/Port | Default |
|---|---|---|---|
| Outbound | NTP time sync | UDP/123 | Enabled |
| Outbound | OS/firmware updates | HTTPS/443 | Enabled (update process) |
| Outbound | VPN tunnel (if enabled) | WireGuard UDP or OpenVPN UDP/TCP | Opt-in |
| Outbound | Tor (if enabled) | TCP/9050 (to Tor guard nodes) | Opt-in |
| Inbound | None | — | No inbound ports open by default |

---

## 7. Consistency with Project Principles

This data-flow design implements the privacy-by-default and local-first principles stated in:
- [`docs/SAFETY-SCOPE-POLICY.md`](SAFETY-SCOPE-POLICY.md)
- [`docs/OCEAN_PROTOCOL.md`](OCEAN_PROTOCOL.md)
- [`docs/SECURITY_ARCHITECTURE.md`](SECURITY_ARCHITECTURE.md)
