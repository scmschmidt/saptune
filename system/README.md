# system

## Overview
Leaf utility package (`package system`) wrapping host access for saptune: /sys, /proc/sys and sysctl parameters, systemd/tuned control, CPU/memory/block-device/filesystem info, cloud service provider (CSP) detection, config-file helpers, CLI argument parsing, logging, the saptune run lock and JSON output structures.

## Interface
- CLI args and flags (`argsAndFlags.go`): `RereadArgs`, `CliArg`, `CliArgs`, `IsFlagSet`, `GetFlagVal`, `ParseCliArgs`, `ChkCliSyntax`.
- Kernel cmdline (`cmdline.go`): `ProcCmdLine`, `ParseCmdline`, `IsInternalGrub`.
- /sys and sysctl (`sys.go`, `sysctl.go`): `GetSysString`, `GetSysChoice`, `GetSysInt`, `SetSysString`, `SetSysInt`, `TestSysString`, `GetSysSearchParam`, `GetNrTags`; `GetSysctlString/Int/Uint64/Uint64Field`, `SetSysctlString/Int/Uint64/Uint64Field`, `ChkForSysctlDoubles`, `CollectGlobalSysctls`, `IsPagecacheAvailable`, `IsValidSysctlLocations`.
- Block devices (`blockdev.go`): `BlockDev`, `IsSched`, `IsNrreq`, `IsRahead`, `IsMsect`, `BlockDeviceIsDisk`, `GetBlockDeviceInfo`, `CollectBlockDeviceInfo`, `GetAvailBlockInfo`.
- Filesystems (`fs.go`): `MountPoint`, `MountPoints` (`GetByMountPoint`, `GetByMountOption`), `ParseMounts`, `ParseMtab`, `ParseFstab`, `ParseProcMounts`, `ParseMtabMounts`, `RemountSHM`, `GetMountOpts`, `IsXFSOption`.
- CPU (`cpu.go`): `GetPerfBias`, `SetPerfBias`, `SecureBootEnabled`, `GetGovernor`, `SetGovernor`, `GetFLInfo`, `SetForceLatency`, `FlTblEntry`, `TranslateCtoL`, `IsValidFL`, `CPUPlatform`, `GetdmaLatency`.
- Memory (`mem.go`): `ParseMeminfo`, `GetMainMemSizeMB`, `GetTotalMemSizeMB`, `GetTotalMemSizePages`, `GetSemaphoreLimits`.
- systemd/tuned (`daemon.go`, `service.go`): `SystemctlEnable/Disable/Start/Stop/Restart/ReloadTryRestart/EnableStart/DisableStop/ResetFailed/DaemonReload/Status/IsEnabled/IsRunning/IsActive`, `SystemdDetectVirt`, `GetSystemState`, `SysIsStarting`, `SysIsRunning`, `IsSystemRunning`, `IsServiceAvailable`, `CmpServiceStates`, `WriteTunedAdmProfile`, `GetTunedProfile`, `TunedAdmOff`, `TunedAdmProfile`, `GetTunedAdmProfile`, `IsSapconfActive`, `GetAvailServices`, `GetServiceName`.
- Cloud provider (`csp.go`, `communication.go`): `CSPInstance`, `CSPInstanceConfig`, `CSPTimeout`, `CSPRetries`, `CSPDetection`, `TCSP`, `GetCSP`, `ResetCloudConfigFile`, `WriteCloudConfigFile`, `GetCSPInstanceInfo`, `DetectCSPInstance`, `DetectAndStoreInstanceType`, `GetCSPInstanceStatusInfo`, `CommParams`, `APIVersionsResponse`, `CommParamInit`, `RetrieveAPIVersion`, `RetrievalRequest`.
- Files (`file.go`): `ReadConfigFile`, `FileIsEmpty`, `EditFile`, `EditAndCheckFile`, `ChkMD5Pair`, `GetMD5Hash`, `CopyFile`, `GetFiles`, `ListDir`, `CleanUpRun`, `GetBackupValue`, `WriteBackupValue`, `AddGap`.
- Security limits (`limits.go`): `SecurityLimitInt`, `SecurityLimitUnlimitedValue`, `SecLimitsEntry`, `SecLimits` (`Get`, `GetOr0`, `Set`, `ToText`, `ToDropIn`, `ApplyDropIn`, `Apply`), `ParseSecLimitsFile`, `ParseSecLimits`, `ToSecurityLimitInt`.
- Logins (`login.go`): `GetCurrentLogins`, `GetTasksMax`, `SetTasksMax`.
- Packages (`rpm.go`): `GetRpmVers`, `CmpRpmVers`, `CheckRpmVers`.
- Kernel modules (`kmod.go`): `KmodTransientDropIn`, `KmodPermanentDropIn`, `KmodTransientHeader`, `KmodPermanentHeader`, `SysfsModuleDir`, `NormalizeModuleName`, `IsModuleLoaded`, `IsModuleBuiltin`, `GetModuleParams`, `SetModuleParam`, `ModprobeLoad`, `ModprobeRemove`, `ModprobeConfig`, `ReadDropInFile`, `WriteDropInFileAtomic`, `RemoveDropInModuleLines`, `AddDropInModuleLine`, `GetDropInModuleState`, `IsEffectiveConfigEntry`.
- Trento (`trento.go`): `CheckAndSetTrento`.
- Lock (`lock.go`, `commandsMap.go`): `SaptuneLock`, `ReleaseSaptuneLock`; the 'realm command' table decides whether a lock is taken.
- Logging (`logging.go`): `DebugLog`, `NoticeLog`, `InfoLog`, `WarningLog`, `ErrorLog`, `ErrorLogNoStdErr`, `LogInit`, `SwitchOffLogging`, `PrintLog`.
- JSON output (`json.go`): `JMsg`, `JMessages`, `JResult`, `JObj`, `JEntry`, `JAppliedSol`, `JNoteList*`, `JPNotes*`, `JStatus*`, `JSol*`, `JInvalid`, `JnotSupportedYet`, `Jcollect`, `JNoteListEntryInit`.
- General (`system.go`, `calc.go`): `SaptuneSectionDir`, `SaptuneSavedStateDir`, `SaptuneParameterStateDir`, `RPMBldVers`, `OSExit`, `ErrorExitOut`, `ErrExitOut`, `InfoOut`, `DmiID`, `IfdefVers`, `SaptuneConfigFile`, `SaptuneConfigTemplate`, `IsUserRoot`, `GetSolutionSelector`, `CmdIsAvailable`, `GetOsVers/GetOsRel/GetOsName`, `IsSLE`, `IsSLE12/15/16`, `KernelRelease`, `CheckForPattern`, `CalledFrom`, `CallingFunc`, `ErrorExit`, `OutIsTerm`, `InitOut`, `SwitchOffOut`, `SwitchOnOut`, `WrapTxt`, `GetDmiID`, `GetHWIdentity`, `StripComment`, `GetVirtStatus`, `Watch`; `MaxI64`, `MaxI`, `MaxU64`, `MinU64`.

