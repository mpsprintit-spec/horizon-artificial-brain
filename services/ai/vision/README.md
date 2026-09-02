# Vision AI

## Purpose

Vision AI is the HC-002 Step 2 module responsible for camera, image, and video perception contracts. It prepares the repository structure for future object detection, scene understanding, depth estimation, optical flow tracking, and low-light enhancement work.

This module is scaffold-only in Step 2. It does not include machine learning models, inference logic, OpenCV, TensorFlow, PyTorch, YOLO, or business logic.

## Responsibilities

- Define Vision AI domain models and contracts.
- Define frame and stream analysis interfaces.
- Define object detection, scene understanding, depth estimation, optical flow tracking, and low-light enhancement interfaces.
- Define repository, service, handler, HTTP, gRPC, and event boundaries.
- Keep Vision AI independent from Sensor Fusion AI, Decision AI, Drone AI, HUD AI, and Learning Engine implementation details.

## Inputs

- `Frame`: a single image frame from glasses, drone, or another camera source.
- `VideoStream`: an ordered video stream with source and mission metadata.
- `FrameCaptured`: an event indicating that a frame is available for analysis.
- `VideoFrameReceived`: an event indicating that a stream frame is available for analysis.

## Outputs

- `VisionResult`: the aggregate result envelope for frame or stream analysis.
- `Detection`: object, person, target, or obstacle detection metadata.
- `SceneContext`: semantic scene interpretation metadata.
- `DepthMap`: depth estimation metadata or an external reference to depth output.
- `OpticalFlow`: motion tracking metadata between frames.
- Domain events such as `FrameAnalyzed`, `ObjectDetected`, `SceneUnderstood`, `DepthEstimated`, and `OpticalFlowTracked`.

## Dependencies

- Media input events and media references.
- Device source metadata.
- Optional mission context metadata.

Vision AI must not call Decision AI directly and must not depend on later HC-002 roadmap steps.

## Public Interfaces

- `FrameAnalyzer`
- `StreamAnalyzer`
- `ObjectDetector`
- `SceneUnderstanding`
- `DepthEstimator`
- `OpticalFlowTracker`
- `LowLightEnhancer`

## Event Flow

```text
FrameCaptured / VideoFrameReceived
  -> Vision AI handlers
  -> Vision AI service contracts
  -> VisionResult and specialized outputs
  -> FrameAnalyzed / ObjectDetected / SceneUnderstood / DepthEstimated / OpticalFlowTracked
```

## Package Layout

```text
services/ai/vision/
  README.md
  main.go
  api/
    grpc/
    http/
  internal/
    event/
    handler/
    model/
    repository/
    service/
  test/
```
