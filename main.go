package main

import (
	"fmt"
	"github.com/SUSE/saptune/actions"
	"github.com/SUSE/saptune/app"
	"github.com/SUSE/saptune/sap/note"
	"github.com/SUSE/saptune/sap/solution"
	"github.com/SUSE/saptune/system"
	"github.com/SUSE/saptune/txtparser"
	"os"
	"os/exec"
	"strings"
)

// constant definitions
const (
	saptuneV1 = "/usr/sbin/saptune_v1"
	saptcheck = "/usr/sbin/saptune_check"
	logFile   = "/var/log/saptune/saptune.log"
)

var tuneApp *app.App                 // application configuration and tuning states
var tuningOptions note.TuningOptions // Collection of tuning options from SAP notes and 3rd party vendors.
// Switch to control log reaction
var logSwitch = map[string]string{"verbose": os.Getenv("SAPTUNE_VERBOSE"), "debug": os.Getenv("SAPTUNE_DEBUG"), "error": os.Getenv("SAPTUNE_ERROR")}

// SaptuneVersion is the saptune version from /etc/sysconfig/saptune
var SaptuneVersion = ""

func main() {
	logSwitchFromConfig(system.SaptuneConfigFile(), logSwitch)
	// special log switch settings for json
	system.InitOut(logSwitch)

	// activate logging
	system.LogInit(logFile, logSwitch)
	// now system.ErrorExit can write to log and os.Stderr. No longer extra
	// care is needed.
	system.InfoLog("saptune (%s) started with '%s'", actions.RPMVersion, strings.Join(os.Args, " "))
	system.InfoLog("build for '%d'", system.IfdefVers())

	if !system.ChkCliSyntax() {
		actions.PrintHelpAndExit(os.Stdout, 1)
	}

	// get saptune version from saptune config file and check file content
	SaptuneVersion = checkSaptuneConfigFile(system.SaptuneConfigFile())

	arg1 := system.CliArg(1)
	if arg1 == "version" || system.IsFlagSet("version") {
		fmt.Printf("current active saptune version is '%s'\n", SaptuneVersion)
		system.Jcollect(SaptuneVersion)
		system.ErrorExit("", 0)
	}
	if arg1 == "help" || system.IsFlagSet("help") {
		system.JnotSupportedYet()
		actions.PrintHelpAndExit(os.Stdout, 0)
	}
	if arg1 == "" {
		actions.PrintHelpAndExit(os.Stdout, 1)
	}

	// All other actions require super user privilege
	if os.Geteuid() != 0 {
		system.ErrorExit("Please run saptune with root privilege.\n", 1)
	}

	if arg1 == "lock" {
		if arg2 := system.CliArg(2); arg2 == "remove" {
			system.JnotSupportedYet()
			system.ReleaseSaptuneLock()
			system.InfoLog("command line triggered remove of lock file '/run/.saptune.lock'\n")
			system.ErrorExit("", 0)
		} else {
			actions.PrintHelpAndExit(os.Stdout, 1)
		}
	}
	callSaptuneCheckScript(arg1)

	// only one instance of saptune should run
	// check and set saptune lock file
	system.SaptuneLock()
	defer system.ReleaseSaptuneLock()

	// cleanup runtime files
	system.CleanUpRun()
	// additional clear ignore flag for the sapconf/saptune service deadlock
	os.Remove("/run/.saptune.ignore")

	// check saptune service drop-in (azure only)
	checkSaptuneServiceDropIn()

	//check, running config exists
	checkWorkingArea()

	switch SaptuneVersion {
	case "1":
		cmd := exec.Command(saptuneV1, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			system.ErrorExit("command '%+s %+v' failed with error '%v'\n", saptuneV1, os.Args, err)
		} else {
			system.ErrorExit("", 0)
		}
	case "2", "3":
		break
	default:
		system.ErrorExit("Wrong saptune version in file '%s': %s", system.SaptuneConfigFile(), SaptuneVersion, 128)
	}

	solutionSelector := system.GetSolutionSelector()
	archSolutions, exist := solution.AllSolutions[solutionSelector]
	system.AddGap(os.Stdout)
	if !exist {
		system.ErrorExit("The system architecture (%s) is not supported.", solutionSelector)
		return
	}
	// Initialise application configuration and tuning procedures
	tuningOptions = note.GetTuningOptions(actions.NoteTuningSheets, actions.ExtraTuningSheets)
	tuneApp = app.InitialiseApp("", "", tuningOptions, archSolutions)

	checkUpdateLeftOvers()
	if err := tuneApp.NoteSanityCheck(); err != nil {
		system.ErrorExit("Error during NoteSanityCheck - '%v'\n", err)
	}
	checkForTuned()
	actions.CheckOrphanedOverrides()
	actions.SelectAction(os.Stdout, tuneApp, SaptuneVersion)
	system.ErrorExit("", 0)
}

