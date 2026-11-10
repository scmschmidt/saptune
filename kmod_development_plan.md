# Development plan: the `[kmod]` section for saptune

Audience: a coding agent that implements the feature in `src/` (Go module `github.com/SUSE/saptune`).
Specification (source of truth for behaviour): `support for kernel modules.md` in this directory. This plan says **how** to build it with the existing code. It does not repeat the behaviour tables; read the specification first.

Status: ready for implementation. The questions Q1 to Q9 are decided; see section 7. If you find something unclear or in conflict with the code, stop work on that part, make a suggestion and ask.

## 1. Goal and scope

Add a `[kmod]` section to saptune tuning sheets (Notes and Overrides). An entry is `MODULE = action[+option...] [PARAM=VALUE...]` with the actions `load`, `unload`, `blacklist`, `block` and the options `keep`, `no-reload`, `permanent`.

In scope: parse, apply, revert and verify of `[kmod]` entries, the two modprobe drop-in files, footnotes in verify output, unit tests, the man page `saptune-note(5)`.

Out of scope:
- `refresh` of `[kmod]` entries. The feature is experimental and only partially implemented.
- Rebuilding the initrd.
- `insmod`, `rmmod`, and forced load or unload. Use `modprobe` only.
- Any change to how saptune orders Notes, handles Overrides or stores parameter history. Reuse the existing rules.

## 2. Rules for the agent

1. Follow the existing patterns. Compare with `[service]` (`src/sap/note/sectService.go`) and `[login]` (`src/sap/note/sectLogin.go`). Do not add a framework, a new state store, or a new dependency.
2. Keep each change small and covered by a unit test written with the code. Existing test style: `src/sap/note/sectLogin_test.go`, `src/system/daemon_test.go`.
3. Change nothing in other sections. A diff outside the files listed in section 5 needs a reason in the commit message.
4. Run `gofmt -d`, `go vet -composites=false ./...` and the unit tests before you report done (see `src/development.md`). There is no `go.mod` under `/home/sschmidt/Projects/saptune_egm`, so run `go test` in a checkout where the module path `github.com/SUSE/saptune` resolves.
5. Update the `README.md` of every package you change (`src/system/README.md`, `src/sap/note/README.md`, and others if their interface changes). Keep the existing format: Overview, Interface, Invariants, Side Effects, Verification.
6. If the specification and this plan disagree, the specification wins; report the difference.
7. If something is unclear, stop work on that part, make a suggestion and ask. Do not guess.

## 3. How saptune handles a section today (what to reuse)

Verified in the source:

