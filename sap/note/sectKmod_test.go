package note

import (
	"fmt"
	"github.com/SUSE/saptune/system"
	"os"
	"path"
	"strings"
	"testing"
)

func TestParseKmodEntryValid(t *testing.T) {
	cases := []struct {
		key      string
		val      string
		expected KmodEntry
	}{
		{
			key: "xen",
			val: "blacklist",
			expected: KmodEntry{
				Module:    "xen",
				Action:    "blacklist",
				Permanent: false,
			},
		},
		{
			key: "kmod:xen",
			val: "blacklist+permanent",
			expected: KmodEntry{
				Module:    "xen",
				Action:    "blacklist",
				Permanent: true,
			},
		},
		{
			key: "vfat",
			val: "block",
			expected: KmodEntry{
				Module:    "vfat",
				Action:    "block",
				Permanent: false,
			},
		},
		{
			key: "kmod:vfat",
			val: "block+permanent",
			expected: KmodEntry{
				Module:    "vfat",
				Action:    "block",
				Permanent: true,
			},
		},
		{
			key: "foobar",
			val: "BLOCK+PERMANENT",
			expected: KmodEntry{
				Module:    "foobar",
				Action:    "block",
				Permanent: true,
			},
		},
	}

	for _, c := range cases {
		entry, err := ParseKmodEntry("", c.key, c.val)
		if err != nil {
			t.Errorf("ParseKmodEntry(%s, %s) unexpected error: %v", c.key, c.val, err)
			continue
		}
		if entry.Module != c.expected.Module || entry.Action != c.expected.Action || entry.Permanent != c.expected.Permanent {
			t.Errorf("ParseKmodEntry(%s, %s): got %+v, expected %+v", c.key, c.val, entry, c.expected)
		}
	}
}

func TestParseKmodEntryInvalid(t *testing.T) {
	invalidCases := []struct {
		name string
		key  string
		val  string
	}{
		{"empty module", "", "blacklist"},
		{"whitespace in module", "xen guest", "blacklist"},
		{"empty value", "xen", ""},
		{"empty option", "xen", "blacklist+"},
		{"repeated permanent", "xen", "blacklist+permanent+permanent"},
		{"invalid option keep on blacklist", "xen", "blacklist+keep"},
		{"invalid option no-reload on blacklist", "xen", "blacklist+no-reload"},
		{"invalid option keep on block", "xen", "block+keep"},
		{"invalid option unknown", "xen", "blacklist+unknown"},
		{"params on blacklist", "xen", "blacklist foo=1"},
		{"params on block", "xen", "block foo=1"},
		{"unknown action", "xen", "foobar"},
		{"load rejected for now", "dummy", "load"},
		{"load+keep rejected for now", "dummy", "load+keep"},
		{"load with params rejected for now", "dummy", "load numdummies=2"},
		{"unload rejected for now", "dummy", "unload"},
		{"unload+keep rejected for now", "dummy", "unload+keep"},
	}

	for _, c := range invalidCases {
		entry, err := ParseKmodEntry("", c.key, c.val)
		if err == nil {
			t.Errorf("case '%s': ParseKmodEntry(%s, %s) expected error, got %+v", c.name, c.key, c.val, entry)
		}
	}

	// Verify error message format contains Note name and entry
	_, err := ParseKmodEntry("test_1", "cifs", "blacklist+keep")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "kmod: Note 'test_1', entry 'cifs = blacklist+keep': option 'keep' not allowed for action 'blacklist'"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error to contain '%s', got '%v'", expectedMsg, err)
	}
}

func TestGetKmodVal(t *testing.T) {
	origSysfs := system.SysfsModuleDir
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	defer func() {
		system.SysfsModuleDir = origSysfs
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
	}()

	tmpDir := t.TempDir()
	system.SysfsModuleDir = tmpDir
	KmodDropInTransientPath = tmpDir + "/transient.conf"
	KmodDropInPermanentPath = tmpDir + "/permanent.conf"

	// 1. Built-in module -> returns PNA
	builtinDir := path.Join(tmpDir, "builtin_mod")
	_ = os.MkdirAll(builtinDir, 0755)
	if val := GetKmodVal("builtin_mod", "blacklist"); val != "PNA" {
		t.Errorf("expected PNA for builtin, got '%s'", val)
	}

	// 2. Unloaded module, no drop-in -> none|unloaded
	if val := GetKmodVal("unloaded_mod", "blacklist"); val != "none|unloaded" {
		t.Errorf("expected 'none|unloaded', got '%s'", val)
	}

	// 3. Drop-in added: transient blacklist
	_ = system.WriteDropInFileAtomic(KmodDropInTransientPath, system.KmodTransientHeader, []string{"blacklist unloaded_mod"})
	if val := GetKmodVal("unloaded_mod", "blacklist"); val != "blacklist|unloaded" {
		t.Errorf("expected 'blacklist|unloaded', got '%s'", val)
	}

	// 4. Loaded module with parameters
	loadedDir := path.Join(tmpDir, "loaded_mod")
	_ = os.MkdirAll(path.Join(loadedDir, "parameters"), 0755)
	_ = os.WriteFile(path.Join(loadedDir, "initstate"), []byte("live\n"), 0644)
	_ = os.WriteFile(path.Join(loadedDir, "parameters", "debug"), []byte("Y\n"), 0644)
	val := GetKmodVal("loaded_mod", "block")
	expected := "none|loaded:{\"debug\":\"Y\"}"
	if val != expected {
		t.Errorf("expected '%s', got '%s'", expected, val)
	}
}