// checkUpdateLeftOvers checks for left over files from the migration of
// saptune version 1 to saptune version 2
func checkUpdateLeftOvers() {
	// check for the /etc/tuned/saptune/tuned.conf file created during
	// the package update from saptune v1 to saptune v2/3
	// give a Warning but go ahead tuning the system
	if system.CheckForPattern("/etc/tuned/saptune/tuned.conf", "#stv1tov2#") {
		system.WarningLog("found file '/etc/tuned/saptune/tuned.conf' left over from the migration of saptune version 1 to saptune version 3. Please check and remove this file as it may work against the settings of some SAP Notes. For more information refer to the man page saptune-migrate(7)")
	}

	if system.CliArg(1) == "configure" && system.CliArg(2) == "reset" {
		return
	}

	// check if old solution or notes are applied
	if tuneApp != nil && (len(tuneApp.NoteApplyOrder) == 0 && (len(tuneApp.TuneForNotes) != 0 || len(tuneApp.TuneForSolutions) != 0)) {
		system.ErrorExit("There are 'old' solutions or notes defined in file '%s'. Seems there were some steps missed during the migration from saptune version 1 to version 3. Please check. Refer to saptune-migrate(7) for more information", system.SaptuneConfigFile())
	}
}

// checkForTuned checks for enabled and/or running tuned and prints out
// a warning message
func checkForTuned() {
	active, _ := system.SystemctlIsRunning(actions.TunedService)
	enabled, _ := system.SystemctlIsEnabled(actions.TunedService)
	if enabled || active {
		system.WarningLog("ATTENTION: tuned service is active, so we may encounter conflicting tuning values")
	}
}

// callSaptuneCheckScript will simply call the saptune_check script
// it's done before the saptune lock is set, but after the check for
// running as root
func callSaptuneCheckScript(arg string) {
	if arg == "check" {
		var err error
		if system.GetFlagVal("format") == "json" {
			var cmdOut []byte
			cmdOut, err = exec.Command(saptcheck, "--json").CombinedOutput()
			system.Jcollect(cmdOut)
		} else {
			var cmd *exec.Cmd
			// call external script saptune_check
			if system.IsFlagSet("force-color") {
				// call saptune_check unbuffered
				cmd = exec.Command("unbuffer", saptcheck)
			} else {
				cmd = exec.Command(saptcheck)
			}
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		}
		if err != nil {
			system.ErrorExit("command '%+s' failed with error '%v'\n", saptcheck, err)
		} else {
			system.ErrorExit("", 0)
		}
	}
}

// checkSaptuneServiceDropIn checks on Azure cloud if saptune service drop-in
// is available
// if not, create the needed directories, the file and make it visible
func checkSaptuneServiceDropIn() {
	if system.GetCSP() != "azure" {
		// not on azure cloude, nothing to do
		return
	}
	saptuneServiceDir := "/etc/systemd/system/saptune.service.d"
	saptuneServiceDropIn := fmt.Sprintf("%s/%s", saptuneServiceDir, "10-after_cloud-init.conf")
	if _, err := os.Stat(saptuneServiceDropIn); err == nil {
		// file exists, nothing to do
		return
	}
	system.NoticeLog("creating saptune service drop-in file...")
	if err := os.MkdirAll(saptuneServiceDir, 0755); err != nil {
		system.ErrorLog("can not create directory '%s', so writing saptune service drop-in file '%s' will not work!", saptuneServiceDir, saptuneServiceDropIn)
		return
	}
	dropInContent := fmt.Sprintf("[Unit]\nAfter=cloud-final.service\n")
	if err := os.WriteFile(saptuneServiceDropIn, []byte(dropInContent), 0644); err == nil {
		system.NoticeLog("file '%s' successfully created!", saptuneServiceDropIn)
		// make drop-in visible for systemd
		// errors are reported by the function, no need to react here
		// is handled during next reboot
		_ = system.SystemctlDaemonReload()
	} else {
		system.ErrorLog("can not write saptune service drop-in file '%s' - %v", saptuneServiceDropIn, err)
	}
}

