package note

import (
	"testing"
)

var paramNote1 = ParameterNoteEntry{
	NoteID: "start",
	Value:  "StartValue",
}
var paramNote2 = ParameterNoteEntry{
	NoteID: "entry2",
	Value:  "AdditionalValue",
}
var paramNote3 = ParameterNoteEntry{
	NoteID: "entry3",
	Value:  "LastValue",
}

func TestGetPathToParameter(t *testing.T) {
	val := GetPathToParameter("FILENAME4TEST")
	if val != "/var/lib/saptune/parameter/FILENAME4TEST" {
		t.Fatalf("parameter file name: %v.\n", val)
	}
}

func TestGetSavedParameterNotes(t *testing.T) {
	val := GetSavedParameterNotes("TEST_PARAMETER")
	if len(val.AllNotes) > 0 {
		t.Fatalf("parameter file for 'TEST_PARAMETER' exists. content: %+v.\n", val)
	}
}

func TestIDInParameterList(t *testing.T) {
	pNotes := ParameterNotes{
		AllNotes: make([]ParameterNoteEntry, 0, 8),
	}

	pNotes.AllNotes = append(pNotes.AllNotes, paramNote1)
	pNotes.AllNotes = append(pNotes.AllNotes, paramNote2)
	pNotes.AllNotes = append(pNotes.AllNotes, paramNote3)
	if !IDInParameterList("entry2", pNotes.AllNotes) {
		t.Fatalf("'entry2' not part of list '%+v'\n", pNotes)
	}
	if IDInParameterList("HUGO", pNotes.AllNotes) {
		t.Fatalf("'HUGO' is part of list '%+v'\n", pNotes)
	}
}

func TestListParams(t *testing.T) {
	val, tsterr := ListParams()
	if tsterr == nil && len(val) > 0 {
		t.Fatalf("there are parameter files stored: '%+v'\n", val)
	}
}

func TestCreateParameterStartValues(t *testing.T) {
	CreateParameterStartValues("TEST_PARAMETER", "TestStartValue")
	val := GetSavedParameterNotes("TEST_PARAMETER")
	if len(val.AllNotes) == 0 {
		t.Fatalf("missing parameter state file 'TEST_PARAMETER': '%+v'\n", val)
	}
	if val.AllNotes[0].NoteID != "start" {
		CleanUpParamFile("TEST_PARAMETER")
		t.Fatalf("wrong content in state file 'TEST_PARAMETER', 'start' is NOT the first entry, instead it's '%+v'\n", val.AllNotes[0].NoteID)
	}
	if val.AllNotes[0].Value != "TestStartValue" {
		CleanUpParamFile("TEST_PARAMETER")
		t.Fatalf("wrong start value in state file 'TEST_PARAMETER': '%+v'\n", val.AllNotes[0].Value)
	}
	CleanUpParamFile("TEST_PARAMETER")

	// empty start value
	CreateParameterStartValues("TEST_PARAMETER", "")
	val = GetSavedParameterNotes("TEST_PARAMETER")
	if len(val.AllNotes) == 0 {
		t.Fatalf("missing parameter state file 'TEST_PARAMETER': '%+v'\n", val)
	}
	if val.AllNotes[0].NoteID != "start" {
		CleanUpParamFile("TEST_PARAMETER")
		t.Fatalf("wrong content in state file 'TEST_PARAMETER', 'start' is NOT the first entry, instead it's '%+v'\n", val.AllNotes[0].NoteID)
	}
	if val.AllNotes[0].Value != "" {
		CleanUpParamFile("TEST_PARAMETER")
		t.Fatalf("wrong start value in state file 'TEST_PARAMETER': '%+v'\n", val.AllNotes[0].Value)
	}
	CleanUpParamFile("TEST_PARAMETER")
}

