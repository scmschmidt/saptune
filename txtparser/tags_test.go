package txtparser

import (
	"github.com/SUSE/saptune/system"
	"os"
	"path"
	"testing"
)

func TestChkOsTags(t *testing.T) {
	tag := "15-*"
	secFields := []string{"rpm", "os=15-*", "arch=amd64"}

	ret := chkOsTags(tag, secFields)
	if !ret {
		t.Error("not matching os version")
	}
	_ = system.CopyFile(path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata/osr12"), "/etc/os-release")
	ret = chkOsTags(tag, secFields)
	if ret {
		t.Error("matching os version, but shouldn't")
	}
	_ = system.CopyFile(path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata/osr16.2"), "/etc/os-release")
	tag = "16.2"
	secFields = []string{"rpm", "os=16.2", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if !ret {
		t.Error("not matching os version")
	}
	tag = "19.0"
	secFields = []string{"rpm", "os=19.0", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if ret {
		t.Error("matching os version, but shouldn't")
	}
	tag = "16.[0,2,4-5,7-]"
	secFields = []string{"rpm", "os=16.[0,2,4-5,7-]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if !ret {
		t.Error("not matching os version")
	}
	tag = "16.[-2,4-5,7-]"
	secFields = []string{"rpm", "os=16.[-2,4-5,7-]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if !ret {
		t.Error("not matching os version")
	}
	tag = "16.[0-1,2-]"
	secFields = []string{"rpm", "os=16.[0-1,2-]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if !ret {
		t.Error("not matching os version")
	}
	tag = "16.[0-2,4-5,7-]"
	secFields = []string{"rpm", "os=16.[0-2,4-5,7-]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if !ret {
		t.Error("not matching os version")
	}
	tag = "16.[-6]"
	secFields = []string{"rpm", "os=16.[-6]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if !ret {
		t.Error("not matching os version")
	}
	tag = "16.[02]"
	secFields = []string{"rpm", "os=16.[02]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if ret {
		t.Error("matching os version, but shouldn't")
	}
	tag = "19.[2]"
	secFields = []string{"rpm", "os=19.[2]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if ret {
		t.Error("matching os version, but shouldn't")
	}
	tag = "16.[-]"
	secFields = []string{"rpm", "os=16.[-]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if ret {
		t.Error("matching os version, but shouldn't")
	}
	tag = "16.[4-]"
	secFields = []string{"rpm", "os=16.[4-]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if ret {
		t.Error("matching os version, but shouldn't")
	}
	tag = "16.[0+2]"
	secFields = []string{"rpm", "os=16.[0+2]", "arch=amd64"}
	ret = chkOsTags(tag, secFields)
	if ret {
		t.Error("matching os version, but shouldn't")
	}
	_ = system.CopyFile("/etc/os-release_OrG", "/etc/os-release")
}

func TestChkHWTags(t *testing.T) {
	system.DmiID = path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata")
	secFields := []string{"sysctl", "vendor=SUSE", "arch=amd64"}
	info := "vendor"
	tag := "SUSE"
	ret := chkHWTags(info, tag, secFields)
	if !ret {
		t.Errorf("tag '%s' does not match content of %s file", tag, info)
	}

	secFields = []string{"sysctl", "model=SUSE saptune", "arch=amd64"}
	info = "model"
	tag = "SUSE saptune"
	ret = chkHWTags(info, tag, secFields)
	if !ret {
		t.Errorf("tag '%s' does not match content of %s file", tag, info)
	}

	tag = "SE sap"
	ret = chkHWTags(info, tag, secFields)
	if !ret {
		t.Errorf("tag '%s' does not match content of %s file", tag, info)
	}

	tag = "hugo"
	ret = chkHWTags(info, tag, secFields)
	if ret {
		t.Errorf("tag '%s' matches content of %s file, but shouldn't", tag, info)
	}

	os.Rename(path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata/product_name"), path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata/product_name_OrG"))
	tag = "SUSE saptune"
	ret = chkHWTags(info, tag, secFields)
	if ret {
		t.Errorf("tag '%s' matches content of %s file, but shouldn't", tag, info)
	}
	os.Rename(path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata/product_name_OrG"), path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata/product_name"))

	system.DmiID = "/sys/class/dmi/id"
}

func TestChkOtherTags(t *testing.T) {
	system.DmiID = path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata")
	secFields := []string{"sysctl", "product_name=SUSE", "arch=amd64"}
	file := "product_name"
	tag := "SUSE"

	ret := chkOtherTags(file, tag, secFields)
	if !ret {
		t.Errorf("tag '%s' does not match content in '%+s'", tag, file)
	}

	tag = "saptune"
	ret = chkOtherTags(file, tag, secFields)
	if !ret {
		t.Errorf("tag '%s' does not match content in '%+s'", tag, file)
	}

	tag = "SE sap"
	ret = chkOtherTags(file, tag, secFields)
	if !ret {
		t.Errorf("tag '%s' does not match content in '%+s'", tag, file)
	}

	tag = "hugo"
	ret = chkOtherTags(file, tag, secFields)
	if ret {
		t.Errorf("tag '%s' does match content in '%+s', but shouldn't", tag, file)
	}
	system.DmiID = "/sys/class/dmi/id"
}

func TestIsTagAvail(t *testing.T) {
	secFields := []string{"block", "blkvendor=HUGO", "blkmodel=EGON"}
	tag := "blkvendor"
	if !isTagAvail(tag, secFields) {
		t.Errorf("tag '%s' is expected to be available, but is repoted as not available\n", tag)
	}
	tag = "blkmodel"
	if !isTagAvail(tag, secFields) {
		t.Errorf("tag '%s' is expected to be available, but is repoted as not available\n", tag)
	}
	tag = "blkpat"
	if isTagAvail(tag, secFields) {
		t.Errorf("tag '%s' is expected to be NOT available, but is repoted as available\n", tag)
	}
	tag = "blkvendor"
	secFields = []string{"block", "blkvendor="}
	if !isTagAvail(tag, secFields) {
		t.Error("expected 'true', because of correct syntax, but got 'false'")
	}
	secFields = []string{"block", "blkvendor"}
	if isTagAvail(tag, secFields) {
		t.Error("expected 'false', because of wrong syntax, but got 'true'")
	}
	secFields = []string{"block", "=EGON"}
	if isTagAvail(tag, secFields) {
		t.Error("expected 'false', because of wrong syntax, but got 'true'")
	}
	secFields = []string{"block", "="}
	if isTagAvail(tag, secFields) {
		t.Error("expected 'false', because of wrong syntax, but got 'true'")
	}
	secFields = []string{"block", ""}
	if isTagAvail(tag, secFields) {
		t.Error("expected 'false', because of wrong syntax, but got 'true'")
	}
}

func TestChkHostnameTags(t *testing.T) {
	origCmd := system.HostnameCmd
	defer func() { system.HostnameCmd = origCmd }()

	tmpDir := t.TempDir()
	mockScript := path.Join(tmpDir, "mock_hostname")
	scriptContent := "#!/bin/sh\necho 'sap-hana-db01'\nexit 0\n"
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}
	system.HostnameCmd = mockScript

	secFields := []string{"sysctl", "hostname=sap-.*"}

	// Matching regex
	if !chkHostnameTags("sap-.*", secFields) {
		t.Errorf("expected 'sap-.*' to match 'sap-hana-db01'")
	}
	if !chkHostnameTags("^sap-hana-db01$", secFields) {
		t.Errorf("expected '^sap-hana-db01$' to match 'sap-hana-db01'")
	}
	if !chkHostnameTags("db01", secFields) {
		t.Errorf("expected 'db01' to match 'sap-hana-db01'")
	}

	// Non-matching regex
	if chkHostnameTags("^db02$", secFields) {
		t.Errorf("expected '^db02$' not to match 'sap-hana-db01'")
	}

	// Invalid regex
	if chkHostnameTags("[a-z", secFields) {
		t.Errorf("expected invalid regex to return false")
	}

	// Non-zero exit code
	system.HostnameCmd = "/usr/bin/false"
	if chkHostnameTags(".*", secFields) {
		t.Errorf("expected non-zero exit code to return false")
	}
}

func TestChkDNSDomainnameTags(t *testing.T) {
	origCmd := system.DnsdomainnameCmd
	defer func() { system.DnsdomainnameCmd = origCmd }()

	tmpDir := t.TempDir()
	mockScript := path.Join(tmpDir, "mock_dnsdomainname")
	scriptContent := "#!/bin/sh\necho 'corp.sap.internal'\nexit 0\n"
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}
	system.DnsdomainnameCmd = mockScript

	secFields := []string{"sysctl", "dnsdomainname=.*sap\\.internal"}

	// Matching regex
	if !chkDNSDomainnameTags(".*\\.sap\\.internal$", secFields) {
		t.Errorf("expected '.*\\.sap\\.internal$' to match 'corp.sap.internal'")
	}
	if !chkDNSDomainnameTags("^corp\\.", secFields) {
		t.Errorf("expected '^corp\\.' to match 'corp.sap.internal'")
	}

	// Non-matching regex
	if chkDNSDomainnameTags("example\\.com", secFields) {
		t.Errorf("expected 'example\\.com' not to match 'corp.sap.internal'")
	}

	// Invalid regex
	if chkDNSDomainnameTags("(", secFields) {
		t.Errorf("expected invalid regex to return false")
	}

	// Non-zero exit code
	system.DnsdomainnameCmd = "/usr/bin/false"
	if chkDNSDomainnameTags(".*", secFields) {
		t.Errorf("expected non-zero exit code to return false")
	}
}

