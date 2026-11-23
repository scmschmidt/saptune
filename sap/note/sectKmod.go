package note

import (
	"encoding/json"
	"fmt"
	"github.com/SUSE/saptune/system"
	"os"
	"strings"
)

// section [kmod]

var (
	// KmodDropInTransientPath is the transient modprobe drop-in location.
	KmodDropInTransientPath = system.KmodTransientDropInPath
	// KmodDropInPermanentPath is the permanent modprobe drop-in location.
	KmodDropInPermanentPath = system.KmodPermanentDropInPath
)

// KmodEntry represents a parsed [kmod] configuration entry.
type KmodEntry struct {
	Module    string
	Action    string
	Permanent bool
	Keep      bool
	NoReload  bool
	Params    map[string]string
}

// cleanKmodModule extracts and validates the kernel module name from key.
func cleanKmodModule(key string) (string, error) {
	mod := strings.TrimSpace(key)
	if strings.HasPrefix(mod, "kmod:") {
		mod = strings.TrimPrefix(mod, "kmod:")
	}
	if mod == "" {
		return "", fmt.Errorf("empty module name")
	}
	if strings.ContainsAny(mod, " \t\r\n") {
		return "", fmt.Errorf("module name '%s' contains whitespace", mod)
	}
	return mod, nil
}

// kmodEntryPrefix formats the logging prefix with Note name and entry if available.
func kmodEntryPrefix(noteID, mod, value string) string {
	valStr := strings.TrimSpace(value)
	var entryStr string
	if mod != "" && valStr != "" {
		entryStr = fmt.Sprintf("'%s = %s'", mod, valStr)
	} else if mod != "" {
		entryStr = fmt.Sprintf("'%s ='", mod)
	} else if valStr != "" {
		entryStr = fmt.Sprintf("'= %s'", valStr)
	}

	if noteID != "" {
		if entryStr != "" {
			return fmt.Sprintf("kmod: Note '%s', entry %s", noteID, entryStr)
		}
		return fmt.Sprintf("kmod: Note '%s'", noteID)
	}
	if entryStr != "" {
		return fmt.Sprintf("kmod: entry %s", entryStr)
	}
	return "kmod"
}

// parseKmodActionAndOptions splits action[+opt1+opt2] into action and option tokens.
func parseKmodActionAndOptions(prefix, actToken string) (string, []string, error) {
	if actToken == "" {
		return "", nil, system.ErrorLog("%s: empty action", prefix)
	}
	parts := strings.Split(actToken, "+")
	action := strings.ToLower(strings.TrimSpace(parts[0]))
	if action == "" {
		return "", nil, system.ErrorLog("%s: empty action", prefix)
	}
	var opts []string
	for _, opt := range parts[1:] {
		trimmed := strings.ToLower(strings.TrimSpace(opt))
		if trimmed == "" {
			return "", nil, system.ErrorLog("%s: empty option in '%s'", prefix, actToken)
		}
		opts = append(opts, trimmed)
	}
	return action, opts, nil
}

// validateKmodActionAndOpts validates options for the given action according to spec.
func validateKmodActionAndOpts(prefix string, entry *KmodEntry, action string, opts []string) error {
	seen := make(map[string]bool)
	for _, opt := range opts {
		if seen[opt] {
			return system.ErrorLog("%s: repeated option '%s' for module '%s'", prefix, opt, entry.Module)
		}
		seen[opt] = true

		switch opt {
		case "permanent":
			if action != "blacklist" && action != "block" {
				return system.ErrorLog("%s: option 'permanent' not allowed for action '%s'", prefix, action)
			}
			entry.Permanent = true
		case "keep":
			if action != "load" && action != "unload" {
				return system.ErrorLog("%s: option 'keep' not allowed for action '%s'", prefix, action)
			}
			entry.Keep = true
		case "no-reload":
			if action != "load" {
				return system.ErrorLog("%s: option 'no-reload' not allowed for action '%s'", prefix, action)
			}
			entry.NoReload = true
		default:
			return system.ErrorLog("%s: unknown option '%s' for action '%s'", prefix, opt, action)
		}
	}
	return nil
}

// parseKmodParams parses PARAM=VALUE tokens into a map.
func parseKmodParams(prefix string, tokens []string) (map[string]string, error) {
	params := make(map[string]string)
	for _, tok := range tokens {
		p := strings.TrimSpace(tok)
		if p == "" {
			continue
		}
		eqIdx := strings.Index(p, "=")
		if eqIdx <= 0 || eqIdx == len(p)-1 {
			return nil, system.ErrorLog("%s: invalid parameter pair '%s', expected PARAM=VALUE", prefix, p)
		}
		k := strings.TrimSpace(p[:eqIdx])
		v := strings.TrimSpace(p[eqIdx+1:])
		params[k] = v
	}
	return params, nil
}

