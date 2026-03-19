# examples/server.py
#
# Minimal aiohttp HTTPS server that accepts POST /jitter requests and logs the
# submitted jitter score.
#
# This file is provided as a self-contained development/testing example.
# It is NOT intended for production deployment as-is.
#
# Usage:
#   1. Generate test certificates (from the repo root):
#        bash scripts/make-certs.sh
#   2. Install dependencies:
#        pip install "aiohttp>=3.8"
#   3. Run the server:
#        python examples/server.py
#
# The server listens on https://localhost:8443 by default.
#
# To enable mutual TLS (mTLS), see the commented block near the bottom of
# this file.
#
# SPDX-License-Identifier: MIT (same as repository root)

from __future__ import annotations

import json
import logging
import ssl
import sys
from pathlib import Path

from aiohttp import web

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s  %(levelname)-8s  %(message)s",
)
log = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Certificate paths (relative to the repo root)
# ---------------------------------------------------------------------------
_REPO_ROOT = Path(__file__).resolve().parent.parent
_CERT_DIR = _REPO_ROOT / "certs"
_SERVER_CERT = _CERT_DIR / "server.crt"
_SERVER_KEY = _CERT_DIR / "server.key"
_CA_CERT = _CERT_DIR / "ca.crt"

# ---------------------------------------------------------------------------
# Route handlers
# ---------------------------------------------------------------------------

async def post_jitter(request: web.Request) -> web.Response:
    """Accept a JSON body ``{"score": <float>}`` and acknowledge receipt."""
    try:
        body = await request.json()
    except (json.JSONDecodeError, Exception) as exc:
        raise web.HTTPBadRequest(reason=f"Invalid JSON: {exc}") from exc

    if "score" not in body:
        raise web.HTTPUnprocessableEntity(reason='Missing required field "score"')

    score = body["score"]
    try:
        score = float(score)
    except (TypeError, ValueError) as exc:
        raise web.HTTPUnprocessableEntity(
            reason=f'"score" must be a number, got {score!r}'
        ) from exc

    log.info("Received jitter score: %.4f", score)

    return web.json_response(
        {"status": "ok", "received_score": score},
        status=200,
    )


# ---------------------------------------------------------------------------
# TLS / SSL context
# ---------------------------------------------------------------------------

def _build_ssl_context() -> ssl.SSLContext:
    """
    Return an SSLContext with TLS 1.0 and 1.1 disabled (TLS 1.2 minimum,
    TLS 1.3 preferred when both peers support it).

    To enable mutual TLS (mTLS) – i.e. require the client to present a
    certificate signed by the test CA – uncomment the two lines marked
    ``# mTLS`` below.
    """
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)

    # Disable legacy protocol versions.
    ctx.minimum_version = ssl.TLSVersion.TLSv1_2  # prefer 1.3, allow 1.2 fallback
    ctx.maximum_version = ssl.TLSVersion.MAXIMUM_SUPPORTED

    # Load server identity.
    ctx.load_cert_chain(certfile=str(_SERVER_CERT), keyfile=str(_SERVER_KEY))

    # --- mTLS: uncomment the following two lines to require client certificates ---
    # ctx.verify_mode = ssl.CERT_REQUIRED                       # mTLS
    # ctx.load_verify_locations(cafile=str(_CA_CERT))           # mTLS
    # ------------------------------------------------------------------------------

    return ctx


# ---------------------------------------------------------------------------
# Application factory
# ---------------------------------------------------------------------------

def create_app() -> web.Application:
    app = web.Application()
    app.router.add_post("/jitter", post_jitter)
    return app


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main() -> None:
    for path in (_SERVER_CERT, _SERVER_KEY):
        if not path.exists():
            log.error(
                "Certificate file not found: %s\n"
                "Run 'bash scripts/make-certs.sh' first.",
                path,
            )
            sys.exit(1)

    ssl_ctx = _build_ssl_context()
    app = create_app()

    log.info("Starting HTTPS server on https://localhost:8443")
    log.info("POST https://localhost:8443/jitter  {'score': <float>}")
    web.run_app(app, host="localhost", port=8443, ssl_context=ssl_ctx)


if __name__ == "__main__":
    main()