func TestOptKmodVal(t *testing.T) {
	// 1. PNA stays PNA
	if val := OptKmodVal("", "test_mod", "blacklist", "PNA"); val != "PNA" {
		t.Errorf("expected 'PNA', got '%s'", val)
	}

	// 2. Blacklist replaces file part, keeps unloaded
	if val := OptKmodVal("", "test_mod", "blacklist", "none|unloaded"); val != "blacklist|unloaded" {
		t.Errorf("expected 'blacklist|unloaded', got '%s'", val)
	}

	// 3. Block replaces file part, keeps loaded params
	actual := "none|loaded:{\"debug\":\"Y\",\"timeout\":\"30\"}"
	expected := "block|loaded:{\"debug\":\"Y\",\"timeout\":\"30\"}"
	if val := OptKmodVal("", "test_mod", "block", actual); val != expected {
		t.Errorf("expected '%s', got '%s'", expected, val)
	}

	// 4. Permanent action includes +permanent
	if val := OptKmodVal("", "test_mod", "blacklist+permanent", "none|unloaded"); val != "blacklist+permanent|unloaded" {
		t.Errorf("expected 'blacklist+permanent|unloaded', got '%s'", val)
	}
	if val := OptKmodVal("", "test_mod", "block+permanent", "none|unloaded"); val != "block+permanent|unloaded" {
		t.Errorf("expected 'block+permanent|unloaded', got '%s'", val)
	}

	// 5. Invalid config returns NA
	if val := OptKmodVal("", "test_mod", "invalid_action", "none|unloaded"); val != "NA" {
		t.Errorf("expected 'NA', got '%s'", val)
	}
}

func TestSetKmodVal(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
	}()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	// 1. Apply transient blacklist
	if err := SetKmodVal("", "xen", "blacklist|unloaded", "blacklist", false); err != nil {
		t.Fatalf("SetKmodVal apply failed: %v", err)
	}
	lines, _ := system.ReadDropInFile(KmodDropInTransientPath)
	if len(lines) == 0 || lines[len(lines)-1] != "blacklist xen" {
		t.Errorf("expected transient drop-in to contain 'blacklist xen', got %+v", lines)
	}

	// 2. Revert transient blacklist to none|unloaded -> file deleted
	if err := SetKmodVal("", "xen", "none|unloaded", "blacklist", true); err != nil {
		t.Fatalf("SetKmodVal revert failed: %v", err)
	}
	if _, err := os.Stat(KmodDropInTransientPath); !os.IsNotExist(err) {
		t.Errorf("expected transient file to be deleted on revert to none")
	}

	// 3. Apply permanent block
	if err := SetKmodVal("", "vfat", "block|unloaded", "block+permanent", false); err != nil {
		t.Fatalf("SetKmodVal permanent apply failed: %v", err)
	}
	permLines, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permLines) == 0 || permLines[len(permLines)-1] != "install vfat /bin/false" {
		t.Errorf("expected permanent drop-in to contain 'install vfat /bin/false', got %+v", permLines)
	}

	// 4. Revert of permanent is skipped
	if err := SetKmodVal("", "vfat", "none|unloaded", "block+permanent", true); err != nil {
		t.Fatalf("SetKmodVal permanent revert failed: %v", err)
	}
	permLinesAfter, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permLinesAfter) == 0 || permLinesAfter[len(permLinesAfter)-1] != "install vfat /bin/false" {
		t.Errorf("expected permanent drop-in to retain 'install vfat /bin/false', got %+v", permLinesAfter)
	}

	// 4b. Revert of permanent note when an earlier note's state is restored (LIFO precedence):
	// permanent entry in /etc is removed, earlier note's state is written to /run
	if err := SetKmodVal("", "vfat", "blacklist|unloaded", "block+permanent", true); err != nil {
		t.Fatalf("SetKmodVal revert permanent to earlier note failed: %v", err)
	}
	if _, err := os.Stat(KmodDropInPermanentPath); !os.IsNotExist(err) {
		t.Errorf("expected permanent file to be removed when earlier note takes over")
	}
	transLinesRestored, _ := system.ReadDropInFile(KmodDropInTransientPath)
	if len(transLinesRestored) == 0 || transLinesRestored[len(transLinesRestored)-1] != "blacklist vfat" {
		t.Errorf("expected transient file to contain restored earlier note 'blacklist vfat', got %+v", transLinesRestored)
	}

	// 5. Stale line cleanup: module changed from block+permanent to blacklist transient
	if err := SetKmodVal("", "vfat", "blacklist|unloaded", "blacklist", false); err != nil {
		t.Fatalf("SetKmodVal cleanup apply failed: %v", err)
	}
	// permanent file should now be deleted because vfat was its only entry
	if _, err := os.Stat(KmodDropInPermanentPath); !os.IsNotExist(err) {
		t.Errorf("expected permanent file to be deleted after stale entry removed")
	}
	// transient file should have blacklist vfat
	transLines, _ := system.ReadDropInFile(KmodDropInTransientPath)
	if len(transLines) == 0 || transLines[len(transLines)-1] != "blacklist vfat" {
		t.Errorf("expected transient file to contain 'blacklist vfat', got %+v", transLines)
	}

	// 6. Revert to earlier note state writes to transient file
	if err := SetKmodVal("", "vfat", "block|unloaded", "blacklist", true); err != nil {
		t.Fatalf("SetKmodVal restore earlier note failed: %v", err)
	}
	transLinesRestored, _ = system.ReadDropInFile(KmodDropInTransientPath)
	if len(transLinesRestored) == 0 || transLinesRestored[len(transLinesRestored)-1] != "install vfat /bin/false" {
		t.Errorf("expected restored earlier note line in transient file, got %+v", transLinesRestored)
	}

	// 7. Compiled-in module is skipped
	builtinDir := path.Join(system.SysfsModuleDir, "mybuiltin")
	_ = os.MkdirAll(builtinDir, 0755)
	if err := SetKmodVal("", "mybuiltin", "PNA", "blacklist", false); err != nil {
		t.Errorf("SetKmodVal builtin returned error: %v", err)
	}
}

