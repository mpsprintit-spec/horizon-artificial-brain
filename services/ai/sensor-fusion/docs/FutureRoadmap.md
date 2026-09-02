# Future Roadmap

## Step 3 Completion Criteria

- Sensor Fusion AI folder structure exists.
- Contracts are documented.
- Domain model placeholders exist.
- Domain event placeholders exist.
- Architecture and pipeline documentation exist.
- Existing HC-001, Step 1, and Step 2 code remains unchanged.

## Later Work

1. Add adapter contracts for camera, IMU, GPS, LiDAR, depth, barometer, and compass inputs.
2. Implement noise filtering behind the `NoiseFilter` contract.
3. Implement IMU and GPS fusion behind fusion contracts.
4. Implement VIO and state estimation behind estimator contracts.
5. Publish Sensor Fusion AI events to downstream modules.
6. Integrate with Decision AI only when HC-002 Step 4 begins.
