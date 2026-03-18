# Security Architecture

> **Not legal advice.** This document describes the security design principles and recommended configuration for the Jitterbugs NVR/terminal. It does not constitute legal advice.

---

## 1. Design Principles

| Principle | Description |
|---|---|
| Local-first | All video, events, and logs are stored on the local device by default. No data leaves the device without explicit operator action. |
| Privacy-by-default | No third-party data sharing, analytics, or telemetry by default. |
| Operator control | The operator (owner) has sole control over what is recorded, retained, and shared. |
| Minimal attack surface | Remote access is opt-in; only the minimum necessary ports and services are exposed. |
| Defense in depth | Encryption at rest, strong authentication, network isolation, and audit logging are layered controls. |

---

## 2. Storage Encryption

### 2.1 At-Rest Encryption Requirements

| Volume | Required | Fallback |
|---|---|---|
| OS / system drive | AES-256 | — |
| Video storage volume | AES-256 | AES-128 (compatibility fallback only, where hardware or throughput constraints prevent AES-256) |
| Evidence export package | AES-256 (when encrypted for transit) | — |

AES-128 is permitted **only** as a compatibility fallback. AES-256 is the default and recommended configuration. Non-standard or non-AES symmetric ciphers are not permitted for at-rest encryption.

### 2.2 Key Management

- Prefer TPM-backed key storage where hardware supports it.
- If no TPM is available, use a strong operator passphrase (minimum 16 characters, randomly generated).
- Store recovery keys offline and separately from the device.
- Do not store encryption keys on the same volume they protect.

---

## 3. Evidence Export Integrity

| Mechanism | Algorithm |
|---|---|
| File integrity hashing | SHA-256 |
| Manifest signing (default recommendation) | RSA-4096 |
| Package encryption (optional, for transit) | AES-256 (GPG or equivalent) |

See [`docs/EVIDENCE_PACKAGE.md`](EVIDENCE_PACKAGE.md) for the full specification.

---

## 4. Authentication and Access Control

- All administrative interfaces (web UI, API, SSH) must be protected by strong passwords and two-factor authentication (2FA).
- Default credentials must be changed at first boot. The system must not operate with factory-default credentials.
- Use role-based access where possible: separate accounts for viewing, export, and administration.
- Audit all authentication events (success, failure, lockout) to the append-only audit log.
- Lock out accounts after repeated failed authentication attempts.

---

## 5. Network Architecture

### 5.1 Recommended Network Isolation

```
Internet
   │
[Router/Firewall]
   ├── [Main LAN]  (user devices)
   └── [NVR VLAN]  (Jitterbugs terminal + cameras) ◄── isolated
```

- Place the NVR/terminal and cameras on a dedicated VLAN or network segment, isolated from general user traffic.
- Block outbound internet access from the NVR VLAN except for:
  - NTP (time sync, UDP/123)
  - OS/firmware update sources (explicit allowlist)
  - VPN relay (if remote access is enabled)

### 5.2 VPN-First Remote Access (Default)

Remote access to the Jitterbugs terminal should use a VPN tunnel as the default method:

- WireGuard or OpenVPN are recommended.
- The terminal initiates an outbound connection to an operator-controlled relay, eliminating the need for inbound port forwarding.
- The relay must be operated by the owner or a trusted provider; do not use free/anonymous VPN services as a relay for the evidence system.
- All remote sessions must authenticate with 2FA.

### 5.3 Firewall Posture

- No inbound ports open by default.
- Outbound-only policy from the NVR segment (allowlist as above).
- Log all blocked traffic for audit.

---

## 6. Optional: Tor Onion Service (Advanced, Opt-In)

> **This section describes an advanced, opt-in configuration. It is not the default, not recommended for most deployments, and requires careful setup. Read the safety constraints before enabling.**

### 6.1 Use Case

Tor onion routing may be considered as an **alternative transport** for remote access in scenarios where:

