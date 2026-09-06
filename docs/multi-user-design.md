# Pennsieve Agent — Multi-User Design

**Status**: Draft for review
**Author**: TBD
**Last updated**: 2026-06-17
**Companions**:
- `pennsieve-agent-service/docs/design.md` (control plane, v1)
- `pennsieve-agent-service/docs/data-plane-design.md` (data plane, in review)

## Why this doc exists

The Pennsieve Agent today assumes a single user per process: one
`~/.pennsieve/config.ini`, one local SQLite database, one gRPC server
on a configured TCP port, one active Pennsieve profile at a time. On
multi-user systems this assumption produces:

- Port collisions when two users start their agents on the same host.
- The local gRPC server having zero authentication, so any user on
  the box can talk to any other user's running agent.
- N processes consuming N WebSocket connections and N upload-worker
  pools on hosts with many active users.
- Operational friction (every user must remember to start their own
  agent on a login node).

We have multi-user deployments active or imminent in four shapes:

1. **Shared lab workstation** — a few researchers, each with their
   own OS account on a single Mac/Linux box.
2. **HPC login node** — many users SSH'd in concurrently, each with
   their own OS account. Headless. Active demand for this is coming
   soon and the user base will be large.
3. **Shared OS account workstation** — three users today share one
   OS login (e.g. `lab-shared`) and authenticate to Pennsieve as
   themselves via `pennsieve profile switch …`. Convenience-driven.
4. **JupyterHub / shared notebook host** — eventually.

PHI is in scope on these systems. We need an architecture that scales
to scenario 2 and isolates users at the OS level where possible.

## Goals

1. **Per-user isolation**: where the OS provides distinct user
   accounts (1, 2, 4), one user's data, credentials, and processes
   must be invisible and inaccessible to another user.
2. **Operational sanity at scale**: a host with 50+ active users does
   not require 50+ independently-managed processes for sysadmins to
   reason about.
3. **Backwards-compatible**: existing single-user installs continue
   to work without forced migration.
4. **Scenario 3 still works**: three Pennsieve identities sharing one
   OS account remain supported with no UX regression.
5. **Cross-platform**: full multi-user support on Linux; user-mode
   installs continue to work on macOS and Windows. (Rationale below.)
6. **Compatible with the v1 control plane and the data-plane design**:
   no rework on the cloud side.

## Non-goals

