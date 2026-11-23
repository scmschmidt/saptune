package actions

import (
	"bytes"
	"fmt"
	"github.com/SUSE/saptune/sap/note"
	"github.com/SUSE/saptune/system"
	"os"
	"path"
	"strings"
	"testing"
)

func TestSetWidthOfColums(t *testing.T) {
	compare := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "IO_SCHEDULER_sr0", ActualValueJS: "cfq", ExpectedValueJS: "cfq"}
	w1 := 2
	w2 := 3
	w3 := 4
	w4 := 5
	tblwdth := 32
	v1, v2, v3, v4 := setWidthOfColums(compare, w1, w2, w3, w4)
	if v1 != w1 {
		t.Error(v1, w1)
	}
	if v2 != 16 {
		t.Error(v2, w2)
	}
	if v3 != w3 || v4 != w4 {
		t.Error(v3, w3, v4, w4)
	}
	compare = note.FieldComparison{ReflectFieldName: "OverrideParams", ReflectMapKey: "IO_SCHEDULER_sr0", ActualValueJS: "cfq", ExpectedValueJS: "cfq"}
	v1, v2, v3, v4 = setWidthOfColums(compare, w1, w2, w3, w4)
	if v1 != 3 {
		t.Error(v1, w1)
	}
	if v2 != w2 || v3 != w3 || v4 != w4 {
		t.Error(v2, w2, v3, w3, v4, w4)
	}
	compare = note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "governor", ActualValueJS: "all-none", ExpectedValueJS: "all-performance"}
	v1, v2, v3, v4 = setWidthOfColums(compare, w1, w2, w3, w4)
	if v1 != w1 {
		t.Error(v1, w1)
	}
	if v2 != 8 {
		t.Error(v2, w2)
	}
	if v3 != 15 {
		t.Error(v3, w3)
	}
	if v4 != 8 {
		t.Error(v4, w4)
	}
	compare = note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "", ActualValueJS: "all-none", ExpectedValueJS: "all-performance"}
	v1, v2, v3, v4 = setWidthOfColums(compare, w1, w2, w3, w4)
	if v1 != w1 || v2 != w2 || v3 != w3 || v4 != w4 {
		t.Error(v1, w1, v2, w2, v3, w3, v4, w4)
	}
	compare = note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "net.ipv4.ip_local_reserved_ports", ActualValueJS: "", ExpectedValueJS: "1089-1090,1095,1099,1200-1599,2000-2002,3200-3399,3500,3600-3699,3900-4001"}
	v1, v2, v3, v4 = setWidthOfColums(compare, w1, w2, w3, w4)
	if v1 != w1 {
		t.Error(v1, w1)
	}
	if v2 != tblwdth {
		t.Error(v2, w2)
	}
	if v3 != tblwdth {
		t.Error(v3, w3)
	}
	if v4 != w4 {
		t.Error(v4, w4)
	}
	compare = note.FieldComparison{ReflectFieldName: "OverrideParams", ReflectMapKey: "net.ipv4.ip_local_reserved_ports", ActualValueJS: "1089-1090,1095,1099,1200-1599,2000-2002,3200-3399,3500,3600-3699,3900-4001", ExpectedValueJS: "1089-1090,1095,1099,1200-1599,2000-2002,3200-3399,3500,3600-3699,3900-4001"}
	v1, v2, v3, v4 = setWidthOfColums(compare, w1, w2, w3, w4)
	if v1 != tblwdth {
		t.Error(v1, w1)
	}
	if v2 != w2 || v3 != w3 || v4 != w4 {
		t.Error(v2, w2, v3, w3, v4, w4)
	}
}

