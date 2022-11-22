**1.2.5**
- Calculating SHA256 for files during upload
- Bug fix to terminate sync-handlers post uploading files
- Updated mechanisms to handle credentials provided in ENV variables.

**1.2.4**
- Added apihost information to the active user GRPC endpoint response

**1.2.3**
- Updated Agent to better handle configuration file parameters and ENV variables
- Fix 'dataset find' command

**1.2.2**
- Refactored Agent to allow users to start agent without config file using ENV Variables

**1.2.0**
- Improved: Wrapping services in Interfaces and restructuring packages
- Fixed: time-out of AWS session due to race-condition

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