func TestAddParameterNoteValues(t *testing.T) {
	AddParameterNoteValues("TEST_PARAMETER", "TestAddValue", "4711")
	val := GetSavedParameterNotes("TEST_PARAMETER")
	if len(val.AllNotes) != 0 {
		t.Fatalf("parameter state file 'TEST_PARAMETER' exists. content: '%+v'\n", val)
	}

	CreateParameterStartValues("TEST_PARAMETER", "TestStartValue")
	AddParameterNoteValues("TEST_PARAMETER", "TestAddValue", "4711")
	val = GetSavedParameterNotes("TEST_PARAMETER")
	if len(val.AllNotes) == 0 {
		t.Fatalf("missing parameter state file 'TEST_PARAMETER': '%+v'\n", val)
	}
	if val.AllNotes[0].NoteID != "start" && val.AllNotes[1].NoteID != "4711" {
		CleanUpParamFile("TEST_PARAMETER")
		t.Fatalf("wrong content in state file 'TEST_PARAMETER': '%+v'\n", val)
	}
	if val.AllNotes[0].Value != "TestStartValue" && val.AllNotes[1].Value != "TestAddValue" {
		CleanUpParamFile("TEST_PARAMETER")
		t.Fatalf("wrong content in state file 'TEST_PARAMETER': '%+v'\n", val)
	}
	if !IDInParameterList("4711", val.AllNotes) {
		CleanUpParamFile("TEST_PARAMETER")
		t.Fatalf("wrong content in state file 'TEST_PARAMETER': '%+v'\n", val)
	}
	CleanUpParamFile("TEST_PARAMETER")
}

func TestGetAllSavedParameters(t *testing.T) {
	CreateParameterStartValues("TEST_PARAMETER_1", "TestStartValue1")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue1", "4711")
	CreateParameterStartValues("TEST_PARAMETER_2", "TestStartValue2")
	AddParameterNoteValues("TEST_PARAMETER_2", "TestAddValue2", "4712")
	CreateParameterStartValues("TEST_PARAMETER_3", "TestStartValue3")
	AddParameterNoteValues("TEST_PARAMETER_3", "TestAddValue3", "4713")

	val := GetAllSavedParameters()
	if val["TEST_PARAMETER_1"].AllNotes[0].NoteID != "start" && val["TEST_PARAMETER_1"].AllNotes[1].NoteID != "4711" {
		CleanUpParamFile("TEST_PARAMETER_1")
		CleanUpParamFile("TEST_PARAMETER_2")
		CleanUpParamFile("TEST_PARAMETER_3")
		t.Fatalf("wrong content in state file '%s': '%+v'\n", "TEST_PARAMETER_1", val["TEST_PARAMETER_1"].AllNotes)
	}
	if val["TEST_PARAMETER_1"].AllNotes[0].Value != "TestStartValue1" && val["TEST_PARAMETER_1"].AllNotes[1].Value != "TestAddValue1" {
		CleanUpParamFile("TEST_PARAMETER_1")
		CleanUpParamFile("TEST_PARAMETER_2")
		CleanUpParamFile("TEST_PARAMETER_3")
		t.Fatalf("wrong content in state file '%s': '%+v'\n", "TEST_PARAMETER_1", val["TEST_PARAMETER_1"].AllNotes)
	}
	if val["TEST_PARAMETER_2"].AllNotes[0].NoteID != "start" && val["TEST_PARAMETER_2"].AllNotes[1].NoteID != "4712" {
		CleanUpParamFile("TEST_PARAMETER_1")
		CleanUpParamFile("TEST_PARAMETER_2")
		CleanUpParamFile("TEST_PARAMETER_3")
		t.Fatalf("wrong content in state file '%s': '%+v'\n", "TEST_PARAMETER_2", val["TEST_PARAMETER_2"].AllNotes)
	}
	if val["TEST_PARAMETER_2"].AllNotes[0].Value != "TestStartValue2" && val["TEST_PARAMETER_2"].AllNotes[1].Value != "TestAddValue2" {
		CleanUpParamFile("TEST_PARAMETER_1")
		CleanUpParamFile("TEST_PARAMETER_2")
		CleanUpParamFile("TEST_PARAMETER_3")
		t.Fatalf("wrong content in state file '%s': '%+v'\n", "TEST_PARAMETER_2", val["TEST_PARAMETER_2"].AllNotes)
	}
	if val["TEST_PARAMETER_3"].AllNotes[0].NoteID != "start" && val["TEST_PARAMETER_3"].AllNotes[1].NoteID != "4713" {
		CleanUpParamFile("TEST_PARAMETER_1")
		CleanUpParamFile("TEST_PARAMETER_2")
		CleanUpParamFile("TEST_PARAMETER_3")
		t.Fatalf("wrong content in state file '%s': '%+v'\n", "TEST_PARAMETER_3", val["TEST_PARAMETER_3"].AllNotes)
	}
	if val["TEST_PARAMETER_3"].AllNotes[0].Value != "TestStartValue3" && val["TEST_PARAMETER_3"].AllNotes[1].Value != "TestAddValue3" {
		CleanUpParamFile("TEST_PARAMETER_1")
		CleanUpParamFile("TEST_PARAMETER_2")
		CleanUpParamFile("TEST_PARAMETER_3")
		t.Fatalf("wrong content in state file '%s': '%+v'\n", "TEST_PARAMETER_3", val["TEST_PARAMETER_3"].AllNotes)
	}
	CleanUpParamFile("TEST_PARAMETER_1")
	CleanUpParamFile("TEST_PARAMETER_2")
	CleanUpParamFile("TEST_PARAMETER_3")
}