func TestKmodINISettings(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		CleanUpParamFile("kmod:xen")
		CleanUpParamFile("kmod:vfat")
		CleanUpParamFile("kmod:cifs")
	}()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	origCmd := system.ModprobeCmd
	defer func() { system.ModprobeCmd = origCmd }()
	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "-c" ]; then
    cat "%s" 2>/dev/null
    cat "%s" 2>/dev/null
    exit 0
fi
exit 0
`, KmodDropInTransientPath, KmodDropInPermanentPath)
	_ = os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	system.ModprobeCmd = fakeScript

	iniContent := `[version]
ID = test_kmod_note
version = 1
date = 10.11.2026
reference = https://me.sap.com/notes/12345
description = Test kmod note

[kmod]
xen = blacklist
vfat = block+permanent
cifs = blacklist+keep
`
	iniPath := path.Join(tmpDir, "test_kmod.ini")
	if err := os.WriteFile(iniPath, []byte(iniContent), 0644); err != nil {
		t.Fatal(err)
	}

	noteObj := INISettings{ConfFilePath: iniPath, ID: "test_kmod_note"}
	initNote, err := noteObj.Initialise()
	if err != nil {
		t.Fatalf("Initialise failed: %v", err)
	}
	initINI := initNote.(INISettings)
	if initINI.SysctlParams["kmod:xen"] != "none|unloaded" {
		t.Errorf("expected 'none|unloaded', got '%s'", initINI.SysctlParams["kmod:xen"])
	}

	optNote, err := noteObj.Initialise()
	if err != nil {
		t.Fatalf("Initialise for opt failed: %v", err)
	}
	optNote, err = optNote.Optimise()
	if err != nil {
		t.Fatalf("Optimise failed: %v", err)
	}
	optINI := optNote.(INISettings)
	if optINI.SysctlParams["kmod:xen"] != "blacklist|unloaded" {
		t.Errorf("expected 'blacklist|unloaded', got '%s'", optINI.SysctlParams["kmod:xen"])
	}
	if optINI.SysctlParams["kmod:vfat"] != "block+permanent|unloaded" {
		t.Errorf("expected 'block+permanent|unloaded', got '%s'", optINI.SysctlParams["kmod:vfat"])
	}
	if optINI.SysctlParams["kmod:cifs"] != "NA" {
		t.Errorf("expected 'NA' for invalid kmod:cifs, got '%s'", optINI.SysctlParams["kmod:cifs"])
	}
	if initINI.Inform["kmod:vfat"] != "" {
		t.Errorf("expected init Inform for kmod:vfat to be empty, got '%s'", initINI.Inform["kmod:vfat"])
	}
	if !strings.Contains(optINI.Inform["kmod:vfat"], "permanent") {
		t.Errorf("expected opt Inform for kmod:vfat to contain 'permanent', got '%s'", optINI.Inform["kmod:vfat"])
	}
	if optINI.Inform["kmod:cifs"] != "invalid" {
		t.Errorf("expected opt Inform for kmod:cifs to be 'invalid', got '%s'", optINI.Inform["kmod:cifs"])
	}

	// Verify CompareNoteFields before apply
	_, compBefore, _ := CompareNoteFields(initINI, optINI, "test_kmod_note", "forVerify")
	if compBefore["SysctlParams[kmod:vfat]"].ExpectedValueJS != "block+permanent" {
		t.Errorf("expected 'block+permanent' in ExpectedValueJS before apply, got '%s'", compBefore["SysctlParams[kmod:vfat]"].ExpectedValueJS)
	}
	if compBefore["SysctlParams[kmod:vfat]"].ActualValueJS != "none|unloaded" {
		t.Errorf("expected 'none|unloaded' in ActualValueJS before apply, got '%s'", compBefore["SysctlParams[kmod:vfat]"].ActualValueJS)
	}
	if compBefore["SysctlParams[kmod:vfat]"].MatchExpectation {
		t.Errorf("expected MatchExpectation to be false before apply")
	}
	if !strings.Contains(compBefore["Inform[kmod:vfat]"].ExpectedValue.(string), "permanent") {
		t.Errorf("expected Inform[kmod:vfat] to contain 'permanent', got '%v'", compBefore["Inform[kmod:vfat]"].ExpectedValue)
	}
	if compBefore["SysctlParams[kmod:cifs]"].ExpectedValueJS != "NA" {
		t.Errorf("expected 'NA' in ExpectedValueJS for kmod:cifs, got '%s'", compBefore["SysctlParams[kmod:cifs]"].ExpectedValueJS)
	}
	if compBefore["SysctlParams[kmod:cifs]"].ActualValueJS != "" {
		t.Errorf("expected '' (empty) in ActualValueJS for kmod:cifs, got '%s'", compBefore["SysctlParams[kmod:cifs]"].ActualValueJS)
	}
	if compBefore["SysctlParams[kmod:cifs]"].MatchExpectation {
		t.Errorf("expected MatchExpectation to be false for kmod:cifs")
	}
	if compBefore["Inform[kmod:cifs]"].ExpectedValue != "invalid" {
		t.Errorf("expected Inform[kmod:cifs] to be 'invalid', got '%v'", compBefore["Inform[kmod:cifs]"].ExpectedValue)
	}

	// Apply
	applyINI := optINI.SetValuesToApply([]string{"kmod:xen", "kmod:vfat", "kmod:cifs"}).(INISettings)
	if err := applyINI.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify Actual values after apply
	appliedNote, err := noteObj.Initialise()
	if err != nil {
		t.Fatalf("Initialise after apply failed: %v", err)
	}
	appliedINI := appliedNote.(INISettings)
	if appliedINI.SysctlParams["kmod:xen"] != "blacklist|unloaded" {
		t.Errorf("expected 'blacklist|unloaded' after apply, got '%s'", appliedINI.SysctlParams["kmod:xen"])
	}
	if appliedINI.SysctlParams["kmod:vfat"] != "block+permanent|unloaded" {
		t.Errorf("expected 'block+permanent|unloaded' after apply, got '%s'", appliedINI.SysctlParams["kmod:vfat"])
	}
	if appliedINI.SysctlParams["kmod:cifs"] != "none|unloaded" {
		t.Errorf("expected 'none|unloaded' for kmod:cifs after apply, got '%s'", appliedINI.SysctlParams["kmod:cifs"])
	}

	// Verify CompareNoteFields after apply
	optAfterNote, err := appliedNote.Optimise()
	if err != nil {
		t.Fatalf("Optimise after apply failed: %v", err)
	}
	optAfterINI := optAfterNote.(INISettings)
	_, compAfter, _ := CompareNoteFields(appliedINI, optAfterINI, "test_kmod_note", "forVerify")
	if compAfter["SysctlParams[kmod:cifs]"].ExpectedValueJS != "NA" {
		t.Errorf("expected 'NA' in ExpectedValueJS for kmod:cifs after apply, got '%s'", compAfter["SysctlParams[kmod:cifs]"].ExpectedValueJS)
	}
	if compAfter["SysctlParams[kmod:cifs]"].ActualValueJS != "" {
		t.Errorf("expected '' (empty) in ActualValueJS for kmod:cifs after apply, got '%s'", compAfter["SysctlParams[kmod:cifs]"].ActualValueJS)
	}
	if compAfter["SysctlParams[kmod:cifs]"].MatchExpectation {
		t.Errorf("expected MatchExpectation to be false for kmod:cifs after apply")
	}
	if compAfter["SysctlParams[kmod:vfat]"].ExpectedValueJS != "block+permanent" {
		t.Errorf("expected 'block+permanent' in ExpectedValueJS after apply, got '%s'", compAfter["SysctlParams[kmod:vfat]"].ExpectedValueJS)
	}
	if compAfter["SysctlParams[kmod:vfat]"].ActualValueJS != "block+permanent|unloaded" {
		t.Errorf("expected 'block+permanent|unloaded' in ActualValueJS after apply, got '%s'", compAfter["SysctlParams[kmod:vfat]"].ActualValueJS)
	}
	if !compAfter["SysctlParams[kmod:vfat]"].MatchExpectation {
		t.Errorf("expected MatchExpectation to be true after apply")
	}
	if !strings.Contains(compAfter["Inform[kmod:vfat]"].ExpectedValue.(string), "permanent") {
		t.Errorf("expected Inform[kmod:vfat] to contain 'permanent', got '%v'", compAfter["Inform[kmod:vfat]"].ExpectedValue)
	}

	transLines, _ := system.ReadDropInFile(KmodDropInTransientPath)
	if len(transLines) == 0 || transLines[len(transLines)-1] != "blacklist xen" {
		t.Errorf("expected transient drop-in to contain 'blacklist xen', got %+v", transLines)
	}
	permLines, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permLines) == 0 || permLines[len(permLines)-1] != "install vfat /bin/false" {
		t.Errorf("expected permanent drop-in to contain 'install vfat /bin/false', got %+v", permLines)
	}

	// Revert
	revertINI := applyINI.SetValuesToApply([]string{"revert"}).(INISettings)
	if err := revertINI.Apply(); err != nil {
		t.Fatalf("Revert failed: %v", err)
	}

	// Transient file should be deleted because xen was reverted
	if _, err := os.Stat(KmodDropInTransientPath); !os.IsNotExist(err) {
		lines, _ := system.ReadDropInFile(KmodDropInTransientPath)
		t.Errorf("expected transient file to be deleted on revert, but exists with lines: %+v", lines)
	}
	// Permanent file should keep vfat because permanent
	permAfter, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permAfter) == 0 || permAfter[len(permAfter)-1] != "install vfat /bin/false" {
		t.Errorf("expected permanent drop-in to retain 'install vfat /bin/false', got %+v", permAfter)
	}
}

func TestKmodSyntaxErrorsUnifiedBehavior(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		CleanUpParamFile("kmod:xen guest")
		CleanUpParamFile("kmod:")
		CleanUpParamFile("kmod:xen")
	}()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	iniContent := `[version]
