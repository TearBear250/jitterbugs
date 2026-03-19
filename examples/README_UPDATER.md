# Auto-Updater Prototype – Developer Guide

> **WARNING: FOR DEMO/TESTING ONLY. DO NOT USE THIS AS A PRODUCTION UPDATE MECHANISM.**
> See [Trust model & production recommendations](#trust-model--production-recommendations) below.

This document walks through every step needed to exercise the auto-updater
locally: generating test keys, signing a manifest, hosting the files over TLS,
and running the updater in check-only or auto-install mode.

---

## Prerequisites

- Python 3.8+ with `pip`
- `openssl` CLI (standard on Linux/macOS)
- Test TLS certificates (generate with `scripts/make-certs.sh` if present, or
  see examples/server.py for a self-signed setup)

Install Python dependencies:

```bash
pip install -r requirements.txt
```

---

## 1 – Generate a test signing keypair

Run the helper script from the **repository root**:

```bash
chmod +x scripts/make-update-keys.sh
./scripts/make-update-keys.sh
```

This creates:

| File | Purpose |
|---|---|
| `update_signing_key.pem` | RSA-2048 private key – **keep secret** |
| `update_public_key.pem` | Corresponding public key – embed in the updater |
| `examples/manifest.json.sig` | Raw binary RSA-PKCS1v15/SHA-256 signature |
| `examples/manifest.json.sig.b64` | Base64-encoded signature (serve this as `manifest.json.sig`) |

> **Important:** `update_signing_key.pem` must not be committed to the repository.
> Add it to `.gitignore`.

---

## 2 – Embed the public key in the updater

Open `examples/updater.py` and replace the placeholder value of
`EMBEDDED_PUBLIC_KEY_PEM` with the content of `update_public_key.pem`:

```python
EMBEDDED_PUBLIC_KEY_PEM = b"""\
-----BEGIN PUBLIC KEY-----
<paste contents of update_public_key.pem here>
-----END PUBLIC KEY-----
"""
```

Alternatively, you can modify `updater.py` to load the key from a file path
supplied via a CLI flag—see the code comments for guidance.

---

## 3 – Sign the manifest

The signing script already signs `examples/manifest_example.json`.
To sign a different manifest:

```bash
openssl dgst -sha256 -sign update_signing_key.pem \
    -out examples/manifest.json.sig  path/to/manifest.json

# Base64-encode for the updater (expects base64 over HTTPS):
base64 < examples/manifest.json.sig > examples/manifest.json.sig.b64
```

Edit `examples/manifest_example.json` to set the correct `version`, artifact
`url`, and `sha256` before signing.  
Compute a real SHA-256 with:

```bash
sha256sum path/to/jitterbugs-EXAMPLE.tar.gz
```

---

## 4 – Host the manifest and artifact over HTTPS

### Option A – Use `examples/server.py` (if present)

```bash
# Start the TLS server (port 8443 by default):
python examples/server.py
```

Place your manifest and artifact in the directory the server exposes as
`/updates/`.

### Option B – Use any HTTPS server

Copy `examples/manifest_example.json` → `manifest.json`, copy
`examples/manifest.json.sig.b64` → `manifest.json.sig`, and your artifact to a
directory served over HTTPS.

> Make sure the server certificate is trusted by the client or pass `--verify`
> pointing to the CA cert.

---

## 5 – Run the updater

### Check-only mode (no install)

```bash
python examples/updater.py \
    --manifest-url https://localhost:8443/updates/manifest.json \
    --sig-url      https://localhost:8443/updates/manifest.json.sig \
    --verify       certs/ca.crt \
    --check-only
```

### Auto-install mode

```bash
python examples/updater.py \
    --manifest-url https://localhost:8443/updates/manifest.json \
    --sig-url      https://localhost:8443/updates/manifest.json.sig \
    --verify       certs/ca.crt \
    --auto
```

### Interactive mode (prompts for confirmation)

```bash
python examples/updater.py \
    --manifest-url https://localhost:8443/updates/manifest.json \
    --sig-url      https://localhost:8443/updates/manifest.json.sig \
    --verify       certs/ca.crt
```

### Override the version file

```bash
python examples/updater.py --version-file /tmp/my_version.txt --check-only
```

### All CLI flags

| Flag | Default | Description |
|---|---|---|
| `--manifest-url` | `https://localhost:8443/updates/manifest.json` | HTTPS URL of the signed manifest |
| `--sig-url` | `https://localhost:8443/updates/manifest.json.sig` | HTTPS URL of the base64 signature |
| `--verify` | `certs/ca.crt` | TLS verification: CA bundle path, `true` (system CAs), or `false` (disable – insecure) |
| `--version-file` | `examples/.version` | Path to the local version file |
| `--check-only` | `False` | Only check; do not download or install |
| `--auto` | `False` | Download and install without prompting |

Logs are written to `examples/updater.log` and echoed to stdout.

---

## 6 – Rollback

If an install fails after the backup copy has been created, the updater
automatically restores the previous version from `examples/backup/`.

To trigger a manual rollback:

```bash
cp examples/backup/<filename> examples/current/<filename>
```

---

## Trust model & production recommendations

### What this prototype demonstrates

| Feature | Implementation |
|---|---|
| Manifest integrity | RSA-PKCS1v15/SHA-256 signature, verified with an embedded public key |
| Transport security | HTTPS (`requests` with configurable CA verification) |
| Artifact integrity | SHA-256 checksum compared to manifest entry |
| Atomic install | Temp file + `os.replace()` swap |
| Rollback | Backup copy kept; restored on install failure |
| Audit trail | Structured log to `examples/updater.log` |

### Why this is NOT production-ready

1. **Embedded public key** – the key is hardcoded in the script.
   Anyone who can modify the script can substitute their own key.
2. **No key rotation** – there is no mechanism to distribute a new public key.
3. **No user consent flow** – `--auto` installs without any confirmation UI.
4. **Poll interval** – no built-in scheduler; callers must schedule via cron/systemd.
5. **No mTLS** – the server cannot authenticate individual clients.

### Recommended production changes

| Area | Recommendation |
|---|---|
| Certificate management | Use ACME (Let's Encrypt or internal CA) for automatic renewal |
| Signing key storage | Store the private signing key in an HSM or secrets manager; never on disk |
| Public key distribution | Deliver the initial public key during first install; update via signed key-rotation manifests |
| Transport | Keep HTTPS; add mTLS (client certificates stored in device TPM/secure enclave) for client authentication |
| Poll interval | Default to ≥ 1 hour; randomize with jitter; use `If-Modified-Since` / ETag to avoid unnecessary traffic |
| User consent | Display a changelog and require explicit user consent before installing |
| Artifact signing | Sign artifacts (not just the manifest) for defence-in-depth |
| Logging | Ship logs to a secure, tamper-evident audit log store |
| Rollback | Test rollback paths in CI; provide a user-facing "undo last update" command |
| Legal | Disclose auto-update behaviour to end users; provide an opt-out |
