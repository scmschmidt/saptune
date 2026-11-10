# note

## Overview
SAP note tuning engine. A `Note` is a set of tunable parameters that can be inspected, optimised and applied. Notes come from INI tuning sheets (`INISettings`) or the built-in `LinuxPagingImprovements`; per-INI-section helpers (sect*.go) read, optimise and set system values. Also tracks the history of values per parameter so changes can be reverted.

## Interface
- `Note` interface (note.go): `Initialise() (Note, error)`, `Optimise() (Note, error)`, `Apply() error`, `Name() string`.
- `TuningOptions map[string]Note`; `GetTuningOptions(saptuneTuningDir, thirdPartyTuningDir)`, `(*TuningOptions).GetSortedIDs`.
- Comparison: `GetNoteHeadData`, `FieldComparison`, `CompareJSValue`, `CompareNoteFields`, `GetVendInfo`.
- `INISettings` (ini.go) implements `Note`, plus `SetValuesToApply`; `INISection*` constants; `LogindConfDir`, `LogindSAPConfFile`.
- `LinuxPagingImprovements` (db.go) implements `Note` (SAP Note 1557506).
- Parameter state (parameter.go): `ParameterNoteEntry`, `ParameterNotes` (`StoreParameter`, `IDInParameterList`, `PositionInParameterList`), `GetPathToParameter`, `ListParams`, `CreateParameterStartValues`, `AddParameterNoteValues`, `GetSavedParameterNotes`, `GetAllSavedParameters`, `ParameterLastNote`, `RevertParameter`, `CleanUpParamFile`, `IsLastNoteOfParameter`, `ChkConflictingParams`.
- Section helpers (Get/Opt/Set triples): `GetBlkVal/OptBlkVal/SetBlkVal`, `GetCPUVal/...`, `GetFSVal/...`, `GetGrubVal/...`, `GetServiceVal/...`, `GetVMVal/...`, `GetLoginVal/...`, `GetLimitsVal/...`, `GetMemVal/...`, `GetPagecacheVal/...`, `GetSysVal/...`, `GetRpmVal/...`, `GetKmodVal/OptKmodVal/SetKmodVal`, `ParseKmodEntry`, `KmodEntry`, and `OptSysctlVal`.
- Depends on [param](../param/README.md) for block-device queue parameters and on `sap` (parent package).

## Invariants
- Notes are immutable; `Initialise`/`Optimise` return copies (note.go: `Note` doc comment).
- Parameter note history is stored in application order (parameter.go: `ParameterNotes` doc comment).
- `StoreParameter` does not overwrite an existing state file unless `overwriteExisting` is true.

## Side Effects
- `Apply`/`Set*Val` modify system state (sysctl, sysfs, services, limits, logind drop-in under `/etc/systemd/logind.conf.d`, etc.).
- Parameter functions read/write JSON state files in `system.SaptuneParameterStateDir` (created with `os.MkdirAll`, mode 0755; files 0644).
- Package globals in ini.go (`ini`, `pc`, `blck`, `flstates`) hold state across calls.

## Verification
- Run: `go test ./...` from the Go module root (note: no `go.mod` exists under `/home/sschmidt/Projects/saptune_egm`, so this currently fails with "directory prefix ... does not contain main module"; imports use `github.com/SUSE/saptune/...`).