| Topic | Where | What to reuse |
|-------|-------|---------------|
| Section names | `src/sap/note/ini.go:16-30` (`INISection*` constants) | Add `INISectionKmod = "kmod"`. |
| Initialise, Optimise, Apply | `src/sap/note/ini.go` (`Initialise` line 82, `Optimise` line 160, `Apply` line 271). Each has a `switch param.Section`. | Add a `case INISectionKmod` to each. |
| Per-section helpers | `sectService.go`, `sectLogin.go`: `Get*Val`, `Opt*Val`, `Set*Val` | New file `sectKmod.go` with `GetKmodVal`, `OptKmodVal`, `SetKmodVal`. `GetServiceVal(key, cfgval)` shows how a getter uses the configured value. |
| Current value and expected value | `vend.SysctlParams[key]` holds the system value after `Initialise` and the expected value after `Optimise`. Verify compares the two strings. | The `[kmod]` state is one string per key. See section 4.2. |
| Parameter history for revert | `src/sap/note/parameter.go`: `CreateParameterStartValues`, `AddParameterNoteValues`, `RevertParameter`, `IsLastNoteOfParameter`. `ini.go` `createParamSavedStates`, `addParamSavedStates`, `setRevertParamValues`. State files live in `system.SaptuneParameterStateDir` (`/run/saptune/parameter`). | This is the "saved state" mechanism. It already handles several Notes touching the same key and the rule that reverting an earlier Note does not undo a later one. Do not add a second store. See (decided, Q1). |
| Revert flow | `ini.go` `Apply`: when `revert` is set, `setRevertParamValues(key)` puts the saved value into `vend.SysctlParams[key]`, then `Set*Val` is called. The section data of the applied Note (`ini`) comes from the stored copy (`GetSectionInfo("sns", ...)`), so the configured value, including options, is available in the loop as `param.Value`. | `SetKmodVal` receives the target state and the configured value. |
| Key syntax | `src/txtparser/ini.go` `splitLineIntoKOV`, `splitSectLine`: sections `grub`, `sys`, `service` prefix the key (`systemd:name`). | Decide the key form. See (decided, Q2). |
| Footnotes in verify | `src/actions/footnote.go`: texts `footnote1` to `footnote17`, `prepareFootnote` calls one `set*` function per footnote and uses the `Inform` map from `Initialise`/`Optimise`. | Add three footnotes (section 4.7). |
| `refresh` | `src/app/parameter.go:77` `isSectionSupportedForRefresh` | Add `INISectionKmod` to the unsupported list, like `INISectionService`. See (decided, Q5). |
| External commands | `src/system/daemon.go`: package variables `systemctlCmd`, `systemddvCmd`, and `exec.Command(...).CombinedOutput()` | Same pattern for `modprobe`: a package variable with the path, so tests can replace it. |
| Logging | `system.DebugLog/InfoLog/WarningLog/ErrorLog` | Use these. |

No existing code handles modprobe, `/sys/module` or modprobe.d files.

## 4. Design for the implementation

### 4.1 Parsing

- One line `MODULE = value` where value is `action[+option...] [PARAM=VALUE...]`. The module name is used as is. It cannot contain spaces.
- Split the value into a small struct (action, options, parameter pairs). Put the parser in `sectKmod.go` as a plain function with its own tests. Do not touch `txtparser` beyond what Q2 requires.
- Invalid entries are logged, printed as an error and skipped, as the specification says. An entry is invalid if an option is not allowed for its action, an option is repeated, or `PARAM=VALUE` is used on any action but `load`.
- The last entry for a module wins. This is existing saptune behaviour, not new code.

### 4.2 The state string

Everything saptune compares, saves and restores for a key is one string. For a module use one string that describes both parts of its state:

`<file>|<module>`

- `<file>` is `none`, `blacklist` or `block`: what saptune's own drop-ins contain for the module.
- `<module>` is `unloaded`, `loaded:<JSON object of all parameters>` or `PNA` for a compiled-in module.

Examples: `none|unloaded`, `block|loaded:{"debug":"N","timeout":"30"}`.

Why one string: `createParamSavedStates` and `addParamSavedStates` store `vend.SysctlParams[key]` as the start value and the value per Note, and revert hands it back. So the start value automatically contains "was the module loaded before apply" and all parameter values, which is the "saved state" of the specification. The specification says "`saved_state` files"; read this as the generic term for the saved state. The specification text stays unchanged (decided, Q1).

- `GetKmodVal(key, cfgval)` returns the current state. For `<module>` read all files in `/sys/module/MODULE/parameters/` (all parameters, as decided).
- `OptKmodVal(key, cfgval, actual)` returns the expected state: the actual state with the configured part replaced.
  - `load`: `<actual file>|loaded:{actual parameters, with the configured ones replaced}`. If the module is not loaded, the object holds only the configured parameters.
  - `unload`: `<actual file>|unloaded`.
  - `blacklist`, `block`: `blacklist|<actual module>` or `block|<actual module>`.
  Because everything not configured is copied from the actual state, verify reports non-compliant only when a configured part differs.
- `SetKmodVal(key, targetState, cfgval, revert)` makes the system match the target state. The same function serves apply and revert, because revert hands it a state string.

### 4.3 What `SetKmodVal` does

