#!/usr/bin/env python3
"""
lpr_demo.py — Conservative License Plate Recognition Demo for jitterbugs
=========================================================================

PURPOSE
-------
This script demonstrates the local LPR pipeline used by jitterbugs:
  1. Detect plate bounding boxes in an input image (YOLO detector).
  2. Crop the detected region and run OCR (EasyOCR).
  3. Normalise the plate text.
  4. Compute plate_hash = sha256(normalised_plate + SITE_SALT).
  5. Emit a plate_event JSON object matching schemas/plate_event.json.

This script is intended for LOCAL TESTING AND CI ONLY.
It does NOT perform any external lookups (DMV, law-enforcement, etc.).
All plate events produced require human review before any further action.

LEGAL WARNING
-------------
  ⚠️  License plate recognition is subject to complex legal requirements that
  vary by jurisdiction.  Do NOT use this script in a production environment
  without first obtaining qualified legal advice.  Automated enforcement is
  NOT implemented here and must NEVER be added without legal sign-off.
  See docs/COMPLIANCE/PLATE_RECOGNITION.md for the full policy.

DEPENDENCIES
------------
  pip install ultralytics easyocr pillow

  - ultralytics  : YOLOv8 object detector (pip install ultralytics)
  - easyocr      : OCR engine (pip install easyocr)
  - pillow       : image loading (pip install pillow)

  If either ultralytics or easyocr is not installed the script falls back to
  stub implementations so that CI can validate the pipeline logic without GPU
  dependencies.

USAGE
-----
  # Minimal — uses stub detector/OCR (no GPU required, suitable for CI):
  python tooling/lpr_demo.py

  # With a real image:
  python tooling/lpr_demo.py --image /path/to/frame.jpg

  # Override the site salt (never commit real salts):
  SITE_SALT=mysecret python tooling/lpr_demo.py --image /path/to/frame.jpg

OUTPUT
------
  JSON printed to stdout matching schemas/plate_event.json.
  plate_text is always null (plain plate text not stored by default).
  plate_hash is sha256(normalised_plate + SITE_SALT).
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import uuid
from datetime import datetime, timezone
from typing import Any

# ---------------------------------------------------------------------------
# Optional heavy dependencies — gracefully degraded for CI environments
# ---------------------------------------------------------------------------

try:
    from ultralytics import YOLO as _YOLO  # type: ignore

    _YOLO_AVAILABLE = True
except ImportError:  # pragma: no cover
    _YOLO_AVAILABLE = False

try:
    import easyocr  # type: ignore

    _EASYOCR_AVAILABLE = True
except ImportError:  # pragma: no cover
    _EASYOCR_AVAILABLE = False

try:
    from PIL import Image as _PILImage  # type: ignore

    _PIL_AVAILABLE = True
except ImportError:  # pragma: no cover
    _PIL_AVAILABLE = False

try:
    import numpy as _np  # type: ignore

    _NUMPY_AVAILABLE = True
except ImportError:  # pragma: no cover
    _NUMPY_AVAILABLE = False


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

# The site salt MUST be provided via the SITE_SALT environment variable in any
# real deployment.  The placeholder value below is intentionally not a secret
# and is used ONLY in the demo/CI context so that tests produce deterministic
# hashes without requiring secret injection.
DEMO_SITE_SALT_PLACEHOLDER = "REPLACE_WITH_SECRET_SITE_SALT"

# Model identifiers used in plate_event.model_version.
YOLO_MODEL_ID = "yolov8n-lp-stub-v0"
OCR_MODEL_ID = "easyocr-stub-v0"

# Minimum confidence thresholds (mirrors plate_policy.yaml defaults).
MIN_DETECTION_CONFIDENCE = 0.60
MIN_PLATE_CONFIDENCE = 0.70


# ---------------------------------------------------------------------------
# Stub implementations (used when real models are not installed)
# ---------------------------------------------------------------------------


class _StubDetection:
    """Minimal stand-in for a YOLO detection result."""

    def __init__(self) -> None:
        # Simulate a single plate detection near the top-left of a 640×480 frame.
        self.boxes = _StubBoxes()
        self.conf = [0.85]


class _StubBoxes:
    def xyxy(self) -> list[list[float]]:
        return [[50.0, 30.0, 250.0, 100.0]]

    def __iter__(self):  # pragma: no cover
        yield [50.0, 30.0, 250.0, 100.0]


def _stub_detect(image_path: str) -> list[dict[str, Any]]:
    """Return a fake detection result for CI/demo when YOLO is not installed."""
    return [
        {
            "box": {"x": 50, "y": 30, "width": 200, "height": 70},
            "detection_confidence": 0.85,
            "crop": None,  # No real crop without PIL
        }
    ]


def _stub_ocr(crop: Any) -> tuple[str, float]:
    """Return a fake OCR result for CI/demo when EasyOCR is not installed."""
    return "ABC1234", 0.92


# ---------------------------------------------------------------------------
# Real implementations (used when models are installed)
# ---------------------------------------------------------------------------


def _yolo_detect(image_path: str, model_path: str = "yolov8n.pt") -> list[dict[str, Any]]:
    """
    Run YOLO detection on image_path.

    NOTE: In a real deployment use a YOLO model fine-tuned for licence plates
    (e.g. trained on UFPR-ALPR or OpenALPR datasets).  The default yolov8n.pt
    detects generic objects; it is used here only as a demonstration.
    """
    if not _YOLO_AVAILABLE:
        return _stub_detect(image_path)

    model = _YOLO(model_path)  # type: ignore[call-arg]
    results = model(image_path, verbose=False)

    detections: list[dict[str, Any]] = []
    if not _PIL_AVAILABLE:
        return _stub_detect(image_path)

    image = _PILImage.open(image_path).convert("RGB")

    for result in results:
        for i, box in enumerate(result.boxes.xyxy.tolist()):
            x1, y1, x2, y2 = (int(v) for v in box)
            conf = float(result.boxes.conf[i])
            if conf < MIN_DETECTION_CONFIDENCE:
                continue
            crop = image.crop((x1, y1, x2, y2))
            detections.append(
                {
                    "box": {
                        "x": x1,
                        "y": y1,
                        "width": x2 - x1,
                        "height": y2 - y1,
                    },
                    "detection_confidence": conf,
                    "crop": crop,
                }
            )
    return detections


def _easyocr_read(crop: Any) -> tuple[str, float]:
    """
    Run EasyOCR on a PIL Image crop.

    Returns (normalised_text, confidence).  Returns ('', 0.0) if nothing
    is detected above MIN_PLATE_CONFIDENCE.
    """
    if not _EASYOCR_AVAILABLE or crop is None:
        return _stub_ocr(crop)

    reader = easyocr.Reader(["en"], gpu=False, verbose=False)  # type: ignore[call-arg]
    if not _NUMPY_AVAILABLE:
        return _stub_ocr(crop)
    results = reader.readtext(_np.array(crop))
    if not results:
        return "", 0.0

    # Pick the highest-confidence result.
    best = max(results, key=lambda r: r[2])
    text: str = best[1]
    confidence: float = float(best[2])

    if confidence < MIN_PLATE_CONFIDENCE:
        return "", confidence

    return normalise_plate(text), confidence


# ---------------------------------------------------------------------------
# Plate text normalisation
# ---------------------------------------------------------------------------


def normalise_plate(raw: str) -> str:
    """
    Normalise a raw OCR plate string.

    Rules:
    - Strip leading/trailing whitespace.
    - Uppercase.
    - Remove internal whitespace.
    - Remove common OCR noise characters: . - _ /
    """
    cleaned = raw.strip().upper()
    for ch in (" ", "\t", ".", "-", "_", "/"):
        cleaned = cleaned.replace(ch, "")
    return cleaned


# ---------------------------------------------------------------------------
# Plate hash
# ---------------------------------------------------------------------------


def compute_plate_hash(normalised_plate: str, site_salt: str) -> str:
    """
    Return sha256(normalised_plate + site_salt) as a lowercase hex string.

    The site_salt MUST be a secret value stored outside source control.
    In the demo/CI context the placeholder salt is used.
    """
    payload = (normalised_plate + site_salt).encode("utf-8")
    return hashlib.sha256(payload).hexdigest()


# ---------------------------------------------------------------------------
# plate_event assembly
# ---------------------------------------------------------------------------


def build_plate_event(
    device_id: str,
    box: dict[str, int],
    plate_text: str,
    plate_confidence: float,
    plate_hash: str,
    model_version: str,
    clip_url: str | None = None,
    notes: str | None = None,
) -> dict[str, Any]:
    """
    Assemble a plate_event dict that conforms to schemas/plate_event.json.

    plate_text is always set to null here because store_plain_plate_text=false
    is the default policy.  If your site policy explicitly permits storing
    plain plate text, pass it via the plate_text parameter — but never enable
    this without legal review.
    """
    return {
        "event_id": str(uuid.uuid4()),
        "device_id": device_id,
        "timestamp_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "location": {
            # Demo uses static placeholder coordinates.
            # In production, supply real device GPS/network coordinates.
            "lat": 0.0,
            "lon": 0.0,
            "accuracy_m": 9999.0,
            "source": "static_config",
        },
        # Plain plate text is NOT stored by default (privacy policy).
        # Change to plate_text if store_plain_plate_text policy is enabled.
        "plate_text": None,
        "plate_confidence": round(plate_confidence, 4),
        "plate_hash": plate_hash,
        "bounding_box": box,
        "clip_url": clip_url,
        "model_version": model_version,
        # Demo uses a placeholder manifest reference.
        "device_manifest_ref": "demo-manifest-ref-not-verified",
        # Demo uses a placeholder signature.  In production this must be a
        # real ed25519/ECDSA signature from the device private key.
        "signed_by_device": "DEMO_SIGNATURE_PLACEHOLDER_NOT_VALID",
        # No external lookup performed in demo.
        "match_source": None,
        "match_confidence": None,
        "notes": notes,
    }


# ---------------------------------------------------------------------------
# Main entry point
# ---------------------------------------------------------------------------


def run_demo(image_path: str | None, site_salt: str, device_id: str) -> list[dict[str, Any]]:
    """
    Run the full LPR demo pipeline on image_path.

    Returns a list of plate_event dicts (one per detected plate).
    """
    if image_path is not None:
        detections = _yolo_detect(image_path)
    else:
        # No image provided — use stub detector for CI/demo.
        detections = _stub_detect("")

    model_version = (
        f"{YOLO_MODEL_ID}/{OCR_MODEL_ID}"
        if not (_YOLO_AVAILABLE and _EASYOCR_AVAILABLE)
        else "yolov8n/{ocr_ver}".format(
            ocr_ver=easyocr.__version__ if _EASYOCR_AVAILABLE else OCR_MODEL_ID  # type: ignore[attr-defined]
        )
    )

    events: list[dict[str, Any]] = []

    for det in detections:
        crop = det.get("crop")
        raw_text, ocr_conf = _easyocr_read(crop) if image_path else _stub_ocr(crop)

        if not raw_text:
            # OCR returned nothing useful — skip this detection.
            continue

        normalised = normalise_plate(raw_text)
        ph = compute_plate_hash(normalised, site_salt)

        event = build_plate_event(
            device_id=device_id,
            box=det["box"],
            plate_text=normalised,
            plate_confidence=ocr_conf,
            plate_hash=ph,
            model_version=model_version,
        )
        events.append(event)

    return events


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="jitterbugs LPR demo — local pipeline only, no external lookups."
    )
    parser.add_argument(
        "--image",
        metavar="PATH",
        default=None,
        help="Path to an input image. If omitted, stub detection is used (suitable for CI).",
    )
    parser.add_argument(
        "--device-id",
        metavar="ID",
        default="demo-device-001",
        help="Device identifier to embed in plate_event (default: demo-device-001).",
    )
    parser.add_argument(
        "--pretty",
        action="store_true",
        default=False,
        help="Pretty-print JSON output (default: false).",
    )
    args = parser.parse_args(argv)

    # Resolve site salt from environment; fall back to demo placeholder.
    site_salt = os.environ.get("SITE_SALT", DEMO_SITE_SALT_PLACEHOLDER)
    if site_salt == DEMO_SITE_SALT_PLACEHOLDER:
        print(
            "WARNING: Using demo placeholder SITE_SALT. "
            "Set SITE_SALT environment variable in production.",
            file=sys.stderr,
        )

    events = run_demo(
        image_path=args.image,
        site_salt=site_salt,
        device_id=args.device_id,
    )

    if not events:
        print("No plate events detected.", file=sys.stderr)
        return 0

    if args.pretty:
        print(json.dumps(events, indent=2))
    else:
        print(json.dumps(events, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    sys.exit(main())
