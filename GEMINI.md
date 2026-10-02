# GEMINI.md - Developer Guide for saptune

## 1. Project Overview

`saptune` is an operating system tuning and configuration management tool developed by SUSE for **SUSE Linux Enterprise Server (SLES) for SAP Applications**. Its primary purpose is to automate and validate system configuration recommendations published in SAP Notes and SUSE Best Practices for running SAP workloads (e.g., SAP HANA, SAP NetWeaver, S/4HANA, SAP ASE, SAP MaxDB).

### Key Features
- **SAP Notes & Solutions**: Implements recommendations from SAP Notes directly and aggregates them into high-level workload profiles called "Solutions".
- **Dynamic Configuration & Customization**: Allows administrators to override parameters, create custom notes/solutions, or drop extra tuning definitions into dedicated configuration directories.
- **System Verification & Compliance**: Validates running system parameters against SAP Note specifications, highlighting compliant and non-compliant settings with color or JSON output.
- **Staging Mechanism**: Allows testing and staging updated note definitions before releasing them into production tuning configurations.
- **Cloud Service Provider (CSP) Detection**: Automatically identifies cloud platforms (Azure, AWS, GCP, Alibaba, Oracle Cloud, IBM VPC) and applies cloud-specific optimizations.
- **Integration**: Works alongside the `saptune_check` verification script (`/usr/sbin/saptune_check`) and supports integration with the Trento monitoring agent.

---

## 2. Architecture & Codebase Map

The project is written in **pure Go** using only standard library packages and internal modules (no third-party dependencies).

```
/home/sschmidt/Projects/saptune/
├── main.go                     # Entry point: argument parsing, lock management, privilege checks, lifecycle
├── main_test.go                # System integration and top-level CLI tests
├── actions/                    # CLI actions and subcommands
│   ├── actions.go              # Action routing (SelectAction), color handling, path constants
│   ├── cmdsyntax.go            # Command-line help and syntax definitions per OS release
│   ├── configureacts.go        # 'saptune configure' implementation
│   ├── noteacts.go             # 'saptune note' actions (apply, revert, verify, show, etc.)
│   ├── serviceacts.go          # 'saptune service' actions (start, stop, takeover, etc.)
│   ├── solutionacts.go         # 'saptune solution' actions (apply, revert, change, etc.)
│   ├── stagingacts.go          # 'saptune staging' actions (enable, diff, release, etc.)
│   └── table.go                # Formatted table output for CLI verification and listing
├── app/                        # Application domain model and state tracking
│   ├── app.go                  # App struct, config loading/saving (TUNE_FOR_*, NOTE_APPLY_ORDER)
│   ├── note.go                 # Note application logic and sanity checks
│   ├── parameter.go            # Parameter change detection, diffing, and value handling
│   ├── solution.go             # Solution application and note orchestration
│   └── state.go                # Serialized note states on disk
├── sap/                        # SAP-specific domain logic and tuning definitions
│   ├── errs.go                 # Common SAP domain error types
│   ├── note/                   # Note parser, INI loader, and section tuning handlers:
│   │   ├── sectBlock.go        # Block device read-ahead and elevator settings
│   │   ├── sectCPU.go          # CPU governors and energy perf bias
│   │   ├── sectFS.go           # Filesystem settings
│   │   ├── sectGrub.go         # Bootloader / GRUB settings
│   │   ├── sectLimits.go       # PAM limits configuration
│   │   ├── sectLogin.go        # Systemd logind / TasksMax configurations
│   │   ├── sectMem.go          # Transparent Huge Pages and memory settings
│   │   ├── sectPagecache.go    # Pagecache limit configurations
│   │   ├── sectRpm.go          # Package dependencies
│   │   ├── sectService.go      # Systemd service state tuning
│   │   ├── sectSys.go          # Kernel sysfs settings
│   │   ├── sectSysctl.go       # Kernel sysctl parameters
│   │   └── sectVM.go           # Virtual memory parameters
│   ├── param/                  # Abstract parameter representations and I/O handlers
│   └── solution/               # Solution definition parser and architecture mappings
├── system/                     # Low-level OS abstraction layer
│   ├── argsAndFlags.go         # CLI flag and argument parser
│   ├── cmdline.go              # Command line invocation helpers
│   ├── commandsMap.go          # Valid command combinations map
│   ├── csp.go                  # DMI / hardware inspection for Cloud Service Providers
│   ├── file.go                 # File utilities and atomic operations
│   ├── json.go                 # JSON formatting and serialization
│   ├── lock.go                 # File locking (/run/.saptune.lock)
│   ├── logging.go              # System logging and debug tracing
│   ├── service.go              # Systemd unit operations via systemctl
│   ├── sysctl.go               # Sysctl manipulation and configuration file generation
│   ├── system.go               # Release detection, architecture selector, root privilege check
│   └── trento.go               # Trento agent configuration integration
├── txtparser/                  # Parsing utilities
│   ├── ini.go                  # INI file parser for SAP Note and Solution definitions
│   ├── section.go              # Section extraction and serialization
│   ├── sysconfig.go            # Parser for /etc/sysconfig style key-value configuration files
│   └── tags.go                 # INI metadata tag processing
├── ospackage/                  # Packaging and distribution assets
│   ├── bin/saptune_check       # Bash diagnostic and system health check script
│   ├── etc/                    # Sysconfig templates
│   ├── man/                    # Man pages (saptune.8, saptune-note.5, saptune-solution.5, etc.)
│   ├── svc/                    # Systemd service unit files (saptune.service)
│   └── usr/share/saptune/      # Base Note definitions, Solution files (.sol), templates
└── testdata/                   # Fixtures for unit and integration testing
```

