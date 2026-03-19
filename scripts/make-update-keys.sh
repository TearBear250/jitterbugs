#!/usr/bin/env bash
# scripts/make-update-keys.sh
#
# Generate a test RSA signing keypair and sign an example manifest.
# Outputs:
#   update_signing_key.pem  – RSA-2048 private key  (KEEP SECRET; NOT FOR PRODUCTION)
#   update_public_key.pem   – Corresponding public key
#   examples/manifest.json.sig – Raw binary RSA-PKCS1v15/SHA-256 signature
#   examples/manifest.json.sig.b64 – Base64-encoded version of the signature
#
# WARNING: NOT FOR PRODUCTION.
#   - Store signing keys in an HSM or a secrets manager in production.
#   - Rotate keys regularly and distribute public key updates via a
#     signed key-rotation manifest.
#   - This script is for local development / demo purposes only.
#
# Prerequisites: openssl (available on most Unix systems)
#
# Usage:
#   chmod +x scripts/make-update-keys.sh
#   ./scripts/make-update-keys.sh
#   # Then edit examples/updater.py to embed the contents of update_public_key.pem
#   # in the EMBEDDED_PUBLIC_KEY_PEM constant.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
EXAMPLES_DIR="${REPO_ROOT}/examples"
MANIFEST="${EXAMPLES_DIR}/manifest_example.json"

echo "=== jitterbugs update key generation (NOT FOR PRODUCTION) ==="

# ---------------------------------------------------------------------------
# 1. Generate RSA-2048 private key
# ---------------------------------------------------------------------------
echo "[1/4] Generating RSA-2048 private key → update_signing_key.pem"
openssl genrsa -out update_signing_key.pem 2048

# ---------------------------------------------------------------------------
# 2. Extract public key
# ---------------------------------------------------------------------------
echo "[2/4] Extracting public key → update_public_key.pem"
openssl rsa -in update_signing_key.pem -pubout -out update_public_key.pem

# ---------------------------------------------------------------------------
# 3. Sign the example manifest (RSA-PKCS1v15, SHA-256)
# ---------------------------------------------------------------------------
if [[ ! -f "${MANIFEST}" ]]; then
    echo "ERROR: manifest not found at ${MANIFEST}"
    echo "       Run from the repository root after creating examples/manifest_example.json."
    exit 1
fi

echo "[3/4] Signing ${MANIFEST} → ${EXAMPLES_DIR}/manifest.json.sig"
openssl dgst -sha256 -sign update_signing_key.pem \
    -out "${EXAMPLES_DIR}/manifest.json.sig" \
    "${MANIFEST}"

# ---------------------------------------------------------------------------
# 4. Base64-encode the signature (the updater client expects base64)
# ---------------------------------------------------------------------------
echo "[4/4] Base64-encoding signature → ${EXAMPLES_DIR}/manifest.json.sig.b64"
base64 < "${EXAMPLES_DIR}/manifest.json.sig" > "${EXAMPLES_DIR}/manifest.json.sig.b64"

echo ""
echo "Done!"
echo ""
echo "Files created:"
echo "  update_signing_key.pem          – Private key (DO NOT COMMIT; add to .gitignore)"
echo "  update_public_key.pem           – Public key  (embed in examples/updater.py)"
echo "  ${EXAMPLES_DIR}/manifest.json.sig     – Raw binary signature"
echo "  ${EXAMPLES_DIR}/manifest.json.sig.b64 – Base64 signature (serve this as manifest.json.sig)"
echo ""
echo "Next steps:"
echo "  1. Copy the contents of update_public_key.pem into the EMBEDDED_PUBLIC_KEY_PEM"
echo "     constant in examples/updater.py."
echo "  2. Serve manifest_example.json and manifest.json.sig.b64 (as manifest.json.sig)"
echo "     via your HTTPS server (see examples/server.py or examples/README_UPDATER.md)."
echo "  3. Run: python examples/updater.py --check-only --verify certs/ca.crt"
echo ""
echo "WARNING: update_signing_key.pem is the private signing key."
echo "         Keep it secret and do NOT commit it to the repository."