// ParseKmodEntry parses a MODULE = value line from the [kmod] section.
// Load and unload actions are rejected for now until implemented.
func ParseKmodEntry(noteID, key, value string) (*KmodEntry, error) {
	trimmedVal := strings.TrimSpace(value)
	mod, err := cleanKmodModule(key)
	prefix := kmodEntryPrefix(noteID, mod, trimmedVal)
	if err != nil {
		return nil, system.ErrorLog("%s: %v", prefix, err)
	}

	fields := strings.Fields(trimmedVal)
	if len(fields) == 0 {
		return nil, system.ErrorLog("%s: missing action for module '%s'", prefix, mod)
	}

	action, opts, err := parseKmodActionAndOptions(prefix, fields[0])
	if err != nil {
		return nil, err
	}

	entry := &KmodEntry{
		Module: mod,
		Action: action,
		Params: make(map[string]string),
	}

	if err := validateKmodActionAndOpts(prefix, entry, action, opts); err != nil {
		return nil, err
	}

	if len(fields) > 1 {
		if action != "load" {
			return nil, system.ErrorLog("%s: parameters specified for non-load action '%s'", prefix, action)
		}
		params, err := parseKmodParams(prefix, fields[1:])
		if err != nil {
			return nil, err
		}
		entry.Params = params
	}

	switch action {
	case "blacklist", "block":
		return entry, nil
	case "load", "unload":
		return nil, system.ErrorLog("%s: action '%s' for module '%s' is not supported yet", prefix, action, mod)
	default:
		return nil, system.ErrorLog("%s: unknown action '%s' for module '%s'", prefix, action, mod)
	}
}