---

## 3. Filesystem Hierarchy & OS Release Differences

Saptune behaves differently depending on the targeted SUSE Linux Enterprise Server version, determined at build time by `system.RPMBldVers` (`system.IfdefVers()`):

| Resource / Purpose | SLES 12 & SLES 15 (`bvers <= 15`) | SLES 16+ (`bvers > 15`) |
| --- | --- | --- |
| **Main Config File** | `/etc/sysconfig/saptune` | `/var/lib/saptune/config/saptune` |
| **Config Template** | `/usr/share/fillup-templates/sysconfig.saptune` | `/usr/share/saptune/saptuneTemplate.conf` |
| **Deprecated Commands** | `saptune daemon ...` available (deprecated) | `saptune daemon ...` removed |
| **Service Control** | `saptune service ...` | `saptune service ...` |

### Key Runtime & Working Directories
- **Working Area** (`/var/lib/saptune/working/`): Active notes (`notes/`) and solutions (`sols/`) used during runtime.
- **Package Area** (`/usr/share/saptune/`): Default definitions distributed by the RPM package.
- **Staging Area** (`/var/lib/saptune/staging/latest/`): New or updated notes waiting to be tested and released.
- **Overrides** (`/etc/saptune/override/`): User-specified overrides for note parameters.
- **Extra Sheets** (`/etc/saptune/extra/`): User-defined custom notes and configurations.
- **Lock File** (`/run/.saptune.lock`): Ensures only a single instance of `saptune` executes at a time.
- **Log File** (`/var/log/saptune/saptune.log`): Saptune execution log.

---

## 4. Development & Build Workflows

### Go Environment & Module Setup
- **No Go Modules**: The project historically uses the traditional `GOPATH` workspace setup with `GO111MODULE=off`. `go.mod` is ignored in `.gitignore`.
- **Source Location**: Sources are expected to reside at `$GOPATH/src/github.com/SUSE/saptune`.
- **Dependencies**: Exclusively standard library Go. Do **not** introduce external third-party dependencies without explicit architectural approval.

### Building saptune
To build version 3 with the appropriate build-time variables injected via ldflags:

```bash
cd $GOPATH/src/github.com/SUSE/saptune

version="3.2.0-test"
bdate=$(date +"%Y/%m/%d")
bvers=15  # Set to 15 for SLES 15 or 16 for SLES 16

go build -buildmode=pie -ldflags \
  "-X 'github.com/SUSE/saptune/actions.RPMVersion=$version' \
   -X 'github.com/SUSE/saptune/actions.RPMDate=$bdate' \
   -X 'github.com/SUSE/saptune/system.RPMBldVers=$bvers'"
```

---

## 5. Testing & Validation

Many tests inspect or mock system-level components (systemd units, sysctl, cgroups, `/usr/share/saptune`, `/etc/saptune`, `/var/lib/saptune`). Because tests alter system files and require elevated permissions, the canonical testing workflow uses a privileged openSUSE container.

### Local Testing with Docker (Canonical Workflow)
The CI pipeline (`.github/workflows/saptune-ut.yml` and `run_saptune_ci_tst.sh`) runs inside `registry.opensuse.org/home/angelabriel/st-ci-base/containers/st-ci-base:latest`:

```bash
# Start the privileged container with cgroups and tmpfs
docker run --name saptune-ci --privileged --tmpfs /run \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw --cgroupns=host -td \
  -v "$(pwd):/app" \
  registry.opensuse.org/home/angelabriel/st-ci-base/containers/st-ci-base

# Execute the CI test suite
docker exec -t saptune-ci /bin/sh -c "cd /app && ./run_saptune_ci_tst.sh"

# Inspect test coverage HTML if needed
go tool cover -html=c.out -o coverage.html

# Cleanup container
docker stop saptune-ci && docker rm saptune-ci
```

### Running Non-Privileged Unit Tests
Packages without hard system dependencies (such as `txtparser` or pure calculation functions) can be tested directly with standard `go test`:

```bash
export GO111MODULE=off
go test -v ./txtparser/...
go test -v ./sap/param/...
```

---

## 6. Code Style, Linting & Standards

- **Formatting**: Always format code using `gofmt -d .` or `gofmt -w .`.
- **Vetting**: Run `go vet -composites=false ./...`.
- **GolangCI-Lint**: The project includes a dedicated `.golangci.yml` configuration (version 2 schema). Key linters enabled include:
  - `bodyclose`, `copyloopvar`, `depguard`, `dogsled`, `dupl`, `errcheck`, `errorlint`, `funlen`, `gocheckcompilerdirectives`, `gochecknoinits`, `goconst`, `gocritic`, `gocyclo`, `gosec`, `govet`, `ineffassign`, `lll`, `misspell`, `nakedret`, `noctx`, `nolintlint`, `revive`, `staticcheck`, `testifylint`, `unconvert`, `unparam`, `unused`, `whitespace`.
- **Staticcheck Configuration**:
  - Root `staticcheck.conf`: `checks = ["inherit", "-ST1005"]`
  - `actions/staticcheck.conf`: `checks = ["inherit", "-ST1018"]`
- **Error Handling**: Do not suppress errors. Use `system.ErrorExit()` or explicit return values where applicable. System exits in tests are mocked via `system.OSExit` and `system.ErrorExitOut`.
- **RPM Changelog & Maintenance**:
  - Packaging is maintained via SUSE IBS (`osc -A https://api.suse.de bco -M SUSE:SLE-12-SP2:Update saptune`).
  - Note or Solution definition changes must reference a Bugzilla (`bsc#`) or Jira (`jsc#`) tracking ID in package changelogs.
