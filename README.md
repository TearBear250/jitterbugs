# jitterbugs

A local-first camera utility for previewing and recording video to local storage.

## What this is

jitterbugs is a simple, privacy-respecting camera/recorder application. It is designed for personal use where the user controls what is captured, stored, and deleted.

**Goals:**
- Camera preview and recording to local storage
- User-controlled start/stop
- Local file management (retain/delete/export)
- Clear, prominent UI indicators when recording is active

**Non-goals:**
- No facial recognition or identity matching
- No emotion detection or behavioral inference
- No cloud uploads by default
- No background or hidden recording

See [docs/SAFETY-SCOPE-POLICY.md](docs/SAFETY-SCOPE-POLICY.md) for the full scope and data-handling policy.

---

## Responsible Use & Privacy

Jitterbugs is designed for **self-protection and incident documentation** by home and small-business operators.

**Privacy-by-default:**
- All video and logs are stored locally. Nothing leaves the device without explicit operator action.
- No telemetry, no background cloud sync, and no third-party data sharing in the default configuration.
- Incident evidence packages are created locally and exported to USB/microSD — not uploaded automatically.

**Allowed uses:**
- Recording your own premises for security purposes
- Exporting footage to support a police report or insurance claim
- Reviewing footage for incidents that occurred on your property

**Not allowed:**
- Doxxing, stalking, or harassment using footage or metadata
- Identity matching or facial recognition from footage
- Psychological profiling or behavioral inference
- Sharing footage publicly during an active law-enforcement investigation
- Providing third parties with persistent or real-time access to your cameras without lawful basis

See [docs/OCEAN_PROTOCOL.md](docs/OCEAN_PROTOCOL.md) for the full operator standard, and [docs/DATA_FLOWS.md](docs/DATA_FLOWS.md) for a complete description of where data is stored and when it may leave the device.

> **Not legal advice.** Requirements vary by jurisdiction. Consult qualified legal counsel for obligations that apply to you.

---

## Law Enforcement Cooperation (Non-Legal Advice)

If an incident occurs, here is a general-purpose, privacy-preserving approach for working with law enforcement. Verify requirements with local legal counsel — this is not legal advice.

### Preserve evidence
- Do not delete, overwrite, or edit footage once an incident is identified.
- If storage may be overwritten by the retention schedule, export the relevant footage immediately using the evidence package tool.
- Retain originals on the NVR until you receive written confirmation from investigators that you may delete them.

### Export and provide
- Export an evidence package to USB or microSD (see [docs/EVIDENCE_PACKAGE.md](docs/EVIDENCE_PACKAGE.md) for the specification).
- The package includes: incident summary, timeline, original video clips, audit log excerpt, system health snapshot, and an integrity manifest (SHA-256 + optional RSA-4096 signature).
- Keep your own copy. Provide a copy to law enforcement or legal counsel upon request, and ask for a case number.

### During an investigation
- Do not share footage publicly while an investigation is active.
- If police request device credentials or network access, respond consistent with local law and on legal counsel's advice. There is no obligation to provide real-time access in most jurisdictions.
- For business operators: follow local CCTV signage, retention, and subject-access-request rules.

### US guidance (general)
- Federal and state wiretapping/privacy laws apply. Audio recording may require all-party consent depending on the state.
- Do not interfere with a federal, state, or local investigation; preserve originals and avoid editing.
- Work through local law enforcement for any cross-jurisdictional matters.

### International guidance (general)
- **EU/UK:** CCTV is subject to GDPR/UK GDPR. Post required signage, respond to subject-access requests, and retain footage only as long as necessary.
- **Canada:** PIPEDA and provincial privacy laws apply; treat footage as personal information.
- **Australia:** The Privacy Act and state surveillance legislation apply.
- **All jurisdictions:** For cross-border incidents, work through your local police, who will coordinate with relevant international bodies (Interpol, ICC) as appropriate. Do not conduct independent international investigations.

See [docs/OSINT_INCIDENT_RESPONSE.md](docs/OSINT_INCIDENT_RESPONSE.md) for the full incident-response workflow and reporting template.

---

## Building & Packaging

See [docs/build.md](docs/build.md) for instructions on building the desktop
application locally (Linux binary, `.deb` package, Windows `.exe`) and for
an overview of the CI pipeline.
