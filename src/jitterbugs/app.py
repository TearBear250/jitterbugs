"""
jitterbugs.app
==============
Consent-based, real-time audio + video feature extraction with an on-screen
overlay.

Features visualised
-------------------
* **Pose** – shoulder tilt angle (MediaPipe Pose)
* **Face** – gaze proxy (iris horizontal offset) and eye-openness ratio
  (MediaPipe FaceMesh)
* **Audio** – fundamental frequency (pitch), a jitter proxy (cycle-to-cycle
  F0 variation), and RMS energy (PyAudio + librosa)

Usage
-----
Press ``q`` to quit.

All processing is local; nothing is written to disk or transmitted.
"""

from __future__ import annotations

import math
import queue
import threading
from typing import Optional

import cv2
import librosa
import mediapipe as mp
import numpy as np
import pyaudio

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
_SAMPLE_RATE = 22050
_AUDIO_CHUNK = 2048          # frames per PyAudio buffer
_AUDIO_FORMAT = pyaudio.paInt16
_CHANNELS = 1

# MediaPipe landmark indices used for face features
_LEFT_IRIS_CENTER = 468
_RIGHT_IRIS_CENTER = 473
_LEFT_EYE_TOP = 159
_LEFT_EYE_BOTTOM = 145
_LEFT_EYE_INNER = 133
_LEFT_EYE_OUTER = 33
_RIGHT_EYE_TOP = 386
_RIGHT_EYE_BOTTOM = 374
_RIGHT_EYE_INNER = 362
_RIGHT_EYE_OUTER = 263

# MediaPipe pose landmark indices for shoulders
_LEFT_SHOULDER = 11
_RIGHT_SHOULDER = 12


# ---------------------------------------------------------------------------
# Audio worker
# ---------------------------------------------------------------------------

class _AudioWorker:
    """Continuously capture mic audio and compute pitch/jitter/RMS features."""

    def __init__(self, sample_rate: int = _SAMPLE_RATE,
                 chunk: int = _AUDIO_CHUNK) -> None:
        self._sr = sample_rate
        self._chunk = chunk
        self._q: queue.Queue[np.ndarray] = queue.Queue(maxsize=4)
        self._running = False
        self._thread: Optional[threading.Thread] = None
        self._pa: Optional[pyaudio.PyAudio] = None
        self._stream = None

        # Latest computed features (thread-safe via GIL for simple floats)
        self.pitch_hz: float = 0.0
        self.jitter_pct: float = 0.0
        self.rms_db: float = -60.0

    # ------------------------------------------------------------------
    def start(self) -> None:
        """Open the default input device and start the capture thread."""
        self._pa = pyaudio.PyAudio()
        self._stream = self._pa.open(
            format=_AUDIO_FORMAT,
            channels=_CHANNELS,
            rate=self._sr,
            input=True,
            frames_per_buffer=self._chunk,
            stream_callback=self._callback,
        )
        self._running = True
        self._thread = threading.Thread(target=self._process_loop,
                                        daemon=True, name="audio-worker")
        self._thread.start()

    def stop(self) -> None:
        """Stop capture and release resources."""
        self._running = False
        if self._stream is not None:
            self._stream.stop_stream()
            self._stream.close()
        if self._pa is not None:
            self._pa.terminate()

    # ------------------------------------------------------------------
    def _callback(self, in_data, frame_count, time_info, status):
        """PyAudio callback: enqueue raw frames for the processing thread."""
        try:
            audio = np.frombuffer(in_data, dtype=np.int16).astype(np.float32)
            audio /= 32768.0  # normalise to [-1, 1]
            self._q.put_nowait(audio)
        except (queue.Full, ValueError, TypeError):
            pass  # drop frame on overflow or malformed data
        return (None, pyaudio.paContinue)

    def _process_loop(self) -> None:
        """Background thread: compute audio features from each chunk."""
        while self._running:
            try:
                audio = self._q.get(timeout=0.5)
            except queue.Empty:
                continue

            # RMS energy in dB
            rms = float(np.sqrt(np.mean(audio ** 2)))
            self.rms_db = 20.0 * math.log10(rms + 1e-9)

            # Fundamental frequency via librosa pyin
            f0, voiced_flag, _ = librosa.pyin(
                audio,
                fmin=librosa.note_to_hz("C2"),
                fmax=librosa.note_to_hz("C7"),
                sr=self._sr,
            )
            voiced = f0[voiced_flag] if voiced_flag is not None else np.array([])
            voiced = voiced[~np.isnan(voiced)]  # discard NaN frames
            if voiced.size >= 2:
                self.pitch_hz = float(np.median(voiced))
                # Jitter proxy: mean absolute cycle-to-cycle difference / mean
                diffs = np.abs(np.diff(voiced))
                self.jitter_pct = float(np.mean(diffs) / (np.mean(voiced) + 1e-9) * 100)
            elif voiced.size == 1:
                self.pitch_hz = float(voiced[0])
                self.jitter_pct = 0.0
            else:
                self.pitch_hz = 0.0
                self.jitter_pct = 0.0


# ---------------------------------------------------------------------------
# Video / overlay helpers
# ---------------------------------------------------------------------------

def _shoulder_tilt_deg(landmarks, img_w: int, img_h: int) -> float:
    """Return signed tilt angle (degrees) between left and right shoulders."""
    lm = landmarks.landmark
    lx = lm[_LEFT_SHOULDER].x * img_w
    ly = lm[_LEFT_SHOULDER].y * img_h
    rx = lm[_RIGHT_SHOULDER].x * img_w
    ry = lm[_RIGHT_SHOULDER].y * img_h
    dx = rx - lx
    dy = ry - ly
    return math.degrees(math.atan2(dy, dx))