func TestChkFqdnTags(t *testing.T) {
	origCmd := system.HostnameCmd
	defer func() { system.HostnameCmd = origCmd }()

	tmpDir := t.TempDir()
	mockScript := path.Join(tmpDir, "mock_fqdn")
	scriptContent := "#!/bin/sh\nif [ \"$1\" = \"-f\" ]; then\n  echo 'sap-app01.prod.acme.com'\n  exit 0\nelse\n  exit 1\nfi\n"
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}
	system.HostnameCmd = mockScript

	secFields := []string{"sysctl", "fqdn=.*\\.prod\\.acme\\.com"}

	// Matching regex
	if !chkFqdnTags("sap-app\\d+\\.prod\\.acme\\.com", secFields) {
		t.Errorf("expected 'sap-app\\d+\\.prod\\.acme\\.com' to match 'sap-app01.prod.acme.com'")
	}
	if !chkFqdnTags(".*\\.acme\\.com$", secFields) {
		t.Errorf("expected '.*\\.acme\\.com$' to match 'sap-app01.prod.acme.com'")
	}

	// Non-matching regex
	if chkFqdnTags("dev\\.acme\\.com", secFields) {
		t.Errorf("expected 'dev\\.acme\\.com' not to match 'sap-app01.prod.acme.com'")
	}

	// Invalid regex
	if chkFqdnTags("[", secFields) {
		t.Errorf("expected invalid regex to return false")
	}

	// Non-zero exit code
	system.HostnameCmd = "/usr/bin/false"
	if chkFqdnTags(".*", secFields) {
		t.Errorf("expected non-zero exit code to return false")
	}
}