ID = test_syntax_err
version = 1
date = 18.11.2026
reference = https://me.sap.com/notes/12345
description = Test kmod syntax error note

[kmod]
xen guest = blacklist
= blacklist
xen =
`
	iniPath := path.Join(tmpDir, "test_syntax.ini")
	if err := os.WriteFile(iniPath, []byte(iniContent), 0644); err != nil {
		t.Fatal(err)
	}

	noteObj := INISettings{ConfFilePath: iniPath, ID: "test_syntax_err"}
	initNote, err := noteObj.Initialise()
	if err != nil {
		t.Fatalf("Initialise failed: %v", err)
	}
	initINI := initNote.(INISettings)

	optNote, err := noteObj.Initialise()
	if err != nil {
		t.Fatalf("Initialise for opt failed: %v", err)
	}
	optNote, err = optNote.Optimise()
	if err != nil {
		t.Fatalf("Optimise failed: %v", err)
	}
	optINI := optNote.(INISettings)

	// Check that all invalid entries have Expected == NA and Inform == invalid
	for _, key := range []string{"kmod:xen guest", "kmod:", "kmod:xen"} {
		if val, ok := optINI.SysctlParams[key]; !ok || val != "NA" {
			t.Errorf("expected SysctlParams[%s] == 'NA', got '%s'", key, val)
		}
		if inf, ok := optINI.Inform[key]; !ok || inf != "invalid" {
			t.Errorf("expected Inform[%s] == 'invalid', got '%s'", key, inf)
		}
	}

	// Verify CompareNoteFields before apply: Expected is NA, Actual is "", match is false
	_, comp, _ := CompareNoteFields(initINI, optINI, "test_syntax_err", "forVerify")
	for _, key := range []string{"kmod:xen guest", "kmod:", "kmod:xen"} {
		ckey := "SysctlParams[" + key + "]"
		if c, ok := comp[ckey]; !ok {
			t.Errorf("missing comparison for %s", ckey)
		} else {
			if c.ExpectedValueJS != "NA" {
				t.Errorf("expected ExpectedValueJS == 'NA' for %s, got '%s'", key, c.ExpectedValueJS)
			}
			if c.ActualValueJS != "" {
				t.Errorf("expected ActualValueJS == '' (empty) for %s, got '%s'", key, c.ActualValueJS)
			}
			if c.MatchExpectation {
				t.Errorf("expected MatchExpectation == false for %s", key)
			}
		}
	}

	// Apply: no drop-ins written
	applyINI := optINI.SetValuesToApply([]string{"kmod:xen guest", "kmod:", "kmod:xen"}).(INISettings)
	if err := applyINI.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if _, err := os.Stat(KmodDropInTransientPath); !os.IsNotExist(err) {
		t.Errorf("expected transient drop-in not to be created for syntax error entries")
	}
	if _, err := os.Stat(KmodDropInPermanentPath); !os.IsNotExist(err) {
		t.Errorf("expected permanent drop-in not to be created for syntax error entries")
	}
}

func TestKmodRevertPermanentRestoresEarlierNote(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		CleanUpParamFile("kmod:cifs")
	}()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	origCmd := system.ModprobeCmd
	defer func() { system.ModprobeCmd = origCmd }()
	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "-c" ]; then
    cat "%s" 2>/dev/null
    cat "%s" 2>/dev/null
    exit 0
fi
exit 0
`, KmodDropInTransientPath, KmodDropInPermanentPath)
	_ = os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	system.ModprobeCmd = fakeScript

	ini1Content := `[version]
ID = test_1
version = 1
date = 18.11.2026
reference = https://me.sap.com/notes/12345
description = Note 1

[kmod]
cifs = blacklist
`
	ini2Content := `[version]
ID = test_2
version = 1
date = 18.11.2026
reference = https://me.sap.com/notes/12345
description = Note 2

[kmod]
cifs = block+permanent
`
	ini1Path := path.Join(tmpDir, "test_1.ini")
	ini2Path := path.Join(tmpDir, "test_2.ini")
	_ = os.WriteFile(ini1Path, []byte(ini1Content), 0644)
	_ = os.WriteFile(ini2Path, []byte(ini2Content), 0644)

	// Step 1: Apply Note 1
	note1 := INISettings{ConfFilePath: ini1Path, ID: "test_1"}
	_, _ = note1.Initialise()
	opt1Note, _ := note1.Initialise()
	opt1Note, _ = opt1Note.Optimise()
	opt1 := opt1Note.(INISettings)
	apply1 := opt1.SetValuesToApply([]string{"kmod:cifs"}).(INISettings)
	if err := apply1.Apply(); err != nil {
		t.Fatalf("Apply note 1 failed: %v", err)
	}
	transLines, _ := system.ReadDropInFile(KmodDropInTransientPath)
	if len(transLines) == 0 || transLines[len(transLines)-1] != "blacklist cifs" {
		t.Fatalf("expected transient drop-in to contain 'blacklist cifs', got %+v", transLines)
	}

	// Step 2: Apply Note 2 (permanent block)
	note2 := INISettings{ConfFilePath: ini2Path, ID: "test_2"}
	_, _ = note2.Initialise()
	opt2Note, _ := note2.Initialise()
	opt2Note, _ = opt2Note.Optimise()
	opt2 := opt2Note.(INISettings)
	apply2 := opt2.SetValuesToApply([]string{"kmod:cifs"}).(INISettings)
	if err := apply2.Apply(); err != nil {
		t.Fatalf("Apply note 2 failed: %v", err)
	}
	permLines, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permLines) == 0 || permLines[len(permLines)-1] != "install cifs /bin/false" {
		t.Fatalf("expected permanent drop-in to contain 'install cifs /bin/false', got %+v", permLines)
	}
	// Note 2 should have removed cifs from transient file
	transLinesAfter2, _ := system.ReadDropInFile(KmodDropInTransientPath)
	for _, l := range transLinesAfter2 {
		if strings.Contains(l, "cifs") {
			t.Fatalf("expected cifs to be removed from transient file after Note 2 apply, got %+v", transLinesAfter2)
		}
	}

	// Step 3: Revert Note 2 -> Note 1's state is restored (LIFO precedence)
	revert2 := apply2.SetValuesToApply([]string{"revert"}).(INISettings)
	if err := revert2.Apply(); err != nil {
		t.Fatalf("Revert note 2 failed: %v", err)
	}
	// Permanent file must NOT contain install cifs /bin/false
	if _, err := os.Stat(KmodDropInPermanentPath); !os.IsNotExist(err) {
		permAfterRevert, _ := system.ReadDropInFile(KmodDropInPermanentPath)
		for _, l := range permAfterRevert {
			if strings.Contains(l, "cifs") {
				t.Errorf("expected permanent drop-in to NOT contain cifs after revert, got %+v", permAfterRevert)
			}
		}
	}
	// Transient file must contain Note 1's restored entry 'blacklist cifs'
	transLinesRestored, _ := system.ReadDropInFile(KmodDropInTransientPath)
	foundRestored := false
	for _, l := range transLinesRestored {
		if l == "blacklist cifs" {
			foundRestored = true
		}
	}
	if !foundRestored {
		t.Errorf("expected transient file to contain restored 'blacklist cifs', got %+v", transLinesRestored)
	}

	// Step 4: Verify Note 1
	init1After, _ := note1.Initialise()
	opt1AfterNote, _ := init1After.Optimise()
	opt1After := opt1AfterNote.(INISettings)
	init1INI := init1After.(INISettings)
	if init1INI.SysctlParams["kmod:cifs"] != "blacklist|unloaded" {
		t.Errorf("expected Note 1 actual state 'blacklist|unloaded', got '%s'", init1INI.SysctlParams["kmod:cifs"])
	}
	allMatch, comp1, _ := CompareNoteFields(init1INI, opt1After, "test_1", "forApply")
	if !allMatch {
		t.Errorf("expected Note 1 to be fully compliant after Note 2 revert, got allMatch = false")
	}
	if !comp1["SysctlParams[kmod:cifs]"].MatchExpectation {
		t.Errorf("expected kmod:cifs MatchExpectation to be true")
	}
}