Compare the target with the current state and change only the difference.

Module part:
- `loaded:{...}` while the module is loaded: for each parameter whose current value differs, write `/sys/module/MODULE/parameters/PARAMETER`. If at least one write fails and `no-reload` is not set, run `modprobe -r MODULE` and `modprobe MODULE PARAM=VALUE...`. The failure cases (module in use, `modprobe` fails after `modprobe -r`, partial writes) are in the specification; log every `modprobe` call and every write.
- `loaded:{...}` while the module is not loaded: `modprobe MODULE PARAM=VALUE...`.
- `unloaded` while the module is loaded: `modprobe -r MODULE`.
- `PNA` (compiled in): warn that the module cannot be handled, change nothing.

File part:
- Lines are `blacklist MODULE` and `install MODULE /bin/false` only.
- First remove the module's lines that the target does not produce, in both `/run/modprobe.d/80-saptune.conf` and `/etc/modprobe.d/80-saptune-permanent.conf`. Then write the line for the target, into the `/etc` file if `permanent` is set in `cfgval`, else into the `/run` file. Delete a file that has no entries left. Write files atomically (temporary file in the same directory, then `os.Rename`; `actions/solutionacts.go` around line 575 uses this pattern). Create `/run/modprobe.d/` if it is missing. Keep the header comment of each file as the specification gives it.
- Only lines in exactly these two forms are touched. Other lines and other files are left alone.
- On revert, skip everything if the configured value has `keep` (load, unload) or `permanent` (blacklist, block).
- When the module was loaded and the action is `blacklist` or `block`, print the warning that the module must be unloaded first. Compiled-in modules: warn and skip.

Cleanup of stale lines is the same code path: the "remove lines the target does not produce" step above runs on every apply, before the new line is written. A line for a module that has no `[kmod]` entry anymore is not removed (specification, Edge Cases).

### 4.4 System helpers

New file `src/system/kmod.go` with small functions and a test file:
- `IsModuleLoaded`, `IsModuleBuiltin` (use `/sys/module/MODULE/initstate`, as the specification says saptune does), `GetModuleParams`, `SetModuleParam`.
- `ModprobeLoad(module, params)`, `ModprobeRemove(module)`, using a package variable `modprobeCmd = "/usr/sbin/modprobe"` (path to be checked on SLES 15 and 16) and `exec.Command(...).CombinedOutput()`.
- `ModprobeConfig()` runs `modprobe -c` and returns its output, for the effective-configuration check in verify.
- Drop-in read and write helpers, which take the file path as a parameter so tests can use a temporary directory.

### 4.5 Verify

Verify compares the actual and expected state strings, as for every other section. The extra rules of the specification need the `Inform` map and `prepareFootnote`:

- `block` and `blacklist` entries: compliant if the string matches (file part right) and the entry is in the effective configuration (`modprobe -c`). If another entry takes precedence, mark non-compliant and add a footnote.
- If the start value of the key says the module was `loaded` and it is still loaded, mark non-compliant and add a footnote. Read the start value from the parameter state file. `parameter.go` has the readers (`GetSavedParameterNotes` and the helpers around it); confirm which function returns the start value.
- With `permanent`, add the initrd footnote.
- Compiled-in module: reported as skipped. `unload` of a module that is not loaded is compliant.

### 4.6 Footnotes

Existing footnotes end at `[17]`. Add:

| No | Suggested text | Used when |
|----|----------------|-----------|
| 18 | ` [18] another modprobe configuration entry takes precedence over the saptune entry` | the entry is not part of the effective configuration |
| 19 | ` [19] the module was already loaded when the entry was applied; blocking or blacklisting does not unload it` | recorded as loaded on apply and still loaded |
| 20 | ` [20] after a permanent entry, re-create the initrd if the module is still loaded after a reboot` | `permanent` is set |

The maintainer asked for these suggestions. See (decided, Q6) for confirmation.

### 4.7 Documentation