## Invariants
- `system.go:OSExit`, `ErrorExitOut`, `ErrExitOut`, `InfoOut` are package variables so tests can replace exit and output behaviour; code should call them rather than `os.Exit` directly.
- `lock.go:stLockFile` (`/run/.saptune.lock`) holds the PID of the owning process; `SaptuneLock` exits (`ErrorExit`, code 11) if another live process holds it, and `saptuneIsLocked` removes locks that are empty or belong to dead processes. Lock creation uses `O_EXCL`.
- `commandsMap.go:defaultCommand` lists valid 'realm command' combinations; unknown combinations still get a lock (`lock.go:SaptuneLock`).
- `trento.go:checkForValidValue` accepts only discovery periods 300, 600, 900, 1800, 3600.
- `sysctl.go:SetSysctlString` writes under `/proc/sys` with dots in the parameter translated to path separators.

## Side Effects
- Reads and writes `/sys`, `/proc/sys` (`sys.go`, `sysctl.go`); reads `/proc/cmdline`, `/sys/class/dmi/id`, mount tables and meminfo.
- Executes external commands via `exec.Command`: `systemctl`, `systemd-detect-virt`, `tuned-adm` (`daemon.go`, `service.go`), `rpm` (`rpm.go`), login tooling (`login.go`), `modprobe` (`kmod.go`).
- Creates/removes `/run/.saptune.lock`; uses state dirs under `/run/saptune/` (`SaptuneSectionDir`, `SaptuneSavedStateDir`, `SaptuneParameterStateDir`).
- Writes `/run/modprobe.d/80-saptune.conf` and `/etc/modprobe.d/80-saptune-permanent.conf` (`kmod.go`).
- Writes `/var/lib/saptune/config/cloud.json` (`csp.go`) and may query CSP metadata endpoints over the network (`communication.go`, `csp.go`).
- Modifies `/etc/trento/agent.yaml` (`trento.go`), security limits drop-ins (`limits.go`), and copies/edits config files (`file.go`).
- Writes to stdout/stderr and the log file (`logging.go`, `system.go:InitOut`); `ErrorExit` terminates the process.

## Verification
- Run: `go test ./...` (Go module under `/home/sschmidt/Projects/saptune_egm/src`)
