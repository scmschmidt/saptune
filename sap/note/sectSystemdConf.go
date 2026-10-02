package note

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/SUSE/saptune/system"
	"github.com/SUSE/saptune/txtparser"
)

// Configurable drop-in and search directories (vars to allow test redirection)
var (
	SystemdSystemDropInDir      = "/run/systemd/system.conf.d"
	SystemdSystemDropInFile     = "/run/systemd/system.conf.d/80-saptune.conf"
	SystemdSystemStateFile      = "/var/lib/saptune/working/systemd-system.conf.json"
	SystemdSystemComplaintsFile = "/run/saptune/systemd-system-complaints.json"

	SystemdUserDropInDir      = "/run/systemd/user.conf.d"
	SystemdUserDropInFile     = "/run/systemd/user.conf.d/80-saptune.conf"
	SystemdUserStateFile      = "/var/lib/saptune/working/systemd-user.conf.json"
	SystemdUserComplaintsFile = "/run/saptune/systemd-user-complaints.json"

	SystemdSystemConfDirsHigher = []string{
		"/etc/systemd/system.conf.d",
	}
	SystemdSystemConfDirsLower = []string{
		"/usr/local/lib/systemd/system.conf.d",
		"/usr/lib/systemd/system.conf.d",
	}

	SystemdUserConfDirsHigher = []string{
		"/etc/systemd/user.conf.d",
	}
	SystemdUserConfDirsLower = []string{
		"/usr/local/lib/systemd/user.conf.d",
		"/usr/lib/systemd/user.conf.d",
	}

	NoteTuningSheets     = "/var/lib/saptune/working/notes"
	ExtraTuningSheets    = "/etc/saptune/extra"
	OverrideTuningSheets = "/etc/saptune/override"
)

const saptuneDropInBase = "80-saptune.conf"

var (
	regDropInLineNum = regexp.MustCompile(`80-saptune\.conf:(\d+)`)
	regKeyInLog      = regexp.MustCompile(`(?:Unknown lvalue|Unknown key name|Unknown assignment)\s+['"]([a-zA-Z0-9_.-]+)['"]`)
)

// CleanSystemdKey strips the (system.conf) or (user.conf) suffix from a key name
func CleanSystemdKey(key string) string {
	key = strings.TrimSuffix(key, " (system.conf)")
	key = strings.TrimSuffix(key, " (user.conf)")
	return strings.TrimSpace(key)
}

// OptSystemdConfVal returns the expected value for a systemd configuration tunable
func OptSystemdConfVal(val string) string {
	return val
}

// GetSystemdDropInPaths returns the drop-in dir, file path, and state file path for the given section
func GetSystemdDropInPaths(section string) (string, string, string) {
	if section == txtparser.INISectionSystemdUser {
		return SystemdUserDropInDir, SystemdUserDropInFile, SystemdUserStateFile
	}
	return SystemdSystemDropInDir, SystemdSystemDropInFile, SystemdSystemStateFile
}

// GetSystemdComplaintsFile returns the complaints file path for the given section
func GetSystemdComplaintsFile(section string) string {
	if section == txtparser.INISectionSystemdUser {
		return SystemdUserComplaintsFile
	}
	return SystemdSystemComplaintsFile
}

// WasDropInGeneratedForNote checks if saptune generated a drop-in for this note at apply
func WasDropInGeneratedForNote(stateFile, dropInFile, noteID string) bool {
	if noteID == "" {
		return false
	}
	appliedNotes := loadAppliedNotes(stateFile, dropInFile)
	for _, n := range appliedNotes {
		if n.NoteID == noteID {
			return true
		}
	}
	return false
}