func TestKmodInformShadowingAndInstallConflict(t *testing.T) {
	origDirs := system.ModprobeDirectories
	origTrans := system.KmodTransientDropInPath
	origPerm := system.KmodPermanentDropInPath
	defer func() {
		system.ModprobeDirectories = origDirs
		system.KmodTransientDropInPath = origTrans
		system.KmodPermanentDropInPath = origPerm
		KmodDropInTransientPath = origTrans
		KmodDropInPermanentPath = origPerm
	}()

	tmpDir := t.TempDir()
	etcDir := path.Join(tmpDir, "etc", "modprobe.d")
	runDir := path.Join(tmpDir, "run", "modprobe.d")
	usrDir := path.Join(tmpDir, "usr", "lib", "modprobe.d")
	_ = os.MkdirAll(etcDir, 0755)
	_ = os.MkdirAll(runDir, 0755)
	_ = os.MkdirAll(usrDir, 0755)

	system.ModprobeDirectories = []string{etcDir, runDir, usrDir}
	system.KmodTransientDropInPath = path.Join(runDir, "80-saptune.conf")
	system.KmodPermanentDropInPath = path.Join(etcDir, "80-saptune-permanent.conf")
	KmodDropInTransientPath = system.KmodTransientDropInPath
	KmodDropInPermanentPath = system.KmodPermanentDropInPath

	// Ensure Saptune's transient drop-in exists with blacklist vfat
	_ = os.WriteFile(KmodDropInTransientPath, []byte("blacklist vfat\n"), 0644)

	// 1. Shadowed by /etc/ -> GetKmodInform contains "masked"
	_ = os.WriteFile(path.Join(etcDir, "80-saptune.conf"), []byte("blacklist vfat\n"), 0644)
	inf := GetKmodInform("testNote", "kmod:vfat", "blacklist", "blacklist|unloaded")
	if !strings.Contains(inf, "masked") {
		t.Errorf("expected GetKmodInform to contain 'masked' when shadowed by /etc/, got '%s'", inf)
	}

	// 2. Shadows lower /usr/lib/ -> GetKmodInform contains "shadowing_warning"
	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := `#!/bin/sh
if [ "$1" = "-c" ]; then
    echo "blacklist vfat"
    exit 0
fi
exit 0
`
	_ = os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	origCmd := system.ModprobeCmd
	defer func() { system.ModprobeCmd = origCmd }()
	system.ModprobeCmd = fakeScript

	_ = os.Remove(path.Join(etcDir, "80-saptune.conf"))
	_ = os.WriteFile(path.Join(usrDir, "80-saptune.conf"), []byte("blacklist vfat\n"), 0644)
	inf2 := GetKmodInform("testNote", "kmod:vfat", "blacklist", "blacklist|unloaded")
	if !strings.Contains(inf2, "shadowing_warning") {
		t.Errorf("expected GetKmodInform to contain 'shadowing_warning' when lower file is shadowed, got '%s'", inf2)
	}
}

