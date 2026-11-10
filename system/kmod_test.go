package system

import (
	"os"
	"path"
	"strings"
	"testing"
)

func TestNormalizeModuleName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"fuse", "fuse"},
		{"tcp-cubic", "tcp_cubic"},
		{"my-test-module", "my_test_module"},
		{"already_normalized", "already_normalized"},
	}

	for _, c := range cases {
		got := NormalizeModuleName(c.input)
		if got != c.expected {
			t.Errorf("NormalizeModuleName(%s): expected %s, got %s", c.input, c.expected, got)
		}
	}
}

func TestSysfsModuleState(t *testing.T) {
	origSysfs := SysfsModuleDir
	defer func() { SysfsModuleDir = origSysfs }()

	tmpDir := t.TempDir()
	SysfsModuleDir = tmpDir

	// Setup fake modules:
	// 1. "loaded_mod" -> initstate = "live\n"
	loadedDir := path.Join(tmpDir, "loaded_mod")
	if err := os.MkdirAll(loadedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(loadedDir, "initstate"), []byte("live\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. "builtin_mod" -> directory exists, no initstate
	builtinDir := path.Join(tmpDir, "builtin_mod")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test loaded module
	if !IsModuleLoaded("loaded_mod") {
		t.Errorf("expected loaded_mod to be loaded")
	}
	if IsModuleBuiltin("loaded_mod") {
		t.Errorf("expected loaded_mod not to be builtin")
	}

	// Test with dash in module name
	if !IsModuleLoaded("loaded-mod") {
		t.Errorf("expected loaded-mod to normalize and report loaded")
	}

	// Test builtin module
	if IsModuleLoaded("builtin_mod") {
		t.Errorf("expected builtin_mod not to be reported as loadable loaded")
	}
	if !IsModuleBuiltin("builtin_mod") {
		t.Errorf("expected builtin_mod to be builtin")
	}

	// Test unknown / unloaded module
	if IsModuleLoaded("unknown_mod") {
		t.Errorf("expected unknown_mod not to be loaded")
	}
	if IsModuleBuiltin("unknown_mod") {
		t.Errorf("expected unknown_mod not to be builtin")
	}
}

func TestSysfsModuleParams(t *testing.T) {
	origSysfs := SysfsModuleDir
	defer func() { SysfsModuleDir = origSysfs }()

	tmpDir := t.TempDir()
	SysfsModuleDir = tmpDir

	paramsDir := path.Join(tmpDir, "test_mod", "parameters")
	if err := os.MkdirAll(paramsDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path.Join(paramsDir, "debug"), []byte("Y\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(paramsDir, "timeout"), []byte("30\n"), 0644); err != nil {
		t.Fatal(err)
	}

	params, err := GetModuleParams("test-mod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["debug"] != "Y" || params["timeout"] != "30" {
		t.Errorf("unexpected params: %+v", params)
	}

	// Test SetModuleParam
	if err := SetModuleParam("test-mod", "timeout", "60"); err != nil {
		t.Fatalf("SetModuleParam error: %v", err)
	}
	updated, _ := GetModuleParams("test_mod")
	if updated["timeout"] != "60" {
		t.Errorf("expected timeout=60, got %s", updated["timeout"])
	}
}

func TestModprobeCommandsWithFake(t *testing.T) {
	origCmd := modprobeCmd
	defer func() { modprobeCmd = origCmd }()

	tmpDir := t.TempDir()
	fakeScript := path.Join(tmpDir, "fake_modprobe.sh")
	scriptContent := `#!/bin/sh
if [ "$1" = "-c" ]; then
    echo "blacklist xen"
    echo "install vfat /bin/false"
    echo "options dummy numdummies=2"
    exit 0
fi
if [ "$1" = "-r" ]; then
    if [ "$2" = "fail_mod" ]; then
        echo "FATAL: Module fail_mod is in use." >&2
        exit 1
    fi
    exit 0
fi
if [ "$1" = "fail_load" ]; then
    echo "FATAL: Module fail_load not found." >&2
    exit 1
fi
exit 0
`
	if err := os.WriteFile(fakeScript, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}
	modprobeCmd = fakeScript

	// Test ModprobeConfig
	cfg, err := ModprobeConfig()
	if err != nil {
		t.Fatalf("ModprobeConfig error: %v", err)
	}
	if !IsEffectiveConfigEntry(cfg, "blacklist xen") {
		t.Errorf("expected 'blacklist xen' to be effective")
	}
	if !IsEffectiveConfigEntry(cfg, "install vfat /bin/false") {
		t.Errorf("expected 'install vfat /bin/false' to be effective")
	}
	if IsEffectiveConfigEntry(cfg, "blacklist other") {
		t.Errorf("expected 'blacklist other' not to be effective")
	}

	// Test ModprobeLoad
	pMap := map[string]string{"numdummies": "2"}
	if err := ModprobeLoad("dummy", pMap); err != nil {
		t.Errorf("ModprobeLoad dummy failed: %v", err)
	}
	if err := ModprobeLoad("fail_load", nil); err == nil {
		t.Errorf("expected ModprobeLoad fail_load to return error")
	}

	// Test ModprobeRemove
	if err := ModprobeRemove("dummy"); err != nil {
		t.Errorf("ModprobeRemove dummy failed: %v", err)
	}
	if err := ModprobeRemove("fail_mod"); err == nil {
		t.Errorf("expected ModprobeRemove fail_mod to return error")
	}
}

func TestDropInFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dropInPath := path.Join(tmpDir, "run", "modprobe.d", "80-saptune.conf")

	// Write drop-in
	lines := []string{"blacklist xen", "install vfat /bin/false"}
	if err := WriteDropInFileAtomic(dropInPath, KmodTransientHeader, lines); err != nil {
		t.Fatalf("WriteDropInFileAtomic error: %v", err)
	}

	// Read back
	readLines, err := ReadDropInFile(dropInPath)
	if err != nil {
		t.Fatalf("ReadDropInFile error: %v", err)
	}
	foundXen := false
	foundVfat := false
	for _, l := range readLines {
		if l == "blacklist xen" {
			foundXen = true
		}
		if l == "install vfat /bin/false" {
			foundVfat = true
		}
	}
	if !foundXen || !foundVfat {
		t.Errorf("expected drop-in to contain xen and vfat, got %+v", readLines)
	}

	// Test RemoveDropInModuleLines
	kept, removed := RemoveDropInModuleLines(readLines, "xen", "")
	if !removed {
		t.Errorf("expected xen to be removed")
	}
	if err := WriteDropInFileAtomic(dropInPath, KmodTransientHeader, kept); err != nil {
		t.Fatalf("WriteDropInFileAtomic error: %v", err)
	}

	// Verify only vfat remains
	updatedLines, _ := ReadDropInFile(dropInPath)
	if strings.Contains(strings.Join(updatedLines, "\n"), "blacklist xen") {
		t.Errorf("blacklist xen still present after removal")
	}

	// Remove remaining line and verify file is deleted
	kept2, _ := RemoveDropInModuleLines(updatedLines, "vfat", "")
	if err := WriteDropInFileAtomic(dropInPath, KmodTransientHeader, kept2); err != nil {
		t.Fatalf("WriteDropInFileAtomic error: %v", err)
	}
	if _, err := os.Stat(dropInPath); !os.IsNotExist(err) {
		t.Errorf("expected drop-in file to be deleted when empty")
	}
}

func TestDropInModuleStateAndPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	transPath := path.Join(tmpDir, "run_dropin.conf")
	permPath := path.Join(tmpDir, "etc_dropin.conf")

	// Neither file exists
	if st := GetDropInModuleState(transPath, permPath, "xen"); st != "none" {
		t.Errorf("expected 'none', got %s", st)
	}

	// Write transient blacklist
	_ = WriteDropInFileAtomic(transPath, KmodTransientHeader, []string{"blacklist xen"})
	if st := GetDropInModuleState(transPath, permPath, "xen"); st != "blacklist" {
		t.Errorf("expected 'blacklist', got %s", st)
	}

	// Permanent file has block for xen (permanent takes precedence)
	_ = WriteDropInFileAtomic(permPath, KmodPermanentHeader, []string{"install xen /bin/false"})
	if st := GetDropInModuleState(transPath, permPath, "xen"); st != "block" {
		t.Errorf("expected 'block', got %s", st)
	}
}

func TestForeignLinesUntouched(t *testing.T) {
	input := []string{
		"# custom header",
		"options mymod foo=bar",
		"blacklist xen",
		"install vfat /bin/false",
		"# end comment",
	}

	// Remove xen
	out, removed := RemoveDropInModuleLines(input, "xen", "")
	if !removed {
		t.Errorf("expected removal of xen")
	}

	expectedCount := 4
	if len(out) != expectedCount {
		t.Errorf("expected %d lines kept, got %d: %+v", expectedCount, len(out), out)
	}

	// Check foreign line is preserved
	hasCustom := false
	for _, l := range out {
		if l == "options mymod foo=bar" {
			hasCustom = true
		}
	}
	if !hasCustom {
		t.Errorf("expected foreign line 'options mymod foo=bar' to be preserved")
	}
}
