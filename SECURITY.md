# Security Policy

## Reporting a Vulnerability

We welcome reports from security researchers, including members of the DEF CON and Black Hat communities.

**Preferred method:** Use [GitHub Security Advisories](https://github.com/backbiten/jitterbugs/security/advisories) to submit a report confidentially. This is the fastest path to acknowledgement and a coordinated fix.

**Alternative:** If Security Advisories are not enabled or accessible, open a regular GitHub issue titled "Security Report" and request a private channel. We will respond and set up a private discussion.

**Scope:** All components in this repository — including the NVR/terminal application, the QAQC scanner, packaging scripts, and CI configuration.

**What to include:**
- A clear description of the vulnerability
- Steps to reproduce or a proof-of-concept
- Your assessment of impact and severity
- Any suggested mitigations (optional but appreciated)

**What to expect:**
- Acknowledgement within 5 business days
- Status update within 14 days
- Coordinated disclosure; we will credit reporters in the release notes unless you prefer to remain anonymous

**Boundaries:**
- There are no special access paths, backdoors, or undocumented APIs in this project. Do not attempt to negotiate or request them.
- This project does not include law-enforcement remote-access features by design; reports about the absence of such features are not security vulnerabilities.
- Social engineering or phishing attempts against maintainers are out of scope.

We treat all good-faith security reports seriously. Thank you for helping keep Jitterbugs users safe.

---

## Privacy Expectations

- Local-first behavior by default; no data leaves the device without explicit operator action
- No cloud uploads by default
- No telemetry or analytics
- Clear indicators while recording
- No profiling, facial recognition, or psychological inference (see [`docs/SAFETY-SCOPE-POLICY.md`](docs/SAFETY-SCOPE-POLICY.md))

See [`docs/SECURITY_ARCHITECTURE.md`](docs/SECURITY_ARCHITECTURE.md) for full security design details.
