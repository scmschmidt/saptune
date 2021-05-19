package actions

import (
	"fmt"
	"github.com/SUSE/saptune/sap/note"
	"github.com/SUSE/saptune/system"
	"github.com/SUSE/saptune/txtparser"
	"io"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// PrintNoteFields Print mismatching fields in the note comparison result.
//func PrintNoteFields(header string, noteComparisons map[string]map[string]note.FieldComparison, printComparison bool) {
func PrintNoteFields(writer io.Writer, header string, noteComparisons map[string]map[string]note.FieldComparison, printComparison bool) {

	// initialise
	compliant := "yes"
	printHead := ""
	noteField := ""
	footnote := make([]string, 10, 10)
	reminder := make(map[string]string)
	override := ""
	comment := ""
	hasDiff := false

	// sort output
	sortkeys := sortNoteComparisonsOutput(noteComparisons)

	// setup table format values
	fmtlen0, fmtlen1, fmtlen2, fmtlen3, fmtlen4, format := setupTableFormat(sortkeys, noteField, noteComparisons, printComparison)

	// print
	noteID := ""
	for _, skey := range sortkeys {
		comment = ""
		keyFields := strings.Split(skey, "§")
		key := keyFields[1]
		printHead, noteID, noteField = getNoteAndVersion(keyFields[0], noteID, noteField, noteComparisons)
		override = strings.Replace(noteComparisons[noteID][fmt.Sprintf("%s[%s]", "OverrideParams", key)].ExpectedValueJS, "\t", " ", -1)
		comparison := noteComparisons[noteID][fmt.Sprintf("%s[%s]", "SysctlParams", key)]
		if comparison.ReflectMapKey == "reminder" {
			reminder[noteID] = reminder[noteID] + comparison.ExpectedValueJS
			continue
		}
		// set compliant information according to the comparison result
		hasDiff, compliant = setCompliant(comparison, hasDiff)

		// check inform map for special settings
		inform := getInformSettings(noteID, noteComparisons, comparison)

		// prepare footnote
		compliant, comment, footnote = prepareFootnote(comparison, compliant, comment, inform, footnote)

		// print table header
		if printHead != "" {
			printHeadline(writer, header, noteID, noteComparisons)
			printTableHeader(writer, format, fmtlen0, fmtlen1, fmtlen2, fmtlen3, fmtlen4, printComparison)
		}

		// print table body
		if printComparison {
			// verify
			fmt.Fprintf(writer, format, noteField, comparison.ReflectMapKey, strings.Replace(comparison.ExpectedValueJS, "\t", " ", -1), override, strings.Replace(comparison.ActualValueJS, "\t", " ", -1), compliant)
		} else {
			// simulate
			fmt.Fprintf(writer, format, comparison.ReflectMapKey, strings.Replace(comparison.ActualValueJS, "\t", " ", -1), strings.Replace(comparison.ExpectedValueJS, "\t", " ", -1), override, comment)
		}
	}
	// print footer
	printTableFooter(writer, header, footnote, reminder, hasDiff)
}

// sortNoteComparisonsOutput sorts the output of the Note comparison
// the reminder section should be the last one
func sortNoteComparisonsOutput(noteCompare map[string]map[string]note.FieldComparison) []string {
	skeys := make([]string, 0, len(noteCompare))
	rkeys := make([]string, 0, len(noteCompare))
	// sort output
	for noteID, comparisons := range noteCompare {
		for _, comparison := range comparisons {
			if comparison.ReflectFieldName == "Inform" {
				// skip inform map to avoid double entries in verify table
				continue
			}
			if len(comparison.ReflectMapKey) != 0 && comparison.ReflectFieldName != "OverrideParams" {
				if comparison.ReflectMapKey != "reminder" {
					skeys = append(skeys, noteID+"§"+comparison.ReflectMapKey)
				} else {
					rkeys = append(rkeys, noteID+"§"+comparison.ReflectMapKey)
				}
			}
		}
	}
	sort.Strings(skeys)
	for _, rem := range rkeys {
		skeys = append(skeys, rem)
	}
	return skeys
}

// setupTableFormat sets the format of the table columns dependent on the content
func setupTableFormat(skeys []string, noteField string, noteCompare map[string]map[string]note.FieldComparison, printComp bool) (int, int, int, int, int, string) {
	var fmtlen0, fmtlen1, fmtlen2, fmtlen3, fmtlen4 int
	format := "\t%s : %s\n"
	// define start values for the column width
	if printComp {
		// verify
		fmtlen0 = 16
		fmtlen1 = 12
		fmtlen2 = 9
		fmtlen3 = 9
		fmtlen4 = 7
	} else {
		// simulate
		fmtlen1 = 12
		fmtlen2 = 10
		fmtlen3 = 15
		fmtlen4 = 9
	}

	for _, skey := range skeys {
		keyFields := strings.Split(skey, "§")
		noteID := keyFields[0]
		comparisons := noteCompare[noteID]
		for _, comparison := range comparisons {
			if comparison.ReflectMapKey == "reminder" || comparison.ReflectFieldName == "Inform" {
				continue
			}
			if printComp {
				// verify
				if len(noteField) > fmtlen0 {
					fmtlen0 = len(noteField)
				}
				// 3:override, 1:mapkey, 2:expval, 4:actval
				fmtlen3, fmtlen1, fmtlen2, fmtlen4 = setWidthOfColums(comparison, fmtlen3, fmtlen1, fmtlen2, fmtlen4)
				format = "   %-" + strconv.Itoa(fmtlen0) + "s | %-" + strconv.Itoa(fmtlen1) + "s | %-" + strconv.Itoa(fmtlen2) + "s | %-" + strconv.Itoa(fmtlen3) + "s | %-" + strconv.Itoa(fmtlen4) + "s | %2s\n"
			} else {
				// simulate
				// 4:override, 1:mapkey, 3:expval, 2:actval
				fmtlen4, fmtlen1, fmtlen3, fmtlen2 = setWidthOfColums(comparison, fmtlen4, fmtlen1, fmtlen3, fmtlen2)
				format = "   %-" + strconv.Itoa(fmtlen1) + "s | %-" + strconv.Itoa(fmtlen2) + "s | %-" + strconv.Itoa(fmtlen3) + "s | %-" + strconv.Itoa(fmtlen4) + "s | %2s\n"
			}
		}
	}
	return fmtlen0, fmtlen1, fmtlen2, fmtlen3, fmtlen4, format
}

// printHeadline prints a headline for the table
func printHeadline(writer io.Writer, header, id string, noteComparisons map[string]map[string]note.FieldComparison) {
	if header != "NONE" {
		nName := txtparser.GetINIFileDescriptiveName(noteComparisons[id]["ConfFilePath"].ActualValue.(string))
		fmt.Fprintf(writer, "\n%s - %s \n\n", id, nName)
	} else {
		fmt.Fprintf(writer, "\n")
	}
}

// printTableHeader prints the header of the table
func printTableHeader(writer io.Writer, format string, col0, col1, col2, col3, col4 int, printComp bool) {
	if printComp {
		// verify
		fmt.Fprintf(writer, format, "SAPNote, Version", "Parameter", "Expected", "Override", "Actual", "Compliant")
		for i := 0; i < col0+col1+col2+col3+col4+28; i++ {
			if i == 3+col0+1 || i == 3+col0+3+col1+1 || i == 3+col0+3+col1+4+col2 || i == 3+col0+3+col1+4+col2+2+col3+1 || i == 3+col0+3+col1+4+col2+2+col3+3+col4+1 {
				fmt.Fprintf(writer, "+")
			} else {
				fmt.Fprintf(writer, "-")
			}
		}
		fmt.Fprintf(writer, "\n")
	} else {
		// simulate
		fmt.Fprintf(writer, format, "Parameter", "Value set", "Value expected", "Override", "Comment")
		for i := 0; i < col1+col2+col3+col4+28; i++ {
			if i == 3+col1+1 || i == 3+col1+3+col2+1 || i == 3+col1+3+col2+3+col3+1 || i == 3+col1+3+col2+3+col3+3+col4+1 {
				fmt.Fprintf(writer, "+")
			} else {
				fmt.Fprintf(writer, "-")
			}
		}
		fmt.Fprintf(writer, "\n")
	}
}

// prepareFootnote prepares the content of the last column and the
// corresponding footnotes
func prepareFootnote(comparison note.FieldComparison, compliant, comment, inform string, footnote []string) (string, string, []string) {
	// set 'unsupported' footnote regarding the architecture
	if runtime.GOARCH == "ppc64le" {
		footnote1 = footnote1IBM
	}
	if system.GetCSP() == "azure" {
		footnote1 = footnote1AZR
	}
	if system.GetCSP() == "aws" {
		footnote1 = footnote1AWS
	}
	switch comparison.ActualValue {
	case "all:none":
		compliant = compliant + " [1]"
		comment = comment + " [1]"
		footnote[0] = footnote1
	case "NA":
		compliant = compliant + " [2]"
		comment = comment + " [2]"
		footnote[1] = footnote2
	}
	if strings.Contains(comparison.ReflectMapKey, "rpm") || strings.Contains(comparison.ReflectMapKey, "grub") {
		compliant = compliant + " [3]"
		comment = comment + " [3]"
		footnote[2] = footnote3
	}
	if strings.Contains(comparison.ReflectMapKey, "grub") {
		compliant = compliant + " [6]"
		comment = comment + " [6]"
		footnote[5] = footnote6
	}
	if comparison.ExpectedValue == "" {
		compliant = compliant + " [7]"
		comment = comment + " [7]"
		footnote[6] = footnote7
	}
	if comparison.ReflectMapKey == "energy_perf_bias" && system.SecureBootEnabled() {
		compliant = compliant + " [8]"
		comment = comment + " [8]"
		footnote[7] = footnote8
	}
	// check inform map for special settings
	compliant, comment, footnote = chkInfo(comparison.ReflectMapKey, compliant, comment, inform, footnote)
	return compliant, comment, footnote
}

// chkInfo checks inform map for special settings in the footnote
func chkInfo(mapKey, compliant, comment, info string, footnote []string) (string, string, []string) {
	// ANGI: future - check for 'nil', if using noteComparisons[noteID][fmt.Sprintf("%s[%s]", "Inform", comparison.ReflectMapKey)].ActualValue.(string) in general
	if mapKey == "force_latency" && info == "hasDiffs" {
		compliant = "no  [4]"
		comment = comment + " [4]"
		footnote[3] = footnote4
	}
	if system.IsSched.MatchString(mapKey) && strings.Contains(info, "NA") {
		compliant = compliant + " [5]"
		comment = comment + " [5]"
		footnote[4] = footnote5
	}
	if system.IsMsect.MatchString(mapKey) && strings.Contains(info, "limited") {
		compliant = compliant + " [9]"
		comment = comment + " [9]"
		footnote[8] = footnote9
	}
	if (system.IsSched.MatchString(mapKey) || system.IsNrreq.MatchString(mapKey) || system.IsRahead.MatchString(mapKey) || system.IsMsect.MatchString(mapKey)) && info != "" {
		// check for double defined parameters
		sect := regexp.MustCompile(`.*\[\w+\].*`)
		inf := strings.Split(info, "§")
		if len(inf) > 1 {
			if inf[0] != "limited" && inf[0] != "NA" {
				info = inf[0]
			} else {
				info = inf[1]
			}
		}
		if info != "limited" && info != "NA" && sect.MatchString(info) {
			compliant = compliant + " [10]"
			comment = comment + " [10]"
			footnote[9] = writeFN(footnote[9], footnote10, info, "SECT")
		}
	}
	if (strings.Contains(mapKey, "THP") || strings.Contains(mapKey, "KSM")) && info != "" {
		compliant = compliant + " [10]"
		comment = comment + " [10]"
		footnote[9] = writeFN(footnote[9], footnote10, info, "SECT")
	}
	if strings.Contains(mapKey, "sys") && info != "" {
		compliant = compliant + " [10]"
		comment = comment + " [10]"
		footnote[9] = writeFN(footnote[9], footnote10, info, "SECT")
	}
	return compliant, comment, footnote
}

// writeFN customizes the text for footnotes by replacing strings/placeholder
func writeFN(footnote, fntxt, info, pat string) string {
	if footnote == "" {
		footnote = strings.Replace(fntxt, pat, info, 1)
	} else {
		footnote = footnote + "\n " + strings.Replace(fntxt, pat, info, 1)
	}
	return footnote
}

// printTableFooter prints the footer of the table
// footnotes and reminder section
func printTableFooter(writer io.Writer, header string, footnote []string, reminder map[string]string, hasDiff bool) {
	if header != "NONE" && !hasDiff {
		fmt.Fprintf(writer, "\n   (no change)\n")
	}
	for _, fn := range footnote {
		if fn != "" {
			fmt.Fprintf(writer, "\n %s", fn)
		}
	}
	fmt.Fprintf(writer, "\n\n")
	for noteID, reminde := range reminder {
		if reminde != "" {
			reminderHead := fmt.Sprintf("Attention for SAP Note %s:\nHints or values not yet handled by saptune. So please read carefully, check and set manually, if needed:\n", noteID)
			fmt.Fprintf(writer, "%s\n", setRedText+reminderHead+reminde+resetTextColor)
		}
	}
}

// getNoteAndVersion sets printHead, noteID, noteField for the next table row
func getNoteAndVersion(kField, nID, nField string, nComparisons map[string]map[string]note.FieldComparison) (string, string, string) {
	pHead := ""
	if kField != nID {
		if nID == "" {
			pHead = "yes"
		}
		nID = kField
		nField = fmt.Sprintf("%s, %s", nID, txtparser.GetINIFileVersionSectionEntry(nComparisons[nID]["ConfFilePath"].ActualValue.(string), "version"))
	}
	return pHead, nID, nField
}

// setCompliant sets compliant information according to the comparison result
func setCompliant(comparison note.FieldComparison, hasd bool) (bool, string) {
	comp := ""
	if !comparison.MatchExpectation {
		hasd = true
		comp = "no "
	} else {
		comp = "yes"
	}
	if comparison.ActualValue.(string) == "all:none" {
		comp = " - "
	}
	return hasd, comp
}

// getInformSettings checks inform map for special settings
func getInformSettings(nID string, nComparisons map[string]map[string]note.FieldComparison, comparison note.FieldComparison) string {
	inf := ""
	if nComparisons[nID][fmt.Sprintf("%s[%s]", "Inform", comparison.ReflectMapKey)].ActualValue != nil {
		inf = nComparisons[nID][fmt.Sprintf("%s[%s]", "Inform", comparison.ReflectMapKey)].ActualValue.(string)
		if inf == "" && nComparisons[nID][fmt.Sprintf("%s[%s]", "Inform", comparison.ReflectMapKey)].ExpectedValue != nil {
			inf = nComparisons[nID][fmt.Sprintf("%s[%s]", "Inform", comparison.ReflectMapKey)].ExpectedValue.(string)
		}
	}
	return inf
}

// setWidthOfColums sets the width of the columns for verify and simulate
// depending on the highest number of characters of the content to be
// displayed
// c1:override, c2:mapkey, c3:expval, c4:actval
func setWidthOfColums(compare note.FieldComparison, c1, c2, c3, c4 int) (int, int, int, int) {
	if len(compare.ReflectMapKey) != 0 {
		if compare.ReflectFieldName == "OverrideParams" && len(compare.ActualValueJS) > c1 {
			c1 = len(compare.ActualValueJS)
			return c1, c2, c3, c4
		}
		if len(compare.ReflectMapKey) > c2 {
			c2 = len(compare.ReflectMapKey)
		}
		if len(compare.ExpectedValueJS) > c3 {
			c3 = len(compare.ExpectedValueJS)
		}
		if len(compare.ActualValueJS) > c4 {
			c4 = len(compare.ActualValueJS)
		}
	}
	return c1, c2, c3, c4
}
