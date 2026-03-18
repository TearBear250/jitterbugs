import cv2
import numpy as np
import pyaudio
from fer import FER
import time

# Initialize the video capture and necessary variables
video_capture = cv2.VideoCapture(0)
emotion_detector = FER()
score_threshold = 0.5

# Initialise PyAudio once outside the loop to avoid repeated resource allocation.
audio_input = pyaudio.PyAudio()

try:
    # Main loop to monitor for jitter
    while True:
        # Capture frame-by-frame
        ret, frame = video_capture.read()
        if not ret:
            break

        # Get the emotion predictions
        emotions = emotion_detector.detect_emotions(frame)
        if emotions:
            emotion_scores = emotions[0]['emotions']
            above_threshold = [v for v in emotion_scores.values() if v >= score_threshold]
            jitter_score = float(np.mean(above_threshold)) if above_threshold else 0
        else:
            jitter_score = 0

        # Here you could add audio variance calculations
        # Capture audio logic goes here...

        # Display the resulting frame with the jitter score
        cv2.putText(frame, f'Jitter Score: {jitter_score:.2f}', (10, 50), cv2.FONT_HERSHEY_SIMPLEX, 1, (255, 255, 255), 2)
        cv2.imshow('Jitter Watch', frame)

        # Break the loop if 'q' is pressed
        if cv2.waitKey(1) & 0xFF == ord('q'):
            break
finally:
    # Release all resources on exit, whether normal or via exception.
    audio_input.terminate()
    video_capture.release()
    cv2.destroyAllWindows()