- The site is behind CGNAT or a carrier-grade NAT that prevents inbound connections.
- No static IP or reliable DNS is available.
- The operator cannot or does not wish to maintain a relay server for the VPN-first approach.

Tor does **not** replace authentication, 2FA, or other security controls. It is a transport option only.

### 6.2 How It Works (Conceptual)

A Tor hidden service (v3 onion service) creates an outbound circuit from the terminal to the Tor network. The operator connects to the `.onion` address using a Tor-capable browser or tool, reaching the terminal's web UI without requiring an inbound port or public IP.

### 6.3 Safety Constraints

All of the following constraints apply if Tor is enabled:

| Constraint | Detail |
|---|---|
| Opt-in only | Tor must be explicitly enabled by the operator. It must not be on by default. |
| Authentication required | The terminal's web UI must still require strong password + 2FA even over Tor. Rate limiting and lockout must be active. |
| No direct camera exposure | Individual cameras must not be accessible directly over Tor; only the terminal's web UI/API should be reachable. |
| No preconfigured onion address | The operator generates the onion address at setup. No onion address is pre-configured or shipped with the software. |
| Not a bypass mechanism | Tor must not be framed or used as a way to bypass owner oversight, law enforcement process, or any other control. The operator remains fully accountable for the system. |
| Audit logging | All onion-service connections must be logged in the audit log, including timestamps. |
| No anonymity claims | Using a Tor onion service does not make the operator anonymous; it is a transport layer only. Physical device location and identity may still be discoverable through other means. |

### 6.4 Setup Guidance (Docs-Only)

> This is documentation guidance only. No Tor code or configuration is shipped in the default build.

1. Install Tor on the NVR/terminal host OS.
2. Configure a v3 hidden service pointing to the terminal's local web UI port (e.g., `127.0.0.1:8080`).
3. Note the generated `.onion` address and store it securely.
4. Access the terminal using Tor Browser (or `torsocks` + a terminal client) from a remote location.
5. All authentication (password + 2FA) remains in effect.
6. Audit the Tor service configuration for any unintended exposure.

Reference: [Tor Project — Set up Your Onion Service](https://community.torproject.org/onion-services/setup/)

### 6.5 When Not to Use Tor for This Purpose

- Do not use Tor if a well-configured VPN relay is available; WireGuard or OpenVPN is simpler, faster, and easier to audit.
- Do not enable Tor as a workaround for weak authentication or an insecure configuration elsewhere.
- Do not use Tor to provide access to third parties (including law enforcement) to your surveillance system. Evidence should be exported as a package and provided through proper channels. See [`docs/EVIDENCE_PACKAGE.md`](EVIDENCE_PACKAGE.md).

---

## 7. Audit Logging

- Maintain an append-only audit log of: authentication events, recording start/stop, export events, configuration changes, user account changes, and remote access sessions.
- Audit logs must be stored separately from video storage where possible.
- Include log integrity verification (hash chain or signed log entries) to detect tampering.
- Retain audit logs for at least as long as the associated video footage.

---

## 8. Software Updates

- Apply OS and firmware security updates promptly.
- Pin specific package versions for the NVR application and verify signatures before installing.
- Do not disable automatic security updates unless a specific operational reason requires manual control; if disabled, define a manual update schedule.

---

## 9. Physical Security

- Physically secure the NVR/terminal (locked cabinet, locked room) to prevent direct-access attacks.
- Use full-disk encryption with a strong passphrase or TPM binding so that a stolen drive reveals nothing.
- Label storage media appropriately and track physical inventory.

---

## 10. Consistency with Project Principles

This architecture is consistent with the Jitterbugs project's:
- Local-first, no-cloud-by-default design (see [`docs/DATA_FLOWS.md`](DATA_FLOWS.md))
- No-profiling stance (see [`docs/SAFETY-SCOPE-POLICY.md`](SAFETY-SCOPE-POLICY.md))
- GPLv3 license and open, auditable codebase
