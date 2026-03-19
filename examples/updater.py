"""
updater.py – Secure auto-updater prototype for jitterbugs.

⚠️  DEMO / NOT FOR PRODUCTION ⚠️
This file is a prototype showing how a signed-manifest auto-updater can work.
Do NOT use this code as-is in production systems.  See examples/README_UPDATER.md
for production hardening recommendations (ACME certs, HSM key storage, mTLS,
user consent, key rotation, etc.).

Behaviour (overview):
  1. Fetch a JSON manifest + detached signature over HTTPS.
  2. Verify the signature with a hard-coded RSA public key (DEMO only –
     replace with your real public key before deploying anything).
  3. Compare the manifest version to the local version file.
  4. If the remote version is newer: download the artifact, verify its
     SHA-256 checksum, write to a temporary directory, then atomically
     swap it into examples/current/ keeping a backup for rollback.
  5. Log every step to examples/updater.log.

CLI flags
---------
  --check-only        Only check whether an update is available; do not install.
  --auto              Download *and* install without prompting (dangerous – demo).
  --manifest-url URL  Override the manifest URL.
  --sig-url URL       Override the signature URL.
  --verify            Verify the local installation integrity and exit.
  --version-file PATH Path to the local version file (default: examples/.version).
  --public-key-file P Path to a PEM public key file (overrides embedded key).

Requirements: requests>=2.28, cryptography>=3.4
"""

# ---------------------------------------------------------------------------
# Standard library
# ---------------------------------------------------------------------------
import argparse
import hashlib
import json
import logging
import os
import shutil
import sys
import tempfile
from pathlib import Path

# ---------------------------------------------------------------------------
# Third-party
# ---------------------------------------------------------------------------
try:
    import requests
except ImportError:
    sys.exit("Missing dependency: install 'requests' (pip install requests)")

try:
    from cryptography.exceptions import InvalidSignature
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import padding
except ImportError:
    sys.exit("Missing dependency: install 'cryptography' (pip install cryptography)")

# ---------------------------------------------------------------------------
# Configuration / defaults
# ---------------------------------------------------------------------------

# ⚠️  DEMO KEY – Replace with your real public key before production use.
# This is a placeholder that will fail signature verification by design;
# it exists only to show where the key should go.
EMBEDDED_PUBLIC_KEY_PEM = b"""\
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0000000000000000DEMO
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
AAAB
-----END PUBLIC KEY-----
"""

# Default paths (relative to examples/ directory)
_HERE = Path(__file__).parent
DEFAULT_VERSION_FILE = _HERE / ".version"
DEFAULT_CURRENT_DIR = _HERE / "current"
DEFAULT_BACKUP_DIR = _HERE / "backup"
DEFAULT_LOG_FILE = _HERE / "updater.log"

# Default URLs (point to localhost for local testing)
DEFAULT_MANIFEST_URL = "https://localhost:8443/updates/manifest.json"
DEFAULT_SIG_URL = "https://localhost:8443/updates/manifest.json.sig"

# ---------------------------------------------------------------------------
# Logging setup
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.StreamHandler(sys.stdout),
    ],
)
log = logging.getLogger("updater")


def _add_file_log_handler(log_path: Path) -> None:
    """Attach a file handler to the root updater logger."""
    log_path.parent.mkdir(parents=True, exist_ok=True)
    fh = logging.FileHandler(log_path)
    fh.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s"))
    log.addHandler(fh)


# ---------------------------------------------------------------------------
# Core helpers
# ---------------------------------------------------------------------------

def _load_public_key(pem_bytes: bytes):
    """Load an RSA public key from PEM bytes.

    Returns a cryptography public-key object, or raises ValueError if the
    PEM data is invalid.
    """
    try:
        return serialization.load_pem_public_key(pem_bytes)
    except (ValueError, TypeError) as exc:
        raise ValueError(f"Failed to load public key: {exc}") from exc