func TestPrintNoteFields(t *testing.T) {
	//os.Args = []string{"saptune", "--format", "json", "note", "list", "--colorscheme", "black", "--force", "--dryrun", "--help", "--version"}
	os.Args = []string{"saptune", "note", "list", "--colorscheme", "black", "--force", "--dryrun", "--help", "--version"}
	system.RereadArgs()

	footnote1 := " [1] setting is not supported by the system"
	if system.GetCSP() == "azure" {
		footnote1 = " [1] setting is not available on Azure instances (see SAP Note 2993054)."
	}

	var printMatchText1 = `
941735 - Configuration drop in for simple tests
			Version 1 from 09.07.2019 

   SAPNote, Version | Parameter                  | Expected             | Override  | Actual               | Compliant
--------------------+----------------------------+----------------------+-----------+----------------------+-----------
     941735, 1      | IO_SCHEDULER_sdb           |                      |           | all:none             |  -  [1] [5] [7]
     941735, 1      | IO_SCHEDULER_sdc           |                      |           | bfq                  | no  [7]
     941735, 1      | IO_SCHEDULER_sdd           |                      |           | bfq                  | no  [7] [10]
     941735, 1      | IO_SCHEDULER_vda           | noop                 |           | all:none             |  -  [1] [5]
     941735, 1      | ShmFileSystemSizeMB        | 1714                 |           | 488                  | no 
     941735, 1      | force_latency              | 70                   |           | all:none             | no  [1] [4]
     941735, 1      | grub:intel_idle.max_cstate | 1                    |           | NA                   |  -  [2] [3] [6]
     941735, 1      | kernel.shmmax              | 18446744073709551615 |           | 18446744073709551615 | yes
     941735, 1      | kernel.shmmni              |                      |           | NA                   |  -  [16] [7]

 ` + footnote1 + `
  [2] setting is not available on the system
  [3] value is only checked, but NOT set
  [4] cpu idle state settings differ
  [5] expected value does not contain a supported scheduler
  [6] grub settings are mostly covered by other settings. See man page saptune-note(5) for details
  [7] parameter value is untouched by default
 [10] parameter is defined twice, see section [sys] 'sys:block.sdd.queue.scheduler' from the other applied notes
 [16] parameter not available on the system, setting not possible

`
	var printMatchText1NoCompl = `
941735 - Configuration drop in for simple tests
			Version 1 from 09.07.2019 

   SAPNote, Version | Parameter                  | Expected             | Override  | Actual               | Compliant
--------------------+----------------------------+----------------------+-----------+----------------------+-----------
     941735, 1      | IO_SCHEDULER_sdc           |                      |           | bfq                  | no  [7]
     941735, 1      | IO_SCHEDULER_sdd           |                      |           | bfq                  | no  [7] [10]
     941735, 1      | ShmFileSystemSizeMB        | 1714                 |           | 488                  | no 
     941735, 1      | force_latency              | 70                   |           | all:none             | no  [1] [4]

 ` + footnote1 + `
  [2] setting is not available on the system
  [3] value is only checked, but NOT set
  [4] cpu idle state settings differ
  [6] grub settings are mostly covered by other settings. See man page saptune-note(5) for details
  [7] parameter value is untouched by default
 [10] parameter is defined twice, see section [sys] 'sys:block.sdd.queue.scheduler' from the other applied notes

`
	var printMatchText2 = `
941735 - Configuration drop in for simple tests
			Version 1 from 09.07.2019 

   Parameter                  | Value set            | Value expected       | Override  | Comment
------------------------------+----------------------+----------------------+-----------+--------------
   IO_SCHEDULER_sdb           | all:none             |                      |           |  [1] [5] [7]
   IO_SCHEDULER_sdc           | bfq                  |                      |           |  [7]
   IO_SCHEDULER_sdd           | bfq                  |                      |           |  [7] [10]
   IO_SCHEDULER_vda           | all:none             | noop                 |           |  [1] [5]
   ShmFileSystemSizeMB        | 488                  | 1714                 |           |   
   force_latency              | all:none             | 70                   |           |  [1] [4]
   grub:intel_idle.max_cstate | NA                   | 1                    |           |  [2] [3] [6]
   kernel.shmmax              | 18446744073709551615 | 18446744073709551615 |           |   
   kernel.shmmni              | NA                   |                      |           |  [16] [7]

 ` + footnote1 + `
  [2] setting is not available on the system
  [3] value is only checked, but NOT set
  [4] cpu idle state settings differ
  [5] expected value does not contain a supported scheduler
  [6] grub settings are mostly covered by other settings. See man page saptune-note(5) for details
  [7] parameter value is untouched by default
 [10] parameter is defined twice, see section [sys] 'sys:block.sdd.queue.scheduler' from the other applied notes
 [16] parameter not available on the system, setting not possible

`
	var printMatchText3 = `
   SAPNote, Version | Parameter                  | Expected             | Override  | Actual               | Compliant
--------------------+----------------------------+----------------------+-----------+----------------------+-----------
     941735, 1      | IO_SCHEDULER_sdb           |                      |           | all:none             |  -  [1] [5] [7]
     941735, 1      | IO_SCHEDULER_sdc           |                      |           | bfq                  | no  [7]
     941735, 1      | IO_SCHEDULER_sdd           |                      |           | bfq                  | no  [7] [10]
     941735, 1      | IO_SCHEDULER_vda           | noop                 |           | all:none             |  -  [1] [5]
     941735, 1      | ShmFileSystemSizeMB        | 1714                 |           | 488                  | no 
     941735, 1      | force_latency              | 70                   |           | all:none             | no  [1] [4]
     941735, 1      | grub:intel_idle.max_cstate | 1                    |           | NA                   |  -  [2] [3] [6]
     941735, 1      | kernel.shmmax              | 18446744073709551615 |           | 18446744073709551615 | yes
     941735, 1      | kernel.shmmni              |                      |           | NA                   |  -  [16] [7]

 ` + footnote1 + `
  [2] setting is not available on the system
  [3] value is only checked, but NOT set
  [4] cpu idle state settings differ
  [5] expected value does not contain a supported scheduler
  [6] grub settings are mostly covered by other settings. See man page saptune-note(5) for details
  [7] parameter value is untouched by default
 [10] parameter is defined twice, see section [sys] 'sys:block.sdd.queue.scheduler' from the other applied notes
 [16] parameter not available on the system, setting not possible

`
	var printMatchText4 = `
   Parameter                  | Value set            | Value expected       | Override  | Comment
------------------------------+----------------------+----------------------+-----------+--------------
   IO_SCHEDULER_sdb           | all:none             |                      |           |  [1] [5] [7]
   IO_SCHEDULER_sdc           | bfq                  |                      |           |  [7]
   IO_SCHEDULER_sdd           | bfq                  |                      |           |  [7] [10]
   IO_SCHEDULER_vda           | all:none             | noop                 |           |  [1] [5]
   ShmFileSystemSizeMB        | 488                  | 1714                 |           |   
   force_latency              | all:none             | 70                   |           |  [1] [4]
   grub:intel_idle.max_cstate | NA                   | 1                    |           |  [2] [3] [6]
   kernel.shmmax              | 18446744073709551615 | 18446744073709551615 |           |   
   kernel.shmmni              | NA                   |                      |           |  [16] [7]

 ` + footnote1 + `
  [2] setting is not available on the system
  [3] value is only checked, but NOT set
  [4] cpu idle state settings differ
  [5] expected value does not contain a supported scheduler
  [6] grub settings are mostly covered by other settings. See man page saptune-note(5) for details
  [7] parameter value is untouched by default
 [10] parameter is defined twice, see section [sys] 'sys:block.sdd.queue.scheduler' from the other applied notes
 [16] parameter not available on the system, setting not possible

`
	checkCorrectMessage := func(t *testing.T, got, want string) {
		t.Helper()
		if got != want {
			fmt.Println("==============")
			fmt.Println(got)
			fmt.Println("==============")
			fmt.Println(want)
			fmt.Println("==============")
			t.Errorf("Output differs from expected one")
		}
	}

	//fcomp1 := note.FieldComparison{ReflectFieldName: "ConfFilePath", ReflectMapKey: "", ActualValue: "/usr/share/saptune/notes/941735", ExpectedValue: "/usr/share/saptune/notes/941735", ActualValueJS: "/usr/share/saptune/notes/941735", ExpectedValueJS: "/usr/share/saptune/notes/941735", MatchExpectation: true}
	cfgFile := fmt.Sprintf("%ssimpleNote.conf", ExtraFilesInGOPATH)
	fcomp1 := note.FieldComparison{ReflectFieldName: "ConfFilePath", ReflectMapKey: "", ActualValue: cfgFile, ExpectedValue: cfgFile, ActualValueJS: cfgFile, ExpectedValueJS: cfgFile, MatchExpectation: true}
	fcomp2 := note.FieldComparison{ReflectFieldName: "ID", ReflectMapKey: "", ActualValue: "941735", ExpectedValue: "941735", ActualValueJS: "941735", ExpectedValueJS: "941735", MatchExpectation: true}
	fcomp3 := note.FieldComparison{ReflectFieldName: "DescriptiveName", ReflectMapKey: "", ActualValue: "Configuration drop in for simple tests\n\t\t\tVersion 1 from 09.07.2019", ExpectedValue: "Configuration drop in for simple tests\n\t\t\tVersion 1 from 09.07.2019", ActualValueJS: "Configuration drop in for simple tests\n\t\t\tVersion 1 from 09.07.2019", ExpectedValueJS: "Configuration drop in for simple tests\n\t\t\tVersion 1 from 09.07.2019", MatchExpectation: true}
	fcomp4 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "ShmFileSystemSizeMB", ActualValue: "488", ExpectedValue: "1714", ActualValueJS: "488", ExpectedValueJS: "1714", MatchExpectation: false}
	fcomp5 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "kernel.shmmax", ActualValue: "18446744073709551615", ExpectedValue: "18446744073709551615", ActualValueJS: "18446744073709551615", ExpectedValueJS: "18446744073709551615", MatchExpectation: true}
	fcomp6 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "IO_SCHEDULER_vda", ActualValue: "all:none", ExpectedValue: "noop", ActualValueJS: "all:none", ExpectedValueJS: "noop", MatchExpectation: false}
	fcomp7 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "grub:intel_idle.max_cstate", ActualValue: "NA", ExpectedValue: "1", ActualValueJS: "NA", ExpectedValueJS: "1", MatchExpectation: false}
	fcomp8 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "force_latency", ActualValue: "all:none", ExpectedValue: "70", ActualValueJS: "all:none", ExpectedValueJS: "70", MatchExpectation: false}
	fcomp9 := note.FieldComparison{ReflectFieldName: "Inform", ReflectMapKey: "force_latency", ActualValue: "hasDiffs", ExpectedValue: "hasDiffs", ActualValueJS: "hasDiffs", ExpectedValueJS: "hasDiffs", MatchExpectation: true}
	fcomp10 := note.FieldComparison{ReflectFieldName: "Inform", ReflectMapKey: "IO_SCHEDULER_vda", ActualValue: "NA", ExpectedValue: "NA", ActualValueJS: "NA", ExpectedValueJS: "NA", MatchExpectation: true}
	fcomp11 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "IO_SCHEDULER_sdb", ActualValue: "all:none", ExpectedValue: "", ActualValueJS: "all:none", ExpectedValueJS: "", MatchExpectation: false}
	fcomp12 := note.FieldComparison{ReflectFieldName: "Inform", ReflectMapKey: "IO_SCHEDULER_sdb", ActualValue: "NA", ExpectedValue: "", ActualValueJS: "NA", ExpectedValueJS: "", MatchExpectation: false}
	fcomp13 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "IO_SCHEDULER_sdc", ActualValue: "bfq", ExpectedValue: "", ActualValueJS: "bfq", ExpectedValueJS: "", MatchExpectation: false}
	fcomp14 := note.FieldComparison{ReflectFieldName: "Inform", ReflectMapKey: "IO_SCHEDULER_sdc", ActualValue: "", ExpectedValue: "bfq", ActualValueJS: "", ExpectedValueJS: "bfq", MatchExpectation: false}
	fcomp15 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "IO_SCHEDULER_sdd", ActualValue: "bfq", ExpectedValue: "", ActualValueJS: "bfq", ExpectedValueJS: "", MatchExpectation: false}
	fcomp16 := note.FieldComparison{ReflectFieldName: "Inform", ReflectMapKey: "IO_SCHEDULER_sdd", ActualValue: "", ExpectedValue: "[sys] 'sys:block.sdd.queue.scheduler' from the other applied notes", ActualValueJS: "", ExpectedValueJS: "[sys] 'sys:block.sdd.queue.scheduler' from the other applied notes", MatchExpectation: false}
	fcomp17 := note.FieldComparison{ReflectFieldName: "SysctlParams", ReflectMapKey: "kernel.shmmni", ActualValue: "PNA", ExpectedValue: "", ActualValueJS: "PNA", ExpectedValueJS: "", MatchExpectation: false}

	map941735 := map[string]note.FieldComparison{"ConfFilePath": fcomp1, "ID": fcomp2, "DescriptiveName": fcomp3, "SysctlParams[ShmFileSystemSizeMB]": fcomp4, "SysctlParams[kernel.shmmax]": fcomp5, "SysctlParams[IO_SCHEDULER_vda]": fcomp6, "SysctlParams[grub:intel_idle.max_cstate]": fcomp7, "SysctlParams[force_latency]": fcomp8, "Inform[force_latency]": fcomp9, "Inform[IO_SCHEDULER_vda]": fcomp10, "SysctlParams[IO_SCHEDULER_sdb]": fcomp11, "Inform[IO_SCHEDULER_sdb]": fcomp12, "SysctlParams[IO_SCHEDULER_sdc]": fcomp13, "Inform[IO_SCHEDULER_sdc]": fcomp14, "SysctlParams[IO_SCHEDULER_sdd]": fcomp15, "Inform[IO_SCHEDULER_sdd]": fcomp16, "SysctlParams[kernel.shmmni]": fcomp17}
	noteComp := map[string]map[string]note.FieldComparison{"941735": map941735}

	os.RemoveAll(system.SaptuneParameterStateDir)
	val, tsterr := note.ListParams()
	if tsterr == nil && len(val) > 0 {
		t.Logf("TestPrintNoteFields: there are parameter files stored: '%+v'\n", val)
	}
	t.Run("verify with header", func(t *testing.T) {
		buffer := bytes.Buffer{}
		PrintNoteFields(&buffer, "HEAD", noteComp, true, nil)
		txt := buffer.String()
		checkCorrectMessage(t, txt, printMatchText1)
	})
	t.Run("simulate with header", func(t *testing.T) {
		buffer := bytes.Buffer{}
		PrintNoteFields(&buffer, "HEAD", noteComp, false, nil)
		txt := buffer.String()
		checkCorrectMessage(t, txt, printMatchText2)
	})
	t.Run("verify without header", func(t *testing.T) {
		buffer := bytes.Buffer{}
		PrintNoteFields(&buffer, "NONE", noteComp, true, nil)
		txt := buffer.String()
		checkCorrectMessage(t, txt, printMatchText3)
	})
	t.Run("simulate without header", func(t *testing.T) {
		buffer := bytes.Buffer{}
		PrintNoteFields(&buffer, "NONE", noteComp, false, nil)
		txt := buffer.String()
		checkCorrectMessage(t, txt, printMatchText4)
	})

	t.Run("verify with header and show-non-compliant", func(t *testing.T) {
		//os.Args = []string{"saptune", "--format", "json", "note", "list", "--colorscheme", "black", "--show-non-compliant", "--force", "--dryrun", "--help", "--version"}
		os.Args = []string{"saptune", "note", "list", "--colorscheme", "black", "--show-non-compliant", "--force", "--dryrun", "--help", "--version"}
		system.RereadArgs()

		buffer := bytes.Buffer{}
		PrintNoteFields(&buffer, "HEAD", noteComp, true, nil)
		txt := buffer.String()
		checkCorrectMessage(t, txt, printMatchText1NoCompl)
	})
}

