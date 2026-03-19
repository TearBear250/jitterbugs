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

## Auto-updater prototype

An experimental, demo-only secure auto-updater is provided in `examples/`.
It fetches a signed manifest over HTTPS, verifies the RSA signature with an
embedded public key, downloads the artifact, checks its SHA-256 checksum, and
installs atomically using a temp-file + swap with backup/rollback support.

See [examples/README_UPDATER.md](examples/README_UPDATER.md) for a full walkthrough,
including key generation, manifest signing, HTTPS hosting, and CLI usage.

> **For demo/testing only – see the README for production recommendations.**

## Building & Packaging

See [docs/build.md](docs/build.md) for instructions on building the desktop
application locally (Linux binary, `.deb` package, Windows `.exe`) and for
an overview of the CI pipeline.