- **Multi-tenant agent process** (Option B in the discussion that led
  here — one process holding multiple users' Pennsieve credentials).
  Rejected on security grounds: a bug in such a process exposes all
  users' PHI at once.
- **Identity federation beyond Pennsieve auth**. The agent
  authenticates users against the existing Pennsieve identity system;
  it does not attempt to do SSO, OIDC, or any of that here.
- **Per-user agent containers**. Out of scope; if the deployment
  target is container-isolated already (k8s, etc.), the user-mode
  install is the right answer there.

## The shape we're picking

**Option C: supervisor + per-user worker** with phased delivery.

A long-lived **supervisor** process runs as a system service under a
dedicated `pennsieve` service account. It listens on a system-wide
Unix socket, authenticates each incoming connection via
`SO_PEERCRED`, and spawns or routes to a **worker** process running
as the connecting user's UID. Workers hold the per-user state we have
today (Pennsieve credentials, SQLite DB, WebSocket connections), one
worker per OS user.

```
              user A's CLI ──┐
                             │ Unix socket (SO_PEERCRED → UID A)
                             ▼
                       ┌──────────────┐
                       │              │
                       │  supervisor  │   runs as `pennsieve` service account
                       │              │   (root capabilities for setuid)
                       │              │
                       └──┬─────┬─────┘
                          │     │
                fork+setuid     fork+setuid
                          │     │
                ┌─────────▼┐  ┌─▼─────────┐
                │ worker A │  │ worker B  │   running as UID A / UID B
                │  (UID A) │  │  (UID B)  │
                │          │  │           │
                │  pkg/cloud│  │ pkg/cloud │
                │  WSClient│  │  WSClient │
                │  SQLite  │  │  SQLite   │
                │  ~A/.pennsieve│ ~B/.pennsieve
                └────┬─────┘  └─────┬─────┘
                     │              │
                     │  control + data planes
                     ▼              ▼
                  pennsieve-agent-service (cloud)
```

For scenarios where this is operational overkill (a personal laptop),
**user-mode install** continues to work: no supervisor, the user runs
their own agent as themselves, listening on a per-user Unix socket.
Same binary, same code paths, just bypassing the supervisor.

The CLI discovers which mode is active by checking
`/run/pennsieve/agent.sock` first (system mode) and then
`$XDG_RUNTIME_DIR/pennsieve/agent.sock` (user mode). Exactly one of
the two should be active on any given host.

### What about scenario 3?

When three users share one OS login (`lab-shared`), they all hit the
same worker — because worker identity is keyed by UID and they all
have the same UID. The worker maintains separate Pennsieve profile
registrations (already supported by the v1 control plane — each
`(installationId, profileName)` pair gets its own `agentId`). The
existing `pennsieve profile switch` flow tells the worker to flip
which Pennsieve identity is "active" for subsequent CLI commands.

A small extension we should consider as part of this work: let the
CLI target a profile per-command (`pennsieve --profile alice upload …`)
rather than requiring an explicit `switch` first. The worker can hold
multiple profile sessions concurrently — the cloud channel design
already supports concurrent registrations from a single
`installationId`. This makes scenario 3 less stateful and removes a
class of "Bob ran an upload but Alice's profile was active" footguns.

Audit attribution for scenario 3 stays at the **Pennsieve profile**
level. The OS can't distinguish Alice from Bob in this scenario, but
the cloud's audit records always include the `agentId`, which is
per-profile, so server-side audit is correctly attributed even when
the OS user is shared.

### Why not Option B (single process, multiple users in memory)

For completeness, the rejected alternative:

- A bug in the agent process sees all users' data.
- A compromised process holds all users' Pennsieve credentials and
  data-plane session keys simultaneously. With PHI on the data plane,
  this widens the blast radius from one user to N.
- Authentication on the local socket becomes a real problem (UID
  doesn't tell you which Pennsieve identity is asking).
- Privilege model is awkward: the process must write files into N
  users' home dirs, which means either running as root and dropping
  to the right UID per operation, or writing as a service account
  and then `chown`ing — both error-prone.

We are explicitly ruling Option B out.

## Phasing

### Phase 0 — Local socket migration (immediate)

Scope: ~1–2 weeks. Lands before HPC users arrive.

- Migrate the agent's local gRPC server from TCP to Unix sockets on
  Linux/macOS and named pipes on Windows.
- Default socket path:
  - Linux: `$XDG_RUNTIME_DIR/pennsieve/agent.sock` (falls back to
    `/tmp/pennsieve-$UID/agent.sock` if `XDG_RUNTIME_DIR` is unset)
  - macOS: `~/Library/Application Support/pennsieve/agent.sock`
  - Windows: `\\.\pipe\pennsieve-<sid>`
- Socket permission `0600` (owner-only) on Unix-y systems.
- Remove the fixed `agent.port` default from config. Keep TCP as an
  override for users who explicitly need it (CI environments, etc.)
  but it is no longer the default.
- Fix `pennsieve agent status` to honestly report state:
  - "not running" when the socket isn't there
  - "running as me" when the socket is mine
  - "running but not mine" when the socket exists but is owned by a
    different UID (this should never happen in user-mode but matters
    for diagnostics)
- Update CLI client code to discover and connect to the socket
  rather than dial a TCP port.

**This phase eliminates the port-collision and cross-user-spying
classes of bugs without introducing the supervisor.** It also lays
the groundwork for Phase 1 — Phase 1 just changes where the socket
lives and adds peer-credential authentication.

### Phase 1 — Supervisor + worker (the real multi-user work)

Scope: a few weeks. Gated on Phase 0 and on this doc landing.

#### 1a. Supervisor binary

Either a new `pennsieve-agent-supervisor` binary, or a
`pennsieve agent supervisor` subcommand on the existing binary. The
latter is simpler to package (one binary, one upgrade) and is the
recommendation here.

Responsibilities:

- Listen on `/run/pennsieve/agent.sock` (overrideable via config).
- On each connection: read the peer's UID via `SO_PEERCRED`.
- Look up the worker for that UID; if absent or dead, spawn one.
- Forward the request to the worker over a per-worker internal Unix
  socket and proxy responses back.
- Track worker liveness; reap dead workers; idle-out workers after
  N minutes (default 30) by sending them a clean shutdown signal.
- Refuse to start if it can't acquire `CAP_SETUID` and `CAP_SETGID`
  (we need them to spawn workers as arbitrary UIDs).

#### 1b. Worker spawning

Two implementation candidates:

1. **`os/exec.Cmd` with `SysProcAttr.Credential`**: spawn the worker
   binary as a child of the supervisor, setting the child's UID/GID
   before `exec`. Requires the supervisor to have `CAP_SETUID`.
   Cleanest control over lifecycle; we own the process tree.
2. **`systemd-run --uid=U --gid=G --user`**: outsource worker spawn
   and lifecycle to systemd, gaining cgroup isolation and journal
   integration for free. The supervisor becomes a thin router. Only
   works on systemd hosts; macOS is a non-starter.

Recommendation: **option 1** for portability and simpler lifecycle
control. We can layer option 2 as an optional integration later if
systemd-cgroup-based resource accounting becomes a requirement.

The supervisor must be able to find the worker binary on the file
system. Approach: ship a single binary, locate it via `/proc/self/exe`
on Linux, fall back to `os.Executable()` elsewhere. Worker mode is
selected by a hidden flag (`pennsieve agent worker --supervisor-fd=N`)
that consumes a pre-opened socket file descriptor.

#### 1c. IPC protocol: supervisor ↔ worker

Two options:

1. **Reuse the existing gRPC API.** The supervisor accepts a gRPC
   request, dials the worker over its internal socket, forwards the
   call. Streaming RPCs (Subscribe, GetTimeseriesRangeForChannels)
   work transparently if we use bidi streaming forwarding.
2. **A simpler frame-forwarding proxy.** The supervisor doesn't
   speak gRPC — it just copies raw bytes between the client socket
   and the worker socket. Lighter, but loses the ability to inspect
   requests at the supervisor layer.

Recommendation: **start with frame-forwarding (option 2)**. The
supervisor's job is authentication and routing, not request
inspection. Forwarding bytes preserves all existing gRPC semantics
(streaming, deadlines, metadata) without a re-implementation. If we
later want policy or rate limiting at the supervisor, we add a
parsing layer then.

The forwarding protocol: supervisor accepts a connection on the
public socket, identifies the UID, opens (or finds an existing) Unix
socket to the worker for that UID, then `io.Copy`s in both
directions until either side closes. One supervisor goroutine pair
per active client connection.

#### 1d. Worker lifecycle

- **Spawn**: supervisor receives a request from UID U, no worker
  exists → fork+setuid+exec the worker binary. Worker binds its
  internal socket at `/run/pennsieve/workers/<uid>.sock` with mode
  `0600` owned by U. Worker writes a readiness byte on its internal
  socket; supervisor reads it before forwarding the first request.
- **Idle**: each worker tracks last-activity timestamp; after N
  minutes idle, it closes its socket and exits. Supervisor detects
  exit via SIGCHLD, cleans up the routing entry.
- **Graceful shutdown**: supervisor SIGTERM signals all workers,
  which finish in-flight requests (with a deadline) and then exit.
  Pending data-plane uploads checkpoint to disk via the existing
  manifest mechanism. The cloud-side `pkg/cloud.Manager` cleanly
  closes its WS.
- **Crash**: if a worker dies abnormally, supervisor logs it, drops
  the routing entry, and any in-flight client requests get
  `Unavailable`. The CLI's existing reconnect logic handles this.

#### 1e. Worker storage and NFS

Workers store SQLite at `~/.pennsieve/agent.sqlite`. On HPC sites this
home is typically NFS-mounted, which is fine when there's one writer
(which we have, by construction — one worker per UID per host).

But: if user U is logged in on **two HPC nodes simultaneously**, each
node spawns its own worker, both writing the same SQLite file on NFS.
Two writers on NFS-SQLite is broken (locking semantics are weak).

Mitigations, in order of preference:

1. **`flock` on startup**: each worker tries `flock(LOCK_EX|LOCK_NB)`
   on `~/.pennsieve/agent.sqlite.lock`. If it fails, the worker logs
   "another agent for this user is already running on host X" and
   exits with a clear error. Other host already has the worker;
   second host's user must connect to the first host. This is the
   minimum viable fix.
2. **Per-host store**: relocate the worker's SQLite to
   `/var/lib/pennsieve/workers/<uid>/agent.sqlite` (system-owned,
   readable+writable by U via tmpfiles.d / systemd-tmpfiles). NFS
   ceases to matter. But this complicates the user-mode (no
   supervisor) install path, which has no privileged daemon to set
   up the directory.
3. **Hybrid**: in supervisor mode, use the per-host store; in
   user-mode, use home-dir storage and rely on the user not running
   on two nodes simultaneously.

Recommendation: ship Phase 1 with mitigation 1 (the `flock` check)
and follow up with mitigation 3 if HPC pain warrants it.

#### 1f. Packaging

- **Debian**: `.deb` containing the binary, a systemd unit, a
  tmpfiles.d entry for `/run/pennsieve`, and a postinst that creates
  the `pennsieve` system user. `apt install pennsieve-agent` and
  done.
- **RHEL / Rocky**: `.rpm` with the same structure.
- **macOS (system mode)**: not a target for Phase 1. macOS users
  continue with the user-mode install (Homebrew formula installs
  binary; systemd unit is replaced by a launchd plist they install
  to `~/Library/LaunchAgents`).
- **Windows**: not a target for Phase 1. Windows users continue with
  user-mode install. If a future multi-user Windows requirement
  surfaces, named pipes + `ImpersonateNamedPipeClient` is the
  equivalent of Unix `SO_PEERCRED` + setuid, but it's a separate
  body of work.

The supervisor is **Linux-only** for Phase 1. macOS and Windows users
get the same multi-user behavior they have today (one user per host,
or shared OS account); they continue with the user-mode install path
and the Phase 0 Unix-socket-with-owner-only-permissions hardening.

### Phase 2 — Optional polish (defer)

Candidates, in no particular order:

- Per-command profile targeting (`pennsieve --profile X upload …`).
- Per-host SQLite storage instead of homedir (mitigation 2 above).
- Systemd cgroup integration via `systemd-run` for resource
  accounting on shared hosts.
- macOS `launchd` system-mode install with `LOCAL_PEERCRED`.
- Windows named-pipe supervisor with `ImpersonateNamedPipeClient`.

None of these block initial Phase 1 delivery.

## Detailed component changes

### `pennsieve-agent` repo

```
cmd/agent/
  start.go            # existing; updated to bind Unix socket by default
  supervisor.go       # NEW: pennsieve agent supervisor
  worker.go           # NEW: pennsieve agent worker --supervisor-fd=N
  status.go           # existing; updated to discover via socket lookup

pkg/
  cloud/              # unchanged — runs inside worker
  server/             # gRPC server; bind to Unix socket instead of TCP
  socket/             # NEW: socket-path resolution, SO_PEERCRED
  supervisor/         # NEW: workerRegistry, spawn, forward, lifecycle

  config/             # updated to make TCP an override, default to socket

packaging/
  debian/             # NEW: control, postinst, systemd unit
  rpm/                # NEW: spec file
  systemd/
    pennsieve-agent.service       # supervisor unit
    tmpfiles.d/pennsieve.conf      # /run/pennsieve setup
```

The single binary supports three modes:

- `pennsieve agent start` — the user-mode behavior (one user, one
  socket under `$XDG_RUNTIME_DIR`).
- `pennsieve agent supervisor` — the supervisor mode (system socket
  at `/run/pennsieve/agent.sock`, spawns workers).
- `pennsieve agent worker --supervisor-fd=N` — internal mode invoked
  by the supervisor; not for end users.

### `pennsieve-agent-service` repo

Almost no changes. The cloud already handles:

- Per-`(installationId, profile)` registration (multiple distinct
  `agentId`s from one host)
- Multiple concurrent WS connections from one host
- Per-`agentId` data-plane sessions (so PHI access is bounded to
  the right Pennsieve user even on a shared host)

A few small additions worth scoping:

- **Hostname-aware audit fields**: extend the registration record
  to capture `(hostname, OSUser, OSUid)`. The supervisor model means
  many `agentId`s share a hostname; surfacing `OSUser` in audit
  records helps with attribution on shared hosts. Phase 1 add.
- **Per-host registration limits**: optional cloud-side guard
  against a runaway host spawning unbounded `installationId`s. Phase
  2; speculative.

## Security analysis

### Threats

| Threat                                                          | Phase 0 | Phase 1 | Notes                                                                          |
|-----------------------------------------------------------------|---------|---------|--------------------------------------------------------------------------------|
| Local user A reads user B's agent socket                         | ✗ → ✓   | ✓       | Socket `0600` in Phase 0; SO_PEERCRED-authenticated in Phase 1                  |
| Local user A reads user B's SQLite (credentials, command history) | depends | ✓       | Phase 0 unchanged (relies on homedir permissions); Phase 1 same                |
| Compromised supervisor process                                   | n/a     | partial | Holds no per-user credentials; can spawn workers as any UID — but no PHI       |
| Compromised worker process                                       | (n/a)   | ✓       | Worker holds one user's credentials only; OS isolation contains blast radius   |
| Supervisor binary tampering                                      | n/a     | partial | Package signature verification (apt/rpm); rely on standard OS package guards   |
| Worker can't determine its peer (which CLI invoked it)            | -       | ✓       | Worker only ever talks to its own supervisor via a socket it created at known path |
| User starts a fake supervisor at the system socket path          | n/a     | ✓       | tmpfiles.d creates `/run/pennsieve` as root-owned; non-root can't write there  |
| Cloud-side compromise sees one host's all users                  | varies  | ✓       | Each worker has its own `agentId` + `agentSecret`; revocation is per-worker    |

### Key security claims

1. **Supervisor holds no PHI and no per-user credentials.** It is a
   router. A supervisor compromise lets an attacker spawn workers as
   arbitrary UIDs (which is what it could already do), but it does
   not directly expose any user's data.
2. **Worker compromise is bounded to one user.** Each worker is a
   separate process running under one UID, with access to one
   user's homedir, holding one user's Pennsieve credentials and
   data-plane session keys.
3. **OS-level isolation is the primary control.** We are
   deliberately delegating cross-user isolation to the kernel rather
   than implementing it in-process. This is the same security model
   used by `sshd`, systemd user services, and most multi-user Unix
   daemons.
4. **Scenario 3 retains profile-level isolation but not OS-level.**
   This is unchanged from today and is the documented limitation of
   shared OS account use.

### What this requires from sysadmins

- The supervisor unit runs as a system service. It requires
  `CAP_SETUID` and `CAP_SETGID` (granted via the systemd unit's
  `AmbientCapabilities`).
- The `pennsieve` system user owns `/run/pennsieve/` and the
  supervisor's runtime state.
- Worker binaries must be readable+executable by all users (standard
  `/usr/bin/pennsieve`, mode 0755).
- The agent does not need to read or modify user homedirs from the
  supervisor — only the worker does, running as the right UID.

## Operational considerations

- **Logging**: supervisor logs to journald (when run under systemd)
  or `/var/log/pennsieve/supervisor.log`. Workers log to journald
  with `_UID=<uid>` so sysadmins can `journalctl _UID=<uid> -u
  pennsieve-agent` to see one user's worker history.
- **Metrics**: supervisor emits Prometheus-style metrics on
  `/var/run/pennsieve/metrics.sock` (Unix socket, readable by node
  exporter): active worker count, requests per worker, spawn/exit
  counts, idle-out counts.
- **Upgrades**: the binary is replaced via package upgrade.
  Supervisor catches SIGHUP and re-execs itself, then sends SIGTERM
  to workers (which finish in-flight work and exit; supervisor
  re-spawns them on demand from the new binary).
- **Diagnostics**: `pennsieve agent supervisor inspect` (run by a
  sysadmin) reports active workers, their UIDs, their last activity,
  their cloud connection state, and queued requests.

## Migration

- **Existing user-mode installs** (the common case today) get the
  Phase 0 Unix-socket migration on next upgrade. Their behavior is
  otherwise unchanged: one user, one socket under
  `$XDG_RUNTIME_DIR`. They do not need the supervisor.
- **Hosts that want multi-user**: sysadmin installs the new package
  (`apt install pennsieve-agent`), which drops the systemd unit and
  starts the supervisor. Users on that host no longer run
  `pennsieve agent start` themselves; their CLI invocations connect
  to the supervisor's socket.
- **Detection in the CLI**: the CLI tries `/run/pennsieve/agent.sock`
  first. If it exists, system-mode is active; the CLI uses it. If
  not, the CLI falls back to user-mode (`$XDG_RUNTIME_DIR/.../...`)
  and may start the user-mode agent on demand the way it does today.
- **Coexistence**: system-mode and user-mode should not coexist on
  the same host. If a user starts a user-mode agent on a host that
  also has a system-mode supervisor, two things compete for the
  same Pennsieve registrations. The CLI should warn and refuse to
  start user-mode if it detects an active supervisor.

## Compliance impact

This work doesn't change HIPAA posture in any direction that wasn't
already addressed by the v1 and data-plane designs. It strengthens
several of those existing controls:

| Control                                | Impact                                                         |
|----------------------------------------|----------------------------------------------------------------|
| §164.312(a)(1) Access control          | Per-OS-user isolation hardens local access controls            |
| §164.312(a)(2)(i) Unique identification | Worker-per-user means audit attribution is per OS user when scenarios 1/2/4; per Pennsieve profile in scenario 3 |
| §164.312(b) Audit controls             | Adds OSUser/Hostname to audit records server-side               |
| §164.312(c)(1) Integrity               | Unchanged                                                       |
| §164.312(e)(1) Transmission security   | Unchanged                                                       |

The data-plane design's session-bound key access remains correct:
each worker is its own `agentId`, so its KMS grants and signed
public keys are scoped to one Pennsieve user even on a shared host.

## Open questions

These need stakeholder input before Phase 1 implementation starts.

1. **Worker idle-timeout default.** 30 minutes is a guess. On HPC
   login nodes where users come and go in seconds (`pennsieve
   upload` then disconnect), an aggressive timeout reduces resource
   use but causes constant cold-start latency. Worth a measurement.
2. **Per-worker resource limits.** Should the supervisor apply
   `setrlimit` or cgroup-based limits per worker (memory, FDs,
   uploads-in-flight)? Cheap insurance against runaway uploads
   destabilizing a shared host.
3. **NFS-homed SQLite mitigation choice.** `flock` check only
   (mitigation 1), or invest in per-host storage (mitigations 2/3)?
   Depends on whether HPC users routinely log into multiple nodes
   in parallel.
4. **HPC compute-node deployment.** When SLURM/PBS schedules a job
   that uses `pennsieve` CLI on a compute node, does the agent get
   installed cluster-wide (every node has a supervisor) or only on
   login nodes? Cluster-wide is the answer for any non-trivial
   workflow that uploads from compute nodes.
5. **Profile targeting per-command** (the scenario 3 polish):
   include in Phase 1 or defer to Phase 2? Small scope, but touches
   every CLI command.
6. **Coexistence vs hard mutual exclusion.** Should the CLI *refuse*
   to start a user-mode agent on a host with an active supervisor,
   or just warn? Refusal is safer but locks users in if the
   supervisor is misconfigured.
7. **Upgrade-in-place behavior.** On apt/yum upgrade, do we restart
   the supervisor immediately (kills in-flight uploads on workers)
   or wait for the next reboot? Probably "graceful re-exec on
   SIGHUP" but worth confirming the desired sysadmin UX.
8. **Per-host registration accounting.** When a host has 50 workers,
   the cloud sees 50 `agentId`s with the same hostname. Should the
   cloud's UI/dashboards group by hostname? Speculative.

## Decision log

| Decision                                       | Choice                                  | Why                                                                              |
|-----------------------------------------------|-----------------------------------------|----------------------------------------------------------------------------------|
| Multi-user architecture                       | Option C (supervisor + per-user worker) | OS isolation + single managed service + no in-process credential pooling          |
| Reject Option B                               | Yes                                     | PHI in scope; can't tolerate one process holding all users' credentials           |
| Reject Option A (status quo) as the only path | Yes                                     | Doesn't scale to HPC user counts; doesn't fit shared-OS-account scenario          |
| Worker spawn mechanism                        | `os/exec` + `setuid`                    | Portable, simple lifecycle; systemd-run can be added later as an option           |
| Supervisor IPC                                | Frame-forwarding (no gRPC at supervisor)| Preserves streaming semantics; supervisor stays minimal                            |
| Local transport                               | Unix sockets / named pipes              | Owner-only permissions; SO_PEERCRED for peer auth                                  |
| Linux-first for system mode                   | Yes                                     | HPC and lab Linux is the urgent target; macOS/Windows stay user-mode in Phase 1   |
| Storage location (worker)                     | Homedir + flock; revisit if HPC needs it | Minimum viable; per-host storage is a follow-up                                   |
| Scenario 3 handling                           | Worker holds multiple profiles concurrently; attribution via Pennsieve profile | Already supported by control plane; no architectural change |
| Cloud-side changes                            | Minimal (host/OSUser audit fields)      | Existing per-(install, profile) model is the right granularity                    |