def _eye_openness(face_lm, img_w: int, img_h: int) -> float:
    """Return a rough eye-openness ratio for the left eye (0 = closed)."""
    lm = face_lm.landmark
    top = np.array([lm[_LEFT_EYE_TOP].x * img_w, lm[_LEFT_EYE_TOP].y * img_h])
    bot = np.array([lm[_LEFT_EYE_BOTTOM].x * img_w,
                    lm[_LEFT_EYE_BOTTOM].y * img_h])
    inn = np.array([lm[_LEFT_EYE_INNER].x * img_w,
                    lm[_LEFT_EYE_INNER].y * img_h])
    out = np.array([lm[_LEFT_EYE_OUTER].x * img_w,
                    lm[_LEFT_EYE_OUTER].y * img_h])
    vertical = float(np.linalg.norm(top - bot))
    horizontal = float(np.linalg.norm(inn - out))
    return vertical / (horizontal + 1e-6)


def _gaze_offset(face_lm, img_w: int, img_h: int) -> float:
    """
    Return normalised horizontal iris offset: negative = left, positive = right.
    """
    lm = face_lm.landmark
    iris_x = lm[_LEFT_IRIS_CENTER].x * img_w
    inner_x = lm[_LEFT_EYE_INNER].x * img_w
    outer_x = lm[_LEFT_EYE_OUTER].x * img_w
    eye_width = abs(inner_x - outer_x) + 1e-6
    center_x = (inner_x + outer_x) / 2.0
    return (iris_x - center_x) / eye_width


def _draw_overlay(frame: np.ndarray,
                  tilt: Optional[float],
                  openness: Optional[float],
                  gaze: Optional[float],
                  pitch: float,
                  jitter: float,
                  rms: float) -> None:
    """Draw feature text overlay onto *frame* in-place."""
    lines = []
    if tilt is not None:
        lines.append(f"Shoulder tilt : {tilt:+.1f} deg")
    if openness is not None:
        lines.append(f"Eye openness  : {openness:.2f}")
    if gaze is not None:
        lines.append(f"Gaze offset   : {gaze:+.2f}")
    if pitch > 0:
        lines.append(f"Pitch         : {pitch:.0f} Hz")
    lines.append(f"Jitter proxy  : {jitter:.1f} %")
    lines.append(f"RMS energy    : {rms:.1f} dB")

    x0, y0 = 12, 30
    font = cv2.FONT_HERSHEY_SIMPLEX
    scale, thickness = 0.6, 1
    line_height = 26

    for i, text in enumerate(lines):
        y = y0 + i * line_height
        # Shadow
        cv2.putText(frame, text, (x0 + 1, y + 1), font,
                    scale, (0, 0, 0), thickness + 1, cv2.LINE_AA)
        # Foreground
        cv2.putText(frame, text, (x0, y), font,
                    scale, (0, 255, 120), thickness, cv2.LINE_AA)


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def run(camera_index: int = 0) -> None:
    """
    Open *camera_index*, start mic capture, and display the feature overlay.

    Parameters
    ----------
    camera_index:
        Index of the webcam to open (default 0).

    Press ``q`` to quit.
    """
    # -- Webcam --
    cap = cv2.VideoCapture(camera_index)
    if not cap.isOpened():
        raise RuntimeError(
            f"Cannot open camera index {camera_index}. "
            "Make sure a webcam is connected and not in use by another app."
        )

    # -- Audio --
    audio_worker = _AudioWorker()
    try:
        audio_worker.start()
        audio_ok = True
    except Exception as exc:  # noqa: BLE001 – PyAudio may fail in headless envs
        import sys
        print(
            f"[jitterbugs] Warning: audio capture unavailable ({exc}). "
            "Audio features will show defaults.",
            file=sys.stderr,
        )
        audio_ok = False

    # -- MediaPipe --
    mp_pose = mp.solutions.pose
    mp_face = mp.solutions.face_mesh

    pose = mp_pose.Pose(
        static_image_mode=False,
        model_complexity=1,
        min_detection_confidence=0.5,
        min_tracking_confidence=0.5,
    )
    face_mesh = mp_face.FaceMesh(
        static_image_mode=False,
        max_num_faces=1,
        refine_landmarks=True,      # required for iris landmarks (468+)
        min_detection_confidence=0.5,
        min_tracking_confidence=0.5,
    )

    try:
        while True:
            ret, frame = cap.read()
            if not ret:
                break

            h, w = frame.shape[:2]
            rgb = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)

            # -- Pose --
            tilt: Optional[float] = None
            pose_result = pose.process(rgb)
            if pose_result.pose_landmarks:
                tilt = _shoulder_tilt_deg(pose_result.pose_landmarks, w, h)

            # -- Face mesh --
            openness: Optional[float] = None
            gaze: Optional[float] = None
            face_result = face_mesh.process(rgb)
            if face_result.multi_face_landmarks:
                fl = face_result.multi_face_landmarks[0]
                openness = _eye_openness(fl, w, h)
                gaze = _gaze_offset(fl, w, h)

            # -- Audio features --
            pitch = audio_worker.pitch_hz if audio_ok else 0.0
            jitter = audio_worker.jitter_pct if audio_ok else 0.0
            rms = audio_worker.rms_db if audio_ok else -60.0

            _draw_overlay(frame, tilt, openness, gaze, pitch, jitter, rms)

            cv2.imshow("Jitterbugs – press q to quit", frame)
            if cv2.waitKey(1) & 0xFF == ord("q"):
                break
    finally:
        cap.release()
        cv2.destroyAllWindows()
        if audio_ok:
            audio_worker.stop()
        pose.close()
        face_mesh.close()
