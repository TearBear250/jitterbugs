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

## Building & Packaging

See [docs/build.md](docs/build.md) for instructions on building the desktop
application locally (Linux binary, `.deb` package, Windows `.exe`) and for
an overview of the CI pipeline.

## Networked TLS examples

The `examples/` directory contains a self-contained demonstration of how to
transmit jitter scores securely over HTTPS, with optional **mutual TLS
(mTLS)**:

- **`examples/server.py`** – aiohttp HTTPS server exposing a `POST /jitter`
  endpoint that accepts `{"score": <float>}`.
- **`examples/client.py`** – `requests` client that POSTs a score to the
  server, presenting a client certificate for mTLS.
- **`scripts/make-certs.sh`** – helper script that generates a local test CA,
  server certificate (`localhost`), and client certificate for local testing.

See [examples/README_TLS.md](examples/README_TLS.md) for step-by-step
instructions, `curl` examples, and guidance on adapting for production
(ACME/Let's Encrypt, HSM/TPM, certificate rotation).