func TestKmodBlacklistExternalInstallFootnote18(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	origCmd := system.ModprobeCmd
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		system.ModprobeCmd = origCmd
		ResetKmodDropInFirstWriteFlags()
		CleanUpParamFile("kmod:dummy")
	}()

	ResetKmodDropInFirstWriteFlags()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := `#!/bin/sh
if [ "$1" = "-c" ]; then
    echo "blacklist dummy"
    echo "install dummy /bin/true"
    exit 0
fi
exit 0
`
	_ = os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	system.ModprobeCmd = fakeScript

	// Saptune's permanent drop-in contains blacklist dummy
	_ = os.WriteFile(KmodDropInPermanentPath, []byte("blacklist dummy\n"), 0644)

	inf := GetKmodInform("testNote", "kmod:dummy", "blacklist+permanent", "blacklist+permanent|unloaded")
	if !strings.Contains(inf, "masked") {
		t.Errorf("expected GetKmodInform to contain 'masked', got '%s'", inf)
	}
	if !strings.Contains(inf, "permanent") {
		t.Errorf("expected GetKmodInform to contain 'permanent', got '%s'", inf)
	}
}

func TestKmodMissingDropInHasNoFootnotes(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		ResetKmodDropInFirstWriteFlags()
		CleanUpParamFile("kmod:cifs")
	}()

	ResetKmodDropInFirstWriteFlags()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	// Neither drop-in exists (e.g. write failed or unapplied note)
	inf := GetKmodInform("test_1", "kmod:cifs", "blacklist+permanent", "blacklist+permanent|unloaded")
	if strings.Contains(inf, "masked") {
		t.Errorf("expected no 'masked' flag when drop-in is missing, got '%s'", inf)
	}
	if !strings.Contains(inf, "permanent") {
		t.Errorf("expected 'permanent' flag for permanent entry, got '%s'", inf)
	}
}