func TestStoreParameter(t *testing.T) {
	paramList := ParameterNotes{
		AllNotes: make([]ParameterNoteEntry, 0, 64),
	}
	param := ParameterNoteEntry{
		NoteID: "start",
		Value:  "TestStartValue1",
	}
	paramList.AllNotes = append(paramList.AllNotes, param)
	param = ParameterNoteEntry{
		NoteID: "4711",
		Value:  "TestAddValue1",
	}
	paramList.AllNotes = append(paramList.AllNotes, param)
	err := StoreParameter("TEST_PARAMETER_1", paramList, true)
	if err != nil {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("failed to store values for parameter '%s' in file: '%+v'\n", "TEST_PARAMETER_1", paramList)
	}
	CleanUpParamFile("TEST_PARAMETER_1")
}

func TestPositionInParameterList(t *testing.T) {
	CreateParameterStartValues("TEST_PARAMETER_1", "TestStartValue1")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue1", "4711")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue2", "4712")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue3", "4713")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue4", "4714")
	noteList := GetSavedParameterNotes("TEST_PARAMETER_1")
	val := PositionInParameterList("4712", noteList.AllNotes)
	if val != 2 {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong position for note '%s': '%v'\n", "4712", val)
	}
	val = PositionInParameterList("start", noteList.AllNotes)
	if val != 0 {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong position for note '%s': '%v'\n", "start", val)
	}
	val = PositionInParameterList("TEST_NON_EXIST", noteList.AllNotes)
	if val != 0 {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong position for note '%s': '%v'\n", "TEST_NON_EXIST", val)
	}
	CleanUpParamFile("TEST_PARAMETER_1")
}

func TestRevertParameter(t *testing.T) {
	// test with non existing parameter file
	val, _ := RevertParameter("TEST_PARAMETER_1", "4712")
	if val != "" {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong parameter '%s' reverted from parameter file '%s'\n", val, "TEST_PARAMETER_1")
	}

	CreateParameterStartValues("TEST_PARAMETER_1", "TestStartValue1")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue1", "4711")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue2", "4712")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue3", "4713")
	AddParameterNoteValues("TEST_PARAMETER_1", "TestAddValue4", "4714")
	val, _ = RevertParameter("TEST_PARAMETER_1", "4712")
	if val != "TestAddValue4" {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong parameter '%s' reverted for note '%s'\n", val, "4712")
	}
	val, _ = RevertParameter("TEST_PARAMETER_1", "4714")
	if val != "TestAddValue3" {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong parameter '%s' reverted for note '%s'\n", val, "4714")
	}
	val, _ = RevertParameter("TEST_PARAMETER_1", "4711")
	if val != "TestAddValue3" {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong parameter '%s' reverted for note '%s'\n", val, "4711")
	}
	val, _ = RevertParameter("TEST_PARAMETER_1", "4713")
	if val != "TestStartValue1" {
		CleanUpParamFile("TEST_PARAMETER_1")
		t.Fatalf("wrong parameter '%s' reverted for note '%s'\n", val, "4713")
	}
	CleanUpParamFile("TEST_PARAMETER_1")
}
