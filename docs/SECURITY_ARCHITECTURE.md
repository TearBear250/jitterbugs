# Security Architecture: VPN-Only / No Inbound Ports

This document describes the recommended network and access-control posture for a
Jitterbugs deployment.  The goal is **owner-only access**: only the property owner
(or an authorized administrator) can view footage or change settings.

---

## Guiding Principles

1. **No inbound ports.**  The home or business router exposes no forwarded ports to
   cameras, NVR/terminal, or management interfaces.
2. **No UPnP.**  UPnP is disabled on all routers and camera devices to prevent
   automatic and uncontrolled port exposure.
3. **Encrypted remote access via VPN only.**  All remote viewing and administration
   traffic travels through an authenticated, encrypted VPN tunnel.
4. **Least-privilege network isolation.**  Cameras are segmented from general-purpose
   LAN traffic using a dedicated VLAN or subnet.
5. **Audit everything.**  All logins, configuration changes, and footage exports are
   logged with timestamps.

---

## Recommended Network Layout

```
Internet
    │
    ▼
[Router / Firewall]
 ├── LAN (192.168.1.0/24)     – owner devices (phones, laptops)
 └── Camera VLAN (10.10.0.0/24) – IP cameras, PoE switch
         │
         ▼
   [Terminal / NVR]
    ├── records camera streams
    ├── runs VPN server (outbound-tunnel model)
    └── exposes NVR UI/API only on LAN + VPN interfaces
```

### Firewall Rules (key defaults)

| Direction | Source | Destination | Action |
|---|---|---|---|
| Inbound (WAN → LAN) | Any | Any | **DROP** (default deny) |
| Camera VLAN → Internet | Camera VLAN | Any | **DROP** |
| Camera VLAN → NVR | Camera VLAN | NVR IP | ALLOW (RTSP/HTTP) |
| LAN → NVR UI | LAN | NVR port 8080 | ALLOW |
| VPN clients → NVR UI | VPN subnet | NVR port 8080 | ALLOW |

---

## Remote Access: Outbound Tunnel / Relay (Option B — No Inbound Ports)

To support remote viewing without opening inbound ports:

1. The **Terminal/NVR** establishes an **outbound** OpenVPN connection to a relay server
   (self-hosted VPS or managed relay) at startup.
2. The **owner's device** (phone, laptop) also connects to the same relay with a
   per-device client certificate.
3. The relay routes owner ↔ terminal traffic over the encrypted tunnel.
4. The home router remains **fully closed to unsolicited inbound connections**.

```
Owner device ──OpenVPN──► Relay ◄──OpenVPN── Terminal/NVR
                               │
                   (relay forwards authorized traffic only)
```

### OpenVPN Configuration Recommendations

- **Certificate-based authentication** — one client certificate per owner device; no
  shared-password auth.
- **TLS 1.2 / 1.3 only** with strong cipher suites (e.g., `AES-256-GCM`).
- **Certificate Revocation List (CRL)** — revoke a lost or stolen device certificate
  immediately.
- Bind NVR UI to **VPN and LAN interfaces only** (never to the WAN/internet interface).
- Enable `tls-auth` or `tls-crypt` to protect the OpenVPN handshake.

---

## VLAN Separation for Camera Networks

Cameras should reside on a dedicated VLAN or subnet, isolated from the general LAN:

- **Why:** Compromised camera firmware cannot reach owner laptops/phones, NAS drives,
  or other home/office devices.
- **How:** Configure a managed switch with a tagged VLAN for the camera ports; configure
  the router/firewall to apply the Camera VLAN rules described above.
- **NVR access:** The Terminal/NVR may have a trunk port (or dual NICs) bridging the
  Camera VLAN and the LAN—but it enforces application-level authentication before
  forwarding any stream to a remote client.

---

## Owner-Only Access Model

| Layer | Mechanism |
|---|---|
| Network | VPN tunnel; cameras unreachable from WAN |
| VPN | Per-device client certificate; CRL for revocation |
| Application | NVR login + optional 2FA |
| Audit | Login events, config changes, and exports logged |

- **No shared credentials.**  Each authorized device receives its own certificate/key.
- **Break-glass recovery.**  Admins should store a revocation-capable backup credential
  offline (e.g., encrypted USB) in case the primary device is lost.

---

## Privacy and Ethical Use

- Jitterbugs is designed for **visible, consent-aware** home and small-business
  surveillance (posted signage where required by local law).
- The software does not include facial recognition, identity matching, or behavioral
  profiling features.
- Footage is stored locally; there are no default cloud uploads.
- Audio recording is disabled by default.

---

## Out of Scope

This document covers network/access architecture only.  For vulnerability reporting, see
[SECURITY.md](../SECURITY.md).  For data-handling and ethical-use policy, see
[docs/SAFETY-SCOPE-POLICY.md](SAFETY-SCOPE-POLICY.md).