// checkWorkingArea checks, if solution and note configs exist in the working
// area
// if not, copy the definition files from the package area into the working area
// Should be covered by package installation but better safe than sorry
func checkWorkingArea() {
	refresh := false
	files := map[string]string{"note": actions.NoteTuningSheets, "solution": actions.SolutionSheets}
	for obj, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// missing working area /var/lib/saptune/working/{notes,sols}
			refresh = true
			fmt.Println()
			system.WarningLog("missing the %ss in the working area, so copy %s definitions from package area to working area", obj, obj)
			if err := os.MkdirAll(file, 0755); err != nil {
				system.ErrorExit("Problems creating directory '%s' - '%v'", file, err)
			}
			if obj == "solution" {
				obj = "sol"
			}
			// package area /usr/share/saptune/{notes,sols}
			packedObjs := fmt.Sprintf("%s%ss/", actions.PackageArea, obj)
			_, files := system.ListDir(packedObjs, "")
			for _, f := range files {
				src := fmt.Sprintf("%s%s", packedObjs, f)
				dest := fmt.Sprintf("%s%s", file, f)
				if err := system.CopyFile(src, dest); err != nil {
					system.ErrorLog("Problems copying '%s' to '%s', continue with next file ...", src, dest)
				}
			}
		}
	}
	if refresh {
		// refresh
		solution.Refresh()
	}
}

// checkSaptuneConfigFile checks the config file /etc/sysconfig/saptune
// if it exists, if it contains all needed variables and for some variables
// checks, if the values is valid
// returns the saptune version
func checkSaptuneConfigFile(saptuneConf string) string {
	if system.CliArg(1) == "configure" && system.CliArg(2) == "reset" {
		// skip check
		return "3"
	}
	missingKey := []string{}
	keyList := actions.MandKeyList()
	sconf, err := txtparser.ParseSysconfigFile(saptuneConf, false)
	if err != nil {
		system.ErrorExit("Checking saptune configuration file - Unable to read file '%s': %v", saptuneConf, err, 128)
	}
	// check, if all needed variables are available in the saptune
	// config file
	for _, key := range keyList {
		if !sconf.IsKeyAvail(key) {
			missingKey = append(missingKey, key)
		}
	}
	if len(missingKey) != 0 {
		system.ErrorExit("File '%s' is broken. Missing variables '%s'", saptuneConf, strings.Join(missingKey, ", "), 128)
	}
	txtparser.GetSysctlExcludes(sconf.GetString("SKIP_SYSCTL_FILES", ""))
	stageVal := sconf.GetString("STAGING", "")
	if stageVal != "true" && stageVal != "false" {
		system.ErrorExit("Variable 'STAGING' from file '%s' contains a wrong value '%s'. Needs to be 'true' or 'false'", saptuneConf, stageVal, 128)
	}

	// check saptune-discovery-period of the Trento Agent
	if sconf.IsKeyAvail("TrentoASDP") {
		_ = system.CheckAndSetTrento("TrentoASDP", sconf.GetString("TrentoASDP", ""), false)
	}

	// set values read from the config file
	saptuneVers := sconf.GetString("SAPTUNE_VERSION", "")
	if saptuneVers != "1" && saptuneVers != "2" && saptuneVers != "3" {
		system.ErrorExit("Wrong saptune version in file '%s': %s", saptuneConf, saptuneVers, 128)
	}
	return saptuneVers
}

// logSwitchFromConfig reads log switch settings from the saptune
// config file
func logSwitchFromConfig(saptuneConf string, lswitch map[string]string) {
	sconf, err := txtparser.ParseSysconfigFile(saptuneConf, false)
	if err != nil {
		system.ErrorExit("Checking saptune configuration file - Unable to read file '%s': %v", saptuneConf, err, 128)
	}
	// Switch Debug on ("on") or off ("off" - default)
	// Switch verbose mode on ("on" - default) or off ("off")
	// Switch error mode on ("on" - default) or off ("off")
	// check, if DEBUG, ERROR or VERBOSE is set in /etc/sysconfig/saptune
	if lswitch["debug"] == "" {
		lswitch["debug"] = sconf.GetString("DEBUG", "off")
	}
	if lswitch["verbose"] == "" {
		lswitch["verbose"] = sconf.GetString("VERBOSE", "on")
	}
	if lswitch["error"] == "" {
		lswitch["error"] = sconf.GetString("ERROR", "on")
	}
}
