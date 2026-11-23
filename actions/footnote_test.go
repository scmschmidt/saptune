package actions

import (
	"github.com/SUSE/saptune/sap/note"
	"strings"
	"testing"
)

func TestKmodFootnotes(t *testing.T) {
	// 1. Masked entry (precedence)
	comp := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:xen",
		ActualValue:      "none|unloaded",
		ExpectedValue:    "blacklist|unloaded",
		MatchExpectation: true,
	}
	footnote := make([]string, 24)
	compliant, comment, fn := prepareFootnote(comp, "yes", "", "masked", footnote)
	if !strings.Contains(compliant, "no ") || !strings.Contains(compliant, "[18]") {
		t.Errorf("expected compliant to contain 'no ' and '[18]', got '%s'", compliant)
	}
	if !strings.Contains(comment, "[18]") {
		t.Errorf("expected comment to contain '[18]', got '%s'", comment)
	}
	if fn[17] != footnote18 {
		t.Errorf("expected footnote[17] == footnote18, got '%s'", fn[17])
	}

	// 2. Loaded on apply and still loaded
	comp2 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:vfat",
		ActualValue:      "block|loaded:{}",
		ExpectedValue:    "block|loaded:{}",
		MatchExpectation: true,
	}
	footnote2 := make([]string, 24)
	compliant2, comment2, fn2 := prepareFootnote(comp2, "yes", "", "loaded_on_apply", footnote2)
	if !strings.Contains(compliant2, "yes") || !strings.Contains(compliant2, "[19]") {
		t.Errorf("expected compliant to contain 'yes' and '[19]', got '%s'", compliant2)
	}
	if !strings.Contains(comment2, "[19]") {
		t.Errorf("expected comment to contain '[19]', got '%s'", comment2)
	}
	if fn2[18] != footnote19 {
		t.Errorf("expected footnote[18] == footnote19, got '%s'", fn2[18])
	}

	// 3. Permanent entry (stays compliant 'yes', adds footnote 20)
	comp3 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:dummy",
		ActualValue:      "block|unloaded",
		ExpectedValue:    "block|unloaded",
		MatchExpectation: true,
	}
	footnote3 := make([]string, 24)
	compliant3, comment3, fn3 := prepareFootnote(comp3, "yes", "", "permanent", footnote3)
	if !strings.Contains(compliant3, "yes") || !strings.Contains(compliant3, "[20]") {
		t.Errorf("expected compliant to remain 'yes' with '[20]', got '%s'", compliant3)
	}
	if !strings.Contains(comment3, "[20]") {
		t.Errorf("expected comment to contain '[20]', got '%s'", comment3)
	}
	if fn3[19] != footnote20 {
		t.Errorf("expected footnote[19] == footnote20, got '%s'", fn3[19])
	}

	// 4. Combined flags
	comp4 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:multi",
		ActualValue:      "block|loaded:{}",
		ExpectedValue:    "block|loaded:{}",
		MatchExpectation: true,
	}
	footnote4 := make([]string, 24)
	compliant4, comment4, fn4 := prepareFootnote(comp4, "yes", "", "masked§loaded_on_apply§permanent", footnote4)
	if !strings.Contains(compliant4, "no ") {
		t.Errorf("expected compliant to be 'no ', got '%s'", compliant4)
	}
	for _, num := range []string{"[18]", "[19]", "[20]"} {
		if !strings.Contains(compliant4, num) || !strings.Contains(comment4, num) {
			t.Errorf("expected %s in compliant and comment: compliant='%s', comment='%s'", num, compliant4, comment4)
		}
	}
	if fn4[17] != footnote18 || fn4[18] != footnote19 || fn4[19] != footnote20 {
		t.Errorf("expected all 3 footnotes populated: %+v", fn4[17:20])
	}

	// 5. Invalid entry flag (footnote 21)
	comp5 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:mod1",
		ActualValue:      "none|unloaded",
		ExpectedValue:    "NA",
		MatchExpectation: false,
	}
	footnote5 := make([]string, 24)
	compliant5, comment5, fn5 := prepareFootnote(comp5, "no", "", "invalid", footnote5)
	if !strings.Contains(compliant5, "no") || !strings.Contains(compliant5, "[21]") {
		t.Errorf("expected compliant to contain 'no' and '[21]', got '%s'", compliant5)
	}
	if !strings.Contains(comment5, "[21]") {
		t.Errorf("expected comment to contain '[21]', got '%s'", comment5)
	}
	if fn5[20] != footnote21 {
		t.Errorf("expected footnote[20] == footnote21, got '%s'", fn5[20])
	}

	// 6. Built-in module flag (footnote 22, no footnote 16)
	comp6 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:printk",
		ActualValue:      "PNA",
		ExpectedValue:    "PNA",
		MatchExpectation: false,
	}
	footnote6 := make([]string, 24)
	compliant6, comment6, fn6 := prepareFootnote(comp6, "no", "", "builtin", footnote6)
	if !strings.Contains(compliant6, "no") || !strings.Contains(compliant6, "[22]") {
		t.Errorf("expected compliant to contain 'no' and '[22]', got '%s'", compliant6)
	}
	if !strings.Contains(comment6, "[22]") {
		t.Errorf("expected comment to contain '[22]', got '%s'", comment6)
	}
	if strings.Contains(compliant6, "[16]") {
		t.Errorf("expected compliant NOT to contain '[16]', got '%s'", compliant6)
	}
	if fn6[21] != footnote22 {
		t.Errorf("expected footnote[21] == footnote22, got '%s'", fn6[21])
	}
	if fn6[15] != "" {
		t.Errorf("expected footnote[15] to be empty (suppressed), got '%s'", fn6[15])
	}

	// 7. Multiple install entries (saptune wins, footnote 23)
	comp7 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:mymod",
		ActualValue:      "block|unloaded",
		ExpectedValue:    "block|unloaded",
		MatchExpectation: true,
	}
	footnote7 := make([]string, 24)
	compliant7, comment7, fn7 := prepareFootnote(comp7, "yes", "", "multiple_install", footnote7)
	if !strings.Contains(compliant7, "yes") || !strings.Contains(compliant7, "[23]") {
		t.Errorf("expected compliant to contain 'yes' and '[23]', got '%s'", compliant7)
	}
	if !strings.Contains(comment7, "[23]") {
		t.Errorf("expected comment to contain '[23]', got '%s'", comment7)
	}
	if fn7[22] != footnote23 {
		t.Errorf("expected footnote[22] == footnote23, got '%s'", fn7[22])
	}

	// 8. Shadowing warning (saptune shadows lower directory file, footnote 24)
	comp8 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:mymod",
		ActualValue:      "blacklist|unloaded",
		ExpectedValue:    "blacklist|unloaded",
		MatchExpectation: true,
	}
	footnote8 := make([]string, 24)
	compliant8, comment8, fn8 := prepareFootnote(comp8, "yes", "", "shadowing_warning", footnote8)
	if !strings.Contains(compliant8, "yes") || !strings.Contains(compliant8, "[24]") {
		t.Errorf("expected compliant to contain 'yes' and '[24]', got '%s'", compliant8)
	}
	if !strings.Contains(comment8, "[24]") {
		t.Errorf("expected comment to contain '[24]', got '%s'", comment8)
	}
	if fn8[23] != footnote24 {
		t.Errorf("expected footnote[23] == footnote24, got '%s'", fn8[23])
	}

	// 9. Deduplication: multiple masked flags must not duplicate [18]
	comp9 := note.FieldComparison{
		ReflectFieldName: "SysctlParams",
		ReflectMapKey:    "kmod:cifs",
		ActualValue:      "block|unloaded",
		ExpectedValue:    "block|unloaded",
		MatchExpectation: false,
	}
	footnote9 := make([]string, 24)
	compliant9, comment9, _ := prepareFootnote(comp9, "yes", "", "masked§shadowing_warning§masked", footnote9)
	if strings.Count(compliant9, "[18]") != 1 {
		t.Errorf("expected [18] to appear exactly once, got '%s'", compliant9)
	}
	if strings.Count(comment9, "[18]") != 1 {
		t.Errorf("expected [18] to appear exactly once in comment, got '%s'", comment9)
	}
}
