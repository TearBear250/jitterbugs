# Terminal Box Hardware Requirements

## Overview

This document specifies the minimum hardware capabilities required for a jitterbugs
terminal box to participate in device location attestation, event detection, and
partner work-order dispatch.

These requirements are designed to ensure that every terminal box can produce a
cryptographically verifiable, tamper-evident location manifest with sufficient accuracy
for compliance and dispatch purposes.

---

## GNSS Receiver

| Requirement              | Specification                                               |
|--------------------------|-------------------------------------------------------------|
| Constellations           | GPS + GLONASS minimum; Galileo and BeiDou recommended       |
| Horizontal accuracy CEP  | ≤ 5 m with clear sky view                                   |
| Cold start TTFF          | ≤ 60 s                                                      |
| Hot start TTFF           | ≤ 5 s                                                       |
| A-GPS support            | Required; assists with indoor / urban-canyon operation      |
| Update rate              | 1 Hz minimum                                                |
| Antenna                  | External patch antenna preferred; internal acceptable if ≥ 26 dBi gain |
| Antenna supervision      | Continuity monitoring with tamper-flag bit 3 (see DEVICE_LOCATION.md) |

**Antenna placement guidance**: Mount the antenna with clear 90° sky view above
horizon.  Avoid placement near metallic surfaces, HVAC vents, or other RF-emitting
equipment.  Use low-loss coaxial cable (< 2 dB insertion loss at 1575 MHz) if routing
to a remote antenna.

---

## Secure Element / TPM

| Requirement              | Specification                                               |
|--------------------------|-------------------------------------------------------------|
| Standard                 | TCG TPM 2.0 or equivalent CC EAL4+ certified SE            |
| Key algorithms           | ed25519 for signing, AES-256-GCM for storage encryption     |
| Key operations           | Private key never exposed outside the SE boundary           |
| Attestation              | Remote attestation (TPM quote or SE-specific equivalent)    |
| Firmware update          | Authenticated, signed update only                           |
| Physical security        | Epoxy-potted or equivalent tamper-evident encapsulation     |

The device private key MUST be generated inside the SE at manufacture and bound to the
device PCB (hardware binding).  Factory provisioning MUST record the public key
fingerprint in the jitterbugs device registry before the unit leaves the factory.

---

## IMU and Tamper Sensors

| Sensor                   | Requirement                                                 |
|--------------------------|-------------------------------------------------------------|
| Accelerometer            | 3-axis, ≥ 100 Hz sample rate, range ±16 g                  |
| Gyroscope                | 3-axis, ≥ 100 Hz sample rate, range ±2000 °/s              |
| Enclosure tamper switch  | Magnetic or mechanical reed switch; triggers bit 0 of `tamper_flags` |
| Vibration threshold      | Configurable; default 2 g peak triggers bit 2              |
| Orientation watchdog     | Alerts when device tilt exceeds 15° from installation baseline |

IMU data is processed on-device; raw samples are not transmitted.  The device firmware
evaluates the `tamper_flags` bitmask before each manifest transmission.

---

## Secure Time

| Requirement              | Specification                                               |
|--------------------------|-------------------------------------------------------------|
| NTP synchronisation      | At boot and every 15 minutes; stratum 2 or better          |
| PTP (IEEE 1588)          | Supported if network provides PTP grandmaster               |
| Monotonic clock          | SE-enforced; software cannot roll back clock                |
| RTC battery backup       | Minimum 5-year battery life; ±5 ppm accuracy at 25 °C      |
| Maximum drift            | ≤ 5 s/day without NTP; manifests outside ±30 s are rejected |

---

## Power Supervision

| Requirement              | Specification                                               |
|--------------------------|-------------------------------------------------------------|
| Input voltage range      | 100–240 V AC, 50/60 Hz                                     |
| Power-fail detection     | Hardware voltage supervisor triggers alert within 100 ms   |
| Backup power             | UPS or supercapacitor bridge for ≥ 5 minutes at idle power |
| Graceful shutdown        | Flush pending manifests and sign a shutdown manifest before power loss |
| Tamper on power removal  | Unexpected power removal logged as a tamper event           |

---

## Fallback Location Methods

The device MUST implement at least one fallback location method when GNSS is
unavailable (e.g., indoor deployment, GNSS jamming, antenna failure):

### Wi-Fi Positioning

| Requirement              | Specification                                               |
|--------------------------|-------------------------------------------------------------|
| Wi-Fi module             | 802.11a/b/g/n/ac (2.4 GHz + 5 GHz)                        |
| Scanning                 | Passive BSSID scan; no association required                 |
| Positioning engine       | OS-level or third-party Wi-Fi positioning API               |
| Typical accuracy         | 10–50 m in urban environments                               |

### Cellular Positioning

| Requirement              | Specification                                               |
|--------------------------|-------------------------------------------------------------|
| Cellular module          | LTE Cat-M1 / NB-IoT minimum; LTE Cat-1 recommended         |
| Positioning              | Cell-ID + RSSI; E-CID if operator supports it              |
| Typical accuracy         | 100–5 000 m                                                 |

Fallback source is recorded in `location.source` (`WiFi` or `Cell`) and `accuracy_m`
is set to the estimated position uncertainty for that source.

---

## Recommended Accuracy Summary

| Scenario                         | Source          | Expected `accuracy_m` |
|----------------------------------|-----------------|-----------------------|
| Clear sky, GNSS fix              | AssistedGNSS    | 3–10 m               |
| Urban canyon, A-GPS              | AssistedGNSS    | 5–20 m               |
| Indoor, strong Wi-Fi             | WiFi            | 10–50 m              |
| Indoor, cellular only            | Cell            | 100–2 000 m          |
| Manual (site config)             | Manual          | Site-defined         |

A device that cannot obtain a fix with `accuracy_m ≤ 5000` MUST omit the `location`
object and set `location_unavailable: true` in the manifest.

---

## Environmental and Certification

| Requirement              | Specification                                               |
|--------------------------|-------------------------------------------------------------|
| Operating temperature    | –10 °C to +55 °C                                           |
| Storage temperature      | –20 °C to +70 °C                                           |
| Ingress protection       | IP54 minimum for indoor; IP65 for outdoor deployment        |
| Shock / vibration        | IEC 60068-2-27 / IEC 60068-2-6                             |
| EMC                      | FCC Part 15 Class B (US); CE Mark / EN 55032 Class B (EU)  |
| Safety                   | UL 60950-1 or IEC 62368-1                                   |