// parseStateString splits a state string <file>|<module> into its components.
func parseStateString(state string) (string, string) {
	parts := strings.SplitN(state, "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "none", "unloaded"
}

// formatModuleState inspects sysfs and returns the <module> part of the state string.
func formatModuleState(mod string) string {
	if !system.IsModuleLoaded(mod) {
		return "unloaded"
	}
	params, err := system.GetModuleParams(mod)
	if err != nil || len(params) == 0 {
		return "loaded:{}"
	}
	bytes, err := json.Marshal(params)
	if err != nil {
		return "loaded:{}"
	}
	return "loaded:" + string(bytes)
}

// GetKmodVal initialises the [kmod] state with the current system settings.
// Returns "PNA" for compiled-in modules, or "<file>|<module>".
func GetKmodVal(key, cfgval string) string {
	mod, err := cleanKmodModule(key)
	if err != nil {
		return "NA"
	}
	if system.IsModuleBuiltin(mod) {
		system.WarningLog("kmod: module '%s' is compiled-in and cannot be handled. Skipping...", mod)
		return "PNA"
	}

	filePart := system.GetDropInModuleState(KmodDropInTransientPath, KmodDropInPermanentPath, mod)
	modPart := formatModuleState(mod)
	return filePart + "|" + modPart
}

// OptKmodVal returns the expected [kmod] state string from the configured value.
func OptKmodVal(noteID, key, cfgval, actual string) string {
	entry, err := ParseKmodEntry(noteID, key, cfgval)
	if err != nil {
		return "NA"
	}
	if actual == "PNA" {
		return "PNA"
	}

	_, actualMod := parseStateString(actual)
	action := entry.Action
	if entry.Permanent {
		action += "+permanent"
	}
	switch entry.Action {
	case "blacklist", "block":
		return action + "|" + actualMod
	default:
		return "NA"
	}
}

// getTargetDropInLine returns the target file action and exact line for the target state.
func getTargetDropInLine(targetState, mod string) (string, string) {
	targetFile, _ := parseStateString(targetState)
	action := strings.TrimSuffix(targetFile, "+permanent")
	switch action {
	case "block":
		return "block", "install " + mod + " /bin/false"
	case "blacklist":
		return "blacklist", "blacklist " + mod
	default:
		return "none", ""
	}
}

var (
	appliedKmodNoteID string
)

// ResetKmodDropInFirstWriteFlags resets the flags used to track the first drop-in write of an apply.
func ResetKmodDropInFirstWriteFlags() {
	appliedKmodNoteID = ""
}

// hasOtherAppliedKmodParams returns true if any note other than currentNoteID is currently managing a kmod parameter.
func hasOtherAppliedKmodParams(currentNoteID string, checkPermanent bool) bool {
	params := GetAllSavedParameters()
	for p, pnotes := range params {
		if strings.HasPrefix(p, "kmod:") {
			for _, entry := range pnotes.AllNotes {
				if entry.NoteID != "start" && entry.NoteID != "" && entry.NoteID != currentNoteID {
					isPerm := strings.Contains(entry.Value, "+permanent")
					if checkPermanent && isPerm {
						return true
					}
					if !checkPermanent && !isPerm {
						return true
					}
				}
			}
		}
	}
	return false
}

// HasActiveKmodConfig inspects Saptune's saved parameter state to determine
// if any transient or permanent kmod entries are currently active across all applied notes.
func HasActiveKmodConfig() (hasTransient bool, hasPermanent bool) {
	params := GetAllSavedParameters()
	for p, pnotes := range params {
		if strings.HasPrefix(p, "kmod:") {
			if len(pnotes.AllNotes) > 0 {
				lastEntry := pnotes.AllNotes[len(pnotes.AllNotes)-1]
				if lastEntry.NoteID != "start" && lastEntry.NoteID != "" {
					val := lastEntry.Value
					if strings.Contains(val, "+permanent") {
						hasPermanent = true
					} else if val != "" && val != "none|unloaded" {
						hasTransient = true
					}
				}
			}
		}
	}
	return hasTransient, hasPermanent
}

// CleanStaleKmodDropIns removes drop-in files from disk if no corresponding
// active kmod configuration exists across all applied notes.
func CleanStaleKmodDropIns() {
	hasTransient, hasPermanent := HasActiveKmodConfig()

	if !hasTransient {
		if st, err := os.Stat(KmodDropInTransientPath); err == nil && !st.IsDir() {
			system.WarningLog("kmod: removing stale drop-in '%s'", KmodDropInTransientPath)
			_ = os.Remove(KmodDropInTransientPath)
		}
	}

	if !hasPermanent {
		if st, err := os.Stat(KmodDropInPermanentPath); err == nil && !st.IsDir() {
			system.WarningLog("kmod: removing stale drop-in '%s'", KmodDropInPermanentPath)
			_ = os.Remove(KmodDropInPermanentPath)
		}
	}
}

// updateKmodDropIns updates both drop-in files to ensure only targetLine is present for mod.
func updateKmodDropIns(mod, targetLine string, isPermanent bool) error {
	transLines, err := system.ReadDropInFile(KmodDropInTransientPath)
	if err != nil {
		system.WarningLog("kmod: cannot read %s - %v", KmodDropInTransientPath, err)
		return nil
	}
	permLines, err := system.ReadDropInFile(KmodDropInPermanentPath)
	if err != nil {
		system.WarningLog("kmod: cannot read %s - %v", KmodDropInPermanentPath, err)
		return nil
	}

	keepTrans := ""
	keepPerm := ""
	if targetLine != "" {
		if isPermanent {
			keepPerm = targetLine
		} else {
			keepTrans = targetLine
		}
	}

	transLines, _ = system.RemoveDropInModuleLines(transLines, mod, keepTrans)
	if keepTrans != "" {
		transLines = system.AddDropInModuleLine(transLines, keepTrans)
	}

	permLines, _ = system.RemoveDropInModuleLines(permLines, mod, keepPerm)
	if keepPerm != "" {
		permLines = system.AddDropInModuleLine(permLines, keepPerm)
	}

	if err := system.WriteDropInFileAtomic(KmodDropInTransientPath, system.KmodTransientHeader, transLines); err != nil {
		system.WarningLog("kmod: cannot update %s - %v", KmodDropInTransientPath, err)
	}
	if err := system.WriteDropInFileAtomic(KmodDropInPermanentPath, system.KmodPermanentHeader, permLines); err != nil {
		system.WarningLog("kmod: cannot update %s - %v", KmodDropInPermanentPath, err)
	}
	return nil
}

// SetKmodVal applies or reverts the kernel module drop-in configuration.
func SetKmodVal(noteID, key, targetState, cfgval string, revert bool) error {
	if targetState == "NA" {
		return nil
	}
	mod, err := cleanKmodModule(key)
	if err != nil {
		return nil
	}
	if targetState == "PNA" || system.IsModuleBuiltin(mod) {
		system.WarningLog("kmod: module '%s' is compiled-in and cannot be handled. Skipping...", mod)
		return nil
	}

	targetAction, targetLine := getTargetDropInLine(targetState, mod)

	entry, _ := ParseKmodEntry(noteID, key, cfgval)
	if revert && entry != nil && entry.Permanent && targetAction == "none" {
		system.DebugLog("kmod: revert skipped for module '%s' due to permanent option", mod)
		return nil
	}

	if !revert && (targetAction == "blacklist" || targetAction == "block") && system.IsModuleLoaded(mod) {
		system.WarningLog("kmod: module '%s' is currently loaded. It must be unloaded first for %s to take effect.", mod, targetAction)
	}

	if !revert && appliedKmodNoteID != noteID {
		appliedKmodNoteID = noteID
		if !hasOtherAppliedKmodParams(noteID, false) {
			if st, err := os.Stat(KmodDropInTransientPath); err == nil && !st.IsDir() {
				system.WarningLog("kmod: removing existing drop-in '%s' before applying configuration", KmodDropInTransientPath)
				_ = os.Remove(KmodDropInTransientPath)
			}
		}
		if !hasOtherAppliedKmodParams(noteID, true) {
			if st, err := os.Stat(KmodDropInPermanentPath); err == nil && !st.IsDir() {
				system.WarningLog("kmod: removing existing drop-in '%s' before applying configuration", KmodDropInPermanentPath)
				_ = os.Remove(KmodDropInPermanentPath)
			}
		}
	}

	if !revert {
		system.CheckModuleDropInShadowing(mod)
	}

	isPermanent := !revert && entry != nil && entry.Permanent
	if err := updateKmodDropIns(mod, targetLine, isPermanent); err != nil {
		return err
	}

	if !revert && targetAction == "block" {
		if cfg, err := system.ModprobeConfig(); err == nil {
			effective, multiple, winning := system.CheckEffectiveInstallStatus(cfg, mod)
			if !effective && winning != "" {
				system.ErrorLog("kmod: module '%s': another install entry '%s' takes precedence over saptune's block rule", mod, winning)
			} else if multiple {
				system.WarningLog("kmod: module '%s': multiple install entries found, saptune's block rule takes precedence", mod)
			}
		}
	}
	if !revert && targetAction == "blacklist" {
		if cfg, err := system.ModprobeConfig(); err == nil {
			installEntries := system.GetModuleInstallEntries(cfg, mod)
			if len(installEntries) > 0 {
				system.ErrorLog("kmod: module '%s': another install entry '%s' takes precedence over saptune's blacklist rule", mod, installEntries[0])
			}
		}
	}
	return nil
}

// GetKmodInform checks for effective configuration precedence, already loaded on apply,
// permanent flags, invalid entry flags, and compiled-in (builtin) flags, returning information tokens separated by § for footnote preparation.
func GetKmodInform(noteID, key, cfgval, targetState string) string {
	if targetState == "NA" {
		return "invalid"
	}
	mod, err := cleanKmodModule(key)
	if err != nil {
		return "invalid"
	}
	if targetState == "PNA" || system.IsModuleBuiltin(mod) {
		return "builtin"
	}
	entry, err := ParseKmodEntry(noteID, key, cfgval)
	if err != nil {
		return "invalid"
	}

	actualDropInState := system.GetDropInModuleState(KmodDropInTransientPath, KmodDropInPermanentPath, mod)

	var flags []string
	addFlag := func(f string) {
		for _, existing := range flags {
			if existing == f {
				return
			}
		}
		flags = append(flags, f)
	}

	lastNote := ParameterLastNote(key)
	if lastNote == "" {
		lastNote = ParameterLastNote("kmod:" + mod)
	}
	if lastNote == "" {
		lastNote = ParameterLastNote(mod)
	}
	isSuperseded := lastNote != "" && lastNote != "start" && lastNote != noteID

	if actualDropInState != "none" && !isSuperseded {
		// Check drop-in file shadowing for this specific module
		shadowedByHigher, shadowsLower := system.CheckModuleDropInShadowing(mod)
		if shadowedByHigher && !entry.Permanent {
			addFlag("masked")
		}
		if shadowsLower {
			addFlag("shadowing_warning")
		}

		// Footnotes 18 & 23: check effective configuration and install conflict
		targetAction, targetLine := getTargetDropInLine(targetState, mod)
		if targetLine != "" {
			if cfg, err := system.ModprobeConfig(); err == nil {
				if targetAction == "block" {
					effective, multiple, _ := system.CheckEffectiveInstallStatus(cfg, mod)
					if !effective {
						addFlag("masked")
					} else if multiple {
						addFlag("multiple_install")
					}
				} else {
					if !system.IsEffectiveConfigEntry(cfg, targetLine) || len(system.GetModuleInstallEntries(cfg, mod)) > 0 {
						addFlag("masked")
					}
				}
			}
		}
	}

	// Footnote 19: check if recorded as loaded on apply and still loaded
	pnotes := GetSavedParameterNotes(key)
	if len(pnotes.AllNotes) == 0 {
		pnotes = GetSavedParameterNotes("kmod:" + mod)
	}
	if len(pnotes.AllNotes) > 0 && pnotes.AllNotes[0].NoteID == "start" {
		startState := pnotes.AllNotes[0].Value
		_, startMod := parseStateString(startState)
		if strings.HasPrefix(startMod, "loaded:") && system.IsModuleLoaded(mod) {
			addFlag("loaded_on_apply")
		}
	}

	// Footnote 20: check if permanent is set
	if entry.Permanent {
		addFlag("permanent")
	}

	return strings.Join(flags, "§")
}
