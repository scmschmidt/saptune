package system

// Gather information about system memory and swap memory.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// string definitions for parsing /proc/meminfo output
const (
	MemMainTotalKey = "MemTotal"
	MemSwapTotalKey = "SwapTotal"
)

// ProcMeminfo is the path to the /proc/meminfo file
var ProcMeminfo = "/proc/meminfo"

// ReadMeminfo parses ProcMeminfo into key(string) - value(uint64) pairs.
func ReadMeminfo() (map[string]uint64, error) {
	memInfo, err := os.ReadFile(ProcMeminfo)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", ProcMeminfo, err)
	}
	infoMap := make(map[string]uint64)
	for _, line := range strings.Split(string(memInfo), "\n") {
		fields := consecutiveSpaces.Split(strings.TrimSpace(line), -1)
		if len(fields) <= 1 {
			continue
		}
		// First field is name with a suffix colon
		name := fields[0]
		if !strings.HasSuffix(name, ":") {
			continue
		}
		// The second field is an integer value
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		infoMap[name[0:len(name)-1]] = value
	}
	return infoMap, nil
}

// GetMeminfoEntry returns the value of a specific key from ProcMeminfo.
func GetMeminfoEntry(key string) (uint64, error) {
	infoMap, err := ReadMeminfo()
	if err != nil {
		return 0, err
	}
	val, ok := infoMap[key]
	if !ok {
		return 0, fmt.Errorf("key '%s' not found in %s", key, ProcMeminfo)
	}
	return val, nil
}

// ParseMeminfo parse /proc/meminfo into key(string) - value(int) pairs.
// Panic on error.
func ParseMeminfo() (infoMap map[string]uint64) {
	var err error
	infoMap, err = ReadMeminfo()
	if err != nil {
		panic(err)
	}
	return
}

// GetMainMemSizeMB return size of system main memory, excluding swap.
// Panic on error.
func GetMainMemSizeMB() uint64 {
	InfoLog("GetMainMemSizeMB - MemTotal is '%+v'\n", ParseMeminfo()[MemMainTotalKey])
	return ParseMeminfo()[MemMainTotalKey] / 1024
}

// GetTotalMemSizeMB return size of system main memory plus swap.
// Panic on error.
func GetTotalMemSizeMB() uint64 {
	InfoLog("GetTotalMemSizeMB - MemTotal is '%+v', SwapTotal is '%+v'\n", ParseMeminfo()[MemMainTotalKey], ParseMeminfo()[MemSwapTotalKey])
	return (ParseMeminfo()[MemMainTotalKey] + ParseMeminfo()[MemSwapTotalKey]) / 1024
}

// GetTotalMemSizePages return size of system main memory plus swap, in pages.
// Panic on error.
func GetTotalMemSizePages() uint64 {
	InfoLog("GetTotalMemSizePages - MemTotal is '%+v', SwapTotal is '%+v', pagesize is '%+v'('%+v')\n", ParseMeminfo()[MemMainTotalKey], ParseMeminfo()[MemSwapTotalKey], os.Getpagesize(), uint64(os.Getpagesize()))
	return (ParseMeminfo()[MemMainTotalKey] + ParseMeminfo()[MemSwapTotalKey]) / uint64(os.Getpagesize())
}

// GetSemaphoreLimits return kernel semaphore limits. Panic on error.
func GetSemaphoreLimits() (msl, mns, opm, mni uint64) {
	field, err := GetSysctlString("kernel.sem")
	if err != nil {
		_ = fmt.Errorf("failed to read kernel.sem values")
	}
	fields := consecutiveSpaces.Split(field, -1)
	if len(fields) < 4 {
		panic(fmt.Errorf("failed to read kernel.sem values: %v", fields))
	}
	for i, val := range []*uint64{&msl, &mns, &opm, &mni} {
		if *val, err = strconv.ParseUint(fields[i], 10, 64); err != nil {
			panic(fmt.Errorf("failed to parse kernel.sem values: %v", err))
		}
	}
	return
}
