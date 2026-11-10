package note

import (
	"encoding/json"
	"github.com/SUSE/saptune/system"
	"strings"
)

// section [kmod]

var (
	// KmodDropInTransientPath is the transient modprobe drop-in location.
	KmodDropInTransientPath = system.KmodTransientDropIn
	// KmodDropInPermanentPath is the permanent modprobe drop-in location.
	KmodDropInPermanentPath = system.KmodPermanentDropIn
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
		return "", system.ErrorLog("kmod: empty module name")
	}
	if strings.ContainsAny(mod, " \t\r\n") {
		return "", system.ErrorLog("kmod: module name '%s' contains whitespace", mod)
	}
	return mod, nil
}

// parseKmodActionAndOptions splits action[+opt1+opt2] into action and option tokens.
func parseKmodActionAndOptions(actToken string) (string, []string, error) {
	if actToken == "" {
		return "", nil, system.ErrorLog("kmod: empty action")
	}
	parts := strings.Split(actToken, "+")
	action := strings.ToLower(strings.TrimSpace(parts[0]))
	var opts []string
	for _, opt := range parts[1:] {
		trimmed := strings.ToLower(strings.TrimSpace(opt))
		if trimmed == "" {
			return "", nil, system.ErrorLog("kmod: empty option in '%s'", actToken)
		}
		opts = append(opts, trimmed)
	}
	return action, opts, nil
}

// validateKmodActionAndOpts validates options for the given action according to spec.
func validateKmodActionAndOpts(entry *KmodEntry, action string, opts []string) error {
	seen := make(map[string]bool)
	for _, opt := range opts {
		if seen[opt] {
			return system.ErrorLog("kmod: repeated option '%s' for module '%s'", opt, entry.Module)
		}
		seen[opt] = true

		switch opt {
		case "permanent":
			if action != "blacklist" && action != "block" {
				return system.ErrorLog("kmod: option 'permanent' not allowed for action '%s'", action)
			}
			entry.Permanent = true
		case "keep":
			if action != "load" && action != "unload" {
				return system.ErrorLog("kmod: option 'keep' not allowed for action '%s'", action)
			}
			entry.Keep = true
		case "no-reload":
			if action != "load" {
				return system.ErrorLog("kmod: option 'no-reload' not allowed for action '%s'", action)
			}
			entry.NoReload = true
		default:
			return system.ErrorLog("kmod: unknown option '%s' for action '%s'", opt, action)
		}
	}
	return nil
}

// parseKmodParams parses PARAM=VALUE tokens into a map.
func parseKmodParams(tokens []string) (map[string]string, error) {
	params := make(map[string]string)
	for _, tok := range tokens {
		p := strings.TrimSpace(tok)
		if p == "" {
			continue
		}
		eqIdx := strings.Index(p, "=")
		if eqIdx <= 0 || eqIdx == len(p)-1 {
			return nil, system.ErrorLog("kmod: invalid parameter pair '%s', expected PARAM=VALUE", p)
		}
		k := strings.TrimSpace(p[:eqIdx])
		v := strings.TrimSpace(p[eqIdx+1:])
		params[k] = v
	}
	return params, nil
}

// ParseKmodEntry parses a MODULE = value line from the [kmod] section.
// Load and unload actions are rejected for now until implemented.
func ParseKmodEntry(key, value string) (*KmodEntry, error) {
	mod, err := cleanKmodModule(key)
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return nil, system.ErrorLog("kmod: missing action for module '%s'", mod)
	}

	action, opts, err := parseKmodActionAndOptions(fields[0])
	if err != nil {
		return nil, err
	}

	entry := &KmodEntry{
		Module: mod,
		Action: action,
		Params: make(map[string]string),
	}

	if err := validateKmodActionAndOpts(entry, action, opts); err != nil {
		return nil, err
	}

	if len(fields) > 1 {
		if action != "load" {
			return nil, system.ErrorLog("kmod: parameters specified for non-load action '%s'", action)
		}
		params, err := parseKmodParams(fields[1:])
		if err != nil {
			return nil, err
		}
		entry.Params = params
	}

	switch action {
	case "blacklist", "block":
		return entry, nil
	case "load", "unload":
		return nil, system.ErrorLog("kmod: action '%s' for module '%s' is not supported yet", action, mod)
	default:
		return nil, system.ErrorLog("kmod: unknown action '%s' for module '%s'", action, mod)
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
func OptKmodVal(key, cfgval, actual string) string {
	if actual == "PNA" {
		return "PNA"
	}
	entry, err := ParseKmodEntry(key, cfgval)
	if err != nil {
		return "NA"
	}

	_, actualMod := parseStateString(actual)
	switch entry.Action {
	case "blacklist":
		return "blacklist|" + actualMod
	case "block":
		return "block|" + actualMod
	default:
		return "NA"
	}
}

// getTargetDropInLine returns the target file action and exact line for the target state.
func getTargetDropInLine(targetState, mod string) (string, string) {
	targetFile, _ := parseStateString(targetState)
	switch targetFile {
	case "block":
		return "block", "install " + mod + " /bin/false"
	case "blacklist":
		return "blacklist", "blacklist " + mod
	default:
		return "none", ""
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
func SetKmodVal(key, targetState, cfgval string, revert bool) error {
	mod, err := cleanKmodModule(key)
	if err != nil {
		return err
	}
	if targetState == "PNA" || system.IsModuleBuiltin(mod) {
		system.WarningLog("kmod: module '%s' is compiled-in and cannot be handled. Skipping...", mod)
		return nil
	}

	entry, _ := ParseKmodEntry(key, cfgval)
	if revert && entry != nil && entry.Permanent {
		system.DebugLog("kmod: revert skipped for module '%s' due to permanent option", mod)
		return nil
	}

	targetAction, targetLine := getTargetDropInLine(targetState, mod)

	if !revert && (targetAction == "blacklist" || targetAction == "block") && system.IsModuleLoaded(mod) {
		system.WarningLog("kmod: module '%s' is currently loaded. It must be unloaded first for %s to take effect.", mod, targetAction)
	}

	isPermanent := !revert && entry != nil && entry.Permanent
	return updateKmodDropIns(mod, targetLine, isPermanent)
}

// GetKmodInform checks for effective configuration precedence, already loaded on apply,
// and permanent flags, returning information tokens separated by § for footnote preparation.
func GetKmodInform(key, cfgval, targetState string) string {
	mod, err := cleanKmodModule(key)
	if err != nil {
		return ""
	}
	var flags []string

	// Footnote 18: check effective configuration
	_, targetLine := getTargetDropInLine(targetState, mod)
	if targetLine != "" {
		if cfg, err := system.ModprobeConfig(); err == nil {
			if !system.IsEffectiveConfigEntry(cfg, targetLine) {
				flags = append(flags, "masked")
			}
		}
	}

	// Footnote 19: check if recorded as loaded on apply and still loaded
	pnotes := GetSavedParameterNotes(key)
	if len(pnotes.AllNotes) > 0 && pnotes.AllNotes[0].NoteID == "start" {
		startState := pnotes.AllNotes[0].Value
		_, startMod := parseStateString(startState)
		if strings.HasPrefix(startMod, "loaded:") && system.IsModuleLoaded(mod) {
			flags = append(flags, "loaded_on_apply")
		}
	}

	// Footnote 20: check if permanent is set
	if entry, _ := ParseKmodEntry(key, cfgval); entry != nil && entry.Permanent {
		flags = append(flags, "permanent")
	}

	return strings.Join(flags, "§")
}
