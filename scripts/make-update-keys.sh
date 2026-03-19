#!/usr/bin/env bash
# =============================================================================
# scripts/make-update-keys.sh
#
# ⚠️  DEMO / NOT FOR PRODUCTION ⚠️
# Generates an RSA signing keypair for local testing of the jitterbugs
# auto-updater prototype, then signs the example manifest.
#
# In production:
#   - Store private keys in an HSM or OS key store — NEVER on disk in plaintext.
#   - Use a CA-issued certificate chain rather than a bare self-generated key.
#   - Automate key rotation with a secure secrets-management solution.
#   - Restrict access to this script (chmod 700) and to the key files.
#
# Usage:
#   bash scripts/make-update-keys.sh [path/to/manifest.json]
#
#   If no argument is given the script looks for examples/manifest_example.json.
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
EXAMPLES_DIR="${REPO_ROOT}/examples"

# Output paths
PRIVATE_KEY="${EXAMPLES_DIR}/update_signing_key.pem"
PUBLIC_KEY="${EXAMPLES_DIR}/update_public_key.pem"

# Manifest to sign (default: examples/manifest_example.json)
MANIFEST="${1:-${EXAMPLES_DIR}/manifest_example.json}"
SIG_BIN="${MANIFEST}.sig"
SIG_B64="${MANIFEST}.sig.b64"

echo "==================================================================="
echo "  jitterbugs update-key generator — DEMO ONLY, NOT FOR PRODUCTION"
echo "==================================================================="
echo

# -------------------------------------------------------------------------
# 1) Generate RSA 4096-bit private key (demo key — keep private!)
# -------------------------------------------------------------------------
echo "[1/4] Generating RSA 4096-bit private key → ${PRIVATE_KEY}"
openssl genrsa -out "${PRIVATE_KEY}" 4096
chmod 600 "${PRIVATE_KEY}"
echo "      Private key written (chmod 600). Keep this file secret."

# -------------------------------------------------------------------------
# 2) Extract the public key
# -------------------------------------------------------------------------
echo "[2/4] Extracting public key → ${PUBLIC_KEY}"
openssl rsa -in "${PRIVATE_KEY}" -pubout -out "${PUBLIC_KEY}"
echo "      Public key written."

# -------------------------------------------------------------------------
# 3) Sign the manifest (binary DER signature)
# -------------------------------------------------------------------------
if [[ ! -f "${MANIFEST}" ]]; then
    echo "WARNING: Manifest file not found at '${MANIFEST}'. Skipping signing step."
    echo "         Create the manifest first and re-run, or pass the path as an argument."
    exit 0
fi

echo "[3/4] Signing manifest → ${SIG_BIN}"
openssl dgst -sha256 -sign "${PRIVATE_KEY}" -out "${SIG_BIN}" "${MANIFEST}"
echo "      Signature written."

# -------------------------------------------------------------------------
# 4) Base64-encode the signature for easy hosting / transport
# -------------------------------------------------------------------------
echo "[4/4] Base64-encoding signature → ${SIG_B64}"
openssl base64 -in "${SIG_BIN}" -out "${SIG_B64}"
echo "      Base64 signature written."

echo
echo "Done.  Files created:"
echo "  Private key : ${PRIVATE_KEY}"
echo "  Public key  : ${PUBLIC_KEY}"
echo "  Signature   : ${SIG_BIN}"
echo "  Sig (b64)   : ${SIG_B64}"
echo
echo "Next steps:"
echo "  1. Copy the contents of ${PUBLIC_KEY} into the EMBEDDED_PUBLIC_KEY_PEM"
echo "     constant in examples/updater.py (or pass --public-key-file at runtime)."
echo "  2. Host manifest.json and manifest.json.sig via examples/server.py."
echo "  3. Run: python examples/updater.py --manifest-url https://localhost:8443/updates/manifest.json \\"
echo "               --sig-url https://localhost:8443/updates/manifest.json.sig \\"
echo "               --public-key-file ${PUBLIC_KEY} --no-verify-tls --check-only"
echo
echo "⚠️  REMINDER: The private key at ${PRIVATE_KEY} must NEVER be committed to"
echo "    version control or deployed to production systems."