func TestChkSecTagsHost(t *testing.T) {
	origHostnameCmd := system.HostnameCmd
	origDnsdomainnameCmd := system.DnsdomainnameCmd
	defer func() {
		system.HostnameCmd = origHostnameCmd
		system.DnsdomainnameCmd = origDnsdomainnameCmd
	}()

	tmpDir := t.TempDir()
	mockHostname := path.Join(tmpDir, "mock_hostname")
	scriptHostname := "#!/bin/sh\nif [ \"$1\" = \"-f\" ]; then\n  echo 'hana01.mycorp.internal'\n  exit 0\nfi\necho 'hana01'\nexit 0\n"
	if err := os.WriteFile(mockHostname, []byte(scriptHostname), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}
	mockDomain := path.Join(tmpDir, "mock_domain")
	scriptDomain := "#!/bin/sh\necho 'mycorp.internal'\nexit 0\n"
	if err := os.WriteFile(mockDomain, []byte(scriptDomain), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	system.HostnameCmd = mockHostname
	system.DnsdomainnameCmd = mockDomain

	// Valid hostname tag
	ok, _ := chkSecTags([]string{"sysctl", "hostname=^hana\\d+$"}, []string{})
	if !ok {
		t.Errorf("expected chkSecTags with hostname tag to be true")
	}

	// Valid dnsdomainname tag
	ok, _ = chkSecTags([]string{"sysctl", "dnsdomainname=^mycorp\\.internal$"}, []string{})
	if !ok {
		t.Errorf("expected chkSecTags with dnsdomainname tag to be true")
	}

	// Valid fqdn tag
	ok, _ = chkSecTags([]string{"sysctl", "fqdn=^hana01\\.mycorp\\.internal$"}, []string{})
	if !ok {
		t.Errorf("expected chkSecTags with fqdn tag to be true")
	}

	// Combined all three tags
	ok, _ = chkSecTags([]string{"sysctl", "hostname=hana.*", "dnsdomainname=mycorp.*", "fqdn=.*\\.internal"}, []string{})
	if !ok {
		t.Errorf("expected chkSecTags with all three host tags to be true")
	}

	// Combined with one non-matching tag
	ok, _ = chkSecTags([]string{"sysctl", "hostname=hana.*", "dnsdomainname=wrongdomain"}, []string{})
	if ok {
		t.Errorf("expected chkSecTags to fail when one tag does not match")
	}
}

func TestParseINIHostTags(t *testing.T) {
	origHostnameCmd := system.HostnameCmd
	origDnsdomainnameCmd := system.DnsdomainnameCmd
	defer func() {
		system.HostnameCmd = origHostnameCmd
		system.DnsdomainnameCmd = origDnsdomainnameCmd
	}()

	tmpDir := t.TempDir()
	mockHostname := path.Join(tmpDir, "mock_hostname")
	scriptHostname := "#!/bin/sh\nif [ \"$1\" = \"-f\" ]; then\n  echo 'hana01.mycorp.internal'\n  exit 0\nfi\necho 'hana01'\nexit 0\n"
	if err := os.WriteFile(mockHostname, []byte(scriptHostname), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}
	mockDomain := path.Join(tmpDir, "mock_domain")
	scriptDomain := "#!/bin/sh\necho 'mycorp.internal'\nexit 0\n"
	if err := os.WriteFile(mockDomain, []byte(scriptDomain), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	system.HostnameCmd = mockHostname
	system.DnsdomainnameCmd = mockDomain

	iniContent := `
[sysctl:hostname=^hana\d+$]
kernel.sem = 1250 256000 100 8192

[sysctl:hostname=^wronghost$]
fs.file-max = 6815744

[sys:dnsdomainname=^mycorp\.internal$]
net.ipv4.tcp_slow_start_after_idle = 1

[sys:dnsdomainname=^othercorp\.internal$]
net.ipv4.tcp_slow_start_after_idle = 0

[vm:fqdn=^hana01\.mycorp\.internal$]
vm.max_map_count = 2147483647

[vm:fqdn=^wrong\.fqdn$]
net.ipv4.tcp_tw_reuse = 1
`

	parsed := ParseINI(iniContent)
	if parsed == nil {
		t.Fatalf("ParseINI returned nil")
	}

	sysctlEntries, exists := parsed.KeyValue["sysctl"]
	if !exists {
		t.Fatalf("expected 'sysctl' section in parsed INI")
	}
	sysEntries, exists := parsed.KeyValue["sys"]
	if !exists {
		t.Fatalf("expected 'sys' section in parsed INI")
	}
	vmEntries, exists := parsed.KeyValue["vm"]
	if !exists {
		t.Fatalf("expected 'vm' section in parsed INI")
	}

	// Should match and exist
	if _, ok := sysctlEntries["kernel.sem"]; !ok {
		t.Errorf("expected 'kernel.sem' to be parsed for matching hostname")
	}
	if _, ok := sysEntries["sys:net.ipv4.tcp_slow_start_after_idle"]; !ok {
		t.Errorf("expected 'sys:net.ipv4.tcp_slow_start_after_idle' to be parsed for matching dnsdomainname")
	}
	if _, ok := vmEntries["vm.max_map_count"]; !ok {
		t.Errorf("expected 'vm.max_map_count' to be parsed for matching fqdn")
	}

	// Should not match and not exist
	if _, ok := sysctlEntries["fs.file-max"]; ok {
		t.Errorf("did not expect 'fs.file-max' for non-matching hostname")
	}
	if sysEntries["sys:net.ipv4.tcp_slow_start_after_idle"].Value != "1" {
		t.Errorf("expected net.ipv4.tcp_slow_start_after_idle value '1', got '%s'", sysEntries["sys:net.ipv4.tcp_slow_start_after_idle"].Value)
	}
	if _, ok := vmEntries["net.ipv4.tcp_tw_reuse"]; ok {
		t.Errorf("did not expect 'net.ipv4.tcp_tw_reuse' for non-matching fqdn")
	}

	// Test ParseINIFile with the same content
	confFilePath := path.Join(tmpDir, "test_note.conf")
	if err := os.WriteFile(confFilePath, []byte(iniContent), 0644); err != nil {
		t.Fatalf("failed to write test note conf: %v", err)
	}

	parsedFile, err := ParseINIFile(confFilePath, false)
	if err != nil {
		t.Fatalf("ParseINIFile failed: %v", err)
	}
	fileSysctl, exists := parsedFile.KeyValue["sysctl"]
	if !exists {
		t.Fatalf("expected 'sysctl' section in parsed file INI")
	}
	if _, ok := fileSysctl["kernel.sem"]; !ok {
		t.Errorf("expected 'kernel.sem' to be parsed in file INI")
	}
	if _, ok := fileSysctl["fs.file-max"]; ok {
		t.Errorf("did not expect 'fs.file-max' in file INI")
	}
}
