"""Entry point: run the Jitterbugs overlay on camera index 0."""

import os
import sys

# Allow running this file directly from the repo root without installing the
# package – insert the src/ tree onto the path before importing.
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "src"))

from jitterbugs.app import run  # noqa: E402

if __name__ == "__main__":
    run(camera_index=0)