#!/usr/bin/env python3
"""
verify_location_signature.py
============================
Utility to verify the ed25519 signature on a jitterbugs device manifest and
validate that the manifest meets basic field and timestamp requirements.

Usage
-----
    python compliance/verify_location_signature.py <manifest.json> <pubkey.pem>

Arguments
    manifest.json   Path to a JSON file containing the device manifest.
    pubkey.pem      Path to a PEM-encoded ed25519 public key for the device.

Exit codes
    0 — manifest verified OK
    1 — verification failed (signature invalid, timestamp skew, missing fields,
        or other constraint violation)

Requirements
    cryptography >= 41.0  (pip install cryptography)

Notes
-----
The manifest is canonicalised (UTF-8, keys sorted, no extra whitespace) before
hashing because that is how the device signs it.  Any cosmetic difference in
field ordering between the received JSON and the signed canonical form is
normalised away before verification.

The 'signature' field itself is excluded from the canonical form that is signed.
"""

from __future__ import annotations

import base64
import hashlib
import json
import sys
from datetime import datetime, timezone, timedelta
from pathlib import Path

REQUIRED_FIELDS = {
    "device_id",
    "timestamp_utc",
    "nonce",
    "firmware_version",
    "model",
    "device_key_id",
    "tamper_flags",
    "signature",
}

MAX_TIMESTAMP_SKEW_SECONDS = 30


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _load_public_key(pem_path: str):
    """Load an ed25519 public key from a PEM file."""
    try:
        from cryptography.hazmat.primitives.serialization import load_pem_public_key
    except ImportError as exc:
        raise SystemExit(
            "ERROR: 'cryptography' package not found. "
            "Install it with: pip install cryptography"
        ) from exc

    pem_bytes = Path(pem_path).read_bytes()
    return load_pem_public_key(pem_bytes)


def _canonical_json(manifest: dict) -> bytes:
    """
    Return the canonical JSON bytes used for signing.

    The 'signature' field is excluded so the digest matches what the device
    signed before it appended the signature.
    """
    payload = {k: v for k, v in manifest.items() if k != "signature"}
    return json.dumps(payload, sort_keys=True, separators=(",", ":"),
                      ensure_ascii=True).encode("utf-8")


def _check_required_fields(manifest: dict) -> list[str]:
    """Return a list of missing required field names."""
    return [f for f in REQUIRED_FIELDS if f not in manifest]


def _check_timestamp(manifest: dict) -> str | None:
    """Return an error string if the timestamp is outside the allowed skew, else None."""
    raw = manifest.get("timestamp_utc", "")
    try:
        ts = datetime.fromisoformat(raw.replace("Z", "+00:00"))
    except ValueError:
        return f"timestamp_utc '{raw}' is not a valid ISO 8601 datetime"

    now = datetime.now(tz=timezone.utc)
    skew = abs((now - ts).total_seconds())
    if skew > MAX_TIMESTAMP_SKEW_SECONDS:
        return (
            f"timestamp_utc skew is {skew:.1f} s "
            f"(max {MAX_TIMESTAMP_SKEW_SECONDS} s). "
            "This manifest may be replayed or the device clock is drifting."
        )
    return None


def _verify_signature(manifest: dict, public_key) -> str | None:
    """
    Verify the ed25519 signature.  Returns an error string on failure, None on success.
    """
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
    from cryptography.exceptions import InvalidSignature

    if not isinstance(public_key, Ed25519PublicKey):
        return "Provided key is not an ed25519 public key"

    sig_b64 = manifest.get("signature", "")
    try:
        # Accept both standard and URL-safe Base64
        sig_bytes = base64.urlsafe_b64decode(sig_b64 + "==")
    except Exception as exc:
        return f"Cannot decode signature field: {exc}"

    canonical = _canonical_json(manifest)
    digest = hashlib.sha256(canonical).digest()

    try:
        public_key.verify(sig_bytes, digest)
    except InvalidSignature:
        return "Signature is INVALID — manifest may have been tampered with"

    return None


# ---------------------------------------------------------------------------
# Main verification routine
# ---------------------------------------------------------------------------