// ExtractSystemdComplainedKeys extracts parameter keys mentioned in systemd error/warning logs
func ExtractSystemdComplainedKeys(dropInFile, logs string) []string {
	if logs == "" {
		return nil
	}

	keySet := make(map[string]bool)

	// Pattern A: match line numbers in 80-saptune.conf
	dropInContent, err := os.ReadFile(dropInFile)
	if err == nil {
		lines := strings.Split(string(dropInContent), "\n")
		matches := regDropInLineNum.FindAllStringSubmatch(logs, -1)
		for _, m := range matches {
			if len(m) > 1 {
				lineNum, err := strconv.Atoi(m[1])
				if err == nil && lineNum > 0 && lineNum <= len(lines) {
					rawLine := strings.TrimSpace(lines[lineNum-1])
					if !strings.HasPrefix(rawLine, "#") && !strings.HasPrefix(rawLine, ";") && strings.Contains(rawLine, "=") {
						eqIdx := strings.Index(rawLine, "=")
						k := strings.TrimSpace(rawLine[:eqIdx])
						if k != "" {
							keySet[k] = true
						}
					}
				}
			}
		}
	}

	// Pattern B: match quoted key names in error messages
	matchesB := regKeyInLog.FindAllStringSubmatch(logs, -1)
	for _, m := range matchesB {
		if len(m) > 1 && m[1] != "" && m[1] != "Manager" {
			keySet[m[1]] = true
		}
	}

	var result []string
	for k := range keySet {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// RecordSystemdComplaints saves complained keys for a note into /run/saptune
func RecordSystemdComplaints(section, noteID string, complainedKeys []string) {
	complaintsFile := GetSystemdComplaintsFile(section)
	complaints := loadComplaints(complaintsFile)
	if len(complainedKeys) > 0 {
		complaints[noteID] = complainedKeys
	} else {
		delete(complaints, noteID)
	}

	if len(complaints) == 0 {
		_ = os.Remove(complaintsFile)
		return
	}
	_ = os.MkdirAll(filepath.Dir(complaintsFile), 0755)
	data, err := json.Marshal(complaints)
	if err == nil {
		_ = os.WriteFile(complaintsFile, data, 0644)
	}
}

// RemoveSystemdComplaints removes complained keys for a note from /run/saptune
func RemoveSystemdComplaints(section, noteID string) {
	complaintsFile := GetSystemdComplaintsFile(section)
	complaints := loadComplaints(complaintsFile)
	delete(complaints, noteID)
	if len(complaints) == 0 {
		_ = os.Remove(complaintsFile)
		return
	}
	data, err := json.Marshal(complaints)
	if err == nil {
		_ = os.WriteFile(complaintsFile, data, 0644)
	}
}

// IsSystemdKeyComplained checks if systemd complained about cleanKey during apply
func IsSystemdKeyComplained(section, noteID, cleanKey string) bool {
	complaintsFile := GetSystemdComplaintsFile(section)
	complaints := loadComplaints(complaintsFile)
	if keys, ok := complaints[noteID]; ok {
		for _, k := range keys {
			if k == cleanKey {
				return true
			}
		}
	}
	// Also check if any note recorded a complaint for this key
	for _, keys := range complaints {
		for _, k := range keys {
			if k == cleanKey {
				return true
			}
		}
	}
	return false
}

func loadComplaints(file string) map[string][]string {
	res := make(map[string][]string)
	data, err := os.ReadFile(file)
	if err == nil {
		_ = json.Unmarshal(data, &res)
	}
	return res
}

var reportedMissingDropIns = make(map[string]bool)

// ResetReportedMissingDropIns resets the de-duplication cache of reported missing drop-in errors
func ResetReportedMissingDropIns() {
	reportedMissingDropIns = make(map[string]bool)
}

// GetSystemdConfVal reads the actual value from the drop-in file and checks for conflicts in other drop-ins.
func GetSystemdConfVal(section, key, noteID string) (string, string, error) {
	_, dropInFile, stateFile := GetSystemdDropInPaths(section)
	cleanKey := CleanSystemdKey(key)

	// Check if the drop-in file exists
	if _, err := os.Stat(dropInFile); os.IsNotExist(err) {
		// The drop-in is dynamically created on apply if needed and removed on revert.
		// It is only expected (and reported as an error) if an applied Note has the
		// new sections and indeed saptune generated this drop-in at apply and it is
		// now missing on verify.
		if WasDropInGeneratedForNote(stateFile, dropInFile, noteID) {
			if !reportedMissingDropIns[dropInFile] {
				reportedMissingDropIns[dropInFile] = true
				system.ErrorLog("Expected systemd drop-in file '%s' does not exist", dropInFile)
				fmt.Fprintf(os.Stderr, "ERROR: Expected systemd drop-in file '%s' does not exist\n", dropInFile)
			}
			// When the drop-in was removed after apply, the parameters that were present
			// in the missing drop-in are marked with footnote [21]
			higherMatches, lowerMatches := CheckSystemdConfDoubles(section, cleanKey)
			informParts := []string{"missing_param:" + dropInFile}
			if len(higherMatches) > 0 {
				informParts = append(informParts, "high:"+strings.Join(higherMatches, ";"))
			}
			if len(lowerMatches) > 0 {
				informParts = append(informParts, "low:"+strings.Join(lowerMatches, ";"))
			}
			return "", strings.Join(informParts, "§"), nil
		}
		return "", "missing_file", nil
	}

	// 1. Did systemd complain about this key during apply?
	if IsSystemdKeyComplained(section, noteID, cleanKey) {
		higherMatches, lowerMatches := CheckSystemdConfDoubles(section, cleanKey)
		informParts := []string{"complaint"}
		if len(higherMatches) > 0 {
			informParts = append(informParts, "high:"+strings.Join(higherMatches, ";"))
		}
		if len(lowerMatches) > 0 {
			informParts = append(informParts, "low:"+strings.Join(lowerMatches, ";"))
		}
		return "", strings.Join(informParts, "§"), nil
	}

	// 2. Read key from drop-in file
	dropInVal, found := readDropInKey(dropInFile, cleanKey)
	if !found {
		higherMatches, lowerMatches := CheckSystemdConfDoubles(section, cleanKey)
		informParts := []string{"missing_param:" + dropInFile}
		if len(higherMatches) > 0 {
			informParts = append(informParts, "high:"+strings.Join(higherMatches, ";"))
		}
		if len(lowerMatches) > 0 {
			informParts = append(informParts, "low:"+strings.Join(lowerMatches, ";"))
		}
		return "", strings.Join(informParts, "§"), nil
	}

	// 3. Key exists and was not complained about. Check conflicts in other drop-ins.
	higherMatches, lowerMatches := CheckSystemdConfDoubles(section, cleanKey)
	informParts := []string{}
	actVal := dropInVal
	if len(higherMatches) > 0 {
		actVal = "?"
		informParts = append(informParts, "high:"+strings.Join(higherMatches, ";"))
	}
	if len(lowerMatches) > 0 {
		informParts = append(informParts, "low:"+strings.Join(lowerMatches, ";"))
	}

	inform := strings.Join(informParts, "§")
	return actVal, inform, nil
}

// readDropInKey extracts the latest assigned value of a key from a drop-in file
func readDropInKey(filePath, targetKey string) (string, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(content), "\n")
	found := false
	val := ""
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if len(line) == 0 || line[0] == '#' || line[0] == ';' || line[0] == '[' {
			continue
		}
		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 {
			continue
		}
		k := strings.TrimSpace(line[:eqIdx])
		if k == targetKey {
			found = true
			val = strings.TrimLeft(line[eqIdx+1:], " \t")
		}
	}
	return val, found
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// CheckSystemdConfDoubles scans drop-in configuration directories for occurrences of cleanKey,
// categorizing matches into higher or lower priority than 80-saptune.conf.
func CheckSystemdConfDoubles(section, cleanKey string) ([]string, []string) {
	var higherMatches []string
	var lowerMatches []string

	runtimeDir, dropInFile, _ := GetSystemdDropInPaths(section)

	var higherDirs []string
	if section == txtparser.INISectionSystemdUser {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			userHomeDir := filepath.Join(home, ".config", "systemd", "user.conf.d")
			higherDirs = append(higherDirs, userHomeDir)
		}
		higherDirs = append(higherDirs, SystemdUserConfDirsHigher...)
	} else {
		higherDirs = append(higherDirs, SystemdSystemConfDirsHigher...)
	}

	var lowerDirs []string
	if section == txtparser.INISectionSystemdUser {
		lowerDirs = append(lowerDirs, SystemdUserConfDirsLower...)
	} else {
		lowerDirs = append(lowerDirs, SystemdSystemConfDirsLower...)
	}

	// 1. Scan Higher Priority Directories (/etc, ~/.config)
	for _, dir := range higherDirs {
		files := getConfFiles(dir)
		for _, f := range files {
			if fileContainsSystemdKey(f, cleanKey) {
				higherMatches = appendUnique(higherMatches, f)
			}
		}
	}

	// 2. Scan Runtime Directory (/run/systemd/{system,user}.conf.d)
	runtimeFiles := getConfFiles(runtimeDir)
	for _, f := range runtimeFiles {
		if f == dropInFile {
			continue // Saptune's own drop-in
		}
		base := filepath.Base(f)
		if fileContainsSystemdKey(f, cleanKey) {
			if base > saptuneDropInBase {
				higherMatches = appendUnique(higherMatches, f)
			} else if base < saptuneDropInBase {
				lowerMatches = appendUnique(lowerMatches, f)
			}
		}
	}

	// 3. Scan Lower Priority Directories (/usr/local/lib, /usr/lib)
	for _, dir := range lowerDirs {
		files := getConfFiles(dir)
		for _, f := range files {
			if fileContainsSystemdKey(f, cleanKey) {
				lowerMatches = appendUnique(lowerMatches, f)
			}
		}
	}

	return higherMatches, lowerMatches
}

func getConfFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".conf") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files
}

func fileContainsSystemdKey(filePath, targetKey string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	lines := strings.Split(string(content), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if len(line) == 0 || line[0] == '#' || line[0] == ';' || line[0] == '[' {
			continue
		}
		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 {
			continue
		}
		k := strings.TrimSpace(line[:eqIdx])
		if k == targetKey {
			return true
		}
	}
	return false
}

// SystemdNoteBlock represents a block of configuration lines in 80-saptune.conf for a specific Note
type SystemdNoteBlock struct {
	NoteID string   `json:"note_id"`
	Lines  []string `json:"lines"`
}

// AppliedNoteEntries stores raw note entries for multi-note parameter management
type AppliedNoteEntries struct {
	NoteID  string               `json:"note_id"`
	Entries []txtparser.INIEntry `json:"entries"`
}

// loadAppliedNotes loads note entries from the state file
func loadAppliedNotes(stateFile, dropInFile string) []AppliedNoteEntries {
	data, err := os.ReadFile(stateFile)
	if err == nil {
		var notes []AppliedNoteEntries
		if err := json.Unmarshal(data, &notes); err == nil && len(notes) > 0 {
			return notes
		}
	}
	// Fallback: parse from dropInFile if stateFile does not exist yet
	contentBytes, err := os.ReadFile(dropInFile)
	if err == nil {
		blocks := parseDropInBlocks(string(contentBytes))
		var notes []AppliedNoteEntries
		for _, b := range blocks {
			var entries []txtparser.INIEntry
			for _, l := range b.Lines {
				eqIdx := strings.Index(l, "=")
				if eqIdx == -1 {
					continue
				}
				k := strings.TrimSpace(l[:eqIdx])
				v := strings.TrimLeft(l[eqIdx+1:], " \t")
				if k != "" && v != "" {
					entries = append(entries, txtparser.INIEntry{
						Key:      k,
						Operator: txtparser.OperatorEqual,
						Value:    v,
					})
				}
			}
			if len(entries) > 0 {
				notes = append(notes, AppliedNoteEntries{
					NoteID:  b.NoteID,
					Entries: entries,
				})
			}
		}
		return notes
	}
	return nil
}

