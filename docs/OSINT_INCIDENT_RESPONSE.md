# OSINT Incident Response — Allowed-Use Workflow

> **Not legal advice.** This document provides general guidance only. Consult qualified legal counsel before acting on any information gathered during an investigation.

---

## 1. Purpose and Scope

This document defines the **allowed** use of open-source intelligence (OSINT) techniques in the context of a security incident affecting your home or small business — for example: a theft, break-in, or vandalism.

The goal is to help you **preserve evidence, prepare a factual report for law enforcement or legal counsel, and assist a legitimate investigation** — not to conduct your own investigation or take direct action.

---

## 2. Strict Prohibitions

The following actions are **prohibited** regardless of circumstances:

| Prohibited action | Why |
|---|---|
| Doxxing: publishing personal information about a suspect | Illegal in many jurisdictions; endangers uninvolved people; compromises investigations |
| Stalking / harassment: using footage or OSINT to monitor or contact individuals | Illegal; reverses victim/suspect roles legally |
| Identity matching from footage | Outside allowed scope; error-prone; potential civil and criminal liability |
| Psychological profiling | Outside allowed scope; see [`docs/SAFETY-SCOPE-POLICY.md`](SAFETY-SCOPE-POLICY.md) |
| Sharing footage on social media or public forums during an active investigation | Can compromise prosecution; may violate court orders |
| Cross-border independent "investigations" | Work through local law enforcement and Interpol/ICC channels only |
| Using OSINT findings to take direct action against a person | This is vigilantism; it can result in criminal charges |

---

## 3. Incident Response Workflow

### Step 1 — Contain and Preserve

- **Do not delete, overwrite, or edit any footage** once an incident is identified.
- Note the time you became aware of the incident.
- If storage is at risk of being overwritten by the retention schedule, immediately export the relevant footage. See [`docs/EVIDENCE_PACKAGE.md`](EVIDENCE_PACKAGE.md).
- Disable automatic deletion for the relevant time range until the investigation is resolved.

### Step 2 — Document the Timeline

Create a factual timeline of events. Include:

- Date and time (with timezone) of the incident as captured by the system
- Camera(s) that recorded the event
- Description of what is observable (actions, directions of movement, items involved)
- Time you exported the footage
- Any other relevant factual observations

Do not include speculation, identity claims, or inferences about intent.

### Step 3 — Export the Evidence Package

Export a structured evidence package to USB or microSD. See [`docs/EVIDENCE_PACKAGE.md`](EVIDENCE_PACKAGE.md) for the full specification.

The package includes:
- Incident summary (factual, templated)
- Timeline log
- Original video clips (unedited)
- Audit log excerpt
- System health snapshot
- SHA-256 manifest and optional RSA-4096 signature

Keep your own copy. Provide a copy to law enforcement or legal counsel upon request.

### Step 4 — Allowed OSINT Research

Allowed OSINT is limited to **supporting a police report or insurance claim** for property you own. Examples of allowed activity:

- Searching stolen-property listing sites (e.g., Craigslist, Facebook Marketplace) for items matching a documented description from your incident
- Reviewing publicly posted security footage of your own premises that you have already published (e.g., your own social media)
- Checking whether a vehicle seen on your property matches a public database of stolen vehicles (where such a database is publicly accessible to citizens)

**Document everything you find**, including the URL, the date you accessed it, and a screenshot or saved copy.

**Do not contact suspects or persons of interest directly.** Provide your findings to law enforcement.

### Step 5 — Prepare the Report

Use the [Incident Report Template](#4-incident-report-template) below to prepare a factual, professional document for law enforcement or legal counsel.

- Stick to observable facts.
- Do not speculate about identity, motive, or intent.
- Attach the evidence package (or reference its hash for chain-of-custody).

### Step 6 — Submit to Law Enforcement

- Bring your evidence package (USB/microSD) and printed or digital report to your local police station, or provide to investigators upon request.
- Ask for a case number and retain it.
- If police request your device credentials or network access information, respond consistent with local law and on legal counsel's advice. You are not required to provide real-time access.
- For cross-border incidents, work through your local police, who will coordinate with relevant international bodies (Interpol, etc.) as appropriate.

---

## 4. Incident Report Template

```
INCIDENT REPORT
===============

Report prepared by (name/role): ____________________________
Date prepared: ____________________________
Case number (if assigned): ____________________________

1. INCIDENT SUMMARY
   Date of incident: ____________________________
   Time of incident (with timezone): ____________________________
   Location: ____________________________
   Brief description (observable facts only):
   _______________________________________________________________
   _______________________________________________________________

2. PROPERTY INVOLVED (if applicable)
   Item description: ____________________________
   Serial number / identifying marks: ____________________________
   Estimated value: ____________________________

3. EVIDENCE COLLECTED
   Evidence package filename: ____________________________
   SHA-256 hash of package: ____________________________
   Date/time of export: ____________________________
   Exported by: ____________________________
   Storage medium: ____________________________

4. CAMERA/SYSTEM DETAILS
   System name/ID: ____________________________
   Firmware version: ____________________________
   Time source (NTP server): ____________________________
   Timezone: ____________________________

5. TIMELINE OF OBSERVABLE EVENTS
   (List each event with camera ID, timestamp, and factual description)
   - [timestamp] [camera ID]: ________________________________
   - [timestamp] [camera ID]: ________________________________
   - [timestamp] [camera ID]: ________________________________

6. OSINT FINDINGS (if any)
   (Describe only; do not act on findings directly)
   Finding: ____________________________
   Source URL: ____________________________
   Date accessed: ____________________________
   Screenshot/copy retained: Yes / No

7. ADDITIONAL NOTES
   _______________________________________________________________
   _______________________________________________________________

DECLARATION
I declare that the information in this report is accurate and
complete to the best of my knowledge.

Signature: ____________________________
Date: ____________________________
```

---

## 5. International Guidance Notes

- **US:** Follow federal and state laws on privacy, wiretapping, and evidence handling. Audio recording requires all-party consent in some states and only one-party consent in others; verify the rule for your state before recording audio. Do not interfere with a federal or state investigation.
- **EU/UK:** CCTV use is subject to GDPR/UK GDPR. Retain footage only as long as necessary; respond to subject-access requests. Do not share footage across borders without legal basis.
- **Canada:** PIPEDA and provincial laws apply. Handle footage as personal information.
- **Australia:** The Privacy Act and state surveillance legislation apply.
- **General (all jurisdictions):** Work through your local law enforcement for all cross-border matters. Do not conduct independent international investigations.

Verify requirements with local legal counsel. This document does not constitute legal advice.
