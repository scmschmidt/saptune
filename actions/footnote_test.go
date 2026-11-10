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
	footnote := make([]string, 20)
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
	footnote2 := make([]string, 20)
	compliant2, comment2, fn2 := prepareFootnote(comp2, "yes", "", "loaded_on_apply", footnote2)
	if !strings.Contains(compliant2, "no ") || !strings.Contains(compliant2, "[19]") {
		t.Errorf("expected compliant to contain 'no ' and '[19]', got '%s'", compliant2)
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
	footnote3 := make([]string, 20)
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
	footnote4 := make([]string, 20)
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
}
