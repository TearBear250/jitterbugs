# OCEAN Protocol — Operator Standard for Home & Small-Business Surveillance

> **Not legal advice.** This document provides general guidance only. Consult qualified legal counsel and your local law enforcement agency for requirements that apply in your jurisdiction.

---

## 1. What OCEAN Means Here

OCEAN is an **operator standard** for responsible, lawful use of the Jitterbugs NVR/terminal in home and small-business environments (convenience stores, pharmacies, restaurants, bars, residential premises).

It covers:

- Lawful use principles
- Privacy zones and signage guidance
- Retention policy guidance for businesses
- Security baseline requirements
- Prohibited uses

OCEAN is **not** a personality or psychological profiling framework. It does not reference personality models or behavioral scoring.

---

## 2. Scope

| Deployment type | Examples |
|---|---|
| Home | Single-family residence, apartment |
| Small business | Convenience store, pharmacy, pizza place, bar, small office |
| Out of scope | Public-area surveillance at scale, law-enforcement infrastructure, facial recognition deployments |

---

## 3. Lawful Use Principles

1. **Record only what you are permitted to record.** Verify that your installation complies with local video surveillance and audio recording laws. Audio recording is often subject to stricter rules than video; when in doubt, disable audio or consult legal counsel.
2. **Surveillance must serve a defined, legitimate purpose.** Acceptable purposes include property protection, employee and customer safety, and evidence collection for incidents that occur on your premises.
3. **Do not use footage for purposes beyond those stated.** Repurposing footage for identity matching, behavioral profiling, or any use unrelated to the original purpose is prohibited under this standard.
4. **Comply with access requests.** Where law requires (e.g., subject-access requests under GDPR or similar frameworks), respond within the required time. Consult legal counsel.

---

## 4. Privacy Zones

- Configure privacy masking for areas where recording is not appropriate (e.g., restrooms, changing areas, neighbouring properties visible through windows).
- Review camera placement at installation and after any renovation or camera adjustment.
- Do not point cameras at public streets or sidewalks in a manner that systematically tracks individuals beyond your premises, unless local law permits and requires it.

---

## 5. Signage Guidance (Businesses)

Posting visible notice of video surveillance is required by law in many jurisdictions. General guidance (verify locally):

- Place signs at all entry points to the monitored area.
- Signs should state (at minimum): that video recording is in operation, and a contact point for enquiries.
- Retain a record of sign placement in your site documentation.

This is guidance only. Legal requirements vary by country, state/province, and sometimes municipality.

---

## 6. Retention Policy Guidance

| Situation | Suggested maximum retention |
|---|---|
| No incident occurred | 30 days (verify local requirements) |
| Incident under investigation | Retain until resolved; consult legal counsel |
| Subject-access request pending | Retain relevant footage until the request is resolved |

- Set a documented retention period and apply it consistently.
- Delete footage when retention period expires, using secure deletion methods.
- Never overwrite footage known to be relevant to an incident.

---

## 7. Security Baseline

| Control | Requirement |
|---|---|
| Disk / video storage encryption | AES-256 minimum; AES-128 only as compatibility fallback |
| Evidence export integrity | SHA-256 manifest hash |
| Evidence export signing | RSA-4096 (default recommendation) |
| Remote access | VPN-first; Tor onion service as opt-in advanced alternative only |
| Authentication | Strong passwords + 2FA on all admin interfaces |
| Software updates | Apply security updates promptly |
| Network isolation | NVR/terminal should be on a dedicated VLAN or isolated network segment where possible |

See [`docs/SECURITY_ARCHITECTURE.md`](SECURITY_ARCHITECTURE.md) for full technical details.

---

## 8. Prohibited Uses

The following are **strictly prohibited** under this standard:

- Facial recognition, identity matching, or biometric profiling from footage
- Emotion detection, psychological profiling, or behavioral inference
- Doxxing: using footage or associated metadata to publish personal information about individuals
- Stalking or harassment: using the system to monitor individuals across locations or for personal harassment
- Sharing footage publicly during an active law-enforcement investigation without explicit authorisation
- Providing real-time or persistent remote access to footage to any third party without the subject's knowledge and lawful authority
- Any use intended to intimidate, coerce, or retaliate against individuals
- Circumventing or interfering with a lawful investigation

---

## 9. Operator Responsibilities (Businesses)

- Designate a responsible operator (person) accountable for the surveillance system.
- Document the purpose of recording, camera locations, retention period, and access controls.
- Restrict access to footage to personnel with a defined need.
- Train authorised staff on proper handling of footage and incident response.
- Review and update this documentation at least annually or when circumstances change.

---

## 10. Consistency with Jitterbugs Project Principles

This standard is consistent with the Jitterbugs project's GPLv3 license, local-first architecture, and no-profiling stance documented in [`docs/SAFETY-SCOPE-POLICY.md`](SAFETY-SCOPE-POLICY.md).
