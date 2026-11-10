package note

import (
	"github.com/SUSE/saptune/system"
	"os"
	"path"
	"testing"
)

func TestParseKmodEntryValid(t *testing.T) {
	cases := []struct {
		key       string
		val       string
		expected  KmodEntry
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
		entry, err := ParseKmodEntry(c.key, c.val)
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
		entry, err := ParseKmodEntry(c.key, c.val)
		if err == nil {
			t.Errorf("case '%s': ParseKmodEntry(%s, %s) expected error, got %+v", c.name, c.key, c.val, entry)
		}
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
	if val := OptKmodVal("test_mod", "blacklist", "PNA"); val != "PNA" {
		t.Errorf("expected 'PNA', got '%s'", val)
	}

	// 2. Blacklist replaces file part, keeps unloaded
	if val := OptKmodVal("test_mod", "blacklist", "none|unloaded"); val != "blacklist|unloaded" {
		t.Errorf("expected 'blacklist|unloaded', got '%s'", val)
	}

	// 3. Block replaces file part, keeps loaded params
	actual := "none|loaded:{\"debug\":\"Y\",\"timeout\":\"30\"}"
	expected := "block|loaded:{\"debug\":\"Y\",\"timeout\":\"30\"}"
	if val := OptKmodVal("test_mod", "block", actual); val != expected {
		t.Errorf("expected '%s', got '%s'", expected, val)
	}

	// 4. Invalid config returns NA
	if val := OptKmodVal("test_mod", "invalid_action", "none|unloaded"); val != "NA" {
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
	if err := SetKmodVal("xen", "blacklist|unloaded", "blacklist", false); err != nil {
		t.Fatalf("SetKmodVal apply failed: %v", err)
	}
	lines, _ := system.ReadDropInFile(KmodDropInTransientPath)
	if len(lines) == 0 || lines[len(lines)-1] != "blacklist xen" {
		t.Errorf("expected transient drop-in to contain 'blacklist xen', got %+v", lines)
	}

	// 2. Revert transient blacklist to none|unloaded -> file deleted
	if err := SetKmodVal("xen", "none|unloaded", "blacklist", true); err != nil {
		t.Fatalf("SetKmodVal revert failed: %v", err)
	}
	if _, err := os.Stat(KmodDropInTransientPath); !os.IsNotExist(err) {
		t.Errorf("expected transient file to be deleted on revert to none")
	}

	// 3. Apply permanent block
	if err := SetKmodVal("vfat", "block|unloaded", "block+permanent", false); err != nil {
		t.Fatalf("SetKmodVal permanent apply failed: %v", err)
	}
	permLines, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permLines) == 0 || permLines[len(permLines)-1] != "install vfat /bin/false" {
		t.Errorf("expected permanent drop-in to contain 'install vfat /bin/false', got %+v", permLines)
	}

	// 4. Revert of permanent is skipped
	if err := SetKmodVal("vfat", "none|unloaded", "block+permanent", true); err != nil {
		t.Fatalf("SetKmodVal permanent revert failed: %v", err)
	}
	permLinesAfter, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permLinesAfter) == 0 || permLinesAfter[len(permLinesAfter)-1] != "install vfat /bin/false" {
		t.Errorf("expected permanent drop-in to retain 'install vfat /bin/false', got %+v", permLinesAfter)
	}

	// 5. Stale line cleanup: module changed from block+permanent to blacklist transient
	if err := SetKmodVal("vfat", "blacklist|unloaded", "blacklist", false); err != nil {
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
	if err := SetKmodVal("vfat", "block|unloaded", "blacklist", true); err != nil {
		t.Fatalf("SetKmodVal restore earlier note failed: %v", err)
	}
	transLinesRestored, _ := system.ReadDropInFile(KmodDropInTransientPath)
	if len(transLinesRestored) == 0 || transLinesRestored[len(transLinesRestored)-1] != "install vfat /bin/false" {
		t.Errorf("expected restored earlier note line in transient file, got %+v", transLinesRestored)
	}

	// 7. Compiled-in module is skipped
	builtinDir := path.Join(system.SysfsModuleDir, "mybuiltin")
	_ = os.MkdirAll(builtinDir, 0755)
	if err := SetKmodVal("mybuiltin", "PNA", "blacklist", false); err != nil {
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
	}()

	tmpDir := t.TempDir()
	KmodDropInTransientPath = path.Join(tmpDir, "80-saptune.conf")
	KmodDropInPermanentPath = path.Join(tmpDir, "80-saptune-permanent.conf")
	system.SysfsModuleDir = path.Join(tmpDir, "sys_module")

	iniContent := `[version]
ID = test_kmod_note
version = 1
date = 10.11.2026
reference = https://me.sap.com/notes/12345
description = Test kmod note

[kmod]
xen = blacklist
vfat = block+permanent
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

	optNote, err := initINI.Optimise()
	if err != nil {
		t.Fatalf("Optimise failed: %v", err)
	}
	optINI := optNote.(INISettings)
	if optINI.SysctlParams["kmod:xen"] != "blacklist|unloaded" {
		t.Errorf("expected 'blacklist|unloaded', got '%s'", optINI.SysctlParams["kmod:xen"])
	}
	if optINI.SysctlParams["kmod:vfat"] != "block|unloaded" {
		t.Errorf("expected 'block|unloaded', got '%s'", optINI.SysctlParams["kmod:vfat"])
	}

	// Apply
	applyINI := optINI.SetValuesToApply([]string{"kmod:xen", "kmod:vfat"}).(INISettings)
	if err := applyINI.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
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
		t.Errorf("expected transient file to be deleted on revert")
	}
	// Permanent file should keep vfat because permanent
	permAfter, _ := system.ReadDropInFile(KmodDropInPermanentPath)
	if len(permAfter) == 0 || permAfter[len(permAfter)-1] != "install vfat /bin/false" {
		t.Errorf("expected permanent drop-in to retain 'install vfat /bin/false', got %+v", permAfter)
	}
}