- `src/ospackage/man/saptune-note.5`: describe the `[kmod]` section: syntax, the four actions, the three options, the drop-in files, the verify footnotes.
- The man page must also say (from the specification): changing modules can disable hardware or prevent boot, the user is responsible for the entries, saptune applies after initrd and udev, and permanent entries may need an initrd rebuild that the user does by hand.

## 5. Work packages (in this order)

Each package ends with passing tests. Do not start the next package before the previous one is done.

1. **System helpers** (`src/system/kmod.go`, `kmod_test.go`, `README.md`): section 4.4. Tests use a temporary directory and a fake `modprobe` script through `modprobeCmd`.
2. **Entry parser** (`src/sap/note/sectKmod.go`, `sectKmod_test.go`): section 4.1. Test every valid and invalid combination in the specification.
3. **State string and Get/Opt** (`sectKmod.go`): section 4.2. Test with fake `/sys/module` trees (the code takes the base path from a package variable, as other `sect*.go` files do for their paths; check `LogindConfDir` in `ini.go`).
4. **Set** (`sectKmod.go`): section 4.3. Tests: each action, apply then revert returns the start state, a failing sysfs write triggers the reload, `no-reload`, `keep`, `permanent`, stale-line removal, atomic write, compiled-in module.
5. **Wiring** (`src/sap/note/ini.go`, `src/txtparser/ini.go` if Q2 needs it, `src/app/parameter.go`): section constant, the three `Initialise`, `Optimise` and `Apply` cases, refresh exclusion. Test with a small `[kmod]` tuning sheet through `INISettings` (see `ini_test.go`).
6. **Verify and footnotes** (`src/actions/footnote.go` and its tests): section 4.5 and 4.6.
7. **Documentation** (`saptune-note.5`, package READMEs): section 4.7.
8. **Integration check**: run the CI script in `src/run_saptune_ci_tst.sh` as described in `src/development.md` and report the result. A real module test needs a container with a harmless module (for example `dummy` or `loop`); do not test with a module that backs a mounted filesystem.

## 6. Test plan summary

- Unit tests per package, as listed. Reuse test helpers from the neighbouring `_test.go` files.
- Cases that must exist: two Notes with the same module (last wins, revert order), revert of a module that was already loaded with identical values, revert of a module that was not loaded, parameter that cannot be written (reload path), `modprobe -r` failure, drop-in line removal when the action changes, file deleted when empty, `permanent` survives revert, foreign lines in the drop-in untouched.

## 7. Decisions

All suggestions were accepted by the maintainer.

| Q | Decision |
|---|----------|
| Q1 | The saved state is the per-key parameter state (`/run/saptune/parameter`, section 4.2). "`saved_state` files" in the specification is the generic term; the specification text stays as it is. |
| Q2 | Key form is `kmod:MODULE`. Add `kmod` to the sections handled in `splitSectLine` in `src/txtparser/ini.go`. |
| Q3 | Keep the module name as given for the key and the drop-in lines. Normalise `-` to `_` only for the `/sys/module` lookup. Log a warning if two keys normalise to the same module. |
| Q4 | Revert to an earlier Note's state writes the restored `block` or `blacklist` line to the transient `/run` file. `permanent` is never restored on revert. |
| Q5 | `[kmod]` is listed as unsupported in `isSectionSupportedForRefresh`, with the existing info message. The word `refresh` in the specification's cleanup text stays; `refresh` itself is not implemented now. |
| Q6 | The footnote texts and numbers 18 to 20 in section 4.6 are approved. |
| Q7 | A compiled-in module returns `PNA` for the module part; the existing footnote `[16]` applies. |
| Q8 | Default `modprobeCmd` is `/usr/sbin/modprobe`. Check the path on SLES 15 and 16 during the integration check (work package 8) and report if it differs. |
| Q9 | Document `[kmod]` in `saptune-note.5` only. Ask the maintainer whether a changelog entry is needed when the work is ready for review. |