// saveAppliedNotes saves note entries to the state file
func saveAppliedNotes(stateFile string, notes []AppliedNoteEntries) error {
	if len(notes) == 0 {
		_ = os.Remove(stateFile)
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(stateFile), 0755)
	data, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0644)
}

// parseDropInBlocks parses an existing 80-saptune.conf into blocks per Note
func parseDropInBlocks(content string) []SystemdNoteBlock {
	var blocks []SystemdNoteBlock
	var currentBlock *SystemdNoteBlock

	lines := strings.Split(content, "\n")
	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "# SAP Note ") {
			if currentBlock != nil {
				blocks = append(blocks, *currentBlock)
			}
			noteID := strings.TrimSpace(strings.TrimPrefix(trimmed, "# SAP Note "))
			currentBlock = &SystemdNoteBlock{
				NoteID: noteID,
				Lines:  []string{},
			}
			continue
		}
		if currentBlock != nil {
			if trimmed != "" {
				currentBlock.Lines = append(currentBlock.Lines, trimmed)
			}
		}
	}
	if currentBlock != nil {
		blocks = append(blocks, *currentBlock)
	}
	return blocks
}

// renderDropInContent renders 80-saptune.conf containing all applied note blocks
func renderDropInContent(blocks []SystemdNoteBlock) string {
	var sb strings.Builder
	sb.WriteString("# This file is managed by saptune. Do not touch!\n[Manager]\n")
	for _, b := range blocks {
		if len(b.Lines) == 0 {
			continue
		}
		sb.WriteString("\n# SAP Note " + b.NoteID + "\n")
		for _, l := range b.Lines {
			sb.WriteString(l + "\n")
		}
	}
	return sb.String()
}