func TestKmodApplyRemovesPreExistingDropInFile(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	origCmd := system.ModprobeCmd
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		system.ModprobeCmd = origCmd
		ResetKmodDropInFirstWriteFlags()
		CleanUpParamFile("kmod:ecrypts")
		CleanUpParamFile("kmod:yellowfin")
	}()

	ResetKmodDropInFirstWriteFlags()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "-c" ]; then
    cat "%s" 2>/dev/null
    cat "%s" 2>/dev/null
    exit 0
fi
exit 0
`, KmodDropInTransientPath, KmodDropInPermanentPath)
	_ = os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	system.ModprobeCmd = fakeScript

	// Pre-create permanent drop-in with foreign line before apply
	preExistingContent := `# This file was created by saptune as permanent drop-in. Do not touch!

install ecryptfs /bin/2
`
	_ = os.WriteFile(KmodDropInPermanentPath, []byte(preExistingContent), 0644)

	iniContent := `[version]
ID = test_pre_existing
version = 1
date = 20.11.2026
reference = https://me.sap.com/notes/12345
description = Test pre-existing overwrite

[kmod]
ecrypts = block+permanent
yellowfin = block+permanent
`
	iniPath := path.Join(tmpDir, "test_pre.ini")
	_ = os.WriteFile(iniPath, []byte(iniContent), 0644)

	noteObj := INISettings{ConfFilePath: iniPath, ID: "test_pre_existing"}
	optNote, _ := noteObj.Initialise()
	optNote, _ = optNote.Optimise()
	optINI := optNote.(INISettings)
	applyINI := optINI.SetValuesToApply([]string{"kmod:ecrypts", "kmod:yellowfin"}).(INISettings)
	if err := applyINI.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	permLines, err := system.ReadDropInFile(KmodDropInPermanentPath)
	if err != nil {
		t.Fatalf("failed to read permanent drop-in: %v", err)
	}

	for _, line := range permLines {
		if strings.Contains(line, "ecryptfs") {
			t.Errorf("pre-existing foreign line 'ecryptfs' must NOT be merged into drop-in, found: %s", line)
		}
	}

	foundEcrypts := false
	foundYellowfin := false
	for _, line := range permLines {
		if line == "install ecrypts /bin/false" {
			foundEcrypts = true
		}
		if line == "install yellowfin /bin/false" {
			foundYellowfin = true
		}
	}
	if !foundEcrypts {
		t.Errorf("expected 'install ecrypts /bin/false' in permanent drop-in, got %+v", permLines)
	}
	if !foundYellowfin {
		t.Errorf("expected 'install yellowfin /bin/false' in permanent drop-in, got %+v", permLines)
	}
}

func TestCleanStaleKmodDropIns(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	origCmd := system.ModprobeCmd
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		system.ModprobeCmd = origCmd
		ResetKmodDropInFirstWriteFlags()
	}()

	ResetKmodDropInFirstWriteFlags()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "-c" ]; then
    cat "%s" 2>/dev/null
    cat "%s" 2>/dev/null
    exit 0
fi
exit 0
`, KmodDropInTransientPath, KmodDropInPermanentPath)
	_ = os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	system.ModprobeCmd = fakeScript

	// Pre-create both drop-ins manually
	_ = os.WriteFile(KmodDropInTransientPath, []byte("install stray1 /bin/1\n"), 0644)
	_ = os.WriteFile(KmodDropInPermanentPath, []byte("install stray2 /bin/2\n"), 0644)

	// Apply a note with NO [kmod] section
	iniContent := `[version]
ID = test_no_kmod
version = 1
date = 20.11.2026
reference = https://me.sap.com/notes/12345
description = Note without kmod

[sysctl]
kernel.pid_max = 32768
`
	iniPath := path.Join(tmpDir, "test_no_kmod.ini")
	_ = os.WriteFile(iniPath, []byte(iniContent), 0644)

	noteObj := INISettings{ConfFilePath: iniPath, ID: "test_no_kmod"}
	optNote, _ := noteObj.Initialise()
	optNote, _ = optNote.Optimise()
	optINI := optNote.(INISettings)
	applyINI := optINI.SetValuesToApply([]string{"kernel.pid_max"}).(INISettings)
	_ = applyINI.Apply()

	// Both drop-ins must be removed because no kmod section is in the active configuration
	if _, err := os.Stat(KmodDropInTransientPath); !os.IsNotExist(err) {
		t.Errorf("expected transient drop-in to be removed when no kmod section is applied")
	}
	if _, err := os.Stat(KmodDropInPermanentPath); !os.IsNotExist(err) {
		t.Errorf("expected permanent drop-in to be removed when no kmod section is applied")
	}
}