def verify_manifest(manifest_path: str, pubkey_path: str) -> bool:
    """
    Verify a manifest file against a public key.

    Prints a human-readable result and returns True on success, False on failure.
    """
    try:
        raw_json = Path(manifest_path).read_text(encoding="utf-8")
        manifest = json.loads(raw_json)
    except (OSError, json.JSONDecodeError) as exc:
        print(f"ERROR loading manifest: {exc}")
        return False

    errors: list[str] = []

    # 1. Required fields
    missing = _check_required_fields(manifest)
    if missing:
        errors.append(f"Missing required fields: {', '.join(sorted(missing))}")

    # 2. Timestamp skew
    ts_err = _check_timestamp(manifest)
    if ts_err:
        errors.append(ts_err)

    # 3. Location / location_unavailable mutual exclusion
    has_location = "location" in manifest
    location_unavailable = manifest.get("location_unavailable", False)
    if not has_location and not location_unavailable:
        errors.append(
            "Manifest has neither 'location' nor 'location_unavailable: true'"
        )
    if has_location and location_unavailable:
        errors.append(
            "'location' and 'location_unavailable: true' are mutually exclusive"
        )

    # 4. Signature verification
    try:
        public_key = _load_public_key(pubkey_path)
    except (OSError, ValueError) as exc:
        errors.append(f"Cannot load public key: {exc}")
        public_key = None

    if public_key is not None:
        sig_err = _verify_signature(manifest, public_key)
        if sig_err:
            errors.append(sig_err)

    # Report
    if errors:
        print("VERIFICATION FAILED")
        for err in errors:
            print(f"  • {err}")
        return False

    device_id = manifest.get("device_id", "(unknown)")
    ts = manifest.get("timestamp_utc", "(unknown)")
    tamper = manifest.get("tamper_flags", 0)
    coarse = manifest.get("coarse", False)
    print("VERIFICATION OK")
    print(f"  device_id       : {device_id}")
    print(f"  timestamp_utc   : {ts}")
    print(f"  tamper_flags    : {tamper:#010b} ({tamper})")
    print(f"  coarse mode     : {coarse}")
    if has_location:
        loc = manifest["location"]
        print(
            f"  location        : {loc['latitude']}, {loc['longitude']} "
            f"±{loc['accuracy_m']} m [{loc['source']}]"
        )
    return True


# ---------------------------------------------------------------------------
# Self-test (placeholder — replace public key and signature with real values)
# ---------------------------------------------------------------------------

SAMPLE_MANIFEST = {
    "device_id": "dev_ABC123",
    "timestamp_utc": datetime.now(tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "nonce": "dGhpcyBpcyBhIHRlc3Qgbm9uY2VhYWFhYWFhYQ",
    "location": {
        "latitude": 37.7749,
        "longitude": -122.4194,
        "accuracy_m": 8.5,
        "source": "AssistedGNSS",
        "hdop": 1.2,
    },
    "coarse": False,
    "firmware_version": "2.4.1",
    "model": "JB-T300",
    "device_key_id": "sha256:" + "a" * 64,
    "tamper_flags": 0,
    "signature": "PLACEHOLDER_REPLACE_WITH_REAL_SIGNATURE" + "a" * 47,
}


def _run_self_test() -> None:
    """
    Demonstrate the field-validation path using a sample manifest.

    NOTE: The signature in SAMPLE_MANIFEST is a placeholder and will fail
    cryptographic verification.  This self-test only validates that field
    checks and timestamp checks work correctly when a valid public-key file
    is not available.
    """
    print("=== Self-test (field validation only — no real key) ===")
    errors: list[str] = []

    missing = _check_required_fields(SAMPLE_MANIFEST)
    if missing:
        errors.append(f"Missing fields: {missing}")

    ts_err = _check_timestamp(SAMPLE_MANIFEST)
    if ts_err:
        errors.append(ts_err)

    if "location" not in SAMPLE_MANIFEST and not SAMPLE_MANIFEST.get(
        "location_unavailable", False
    ):
        errors.append("No location data")

    if errors:
        print("Self-test field checks FAILED:")
        for e in errors:
            print(f"  {e}")
    else:
        print("Self-test field checks PASSED (signature check skipped — no real key).")
        print(
            "  To test full verification supply a real manifest and matching "
            "public key:\n"
            "      python compliance/verify_location_signature.py "
            "manifest.json device_pubkey.pem"
        )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    if len(sys.argv) == 1:
        _run_self_test()
        sys.exit(0)

    if len(sys.argv) != 3:
        print(
            f"Usage: {sys.argv[0]} <manifest.json> <pubkey.pem>\n"
            "       or run without arguments for a self-test."
        )
        sys.exit(1)

    ok = verify_manifest(sys.argv[1], sys.argv[2])
    sys.exit(0 if ok else 1)