// GetNoteEffectiveSectionEntries parses the note file and override, returning whether the note was found on disk and its effective entries for section
func GetNoteEffectiveSectionEntries(section, noteID string) (bool, []txtparser.INIEntry) {
	if noteID == "" {
		return false, nil
	}

	var noteFile string
	candidates := []string{
		filepath.Join(ExtraTuningSheets, noteID+".conf"),
		filepath.Join(NoteTuningSheets, noteID),
		filepath.Join("/usr/share/saptune/notes", noteID),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			noteFile = c
			break
		}
	}
	if noteFile == "" {
		return false, nil
	}

	ini, err := txtparser.ParseINIFile(noteFile, false)
	if err != nil || ini == nil {
		return true, nil
	}

	// Check for overrides
	override, ow := txtparser.GetOverrides("ovw", noteID)

	var entries []txtparser.INIEntry
	for _, entry := range ini.AllValues {
		if entry.Section == section {
			if override && ow != nil && len(ow.KeyValue[section]) > 0 {
				if owEntry, ok := ow.KeyValue[section][entry.Key]; ok {
					if owEntry.Value == "" || owEntry.Value == "untouched" {
						continue
					}
					entry.Value = owEntry.Value
					if owEntry.Operator != "" {
						entry.Operator = owEntry.Operator
					}
				}
			}
			if entry.Value != "" && entry.Value != "untouched" {
				entries = append(entries, entry)
			}
		}
	}
	return true, entries
}