func TestGetColorScheme(t *testing.T) {
	os.Args = []string{"saptune", "status"}
	system.RereadArgs()
	saptuneSysconfig = path.Join(os.Getenv("GOPATH"), "/src/github.com/SUSE/saptune/testdata/etc/sysconfig/saptune")
	expColorScheme := "full-green-zebra"
	colorScheme := getColorScheme()
	if colorScheme != expColorScheme {
		t.Errorf("got: %+v, expected: %+v\n", colorScheme, expColorScheme)
	}
}

func TestColorPrint(t *testing.T) {
	format := `   %-16s | %-26s | %-15s | %-11s | %-14s | %2s
`
	colorScheme := "full-green-zebra"
	compliant := "yes"
	expFormat := `[32m   %-16s | %-26s | %-15s | %-11s | %-14s | %2s
[0m`
	cFormat, cCompl := colorPrint(format, compliant, colorScheme)
	if cFormat != expFormat {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, expFormat)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = "no  [2] [3] [6]"
	expFormat = `[31m   %-16s | %-26s | %-15s | %-11s | %-14s | %2s
[0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != expFormat {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, expFormat)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "cmpl-green-zebra"
	compliant = "yes"
	expCompl := `[32myes[0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != expCompl {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, expCompl)
	}

	compliant = "no  [2] [3] [6]"
	expCompl = `[31mno  [2] [3] [6][0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != expCompl {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, expCompl)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "full-blue-zebra"
	compliant = "yes"
	expFormat = `[34m   %-16s | %-26s | %-15s | %-11s | %-14s | %2s
[0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != expFormat {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, expFormat)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = "no  [2] [3] [6]"
	expFormat = `[33m   %-16s | %-26s | %-15s | %-11s | %-14s | %2s
[0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != expFormat {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, expFormat)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "cmpl-blue-zebra"
	compliant = "yes"
	expCompl = `[34myes[0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != expCompl {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, expCompl)
	}

	compliant = "no  [2] [3] [6]"
	expCompl = `[33mno  [2] [3] [6][0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != expCompl {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, expCompl)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "full-red-noncmpl"
	compliant = "yes"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = "no  [2] [3] [6]"
	expFormat = `[31m   %-16s | %-26s | %-15s | %-11s | %-14s | %2s
[0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != expFormat {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, expFormat)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "red-noncmpl"
	compliant = "yes"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = "no  [2] [3] [6]"
	expCompl = `[31mno  [2] [3] [6][0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != expCompl {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, expCompl)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "full-yellow-noncmpl"
	compliant = "yes"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = "no  [2] [3] [6]"
	expFormat = `[33m   %-16s | %-26s | %-15s | %-11s | %-14s | %2s
[0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != expFormat {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, expFormat)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "yellow-noncmpl"
	compliant = "yes"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	compliant = "no  [2] [3] [6]"
	expCompl = `[33mno  [2] [3] [6][0m`
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != expCompl {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, expCompl)
	}

	compliant = " -  [1]"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}

	colorScheme = "black"
	cFormat, cCompl = colorPrint(format, compliant, colorScheme)
	if cFormat != format {
		t.Errorf("got: %+v, expected: %+v\n", cFormat, format)
	}
	if cCompl != compliant {
		t.Errorf("got: %+v, expected: %+v\n", cCompl, compliant)
	}
}

func TestPrintNoteFieldsKmod(t *testing.T) {
	os.Args = []string{"saptune", "note", "verify", "--colorscheme", "black", "--force", "--dryrun", "--help", "--version"}
	system.RereadArgs()

	cfgFile := fmt.Sprintf("%ssimpleNote.conf", ExtraFilesInGOPATH)
	fcomp1 := note.FieldComparison{ReflectFieldName: "ConfFilePath", ReflectMapKey: "", ActualValue: cfgFile, ExpectedValue: cfgFile, ActualValueJS: cfgFile, ExpectedValueJS: cfgFile, MatchExpectation: true}
	fcomp2 := note.FieldComparison{ReflectFieldName: "ID", ReflectMapKey: "", ActualValue: "testkmod", ExpectedValue: "testkmod", ActualValueJS: "testkmod", ExpectedValueJS: "testkmod", MatchExpectation: true}
	fcompXen := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:xen",
		ActualValue:      "none|unloaded",
		ExpectedValue:    "blacklist",
		ActualValueJS:    "none|unloaded",
		ExpectedValueJS:  "blacklist|unloaded",
		MatchExpectation: false,
	}
	fcompVfat := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:vfat",
		ActualValue:      "block|unloaded",
		ExpectedValue:    "block",
		ActualValueJS:    "block|unloaded",
		ExpectedValueJS:  "block|unloaded",
		MatchExpectation: true,
	}
	fcompFloppy := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:floppy",
		ActualValue:      "blacklist+permanent|unloaded",
		ExpectedValue:    "blacklist+permanent",
		ActualValueJS:    "blacklist+permanent|unloaded",
		ExpectedValueJS:  "blacklist+permanent|unloaded",
		MatchExpectation: true,
	}
	fcompFloppyInform := note.FieldComparison{
		ReflectFieldName: "Inform",
		ReflectMapKey:    "kmod:floppy",
		ActualValue:      "",
		ExpectedValue:    "permanent",
		ActualValueJS:    "",
		ExpectedValueJS:  "permanent",
		MatchExpectation: true,
	}
	fcompInvalid := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:mod1",
		ActualValue:      "",
		ExpectedValue:    "NA",
		ActualValueJS:    "",
		ExpectedValueJS:  "NA",
		MatchExpectation: false,
	}
	fcompInvalidInform := note.FieldComparison{
		ReflectFieldName: "Inform",
		ReflectMapKey:    "kmod:mod1",
		ActualValue:      "",
		ExpectedValue:    "invalid",
		ActualValueJS:    "",
		ExpectedValueJS:  "invalid",
		MatchExpectation: true,
	}
	fcompDummy := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:dummy",
		ActualValue:      "blacklist|loaded:{}",
		ExpectedValue:    "blacklist",
		ActualValueJS:    "blacklist|loaded:{}",
		ExpectedValueJS:  "blacklist|loaded:{}",
		MatchExpectation: true,
	}
	fcompDummyInform := note.FieldComparison{
		ReflectFieldName: "Inform",
		ReflectMapKey:    "kmod:dummy",
		ActualValue:      "",
		ExpectedValue:    "loaded_on_apply",
		ActualValueJS:    "",
		ExpectedValueJS:  "loaded_on_apply",
		MatchExpectation: true,
	}
	fcompPrintk := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:printk",
		ActualValue:      "PNA",
		ExpectedValue:    "PNA",
		ActualValueJS:    "PNA",
		ExpectedValueJS:  "PNA",
		MatchExpectation: false,
	}
	fcompPrintkInform := note.FieldComparison{
		ReflectFieldName: "Inform",
		ReflectMapKey:    "kmod:printk",
		ActualValue:      "",
		ExpectedValue:    "builtin",
		ActualValueJS:    "",
		ExpectedValueJS:  "builtin",
		MatchExpectation: true,
	}

	mapTest := map[string]note.FieldComparison{
		"ConfFilePath":              fcomp1,
		"ID":                        fcomp2,
		"SysctlParams[kmod:xen]":    fcompXen,
		"SysctlParams[kmod:vfat]":   fcompVfat,
		"SysctlParams[kmod:floppy]": fcompFloppy,
		"Inform[kmod:floppy]":       fcompFloppyInform,
		"SysctlParams[kmod:mod1]":   fcompInvalid,
		"Inform[kmod:mod1]":         fcompInvalidInform,
		"SysctlParams[kmod:dummy]":  fcompDummy,
		"Inform[kmod:dummy]":        fcompDummyInform,
		"SysctlParams[kmod:printk]": fcompPrintk,
		"Inform[kmod:printk]":       fcompPrintkInform,
	}
	noteComp := map[string]map[string]note.FieldComparison{"testkmod": mapTest}

	buffer := bytes.Buffer{}
	result := system.JPNotes{}
	PrintNoteFields(&buffer, "NONE", noteComp, true, &result)
	output := buffer.String()

	if !strings.Contains(output, "[20] after a permanent entry, re-create the initrd if the module is still loaded after a reboot") {
		t.Errorf("output should contain footnote [20]: %s", output)
	}
	if !strings.Contains(output, "[21] due to an invalid entry, the module config was not done during apply") {
		t.Errorf("output should contain footnote [21]: %s", output)
	}
	if !strings.Contains(output, "[22] module is compiled into the kernel, setting not possible") {
		t.Errorf("output should contain footnote [22]: %s", output)
	}

	if strings.Contains(output, "blacklist|unloaded") {
		t.Errorf("output should not contain 'blacklist|unloaded' in Expected column: %s", output)
	}
	if strings.Contains(output, "block|unloaded") && !strings.Contains(output, "block |") {
		// block|unloaded must not appear in Expected column
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "kmod:vfat") {
				fields := strings.Split(line, " | ")
				if len(fields) >= 5 {
					expField := strings.TrimSpace(fields[2])
					actField := strings.TrimSpace(fields[4])
					if expField != "block" {
						t.Errorf("expected 'block' in Expected column for kmod:vfat, got '%s'", expField)
					}
					if actField != "block|unloaded" {
						t.Errorf("expected 'block|unloaded' in Actual column for kmod:vfat, got '%s'", actField)
					}
				}
			}
			if strings.Contains(line, "kmod:xen") {
				fields := strings.Split(line, " | ")
				if len(fields) >= 5 {
					expField := strings.TrimSpace(fields[2])
					actField := strings.TrimSpace(fields[4])
					if expField != "blacklist" {
						t.Errorf("expected 'blacklist' in Expected column for kmod:xen, got '%s'", expField)
					}
					if actField != "none|unloaded" {
						t.Errorf("expected 'none|unloaded' in Actual column for kmod:xen, got '%s'", actField)
					}
				}
			}
			if strings.Contains(line, "kmod:floppy") {
				fields := strings.Split(line, " | ")
				if len(fields) >= 5 {
					expField := strings.TrimSpace(fields[2])
					actField := strings.TrimSpace(fields[4])
					if expField != "blacklist+permanent" {
						t.Errorf("expected 'blacklist+permanent' in Expected column for kmod:floppy, got '%s'", expField)
					}
					if actField != "blacklist+permanent|unloaded" {
						t.Errorf("expected 'blacklist+permanent|unloaded' in Actual column for kmod:floppy, got '%s'", actField)
					}
				}
			}
			if strings.Contains(line, "kmod:dummy") {
				fields := strings.Split(line, " | ")
				if len(fields) >= 6 {
					expField := strings.TrimSpace(fields[2])
					actField := strings.TrimSpace(fields[4])
					complField := strings.TrimSpace(fields[5])
					if expField != "blacklist" {
						t.Errorf("expected 'blacklist' in Expected column for kmod:dummy, got '%s'", expField)
					}
					if actField != "blacklist|loaded" {
						t.Errorf("expected 'blacklist|loaded' in Actual column for kmod:dummy, got '%s'", actField)
					}
					if !strings.Contains(complField, "yes") || !strings.Contains(complField, "[19]") {
						t.Errorf("expected 'yes [19]' in Compliant column for kmod:dummy, got '%s'", complField)
					}
				}
			}
			if strings.Contains(line, "kmod:mod1") {
				fields := strings.Split(line, " | ")
				if len(fields) >= 6 {
					expField := strings.TrimSpace(fields[2])
					actField := strings.TrimSpace(fields[4])
					complField := strings.TrimSpace(fields[5])
					if expField != "NA" {
						t.Errorf("expected 'NA' in Expected column for kmod:mod1, got '%s'", expField)
					}
					if actField != "" {
						t.Errorf("expected '' (empty) in Actual column for kmod:mod1, got '%s'", actField)
					}
					if !strings.Contains(complField, "no") || !strings.Contains(complField, "[21]") {
						t.Errorf("expected 'no [21]' in Compliant column for kmod:mod1, got '%s'", complField)
					}
				}
			}
			if strings.Contains(line, "kmod:printk") {
				fields := strings.Split(line, " | ")
				if len(fields) >= 6 {
					expField := strings.TrimSpace(fields[2])
					actField := strings.TrimSpace(fields[4])
					complField := strings.TrimSpace(fields[5])
					if expField != "PNA" {
						t.Errorf("expected 'PNA' in Expected column for kmod:printk, got '%s'", expField)
					}
					if actField != "NA" {
						t.Errorf("expected 'NA' in Actual column for kmod:printk, got '%s'", actField)
					}
					if !strings.Contains(complField, "no") || !strings.Contains(complField, "[22]") {
						t.Errorf("expected 'no [22]' in Compliant column for kmod:printk, got '%s'", complField)
					}
					if strings.Contains(complField, "[16]") || strings.Contains(complField, "[20]") {
						t.Errorf("expected [16] and [20] suppressed in Compliant column for kmod:printk, got '%s'", complField)
					}
				}
			}
		}
	}

	for _, vl := range result.Verifications {
		if vl.Parameter == "kmod:xen" {
			if vl.ExpValue != "blacklist" {
				t.Errorf("MRO ExpValue for kmod:xen: got '%s', want 'blacklist'", vl.ExpValue)
			}
			if vl.ActValue == nil || *vl.ActValue != "none|unloaded" {
				t.Errorf("MRO ActValue for kmod:xen: got '%v', want 'none|unloaded'", vl.ActValue)
			}
		}
		if vl.Parameter == "kmod:vfat" {
			if vl.ExpValue != "block" {
				t.Errorf("MRO ExpValue for kmod:vfat: got '%s', want 'block'", vl.ExpValue)
			}
			if vl.ActValue == nil || *vl.ActValue != "block|unloaded" {
				t.Errorf("MRO ActValue for kmod:vfat: got '%v', want 'block|unloaded'", vl.ActValue)
			}
		}
		if vl.Parameter == "kmod:floppy" {
			if vl.ExpValue != "blacklist+permanent" {
				t.Errorf("MRO ExpValue for kmod:floppy: got '%s', want 'blacklist+permanent'", vl.ExpValue)
			}
			if vl.ActValue == nil || *vl.ActValue != "blacklist+permanent|unloaded" {
				t.Errorf("MRO ActValue for kmod:floppy: got '%v', want 'blacklist+permanent|unloaded'", vl.ActValue)
			}
		}
		if vl.Parameter == "kmod:dummy" {
			if vl.ExpValue != "blacklist" {
				t.Errorf("MRO ExpValue for kmod:dummy: got '%s', want 'blacklist'", vl.ExpValue)
			}
			if vl.ActValue == nil || *vl.ActValue != "blacklist|loaded" {
				t.Errorf("MRO ActValue for kmod:dummy: got '%v', want 'blacklist|loaded'", vl.ActValue)
			}
		}
		if vl.Parameter == "kmod:mod1" {
			if vl.ExpValue != "NA" {
				t.Errorf("MRO ExpValue for kmod:mod1: got '%s', want 'NA'", vl.ExpValue)
			}
			if vl.ActValue == nil || *vl.ActValue != "" {
				t.Errorf("MRO ActValue for kmod:mod1: got '%v', want ''", vl.ActValue)
			}
		}
		if vl.Parameter == "kmod:printk" {
			if vl.ExpValue != "PNA" {
				t.Errorf("MRO ExpValue for kmod:printk: got '%s', want 'PNA'", vl.ExpValue)
			}
			if vl.ActValue != nil {
				t.Errorf("MRO ActValue for kmod:printk: got '%v', want nil", *vl.ActValue)
			}
		}
	}
}
