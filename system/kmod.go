package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
)

// Default drop-in file paths and headers for saptune kernel module management.
const (
	KmodTransientDropIn = "/run/modprobe.d/80-saptune.conf"
	KmodPermanentDropIn = "/etc/modprobe.d/80-saptune-permanent.conf"

	KmodTransientHeader = "# This file is managed by saptune. Do not touch!"
	KmodPermanentHeader = "# This file was created by saptune as permanent drop-in. Do not touch!"
)

// SysfsModuleDir and ModprobeCmd are package variables so unit tests can override them.
var (
	SysfsModuleDir          = "/sys/module"
	ModprobeCmd             = "/usr/sbin/modprobe"
	modprobeCmd             = ModprobeCmd
	KmodTransientDropInPath = KmodTransientDropIn
	KmodPermanentDropInPath = KmodPermanentDropIn
)

// NormalizeModuleName normalises '-' to '_' for /sys/module lookups.
func NormalizeModuleName(module string) string {
	return strings.ReplaceAll(module, "-", "_")
}

// IsModuleLoaded checks whether a module is loaded into the kernel via sysfs.
func IsModuleLoaded(module string) bool {
	norm := NormalizeModuleName(module)
	initstatePath := path.Join(SysfsModuleDir, norm, "initstate")
	content, err := os.ReadFile(initstatePath)
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(content))
	return state == "live" || state == "coming"
}

// IsModuleBuiltin checks whether a module is built into the kernel.
func IsModuleBuiltin(module string) bool {
	norm := NormalizeModuleName(module)
	modDir := path.Join(SysfsModuleDir, norm)
	stat, err := os.Stat(modDir)
	if err != nil || !stat.IsDir() {
		return false
	}
	initstatePath := path.Join(modDir, "initstate")
	_, err = os.Stat(initstatePath)
	return os.IsNotExist(err)
}

// GetModuleParams returns all runtime parameter names and values from sysfs.
func GetModuleParams(module string) (map[string]string, error) {
	norm := NormalizeModuleName(module)
	paramsDir := path.Join(SysfsModuleDir, norm, "parameters")
	entries, err := os.ReadDir(paramsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	params := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		pPath := path.Join(paramsDir, entry.Name())
		content, err := os.ReadFile(pPath)
		if err != nil {
			DebugLog("GetModuleParams: cannot read %s - %v", pPath, err)
			continue
		}
		params[entry.Name()] = strings.TrimRight(string(content), "\n")
	}
	return params, nil
}

// SetModuleParam writes a parameter value directly into sysfs.
func SetModuleParam(module, paramName, value string) error {
	norm := NormalizeModuleName(module)
	paramPath := path.Join(SysfsModuleDir, norm, "parameters", paramName)
	DebugLog("SetModuleParam: writing '%s' to %s", value, paramPath)
	err := os.WriteFile(paramPath, []byte(value+"\n"), 0644)
	if err != nil {
		return ErrorLog("SetModuleParam: failed writing '%s' to %s - %v", value, paramPath, err)
	}
	return nil
}