def _fetch(url: str, verify_tls) -> bytes:
    """Fetch *url* via HTTPS and return the response body as bytes.

    *verify_tls* is passed directly to requests (True / False / CA bundle path).
    """
    log.info("Fetching %s", url)
    resp = requests.get(url, timeout=30, verify=verify_tls)
    resp.raise_for_status()
    return resp.content


def _verify_manifest_signature(
    manifest_bytes: bytes,
    sig_bytes: bytes,
    public_key,
) -> None:
    """Verify the RSA-PKCS1v15/SHA-256 signature of *manifest_bytes*.

    Raises cryptography.exceptions.InvalidSignature on failure.
    """
    public_key.verify(
        sig_bytes,
        manifest_bytes,
        padding.PKCS1v15(),
        hashes.SHA256(),
    )
    log.info("Manifest signature verified OK.")


def _sha256_of_bytes(data: bytes) -> str:
    """Return the lowercase hex SHA-256 digest of *data*."""
    return hashlib.sha256(data).hexdigest()


def _read_local_version(version_file: Path) -> str:
    """Read the local version string from *version_file*.

    Returns "0" if the file does not exist (i.e. nothing is installed).
    """
    if not version_file.exists():
        log.warning("Version file %s not found; treating local version as '0'.", version_file)
        return "0"
    return version_file.read_text(encoding="utf-8").strip()


def _write_local_version(version_file: Path, version: str) -> None:
    """Write *version* to *version_file*."""
    version_file.parent.mkdir(parents=True, exist_ok=True)
    version_file.write_text(version + "\n", encoding="utf-8")
    log.info("Local version updated to %s.", version)


def _version_is_newer(remote: str, local: str) -> bool:
    """Return True when *remote* version string is newer than *local*.

    Uses a simple lexicographic comparison suitable for ISO-date or
    semver-like strings such as "2026.03.19" or "1.2.3".
    For production use a proper version-comparison library (e.g. packaging).
    """
    return remote > local


# ---------------------------------------------------------------------------
# Install / rollback helpers
# ---------------------------------------------------------------------------

def _atomic_install(tmp_dir: Path, current_dir: Path, backup_dir: Path) -> None:
    """Atomically swap *tmp_dir* into *current_dir*, keeping a backup.

    Steps:
      1. If *backup_dir* exists, remove it.
      2. If *current_dir* exists, rename it to *backup_dir*.
      3. Rename *tmp_dir* to *current_dir*.

    This is as close to atomic as we can get with stdlib; a true atomic swap
    requires an OS-level rename (POSIX rename(2) is atomic on the same
    filesystem, which shutil.move relies on when src and dst are on the same
    device).
    """
    if backup_dir.exists():
        shutil.rmtree(backup_dir)
        log.info("Removed old backup at %s.", backup_dir)

    if current_dir.exists():
        shutil.move(str(current_dir), str(backup_dir))
        log.info("Backed up previous install to %s.", backup_dir)

    shutil.move(str(tmp_dir), str(current_dir))
    log.info("Installed update to %s.", current_dir)


def _rollback(current_dir: Path, backup_dir: Path) -> None:
    """Restore *backup_dir* to *current_dir* after a failed install."""
    if not backup_dir.exists():
        log.error("No backup found at %s; cannot roll back.", backup_dir)
        return
    if current_dir.exists():
        shutil.rmtree(current_dir)
    shutil.move(str(backup_dir), str(current_dir))
    log.info("Rolled back to backup at %s.", current_dir)


# ---------------------------------------------------------------------------
# High-level update flow
# ---------------------------------------------------------------------------

def check_for_update(
    manifest_url: str,
    sig_url: str,
    public_key,
    version_file: Path,
    verify_tls,
) -> tuple[bool, dict]:
    """Return (update_available, manifest_dict).

    Raises on network or verification errors.
    """
    manifest_bytes = _fetch(manifest_url, verify_tls)
    sig_bytes = _fetch(sig_url, verify_tls)

    try:
        _verify_manifest_signature(manifest_bytes, sig_bytes, public_key)
    except InvalidSignature:
        raise RuntimeError(
            "⚠️  Manifest signature verification FAILED.  Aborting update.  "
            "This could indicate tampering or a misconfigured demo key."
        )

    manifest = json.loads(manifest_bytes.decode("utf-8"))
    remote_version = manifest.get("version", "0")
    local_version = _read_local_version(version_file)

    log.info("Remote version: %s  |  Local version: %s", remote_version, local_version)

    update_available = _version_is_newer(remote_version, local_version)
    if update_available:
        log.info("Update available: %s → %s", local_version, remote_version)
    else:
        log.info("Already up to date (local=%s, remote=%s).", local_version, remote_version)

    return update_available, manifest