// ApplySystemdConf creates, updates, or reverts drop-in entries for a given section and note.
// If a parameter is defined in multiple Notes, the latest Note wins and its value is written in the drop-in.
// When a Note is reverted, parameters that are also defined in other applied Notes are kept with the value of the now latest Note.
// Returns (changed bool, err error).
func ApplySystemdConf(section, noteID string, entries []txtparser.INIEntry, revert bool) (bool, error) {
	dropInDir, dropInFile, stateFile := GetSystemdDropInPaths(section)

	appliedNotes := loadAppliedNotes(stateFile, dropInFile)
	_, dropInStatErr := os.Stat(dropInFile)
	dropInExists := (dropInStatErr == nil)

	if revert {
		if !dropInExists && len(appliedNotes) == 0 {
			system.NoticeLog("Drop-in file '%s' does not exist during revert", dropInFile)
			return false, nil
		}

		// Remove noteID from appliedNotes
		var newApplied []AppliedNoteEntries
		found := false
		for _, n := range appliedNotes {
			if n.NoteID == noteID {
				found = true
				continue
			}
			newApplied = append(newApplied, n)
		}
		if !found {
			// Note had no entries in this section
			return false, nil
		}
		appliedNotes = newApplied
	} else {
		// Apply mode: filter out untouched / empty entries
		var validEntries []txtparser.INIEntry
		for _, entry := range entries {
			if entry.Value == "" || entry.Value == "untouched" {
				continue
			}
			validEntries = append(validEntries, entry)
		}
		if len(validEntries) == 0 {
			return false, nil
		}

		// Update or append noteID in appliedNotes (preserving sequence, latest at the end)
		updated := false
		for i, n := range appliedNotes {
			if n.NoteID == noteID {
				appliedNotes[i].Entries = validEntries
				curr := appliedNotes[i]
				appliedNotes = append(appliedNotes[:i], appliedNotes[i+1:]...)
				appliedNotes = append(appliedNotes, curr)
				updated = true
				break
			}
		}
		if !updated {
			appliedNotes = append(appliedNotes, AppliedNoteEntries{
				NoteID:  noteID,
				Entries: validEntries,
			})
		}
	}

	// Save updated state
	_ = saveAppliedNotes(stateFile, appliedNotes)

	// If no applied notes remain for this section, remove the drop-in file
	if len(appliedNotes) == 0 {
		if err := os.Remove(dropInFile); err != nil && !os.IsNotExist(err) {
			system.ErrorLog("Failed to remove drop-in file '%s': %v", dropInFile, err)
			fmt.Fprintf(os.Stderr, "Failed to remove drop-in file '%s': %v\n", dropInFile, err)
			return false, err
		}
		return true, nil
	}

	// Determine for each parameter the latest applied Note that defines it (evaluating effective tuning)
	latestNoteForParam := make(map[string]string)
	latestEntryForParam := make(map[string]txtparser.INIEntry)
	var paramOrder []string

	for _, n := range appliedNotes {
		var effectiveEntries []txtparser.INIEntry
		noteFound, diskEntries := GetNoteEffectiveSectionEntries(section, n.NoteID)
		if noteFound {
			effectiveEntries = diskEntries
		} else {
			// Fallback: note file not found on disk (e.g. in unit tests)
			effectiveEntries = n.Entries
		}

		for _, entry := range effectiveEntries {
			cleanKey := CleanSystemdKey(entry.Key)
			if _, exists := latestNoteForParam[cleanKey]; !exists {
				paramOrder = append(paramOrder, cleanKey)
			}
			latestNoteForParam[cleanKey] = n.NoteID
			latestEntryForParam[cleanKey] = entry
		}
	}

	// Group parameters under their winning owner note
	linesPerNote := make(map[string][]string)
	for _, n := range appliedNotes {
		linesPerNote[n.NoteID] = []string{}
	}

	for _, cleanKey := range paramOrder {
		owner := latestNoteForParam[cleanKey]
		entry := latestEntryForParam[cleanKey]
		val := entry.Value
		if entry.Operator == txtparser.OperatorResetAssign {
			linesPerNote[owner] = append(linesPerNote[owner], cleanKey+"=")
			linesPerNote[owner] = append(linesPerNote[owner], cleanKey+"="+val)
		} else {
			linesPerNote[owner] = append(linesPerNote[owner], cleanKey+"="+val)
		}
	}

	var blocks []SystemdNoteBlock
	for _, n := range appliedNotes {
		lines := linesPerNote[n.NoteID]
		if len(lines) > 0 {
			blocks = append(blocks, SystemdNoteBlock{
				NoteID: n.NoteID,
				Lines:  lines,
			})
		}
	}

	if len(blocks) == 0 {
		if err := os.Remove(dropInFile); err != nil && !os.IsNotExist(err) {
			system.ErrorLog("Failed to remove drop-in file '%s': %v", dropInFile, err)
			fmt.Fprintf(os.Stderr, "Failed to remove drop-in file '%s': %v\n", dropInFile, err)
			return false, err
		}
		return true, nil
	}

	if err := os.MkdirAll(dropInDir, 0755); err != nil {
		system.ErrorLog("Failed to create drop-in directory '%s': %v", dropInDir, err)
		fmt.Fprintf(os.Stderr, "Failed to create drop-in directory '%s': %v\n", dropInDir, err)
		return false, err
	}

	rendered := renderDropInContent(blocks)

	if dropInExists {
		system.NoticeLog("Drop-in file '%s' already exists, overwriting", dropInFile)
	}
	if err := os.WriteFile(dropInFile, []byte(rendered), 0644); err != nil {
		system.ErrorLog("Failed to write drop-in file '%s': %v", dropInFile, err)
		fmt.Fprintf(os.Stderr, "Failed to write drop-in file '%s': %v\n", dropInFile, err)
		return false, err
	}

	return true, nil
}