// ModprobeLoad executes modprobe to load a module with optional parameters.
func ModprobeLoad(module string, params map[string]string) error {
	args := []string{module}
	if len(params) > 0 {
		var keys []string
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			args = append(args, fmt.Sprintf("%s=%s", k, params[k]))
		}
	}
	DebugLog("ModprobeLoad: executing %s %v", ModprobeCmd, args)
	out, err := exec.Command(ModprobeCmd, args...).CombinedOutput()
	if err != nil {
		return ErrorLog("ModprobeLoad: %s %v failed (%v) - %s", ModprobeCmd, args, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ModprobeRemove executes modprobe -r to unload a module.
func ModprobeRemove(module string) error {
	DebugLog("ModprobeRemove: executing %s -r %s", ModprobeCmd, module)
	out, err := exec.Command(ModprobeCmd, "-r", module).CombinedOutput()
	if err != nil {
		return ErrorLog("ModprobeRemove: %s -r %s failed (%v) - %s", ModprobeCmd, module, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ModprobeConfig runs modprobe -c and returns the effective configuration output.
func ModprobeConfig() (string, error) {
	out, err := exec.Command(ModprobeCmd, "-c").CombinedOutput()
	if err != nil {
		return "", ErrorLog("ModprobeConfig: %s -c failed (%v) - %s", ModprobeCmd, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ReadDropInFile reads all lines of a drop-in configuration file.
func ReadDropInFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// filterActiveEntries filters lines returning only non-comment, non-empty configuration lines.
func filterActiveEntries(lines []string) []string {
	var active []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		active = append(active, trimmed)
	}
	return active
}

// WriteDropInFileAtomic writes lines atomically to filePath with the provided header.
// If there are no active configuration lines remaining, the file is deleted.
func WriteDropInFileAtomic(filePath, header string, lines []string) error {
	active := filterActiveEntries(lines)
	if len(active) == 0 {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			WarningLog("WriteDropInFileAtomic: failed to remove empty file %s - %v", filePath, err)
			return err
		}
		DebugLog("WriteDropInFileAtomic: deleted empty drop-in %s", filePath)
		return nil
	}

	dir := path.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		WarningLog("WriteDropInFileAtomic: failed to create directory %s - %v", dir, err)
		return err
	}

	tmpFile, err := os.CreateTemp(dir, path.Base(filePath)+".tmp*")
	if err != nil {
		WarningLog("WriteDropInFileAtomic: failed to create temp file in %s - %v", dir, err)
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	var content strings.Builder
	content.WriteString(strings.TrimSpace(header) + "\n\n")
	for _, entry := range active {
		content.WriteString(entry + "\n")
	}

	if _, err := tmpFile.WriteString(content.String()); err != nil {
		tmpFile.Close()
		WarningLog("WriteDropInFileAtomic: failed writing temp file %s - %v", tmpName, err)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		WarningLog("WriteDropInFileAtomic: failed closing temp file %s - %v", tmpName, err)
		return err
	}

	if err := os.Rename(tmpName, filePath); err != nil {
		WarningLog("WriteDropInFileAtomic: failed renaming %s to %s - %v", tmpName, filePath, err)
		return err
	}
	DebugLog("WriteDropInFileAtomic: successfully updated %s", filePath)
	return nil
}

// FileHasSaptuneHeader checks if the first non-empty line of filePath matches expectedHeader.
func FileHasSaptuneHeader(filePath, expectedHeader string) bool {
	lines, err := ReadDropInFile(filePath)
	if err != nil || len(lines) == 0 {
		return false
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed == expectedHeader
		}
	}
	return false
}

// isKmodManagedLine reports whether line is an exact blacklist or install /bin/false line for module.
func isKmodManagedLine(line, module string) bool {
	trimmed := strings.TrimSpace(line)
	bl := "blacklist " + module
	ins := "install " + module + " /bin/false"
	return trimmed == bl || trimmed == ins
}

// RemoveDropInModuleLines removes any blacklist or block lines for module except keepLine.
func RemoveDropInModuleLines(lines []string, module, keepLine string) ([]string, bool) {
	var kept []string
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isKmodManagedLine(trimmed, module) && trimmed != keepLine {
			InfoLog("kmod: removing line '%s'", trimmed)
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	return kept, removed
}

// AddDropInModuleLine adds newLine to lines if not already present.
func AddDropInModuleLine(lines []string, newLine string) []string {
	trimmedNew := strings.TrimSpace(newLine)
	if trimmedNew == "" {
		return lines
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == trimmedNew {
			return lines
		}
	}
	return append(lines, trimmedNew)
}

// GetDropInModuleState reads drop-in files to determine if module is "block", "blacklist", or "none".
func GetDropInModuleState(transientPath, permanentPath, module string) string {
	checkFile := func(p string) string {
		lines, err := ReadDropInFile(p)
		if err != nil {
			return ""
		}
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "install "+module+" /bin/false" {
				return "block"
			}
			if trimmed == "blacklist "+module {
				return "blacklist"
			}
		}
		return ""
	}

	if st := checkFile(permanentPath); st != "" {
		return st + "+permanent"
	}
	if st := checkFile(transientPath); st != "" {
		return st
	}
	return "none"
}

// IsEffectiveConfigEntry checks if targetLine is present as a whole line in modprobe -c output.
func IsEffectiveConfigEntry(configOutput, targetLine string) bool {
	target := strings.TrimSpace(targetLine)
	scanner := bufio.NewScanner(strings.NewReader(configOutput))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == target {
			return true
		}
	}
	return false
}

// ModprobeDirectories lists the modprobe.d search directories in order of precedence.
var ModprobeDirectories = []string{
	"/etc/modprobe.d",
	"/run/modprobe.d",
	"/usr/local/lib/modprobe.d",
	"/usr/lib/modprobe.d",
	"/lib/modprobe.d",
}

// FileContainsModule reports whether filePath contains any configuration entry for module.
func FileContainsModule(filePath, module string) bool {
	lines, err := ReadDropInFile(filePath)
	if err != nil {
		return false
	}
	norm := NormalizeModuleName(module)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && NormalizeModuleName(fields[1]) == norm {
			return true
		}
	}
	return false
}

// CheckModuleDropInShadowing inspects ModprobeDirectories for files that shadow
// or are shadowed by saptune's drop-in configuration files for a specific module.
// If module is empty, checks for any file existence.
// Returns:
//
//	shadowedByHigher: true if a higher-priority directory contains 80-saptune.conf with an entry for module (ERROR)
//	shadowsLower: true if a lower-priority directory contains 80-saptune.conf or 80-saptune-permanent.conf with an entry for module (WARNING)
func CheckModuleDropInShadowing(module string) (shadowedByHigher bool, shadowsLower bool) {
	transName := path.Base(KmodTransientDropInPath)
	permName := path.Base(KmodPermanentDropInPath)

	transDir := path.Dir(KmodTransientDropInPath)
	permDir := path.Dir(KmodPermanentDropInPath)

	// Check transient:
	foundTransDir := false
	for _, dir := range ModprobeDirectories {
		if dir == transDir {
			foundTransDir = true
			continue
		}
		fPath := path.Join(dir, transName)
		if st, err := os.Stat(fPath); err == nil && !st.IsDir() {
			if module == "" || FileContainsModule(fPath, module) {
				if !foundTransDir {
					// Higher directory shadows saptune!
					ErrorLog("kmod: module '%s': '%s' shadows saptune transient configuration in '%s'", module, fPath, transDir)
					shadowedByHigher = true
				} else {
					// Lower directory is shadowed by saptune
					WarningLog("kmod: module '%s': '%s' is shadowed by saptune transient configuration in '%s'", module, fPath, transDir)
					shadowsLower = true
				}
			}
		}
	}

	// Check permanent:
	foundPermDir := false
	for _, dir := range ModprobeDirectories {
		if dir == permDir {
			foundPermDir = true
			continue
		}
		fPath := path.Join(dir, permName)
		if st, err := os.Stat(fPath); err == nil && !st.IsDir() {
			if module == "" || FileContainsModule(fPath, module) {
				if !foundPermDir {
					ErrorLog("kmod: module '%s': '%s' shadows saptune permanent configuration in '%s'", module, fPath, permDir)
					shadowedByHigher = true
				} else {
					WarningLog("kmod: module '%s': '%s' is shadowed by saptune permanent configuration in '%s'", module, fPath, permDir)
					shadowsLower = true
				}
			}
		}
	}

	return shadowedByHigher, shadowsLower
}

// CheckDropInShadowing inspects ModprobeDirectories for files that shadow
// or are shadowed by saptune's drop-in configuration files.
func CheckDropInShadowing() (shadowedByHigher bool, shadowsLower bool) {
	return CheckModuleDropInShadowing("")
}

// GetModuleInstallEntries returns all install entries found in configOutput for module.
func GetModuleInstallEntries(configOutput, module string) []string {
	norm := NormalizeModuleName(module)
	var installLines []string
	scanner := bufio.NewScanner(strings.NewReader(configOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "install ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				entryMod := NormalizeModuleName(fields[1])
				if entryMod == norm {
					installLines = append(installLines, line)
				}
			}
		}
	}
	return installLines
}

// CheckEffectiveInstallStatus checks modprobe -c for install entries for the specified module.
// Returns:
//
//	effective: true if saptune's block entry (install <mod> /bin/false) is the active/winning entry (i.e. the FIRST install entry found)
//	multiple: true if more than one install entry was found for this module
//	winningEntry: the first install entry found for this module
func CheckEffectiveInstallStatus(configOutput, module string) (effective bool, multiple bool, winningEntry string) {
	installLines := GetModuleInstallEntries(configOutput, module)
	if len(installLines) == 0 {
		return false, false, ""
	}

	winningEntry = installLines[0]
	fields := strings.Fields(winningEntry)
	if len(fields) >= 3 && fields[2] == "/bin/false" {
		effective = true
	} else {
		effective = false
	}
	multiple = len(installLines) > 1
	return effective, multiple, winningEntry
}
