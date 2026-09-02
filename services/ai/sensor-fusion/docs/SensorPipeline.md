# Sensor Pipeline

## Purpose

The future sensor pipeline will normalize sensor data and produce a unified world state for downstream Horizon Core modules.

## Planned Flow

```text
Camera / IMU / GPS / LiDAR / Depth / Barometer / Compass
  -> SensorFrame
  -> NoiseFilter
  -> IMUFusion
  -> GPSFusion
  -> VIOEstimator
  -> StateEstimator
  -> SensorFusionEngine
  -> WorldState
```

## Step 3 Boundary

Step 3 defines only contracts and placeholders. It does not include filtering mathematics, SLAM, EKF, Kalman filters, or navigation decisions.