func TestKmodSupersededByLaterNoteFootnote17(t *testing.T) {
	origTransient := KmodDropInTransientPath
	origPermanent := KmodDropInPermanentPath
	origSysfs := system.SysfsModuleDir
	origCmd := system.ModprobeCmd
	defer func() {
		KmodDropInTransientPath = origTransient
		KmodDropInPermanentPath = origPermanent
		system.SysfsModuleDir = origSysfs
		system.ModprobeCmd = origCmd
		ResetKmodDropInFirstWriteFlags()
		CleanUpParamFile("kmod:cifs")
	}()

	ResetKmodDropInFirstWriteFlags()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "-c" ]; then
    cat "%s" 2>/dev/null
    cat "%s" 2>/dev/null
    exit 0
fi
exit 0
`, KmodDropInTransientPath, KmodDropInPermanentPath)
	_ = os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	system.ModprobeCmd = fakeScript

	ini1Content := `[version]
ID = test_1
version = 1
date = 18.11.2026
reference = https://me.sap.com/notes/12345
description = Note 1

[kmod]
cifs = blacklist
`
	ini2Content := `[version]
ID = test_2
version = 1
date = 18.11.2026
reference = https://me.sap.com/notes/12345
description = Note 2

[kmod]
cifs = block
`
	ini1Path := path.Join(tmpDir, "test_1.ini")
	ini2Path := path.Join(tmpDir, "test_2.ini")
	_ = os.WriteFile(ini1Path, []byte(ini1Content), 0644)
	_ = os.WriteFile(ini2Path, []byte(ini2Content), 0644)

	// Step 1: Apply Note 1
	note1 := INISettings{ConfFilePath: ini1Path, ID: "test_1"}
	_, _ = note1.Initialise()
	opt1Note, _ := note1.Initialise()
	opt1Note, _ = opt1Note.Optimise()
	opt1 := opt1Note.(INISettings)
	apply1 := opt1.SetValuesToApply([]string{"kmod:cifs"}).(INISettings)
	if err := apply1.Apply(); err != nil {
		t.Fatalf("Apply note 1 failed: %v", err)
	}

	// Step 2: Apply Note 2
	note2 := INISettings{ConfFilePath: ini2Path, ID: "test_2"}
	_, _ = note2.Initialise()
	opt2Note, _ := note2.Initialise()
	opt2Note, _ = opt2Note.Optimise()
	opt2 := opt2Note.(INISettings)
	apply2 := opt2.SetValuesToApply([]string{"kmod:cifs"}).(INISettings)
	if err := apply2.Apply(); err != nil {
		t.Fatalf("Apply note 2 failed: %v", err)
	}

	// Note 2 is now the ruling note for kmod:cifs
	if last := ParameterLastNote("kmod:cifs"); last != "test_2" {
		t.Fatalf("expected last note for kmod:cifs to be 'test_2', got '%s'", last)
	}

	// Step 3: Verify Note 1 (superseded by Note 2)
	init1After, _ := note1.Initialise()
	init1INI := init1After.(INISettings)
	opt1AfterNote, _ := init1After.Optimise()
	opt1After := opt1AfterNote.(INISettings)

	// GetKmodInform for Note 1 should NOT contain "masked"
	inf1 := opt1After.Inform["kmod:cifs"]
	if strings.Contains(inf1, "masked") {
		t.Errorf("expected Inform for superseded Note 1 to NOT contain 'masked', got '%s'", inf1)
	}

	// CompareNoteFields for Note 1 should be compliant (allMatch = true) under forVerify
	allMatch1, comp1, _ := CompareNoteFields(init1INI, opt1After, "test_1", "forVerify")
	if !allMatch1 {
		t.Errorf("expected Note 1 to be compliant (allMatch = true) when superseded, got allMatch = false")
	}
	if !comp1["SysctlParams[kmod:cifs]"].MatchExpectation {
		t.Errorf("expected MatchExpectation = true for superseded kmod:cifs in Note 1")
	}

	// Step 4: Verify Note 2 (active ruling note)
	init2After, _ := note2.Initialise()
	init2INI := init2After.(INISettings)
	opt2AfterNote, _ := init2After.Optimise()
	opt2After := opt2AfterNote.(INISettings)

	allMatch2, comp2, _ := CompareNoteFields(init2INI, opt2After, "test_2", "forVerify")
	if !allMatch2 {
		t.Errorf("expected ruling Note 2 to be compliant (allMatch = true), got allMatch = false")
	}
	if !comp2["SysctlParams[kmod:cifs]"].MatchExpectation {
		t.Errorf("expected MatchExpectation = true for ruling kmod:cifs in Note 2")
	}
}