def download_and_install(
    manifest: dict,
    version_file: Path,
    current_dir: Path,
    backup_dir: Path,
    verify_tls,
) -> None:
    """Download the artifact described in *manifest*, verify, and install.

    Only the first file entry is used in this demo.
    """
    files = manifest.get("files", [])
    if not files:
        raise RuntimeError("Manifest contains no file entries.")

    file_info = files[0]
    artifact_url = file_info["url"]
    expected_sha256 = file_info["sha256"]
    filename = file_info.get("filename", "update.tar.gz")

    log.info("Downloading artifact: %s", artifact_url)
    artifact_bytes = _fetch(artifact_url, verify_tls)

    # Verify checksum
    actual_sha256 = _sha256_of_bytes(artifact_bytes)
    if actual_sha256 != expected_sha256:
        raise RuntimeError(
            f"SHA-256 checksum mismatch for {filename}.\n"
            f"  expected: {expected_sha256}\n"
            f"  actual:   {actual_sha256}\n"
            "Aborting – do not install a tampered artifact."
        )
    log.info("SHA-256 checksum verified: %s", actual_sha256)

    # Write to a temporary directory, then atomically swap into place
    with tempfile.TemporaryDirectory(prefix="jitterbugs_update_") as tmp_root:
        tmp_dir = Path(tmp_root) / "install"
        tmp_dir.mkdir()
        artifact_path = tmp_dir / filename
        artifact_path.write_bytes(artifact_bytes)
        log.info("Artifact written to temporary path: %s", artifact_path)

        try:
            _atomic_install(tmp_dir, current_dir, backup_dir)
        except Exception as exc:
            log.error("Install failed: %s; attempting rollback.", exc)
            _rollback(current_dir, backup_dir)
            raise

    new_version = manifest.get("version", "unknown")
    _write_local_version(version_file, new_version)
    log.info("✅  Update to version %s installed successfully.", new_version)


# ---------------------------------------------------------------------------
# Verify existing installation
# ---------------------------------------------------------------------------

def verify_installation(manifest: dict, current_dir: Path, verify_tls) -> bool:
    """Re-download artifact and compare SHA-256 against installed file.

    Returns True if verification passes, False otherwise.

    ⚠️  This re-downloads the artifact to compare; for large files consider
    storing the expected checksum locally instead.
    """
    files = manifest.get("files", [])
    if not files:
        log.warning("Manifest has no file entries; nothing to verify.")
        return True

    file_info = files[0]
    filename = file_info.get("filename", "update.tar.gz")
    expected_sha256 = file_info["sha256"]

    installed_path = current_dir / filename
    if not installed_path.exists():
        log.warning("Installed file %s not found.", installed_path)
        return False

    actual_sha256 = _sha256_of_bytes(installed_path.read_bytes())
    if actual_sha256 != expected_sha256:
        log.error(
            "Integrity check FAILED for %s.\n  expected: %s\n  actual:   %s",
            installed_path, expected_sha256, actual_sha256,
        )
        return False

    log.info("Integrity check passed for %s.", installed_path)
    return True


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def _build_arg_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="updater",
        description=(
            "jitterbugs auto-updater prototype (DEMO – NOT FOR PRODUCTION).\n"
            "Fetches a signed manifest over HTTPS, verifies the RSA signature,\n"
            "checks for a newer version, and optionally downloads + installs it."
        ),
    )
    parser.add_argument(
        "--check-only",
        action="store_true",
        help="Check whether an update is available but do not install.",
    )
    parser.add_argument(
        "--auto",
        action="store_true",
        help="Download and install without prompting (DEMO – use with care).",
    )
    parser.add_argument(
        "--manifest-url",
        default=DEFAULT_MANIFEST_URL,
        metavar="URL",
        help=f"Manifest URL (default: {DEFAULT_MANIFEST_URL}).",
    )
    parser.add_argument(
        "--sig-url",
        default=DEFAULT_SIG_URL,
        metavar="URL",
        help=f"Detached signature URL (default: {DEFAULT_SIG_URL}).",
    )
    parser.add_argument(
        "--verify",
        action="store_true",
        help="Verify the integrity of the current installation and exit.",
    )
    parser.add_argument(
        "--version-file",
        default=str(DEFAULT_VERSION_FILE),
        metavar="PATH",
        help=f"Local version file (default: {DEFAULT_VERSION_FILE}).",
    )
    parser.add_argument(
        "--public-key-file",
        default=None,
        metavar="PATH",
        help="Path to a PEM public key file (overrides the embedded demo key).",
    )
    parser.add_argument(
        "--no-verify-tls",
        action="store_true",
        help=(
            "Disable TLS certificate verification (INSECURE – for local testing only).\n"
            "Pass a CA bundle path via REQUESTS_CA_BUNDLE env var for self-signed certs."
        ),
    )
    return parser


