**1.1.20**
- Added Version method to GRPC server instead of CLI.

**1.1.17**
- Version command in agent
- non-blocking manifest synchronization
- Checking for finalized files for 15min after upload complete
- Parallelized syncing of manifest

**1.1.16**
- Improved checking for existing process for Pennsieve Agent when starting service.
- Improved mechanism for stopping Pennsieve Agent using GRPC command instead of PID.

**1.1.15**
- Adding centralized error handling for GRPC errors
- (fix) Adding preRun action on Dataset List, and Find methods