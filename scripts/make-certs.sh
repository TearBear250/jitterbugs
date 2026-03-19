#!/usr/bin/env bash
# scripts/make-certs.sh
#
# Generate a local test CA, server certificate (localhost), and client
# certificate for testing TLS / mutual-TLS (mTLS) locally.
#
# !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
# WARNING: NOT FOR PRODUCTION USE.
# These certificates are self-signed, use short validity periods, and are
# intended ONLY for local development and testing.  In production, obtain
# certificates from a trusted CA (e.g. Let's Encrypt / ACME) or an HSM/TPM.
# !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
#
# Usage:
#   bash scripts/make-certs.sh
#
# Output files (relative to the repo root):
#   certs/ca.key        – CA private key
#   certs/ca.crt        – CA certificate (self-signed)
#   certs/server.key    – server private key
#   certs/server.crt    – server certificate (signed by test CA, CN=localhost)
#   certs/client.key    – client private key
#   certs/client.crt    – client certificate (signed by test CA)
#   certs/client.pem    – combined client cert + key (for requests / curl)

set -euo pipefail

CERT_DIR="certs"
DAYS=365

echo "==> Creating output directory: ${CERT_DIR}/"
mkdir -p "${CERT_DIR}"

# ---------------------------------------------------------------------------
# 1. Test Certificate Authority (CA)
# ---------------------------------------------------------------------------
echo "==> Generating test CA key and self-signed certificate..."
openssl genrsa -out "${CERT_DIR}/ca.key" 4096

openssl req -x509 -new -nodes \
    -key "${CERT_DIR}/ca.key" \
    -sha256 \
    -days "${DAYS}" \
    -subj "/CN=JitterbugTestCA/O=JitterbugsDev/C=US" \
    -out "${CERT_DIR}/ca.crt"

# ---------------------------------------------------------------------------
# 2. Server certificate (localhost)
# ---------------------------------------------------------------------------
echo "==> Generating server key and CSR..."
openssl genrsa -out "${CERT_DIR}/server.key" 4096

openssl req -new \
    -key "${CERT_DIR}/server.key" \
    -subj "/CN=localhost/O=JitterbugsDev/C=US" \
    -out "${CERT_DIR}/server.csr"

echo "==> Signing server certificate with test CA..."
openssl x509 -req \
    -in "${CERT_DIR}/server.csr" \
    -CA "${CERT_DIR}/ca.crt" \
    -CAkey "${CERT_DIR}/ca.key" \
    -CAcreateserial \
    -days "${DAYS}" \
    -sha256 \
    -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1") \
    -out "${CERT_DIR}/server.crt"

rm -f "${CERT_DIR}/server.csr"

# ---------------------------------------------------------------------------
# 3. Client certificate (for mTLS)
# ---------------------------------------------------------------------------
echo "==> Generating client key and CSR..."
openssl genrsa -out "${CERT_DIR}/client.key" 4096

openssl req -new \
    -key "${CERT_DIR}/client.key" \
    -subj "/CN=JitterbugTestClient/O=JitterbugsDev/C=US" \
    -out "${CERT_DIR}/client.csr"

echo "==> Signing client certificate with test CA..."
openssl x509 -req \
    -in "${CERT_DIR}/client.csr" \
    -CA "${CERT_DIR}/ca.crt" \
    -CAkey "${CERT_DIR}/ca.key" \
    -CAcreateserial \
    -days "${DAYS}" \
    -sha256 \
    -out "${CERT_DIR}/client.crt"

rm -f "${CERT_DIR}/client.csr"

# ---------------------------------------------------------------------------
# 4. Combined client PEM (cert + key) for use with requests / curl
# ---------------------------------------------------------------------------
echo "==> Creating combined client.pem (cert + key)..."
cat "${CERT_DIR}/client.crt" "${CERT_DIR}/client.key" > "${CERT_DIR}/client.pem"

# Restrict key file permissions
chmod 600 "${CERT_DIR}/ca.key" "${CERT_DIR}/server.key" \
          "${CERT_DIR}/client.key" "${CERT_DIR}/client.pem"

echo ""
echo "Done.  Files written to ${CERT_DIR}/:"
ls -1 "${CERT_DIR}/"
echo ""
echo "Remember: these certificates are for LOCAL TESTING ONLY."