def main(argv=None) -> int:  # noqa: C901 – complexity acceptable for a demo
    """Entry-point for the updater CLI.  Returns an exit code."""
    parser = _build_arg_parser()
    args = parser.parse_args(argv)

    # Attach file log handler
    _add_file_log_handler(DEFAULT_LOG_FILE)

    log.info("=== jitterbugs updater prototype (DEMO) ===")
    log.warning(
        "This is a DEMO updater.  The embedded public key is a placeholder "
        "and signature verification will fail unless you replace it with your "
        "real key (see --public-key-file or examples/README_UPDATER.md)."
    )

    # Load public key
    if args.public_key_file:
        pem_bytes = Path(args.public_key_file).read_bytes()
        log.info("Using public key from file: %s", args.public_key_file)
    else:
        pem_bytes = EMBEDDED_PUBLIC_KEY_PEM
        log.warning("Using embedded DEMO public key – replace for production use.")

    try:
        public_key = _load_public_key(pem_bytes)
    except ValueError as exc:
        log.error("Cannot load public key: %s", exc)
        return 1

    version_file = Path(args.version_file)
    current_dir = DEFAULT_CURRENT_DIR
    backup_dir = DEFAULT_BACKUP_DIR
    verify_tls = not args.no_verify_tls

    if not verify_tls:
        log.warning("TLS verification DISABLED – for local testing only.")

    # --verify mode: check installed file integrity
    if args.verify:
        log.info("Verify mode: fetching manifest to compare checksums.")
        try:
            _, manifest = check_for_update(
                args.manifest_url, args.sig_url, public_key, version_file, verify_tls
            )
        except Exception as exc:
            log.error("Failed to fetch/verify manifest: %s", exc)
            return 1
        ok = verify_installation(manifest, current_dir, verify_tls)
        return 0 if ok else 1

    # Normal check / install flow
    try:
        update_available, manifest = check_for_update(
            args.manifest_url, args.sig_url, public_key, version_file, verify_tls
        )
    except Exception as exc:
        log.error("Update check failed: %s", exc)
        return 1

    if not update_available:
        return 0

    if args.check_only:
        log.info("--check-only: not installing.  New version: %s", manifest.get("version"))
        return 0

    if not args.auto:
        # Interactive confirmation (not required in --auto mode)
        try:
            answer = input(
                f"Install version {manifest.get('version')}? [y/N] "
            ).strip().lower()
        except (EOFError, KeyboardInterrupt):
            print()
            log.info("Update cancelled by user.")
            return 0
        if answer not in ("y", "yes"):
            log.info("Update declined by user.")
            return 0

    try:
        download_and_install(manifest, version_file, current_dir, backup_dir, verify_tls)
    except Exception as exc:
        log.error("Update installation failed: %s", exc)
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
